# Module Design - US-010 (AI Suggestion Engine)

This document describes the design, AI generation pipeline, OpenAI prompt guidelines, and database structures implemented in `search-service` for **US-010**.

---

## 1. Overview
Under US-010, `search-service` runs a background/batch processing workflow to analyze search failures (queries that yielded `0` results). It calls OpenAI's GPT API to generate synonym mappings or spelling corrections based on product catalog contexts, saving them as pending suggestions.

---

## 2. Directory & Structure
* `internal/service/ai_service.go`: Fetches failed queries from logs, retrieves catalog keywords, orchestrates AI analysis, and persists suggestions.
* `internal/infrastructure/ai/ai.go`: Builds the JSON request payload and calls OpenAI's chat completions API.
* `internal/repository/search_repository.go`: Queries database logs and inserts/updates `ai_suggestions` tables.

---

## 3. Database Schema
AI proposed dictionaries are persisted in PostgreSQL:

### `search_svc.ai_suggestions`
* **id**: Unique suggestion UUID.
* **tenant_id**: Identifies the marketplace.
* **suggestion_type**: Either `synonym` (expanding query) or `typo` (auto-correcting spelling).
* **source_value**: The user query containing the spelling error or alternate phrase (e.g. `iphne` or `cafe`).
* **suggested_value**: The corrected word or synonym matched from catalog listings (e.g. `iphone` or `cà phê`).
* **confidence_score**: AI-evaluated probability index (`decimal(5,4)` e.g., `0.9500`).
* **status**: Default is `pending` (awaits Admin approval).

---

## 4. Suggestion Pipeline (Batch Job)

```
Query Postgres `search_logs` (Find distinct queries where `result_count = 0`)
  |
  v
Fetch active keywords from `products` catalog
  |
  v
Build OpenAI Prompt (Pass failed queries & catalog context)
  |
  v
Invoke OpenAI Chat API (gpt-4o-mini)
  |
  v
Parse JSON Array response
  |
  v
Upsert records in PostgreSQL `ai_suggestions` as "pending"
```

---

## 5. OpenAI Prompting & Constraint Design
To prevent hallucinated words, the AI wrapper strictly constraints OpenAI's completions:
* **System Prompt**: Enforces OpenAI to act as a localized e-commerce search dictionary assistant.
* **Catalog Constraints**: Provides a list of active keywords currently indexable in the tenant's catalog.
* **Output Format**: Enforces output to strictly match a JSON schema:
  ```json
  [
    {
      "suggestion_type": "typo",
      "source_value": "iphne",
      "suggested_value": "iphone",
      "confidence_score": 0.98
    }
  ]
  ```
* **Failure Guard**: The service validates that `suggested_value` actually exists in the catalog context before saving it, ensuring high reliability.
