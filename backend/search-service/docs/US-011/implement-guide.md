# Module Design - US-011 (Synonym Management)

This document describes the design, REST APIs, bidirectional mapping logic, and cache invalidation strategies implemented in `search-service` for **US-011**.

---

## 1. Overview
Under US-011, `search-service` provides management APIs for administrators to configure dictionaries: Synonym rules (`search_synonyms`) and Spellcheck maps (`spellcheck_dictionary`). Changes to dictionaries invalidate caching layers instantly.

---

## 2. Directory & Structure
* `internal/handler/rest/v1/admin_handler.go`: Exposes endpoints for creating, retrieving, and deleting synonym/spellcheck configurations.
* `internal/repository/search_repository.go`: Performs GORM SQL transactions to create or delete rule records in PostgreSQL.
* `internal/infrastructure/redis/`: Cache library exposing `DeleteTenantCache` to invalidate search/suggestion logs.

---

## 3. API Endpoints

### 3.1 Synonyms API
* `GET /admin/synonyms` (Headers: `X-Tenant-ID`): Returns all synonyms for the tenant.
* `POST /admin/synonyms` (Body: JSON containing `keyword`, `synonym`, and `is_bidirectional`):
  * **Bidirectional Logic**: If `is_bidirectional: true`, the handler inserts two distinct records: `keyword -> synonym` AND `synonym -> keyword`.
* `DELETE /admin/synonyms/:id` (Headers: `X-Tenant-ID`): Deletes the synonym rule.

### 3.2 Spellcheck API
* `GET /admin/spellchecks`: Returns typo dictionary mappings.
* `POST /admin/spellchecks` (Body: `typo_word`, `correct_word`): Registers a spelling auto-correction rule.
* `DELETE /admin/spellchecks/:id`: Removes a spelling rule.

---

## 4. Workflows & Cache Invalidation
When an administrator modifies dictionaries:
1. **Database Write**: The record is saved or deleted via GORM.
2. **Cache Purge**: The API handler calls `cache.DeleteTenantCache(ctx, tenantID)`:
   * It deletes all search caches (`search:{tenant_id}:*`) and suggestion caches (`suggest:{tenant_id}:*`) from Redis.
3. **Effect**: The next search query by any buyer triggers a cache miss, fetches the newly modified dictionary rules from PostgreSQL, and indexes/caches them dynamically, ensuring zero search delay for catalog updates.
4. **Response**: Returns a JSON success status confirming deletion/creation.
