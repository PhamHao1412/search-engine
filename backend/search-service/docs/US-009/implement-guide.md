# Module Design - US-009 (Search Analytics)

This document describes the design, database models, background aggregation processing, manual trigger APIs, and autocomplete click tracking mechanisms implemented in `search-service` for **US-009**.

---

## 1. Overview
Under US-009, `search-service` records raw search and click events, aggregates them daily into summary tables for sub-200ms performance, and displays statistical reports to Marketplace Admins.
Logging raw telemetry is performed asynchronously via background Goroutines to prevent introducing DB writes delay into the search thread.

---

## 2. Directory & Structure
* `internal/handler/rest/v1/search_handler.go`: Receives search & click requests, validates input, and hands off payload to SearchService.
* `internal/handler/rest/v1/admin_handler.go`: Exposes endpoints for Admin metrics retrieval and manual aggregation triggering.
* `internal/service/search_service.go`: Processes search requests, records raw logs asynchronously, and tracks product click conversions.
* `internal/service/analytics_service.go`: Aggregates raw logs using UPSERT, calculates click CTRs/positions, fetches zero result keywords, and cleans up historical logs.
* `internal/repository/analytics_repository.go`: Executes raw GORM and native PostgreSQL queries to retrieve summary statistics and perform daily aggregation.
* `cmd/workerd/main.go`: Cron scheduler initiating background jobs for aggregation (hourly) and cleanup (daily).

---

## 3. Database Schema
Analytics tables reside in PostgreSQL under the `search_svc` schema:

### 3.1 Raw Telemetry Tables (OLTP)
*   **`search_svc.search_logs`**:
    *   `id` (UUID, PK)
    *   `tenant_id` (UUID)
    *   `query` (TEXT): Raw search query.
    *   `normalized_query` (TEXT): Query normalized to lowercase and single spaces.
    *   `result_count` (INT): Total items returned from OpenSearch.
    *   `searched_at` (TIMESTAMP)
*   **`search_svc.click_logs`**:
    *   `id` (UUID, PK)
    *   `tenant_id` (UUID)
    *   `search_log_id` (UUID, FK cascading from `search_logs.id`): References the originating search.
    *   `query` (TEXT)
    *   `product_id` (UUID)
    *   `click_position` (INT): The 1-based index position of the product when clicked.
    *   `clicked_at` (TIMESTAMP)

### 3.2 Pre-Aggregated Summary Tables (OLAP)
*   **`search_svc.daily_query_analytics`**:
    *   `id` (UUID, PK)
    *   `tenant_id` (UUID)
    *   `query` (TEXT)
    *   `date` (DATE)
    *   `search_count` (INT)
    *   `click_count` (INT)
    *   `zero_result_count` (INT)
    *   `sum_click_position` (INT): Sum of positions clicked (used to compute weighted average position).
    *   *Constraints*: Unique index on `(tenant_id, query, date)`
*   **`search_svc.daily_category_analytics`**:
    *   `id` (UUID, PK)
    *   `tenant_id` (UUID)
    *   `category_id` (UUID)
    *   `category_name` (TEXT)
    *   `date` (DATE)
    *   `search_count` (INT)
    *   `click_count` (INT)
    *   *Constraints*: Unique index on `(tenant_id, category_id, date)`

---

## 4. Key Workflows

### 4.1 Autocomplete Click Tracking & Virtual Search Log
When a user clicks a product suggestion directly from the autocomplete dropdown menu without submitting a search:
1. The frontend invokes `POST /api/v1/search/click` with `search_log_id` left empty (`""`).
2. The backend validator permits an empty string for `search_log_id`.
3. In `TrackClick`, the backend generates a random search log UUID, normalizes the query, and inserts a **Virtual Search Log** record with `result_count = 1`.
4. The click log is then created and linked directly to this virtual search log.
This preserves foreign key integrity and ensures CTR stays correct (1 search, 1 click = 100% CTR) for autocomplete-originated navigation.

### 4.2 Aggregation Job (UPSERT)
Calculates summary analytics day-by-day:
1. Fetches raw searches, clicks, and click category attributions.
2. Group query stats:
   - For `search_logs`: increment `search_count`, increment `zero_result_count` if `result_count = 0`.
   - For `click_logs`: increment `click_count` and add click position to `sum_click_position`.
3. Attributes query searches to product categories:
   - Find the primary category for each query (the category receiving the most clicks for that query).
   - Attribute search counts of the query to its primary category.
4. Executes `SaveDailyQueryAnalytics` and `SaveDailyCategoryAnalytics` using `ON CONFLICT DO UPDATE` (UPSERT) to overwrite entries for today and yesterday.

### 4.3 Background Cron Jobs (workerd)
Automates the calculations and storage management of telemetry data:
1. **Analytics Aggregator Job**: Runs hourly (`0 * * * *` or customized via `ANALYTICS_CRON`). Aggregates logs for both **today** and **yesterday** to ensure eventual consistency of cross-day clicks/searches.
2. **Log Retention Cleanup Job**: Runs daily at 2 AM (`0 2 * * *` or customized via `CLEANUP_CRON`). Automatically purges raw records from `search_logs` and `click_logs` older than **90 days** to manage database storage constraints.

---

## 5. API Reference

### 5.1 GET `/api/v1/admin/analytics/summary`
Retrieves aggregated statistics and lists for the dashboard.
*   **Headers**: `X-Tenant-ID` (required)
*   **Params**: `range` (`today` | `7days` | `30days`, default: `30days`)
*   **Response Structure**:
    ```json
    {
      "summary": {
        "total_searches": 1500,
        "zero_result_searches": 50,
        "ctr": 12.5,
        "avg_click_position": 2.3,
        "spellcheck_rules_count": 45,
        "synonym_rules_count": 12
      },
      "zero_results": [
        {
          "query": "giay nam sieu nhe",
          "search_count": 25,
          "ai_suggestion_status": "Chờ duyệt"
        }
      ],
      "category_analytics": [
        {
          "category_id": "uuid-here",
          "category_name": "Giày dép Nam",
          "search_count": 500,
          "click_count": 75,
          "ctr": 15.0
        }
      ]
    }
    ```
*   **Logic**:
    - Calculates overall CTR: `(SUM(click_count) / SUM(search_count)) * 100` (rounded to 1 decimal place).
    - Calculates average click position: `SUM(sum_click_position) / SUM(click_count)` (rounded to 1 decimal place).
    - Left-joins the zero-result queries with the latest `status` from `ai_suggestions`, translating statuses to user-friendly Vietnamese labels (`Chờ duyệt`, `Đã gợi ý sửa đổi`, `Đã bác bỏ`, or defaulting to `Chờ AI quét`).
    - Counts active active rules from `spellcheck_dictionary` and `search_synonyms`.

### 5.2 POST `/api/v1/admin/analytics/trigger`
Triggers manual analytics aggregation for troubleshooting or backfilling.
*   **Headers**: `X-Tenant-ID` (required)
*   **Params (Optional)**: `start_date` (YYYY-MM-DD), `end_date` (YYYY-MM-DD)
*   **Logic**:
    - If parameters are omitted, runs aggregation for yesterday and today.
    - If `start_date` and `end_date` are specified, validates both formats and sequence, then performs aggregation sequentially day-by-day in the given range.
