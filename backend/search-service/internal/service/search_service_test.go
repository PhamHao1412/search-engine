package service_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"search-service/internal/entity"
	"search-service/internal/service"
)

type MockAnalyticsRepository struct {
	SaveSearchLogFn func(ctx context.Context, searchLogID, tenantID, query, normalizedQuery string, resultCount int) error
	SaveClickLogFn  func(ctx context.Context, searchLogID, tenantID, query, productID string, position int) error
	CallsCount      int
}

func (m *MockAnalyticsRepository) SaveSearchLog(ctx context.Context, searchLogID, tenantID, query, normalizedQuery string, resultCount int) error {
	m.CallsCount++
	if m.SaveSearchLogFn != nil {
		return m.SaveSearchLogFn(ctx, searchLogID, tenantID, query, normalizedQuery, resultCount)
	}
	return nil
}

func (m *MockAnalyticsRepository) SaveClickLog(ctx context.Context, searchLogID, tenantID, query, productID string, position int) error {
	m.CallsCount++
	if m.SaveClickLogFn != nil {
		return m.SaveClickLogFn(ctx, searchLogID, tenantID, query, productID, position)
	}
	return nil
}

func TestSearchService_CacheHit(t *testing.T) {
	indexer := NewMockProductIndexer()
	cache := &MockProductCache{}
	analytics := &MockAnalyticsRepository{}

	// Setup Cache Hit
	cachedProducts := []map[string]interface{}{
		{"id": "p-1", "product_name_vi": "Sản phẩm cached"},
	}
	cache.GetCachedSearchFn = func(ctx context.Context, tenantID, query string, page, pageSize int) ([]map[string]interface{}, int, string, bool, error) {
		return cachedProducts, 1, "cached-log-id", true, nil
	}

	// Verify Indexer is NOT called
	indexer.SearchProductsFn = func(ctx context.Context, tenantID, query string, from, size int) ([]map[string]interface{}, int, error) {
		t.Fatal("Indexer should not be called on cache hit")
		return nil, 0, nil
	}

	svc := service.NewSearchService(indexer, cache, analytics, nil)
	res, total, searchLogID, err := svc.Search(context.Background(), "tenant-1", "test-query", 1, 20)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if searchLogID == "" {
		t.Error("Expected searchLogID to be generated and returned, got empty")
	}

	if len(res) != 1 || res[0]["id"] != "p-1" {
		t.Errorf("Expected cached product, got: %v", res)
	}
	if total != 1 {
		t.Errorf("Expected total 1, got %d", total)
	}

	// Wait briefly: Strategy 1 does NOT call analytics repository on Cache Hit
	time.Sleep(50 * time.Millisecond)
	if analytics.CallsCount != 0 {
		t.Errorf("Expected analytics repository to not be called on cache hit, got %d", analytics.CallsCount)
	}
}

func TestSearchService_CacheMiss(t *testing.T) {
	indexer := NewMockProductIndexer()
	cache := &MockProductCache{}
	analytics := &MockAnalyticsRepository{}

	// Setup Cache Miss
	cache.GetCachedSearchFn = func(ctx context.Context, tenantID, query string, page, pageSize int) ([]map[string]interface{}, int, string, bool, error) {
		return nil, 0, "", false, nil
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
	cache.CacheSearchFn = func(ctx context.Context, tenantID, query string, page, pageSize int, data []map[string]interface{}, total int, searchLogID string) error {
		cacheSetCalled = true
		if len(data) != 1 || data[0]["id"] != "p-2" {
			t.Errorf("Expected data indexed to be cached, got %v", data)
		}
		if total != 1 {
			t.Errorf("Expected total 1, got %d", total)
		}
		if searchLogID == "" {
			t.Error("Expected searchLogID to be passed to CacheSearch")
		}
		return nil
	}

	svc := service.NewSearchService(indexer, cache, analytics, nil)
	res, total, _, err := svc.Search(context.Background(), "tenant-1", "test-query", 1, 20)
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
	if analytics.CallsCount != 1 {
		t.Errorf("Expected analytics repository to be called once, got %d", analytics.CallsCount)
	}
}

func TestSearchService_Normalization(t *testing.T) {
	indexer := NewMockProductIndexer()
	cache := &MockProductCache{}
	analytics := &MockAnalyticsRepository{}

	cache.GetCachedSearchFn = func(ctx context.Context, tenantID, query string, page, pageSize int) ([]map[string]interface{}, int, string, bool, error) {
		// Verify query is normalized when checking cache
		if query != "ca phe sua" {
			t.Errorf("Expected query to be normalized as 'ca phe sua', got: '%s'", query)
		}
		return nil, 0, "", false, nil
	}

	indexer.SearchProductsFn = func(ctx context.Context, tenantID, query string, from, size int) ([]map[string]interface{}, int, error) {
		// Verify query is normalized when searching in OpenSearch
		if query != "ca phe sua" {
			t.Errorf("Expected query to be normalized as 'ca phe sua', got: '%s'", query)
		}
		return nil, 0, nil
	}

	svc := service.NewSearchService(indexer, cache, analytics, nil)
	_, _, _, err := svc.Search(context.Background(), "tenant-1", "   Ca   phe   Sua   ", 1, 20)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestSearchService_QueryLengthLimit(t *testing.T) {
	indexer := NewMockProductIndexer()
	cache := &MockProductCache{}
	analytics := &MockAnalyticsRepository{}

	svc := service.NewSearchService(indexer, cache, analytics, nil)
	longQuery := strings.Repeat("a", 101)
	_, _, _, err := svc.Search(context.Background(), "tenant-1", longQuery, 1, 20)
	if err == nil {
		t.Fatal("Expected error for query longer than 100 characters, got nil")
	}
	if !strings.Contains(err.Error(), "exceeds 100 characters") {
		t.Errorf("Expected exceeds length error message, got: %v", err)
	}
}

func TestSearchService_TrackClick(t *testing.T) {
	indexer := NewMockProductIndexer()
	cache := &MockProductCache{}
	analytics := &MockAnalyticsRepository{}

	var saveClickCalled bool
	analytics.SaveClickLogFn = func(ctx context.Context, searchLogID, tenantID, query, productID string, position int) error {
		saveClickCalled = true
		if searchLogID != "log-123" || tenantID != "tenant-1" || query != "coffee" || productID != "prod-456" || position != 2 {
			t.Errorf("Unexpected parameters: searchLogID=%s, tenantID=%s, query=%s, productID=%s, position=%d",
				searchLogID, tenantID, query, productID, position)
		}
		return nil
	}

	svc := service.NewSearchService(indexer, cache, analytics, nil)

	// Test valid case
	err := svc.TrackClick(context.Background(), "tenant-1", "log-123", "prod-456", "coffee", 2)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if !saveClickCalled {
		t.Error("Expected SaveClickLog to be called")
	}

	// Test invalid position
	err = svc.TrackClick(context.Background(), "tenant-1", "log-123", "prod-456", "coffee", 0)
	if err == nil {
		t.Fatal("Expected error for invalid position <= 0, got nil")
	}
}

func TestSearchService_Suggest_CacheHit(t *testing.T) {
	indexer := NewMockProductIndexer()
	cache := &MockProductCache{}
	analytics := &MockAnalyticsRepository{}

	// Setup Cache Hit
	cachedSuggestions := []entity.Suggestion{
		{ID: "p-1", Text: "Bàn phím cơ Akko", Brand: "Akko", Price: 150000},
		{ID: "p-2", Text: "Bàn phím cơ Ajazz", Brand: "Ajazz", Price: 80000},
	}
	cache.GetCachedSuggestionsFn = func(ctx context.Context, tenantID, query string) ([]entity.Suggestion, bool, error) {
		if tenantID != "tenant-1" || query != "ban" {
			t.Errorf("Unexpected cache hit query params: tenantID=%s, query=%s", tenantID, query)
		}
		return cachedSuggestions, true, nil
	}

	// Verify Indexer is NOT called
	indexer.SuggestProductsFn = func(ctx context.Context, tenantID, query string) ([]entity.Suggestion, error) {
		t.Fatal("Indexer should not be called on cache hit")
		return nil, nil
	}

	svc := service.NewSearchService(indexer, cache, analytics, nil)
	res, err := svc.Suggest(context.Background(), "tenant-1", "ban")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(res) != 2 || res[0].Text != "Bàn phím cơ Akko" {
		t.Errorf("Unexpected suggestions: %v", res)
	}
}

func TestSearchService_Suggest_CacheMiss(t *testing.T) {
	indexer := NewMockProductIndexer()
	cache := &MockProductCache{}
	analytics := &MockAnalyticsRepository{}

	// Setup Cache Miss
	cache.GetCachedSuggestionsFn = func(ctx context.Context, tenantID, query string) ([]entity.Suggestion, bool, error) {
		return nil, false, nil
	}

	// Mock Indexer call
	indexedSuggestions := []entity.Suggestion{
		{ID: "p-3", Text: "Logitech G Pro", Brand: "Logitech", Price: 3190000},
		{ID: "p-4", Text: "Logitech MX Master", Brand: "Logitech", Price: 2490000},
	}
	indexer.SuggestProductsFn = func(ctx context.Context, tenantID, query string) ([]entity.Suggestion, error) {
		if tenantID != "tenant-1" || query != "logi" {
			t.Errorf("Unexpected indexer query params: tenantID=%s, query=%s", tenantID, query)
		}
		return indexedSuggestions, nil
	}

	// Mock Cache Set
	var cacheSetCalled bool
	cache.CacheSuggestionsFn = func(ctx context.Context, tenantID, query string, suggestions []entity.Suggestion) error {
		cacheSetCalled = true
		if len(suggestions) != 2 || suggestions[0].Text != "Logitech G Pro" {
			t.Errorf("Unexpected cached suggestions: %v", suggestions)
		}
		return nil
	}

	svc := service.NewSearchService(indexer, cache, analytics, nil)
	res, err := svc.Suggest(context.Background(), "tenant-1", "logi")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(res) != 2 || res[0].Text != "Logitech G Pro" {
		t.Errorf("Unexpected suggestions: %v", res)
	}

	if !cacheSetCalled {
		t.Error("Expected CacheSuggestions to be called")
	}
}

func TestSearchService_Suggest_ShortQuery(t *testing.T) {
	indexer := NewMockProductIndexer()
	cache := &MockProductCache{}
	analytics := &MockAnalyticsRepository{}

	svc := service.NewSearchService(indexer, cache, analytics, nil)

	// Short query (< 2 characters) should immediately return empty results without hitting cache/indexer
	res, err := svc.Suggest(context.Background(), "tenant-1", "a")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(res) != 0 {
		t.Errorf("Expected 0 suggestions for short query, got: %v", res)
	}
}
