package repository

import (
	"context"
	cryptoRand "crypto/rand"
	"fmt"
	"time"

	"search-service/internal/entity"
	"search-service/internal/service"

	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type searchRepository struct {
	db *gorm.DB
}

// NewSearchRepository creates a new SearchRepository instance
func NewSearchRepository(db *gorm.DB) service.SearchRepository {
	return &searchRepository{db: db}
}

func (r *searchRepository) SaveTranslation(ctx context.Context, t *entity.ProductTranslation) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "product_id"}, {Name: "language_code"}},
		DoUpdates: clause.AssignmentColumns([]string{"name_translated", "description_translated", "updated_at"}),
	}).Create(t).Error
}

func (r *searchRepository) SaveSyncJob(ctx context.Context, job *entity.SearchSyncJob) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "product_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"status", "error_message", "retry_count", "text_hash", "updated_at"}),
	}).Create(job).Error
}

func (r *searchRepository) GetSyncJobByProductID(ctx context.Context, productID string) (*entity.SearchSyncJob, error) {
	var job entity.SearchSyncJob
	err := r.db.WithContext(ctx).First(&job, "product_id = ?", productID).Error
	if err != nil {
		return nil, err
	}
	return &job, nil
}

func (r *searchRepository) GetFailedSyncJobs(ctx context.Context) ([]entity.SearchSyncJob, error) {
	var jobs []entity.SearchSyncJob
	err := r.db.WithContext(ctx).Where("status IN ? AND retry_count < ?", []string{"failed_translation", "failed_ai"}, 5).Find(&jobs).Error
	return jobs, err
}

func (r *searchRepository) GetProductByID(ctx context.Context, id string) (*entity.Product, error) {
	var p entity.Product
	err := r.db.WithContext(ctx).First(&p, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *searchRepository) GetAllProductsByTenantID(ctx context.Context, tenantID string) ([]entity.Product, error) {
	var products []entity.Product
	err := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID).Find(&products).Error
	return products, err
}

func (r *searchRepository) GetSpellcheckRule(ctx context.Context, tenantID, typoWord string) (*entity.SpellcheckDictionary, error) {
	var rule entity.SpellcheckDictionary
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND typo_word = ? AND status = 'active'", tenantID, typoWord).
		First(&rule).Error
	if err != nil {
		return nil, err
	}
	return &rule, nil
}

func (r *searchRepository) GetActiveTenants(ctx context.Context) ([]string, error) {
	var tenants []string
	err := r.db.WithContext(ctx).Table("search_svc.search_logs").Select("DISTINCT tenant_id").Find(&tenants).Error
	return tenants, err
}

func (r *searchRepository) GetZeroResultQueries(ctx context.Context, tenantID string, limit int) ([]service.QueryStat, error) {
	var stats []service.QueryStat
	err := r.db.WithContext(ctx).Table("search_svc.search_logs").
		Select("normalized_query as query, COUNT(*) as count").
		Where("tenant_id::text = ? AND result_count = ? AND normalized_query != '' AND normalized_query IS NOT NULL AND normalized_query NOT IN (SELECT source_value FROM search_svc.ai_suggestions WHERE tenant_id = ?)", tenantID, 0, tenantID).
		Group("normalized_query").
		Order("count DESC").
		Limit(limit).
		Scan(&stats).Error
	return stats, err
}

func (r *searchRepository) GetLowCTRQueries(ctx context.Context, tenantID string, limit int) ([]service.QueryStat, error) {
	var stats []service.QueryStat
	err := r.db.WithContext(ctx).
		Table("search_svc.search_logs sl").
		Select("sl.normalized_query as query, COUNT(sl.id) as count").
		Joins("LEFT JOIN search_svc.click_logs cl ON sl.id = cl.search_log_id").
		Where("sl.tenant_id::text = ? AND sl.result_count > 0 AND sl.normalized_query != '' AND sl.normalized_query IS NOT NULL AND sl.normalized_query NOT IN (SELECT source_value FROM search_svc.ai_suggestions WHERE tenant_id = ?)", tenantID, tenantID).
		Group("sl.normalized_query").
		Having("COUNT(sl.id) >= ? AND (COUNT(cl.id) * 100.0 / COUNT(sl.id)) < ?", 3, 5.0).
		Order("count DESC").
		Limit(limit).
		Scan(&stats).Error
	return stats, err
}

func (r *searchRepository) SaveAISuggestion(ctx context.Context, sugg *entity.AISuggestion) error {
	var count int64
	err := r.db.WithContext(ctx).Table("search_svc.ai_suggestions").
		Where("tenant_id = ? AND suggestion_type = ? AND source_value = ? AND suggested_value = ?",
			sugg.TenantID, sugg.SuggestionType, sugg.SourceValue, sugg.SuggestedValue).
		Count(&count).Error
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	if sugg.ID == "" {
		sugg.ID = r.newUUID()
	}
	return r.db.WithContext(ctx).Create(sugg).Error
}

func (r *searchRepository) GetAISuggestionByID(ctx context.Context, id string) (*entity.AISuggestion, error) {
	var sugg entity.AISuggestion
	err := r.db.WithContext(ctx).First(&sugg, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &sugg, nil
}

func (r *searchRepository) UpdateAISuggestionStatus(ctx context.Context, id, status string) error {
	return r.db.WithContext(ctx).Table("search_svc.ai_suggestions").Where("id = ?", id).Update("status", status).Error
}

func (r *searchRepository) SaveSpellcheckDictionary(ctx context.Context, entry *entity.SpellcheckDictionary) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "tenant_id"}, {Name: "typo_word"}, {Name: "correct_word"}},
		DoUpdates: clause.AssignmentColumns([]string{"status", "updated_at"}),
	}).Create(entry).Error
}

func (r *searchRepository) SaveSearchSynonym(ctx context.Context, entry *entity.SearchSynonym) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "tenant_id"}, {Name: "keyword"}, {Name: "synonym"}},
		DoUpdates: clause.AssignmentColumns([]string{"status", "updated_at"}),
	}).Create(entry).Error
}

func (r *searchRepository) GetAISuggestions(ctx context.Context, params service.GetAISuggestionsParams) ([]entity.AISuggestion, int64, error) {
	var list []entity.AISuggestion
	var total int64

	q := r.db.WithContext(ctx).Table("search_svc.ai_suggestions").Where("tenant_id = ?", params.TenantID)
	if params.Status != "" {
		q = q.Where("status = ?", params.Status)
	}
	if params.SuggestionType != "" {
		q = q.Where("suggestion_type = ?", params.SuggestionType)
	}
	if params.Search != "" {
		searchQuery := "%" + strings.ToLower(params.Search) + "%"
		q = q.Where("LOWER(source_value) LIKE ? OR LOWER(suggested_value) LIKE ?", searchQuery, searchQuery)
	}

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (params.Page - 1) * params.PageSize
	err := q.Order("created_at DESC, id DESC").Offset(offset).Limit(params.PageSize).Find(&list).Error
	return list, total, err
}

func (r *searchRepository) ApproveAISuggestion(ctx context.Context, tenantID, id string) (*entity.AISuggestion, error) {
	var sugg entity.AISuggestion
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Table("search_svc.ai_suggestions").Where("id = ? AND tenant_id = ?", id, tenantID).First(&sugg).Error; err != nil {
			return err
		}
		if sugg.Status != "pending" {
			return fmt.Errorf("suggestion is not pending")
		}

		sugg.Status = "approved"
		if err := tx.Table("search_svc.ai_suggestions").Where("id = ?", id).Update("status", "approved").Error; err != nil {
			return err
		}

		if sugg.SuggestionType == "typo" {
			entry := &entity.SpellcheckDictionary{
				ID:          r.newUUID(),
				TenantID:    tenantID,
				TypoWord:    sugg.SourceValue,
				CorrectWord: sugg.SuggestedValue,
				Status:      "active",
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			}
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "tenant_id"}, {Name: "typo_word"}, {Name: "correct_word"}},
				DoUpdates: clause.AssignmentColumns([]string{"status", "updated_at"}),
			}).Create(entry).Error; err != nil {
				return err
			}
		} else if sugg.SuggestionType == "synonym" {
			entry := &entity.SearchSynonym{
				ID:        r.newUUID(),
				TenantID:  tenantID,
				Keyword:   sugg.SourceValue,
				Synonym:   sugg.SuggestedValue,
				Status:    "active",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "tenant_id"}, {Name: "keyword"}, {Name: "synonym"}},
				DoUpdates: clause.AssignmentColumns([]string{"status", "updated_at"}),
			}).Create(entry).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &sugg, nil
}

func (r *searchRepository) newUUID() string {
	b := make([]byte, 16)
	_, _ = cryptoRand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

func (r *searchRepository) GetSpellcheckRules(ctx context.Context, tenantID string) ([]entity.SpellcheckDictionary, error) {
	var rules []entity.SpellcheckDictionary
	err := r.db.WithContext(ctx).Table("search_svc.spellcheck_dictionary").Where("tenant_id = ? AND status = 'active'", tenantID).Order("updated_at DESC").Find(&rules).Error
	return rules, err
}

func (r *searchRepository) GetSearchSynonyms(ctx context.Context, tenantID string) ([]entity.SearchSynonym, error) {
	var rules []entity.SearchSynonym
	err := r.db.WithContext(ctx).Table("search_svc.search_synonyms").Where("tenant_id = ? AND status = 'active'", tenantID).Order("updated_at DESC").Find(&rules).Error
	return rules, err
}

func (r *searchRepository) GetTenantContextSummary(ctx context.Context, tenantID string) (string, error) {
	var tenantName string
	err := r.db.WithContext(ctx).Table("product_svc.tenants").
		Select("name").Where("id = ?", tenantID).
		Row().Scan(&tenantName)
	if err != nil {
		tenantName = "Unknown"
	}

	var businessDomain string
	_ = r.db.WithContext(ctx).Table("product_svc.tenant_configs").
		Select("config_value").
		Where("tenant_id = ? AND config_key = 'business_domain'", tenantID).
		Row().Scan(&businessDomain)

	var categories []string
	_ = r.db.WithContext(ctx).Table("product_svc.categories").
		Where("tenant_id = ?", tenantID).
		Pluck("name", &categories)

	var brands []string
	_ = r.db.WithContext(ctx).Table("product_svc.products").
		Where("tenant_id = ? AND status = 'active' AND brand IS NOT NULL AND brand != ''", tenantID).
		Distinct("brand").
		Pluck("brand", &brands)

	summaryParts := []string{
		fmt.Sprintf("Store Name: %s", tenantName),
	}
	if businessDomain != "" {
		summaryParts = append(summaryParts, fmt.Sprintf("Business Domain: %s", businessDomain))
	}
	if len(categories) > 0 {
		summaryParts = append(summaryParts, fmt.Sprintf("Product Categories: %s", strings.Join(categories, ", ")))
	}
	if len(brands) > 0 {
		summaryParts = append(summaryParts, fmt.Sprintf("Brands Sold: %s", strings.Join(brands, ", ")))
	}

	return strings.Join(summaryParts, " | "), nil
}

func (r *searchRepository) GetAllTenants(ctx context.Context) ([]entity.Tenant, error) {
	var list []entity.Tenant
	err := r.db.WithContext(ctx).Order("name ASC").Find(&list).Error
	return list, err
}

func (r *searchRepository) DeleteSpellcheckRule(ctx context.Context, tenantID, id string) error {
	return r.db.WithContext(ctx).Table("search_svc.spellcheck_dictionary").
		Where("tenant_id = ? AND id = ?", tenantID, id).
		Delete(&entity.SpellcheckDictionary{}).Error
}

func (r *searchRepository) DeleteSearchSynonym(ctx context.Context, tenantID, id string) error {
	return r.db.WithContext(ctx).Table("search_svc.search_synonyms").
		Where("tenant_id = ? AND id = ?", tenantID, id).
		Delete(&entity.SearchSynonym{}).Error
}
