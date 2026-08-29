package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"search-service/internal/entity"
	"strings"
	"time"
)

type SearchService interface {
	Search(ctx context.Context, tenantID, query, lang string, page, pageSize int) ([]map[string]interface{}, int, string, string, bool, error)
	TrackClick(ctx context.Context, tenantID, searchLogID, productID, query string, position int) error
	Suggest(ctx context.Context, tenantID, query, lang string) ([]entity.Suggestion, error)
	GetProductByID(ctx context.Context, tenantID, productID string) (*entity.Product, error)
	GetHotKeywords(ctx context.Context, tenantID string, lang string, limit int) ([]string, error)
}

type AnalyticsRepository interface {
	SaveSearchLog(ctx context.Context, searchLogID, tenantID, query, normalizedQuery string, resultCount int) error
	SaveClickLog(ctx context.Context, searchLogID, tenantID, query, productID string, position int) error

	// Pre-aggregated analytics jobs
	GetRawSearchLogs(ctx context.Context, start, end time.Time) ([]entity.SearchLog, error)
	GetRawClickLogs(ctx context.Context, start, end time.Time) ([]entity.ClickLog, error)
	GetClickLogsWithProductInfo(ctx context.Context, start, end time.Time) ([]entity.ClickLogWithCategory, error)
	SaveDailyQueryAnalytics(ctx context.Context, records []entity.DailyQueryAnalytics) error
	SaveDailyCategoryAnalytics(ctx context.Context, records []entity.DailyCategoryAnalytics) error

	// Query summary dashboard data
	GetAnalyticsSummary(ctx context.Context, tenantID string, start, end time.Time) (entity.AnalyticsSummary, error)
	GetZeroResultQueries(ctx context.Context, tenantID string, start, end time.Time, limit int) ([]entity.ZeroResultQueryDetail, error)
	GetCategoryAnalytics(ctx context.Context, tenantID string, start, end time.Time) ([]entity.CategoryAnalyticsDetail, error)
	GetSpellcheckRulesCount(ctx context.Context, tenantID string) (int, error)
	GetSynonymRulesCount(ctx context.Context, tenantID string) (int, error)
	DeleteRawLogsOlderThan(ctx context.Context, before time.Time) (int64, error)
}

type searchService struct {
	indexer   ProductIndexer
	cache     ProductCache
	analytics AnalyticsRepository
	repo      SearchRepository
}

func NewSearchService(indexer ProductIndexer, cache ProductCache, analytics AnalyticsRepository, repo SearchRepository) SearchService {
	return &searchService{
		indexer:   indexer,
		cache:     cache,
		analytics: analytics,
		repo:      repo,
	}
}

func (s *searchService) Search(ctx context.Context, tenantID, query, lang string, page, pageSize int) ([]map[string]interface{}, int, string, string, bool, error) {
	// Generate search_log_id (UUID)
	searchLogID := s.newUUID()

	// Normalize query
	normalized := strings.ToLower(strings.Join(strings.Fields(query), " "))

	if len(normalized) > 100 {
		return nil, 0, "", "", false, fmt.Errorf("search query exceeds 100 characters")
	}

	// Apply custom dictionary spellcheck correction
	searchQuery := normalized
	spellcheckCorrected, autoCorrected := s.correctQuerySpelling(ctx, tenantID, normalized)
	if autoCorrected {
		searchQuery = spellcheckCorrected
	} else {
		spellcheckCorrected = ""
	}

	// Check cache for search results
	cachedData, total, cachedSearchLogID, found, err := s.cache.GetCachedSearch(ctx, tenantID, searchQuery, lang, page, pageSize)
	if err == nil && found {
		// Cache Hit: No DB write, return cached search log ID directly
		return cachedData, total, cachedSearchLogID, spellcheckCorrected, autoCorrected, nil
	}

	// Query OpenSearch for products
	from := (page - 1) * pageSize
	if from < 0 {
		from = 0
	}
	size := pageSize
	if size <= 0 {
		size = 20
	}

	// Load synonyms for current tenant
	synonyms, synErr := s.loadSynonyms(ctx, tenantID)
	if synErr != nil {
		log.Printf("Warning: Failed to load synonyms for tenant %s: %v", tenantID, synErr)
	}

	// Load translations for current tenant
	translations, transErr := s.loadTranslations(ctx, tenantID)
	if transErr != nil {
		log.Printf("Warning: Failed to load translations for tenant %s: %v", tenantID, transErr)
	}

	// Merge translations into synonyms map for seamless Query Expansion
	for k, v := range translations {
		synonyms[k] = append(synonyms[k], v...)
	}

	// Expand search query with synonyms
	synonymSegments := s.ExpandQuery(searchQuery, synonyms)

	products, total, opensearchSuggest, err := s.indexer.SearchProducts(ctx, tenantID, searchQuery, synonymSegments, lang, from, size)
	if err != nil {
		return nil, 0, "", "", false, err
	}

	// Fallback to OpenSearch spell suggestion if custom dictionary did not correct the query
	if !autoCorrected && opensearchSuggest != "" && strings.ToLower(opensearchSuggest) != searchQuery {
		spellcheckCorrected = opensearchSuggest
		autoCorrected = false
	}

	// Cache search results
	if cacheErr := s.cache.CacheSearch(ctx, tenantID, searchQuery, lang, page, pageSize, products, total, searchLogID); cacheErr != nil {
		log.Printf("Warning: Failed to save search results to cache: %v", cacheErr)
	}

	// Publish search log analytics event
	go func() {
		asyncCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.analytics.SaveSearchLog(asyncCtx, searchLogID, tenantID, query, searchQuery, total); err != nil {
			log.Printf("failed to save search log: %v", err)
		}
	}()

	return products, total, searchLogID, spellcheckCorrected, autoCorrected, nil
}

func (s *searchService) TrackClick(ctx context.Context, tenantID, searchLogID, productID, query string, position int) error {
	if position <= 0 {
		return fmt.Errorf("click position must be greater than 0")
	}

	actualSearchLogID := searchLogID
	if actualSearchLogID == "" {
		// Generate virtual search log for autocomplete click
		actualSearchLogID = s.newUUID()
		normalized := strings.ToLower(strings.Join(strings.Fields(query), " "))
		// Save virtual search log with result_count = 1
		if err := s.analytics.SaveSearchLog(ctx, actualSearchLogID, tenantID, query, normalized, 1); err != nil {
			log.Printf("Warning: failed to save virtual search log for autocomplete click: %v", err)
			// Still proceed to save click log even if search log save failed, to preserve conversion metrics
		}
	}

	return s.analytics.SaveClickLog(ctx, actualSearchLogID, tenantID, query, productID, position)
}

func (s *searchService) Suggest(ctx context.Context, tenantID, query, lang string) ([]entity.Suggestion, error) {
	normalized := strings.ToLower(strings.TrimSpace(query))
	if len(normalized) < 2 {
		return []entity.Suggestion{}, nil
	}

	// Correct search prefix using custom dictionary while typing
	suggestQuery := normalized
	corrected, autoCorrected := s.correctQuerySpelling(ctx, tenantID, normalized)
	if autoCorrected {
		suggestQuery = corrected
	}

	// Load translations for current tenant to expand query if exactly matches
	translations, transErr := s.loadTranslations(ctx, tenantID)
	if transErr == nil {
		if translatedVals, exists := translations[suggestQuery]; exists && len(translatedVals) > 0 {
			suggestQuery = translatedVals[0]
		}
	}

	cacheKey := suggestQuery + ":" + lang
	cached, found, err := s.cache.GetCachedSuggestions(ctx, tenantID, cacheKey)
	if err == nil && found {
		return cached, nil
	}
	if err != nil {
		log.Printf("failed to get cached suggestions: %v", err)
	}

	suggestions, err := s.indexer.SuggestProducts(ctx, tenantID, suggestQuery, lang)
	if err != nil {
		log.Printf("failed to get suggestions from indexer: %v", err)
		return []entity.Suggestion{}, nil
	}

	if err := s.cache.CacheSuggestions(ctx, tenantID, cacheKey, suggestions); err != nil {
		log.Printf("failed to cache suggestions: %v", err)
	}

	return suggestions, nil
}

func (s *searchService) correctQuerySpelling(ctx context.Context, tenantID, query string) (string, bool) {
	if s.repo == nil {
		return query, false
	}
	trimmed := strings.ToLower(strings.TrimSpace(query))
	if trimmed == "" {
		return query, false
	}

	// Check full query phrase in custom dictionary
	correctVal, found, err := s.cache.GetCachedSpellcheck(ctx, tenantID, trimmed)
	if err == nil && found {
		if correctVal != "" && correctVal != "-" {
			return correctVal, true
		}
	} else {
		rule, err := s.repo.GetSpellcheckRule(ctx, tenantID, trimmed)
		if err == nil && rule != nil && rule.CorrectWord != "" {
			_ = s.cache.CacheSpellcheck(ctx, tenantID, trimmed, rule.CorrectWord)
			return rule.CorrectWord, true
		} else {
			_ = s.cache.CacheSpellcheck(ctx, tenantID, trimmed, "-")
		}
	}

	// Check each individual word in custom dictionary
	words := strings.Fields(trimmed)
	if len(words) == 0 {
		return query, false
	}

	corrected := false
	correctedWords := make([]string, len(words))

	for i, word := range words {
		cleanedWord := strings.Trim(word, ".,!?\"'")
		if cleanedWord == "" {
			correctedWords[i] = word
			continue
		}

		// Check Cache
		cVal, f, err := s.cache.GetCachedSpellcheck(ctx, tenantID, cleanedWord)
		if err == nil && f {
			if cVal != "" && cVal != "-" {
				correctedWords[i] = cVal
				corrected = true
			} else {
				correctedWords[i] = word
			}
			continue
		}

		// Check DB
		rule, err := s.repo.GetSpellcheckRule(ctx, tenantID, cleanedWord)
		if err == nil && rule != nil && rule.CorrectWord != "" {
			correctedWords[i] = rule.CorrectWord
			corrected = true
			_ = s.cache.CacheSpellcheck(ctx, tenantID, cleanedWord, rule.CorrectWord)
		} else {
			correctedWords[i] = word
			_ = s.cache.CacheSpellcheck(ctx, tenantID, cleanedWord, "-")
		}
	}

	if corrected {
		return strings.Join(correctedWords, " "), true
	}
	return query, false
}

func (s *searchService) newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

func (s *searchService) GetProductByID(ctx context.Context, tenantID, productID string) (*entity.Product, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("repository not configured")
	}
	p, err := s.repo.GetProductByID(ctx, productID)
	if err != nil {
		return nil, err
	}
	if p.TenantID != tenantID {
		return nil, fmt.Errorf("product not found under this tenant")
	}
	return p, nil
}

func (s *searchService) loadSynonyms(ctx context.Context, tenantID string) (map[string][]string, error) {
	if s.repo == nil {
		return make(map[string][]string), nil
	}

	if s.cache != nil {
		cached, found, err := s.cache.GetCachedSynonyms(ctx, tenantID)
		if err == nil && found {
			return cached, nil
		}
	}

	// Retrieve from Postgres
	dbRules, err := s.repo.GetSearchSynonyms(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	synonyms := make(map[string][]string)
	for _, rule := range dbRules {
		key := removeDiacritics(strings.ToLower(strings.TrimSpace(rule.Keyword)))
		val := strings.ToLower(strings.TrimSpace(rule.Synonym))
		if key != "" && val != "" {
			synonyms[key] = append(synonyms[key], val)
		}
	}

	if s.cache != nil {
		_ = s.cache.CacheSynonyms(ctx, tenantID, synonyms)
	}

	return synonyms, nil
}

func (s *searchService) loadTranslations(ctx context.Context, tenantID string) (map[string][]string, error) {
	if s.repo == nil {
		return make(map[string][]string), nil
	}

	if s.cache != nil {
		cached, found, err := s.cache.GetCachedTranslations(ctx, tenantID)
		if err == nil && found {
			return cached, nil
		}
	}

	// Retrieve from Postgres
	dbRules, err := s.repo.GetSearchTranslations(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	translations := make(map[string][]string)
	for _, rule := range dbRules {
		key := removeDiacritics(strings.ToLower(strings.TrimSpace(rule.Keyword)))
		val := strings.ToLower(strings.TrimSpace(rule.Translation))
		if key != "" && val != "" {
			translations[key] = append(translations[key], val)
		}
	}

	if s.cache != nil {
		_ = s.cache.CacheTranslations(ctx, tenantID, translations)
	}

	return translations, nil
}

func (s *searchService) ExpandQuery(query string, synonyms map[string][]string) [][]string {
	words := strings.Fields(query)
	n := len(words)
	if n == 0 {
		return nil
	}

	noAccentWords := make([]string, n)
	for i, w := range words {
		noAccentWords[i] = removeDiacritics(w)
	}

	var segments [][]string
	i := 0
	for i < n {
		matched := false
		for length := n - i; length >= 2; length-- {
			phraseNoAccent := strings.Join(noAccentWords[i:i+length], " ")
			if expanded, found := synonyms[phraseNoAccent]; found {
				originalPhrase := strings.Join(words[i:i+length], " ")
				segment := []string{originalPhrase}
				seen := map[string]bool{originalPhrase: true}
				for _, exp := range expanded {
					if !seen[exp] {
						segment = append(segment, exp)
						seen[exp] = true
					}
				}
				segments = append(segments, segment)
				i += length
				matched = true
				break
			}
		}

		if !matched {
			wordNoAccent := noAccentWords[i]
			if expanded, found := synonyms[wordNoAccent]; found {
				originalWord := words[i]
				segment := []string{originalWord}
				seen := map[string]bool{originalWord: true}
				for _, exp := range expanded {
					if !seen[exp] {
						segment = append(segment, exp)
						seen[exp] = true
					}
				}
				segments = append(segments, segment)
			} else {
				segments = append(segments, []string{words[i]})
			}
			i++
		}
	}

	return segments
}

func removeDiacritics(s string) string {
	s = strings.ToLower(s)
	replacer := strings.NewReplacer(
		"à", "a", "á", "a", "ả", "a", "ã", "a", "ạ", "a",
		"ă", "a", "ằ", "a", "ắ", "a", "ẳ", "a", "ẵ", "a", "ặ", "a",
		"â", "a", "ầ", "a", "ấ", "a", "ẩ", "a", "ẫ", "a", "ậ", "a",
		"đ", "d",
		"è", "e", "é", "e", "ẻ", "e", "ẽ", "e", "ẹ", "e",
		"ê", "e", "ề", "e", "ế", "e", "ể", "e", "ễ", "e", "ệ", "e",
		"ì", "i", "í", "i", "ỉ", "i", "ĩ", "i", "ị", "i",
		"ò", "o", "ó", "o", "ỏ", "o", "õ", "o", "ọ", "o",
		"ô", "o", "ồ", "o", "ố", "o", "ổ", "o", "ỗ", "o", "ộ", "o",
		"ơ", "o", "ờ", "o", "ớ", "o", "ở", "o", "ỡ", "o", "ợ", "o",
		"ù", "u", "ú", "u", "ủ", "u", "ũ", "u", "ụ", "u",
		"ư", "u", "ừ", "u", "ứ", "u", "ử", "u", "ữ", "u", "ự", "u",
		"ỳ", "y", "ý", "y", "ỷ", "y", "ỹ", "y", "ỵ", "y",
	)
	return replacer.Replace(s)
}

// Normalize raw query text against actual product title
func normalizeQueryWithProduct(query string, productTitle string) string {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return ""
	}
	productTitle = strings.TrimSpace(productTitle)
	if productTitle == "" {
		return ""
	}

	queryWords := strings.Fields(query)
	productWords := strings.Fields(productTitle)

	productWordSet := make(map[string]string)
	for _, w := range productWords {
		normW := removeDiacritics(strings.ToLower(w))
		cleanW := strings.Trim(normW, ".,-()[]{}!?;:\"'")
		if cleanW != "" {
			productWordSet[cleanW] = w
		}
	}

	var matchedWords []string
	for _, qw := range queryWords {
		normQW := removeDiacritics(qw)
		cleanQW := strings.Trim(normQW, ".,-()[]{}!?;:\"'")
		if cleanQW == "" {
			continue
		}

		originalWord, exists := productWordSet[cleanQW]
		if !exists {
			return ""
		}
		matchedWords = append(matchedWords, originalWord)
	}

	if len(matchedWords) > 0 {
		return strings.Join(matchedWords, " ")
	}

	return ""
}

// Retrieve dynamic search suggestions based on top search logs and index references
func (s *searchService) GetHotKeywords(ctx context.Context, tenantID string, lang string, limit int) ([]string, error) {
	logs, err := s.repo.GetTopQueries(ctx, tenantID, limit*3)
	if err != nil {
		return nil, err
	}

	var keywords []string
	seen := make(map[string]bool)

	nameField := "product_name_vi"
	if lang == "en" {
		nameField = "product_name_en"
	} else if lang == "th" {
		nameField = "product_name_th"
	}

	for _, logEntry := range logs {
		if len(keywords) >= limit {
			break
		}

		products, _, _, err := s.indexer.SearchProducts(ctx, tenantID, logEntry.NormalizedQuery, nil, lang, 0, 1)
		if err != nil || len(products) == 0 {
			continue
		}

		firstProduct := products[0]
		productName, _ := firstProduct[nameField].(string)
		if productName == "" && nameField != "product_name_vi" {
			productName, _ = firstProduct["product_name_vi"].(string)
		}

		norm := normalizeQueryWithProduct(logEntry.NormalizedQuery, productName)
		if norm == "" {
			continue
		}

		key := removeDiacritics(strings.ToLower(norm))
		if !seen[key] {
			seen[key] = true
			keywords = append(keywords, norm)
		}
	}

	if len(keywords) < limit {
		defaultKeywords := []string{"Bàn phím cơ", "Chuột không dây", "Keycap", "Tai nghe", "Bàn di chuột"}
		if lang == "en" {
			defaultKeywords = []string{"Mechanical Keyboard", "Wireless Mouse", "Keycap", "Headphone", "Mousepad"}
		} else if lang == "th" {
			defaultKeywords = []string{"คีย์บอร์ดกลไก", "เมาส์ไร้สาย", "คีย์แคป", "หูฟัง", "แผ่นรองเมาส์"}
		}

		for _, dk := range defaultKeywords {
			if len(keywords) >= limit {
				break
			}
			key := removeDiacritics(strings.ToLower(dk))
			if !seen[key] {
				seen[key] = true
				keywords = append(keywords, dk)
			}
		}
	}

	return keywords, nil
}
