package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"time"

	"search-service/internal/entity"
)

type AnalyticsService interface {
	AggregateAnalytics(ctx context.Context, targetDate time.Time) error
	GetAnalyticsSummary(ctx context.Context, tenantID string, rangeFilter string) (map[string]interface{}, error)
	DeleteOldRawLogs(ctx context.Context, retentionDays int) (int64, error)
	TriggerAggregation(ctx context.Context) error
}

type analyticsService struct {
	repo AnalyticsRepository
}

func NewAnalyticsService(repo AnalyticsRepository) AnalyticsService {
	return &analyticsService{repo: repo}
}

func (s *analyticsService) AggregateAnalytics(ctx context.Context, targetDate time.Time) error {
	// 1. Calculate date boundaries (00:00:00 to 23:59:59 in target timezone)
	loc := targetDate.Location()
	start := time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), 0, 0, 0, 0, loc)
	end := start.AddDate(0, 0, 1)

	log.Printf("[AnalyticsService] Aggregating logs for date: %s (Range: %s to %s)...", start.Format("2006-01-02"), start.Format(time.RFC3339), end.Format(time.RFC3339))

	// 2. Fetch raw search and click logs
	searchLogs, err := s.repo.GetRawSearchLogs(ctx, start, end)
	if err != nil {
		return fmt.Errorf("failed to fetch raw search logs: %w", err)
	}

	clickLogs, err := s.repo.GetRawClickLogs(ctx, start, end)
	if err != nil {
		return fmt.Errorf("failed to fetch raw click logs: %w", err)
	}

	clickCats, err := s.repo.GetClickLogsWithProductInfo(ctx, start, end)
	if err != nil {
		return fmt.Errorf("failed to fetch click categories: %w", err)
	}

	// 3. Process Daily Query Analytics
	type queryKey struct {
		tenantID string
		query    string
	}

	queryStats := make(map[queryKey]*entity.DailyQueryAnalytics)

	// Aggregate Search Logs
	for _, l := range searchLogs {
		k := queryKey{tenantID: l.TenantID, query: l.NormalizedQuery}
		stat, ok := queryStats[k]
		if !ok {
			stat = &entity.DailyQueryAnalytics{
				ID:        s.newUUID(),
				TenantID:  l.TenantID,
				Query:     l.NormalizedQuery,
				Date:      start,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			queryStats[k] = stat
		}
		stat.SearchCount++
		if l.ResultCount == 0 {
			stat.ZeroResultCount++
		}
	}

	// Aggregate Click Logs
	for _, c := range clickLogs {
		k := queryKey{tenantID: c.TenantID, query: c.Query}
		stat, ok := queryStats[k]
		if !ok {
			stat = &entity.DailyQueryAnalytics{
				ID:        s.newUUID(),
				TenantID:  c.TenantID,
				Query:     c.Query,
				Date:      start,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			queryStats[k] = stat
		}
		stat.ClickCount++
		stat.SumClickPosition += c.ClickPosition
	}

	// Flatten Query Analytics map to slice
	queryRecords := make([]entity.DailyQueryAnalytics, 0, len(queryStats))
	for _, record := range queryStats {
		queryRecords = append(queryRecords, *record)
	}

	// 4. Process Daily Category Analytics
	// Step 4.1: Find the primary category for each query
	type primaryCatInfo struct {
		categoryID   string
		categoryName string
		clickCount   int
	}
	primaryCategories := make(map[queryKey]primaryCatInfo)

	for _, cc := range clickCats {
		k := queryKey{tenantID: cc.TenantID, query: cc.Query}
		currentBest, exists := primaryCategories[k]
		if !exists || cc.ClickCount > currentBest.clickCount {
			primaryCategories[k] = primaryCatInfo{
				categoryID:   cc.CategoryID,
				categoryName: cc.CategoryName,
				clickCount:   cc.ClickCount,
			}
		}
	}

	// Step 4.2: Aggregate search_count and click_count by category
	type catKey struct {
		tenantID   string
		categoryID string
	}
	categoryStats := make(map[catKey]*entity.DailyCategoryAnalytics)

	// Attribute Search Count to categories (based on query's primary category)
	for k, stat := range queryStats {
		catInfo, hasCat := primaryCategories[k]
		if !hasCat {
			continue
		}

		ck := catKey{tenantID: k.tenantID, categoryID: catInfo.categoryID}
		catStat, ok := categoryStats[ck]
		if !ok {
			catStat = &entity.DailyCategoryAnalytics{
				ID:           s.newUUID(),
				TenantID:     k.tenantID,
				CategoryID:   catInfo.categoryID,
				CategoryName: catInfo.categoryName,
				Date:         start,
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			}
			categoryStats[ck] = catStat
		}
		catStat.SearchCount += stat.SearchCount
	}

	// Add Clicks directly from product categories
	for _, cc := range clickCats {
		ck := catKey{tenantID: cc.TenantID, categoryID: cc.CategoryID}
		catStat, ok := categoryStats[ck]
		if !ok {
			catStat = &entity.DailyCategoryAnalytics{
				ID:           s.newUUID(),
				TenantID:     cc.TenantID,
				CategoryID:   cc.CategoryID,
				CategoryName: cc.CategoryName,
				Date:         start,
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			}
			categoryStats[ck] = catStat
		}
		catStat.ClickCount += cc.ClickCount
	}

	// Flatten Category Analytics map to slice
	categoryRecords := make([]entity.DailyCategoryAnalytics, 0, len(categoryStats))
	for _, record := range categoryStats {
		categoryRecords = append(categoryRecords, *record)
	}

	// 5. Save to database (UPSERT)
	if err := s.repo.SaveDailyQueryAnalytics(ctx, queryRecords); err != nil {
		return fmt.Errorf("failed to save daily query analytics: %w", err)
	}

	if err := s.repo.SaveDailyCategoryAnalytics(ctx, categoryRecords); err != nil {
		return fmt.Errorf("failed to save daily category analytics: %w", err)
	}

	log.Printf("[AnalyticsService] Aggregation completed. Saved %d query analytics and %d category analytics records.", len(queryRecords), len(categoryRecords))
	return nil
}

func (s *analyticsService) GetAnalyticsSummary(ctx context.Context, tenantID string, rangeFilter string) (map[string]interface{}, error) {
	// 1. Calculate time boundaries based on rangeFilter
	now := time.Now()
	var start, end time.Time

	switch rangeFilter {
	case "today":
		start = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		end = start.AddDate(0, 0, 1)
	case "7days":
		start = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, -6)
		end = now
	case "30days":
		fallthrough
	default:
		start = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, -29)
		end = now
	}

	// 2. Query summary data
	summary, err := s.repo.GetAnalyticsSummary(ctx, tenantID, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to query analytics summary: %w", err)
	}

	zeroResults, err := s.repo.GetZeroResultQueries(ctx, tenantID, start, end, 10)
	if err != nil {
		return nil, fmt.Errorf("failed to query zero result queries: %w", err)
	}

	categoryAnalytics, err := s.repo.GetCategoryAnalytics(ctx, tenantID, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to query category analytics: %w", err)
	}

	spellcheckCount, err := s.repo.GetSpellcheckRulesCount(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to query spellcheck rules count: %w", err)
	}

	synonymCount, err := s.repo.GetSynonymRulesCount(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to query synonym rules count: %w", err)
	}

	// Calculate overall CTR and average position
	ctr := 0.0
	if summary.TotalSearches > 0 {
		ctr = float64(summary.ClickCount) / float64(summary.TotalSearches) * 100
		// Round to 1 decimal place
		ctr = float64(int(ctr*10)) / 10.0
	}

	avgPos := 0.0
	if summary.ClickCount > 0 {
		avgPos = float64(summary.SumClickPosition) / float64(summary.ClickCount)
		// Round to 1 decimal place
		avgPos = float64(int(avgPos*10)) / 10.0
	}

	result := map[string]interface{}{
		"summary": map[string]interface{}{
			"total_searches":         summary.TotalSearches,
			"zero_result_searches":   summary.ZeroResultSearches,
			"ctr":                    ctr,
			"avg_click_position":     avgPos,
			"spellcheck_rules_count": spellcheckCount,
			"synonym_rules_count":    synonymCount,
		},
		"zero_results":       zeroResults,
		"category_analytics": categoryAnalytics,
	}

	return result, nil
}

func (s *analyticsService) DeleteOldRawLogs(ctx context.Context, retentionDays int) (int64, error) {
	if retentionDays <= 0 {
		retentionDays = 90
	}
	before := time.Now().AddDate(0, 0, -retentionDays)
	log.Printf("[AnalyticsService] Deleting raw logs older than %s...", before.Format("2006-01-02"))
	return s.repo.DeleteRawLogsOlderThan(ctx, before)
}

func (s *analyticsService) TriggerAggregation(ctx context.Context) error {
	log.Println("[AnalyticsService] TriggerAggregation manually called.")
	// Run aggregation for yesterday and today
	if err := s.AggregateAnalytics(ctx, time.Now().AddDate(0, 0, -1)); err != nil {
		return err
	}
	return s.AggregateAnalytics(ctx, time.Now())
}

func (s *analyticsService) newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
