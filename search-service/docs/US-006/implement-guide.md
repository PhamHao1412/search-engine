# Module Design - US-006 (Multilingual Search)

This document describes the design, automatic translation pipeline, flat index mapping, and target query field selection implemented in `search-service` for **US-006**.

---

## 1. Overview
Under US-006, `search-service` supports end-to-end multilingual product search. Catalog items ingested in one language (e.g. Vietnamese) are dynamically translated into alternate marketplace languages (English, Thai), stored, and queried based on the buyer's localized interface language.

---

## 2. Directory & Structure
* `internal/service/sync_service.go`: Dynamic translation pipeline using Google Translate API, storing translations to database and building the flat OpenSearch document.
* `internal/infrastructure/translate/translate.go`: Custom HTTP translator wrapping Google Translate's free GTX endpoint.
* `internal/handler/rest/v1/search_handler.go`: Extracts the language selection from the header `X-Language-Key`.
* `internal/infrastructure/opensearch/indexer.go`: Configures language-specific field targets (`getTargetFields`) and boosts.

---

## 3. Translation Pipeline (Ingestion-time)
When the worker consumes a `ProductCreated` event, it invokes translation services dynamically:

1. **Detect Original Language**: Reads `original_language` (e.g., `vi`).
2. **Translate Loop**: For each alternate language (`en`, `th`), calls `translator.Translate()`.
3. **Database Persistence**: Stores translations in `product_svc.product_translations` to serve as a persistent cache and support GORM catalog detail views.
4. **Cache/Alias Sync**: Generates the flat OpenSearch document containing all translation fields:
   * `product_name_vi`, `product_name_en`, `product_name_th`
   * `description_vi`, `description_en`, `description_th`
5. **Autocompletion Setup**: Combines all translated names into a single string for the `suggest` field.

---

## 4. Query Handling (Search-time)
1. **Header Extraction**: The REST handler extracts the buyer's language preference from `X-Language-Key` (normalizing `vn` $\rightarrow$ `vi`, defaults to `vi` if invalid or missing).
2. **Dynamic Target Selection (`getTargetFields`)**: Based on the selected language, the indexer queries all fields but boosts the matched language field high (`5.0`) and the alternates lower (`1.5`).

### Example field sets for search query:
* **If `X-Language-Key: en`**:
  * `product_name_en` (boost: `5.0`)
  * `product_name_vi`, `product_name_th` (boost: `1.5`)
  * `category_name` (boost: `3.0`)
  * `description_en` (boost: `1.0`)
  * `description_vi`, `description_th` (boost: `0.5`)
  * `brand`, `suggest` (boost: `1.0`)

* **If `X-Language-Key: vi`**:
  * `product_name_vi` (boost: `5.0`)
  * `product_name_en`, `product_name_th` (boost: `1.5`)
  * `category_name` (boost: `3.0`)
  * `description_vi` (boost: `1.0`)
  * `description_en`, `description_th` (boost: `0.5`)
  * `brand`, `suggest` (boost: `1.0`)
