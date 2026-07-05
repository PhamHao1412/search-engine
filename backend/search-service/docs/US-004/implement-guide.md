# Module Design - US-004 (Spellcheck)

This document describes the design, database dictionaries, spelling correction workflows, and OpenSearch Phrase Suggester configuration implemented in `search-service` for **US-004**.

---

## 1. Overview
Under US-004, `search-service` provides search query auto-correction and suggestions for typos. It relies on a two-tier mechanism: a local database dictionary (high-speed lookup) and OpenSearch's Phrase Suggester (algorithmic fallback).

---

## 2. Directory & Structure
* `internal/service/search_service.go`: Controls the two-tier spellcheck logic (`correctQuerySpelling` and OpenSearch suggestions extraction).
* `internal/repository/search_repository.go`: Queries `spellcheck_dictionary` tables in PostgreSQL.
* `internal/infrastructure/opensearch/indexer.go`: Configures the `suggest` block containing the `phrase` suggester parameters inside `SearchProducts`.

---

## 3. Two-Tier Correction Workflow

```
User Query (typo: e.g. "iphne")
  |
  v
Tier 1: Check PostgreSQL/Redis Cache (`spellcheck_dictionary`)
  |
  +---> Found? ---> Replace query text (e.g. "iphone"), set autoCorrected = true
  |
  +---> Not Found? ---> Keep query text, proceed to Search
                         |
                         v
                    Query OpenSearch (runs Search + Phrase Suggester)
                         |
                         +---> OpenSearch Suggestion? ---> Return spellcheck_corrected = "iphone", autoCorrected = false
```

---

## 4. Database Dictionary (Tier 1)
Admin-configured typos are persisted in PostgreSQL:

### `search_svc.spellcheck_dictionary`
* **typo_word**: The incorrect word (e.g., `iphne`).
* **correct_word**: The replacement word (e.g., `iphone`).
* **status**: Set to `active`.

At search time, the query is normalized and matched against `typo_word`. If a match is found, the search query is rewritten before hitting OpenSearch, setting `auto_corrected: true` in the API output.

---

## 5. OpenSearch Phrase Suggester (Tier 2)
If the query is not corrected by Tier 1, OpenSearch's phrase suggester runs concurrently with the search request.

### Query Structure (Embedded in Search Query)
```json
{
  "suggest": {
    "suggest_vi": {
      "text": "<query>",
      "phrase": {
        "field": "product_name_vi",
        "size": 1,
        "confidence": 0.8,
        "direct_generator": [
          {
            "field": "product_name_vi",
            "suggest_mode": "missing"
          }
        ]
      }
    },
    "suggest_en": {
      "text": "<query>",
      "phrase": {
        "field": "product_name_en",
        "size": 1,
        "confidence": 0.8,
        "direct_generator": [
          {
            "field": "product_name_en",
            "suggest_mode": "missing"
          }
        ]
      }
    }
  }
}
```

### Response Processing
If a phrase suggestion exists in `suggest_vi` or `suggest_en` and has a higher score, the search service extracts it and returns it to the client as `spellcheck_corrected` with `auto_corrected: false`. This prompts the client UI to display: *"Did you mean: **iphone**?"*.
