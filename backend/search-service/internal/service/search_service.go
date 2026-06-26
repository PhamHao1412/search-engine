package service

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"
)

type SearchService interface {
	Search(ctx context.Context, tenantID, query string, page, pageSize int) ([]map[string]interface{}, int, error)
}

type AnalyticsPublisher interface {
	PublishSearchLog(ctx context.Context, tenantID, query, normalizedQuery string, resultCount int) error
}

type searchService struct {
	indexer   ProductIndexer
	cache     ProductCache
	publisher AnalyticsPublisher
}

func NewSearchService(indexer ProductIndexer, cache ProductCache, publisher AnalyticsPublisher) SearchService {
	return &searchService{
		indexer:   indexer,
		cache:     cache,
		publisher: publisher,
	}
}

func (s *searchService) Search(ctx context.Context, tenantID, query string, page, pageSize int) ([]map[string]interface{}, int, error) {
	// 1. Normalize
	normalized := strings.ToLower(strings.Join(strings.Fields(query), " "))

	if len(normalized) > 100 {
		return nil, 0, fmt.Errorf("search query exceeds 100 characters")
	}

	// 2. Cache Lookup
	cachedData, total, found, err := s.cache.GetCachedSearch(ctx, tenantID, normalized, page, pageSize)
	if err == nil && found {
		// Cache Hit
		go func() {
			asyncCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := s.publisher.PublishSearchLog(asyncCtx, tenantID, query, normalized, total); err != nil {
				log.Printf("failed to publish cached search log: %v", err)
			}
		}()
		return cachedData, total, nil
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
		return nil, 0, err
	}

	// 4. Cache Write
	if cacheErr := s.cache.CacheSearch(ctx, tenantID, normalized, page, pageSize, products, total); cacheErr != nil {
		log.Printf("Warning: Failed to save search results to cache: %v", cacheErr)
	}

	// 5. Analytics Event
	go func() {
		asyncCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.publisher.PublishSearchLog(asyncCtx, tenantID, query, normalized, total); err != nil {
			log.Printf("failed to publish search log: %v", err)
		}
	}()

	return products, total, nil
}
