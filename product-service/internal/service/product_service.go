package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	"product-service/internal/entity"
)

type ProductRepository interface {
	CreateProduct(ctx context.Context, p *entity.Product) error
	GetTenant(ctx context.Context, tenantID string) (*entity.Tenant, error)
	CreateTenant(ctx context.Context, t *entity.Tenant) error
}

type EventPublisher interface {
	PublishProductCreated(ctx context.Context, p *entity.Product) error
	Close() error
}

type ProductService interface {
	CreateProduct(ctx context.Context, tenantID string, req entity.Product) (*entity.Product, error)
}

type productService struct {
	repo           ProductRepository
	eventPublisher EventPublisher
}

func NewProductService(repo ProductRepository, ep EventPublisher) ProductService {
	return &productService{
		repo:           repo,
		eventPublisher: ep,
	}
}

func (s *productService) CreateProduct(ctx context.Context, tenantID string, p entity.Product) (*entity.Product, error) {
	// 1. Verify / Auto-create tenant for local testing
	_, err := s.repo.GetTenant(ctx, tenantID)
	if err != nil {
		demoTenant := &entity.Tenant{
			ID:        tenantID,
			Name:      "Demo Tenant " + tenantID[:8],
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if err := s.repo.CreateTenant(ctx, demoTenant); err != nil {
			return nil, fmt.Errorf("failed to auto-create tenant: %w", err)
		}
	}

	// 2. Set default values
	p.ID = s.newUUID()
	p.TenantID = tenantID
	if p.OriginalLanguage == "" {
		p.OriginalLanguage = "vi"
	}
	p.Status = "active"
	p.CreatedAt = time.Now()
	p.UpdatedAt = time.Now()

	// 3. Save to DB
	if err := s.repo.CreateProduct(ctx, &p); err != nil {
		return nil, fmt.Errorf("failed to save product: %w", err)
	}

	// 4. Publish Event
	// Async publish event to avoid blocking HTTP response
	go func() {
		pubCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.eventPublisher.PublishProductCreated(pubCtx, &p); err != nil {
			fmt.Printf("Warning: Failed to publish Kafka event for product %s: %v\n", p.ID, err)
		}
	}()

	return &p, nil
}

func (s *productService) newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
