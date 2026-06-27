package repository

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	"gorm.io/gorm"
	"search-service/internal/entity"
	"search-service/internal/service"
)

type analyticsRepository struct {
	db *gorm.DB
}

// NewAnalyticsRepository creates a new instance of AnalyticsRepository that saves logs directly via GORM
func NewAnalyticsRepository(db *gorm.DB) service.AnalyticsRepository {
	return &analyticsRepository{db: db}
}

func (r *analyticsRepository) SaveSearchLog(ctx context.Context, searchLogID, tenantID, query, normalizedQuery string, resultCount int) error {
	logEntry := &entity.SearchLog{
		ID:              searchLogID,
		TenantID:        tenantID,
		Query:           query,
		NormalizedQuery: normalizedQuery,
		ResultCount:     resultCount,
		SearchedAt:      time.Now(),
	}
	return r.db.WithContext(ctx).Create(logEntry).Error
}

func (r *analyticsRepository) SaveClickLog(ctx context.Context, searchLogID, tenantID, query, productID string, position int) error {
	clickLog := &entity.ClickLog{
		ID:            r.newUUID(),
		TenantID:      tenantID,
		SearchLogID:   searchLogID,
		Query:         query,
		ProductID:     productID,
		ClickPosition: position,
		ClickedAt:     time.Now(),
	}
	return r.db.WithContext(ctx).Create(clickLog).Error
}

func (r *analyticsRepository) newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
