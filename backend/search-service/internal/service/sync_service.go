package service

import (
	"context"
	"crypto/md5"
	"crypto/rand"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"search-service/internal/entity"

	"gorm.io/gorm"
)

// SearchRepository defines the database interactions for search-service GORM repo
type SearchRepository interface {
	SaveTranslation(ctx context.Context, t *entity.ProductTranslation) error
	SaveSyncJob(ctx context.Context, job *entity.SearchSyncJob) error
	GetSyncJobByProductID(ctx context.Context, productID string) (*entity.SearchSyncJob, error)
	GetFailedSyncJobs(ctx context.Context) ([]entity.SearchSyncJob, error)
	GetProductByID(ctx context.Context, id string) (*entity.Product, error)
	GetAllProductsByTenantID(ctx context.Context, tenantID string) ([]entity.Product, error)
}

// ProductIndexer defines indexing operations for search indexing engine (OpenSearch)
type ProductIndexer interface {
	IndexProduct(ctx context.Context, doc map[string]interface{}, productID string) error
	UpdateProduct(ctx context.Context, doc map[string]interface{}, productID string) error
	EnsureIndex(ctx context.Context)
	SearchProducts(ctx context.Context, tenantID, query string, from, size int) ([]map[string]interface{}, int, error)
	SuggestProducts(ctx context.Context, tenantID, query string) ([]entity.Suggestion, error)
}

type ProductCache interface {
	CacheProduct(ctx context.Context, tenantID, productID string, data map[string]interface{}) error
	GetCachedSearch(ctx context.Context, tenantID, query string, page, pageSize int) ([]map[string]interface{}, int, string, bool, error)
	CacheSearch(ctx context.Context, tenantID, query string, page, pageSize int, data []map[string]interface{}, total int, searchLogID string) error
	GetCachedSuggestions(ctx context.Context, tenantID, query string) ([]entity.Suggestion, bool, error)
	CacheSuggestions(ctx context.Context, tenantID, query string, suggestions []entity.Suggestion) error
}

// TranslationService defines translation operations
type TranslationService interface {
	Translate(ctx context.Context, text, targetLang string) (string, error)
}

// TagGenerator defines search tag generation logic
type TagGenerator interface {
	GenerateSearchTags(ctx context.Context, name, description string) ([]string, error)
}

// SyncService orchestrates syncing incoming product ingestion events
type SyncService interface {
	SyncProduct(ctx context.Context, p entity.Product) error
	ReprocessFailedJobs(ctx context.Context) error
	SyncAllProductsByTenant(ctx context.Context, tenantID string) (int, error)
}

type syncService struct {
	repo         SearchRepository
	indexer      ProductIndexer
	cache        ProductCache
	translator   TranslationService
	tagGenerator TagGenerator
}

// NewSyncService creates a new SyncService instance
func NewSyncService(
	repo SearchRepository,
	indexer ProductIndexer,
	cache ProductCache,
	translator TranslationService,
	tagGenerator TagGenerator,
) SyncService {
	return &syncService{
		repo:         repo,
		indexer:      indexer,
		cache:        cache,
		translator:   translator,
		tagGenerator: tagGenerator,
	}
}

func (s *syncService) SyncProduct(ctx context.Context, product entity.Product) error {
	log.Printf("Syncing product in Search: %s (ID: %s)\n", product.Name, product.ID)

	// Fetch or create the sync job status
	job, err := s.repo.GetSyncJobByProductID(ctx, product.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			job = &entity.SearchSyncJob{
				ID:        s.newUUID(),
				TenantID:  product.TenantID,
				ProductID: product.ID,
				Status:    "pending",
				CreatedAt: time.Now(),
			}
		} else {
			log.Printf("Warning: Failed to fetch sync job status: %v", err)
		}
	}
	job.UpdatedAt = time.Now()

	// Text Hash Check
	currentHash := s.calculateTextHash(product.Name, product.Description)
	textChanged := job.TextHash != currentHash

	var syncStatus = "success"
	var syncErrorMsg string

	var nameEN, descEN, nameTH, descTH string
	var tags []string
	var searchTagsStr string

	if !textChanged && job.Status == "success" {
		log.Printf("[SyncProduct] Text hash unchanged for product %s. Skipping translation/AI calls and performing partial OpenSearch update.\n", product.ID)

		updateDoc := map[string]interface{}{
			"brand":     product.Brand,
			"price":     product.Price,
			"image_url": product.ImageURL,
			"inventory": product.Inventory,
			"featured":  product.Featured,
			"status":    product.Status,
		}

		if opensearchErr := s.indexer.UpdateProduct(ctx, updateDoc, product.ID); opensearchErr != nil {
			log.Printf("Failed to perform partial update in OpenSearch: %v. Falling back to full index.", opensearchErr)
			textChanged = true // Fall back to full index if update fails
		} else {
			// Update the Redis cache as well
			cacheData := map[string]interface{}{
				"product":   product,
				"cached_at": time.Now().Format(time.RFC3339),
			}
			if err := s.cache.CacheProduct(ctx, product.TenantID, product.ID, cacheData); err != nil {
				log.Printf("Failed to cache updated product in Redis: %v", err)
			}

			job.UpdatedAt = time.Now()
			if err := s.repo.SaveSyncJob(ctx, job); err != nil {
				log.Printf("Warning: Failed to save sync job status: %v", err)
			}
			return nil
		}
	}

	// Full indexing flow:
	job.TextHash = currentHash

	// 1. Translate to EN and TH
	if product.OriginalLanguage == "vi" {
		nameEN, err = s.translator.Translate(ctx, product.Name, "en")
		if err != nil {
			log.Printf("Translate EN failed: %v", err)
			syncStatus = "failed_translation"
			syncErrorMsg = fmt.Sprintf("Translate EN failed: %v", err)
			nameEN = product.Name
		} else {
			descEN, _ = s.translator.Translate(ctx, product.Description, "en")
		}

		nameTH, err = s.translator.Translate(ctx, product.Name, "th")
		if err != nil {
			log.Printf("Translate TH failed: %v", err)
			if syncErrorMsg == "" {
				syncStatus = "failed_translation"
				syncErrorMsg = fmt.Sprintf("Translate TH failed: %v", err)
			}
			nameTH = product.Name
		} else {
			descTH, _ = s.translator.Translate(ctx, product.Description, "th")
		}
	} else {
		nameEN = product.Name
		descEN = product.Description
		nameTH = product.Name
		descTH = product.Description
	}

	// 2. Generate search tags via AI
	tags, err = s.tagGenerator.GenerateSearchTags(ctx, product.Name, product.Description)
	if err != nil {
		log.Printf("AI GenerateSearchTags failed: %v", err)
		if syncStatus == "success" {
			syncStatus = "failed_ai"
			syncErrorMsg = fmt.Sprintf("AI Gen tags failed: %v", err)
		}
		tags = []string{"sảnphẩm", "amaze"}
	}
	searchTagsStr = strings.Join(tags, " ")

	// 3. Save translations to Postgres (schema product_svc)
	translationEN := &entity.ProductTranslation{
		ID:                    s.newUUID(),
		TenantID:              product.TenantID,
		ProductID:             product.ID,
		LanguageCode:          "en",
		NameTranslated:        nameEN,
		DescriptionTranslated: descEN,
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
	}
	if err := s.repo.SaveTranslation(ctx, translationEN); err != nil {
		log.Printf("Failed to save EN translation: %v", err)
	}

	translationTH := &entity.ProductTranslation{
		ID:                    s.newUUID(),
		TenantID:              product.TenantID,
		ProductID:             product.ID,
		LanguageCode:          "th",
		NameTranslated:        nameTH,
		DescriptionTranslated: descTH,
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
	}
	if err := s.repo.SaveTranslation(ctx, translationTH); err != nil {
		log.Printf("Failed to save TH translation: %v", err)
	}

	// 4. Index OpenSearch
	var suggestNames []string
	suggestSeen := make(map[string]bool)
	for _, n := range []string{product.Name, nameEN, nameTH} {
		n = strings.TrimSpace(n)
		if n != "" && !suggestSeen[n] {
			suggestSeen[n] = true
			suggestNames = append(suggestNames, n)
		}
	}
	suggestText := strings.Join(suggestNames, " ")

	doc := map[string]interface{}{
		"id":              product.ID,
		"tenant_id":       product.TenantID,
		"product_name_vi": product.Name,
		"product_name_en": nameEN,
		"product_name_th": nameTH,
		"description_vi":  product.Description,
		"description_en":  descEN,
		"description_th":  descTH,
		"brand":           product.Brand,
		"price":           product.Price,
		"image_url":       product.ImageURL,
		"inventory":       product.Inventory,
		"featured":        product.Featured,
		"status":          product.Status,
		"search_tags":     searchTagsStr,
		"suggest":         suggestText,
	}

	var opensearchErr error
	if opensearchErr = s.indexer.IndexProduct(ctx, doc, product.ID); opensearchErr != nil {
		log.Printf("Failed to index product in OpenSearch: %v", opensearchErr)
		syncStatus = "failed_opensearch"
		syncErrorMsg = fmt.Sprintf("OpenSearch index failed: %v", opensearchErr)
	}

	// 5. Cache in Redis
	cacheData := map[string]interface{}{
		"product": product,
		"translations": []map[string]string{
			{"lang": "en", "name": nameEN, "description": descEN},
			{"lang": "th", "name": nameTH, "description": descTH},
		},
		"search_tags": tags,
		"cached_at":   time.Now().Format(time.RFC3339),
	}
	if err := s.cache.CacheProduct(ctx, product.TenantID, product.ID, cacheData); err != nil {
		log.Printf("Failed to cache in Redis: %v", err)
	}

	// Update sync job
	if job != nil {
		job.Status = syncStatus
		if syncErrorMsg != "" {
			job.ErrorMessage = &syncErrorMsg
		} else {
			job.ErrorMessage = nil
		}
		if err := s.repo.SaveSyncJob(ctx, job); err != nil {
			log.Printf("Warning: Failed to save sync job status: %v", err)
		}
	}

	if opensearchErr != nil {
		return fmt.Errorf("failed to index in OpenSearch: %w", opensearchErr)
	}

	log.Printf("Successfully completed search indexing sync for product %s with status %s\n", product.ID, syncStatus)
	return nil
}

func (s *syncService) calculateTextHash(name, description string) string {
	hasher := md5.New()
	hasher.Write([]byte(name + "|" + description))
	return fmt.Sprintf("%x", hasher.Sum(nil))
}

func (s *syncService) ReprocessFailedJobs(ctx context.Context) error {
	log.Println("[Reprocessor] Scanning for failed sync jobs...")
	jobs, err := s.repo.GetFailedSyncJobs(ctx)
	if err != nil {
		return err
	}

	if len(jobs) == 0 {
		return nil
	}

	log.Printf("[Reprocessor] Found %d failed sync jobs to reprocess.\n", len(jobs))
	for _, job := range jobs {
		log.Printf("[Reprocessor] Retrying sync for product %s (Status: %s, Attempt: %d)\n", job.ProductID, job.Status, job.RetryCount+1)

		// Fetch original product details
		product, err := s.repo.GetProductByID(ctx, job.ProductID)
		if err != nil {
			log.Printf("[Reprocessor] Error: Failed to fetch original product details for ID %s: %v. Skipping.\n", job.ProductID, err)
			continue
		}

		// Increment retry count
		job.RetryCount++
		job.UpdatedAt = time.Now()
		if err := s.repo.SaveSyncJob(ctx, &job); err != nil {
			log.Printf("[Reprocessor] Warning: Failed to increment retry count for job %s: %v", job.ID, err)
		}

		// Re-run SyncProduct
		err = s.SyncProduct(ctx, *product)
		if err != nil {
			log.Printf("[Reprocessor] Error: Reprocessing product %s failed: %v\n", job.ProductID, err)
		} else {
			log.Printf("[Reprocessor] Success: Product %s reprocessed successfully and index updated.\n", job.ProductID)
		}
	}
	return nil
}

func (s *syncService) SyncAllProductsByTenant(ctx context.Context, tenantID string) (int, error) {
	log.Printf("[SyncService] Fetching all products from database for tenant %s...\n", tenantID)
	products, err := s.repo.GetAllProductsByTenantID(ctx, tenantID)
	if err != nil {
		return 0, err
	}

	log.Printf("[SyncService] Found %d products to sync.\n", len(products))
	successCount := 0
	for _, product := range products {
		if err := s.SyncProduct(ctx, product); err != nil {
			log.Printf("[SyncService] Failed to sync product %s: %v\n", product.ID, err)
		} else {
			successCount++
		}
	}
	return successCount, nil
}

func (s *syncService) newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
