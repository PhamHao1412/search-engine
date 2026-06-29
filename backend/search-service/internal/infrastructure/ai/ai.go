package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"search-service/internal/entity"
	"search-service/internal/service"
)

type aiService struct {
	apiKey string
	model  string
	client *http.Client
}

// NewTagGenerator creates a new TagGenerator using OpenAI API
func NewTagGenerator(apiKey, model string) service.TagGenerator {
	if model == "" {
		model = "gpt-4o-mini"
	}
	return &aiService{
		apiKey: apiKey,
		model:  model,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (gw *aiService) GenerateSearchTags(ctx context.Context, name, description string) ([]string, error) {
	if gw.apiKey == "" || strings.HasPrefix(gw.apiKey, "YOUR_") {
		// Mock Tag Generator for local testing when API key is missing
		words := strings.Fields(strings.ToLower(name + " " + description))
		tagsMap := make(map[string]bool)
		stopwords := map[string]bool{"và": true, "của": true, "cho": true, "tại": true, "trong": true, "a": true, "an": true, "the": true, "is": true, "with": true, "for": true}

		for _, w := range words {
			w = strings.Trim(w, ",.?!;:\"'()[]{}*&^%$#@-_")
			if len(w) > 2 && !stopwords[w] {
				tagsMap[w] = true
			}
			if len(tagsMap) >= 5 {
				break
			}
		}

		var tags []string
		for t := range tagsMap {
			tags = append(tags, t)
		}
		return tags, nil
	}

	url := "https://api.openai.com/v1/chat/completions"
	prompt := fmt.Sprintf("Extract 5 relevant search keywords/tags from the product name: '%s' and description: '%s'. Return them as a simple comma-separated list. Do not output anything else.", name, description)

	reqBody, _ := json.Marshal(map[string]interface{}{
		"model": gw.model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"temperature": 0.2,
	})

	var resp *http.Response
	var body []byte
	maxAttempts := 3
	backoff := 500 * time.Millisecond

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBody))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", gw.apiKey))

		resp, err = gw.client.Do(req)
		if err != nil {
			if attempt == maxAttempts {
				return nil, err
			}
			log.Printf("[AIService] Attempt %d failed with network error: %v. Retrying in %v...", attempt, err, backoff)
			time.Sleep(backoff)
			backoff *= 2
			continue
		}

		body, err = io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			if attempt == maxAttempts {
				return nil, err
			}
			log.Printf("[AIService] Attempt %d failed to read response body: %v. Retrying in %v...", attempt, err, backoff)
			time.Sleep(backoff)
			backoff *= 2
			continue
		}

		// Retry on 503 (Service Unavailable) or 429 (Too Many Requests / Rate limit)
		if resp.StatusCode == http.StatusServiceUnavailable || resp.StatusCode == http.StatusTooManyRequests {
			if attempt < maxAttempts {
				log.Printf("[AIService] Attempt %d failed with status %d (Temporary Error). Retrying in %v...", attempt, resp.StatusCode, backoff)
				time.Sleep(backoff)
				backoff *= 2
				continue
			}
		}

		if resp.StatusCode != http.StatusOK {
			log.Printf("[AIService] OpenAI API error (status %d): %s", resp.StatusCode, string(body))
			return nil, fmt.Errorf("openai api returned status %d: %s", resp.StatusCode, string(body))
		}

		break
	}

	var chatResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return nil, err
	}

	if len(chatResp.Choices) > 0 {
		content := chatResp.Choices[0].Message.Content
		rawTags := strings.Split(content, ",")
		var tags []string
		for _, t := range rawTags {
			t = strings.TrimSpace(t)
			if t != "" {
				tags = append(tags, strings.ToLower(t))
			}
		}
		return tags, nil
	}
	return nil, fmt.Errorf("empty suggestions from OpenAI response")
}

func (gw *aiService) AnalyzeKeywords(ctx context.Context, keywords []string, tenantContext string) ([]entity.AISuggestion, error) {
	if len(keywords) == 0 {
		return nil, nil
	}
	if gw.apiKey == "" {
		return nil, fmt.Errorf("OpenAI API key is not configured")
	}

	url := "https://api.openai.com/v1/chat/completions"

	prompt := fmt.Sprintf(`You are an AI search dictionary optimizer. You are given a list of search queries that failed or had very low engagement in an e-commerce store, along with a context list of products currently sold in the store.
Analyze these queries and suggest corrections: either spelling corrections (typos) or synonyms.
Do not suggest corrections for queries that are correct or make no sense.

Guidelines to distinguish "suggestion_type":
- "typo": Use this ONLY when the original query contains obvious spelling mistakes, character slips, missing letters, or missing tone marks/accents in Vietnamese (e.g., "chuot" -> "chuột", "bàn phim" -> "bàn phím", "ako" -> "akko", "logitek" -> "logitech"). The source and target are meant to be the exact same word, but the source is misspelled.
- "synonym": Use this when the original query consists of correctly spelled words, but is an alternative term, synonym, translation, abbreviation, or related category for the products sold (e.g., "chuột gaming" -> "chuột chơi game", "keyboard" -> "bàn phím", "bàn phím bluetooth" -> "bàn phím không dây"). The source and target are different words/phrases that share the same meaning.

Failed Search Queries:
%s

Products Context:
%s

Your response must be a JSON object with a single field "suggestions" containing an array of suggestion objects.
Each suggestion object must have the following fields:
- "suggestion_type": Either "typo" or "synonym"
- "source_value": The original search query
- "suggested_value": The proposed correct search query or synonym
- "confidence_score": A decimal number between 0.0 and 1.0 representing your confidence
- "reason": A short explanation in English of why you made this suggestion`, strings.Join(keywords, ", "), tenantContext)

	reqBody, _ := json.Marshal(map[string]interface{}{
		"model": gw.model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"response_format": map[string]string{"type": "json_object"},
		"temperature":     0.2,
	})

	var resp *http.Response
	var body []byte
	maxAttempts := 3
	backoff := 500 * time.Millisecond

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBody))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", gw.apiKey))

		resp, err = gw.client.Do(req)
		if err != nil {
			if attempt == maxAttempts {
				return nil, err
			}
			log.Printf("[AIService] Attempt %d failed with network error: %v. Retrying in %v...", attempt, err, backoff)
			time.Sleep(backoff)
			backoff *= 2
			continue
		}

		body, err = io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			if attempt == maxAttempts {
				return nil, err
			}
			log.Printf("[AIService] Attempt %d failed to read response body: %v. Retrying in %v...", attempt, err, backoff)
			time.Sleep(backoff)
			backoff *= 2
			continue
		}

		if resp.StatusCode == http.StatusServiceUnavailable || resp.StatusCode == http.StatusTooManyRequests {
			if attempt < maxAttempts {
				log.Printf("[AIService] Attempt %d failed with status %d. Retrying in %v...", attempt, resp.StatusCode, backoff)
				time.Sleep(backoff)
				backoff *= 2
				continue
			}
		}

		if resp.StatusCode != http.StatusOK {
			log.Printf("[AIService] OpenAI API error (status %d): %s", resp.StatusCode, string(body))
			return nil, fmt.Errorf("openai api returned status %d: %s", resp.StatusCode, string(body))
		}

		break
	}

	var chatResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return nil, err
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("empty response from OpenAI")
	}

	var suggestionsContainer struct {
		Suggestions []struct {
			SuggestionType  string  `json:"suggestion_type"`
			SourceValue     string  `json:"source_value"`
			SuggestedValue  string  `json:"suggested_value"`
			ConfidenceScore float64 `json:"confidence_score"`
			Reason          string  `json:"reason"`
		} `json:"suggestions"`
	}

	if err := json.Unmarshal([]byte(chatResp.Choices[0].Message.Content), &suggestionsContainer); err != nil {
		return nil, err
	}

	var suggestions []entity.AISuggestion
	for _, s := range suggestionsContainer.Suggestions {
		suggestions = append(suggestions, entity.AISuggestion{
			SuggestionType:  s.SuggestionType,
			SourceValue:     s.SourceValue,
			SuggestedValue:  s.SuggestedValue,
			ConfidenceScore: s.ConfidenceScore,
			Status:          "pending",
			CreatedAt:       time.Now(),
		})
	}

	return suggestions, nil
}
