package service_test

import (
	"context"
	"errors"
	"search-service/internal/entity"
	"search-service/internal/service"
	"testing"
)

type MockKeywordAnalyzer struct {
	AnalyzeKeywordsFn func(ctx context.Context, keywords []string, productsContext string) ([]entity.AISuggestion, error)
}

func (m *MockKeywordAnalyzer) AnalyzeKeywords(ctx context.Context, keywords []string, productsContext string) ([]entity.AISuggestion, error) {
	if m.AnalyzeKeywordsFn != nil {
		return m.AnalyzeKeywordsFn(ctx, keywords, productsContext)
	}
	return nil, nil
}

func TestAISuggestionService_GenerateSuggestions_Success(t *testing.T) {
	repo := NewMockSearchRepository()
	analyzer := &MockKeywordAnalyzer{}

	// Setup repository behavior for active tenants
	repo.GetActiveTenantsFn = func(ctx context.Context) ([]string, error) {
		return []string{"tenant-1"}, nil
	}

	// Setup repository behavior for query stats
	repo.GetZeroResultQueriesFn = func(ctx context.Context, tenantID string, limit int) ([]service.QueryStat, error) {
		return []service.QueryStat{
			{Query: "ako", Count: 10},
			{Query: "logitek", Count: 5},
		}, nil
	}

	repo.GetLowCTRQueriesFn = func(ctx context.Context, tenantID string, limit int) ([]service.QueryStat, error) {
		return []service.QueryStat{
			{Query: "chuot", Count: 20},
		}, nil
	}

	// Setup repository behavior for active products context
	repo.GetTenantContextSummaryFn = func(ctx context.Context, tenantID string) (string, error) {
		return "Store Name: Tenant A | Product Categories: Keyboard, Mouse | Brands Sold: Akko, Logitech", nil
	}

	// Verify keywords and products context are formatted correctly
	analyzer.AnalyzeKeywordsFn = func(ctx context.Context, keywords []string, productsContext string) ([]entity.AISuggestion, error) {
		if len(keywords) != 3 {
			t.Errorf("expected 3 keywords, got %d", len(keywords))
		}
		if keywords[0] != "ako" || keywords[1] != "logitek" || keywords[2] != "chuot" {
			t.Errorf("incorrect keyword array content: %v", keywords)
		}
		if !testingContains(productsContext, "Akko") || !testingContains(productsContext, "Logitech") {
			t.Errorf("incorrect products context: %s", productsContext)
		}

		return []entity.AISuggestion{
			{SourceValue: "ako", SuggestedValue: "akko", SuggestionType: "typo", ConfidenceScore: 0.95},
			{SourceValue: "logitek", SuggestedValue: "logitech", SuggestionType: "typo", ConfidenceScore: 0.98},
		}, nil
	}

	var savedSuggestions []*entity.AISuggestion
	repo.SaveAISuggestionFn = func(ctx context.Context, sugg *entity.AISuggestion) error {
		savedSuggestions = append(savedSuggestions, sugg)
		return nil
	}

	aiSvc := service.NewAISuggestionService(repo, analyzer)
	err := aiSvc.GenerateAISuggestions(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(savedSuggestions) != 2 {
		t.Fatalf("expected 2 suggestions saved, got %d", len(savedSuggestions))
	}

	if savedSuggestions[0].SourceValue != "ako" || savedSuggestions[0].SuggestedValue != "akko" {
		t.Errorf("incorrect saved suggestion content at index 0")
	}
	if savedSuggestions[1].SourceValue != "logitek" || savedSuggestions[1].SuggestedValue != "logitech" {
		t.Errorf("incorrect saved suggestion content at index 1")
	}
}

func TestAISuggestionService_GenerateSuggestions_ErrorTenant(t *testing.T) {
	repo := NewMockSearchRepository()
	analyzer := &MockKeywordAnalyzer{}

	repo.GetActiveTenantsFn = func(ctx context.Context) ([]string, error) {
		return nil, errors.New("database query error")
	}

	aiSvc := service.NewAISuggestionService(repo, analyzer)
	err := aiSvc.GenerateAISuggestions(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func testingContains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || stringsContains(s, substr))
}

func stringsContains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
