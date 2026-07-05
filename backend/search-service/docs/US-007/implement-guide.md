# Module Design - US-007 (Ranking Engine)

This document describes the design, scoring multipliers, exact phrase boosting, and OpenSearch `function_score` query structure implemented in `search-service` for **US-007**.

---

## 1. Overview
Under US-007, `search-service` calculates a dynamic search result rank. The relevance score combines standard text relevance (BM25) with exact match boosts and business modifiers (demoting out-of-stock items, boosting featured listings).

---

## 2. Directory & Structure
* `internal/infrastructure/opensearch/indexer.go`: Compiles the `function_score` query block, including filters, weights, and score modes.
* `cmd/serverd/` and `cmd/workerd/`: Read featured boost and inventory decay settings from environment variables and inject them into `NewOpenSearchIndexer`.

---

## 3. Scoring Architecture
The final score of a product document is determined by combining the textual match score (BM25) with custom business weights:

$$\text{Final Score} = \text{Textual Match Score} \times \text{Featured Modifier} \times \text{Inventory Modifier}$$

---

## 4. Query Components

### 4.1 Exact Match Boosts (Textual)
To prioritize products matching exact word orders over scattered keyword matches, `match_phrase` queries are evaluated inside the query `should` block with high boosts:
* `product_name_vi` (boost: `5.0`)
* `product_name_en` (boost: `3.0`)
* `product_name_th` (boost: `3.0`)

---

### 4.2 Business Scoring Modifiers (Function Score)
The textual search query (`innerQuery`) is wrapped in a `function_score` envelope containing specific filter weights:

1. **Featured Product Boost**:
   * **Filter**: `{"term": {"featured": true}}`
   * **Weight**: `featuredBoost` (Default: `1.2`). Multiplies the score of featured items.
2. **Inventory Decay (Out-of-stock Demotion)**:
   * **Filter**: `{"term": {"inventory": 0}}`
   * **Weight**: `inventoryDecay` (Default: `0.5`). Halves the score of out-of-stock items, pushing them to the bottom without hiding them completely.

### Query Construction Example
```json
{
  "query": {
    "function_score": {
      "query": {
        "bool": {
          "must": [ ... ],
          "should": [
            {
              "match_phrase": {
                "product_name_vi": {
                  "query": "<user-query>",
                  "boost": 5.0
                }
              }
            }
          ]
        }
      },
      "functions": [
        {
          "filter": { "term": { "featured": true } },
          "weight": 1.2
        },
        {
          "filter": { "term": { "inventory": 0 } },
          "weight": 0.5
        }
      ],
      "score_mode": "multiply",
      "boost_mode": "multiply"
    }
  }
}
```
* **score_mode: multiply**: Multiplies weights of matched functions together (e.g., a featured product that is out of stock gets $1.2 \times 0.5 = 0.6$ modifier).
* **boost_mode: multiply**: Multiplies the final combined function weight with the initial textual relevance score.
