# API Doc - US-001 (Product Ingestion)

This document defines the REST API endpoints and event messaging contracts implemented in `product-service` under **US-001**.

---

## 1. REST Endpoints

### Create Product
Creates a new product in the database and triggers search ingestion by publishing a message to Kafka.

* **URL**: `/api/v1/products`
* **Method**: `POST`
* **Headers**:
  * `Content-Type: application/json`
  * `X-Tenant-ID: <string>` (Required. Unique tenant identifier)

* **Request Body (JSON)**:
  ```json
  {
    "name": "Bàn phím cơ không dây Bluetooth Ajazz AC081",
    "description": "Bàn phím cơ cao cấp với switch cherry xịn xò, hỗ trợ kết nối 3 chế độ bluetooth, type-c và 2.4G.",
    "category_id": "95dd98f6-cdb2-48cc-b4f7-8efcfd8db5eb",
    "brand": "Ajazz",
    "price": 1250000.0,
    "image_url": "https://example.com/ajazz-ac081.png",
    "inventory": 50,
    "original_language": "vi",
    "featured": true
  }
  ```

* **Response - Success (201 Created)**:
  ```json
  {
    "message": "Product created successfully",
    "product": {
      "id": "95dd98f6-cdb2-48cc-b4f7-8efcfd8db5eb",
      "tenant_id": "d3b07384-d113-4956-a5db-251d50c18d01",
      "name": "Bàn phím cơ không dây Bluetooth Ajazz AC081",
      "description": "Bàn phím cơ cao cấp với switch cherry xịn xò, hỗ trợ kết nối 3 chế độ bluetooth, type-c và 2.4G.",
      "category_id": "95dd98f6-cdb2-48cc-b4f7-8efcfd8db5eb",
      "brand": "Ajazz",
      "price": 1250000.0,
      "image_url": "https://example.com/ajazz-ac081.png",
      "inventory": 50,
      "original_language": "vi",
      "featured": true,
      "status": "active",
      "created_at": "2026-06-27T01:50:00+07:00",
      "updated_at": "2026-06-27T01:50:00+07:00"
    }
  }
  ```

* **Response - Error (400 Bad Request)**:
  *Validation failure.*
  ```json
  {
    "error": "Missing X-Tenant-ID header"
  }
  ```

* **Response - Error (500 Internal Server Error)**:
  ```json
  {
    "error": "Failed to publish ingestion event to Kafka: connection refused"
  }
  ```

---

## 2. Event Messages (Kafka)

### Ingestion Event
Published to notify search-service of product mutations.

* **Topic**: `product-ingestion-events`
* **Key**: Product ID (`95dd98f6-cdb2-48cc-b4f7-8efcfd8db5eb`)
* **Payload (JSON)**:
  ```json
  {
    "id": "e2298c56-3407-4e3a-b8eb-c397c02b281f",
    "event_type": "ProductCreated",
    "timestamp": "2026-06-27T01:50:00Z",
    "data": {
      "id": "95dd98f6-cdb2-48cc-b4f7-8efcfd8db5eb",
      "tenant_id": "d3b07384-d113-4956-a5db-251d50c18d01",
      "name": "Bàn phím cơ không dây Bluetooth Ajazz AC081",
      "description": "Bàn phím cơ cao cấp với switch cherry xịn xò, hỗ trợ kết nối 3 chế độ bluetooth, type-c và 2.4G.",
      "category_id": "95dd98f6-cdb2-48cc-b4f7-8efcfd8db5eb",
      "brand": "Ajazz",
      "price": 1250000.0,
      "image_url": "https://example.com/ajazz-ac081.png",
      "inventory": 50,
      "original_language": "vi",
      "featured": true,
      "status": "active",
      "created_at": "2026-06-27T01:50:00+07:00",
      "updated_at": "2026-06-27T01:50:00+07:00"
    }
  }
  ```
