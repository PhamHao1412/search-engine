package service

import (
	"context"
	"fmt"
	"log"
)

type AISuggestionService interface {
	GenerateAISuggestions(ctx context.Context) error
}

type aiSuggestionService struct {
	repo     SearchRepository
	analyzer KeywordAnalyzer
}

func NewAISuggestionService(repo SearchRepository, analyzer KeywordAnalyzer) AISuggestionService {
	return &aiSuggestionService{
		repo:     repo,
		analyzer: analyzer,
	}
}

func (s *aiSuggestionService) GenerateAISuggestions(ctx context.Context) error {
	// Fetch active tenants from logs
	tenants, err := s.repo.GetActiveTenants(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch active tenants: %w", err)
	}

	for _, tenantID := range tenants {
		// Fetch top 50 zero-result queries
		zeroQueries, err := s.repo.GetZeroResultQueries(ctx, tenantID, 100)
		if err != nil {
			log.Printf("[AISuggestionWorker] Failed to fetch zero-result queries for tenant %s: %v", tenantID, err)
			continue
		}

		// Fetch top 50 low-CTR queries
		lowCTRQueries, err := s.repo.GetLowCTRQueries(ctx, tenantID, 50)
		if err != nil {
			log.Printf("[AISuggestionWorker] Failed to fetch low-CTR queries for tenant %s: %v", tenantID, err)
			continue
		}

		// Aggregate queries to process
		queryMap := make(map[string]bool)
		var keywords []string
		for _, q := range zeroQueries {
			if len(keywords) < 100 && !queryMap[q.Query] {
				queryMap[q.Query] = true
				keywords = append(keywords, q.Query)
			}
		}
		for _, q := range lowCTRQueries {
			if len(keywords) < 100 && !queryMap[q.Query] {
				queryMap[q.Query] = true
				keywords = append(keywords, q.Query)
			}
		}

		if len(keywords) == 0 {
			continue
		}

		// Fetch tenant catalog summary context for the LLM
		tenantContext, err := s.repo.GetTenantContextSummary(ctx, tenantID)
		if err != nil {
			log.Printf("[AISuggestionWorker] Failed to fetch context summary for tenant %s: %v", tenantID, err)
			continue
		}

		// Call OpenAI to analyze the keywords
		suggestions, err := s.analyzer.AnalyzeKeywords(ctx, keywords, tenantContext)
		if err != nil {
			log.Printf("[AISuggestionWorker] Failed to analyze keywords for tenant %s: %v", tenantID, err)
			continue
		}

		// Save suggestions into DB
		for _, sugg := range suggestions {
			sugg.TenantID = tenantID
			sugg.Status = "pending"
			if err := s.repo.SaveAISuggestion(ctx, &sugg); err != nil {
				log.Printf("[AISuggestionWorker] Failed to save suggestion for tenant %s (from '%s' to '%s'): %v",
					tenantID, sugg.SourceValue, sugg.SuggestedValue, err)
			}
		}
	}

	return nil
}
