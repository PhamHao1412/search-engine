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
