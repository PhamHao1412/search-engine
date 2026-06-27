package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"strings"
	"time"
)

type SearchService interface {
	Search(ctx context.Context, tenantID, query string, page, pageSize int) ([]map[string]interface{}, int, string, error)
	TrackClick(ctx context.Context, tenantID, searchLogID, productID, query string, position int) error
}

type AnalyticsRepository interface {
	SaveSearchLog(ctx context.Context, searchLogID, tenantID, query, normalizedQuery string, resultCount int) error
	SaveClickLog(ctx context.Context, searchLogID, tenantID, query, productID string, position int) error
}

type searchService struct {
	indexer   ProductIndexer
	cache     ProductCache
	analytics AnalyticsRepository
}

func NewSearchService(indexer ProductIndexer, cache ProductCache, analytics AnalyticsRepository) SearchService {
	return &searchService{
		indexer:   indexer,
		cache:     cache,
		analytics: analytics,
	}
}

func (s *searchService) Search(ctx context.Context, tenantID, query string, page, pageSize int) ([]map[string]interface{}, int, string, error) {
	// Generate search_log_id (UUID)
	searchLogID := s.newUUID()

	// 1. Normalize
	normalized := strings.ToLower(strings.Join(strings.Fields(query), " "))

	if len(normalized) > 100 {
		return nil, 0, "", fmt.Errorf("search query exceeds 100 characters")
	}

	// 2. Cache Lookup
	cachedData, total, cachedSearchLogID, found, err := s.cache.GetCachedSearch(ctx, tenantID, normalized, page, pageSize)
	if err == nil && found {
		// Cache Hit: No DB write, return cached search log ID directly
		return cachedData, total, cachedSearchLogID, nil
	}

	// 3. OpenSearch Query
	from := (page - 1) * pageSize
	if from < 0 {
		from = 0
	}
	size := pageSize
	if size <= 0 {
		size = 20
	}

	products, total, err := s.indexer.SearchProducts(ctx, tenantID, normalized, from, size)
	if err != nil {
		return nil, 0, "", err
	}

	// 4. Cache Write
	if cacheErr := s.cache.CacheSearch(ctx, tenantID, normalized, page, pageSize, products, total, searchLogID); cacheErr != nil {
		log.Printf("Warning: Failed to save search results to cache: %v", cacheErr)
	}

	// 5. Analytics Event
	go func() {
		asyncCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.analytics.SaveSearchLog(asyncCtx, searchLogID, tenantID, query, normalized, total); err != nil {
			log.Printf("failed to save search log: %v", err)
		}
	}()

	return products, total, searchLogID, nil
}

func (s *searchService) TrackClick(ctx context.Context, tenantID, searchLogID, productID, query string, position int) error {
	if position <= 0 {
		return fmt.Errorf("click position must be greater than 0")
	}
	return s.analytics.SaveClickLog(ctx, searchLogID, tenantID, query, productID, position)
}

func (s *searchService) newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
