# Module Design - US-012 (Approve AI Suggestion)

This document describes the design, REST APIs, approval/rejection database logic, dictionary propagation, and cache purging implemented in `search-service` for **US-012**.

---

## 1. Overview
Under US-012, `search-service` provides APIs to manage proposed AI suggestions. Administrators can approve or reject these suggestions. Approving an AI suggestion dynamically promotes it to the active synonym or spellcheck dictionary.

---

## 2. Directory & Structure
* `internal/handler/rest/v1/admin_handler.go`: Exposes endpoints for managing AI suggestion approvals/rejections and retrieving proposed listings.
* `internal/repository/search_repository.go`: Contains `ApproveAISuggestion` and `UpdateAISuggestionStatus` performing database updates inside PostgreSQL transactions.
* `internal/infrastructure/redis/`: Caching purge support.

---

## 3. Workflow for Approving Suggestions

```
Admin clicks "Approve" on UI
  |
  v
`POST /admin/ai-suggestions/:id/approve`
  |
  v
Fetch AI Suggestion from PostgreSQL (`search_svc.ai_suggestions`)
  |
  +---> Type: "synonym"? ---> Insert record into `search_svc.search_synonyms`
  |
  +---> Type: "typo"?    ---> Insert record into `search_svc.spellcheck_dictionary`
  |
  v
Update Suggestion Status to "approved" in `search_svc.ai_suggestions`
  |
  v
Delete Tenant Cache in Redis (calls `DeleteTenantCache`)
  |
  v
Return JSON success confirmation
```

---

## 4. Workflows

### 4.1 Retrieve Suggestions List
* `GET /admin/ai-suggestions` (Query params: `status`, `type`, `search`, `page`, `page_size`):
  * Supports listing pending suggestions with server-side pagination.

### 4.2 Rejection Logic
* `POST /admin/ai-suggestions/:id/reject`:
  * Fetches the suggestion by ID.
  * Updates its status to `rejected` in `search_svc.ai_suggestions`.
  * *Note: Does not perform cache invalidation or dictionary propagation, saving CPU cycles.*
