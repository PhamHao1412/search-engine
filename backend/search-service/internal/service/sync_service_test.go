package service_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"gorm.io/gorm"
	"search-service/internal/entity"
	"search-service/internal/service"
)

// MockSearchRepository implements service.SearchRepository
type MockSearchRepository struct {
	SaveTranslationFn          func(ctx context.Context, t *entity.ProductTranslation) error
	SaveSyncJobFn              func(ctx context.Context, job *entity.SearchSyncJob) error
	GetSyncJobByProductIDFn    func(ctx context.Context, productID string) (*entity.SearchSyncJob, error)
	GetFailedSyncJobsFn        func(ctx context.Context) ([]entity.SearchSyncJob, error)
	GetProductByIDFn           func(ctx context.Context, id string) (*entity.Product, error)
	GetAllProductsByTenantIDFn func(ctx context.Context, tenantID string) ([]entity.Product, error)

	SavedJobs         map[string]*entity.SearchSyncJob
	SavedTranslations []*entity.ProductTranslation
}

func NewMockSearchRepository() *MockSearchRepository {
	return &MockSearchRepository{
		SavedJobs: make(map[string]*entity.SearchSyncJob),
	}
}

func (m *MockSearchRepository) SaveTranslation(ctx context.Context, t *entity.ProductTranslation) error {
	m.SavedTranslations = append(m.SavedTranslations, t)
	if m.SaveTranslationFn != nil {
		return m.SaveTranslationFn(ctx, t)
	}
	return nil
}

func (m *MockSearchRepository) SaveSyncJob(ctx context.Context, job *entity.SearchSyncJob) error {
	m.SavedJobs[job.ProductID] = job
	if m.SaveSyncJobFn != nil {
		return m.SaveSyncJobFn(ctx, job)
	}
	return nil
}

func (m *MockSearchRepository) GetSyncJobByProductID(ctx context.Context, productID string) (*entity.SearchSyncJob, error) {
	if m.GetSyncJobByProductIDFn != nil {
		return m.GetSyncJobByProductIDFn(ctx, productID)
	}
	job, ok := m.SavedJobs[productID]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return job, nil
}

func (m *MockSearchRepository) GetFailedSyncJobs(ctx context.Context) ([]entity.SearchSyncJob, error) {
	if m.GetFailedSyncJobsFn != nil {
		return m.GetFailedSyncJobsFn(ctx)
	}
	var failed []entity.SearchSyncJob
	for _, job := range m.SavedJobs {
		if (job.Status == "failed_translation" || job.Status == "failed_ai") && job.RetryCount < 5 {
			failed = append(failed, *job)
		}
	}
	return failed, nil
}

func (m *MockSearchRepository) GetProductByID(ctx context.Context, id string) (*entity.Product, error) {
	if m.GetProductByIDFn != nil {
		return m.GetProductByIDFn(ctx, id)
	}
	return &entity.Product{
		ID:               id,
		TenantID:         "test-tenant",
		Name:             "Sản phẩm test",
		Description:      "Mô tả sản phẩm test",
		Price:            100.0,
		OriginalLanguage: "vi",
	}, nil
}

func (m *MockSearchRepository) GetAllProductsByTenantID(ctx context.Context, tenantID string) ([]entity.Product, error) {
	if m.GetAllProductsByTenantIDFn != nil {
		return m.GetAllProductsByTenantIDFn(ctx, tenantID)
	}
	return nil, nil
}

// MockProductIndexer implements service.ProductIndexer
type MockProductIndexer struct {
	IndexProductFn   func(ctx context.Context, doc map[string]interface{}, productID string) error
	SearchProductsFn func(ctx context.Context, tenantID, query string, from, size int) ([]map[string]interface{}, int, error)
	IndexedDocs      map[string]map[string]interface{}
}

func NewMockProductIndexer() *MockProductIndexer {
	return &MockProductIndexer{
		IndexedDocs: make(map[string]map[string]interface{}),
	}
}

func (m *MockProductIndexer) IndexProduct(ctx context.Context, doc map[string]interface{}, productID string) error {
	m.IndexedDocs[productID] = doc
	if m.IndexProductFn != nil {
		return m.IndexProductFn(ctx, doc, productID)
	}
	return nil
}

func (m *MockProductIndexer) EnsureIndex(ctx context.Context) {}

func (m *MockProductIndexer) SearchProducts(ctx context.Context, tenantID, query string, from, size int) ([]map[string]interface{}, int, error) {
	if m.SearchProductsFn != nil {
		return m.SearchProductsFn(ctx, tenantID, query, from, size)
	}
	return nil, 0, nil
}

// MockProductCache implements service.ProductCache
type MockProductCache struct {
	CacheProductFn    func(ctx context.Context, tenantID, productID string, data map[string]interface{}) error
	GetCachedSearchFn func(ctx context.Context, tenantID, query string, page, pageSize int) ([]map[string]interface{}, int, bool, error)
	CacheSearchFn     func(ctx context.Context, tenantID, query string, page, pageSize int, data []map[string]interface{}, total int) error
}

func (m *MockProductCache) CacheProduct(ctx context.Context, tenantID, productID string, data map[string]interface{}) error {
	if m.CacheProductFn != nil {
		return m.CacheProductFn(ctx, tenantID, productID, data)
	}
	return nil
}

func (m *MockProductCache) GetCachedSearch(ctx context.Context, tenantID, query string, page, pageSize int) ([]map[string]interface{}, int, bool, error) {
	if m.GetCachedSearchFn != nil {
		return m.GetCachedSearchFn(ctx, tenantID, query, page, pageSize)
	}
	return nil, 0, false, nil
}

func (m *MockProductCache) CacheSearch(ctx context.Context, tenantID, query string, page, pageSize int, data []map[string]interface{}, total int) error {
	if m.CacheSearchFn != nil {
		return m.CacheSearchFn(ctx, tenantID, query, page, pageSize, data, total)
	}
	return nil
}

// MockTranslationService implements service.TranslationService
type MockTranslationService struct {
	TranslateFn func(ctx context.Context, text, targetLang string) (string, error)
}

func (m *MockTranslationService) Translate(ctx context.Context, text, targetLang string) (string, error) {
	if m.TranslateFn != nil {
		return m.TranslateFn(ctx, text, targetLang)
	}
	return text + "_" + targetLang, nil
}

// MockTagGenerator implements service.TagGenerator
type MockTagGenerator struct {
	GenerateSearchTagsFn func(ctx context.Context, name, description string) ([]string, error)
}

func (m *MockTagGenerator) GenerateSearchTags(ctx context.Context, name, description string) ([]string, error) {
	if m.GenerateSearchTagsFn != nil {
		return m.GenerateSearchTagsFn(ctx, name, description)
	}
	return []string{"tag1", "tag2"}, nil
}

func TestSyncProduct_Success(t *testing.T) {
	repo := NewMockSearchRepository()
	indexer := NewMockProductIndexer()
	cache := &MockProductCache{}
	translator := &MockTranslationService{}
	tagsGen := &MockTagGenerator{}

	syncSvc := service.NewSyncService(repo, indexer, cache, translator, tagsGen)

	product := entity.Product{
		ID:               "prod-123",
		TenantID:         "tenant-abc",
		Name:             "Bàn phím cơ",
		Description:      "Bàn phím cơ giá rẻ",
		Price:            50.0,
		OriginalLanguage: "vi",
	}

	err := syncSvc.SyncProduct(context.Background(), product)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify sync job status
	job, ok := repo.SavedJobs[product.ID]
	if !ok {
		t.Fatal("Sync job was not saved")
	}
	if job.Status != "success" {
		t.Errorf("Expected job status to be 'success', got '%s'", job.Status)
	}
	if job.ErrorMessage != nil {
		t.Errorf("Expected nil error message, got: %s", *job.ErrorMessage)
	}

	// Verify indexing content
	doc, ok := indexer.IndexedDocs[product.ID]
	if !ok {
		t.Fatal("Product was not indexed in OpenSearch")
	}
	if doc["product_name_en"] != "Bàn phím cơ_en" {
		t.Errorf("Expected translated EN name, got: %s", doc["product_name_en"])
	}
	if !strings.Contains(doc["search_tags"].(string), "tag1") {
		t.Errorf("Expected search tags, got: %s", doc["search_tags"])
	}
}

func TestSyncProduct_TranslationFailure(t *testing.T) {
	repo := NewMockSearchRepository()
	indexer := NewMockProductIndexer()
	cache := &MockProductCache{}
	translator := &MockTranslationService{
		TranslateFn: func(ctx context.Context, text, targetLang string) (string, error) {
			return "", errors.New("translation service offline")
		},
	}
	tagsGen := &MockTagGenerator{}

	syncSvc := service.NewSyncService(repo, indexer, cache, translator, tagsGen)

	product := entity.Product{
		ID:               "prod-123",
		TenantID:         "tenant-abc",
		Name:             "Bàn phím cơ",
		Description:      "Bàn phím cơ giá rẻ",
		Price:            50.0,
		OriginalLanguage: "vi",
	}

	err := syncSvc.SyncProduct(context.Background(), product)
	// Should NOT return error because business logic dictates eventual consistency for translation errors
	if err != nil {
		t.Fatalf("Expected no block error for translation failure, got: %v", err)
	}

	// Verify sync job status
	job, ok := repo.SavedJobs[product.ID]
	if !ok {
		t.Fatal("Sync job was not saved")
	}
	if job.Status != "failed_translation" {
		t.Errorf("Expected job status to be 'failed_translation', got '%s'", job.Status)
	}
	if job.ErrorMessage == nil || !strings.Contains(*job.ErrorMessage, "translation service offline") {
		t.Errorf("Expected error message containing translation offline, got: %v", job.ErrorMessage)
	}

	// Verify indexing content (should index with original Vietnamese name as fallback)
	doc, ok := indexer.IndexedDocs[product.ID]
	if !ok {
		t.Fatal("Product was not indexed in OpenSearch")
	}
	if doc["product_name_en"] != "Bàn phím cơ" {
		t.Errorf("Expected fallback to original name, got: %s", doc["product_name_en"])
	}
}

func TestSyncProduct_AIFailure(t *testing.T) {
	repo := NewMockSearchRepository()
	indexer := NewMockProductIndexer()
	cache := &MockProductCache{}
	translator := &MockTranslationService{}
	tagsGen := &MockTagGenerator{
		GenerateSearchTagsFn: func(ctx context.Context, name, description string) ([]string, error) {
			return nil, errors.New("openai rate limit")
		},
	}

	syncSvc := service.NewSyncService(repo, indexer, cache, translator, tagsGen)

	product := entity.Product{
		ID:               "prod-123",
		TenantID:         "tenant-abc",
		Name:             "Bàn phím cơ",
		Description:      "Bàn phím cơ giá rẻ",
		Price:            50.0,
		OriginalLanguage: "vi",
	}

	err := syncSvc.SyncProduct(context.Background(), product)
	// Should NOT return error because business logic dictates eventual consistency for AI errors
	if err != nil {
		t.Fatalf("Expected no block error for AI failure, got: %v", err)
	}

	// Verify sync job status
	job, ok := repo.SavedJobs[product.ID]
	if !ok {
		t.Fatal("Sync job was not saved")
	}
	if job.Status != "failed_ai" {
		t.Errorf("Expected job status to be 'failed_ai', got '%s'", job.Status)
	}

	// Verify indexing content (should use fallback default tags)
	doc, ok := indexer.IndexedDocs[product.ID]
	if !ok {
		t.Fatal("Product was not indexed in OpenSearch")
	}
	tagsStr := doc["search_tags"].(string)
	if !strings.Contains(tagsStr, "sảnphẩm") && !strings.Contains(tagsStr, "amaze") {
		t.Errorf("Expected fallback default tags, got: %s", tagsStr)
	}
}

func TestSyncProduct_OpenSearchFailure(t *testing.T) {
	repo := NewMockSearchRepository()
	indexer := &MockProductIndexer{
		IndexedDocs: make(map[string]map[string]interface{}),
		IndexProductFn: func(ctx context.Context, doc map[string]interface{}, productID string) error {
			return errors.New("opensearch connection refused")
		},
	}
	cache := &MockProductCache{}
	translator := &MockTranslationService{}
	tagsGen := &MockTagGenerator{}

	syncSvc := service.NewSyncService(repo, indexer, cache, translator, tagsGen)

	product := entity.Product{
		ID:               "prod-123",
		TenantID:         "tenant-abc",
		Name:             "Bàn phím cơ",
		Description:      "Bàn phím cơ giá rẻ",
		Price:            50.0,
		OriginalLanguage: "vi",
	}

	err := syncSvc.SyncProduct(context.Background(), product)
	// MUST return error to signal Kafka consumer to publish to DLQ
	if err == nil {
		t.Fatal("Expected error for OpenSearch failure, got nil")
	}

	// Verify sync job status
	job, ok := repo.SavedJobs[product.ID]
	if !ok {
		t.Fatal("Sync job was not saved")
	}
	if job.Status != "failed_opensearch" {
		t.Errorf("Expected job status to be 'failed_opensearch', got '%s'", job.Status)
	}
}

func TestReprocessFailedJobs(t *testing.T) {
	repo := NewMockSearchRepository()
	indexer := NewMockProductIndexer()
	cache := &MockProductCache{}
	translator := &MockTranslationService{}
	tagsGen := &MockTagGenerator{}

	// Seed database with a failed job
	failedJob := &entity.SearchSyncJob{
		ID:         "job-1",
		TenantID:   "tenant-abc",
		ProductID:  "prod-123",
		Status:     "failed_translation",
		RetryCount: 1,
	}
	repo.SavedJobs[failedJob.ProductID] = failedJob

	syncSvc := service.NewSyncService(repo, indexer, cache, translator, tagsGen)

	err := syncSvc.ReprocessFailedJobs(context.Background())
	if err != nil {
		t.Fatalf("ReprocessFailedJobs failed: %v", err)
	}

	// Verify the job was retried and updated to success
	job, ok := repo.SavedJobs[failedJob.ProductID]
	if !ok {
		t.Fatal("Sync job missing")
	}
	if job.Status != "success" {
		t.Errorf("Expected job to recover to 'success', got '%s'", job.Status)
	}
	if job.RetryCount != 2 {
		t.Errorf("Expected retry count to increment to 2, got %d", job.RetryCount)
	}
}
