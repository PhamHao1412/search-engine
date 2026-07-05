# Module Design - US-005 (Synonym Expansion)

This document describes the design, query expansion logic, and OpenSearch search-time query structure implemented in `search-service` for **US-005**.

---

## 1. Overview
Under US-005, `search-service` supports search-time synonym expansion. This allows users to find products matching alternate words (e.g. searching `'cafe'` or `'cà phê'` matches products titled `'Coffee'`) without having to reindex documentation.

---

## 2. Directory & Structure
* `internal/service/search_service.go`: Fetches synonym/translation maps (`loadSynonyms` and `loadTranslations`), merges them, and runs the token expansion parser (`ExpandQuery`).
* `internal/repository/search_repository.go`: Queries `search_synonyms` and `search_translations` tables from PostgreSQL.
* `internal/infrastructure/opensearch/indexer.go`: Receives the expanded segments and constructs the OpenSearch boolean query clauses.

---

## 3. Workflow & Merger
To provide unified query expansion, the service merges static translations into the synonyms dictionary at search-time:

1. **Load Synonyms**: Retrieves database entries from `search_svc.search_synonyms` (e.g., `coffee` $\rightarrow$ `cafe`).
2. **Load Translations**: Retrieves database entries from `search_svc.search_translations` (e.g., `coffee` $\rightarrow$ `cà phê`).
3. **Merge**: Appends translation values into the synonyms slice. The merged map represents the tenant's complete expansion dictionary.

---

## 4. Query Expansion Logic (`ExpandQuery`)
The query is split into individual words and analyzed for contiguous phrase or single-word dictionary matches:
* **Phrase matching**: Matches multi-word synonyms (e.g., `vietnamese coffee` $\rightarrow$ `cà phê phin`).
* **Word matching**: Matches single words.
* **Output**: Produces a slice of segments `[][]string`. For example, `coffee robusta` expands to:
  `[["coffee", "cafe", "cà phê"], ["robusta"]]`

---

## 5. OpenSearch Query Construction
During the search execution in OpenSearch, each segment is compiled into a search clause:

### Single Term Segment (No Synonyms)
Evaluates to a standard `multi_match` clause requiring 100% of terms matching:
```json
{
  "multi_match": {
    "query": "robusta",
    "fields": ["product_name_vi^5.0", "product_name_en^1.5", "category_name^3.0"],
    "type": "best_fields",
    "minimum_should_match": "100%"
  }
}
```

### Expanded Segment (Has Synonyms)
Wrapped in a `bool` query with a `should` block where:
* The original term is evaluated with normal weight.
* The synonym terms are evaluated with a **boost of `0.6`** to prioritize exact matching over synonym matching.
* `minimum_should_match` is set to `1`.

```json
{
  "bool": {
    "should": [
      {
        "multi_match": {
          "query": "coffee",
          "fields": ["product_name_vi^5.0", "product_name_en^1.5"],
          "type": "best_fields",
          "minimum_should_match": "100%"
        }
      },
      {
        "multi_match": {
          "query": "cafe",
          "fields": ["product_name_vi^5.0", "product_name_en^1.5"],
          "type": "best_fields",
          "minimum_should_match": "100%",
          "boost": 0.6
        }
      },
      {
        "multi_match": {
          "query": "cà phê",
          "fields": ["product_name_vi^5.0", "product_name_en^1.5"],
          "type": "best_fields",
          "minimum_should_match": "100%",
          "boost": 0.6
        }
      }
    ],
    "minimum_should_match": 1
  }
}
```
*These segment clauses are aggregated inside the main search query's `must` block, requiring at least `50%` of the query segments to match.*
