package repository

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"search-service/internal/entity"
	"search-service/internal/service"
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
		DoUpdates: clause.AssignmentColumns([]string{"status", "error_message", "retry_count", "updated_at"}),
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
