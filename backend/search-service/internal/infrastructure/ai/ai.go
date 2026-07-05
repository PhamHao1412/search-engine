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

	prompt := fmt.Sprintf(`You are an AI search dictionary optimizer. You are given a list of search queries that failed or had very low engagement in an e-commerce store, along with a context describing the products, brands, categories, and business domain of the store.

Analyze these queries and suggest corrections: either spelling corrections (typos) or synonyms.

Do not suggest corrections for queries that are already correct, too ambiguous, unrelated to shopping, or have insufficient evidence.

Guidelines to distinguish "suggestion_type":

* "typo": Use this when the source query and suggested value refer to the same intended entity, differing by spelling mistakes, missing/extra/transposed characters, keyboard slips, missing accents/diacritics, OR multiple compounded character-level errors (e.g. dropped letter + swapped letter). Typos are often heavily distorted — do not require near-identical spelling. Judge by whether a human shopper typing quickly and inaccurately would plausibly produce this string while intending the target entity.
* "synonym": Use this when the source query is an abbreviation, alias, shortened form, alternative name, translation, common nickname, product family, or related shopping term that refers to the same shopping intent.

Important rules:

1. Tenant Context Priority:
   Before suggesting any correction, first determine whether the query could refer to any shopping-related entity represented by the Tenant Context, including brands, product names, product families, product lines, categories, or common shopping terms.

2. Canonical Entity Names:
   When correcting queries that match or closely resemble entities represented in the Tenant Context, always suggest the canonical entity name.
   If the exact entity is explicitly listed in the Tenant Context, preserve its exact spelling and capitalization.
   If the exact entity is not explicitly listed but is a well-known product family strongly associated with a supported brand or category in the Tenant Context (e.g. a brand sells phones, and the query resembles a well-known phone model from that brand), you should suggest that canonical product family name. Do not withhold a suggestion merely because the exact product name string is absent from the Tenant Context — brand/category presence is sufficient grounding.

3. Partial Match Handling:
   Queries may be abbreviations, prefixes, shortened forms, incomplete words, truncated searches, or partially typed entities. Treat these as valid evidence when there is a strong and unambiguous shopping intent.

4. Correction Priority:
   If multiple corrections are possible, always prefer the one most closely related to the Tenant Context.
   If there is no direct Tenant Context match, you may infer a well-known shopping entity only when:
   - it clearly belongs to a supported brand or category,
   - the typo similarity is reasonably high (moderate character-level distortion is acceptable, not just single-character edits),
   - the intended entity is commonly recognized by shoppers,
   - and there is little ambiguity.
   Otherwise, ignore the query.

5. Strict Tenant Context Relevance:
   Do NOT generate suggestions for queries unrelated to the store's business domain.
   Ignore random words, unrelated brands, unrelated products, meaningless text, or queries whose intended meaning cannot be determined with reasonable confidence.

6. Evidence Requirement:
   Generate a suggestion whenever there is reasonable evidence that the intended search is a shopping-related entity relevant to the Tenant Context, even if the query is significantly distorted. Only withhold a suggestion when two or more distinct entities are similarly plausible interpretations (genuine ambiguity), or when the query has no plausible connection to the store's domain at all.

7. Existing Term Protection:
   Do not generate a suggestion if the query already exactly matches a valid brand, category, product, product family, or shopping term in the Tenant Context.

8. Confidence Score:
   Use higher confidence when the corrected entity explicitly exists in the Tenant Context.
   Use moderate confidence when the correction is inferred from a well-known product family associated with a supported brand, or when the query is heavily distorted but the intended entity is still clearly identifiable.
   Use low confidence only when the evidence is weaker but still sufficiently reliable to be worth surfacing.

9. Output Rules:
   Always return valid JSON.
   If no suggestions should be generated, return:

   {
     "suggestions": []
   }

Examples (for calibration only, not part of the actual query list):
- Query "ihpeo" in a store selling Apple products → suggest {"suggestion_type": "typo", "source_value": "ihpeo", "suggested_value": "iPhone", "confidence_score": 0.75, "reason": "Heavily distorted spelling of iPhone, a well-known Apple product family; store sells Apple."}
- Query "tainghe" in a store with category "Tai nghe" → suggest {"suggestion_type": "typo", "source_value": "tainghe", "suggested_value": "Tai nghe", "confidence_score": 0.9, "reason": "Missing space, matches existing category exactly."}
- Query "503" or "admin" → no suggestion, not shopping-related.

Failed Search Queries:
%s

Tenant Context:
%s

Your response must be a JSON object with a single field "suggestions" containing an array of suggestion objects.

Each suggestion object must have the following fields:

* "suggestion_type": Either "typo" or "synonym"
* "source_value": The original search query
* "suggested_value": The proposed corrected query or synonym
* "confidence_score": A decimal number between 0.0 and 1.0 representing your confidence
* "reason": A short explanation in English describing why the suggestion was generated
`, strings.Join(keywords, ", "), tenantContext)

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
