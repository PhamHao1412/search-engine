# Module Design - US-002 (Product Search)

This document describes the design, indexing analyzer mapping, Redis caching strategies, and OpenSearch query formulas implemented in `search-service` for **US-002**.

---

## 1. Overview
Under US-002, `search-service` exposes the product search API server (`serverd`). It integrates high-performance text searches, caching, relevancy boosting, and analytical telemetry.

---

## 2. Directory & Structure
* `cmd/serverd/`: API entry point. Bootstraps database, cache, indexer, GORM-based analytics publisher, and binds handlers.
* `internal/handler/rest/v1/search_handler.go`: Parses REST requests and returns standard responses.
* `internal/service/search_service.go`: Orchestrates caching lookup $\rightarrow$ query fallback $\rightarrow$ caching write $\rightarrow$ direct GORM database logging.
* `internal/infrastructure/opensearch/`: Constructs search JSON bodies and handles client network calls.
* `internal/infrastructure/redis/`: Interacts with Redis to store and retrieve queries.

---

## 3. Search Engine Mapping & Accent Folding

To support searching Vietnamese text with or without accents (e.g. searching `'ban phim'` should match `'Bàn phím'`), the search index uses a custom analyzer:

```json
{
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
    }
  }
}
```

* **Target Fields**: `product_name_vi`, `description_vi`, and `search_tags` are configured to use `vi_ascii_analyzer`.
* **Action**: Converts characters with diacritics into ASCII equivalents (e.g., `à` $\rightarrow$ `a`). This processes query strings and index structures identically, allowing accent-insensitive matches.
* **Aliases**: `products_v1` is created as a physical index. A virtual **alias** `products` is configured to point to it. The codebase writes and searches exclusively through the `products` alias to allow zero-downtime reindexing.

---

## 4. Caching Strategies (Redis)
To optimize performance and scale searches:
* **Key Format**: `search:{tenant_id}:{normalized_query}:{page}:{page_size}`
* **Query Normalization**: Queries are lowercased and spaces are stripped/normalized prior to cache lookup to improve hit rates.
* **Cache Expiry (TTL)**: 10 minutes.
* **Cache Penetration Protection (Empty Caching)**: When OpenSearch returns 0 results for a query, the empty response `{"total":0,"products":[]}` is saved in Redis anyway. Subsequent duplicate searches trigger a **Cache Hit** immediately without hitting OpenSearch.

---

## 5. Ranking Algorithm (OpenSearch Function Score)

Relevance scoring is calculated dynamically inside OpenSearch:

1. **Multi-field Weightings**:
   * `product_name_vi` (boost: `2.0`)
   * `product_name_en` (boost: `1.5`)
   * `product_name_th` (boost: `1.5`)
   * `description_vi` (boost: `0.8`)
   * `brand` (boost: `1.0`)
   * `search_tags` (boost: `1.0`)
2. **Relevance Boost & Decay (Function Score)**:
   * **Featured Boost**: If product has `featured = true`, its score is multiplied by `1.2`.
   * **Inventory Decay**: If product has `inventory = 0` (out of stock), its score is multiplied by `0.5` (demoting it to bottom lists).
3. **Combination**: Score and boost variables are combined using `multiply` mode.
