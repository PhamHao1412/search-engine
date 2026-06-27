# API Contract - US-002 (Product Search)

This document defines the REST API search endpoints and analytical logging schemas implemented in `search-service` for **US-002**.

---

## 1. REST Endpoints

### Search Products
Retrieves relevance-ranked product matches from OpenSearch.

* **URL**: `/api/v1/search`
* **Method**: `GET`
* **Headers**:
  * `X-Tenant-ID: <string>` (Required. Identifies tenant)
* **Query Parameters**:
  * `q` (Required. Search query string, length `<= 100` characters)
  * `page` (Optional. Default: `1`. Must be `> 0`)
  * `page_size` (Optional. Default: `20`. Must be `> 0`)

* **Response - Success (200 OK - Cache Hit or Miss)**:
  ```json
  {
    "page": 1,
    "page_size": 20,
    "total": 1,
    "total_pages": 1,
    "products": [
      {
        "id": "95dd98f6-cdb2-48cc-b4f7-8efcfd8db5eb",
        "tenant_id": "d3b07384-d113-4956-a5db-251d50c18d01",
        "product_name_vi": "Bàn phím cơ không dây Bluetooth Ajazz AC081",
        "product_name_en": "Ajazz AC081 Bluetooth wireless mechanical keyboard",
        "product_name_th": "Ajazz AC081 คีย์บอร์ดไร้สาย Bluetooth",
        "description_vi": "Bàn phím cơ cao cấp với switch cherry xịn xò, hỗ trợ kết nối 3 chế độ bluetooth, type-c và 2.4G.",
        "description_en": "High-end mechanical keyboard with genuine cherry switch, supporting 3-mode connection bluetooth, type-c and 2.4G.",
        "description_th": "คีย์บอร์ดเชิงกลระดับไฮเอนด์พร้อมสวิตช์เชอร์รี่แท้ รองรับการเชื่อมต่อบลูทูธ 3 โหมด, type-c และ 2.4G",
        "brand": "Ajazz",
        "price": 1250000.0,
        "inventory": 50,
        "featured": true,
        "status": "active",
        "search_tags": "bàn phím cơ không dây bluetooth switch cherry kết nối đa chế độ"
      }
    ]
  }
  ```

* **Response - Error (400 Bad Request)**:
  *Occurs if validation constraints fail or header is missing.*
  ```json
  {
    "error": "X-Tenant-ID header is required"
  }
  ```

* **Response - Error (503 Service Unavailable)**:
  *Occurs if OpenSearch is offline and no stale Redis cache exists.*
  ```json
  {
    "error": "Search Service Unavailable: opensearch connection refused"
  }
  ```

---

### Sync All Products (Index Recovery API)
Utility endpoint to force-sync all database records belonging to a tenant to OpenSearch index.

* **URL**: `/api/v1/search/sync`
* **Method**: `POST`
* **Headers**:
  * `X-Tenant-ID: <string>` (Required. Target tenant to sync)

* **Response - Success (200 OK)**:
  ```json
  {
    "message": "Products sync completed successfully",
    "synced_count": 5,
    "tenant_id": "d3b07384-d113-4956-a5db-251d50c18d01"
  }
  ```

---

## 2. Analytical Database Logging (PostgreSQL GORM)

### Search Analytics Log
Written directly (asynchronously via goroutine) to the database when a search is executed to track user queries.

* **Database Table**: `search.search_logs`
* **Model Structure**:
  ```go
  type SearchLog struct {
      ID              string    `gorm:"type:uuid;primaryKey" json:"id"`
      TenantID        string    `gorm:"type:uuid;not null" json:"tenant_id"`
      Query           string    `gorm:"type:varchar(255);not null" json:"query"`
      NormalizedQuery string    `gorm:"type:varchar(255);not null" json:"normalized_query"`
      ResultCount     int       `gorm:"type:integer;not null;default:0" json:"result_count"`
      SearchedAt      time.Time `json:"searched_at"`
  }
  ```

