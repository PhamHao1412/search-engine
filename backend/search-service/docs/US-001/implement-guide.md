# Module Design - US-001 (Ingestion & Processing)

This document describes the design, schema tables, and internal reprocessor workflows in `search-service` for **US-001**.

---

## 1. Overview
The `search-service` runs a background worker (`workerd`) that consumes product ingestion events, translates metadata to EN/TH languages, automatically tags product keyword indexes using AI, and performs indexing on OpenSearch.

---

## 2. Directory & Structure
* `cmd/workerd/`: Bootstraps Kafka Consumer, scheduler, and runs the sync worker.
* `internal/service/sync_service.go`: Processes translated elements, AI keywords, and saves DB statuses.
* `internal/infrastructure/translate/`: Stub wrapper for Google Translate.
* `internal/infrastructure/ai/`: Wraps OpenAI API for search tag generation.
* `internal/repository/search_repository.go`: Handles SQL writes to postgres.

---

## 3. Database Schema
Isolated in the PostgreSQL schema `search_svc`.

### `search_svc.product_translations`
Stores translated names/descriptions.
*Unique constraint:* `uq_product_translation` on `(product_id, language_code)`. Uses GORM `Create` with `clause.OnConflict` (Upsert) to prevent duplicate key constraint errors.

### `search_svc.search_sync_jobs`
Tracks individual product synchronization statuses to assure eventual consistency.

| Column | Type | Constraints | Description |
| :--- | :--- | :--- | :--- |
| `id` | `uuid` | Primary Key | Job ID |
| `tenant_id` | `varchar(255)` | Not Null | Partition ID |
| `product_id` | `uuid` | Not Null, Unique | Product mapping |
| `status` | `varchar(50)` | Not Null | `success`, `pending`, `failed_translation`, `failed_ai`, `failed_opensearch` |
| `retry_count` | `integer` | Default: `0` | Re-sync attempts |
| `error_message` | `text` | | Failure description |

---

## 4. Workflows

### Async Ingestion Pipeline (Worker)
1. **Kafka Event**: The worker consumes `ProductCreated` message.
2. **Sync Job Tracking**: Inserts `pending` status record into `search_sync_jobs`.
3. **Translation**: Invokes translator to translate Name/Description to EN and TH. If it fails, sets status to `failed_translation`.
4. **AI Search Tags**: Calls OpenAI to generate search tags. If it fails, sets status to `failed_ai`.
5. **Postgres Save**: Persists translated properties.
6. **OpenSearch Indexing**: Indexes the product document (with Vietnamese, English, Thai names and AI tags). If it fails, sets status to `failed_opensearch`.
7. **Complete**: Sets job status to `success` on completion.

### Reprocessor Cron Job
* A cron job scheduled using `github.com/robfig/cron/v3` runs every minute (default: `*/1 * * * *`).
* Queries the database for failed sync jobs (`status IN ('failed_translation', 'failed_ai')` and `retry_count < 5`).
* Automatically increments the retry count and re-triggers the synchronization pipeline.
