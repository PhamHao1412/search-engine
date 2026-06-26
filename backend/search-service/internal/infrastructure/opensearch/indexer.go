package opensearch

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	opensearchgo "github.com/opensearch-project/opensearch-go/v2"
	"github.com/opensearch-project/opensearch-go/v2/opensearchapi"
	"search-service/internal/service"
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

func (idx *opensearchIndexer) EnsureIndex(ctx context.Context) {
	indexName := "products_v1"
	aliasName := "products"

	existsReq := opensearchapi.CatIndicesRequest{Index: []string{indexName}}
	res, err := existsReq.Do(ctx, idx.client)
	if err == nil {
		defer res.Body.Close()
		if res.StatusCode == http.StatusOK {
			return
		}
	}

	mapping := `{
		"settings": {
			"analysis": {
				"analyzer": {
					"vi_ascii_analyzer": {
						"type": "custom",
						"tokenizer": "standard",
						"filter": [
							"lowercase",
							"asciifolding"
						]
					}
				}
			},
			"index": {
				"number_of_shards": 1,
				"number_of_replicas": 0
			}
		},
		"mappings": {
			"properties": {
				"id": { "type": "keyword" },
				"tenant_id": { "type": "keyword" },
				"product_name_vi": { "type": "text", "analyzer": "vi_ascii_analyzer" },
				"product_name_en": { "type": "text", "analyzer": "english" },
				"product_name_th": { "type": "text", "analyzer": "thai" },
				"description_vi": { "type": "text", "analyzer": "vi_ascii_analyzer" },
				"description_en": { "type": "text", "analyzer": "english" },
				"description_th": { "type": "text", "analyzer": "thai" },
				"brand": { "type": "keyword" },
				"price": { "type": "double" },
				"inventory": { "type": "integer" },
				"featured": { "type": "boolean" },
				"status": { "type": "keyword" },
				"search_tags": { "type": "text", "analyzer": "vi_ascii_analyzer" },
				"suggest": {
					"type": "completion",
					"contexts": [
						{
							"name": "tenant_context",
							"type": "category",
							"path": "tenant_id"
						}
					]
				}
			}
		}
	}`

	createReq := opensearchapi.IndicesCreateRequest{
		Index: indexName,
		Body:  strings.NewReader(mapping),
	}
	cRes, err := createReq.Do(ctx, idx.client)
	if err == nil {
		cRes.Body.Close()
	}

	aliasBody := fmt.Sprintf(`{"actions": [{"add": {"index": "%s", "alias": "%s"}}]}`, indexName, aliasName)
	aliasReq := opensearchapi.IndicesUpdateAliasesRequest{Body: strings.NewReader(aliasBody)}
	aRes, err := aliasReq.Do(ctx, idx.client)
	if err == nil {
		aRes.Body.Close()
	}
}

func (idx *opensearchIndexer) SearchProducts(ctx context.Context, tenantID, query string, from, size int) ([]map[string]interface{}, int, error) {
	// Build OpenSearch JSON query
	queryObj := map[string]interface{}{
		"from": from,
		"size": size,
		"query": map[string]interface{}{
			"function_score": map[string]interface{}{
				"query": map[string]interface{}{
					"bool": map[string]interface{}{
						"must": []interface{}{
							map[string]interface{}{
								"term": map[string]interface{}{
									"tenant_id": tenantID,
								},
							},
						},
						"should": []interface{}{
							map[string]interface{}{
								"multi_match": map[string]interface{}{
									"query": query,
									"fields": []string{
										"product_name_vi^2.0",
										"product_name_en^1.5",
										"product_name_th^1.5",
										"description_vi^0.8",
										"description_en^0.8",
										"description_th^0.8",
										"brand^1.0",
										"search_tags^1.0",
									},
									"type": "best_fields",
								},
							},
							map[string]interface{}{
								"match_phrase": map[string]interface{}{
									"product_name_vi": map[string]interface{}{
										"query": query,
										"boost": 4.0,
									},
								},
							},
							map[string]interface{}{
								"match_phrase": map[string]interface{}{
									"product_name_en": map[string]interface{}{
										"query": query,
										"boost": 3.0,
									},
								},
							},
							map[string]interface{}{
								"match_phrase": map[string]interface{}{
									"product_name_th": map[string]interface{}{
										"query": query,
										"boost": 3.0,
									},
								},
							},
						},
						"minimum_should_match": 1,
					},
				},
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

	bodyBytes, err := json.Marshal(queryObj)
	if err != nil {
		return nil, 0, err
	}

	req := opensearchapi.SearchRequest{
		Index: []string{"products"},
		Body:  strings.NewReader(string(bodyBytes)),
	}

	res, err := req.Do(ctx, idx.client)
	if err != nil {
		return nil, 0, err
	}
	defer res.Body.Close()

	if res.IsError() {
		errBytes, _ := io.ReadAll(res.Body)
		return nil, 0, fmt.Errorf("opensearch search error: %s", string(errBytes))
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
	}

	if err := json.NewDecoder(res.Body).Decode(&searchResponse); err != nil {
		return nil, 0, err
	}

	products := make([]map[string]interface{}, 0, len(searchResponse.Hits.Hits))
	for _, hit := range searchResponse.Hits.Hits {
		products = append(products, hit.Source)
	}

	return products, searchResponse.Hits.Total.Value, nil
}
