# API Contract - US-008 (Click Tracking)

This document defines the REST API analytical logging endpoints and payload schemas implemented in `search-service` for **US-008**.

---

## 1. REST Endpoints

### Track Product Click
Logs a buyer's click action on a product in the search results page.

* **URL**: `/api/v1/analytics/click`
* **Method**: `POST`
* **Headers**:
  * `X-Tenant-ID: <string>` (Required. Identifies tenant)
  * `Content-Type: application/json`
* **Request Payload (JSON)**:
  ```json
  {
    "search_log_id": "95dd98f6-cdb2-48cc-b4f7-8efcfd8db5eb",
    "product_id": "00000000-0000-0000-0000-000000000000",
    "query": "ban phim",
    "position": 2
  }
  ```
  * **Payload Validation**:
    * `search_log_id`: Required. Must be a valid UUID.
    * `product_id`: Required. Must be a valid UUID.
    * `query`: Required. The search query string that led to the search.
    * `position`: Required. The product position in the search result list (1-indexed, must be `> 0`).

* **Response - Success (200 OK)**:
  *Sent immediately after pushing to background queue, taking < 2ms.*
  ```json
  {
    "status": "success"
  }
  ```

* **Response - Error (400 Bad Request)**:
  *Occurs if validation constraints fail or headers/payload are missing.*
  ```json
  {
    "error": "X-Tenant-ID header is required"
  }
  ```
  or
  ```json
  {
    "error": "position must be greater than 0"
  }
  ```

---

## 2. Database Storage (PostgreSQL GORM)

### Click Logs Table
Stored in `search_svc.click_logs` table (using GORM).

* **Model Struct**:
  ```go
  type ClickLog struct {
      ID            string    `gorm:"type:uuid;primaryKey" json:"id"`
      TenantID      string    `gorm:"type:uuid;not null" json:"tenant_id"`
      SearchLogID   string    `gorm:"type:uuid;not null" json:"search_log_id"`
      Query         string    `gorm:"type:varchar(255);not null" json:"query"`
      ProductID     string    `gorm:"type:uuid;not null" json:"product_id"`
      ClickPosition int       `gorm:"type:integer;not null;default:1" json:"position"`
      ClickedAt     time.Time `json:"clicked_at"`
  }
  ```
