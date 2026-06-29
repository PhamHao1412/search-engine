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
	GetSpellcheckRuleFn        func(ctx context.Context, tenantID, typoWord string) (*entity.SpellcheckDictionary, error)
	GetActiveTenantsFn         func(ctx context.Context) ([]string, error)
	GetZeroResultQueriesFn     func(ctx context.Context, tenantID string, limit int) ([]service.QueryStat, error)
	GetLowCTRQueriesFn         func(ctx context.Context, tenantID string, limit int) ([]service.QueryStat, error)
	SaveAISuggestionFn         func(ctx context.Context, sugg *entity.AISuggestion) error
	GetAISuggestionByIDFn      func(ctx context.Context, id string) (*entity.AISuggestion, error)
	UpdateAISuggestionStatusFn func(ctx context.Context, id, status string) error
	SaveSpellcheckDictionaryFn func(ctx context.Context, entry *entity.SpellcheckDictionary) error
	SaveSearchSynonymFn        func(ctx context.Context, entry *entity.SearchSynonym) error
	GetAISuggestionsFn         func(ctx context.Context, tenantID, status, suggestionType string) ([]entity.AISuggestion, error)
	ApproveAISuggestionFn      func(ctx context.Context, tenantID, id string) (*entity.AISuggestion, error)
	GetSpellcheckRulesFn       func(ctx context.Context, tenantID string) ([]entity.SpellcheckDictionary, error)
	GetSearchSynonymsFn        func(ctx context.Context, tenantID string) ([]entity.SearchSynonym, error)
	GetTenantContextSummaryFn  func(ctx context.Context, tenantID string) (string, error)

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

func (m *MockSearchRepository) GetSpellcheckRule(ctx context.Context, tenantID, typoWord string) (*entity.SpellcheckDictionary, error) {
	if m.GetSpellcheckRuleFn != nil {
		return m.GetSpellcheckRuleFn(ctx, tenantID, typoWord)
	}
	return nil, nil
}

func (m *MockSearchRepository) GetActiveTenants(ctx context.Context) ([]string, error) {
	if m.GetActiveTenantsFn != nil {
		return m.GetActiveTenantsFn(ctx)
	}
	return nil, nil
}

func (m *MockSearchRepository) GetZeroResultQueries(ctx context.Context, tenantID string, limit int) ([]service.QueryStat, error) {
	if m.GetZeroResultQueriesFn != nil {
		return m.GetZeroResultQueriesFn(ctx, tenantID, limit)
	}
	return nil, nil
}

func (m *MockSearchRepository) GetLowCTRQueries(ctx context.Context, tenantID string, limit int) ([]service.QueryStat, error) {
	if m.GetLowCTRQueriesFn != nil {
		return m.GetLowCTRQueriesFn(ctx, tenantID, limit)
	}
	return nil, nil
}

func (m *MockSearchRepository) SaveAISuggestion(ctx context.Context, sugg *entity.AISuggestion) error {
	if m.SaveAISuggestionFn != nil {
		return m.SaveAISuggestionFn(ctx, sugg)
	}
	return nil
}

func (m *MockSearchRepository) GetAISuggestionByID(ctx context.Context, id string) (*entity.AISuggestion, error) {
	if m.GetAISuggestionByIDFn != nil {
		return m.GetAISuggestionByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *MockSearchRepository) UpdateAISuggestionStatus(ctx context.Context, id, status string) error {
	if m.UpdateAISuggestionStatusFn != nil {
		return m.UpdateAISuggestionStatusFn(ctx, id, status)
	}
	return nil
}

func (m *MockSearchRepository) SaveSpellcheckDictionary(ctx context.Context, entry *entity.SpellcheckDictionary) error {
	if m.SaveSpellcheckDictionaryFn != nil {
		return m.SaveSpellcheckDictionaryFn(ctx, entry)
	}
	return nil
}

func (m *MockSearchRepository) SaveSearchSynonym(ctx context.Context, entry *entity.SearchSynonym) error {
	if m.SaveSearchSynonymFn != nil {
		return m.SaveSearchSynonymFn(ctx, entry)
	}
	return nil
}

func (m *MockSearchRepository) GetAISuggestions(ctx context.Context, tenantID, status, suggestionType string) ([]entity.AISuggestion, error) {
	if m.GetAISuggestionsFn != nil {
		return m.GetAISuggestionsFn(ctx, tenantID, status, suggestionType)
	}
	return nil, nil
}

func (m *MockSearchRepository) ApproveAISuggestion(ctx context.Context, tenantID, id string) (*entity.AISuggestion, error) {
	if m.ApproveAISuggestionFn != nil {
		return m.ApproveAISuggestionFn(ctx, tenantID, id)
	}
	return nil, nil
}

func (m *MockSearchRepository) GetSpellcheckRules(ctx context.Context, tenantID string) ([]entity.SpellcheckDictionary, error) {
	if m.GetSpellcheckRulesFn != nil {
		return m.GetSpellcheckRulesFn(ctx, tenantID)
	}
	return nil, nil
}

func (m *MockSearchRepository) GetSearchSynonyms(ctx context.Context, tenantID string) ([]entity.SearchSynonym, error) {
	if m.GetSearchSynonymsFn != nil {
		return m.GetSearchSynonymsFn(ctx, tenantID)
	}
	return nil, nil
}

func (m *MockSearchRepository) GetTenantContextSummary(ctx context.Context, tenantID string) (string, error) {
	if m.GetTenantContextSummaryFn != nil {
		return m.GetTenantContextSummaryFn(ctx, tenantID)
	}
	return "", nil
}

// MockProductIndexer implements service.ProductIndexer
type MockProductIndexer struct {
	IndexProductFn    func(ctx context.Context, doc map[string]interface{}, productID string) error
	UpdateProductFn   func(ctx context.Context, doc map[string]interface{}, productID string) error
	SearchProductsFn  func(ctx context.Context, tenantID, query string, from, size int) ([]map[string]interface{}, int, string, error)
	SuggestProductsFn func(ctx context.Context, tenantID, query string) ([]entity.Suggestion, error)
	IndexedDocs       map[string]map[string]interface{}
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

func (m *MockProductIndexer) UpdateProduct(ctx context.Context, doc map[string]interface{}, productID string) error {
	if m.IndexedDocs[productID] == nil {
		m.IndexedDocs[productID] = make(map[string]interface{})
	}
	for k, v := range doc {
		m.IndexedDocs[productID][k] = v
	}
	if m.UpdateProductFn != nil {
		return m.UpdateProductFn(ctx, doc, productID)
	}
	return nil
}

func (m *MockProductIndexer) EnsureIndex(ctx context.Context) {}

func (m *MockProductIndexer) SearchProducts(ctx context.Context, tenantID, query string, from, size int) ([]map[string]interface{}, int, string, error) {
	if m.SearchProductsFn != nil {
		return m.SearchProductsFn(ctx, tenantID, query, from, size)
	}
	return nil, 0, "", nil
}

func (m *MockProductIndexer) SuggestProducts(ctx context.Context, tenantID, query string) ([]entity.Suggestion, error) {
	if m.SuggestProductsFn != nil {
		return m.SuggestProductsFn(ctx, tenantID, query)
	}
	return nil, nil
}

// MockProductCache implements service.ProductCache
type MockProductCache struct {
	CacheProductFn         func(ctx context.Context, tenantID, productID string, data map[string]interface{}) error
	GetCachedSearchFn      func(ctx context.Context, tenantID, query string, page, pageSize int) ([]map[string]interface{}, int, string, bool, error)
	CacheSearchFn          func(ctx context.Context, tenantID, query string, page, pageSize int, data []map[string]interface{}, total int, searchLogID string) error
	GetCachedSuggestionsFn func(ctx context.Context, tenantID, query string) ([]entity.Suggestion, bool, error)
	CacheSuggestionsFn     func(ctx context.Context, tenantID, query string, suggestions []entity.Suggestion) error
	GetCachedSpellcheckFn  func(ctx context.Context, tenantID, typoWord string) (string, bool, error)
	CacheSpellcheckFn      func(ctx context.Context, tenantID, typoWord, correctWord string) error
}

func (m *MockProductCache) CacheProduct(ctx context.Context, tenantID, productID string, data map[string]interface{}) error {
	if m.CacheProductFn != nil {
		return m.CacheProductFn(ctx, tenantID, productID, data)
	}
	return nil
}

func (m *MockProductCache) GetCachedSearch(ctx context.Context, tenantID, query string, page, pageSize int) ([]map[string]interface{}, int, string, bool, error) {
	if m.GetCachedSearchFn != nil {
		return m.GetCachedSearchFn(ctx, tenantID, query, page, pageSize)
	}
	return nil, 0, "", false, nil
}

func (m *MockProductCache) CacheSearch(ctx context.Context, tenantID, query string, page, pageSize int, data []map[string]interface{}, total int, searchLogID string) error {
	if m.CacheSearchFn != nil {
		return m.CacheSearchFn(ctx, tenantID, query, page, pageSize, data, total, searchLogID)
	}
	return nil
}

func (m *MockProductCache) GetCachedSuggestions(ctx context.Context, tenantID, query string) ([]entity.Suggestion, bool, error) {
	if m.GetCachedSuggestionsFn != nil {
		return m.GetCachedSuggestionsFn(ctx, tenantID, query)
	}
	return nil, false, nil
}

func (m *MockProductCache) CacheSuggestions(ctx context.Context, tenantID, query string, suggestions []entity.Suggestion) error {
	if m.CacheSuggestionsFn != nil {
		return m.CacheSuggestionsFn(ctx, tenantID, query, suggestions)
	}
	return nil
}

func (m *MockProductCache) GetCachedSpellcheck(ctx context.Context, tenantID, typoWord string) (string, bool, error) {
	if m.GetCachedSpellcheckFn != nil {
		return m.GetCachedSpellcheckFn(ctx, tenantID, typoWord)
	}
	return "", false, nil
}

func (m *MockProductCache) CacheSpellcheck(ctx context.Context, tenantID, typoWord, correctWord string) error {
	if m.CacheSpellcheckFn != nil {
		return m.CacheSpellcheckFn(ctx, tenantID, typoWord, correctWord)
	}
	return nil
}

func (m *MockProductCache) DeleteSpellcheckCache(ctx context.Context, tenantID, typoWord string) error {
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

func TestSyncProduct_TextHashCheck(t *testing.T) {
	repo := NewMockSearchRepository()
	indexer := NewMockProductIndexer()
	cache := &MockProductCache{}

	// Track calls
	translateCalled := false
	translator := &MockTranslationService{
		TranslateFn: func(ctx context.Context, text, targetLang string) (string, error) {
			translateCalled = true
			return text + "_" + targetLang, nil
		},
	}

	aiCalled := false
	tagsGen := &MockTagGenerator{
		GenerateSearchTagsFn: func(ctx context.Context, name, description string) ([]string, error) {
			aiCalled = true
			return []string{"keyboard"}, nil
		},
	}

	syncSvc := service.NewSyncService(repo, indexer, cache, translator, tagsGen)

	product := entity.Product{
		ID:               "prod-123",
		TenantID:         "tenant-abc",
		Name:             "Bàn phím cơ",
		Description:      "Bàn phím cơ giá rẻ",
		Price:            50.0,
		Inventory:        100,
		OriginalLanguage: "vi",
	}

	// 1st sync (full flow)
	err := syncSvc.SyncProduct(context.Background(), product)
	if err != nil {
		t.Fatalf("1st sync failed: %v", err)
	}

	if !translateCalled || !aiCalled {
		t.Error("Expected translate and AI to be called on first sync")
	}

	// Reset tracking
	translateCalled = false
	aiCalled = false

	// Update non-text fields (inventory and price)
	product.Inventory = 99
	product.Price = 45.0

	// Track UpdateProduct call
	updateCalled := false
	indexer.UpdateProductFn = func(ctx context.Context, doc map[string]interface{}, productID string) error {
		updateCalled = true
		if doc["inventory"] != 99 || doc["price"] != 45.0 {
			t.Errorf("Expected inventory=99 and price=45.0 in update payload, got: %v", doc)
		}
		return nil
	}

	// 2nd sync (should use partial update since text is identical)
	err = syncSvc.SyncProduct(context.Background(), product)
	if err != nil {
		t.Fatalf("2nd sync failed: %v", err)
	}

	if translateCalled || aiCalled {
		t.Error("Expected translate and AI to be skipped on duplicate text hash")
	}

	if !updateCalled {
		t.Error("Expected UpdateProduct to be called instead of IndexProduct")
	}
}
