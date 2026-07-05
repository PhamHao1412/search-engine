# Module Design - US-003 (Autocomplete)

This document describes the design, indexing mapping, Redis caching strategies, and OpenSearch query structure implemented in `search-service` for **US-003**.

---

## 1. Overview
Under US-003, `search-service` exposes the autocomplete query endpoint `/suggest`. It supports instant suggestion search as the user types, isolated by `tenant_id` context.

---

## 2. Directory & Structure
* `internal/handler/rest/v1/search_handler.go`: Exposes `Suggest` API endpoint. Extracts headers (`X-Tenant-ID`, `X-Language-Key`) and parses the prefix query parameter `q`.
* `internal/service/search_service.go`: Controls cached lookup, calls spelling correction, invokes OpenSearch query, and caches responses.
* `internal/infrastructure/opensearch/indexer.go`: Contains `SuggestProducts` querying OpenSearch index using custom analyzer matching.
* `internal/infrastructure/redis/`: Cache storage for autocomplete suggestions.

---

## 3. OpenSearch N-gram Setup for Autocomplete
To support instant suggestion matching on partial inputs (e.g. typing `'rob'` matches `'Robusta'`), the `suggest` field uses a custom n-gram analyzer:

### Settings & Filters
```json
{
  "analysis": {
    "filter": {
      "autocomplete_filter": {
        "type": "ngram",
        "min_gram": 2,
        "max_gram": 10
      }
    },
    "analyzer": {
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
  }
}
```

### Mappings
```json
"suggest": {
  "type": "text",
  "analyzer": "autocomplete_analyzer",
  "search_analyzer": "vi_ascii_analyzer"
}
```
* **Index-time**: `suggest` uses `autocomplete_analyzer` to split names into overlapping shingles (e.g., `ro`, `rob`, `robu`, etc.).
* **Search-time**: Uses `vi_ascii_analyzer` (standard tokenizer + lowercase + asciifolding) so that user input matches sub-shingles without further n-gram expansion, ensuring correct matching.

---

## 4. Query Construction
The suggestions search is performed on two tiers in a single boolean `should` query:

1. **Suggest Field Match**: Matches `suggest` field directly with an `and` operator (boost: `2.0`).
2. **Multi-field Match**: Multi-match across product names in different languages and brand (with `and` operator) using language-specific boosts:
   * Current language (e.g., `product_name_vi`): boost `5.0`
   * Alternate languages (e.g., `product_name_en`, `product_name_th`): boost `1.5`
   * `brand`

### Query Example (JSON payload)
```json
{
  "size": 10,
  "query": {
    "bool": {
      "must": [
        { "term": { "tenant_id": "<tenant_id>" } },
        {
          "bool": {
            "should": [
              {
                "match": {
                  "suggest": {
                    "query": "<user-query>",
                    "operator": "and",
                    "boost": 2.0
                  }
                }
              },
              {
                "multi_match": {
                  "query": "<user-query>",
                  "fields": [
                    "product_name_vi^5.0",
                    "product_name_en^1.5",
                    "product_name_th^1.5",
                    "brand"
                  ],
                  "operator": "and"
                }
              }
            ],
            "minimum_should_match": 1
          }
        }
      ]
    }
  }
}
```

---

## 5. Caching Strategies (Redis)
* **Key Format**: `suggest:{tenant_id}:{query}:{lang}`
* **Query Normalization**: Cleaned of trailing white spaces, lowercased.
* **Cache Expiry (TTL)**: 10 minutes.
* **Cache Invalidation**: Whenever synonyms, spellchecks, or products are mutated, the tenant's cache keys are invalidated (`DeleteTenantCache`).
