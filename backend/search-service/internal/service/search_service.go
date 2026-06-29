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
	Search(ctx context.Context, tenantID, query string, page, pageSize int) ([]map[string]interface{}, int, string, string, bool, error)
	TrackClick(ctx context.Context, tenantID, searchLogID, productID, query string, position int) error
	Suggest(ctx context.Context, tenantID, query string) ([]entity.Suggestion, error)
	GetProductByID(ctx context.Context, tenantID, productID string) (*entity.Product, error)
}

type AnalyticsRepository interface {
	SaveSearchLog(ctx context.Context, searchLogID, tenantID, query, normalizedQuery string, resultCount int) error
	SaveClickLog(ctx context.Context, searchLogID, tenantID, query, productID string, position int) error
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

func (s *searchService) Search(ctx context.Context, tenantID, query string, page, pageSize int) ([]map[string]interface{}, int, string, string, bool, error) {
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
	cachedData, total, cachedSearchLogID, found, err := s.cache.GetCachedSearch(ctx, tenantID, searchQuery, page, pageSize)
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

	products, total, opensearchSuggest, err := s.indexer.SearchProducts(ctx, tenantID, searchQuery, from, size)
	if err != nil {
		return nil, 0, "", "", false, err
	}

	// Fallback to OpenSearch spell suggestion if custom dictionary did not correct the query
	if !autoCorrected && opensearchSuggest != "" && strings.ToLower(opensearchSuggest) != searchQuery {
		spellcheckCorrected = opensearchSuggest
		autoCorrected = false
	}

	// Cache search results
	if cacheErr := s.cache.CacheSearch(ctx, tenantID, searchQuery, page, pageSize, products, total, searchLogID); cacheErr != nil {
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
	return s.analytics.SaveClickLog(ctx, searchLogID, tenantID, query, productID, position)
}

func (s *searchService) Suggest(ctx context.Context, tenantID, query string) ([]entity.Suggestion, error) {
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

	cached, found, err := s.cache.GetCachedSuggestions(ctx, tenantID, suggestQuery)
	if err == nil && found {
		return cached, nil
	}
	if err != nil {
		log.Printf("failed to get cached suggestions: %v", err)
	}

	suggestions, err := s.indexer.SuggestProducts(ctx, tenantID, suggestQuery)
	if err != nil {
		log.Printf("failed to get suggestions from indexer: %v", err)
		return []entity.Suggestion{}, nil
	}

	if err := s.cache.CacheSuggestions(ctx, tenantID, suggestQuery, suggestions); err != nil {
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
