package repository

import (
	"context"

	"gorm.io/gorm"
	"product-service/internal/entity"
	"product-service/internal/service"
)

type productRepository struct {
	db *gorm.DB
}

// NewProductRepository creates a new ProductRepository instance
func NewProductRepository(db *gorm.DB) service.ProductRepository {
	return &productRepository{db: db}
}

func (r *productRepository) CreateProduct(ctx context.Context, p *entity.Product) error {
	return r.db.WithContext(ctx).Create(p).Error
}

func (r *productRepository) GetTenant(ctx context.Context, tenantID string) (*entity.Tenant, error) {
	var t entity.Tenant
	err := r.db.WithContext(ctx).First(&t, "id = ?", tenantID).Error
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *productRepository) CreateTenant(ctx context.Context, t *entity.Tenant) error {
	return r.db.WithContext(ctx).Create(t).Error
}
