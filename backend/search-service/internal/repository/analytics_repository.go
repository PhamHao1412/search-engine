package repository

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

func (r *analyticsRepository) GetRawSearchLogs(ctx context.Context, start, end time.Time) ([]entity.SearchLog, error) {
	var logs []entity.SearchLog
	err := r.db.WithContext(ctx).Table("search_svc.search_logs").
		Where("searched_at >= ? AND searched_at < ?", start, end).
		Find(&logs).Error
	return logs, err
}

func (r *analyticsRepository) GetRawClickLogs(ctx context.Context, start, end time.Time) ([]entity.ClickLog, error) {
	var logs []entity.ClickLog
	err := r.db.WithContext(ctx).Table("search_svc.click_logs").
		Where("clicked_at >= ? AND clicked_at < ?", start, end).
		Find(&logs).Error
	return logs, err
}

func (r *analyticsRepository) GetClickLogsWithProductInfo(ctx context.Context, start, end time.Time) ([]entity.ClickLogWithCategory, error) {
	var results []entity.ClickLogWithCategory
	err := r.db.WithContext(ctx).Table("search_svc.click_logs cl").
		Select("cl.tenant_id, cl.query, p.category_id, cat.name as category_name, COUNT(*) as click_count").
		Joins("JOIN product_svc.products p ON cl.product_id = p.id").
		Joins("JOIN product_svc.categories cat ON p.category_id = cat.id").
		Where("cl.clicked_at >= ? AND cl.clicked_at < ?", start, end).
		Group("cl.tenant_id, cl.query, p.category_id, cat.name").
		Scan(&results).Error
	return results, err
}

func (r *analyticsRepository) SaveDailyQueryAnalytics(ctx context.Context, records []entity.DailyQueryAnalytics) error {
	if len(records) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "tenant_id"}, {Name: "query"}, {Name: "date"}},
		DoUpdates: clause.AssignmentColumns([]string{"search_count", "click_count", "zero_result_count", "sum_click_position", "updated_at"}),
	}).Create(&records).Error
}

func (r *analyticsRepository) SaveDailyCategoryAnalytics(ctx context.Context, records []entity.DailyCategoryAnalytics) error {
	if len(records) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "tenant_id"}, {Name: "category_id"}, {Name: "date"}},
		DoUpdates: clause.AssignmentColumns([]string{"search_count", "click_count", "updated_at"}),
	}).Create(&records).Error
}

func (r *analyticsRepository) GetAnalyticsSummary(ctx context.Context, tenantID string, start, end time.Time) (entity.AnalyticsSummary, error) {
	var summary entity.AnalyticsSummary
	err := r.db.WithContext(ctx).Table("search_svc.daily_query_analytics").
		Select("COALESCE(SUM(search_count), 0) as total_searches, COALESCE(SUM(zero_result_count), 0) as zero_result_searches, COALESCE(SUM(click_count), 0) as click_count, COALESCE(SUM(sum_click_position), 0) as sum_click_position").
		Where("tenant_id = ? AND date >= ? AND date <= ?", tenantID, start, end).
		Scan(&summary).Error
	return summary, err
}

func (r *analyticsRepository) GetZeroResultQueries(ctx context.Context, tenantID string, start, end time.Time, limit int) ([]entity.ZeroResultQueryDetail, error) {
	var results []entity.ZeroResultQueryDetail

	err := r.db.WithContext(ctx).Raw(`
		WITH zero_queries AS (
			SELECT query, SUM(zero_result_count) as search_count
			FROM search_svc.daily_query_analytics
			WHERE tenant_id = ? AND date >= ? AND date <= ? AND zero_result_count > 0
			GROUP BY query
			ORDER BY search_count DESC
			LIMIT ?
		)
		SELECT 
			zq.query,
			zq.search_count,
			COALESCE(
				CASE 
					WHEN ai.status = 'pending' THEN 'Chờ duyệt'
					WHEN ai.status = 'approved' THEN 'Đã gợi ý sửa đổi'
					WHEN ai.status = 'rejected' THEN 'Đã bác bỏ'
					ELSE 'Chờ AI quét'
				END,
				'Chờ AI quét'
			) as ai_suggestion_status
		FROM zero_queries zq
		LEFT JOIN (
			SELECT DISTINCT ON (source_value) source_value, status 
			FROM search_svc.ai_suggestions 
			WHERE tenant_id = ? 
			ORDER BY source_value, created_at DESC
		) ai ON ai.source_value = zq.query
	`, tenantID, start, end, limit, tenantID).Scan(&results).Error

	return results, err
}

func (r *analyticsRepository) GetCategoryAnalytics(ctx context.Context, tenantID string, start, end time.Time) ([]entity.CategoryAnalyticsDetail, error) {
	var results []entity.CategoryAnalyticsDetail
	err := r.db.WithContext(ctx).Table("search_svc.daily_category_analytics").
		Select("category_id, category_name, COALESCE(SUM(search_count), 0) as search_count, COALESCE(SUM(click_count), 0) as click_count, "+
			"CASE WHEN SUM(search_count) > 0 THEN ROUND((SUM(click_count)::numeric / SUM(search_count)::numeric) * 100, 1) ELSE 0 END as ctr").
		Where("tenant_id = ? AND date >= ? AND date <= ?", tenantID, start, end).
		Group("category_id, category_name").
		Order("search_count DESC").
		Scan(&results).Error
	return results, err
}

func (r *analyticsRepository) GetSpellcheckRulesCount(ctx context.Context, tenantID string) (int, error) {
	var count int64
	err := r.db.WithContext(ctx).Table("search_svc.spellcheck_dictionary").
		Where("tenant_id = ? AND status = 'active'", tenantID).
		Count(&count).Error
	return int(count), err
}

func (r *analyticsRepository) GetSynonymRulesCount(ctx context.Context, tenantID string) (int, error) {
	var count int64
	err := r.db.WithContext(ctx).Table("search_svc.search_synonyms").
		Where("tenant_id = ? AND status = 'active'", tenantID).
		Count(&count).Error
	return int(count), err
}

func (r *analyticsRepository) DeleteRawLogsOlderThan(ctx context.Context, before time.Time) (int64, error) {
	res := r.db.WithContext(ctx).Exec("DELETE FROM search_svc.search_logs WHERE searched_at < ?", before)
	return res.RowsAffected, res.Error
}
