package service_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"search-service/internal/service"
)

type MockAnalyticsPublisher struct {
	PublishSearchLogFn func(ctx context.Context, tenantID, query, normalizedQuery string, resultCount int) error
	CallsCount         int
}

func (m *MockAnalyticsPublisher) PublishSearchLog(ctx context.Context, tenantID, query, normalizedQuery string, resultCount int) error {
	m.CallsCount++
	if m.PublishSearchLogFn != nil {
		return m.PublishSearchLogFn(ctx, tenantID, query, normalizedQuery, resultCount)
	}
	return nil
}

func TestSearchService_CacheHit(t *testing.T) {
	indexer := NewMockProductIndexer()
	cache := &MockProductCache{}
	publisher := &MockAnalyticsPublisher{}

	// Setup Cache Hit
	cachedProducts := []map[string]interface{}{
		{"id": "p-1", "product_name_vi": "Sản phẩm cached"},
	}
	cache.GetCachedSearchFn = func(ctx context.Context, tenantID, query string, page, pageSize int) ([]map[string]interface{}, int, bool, error) {
		return cachedProducts, 1, true, nil
	}

	// Verify Indexer is NOT called
	indexer.SearchProductsFn = func(ctx context.Context, tenantID, query string, from, size int) ([]map[string]interface{}, int, error) {
		t.Fatal("Indexer should not be called on cache hit")
		return nil, 0, nil
	}

	svc := service.NewSearchService(indexer, cache, publisher)
	res, total, err := svc.Search(context.Background(), "tenant-1", "test-query", 1, 20)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(res) != 1 || res[0]["id"] != "p-1" {
		t.Errorf("Expected cached product, got: %v", res)
	}
	if total != 1 {
		t.Errorf("Expected total 1, got %d", total)
	}

	// Wait briefly to allow goroutine analytics publication
	time.Sleep(50 * time.Millisecond)
	if publisher.CallsCount != 1 {
		t.Errorf("Expected analytics publisher to be called once, got %d", publisher.CallsCount)
	}
}

func TestSearchService_CacheMiss(t *testing.T) {
	indexer := NewMockProductIndexer()
	cache := &MockProductCache{}
	publisher := &MockAnalyticsPublisher{}

	// Setup Cache Miss
	cache.GetCachedSearchFn = func(ctx context.Context, tenantID, query string, page, pageSize int) ([]map[string]interface{}, int, bool, error) {
		return nil, 0, false, nil
	}

	// Mock Indexer call
	indexedProducts := []map[string]interface{}{
		{"id": "p-2", "product_name_vi": "Sản phẩm indexed"},
	}
	indexer.SearchProductsFn = func(ctx context.Context, tenantID, query string, from, size int) ([]map[string]interface{}, int, error) {
		return indexedProducts, 1, nil
	}

	// Mock Cache Set
	var cacheSetCalled bool
	cache.CacheSearchFn = func(ctx context.Context, tenantID, query string, page, pageSize int, data []map[string]interface{}, total int) error {
		cacheSetCalled = true
		if len(data) != 1 || data[0]["id"] != "p-2" {
			t.Errorf("Expected data indexed to be cached, got %v", data)
		}
		if total != 1 {
			t.Errorf("Expected total 1, got %d", total)
		}
		return nil
	}

	svc := service.NewSearchService(indexer, cache, publisher)
	res, total, err := svc.Search(context.Background(), "tenant-1", "test-query", 1, 20)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(res) != 1 || res[0]["id"] != "p-2" {
		t.Errorf("Expected indexed product, got %v", res)
	}
	if total != 1 {
		t.Errorf("Expected total 1, got %d", total)
	}
	if !cacheSetCalled {
		t.Error("Expected CacheSearch to be called on cache miss")
	}

	// Wait briefly to allow goroutine analytics publication
	time.Sleep(50 * time.Millisecond)
	if publisher.CallsCount != 1 {
		t.Errorf("Expected analytics publisher to be called once, got %d", publisher.CallsCount)
	}
}

func TestSearchService_Normalization(t *testing.T) {
	indexer := NewMockProductIndexer()
	cache := &MockProductCache{}
	publisher := &MockAnalyticsPublisher{}

	cache.GetCachedSearchFn = func(ctx context.Context, tenantID, query string, page, pageSize int) ([]map[string]interface{}, int, bool, error) {
		// Verify query is normalized when checking cache
		if query != "ca phe sua" {
			t.Errorf("Expected query to be normalized as 'ca phe sua', got: '%s'", query)
		}
		return nil, 0, false, nil
	}

	indexer.SearchProductsFn = func(ctx context.Context, tenantID, query string, from, size int) ([]map[string]interface{}, int, error) {
		// Verify query is normalized when searching in OpenSearch
		if query != "ca phe sua" {
			t.Errorf("Expected query to be normalized as 'ca phe sua', got: '%s'", query)
		}
		return nil, 0, nil
	}

	svc := service.NewSearchService(indexer, cache, publisher)
	_, _, err := svc.Search(context.Background(), "tenant-1", "   Ca   phe   Sua   ", 1, 20)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestSearchService_QueryLengthLimit(t *testing.T) {
	indexer := NewMockProductIndexer()
	cache := &MockProductCache{}
	publisher := &MockAnalyticsPublisher{}

	svc := service.NewSearchService(indexer, cache, publisher)
	longQuery := strings.Repeat("a", 101)
	_, _, err := svc.Search(context.Background(), "tenant-1", longQuery, 1, 20)
	if err == nil {
		t.Fatal("Expected error for query longer than 100 characters, got nil")
	}
	if !strings.Contains(err.Error(), "exceeds 100 characters") {
		t.Errorf("Expected exceeds length error message, got: %v", err)
	}
}
