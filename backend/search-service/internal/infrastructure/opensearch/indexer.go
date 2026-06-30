package opensearch

import (
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

	opensearchgo "github.com/opensearch-project/opensearch-go/v2"
	"github.com/opensearch-project/opensearch-go/v2/opensearchapi"
)

type opensearchIndexer struct {
	client *opensearchgo.Client
}

// NewOpenSearchIndexer creates a new ProductIndexer using OpenSearch
func NewOpenSearchIndexer(client *opensearchgo.Client) service.ProductIndexer {
	return &opensearchIndexer{client: client}
}

func (idx *opensearchIndexer) IndexProduct(ctx context.Context, doc map[string]interface{}, productID string) error {
	docJSON, err := json.Marshal(doc)
	if err != nil {
		return err
	}

	req := opensearchapi.IndexRequest{
		Index:      "products",
		DocumentID: productID,
		Body:       strings.NewReader(string(docJSON)),
		Refresh:    "true",
	}

	res, err := req.Do(ctx, idx.client)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		bodyBytes, _ := io.ReadAll(res.Body)
		return fmt.Errorf("opensearch error: %s", string(bodyBytes))
	}
	return nil
}

func (idx *opensearchIndexer) UpdateProduct(ctx context.Context, doc map[string]interface{}, productID string) error {
	updateDoc := map[string]interface{}{
		"doc": doc,
	}
	docJSON, err := json.Marshal(updateDoc)
	if err != nil {
		return err
	}

	req := opensearchapi.UpdateRequest{
		Index:      "products",
		DocumentID: productID,
		Body:       strings.NewReader(string(docJSON)),
		Refresh:    "true",
	}

	res, err := req.Do(ctx, idx.client)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		bodyBytes, _ := io.ReadAll(res.Body)
		return fmt.Errorf("opensearch update error: %s", string(bodyBytes))
	}
	return nil
}

func (idx *opensearchIndexer) EnsureIndex(ctx context.Context) {
	indexName := "products_v1"
	aliasName := "products"

	// Wait for OpenSearch to be online (up to 20 seconds)
	var res *opensearchapi.Response
	var err error
	for i := 0; i < 10; i++ {
		existsReq := opensearchapi.CatIndicesRequest{Index: []string{indexName}}
		res, err = existsReq.Do(ctx, idx.client)
		if err == nil {
			break
		}
		log.Printf("[OpenSearch] Waiting for OpenSearch to be online... (attempt %d/10)", i+1)
		time.Sleep(2 * time.Second)
	}

	if err != nil {
		log.Printf("[OpenSearch] CRITICAL: OpenSearch is offline after 20 seconds, skipping index check: %v", err)
		return
	}

	defer res.Body.Close()
	if res.StatusCode == http.StatusOK {
		log.Printf("[OpenSearch] Index %s already exists.", indexName)
		return
	}

	// Double check if a concrete index named aliasName already exists
	checkAliasReq := opensearchapi.CatIndicesRequest{Index: []string{aliasName}}
	aliasRes, checkErr := checkAliasReq.Do(ctx, idx.client)
	if checkErr == nil {
		defer aliasRes.Body.Close()
		if aliasRes.StatusCode == http.StatusOK {
			log.Printf("[OpenSearch] WARNING: A concrete index named '%s' already exists instead of an alias! Trying to delete it to prevent collision...", aliasName)
			deleteReq := opensearchapi.IndicesDeleteRequest{Index: []string{aliasName}}
			delRes, delErr := deleteReq.Do(ctx, idx.client)
			if delErr == nil {
				delRes.Body.Close()
				log.Printf("[OpenSearch] Successfully deleted concrete index '%s'.", aliasName)
			} else {
				log.Printf("[OpenSearch] Failed to delete concrete index '%s': %v", aliasName, delErr)
			}
		}
	}

	mapping := `{
		"settings": {
			"analysis": {
				"filter": {
					"autocomplete_filter": {
						"type": "ngram",
						"min_gram": 2,
						"max_gram": 10
					}
				},
				"analyzer": {
					"vi_ascii_analyzer": {
						"type": "custom",
						"tokenizer": "standard",
						"filter": [
							"lowercase",
							"asciifolding"
						]
					},
					"autocomplete_analyzer": {
						"type": "custom",
						"tokenizer": "standard",
						"filter": [
							"lowercase",
							"asciifolding",
							"autocomplete_filter"
						]
					}
				}
			},
			"index": {
				"number_of_shards": 1,
				"number_of_replicas": 0,
				"max_ngram_diff": 8
			}
		},
		"mappings": {
			"properties": {
				"id": { "type": "keyword" },
				"tenant_id": { "type": "keyword" },
				"category_id": { "type": "keyword" },
				"product_name_vi": { "type": "text", "analyzer": "vi_ascii_analyzer" },
				"product_name_en": { "type": "text", "analyzer": "english" },
				"product_name_th": { "type": "text", "analyzer": "thai" },
				"description_vi": { "type": "text", "analyzer": "vi_ascii_analyzer" },
				"description_en": { "type": "text", "analyzer": "english" },
				"description_th": { "type": "text", "analyzer": "thai" },
				"brand": { "type": "keyword" },
				"price": { "type": "double" },
				"image_url": { "type": "keyword" },
				"inventory": { "type": "integer" },
				"featured": { "type": "boolean" },
				"status": { "type": "keyword" },
				"search_tags": { "type": "text", "analyzer": "vi_ascii_analyzer" },
				"suggest": {
					"type": "text",
					"analyzer": "autocomplete_analyzer",
					"search_analyzer": "vi_ascii_analyzer"
				}
			}
		}
	}`

	log.Printf("[OpenSearch] Creating index %s...", indexName)
	createReq := opensearchapi.IndicesCreateRequest{
		Index: indexName,
		Body:  strings.NewReader(mapping),
	}
	cRes, cErr := createReq.Do(ctx, idx.client)
	if cErr != nil {
		log.Printf("[OpenSearch] Failed to create index %s: %v", indexName, cErr)
		return
	}
	defer cRes.Body.Close()

	if cRes.IsError() {
		bodyBytes, _ := io.ReadAll(cRes.Body)
		log.Printf("[OpenSearch] Failed to create index %s. Response error: %s", indexName, string(bodyBytes))
		return
	}

	log.Printf("[OpenSearch] Creating alias %s -> %s...", aliasName, indexName)
	aliasBody := fmt.Sprintf(`{"actions": [{"add": {"index": "%s", "alias": "%s"}}]}`, indexName, aliasName)
	aliasReq := opensearchapi.IndicesUpdateAliasesRequest{Body: strings.NewReader(aliasBody)}
	aRes, aErr := aliasReq.Do(ctx, idx.client)
	if aErr != nil {
		log.Printf("[OpenSearch] Failed to create alias %s -> %s: %v", aliasName, indexName, aErr)
		return
	}
	defer aRes.Body.Close()

	if aRes.IsError() {
		bodyBytes, _ := io.ReadAll(aRes.Body)
		log.Printf("[OpenSearch] Failed to update alias. Response error: %s", string(bodyBytes))
	} else {
		log.Printf("[OpenSearch] Successfully configured index %s and alias %s.", indexName, aliasName)
	}
}

func (idx *opensearchIndexer) SearchProducts(ctx context.Context, tenantID, query string, synonymSegments [][]string, from, size int) ([]map[string]interface{}, int, string, error) {
	var innerQuery map[string]interface{}
	trimmedQuery := strings.TrimSpace(query)
	if trimmedQuery == "" {
		innerQuery = map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []interface{}{
					map[string]interface{}{
						"term": map[string]interface{}{
							"tenant_id": tenantID,
						},
					},
					map[string]interface{}{
						"match_all": map[string]interface{}{},
					},
				},
			},
		}
	} else {
		targetFields := []string{
			"product_name_vi^4",
			"product_name_en^2",
			"description_vi",
			"description_en",
			"brand",
			"search_tags",
			"suggest",
		}

		searchClauses := make([]interface{}, 0)

		// Process synonym segments
		for _, segment := range synonymSegments {
			if len(segment) == 0 {
				continue
			}

			if len(segment) == 1 {
				// Standard token match (no synonyms)
				searchClauses = append(searchClauses, map[string]interface{}{
					"multi_match": map[string]interface{}{
						"query":                segment[0],
						"fields":               targetFields,
						"type":                 "best_fields",
						"minimum_should_match": "100%",
					},
				})
			} else {
				// Synonym segment: wrap in a bool should block
				shouldClauses := make([]interface{}, 0, len(segment))

				// Original term (boost = 1.0)
				shouldClauses = append(shouldClauses, map[string]interface{}{
					"multi_match": map[string]interface{}{
						"query":                segment[0],
						"fields":               targetFields,
						"type":                 "best_fields",
						"minimum_should_match": "100%",
					},
				})

				// Synonym terms (boost = 0.6)
				for _, syn := range segment[1:] {
					shouldClauses = append(shouldClauses, map[string]interface{}{
						"multi_match": map[string]interface{}{
							"query":                syn,
							"fields":               targetFields,
							"type":                 "best_fields",
							"minimum_should_match": "100%",
							"boost":                0.6,
						},
					})
				}

				// The search clause requires matching at least one in shouldClauses
				searchClauses = append(searchClauses, map[string]interface{}{
					"bool": map[string]interface{}{
						"should":               shouldClauses,
						"minimum_should_match": 1,
					},
				})
			}
		}

		// Filters that must match
		mustClauses := []interface{}{
			map[string]interface{}{
				"term": map[string]interface{}{
					"tenant_id": tenantID,
				},
			},
		}

		// Only add search query block if there are search clauses
		if len(searchClauses) > 0 {
			mustClauses = append(mustClauses, map[string]interface{}{
				"bool": map[string]interface{}{
					"should":               searchClauses,
					"minimum_should_match": "50%",
				},
			})
		}

		innerQuery = map[string]interface{}{
			"bool": map[string]interface{}{
				"must": mustClauses,
				"should": []interface{}{
					map[string]interface{}{
						"match_phrase": map[string]interface{}{
							"product_name_vi": map[string]interface{}{
								"query": trimmedQuery,
								"boost": 5.0,
							},
						},
					},
					map[string]interface{}{
						"match_phrase": map[string]interface{}{
							"product_name_en": map[string]interface{}{
								"query": trimmedQuery,
								"boost": 3.0,
							},
						},
					},
					map[string]interface{}{
						"match_phrase": map[string]interface{}{
							"product_name_th": map[string]interface{}{
								"query": trimmedQuery,
								"boost": 3.0,
							},
						},
					},
				},
			},
		}
	}

	queryObj := map[string]interface{}{
		"from": from,
		"size": size,
		"query": map[string]interface{}{
			"function_score": map[string]interface{}{
				"query": innerQuery,
				"functions": []interface{}{
					map[string]interface{}{
						"filter": map[string]interface{}{
							"term": map[string]interface{}{
								"featured": true,
							},
						},
						"weight": 1.2,
					},
					map[string]interface{}{
						"filter": map[string]interface{}{
							"term": map[string]interface{}{
								"inventory": 0,
							},
						},
						"weight": 0.5,
					},
				},
				"score_mode": "multiply",
				"boost_mode": "multiply",
			},
		},
	}

	if trimmedQuery != "" {
		queryObj["suggest"] = map[string]interface{}{
			"suggest_vi": map[string]interface{}{
				"text": trimmedQuery,
				"phrase": map[string]interface{}{
					"field":      "product_name_vi",
					"size":       1,
					"confidence": 0.8,
					"direct_generator": []interface{}{
						map[string]interface{}{
							"field":        "product_name_vi",
							"suggest_mode": "missing",
						},
					},
				},
			},
			"suggest_en": map[string]interface{}{
				"text": trimmedQuery,
				"phrase": map[string]interface{}{
					"field":      "product_name_en",
					"size":       1,
					"confidence": 0.8,
					"direct_generator": []interface{}{
						map[string]interface{}{
							"field":        "product_name_en",
							"suggest_mode": "missing",
						},
					},
				},
			},
		}
	}

	bodyBytes, err := json.Marshal(queryObj)
	if err != nil {
		return nil, 0, "", err
	}

	log.Printf("[DEBUG SearchProducts Query] JSON payload: %s", string(bodyBytes))

	req := opensearchapi.SearchRequest{
		Index: []string{"products"},
		Body:  strings.NewReader(string(bodyBytes)),
	}

	res, err := req.Do(ctx, idx.client)
	if err != nil {
		return nil, 0, "", err
	}
	defer res.Body.Close()

	if res.IsError() {
		errBytes, _ := io.ReadAll(res.Body)
		return nil, 0, "", fmt.Errorf("opensearch search error: %s", string(errBytes))
	}

	var searchResponse struct {
		Hits struct {
			Total struct {
				Value int `json:"value"`
			} `json:"total"`
			Hits []struct {
				Source map[string]interface{} `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
		Suggest map[string][]struct {
			Options []struct {
				Text  string  `json:"text"`
				Score float64 `json:"score"`
			} `json:"options"`
		} `json:"suggest"`
	}

	if err := json.NewDecoder(res.Body).Decode(&searchResponse); err != nil {
		return nil, 0, "", err
	}

	products := make([]map[string]interface{}, 0, len(searchResponse.Hits.Hits))
	for _, hit := range searchResponse.Hits.Hits {
		products = append(products, hit.Source)
	}

	var suggestedQuery string
	if searchResponse.Suggest != nil {
		if opts, ok := searchResponse.Suggest["suggest_vi"]; ok && len(opts) > 0 && len(opts[0].Options) > 0 {
			suggestedQuery = opts[0].Options[0].Text
		} else if opts, ok := searchResponse.Suggest["suggest_en"]; ok && len(opts) > 0 && len(opts[0].Options) > 0 {
			suggestedQuery = opts[0].Options[0].Text
		}
	}

	return products, searchResponse.Hits.Total.Value, suggestedQuery, nil
}

func (idx *opensearchIndexer) SuggestProducts(ctx context.Context, tenantID, query string) ([]entity.Suggestion, error) {
	suggestQuery := map[string]interface{}{
		"_source": []string{"id", "brand", "price", "product_name_vi", "product_name_en", "product_name_th", "image_url", "inventory"},
		"size":    10,
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []interface{}{
					map[string]interface{}{
						"term": map[string]interface{}{
							"tenant_id": tenantID,
						},
					},
					map[string]interface{}{
						"bool": map[string]interface{}{
							"should": []interface{}{
								map[string]interface{}{
									"match": map[string]interface{}{
										"suggest": map[string]interface{}{
											"query":    query,
											"operator": "and",
											"boost":    2.0,
										},
									},
								},
								map[string]interface{}{
									"multi_match": map[string]interface{}{
										"query":    query,
										"fields":   []string{"product_name_vi", "product_name_en", "product_name_th", "brand"},
										"operator": "and",
									},
								},
							},
							"minimum_should_match": 1,
						},
					},
				},
			},
		},
	}

	bodyBytes, err := json.Marshal(suggestQuery)
	if err != nil {
		return nil, err
	}

	req := opensearchapi.SearchRequest{
		Index: []string{"products"},
		Body:  strings.NewReader(string(bodyBytes)),
	}

	res, err := req.Do(ctx, idx.client)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.IsError() {
		errBytes, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("opensearch suggest error: %s", string(errBytes))
	}

	var suggestResponse struct {
		Hits struct {
			Hits []struct {
				Source struct {
					ID            string  `json:"id"`
					Brand         string  `json:"brand"`
					Price         float64 `json:"price"`
					ProductNameVI string  `json:"product_name_vi"`
					ProductNameEN string  `json:"product_name_en"`
					ProductNameTH string  `json:"product_name_th"`
					ImageURL      string  `json:"image_url"`
					Inventory     int     `json:"inventory"`
				} `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := json.NewDecoder(res.Body).Decode(&suggestResponse); err != nil {
		return nil, err
	}

	uniqueSuggestions := make([]entity.Suggestion, 0, len(suggestResponse.Hits.Hits))
	seen := make(map[string]bool)
	for _, hit := range suggestResponse.Hits.Hits {
		val := strings.TrimSpace(hit.Source.ProductNameVI)
		if val != "" && !seen[val] {
			seen[val] = true

			suggestion := entity.Suggestion{
				ID:            hit.Source.ID,
				Text:          val,
				Brand:         hit.Source.Brand,
				Price:         hit.Source.Price,
				ProductNameVI: hit.Source.ProductNameVI,
				ProductNameEN: hit.Source.ProductNameEN,
				ProductNameTH: hit.Source.ProductNameTH,
				ImageURL:      hit.Source.ImageURL,
				Inventory:     hit.Source.Inventory,
			}
			uniqueSuggestions = append(uniqueSuggestions, suggestion)
		}
	}

	return uniqueSuggestions, nil
}
