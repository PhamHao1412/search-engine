package translate

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"search-service/internal/service"
)

type translator struct {
	client *http.Client
}

// NewTranslationService creates a new TranslationService using Google Translate's free GTX API
func NewTranslationService() service.TranslationService {
	return &translator{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Translate translates the given text to targetLang using the free Google Translate API
func (t *translator) Translate(ctx context.Context, text, targetLang string) (string, error) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return "", nil
	}

	// Assuming source language is Vietnamese (vi) since product-service ingests products in Vietnamese
	sourceLang := "vi"
	if targetLang == "vi" {
		sourceLang = "en" // Fallback if target is Vietnamese
	}

	apiURL := fmt.Sprintf(
		"https://translate.googleapis.com/translate_a/single?client=gtx&sl=%s&tl=%s&dt=t&q=%s",
		sourceLang,
		targetLang,
		url.QueryEscape(trimmed),
	)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return "", err
	}

	resp, err := t.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("google translate returned status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var raw []interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return "", err
	}

	if len(raw) > 0 {
		firstLevel, ok := raw[0].([]interface{})
		if !ok {
			return "", fmt.Errorf("invalid google translate response format")
		}

		var translatedParts []string
		for _, part := range firstLevel {
			partArray, ok := part.([]interface{})
			if ok && len(partArray) > 0 {
				if str, ok := partArray[0].(string); ok {
					translatedParts = append(translatedParts, str)
				}
			}
		}

		if len(translatedParts) > 0 {
			return strings.Join(translatedParts, ""), nil
		}
	}

	return "", fmt.Errorf("failed to extract translation from response")
}
