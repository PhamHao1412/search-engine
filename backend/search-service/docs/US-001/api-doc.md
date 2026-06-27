# API Contract - US-001 (Product Ingestion Sync)

This document defines the Kafka consumption details and database structures implemented in `search-service` for **US-001**.

---

## 1. Event Consumption (Kafka)

### Ingestion Event Consumer
Listens to `product-ingestion-events` and updates OpenSearch indices, Redis caches, and GORM sync status.

* **Topic**: `product-ingestion-events`
* **Group ID**: `search-indexer-group`
* **Key**: Product ID
* **Expected Payload (JSON)**:
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

---

## 2. Event Dead-Letter Queue (DLQ)

If processing fails after validation (e.g. invalid event schema), the event is published to the DLQ topic.

* **Topic**: `product-ingestion-events-dlq`
* **Key**: Product ID
* **Payload**: Raw unprocessable string
