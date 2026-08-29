# SwiftSearch Engine Backend

Event-driven search and product ingestion system built with Go, Kafka, OpenSearch, PostgreSQL, and Redis.

The repository is structured as two decoupled services:
- **`product-service`**: Handles product creation, storage in PostgreSQL, and publishing change events to Kafka.
- **`search-service`**: Consists of an HTTP API server (`serverd`) for search/admin queries and a background worker (`workerd`) for Kafka message consumption, OpenSearch indexing, Redis caching, translation, and scheduled batch jobs.

---

## Architecture Overview

```
                        +-----------------------+
                        |   Client / Frontend   |
                        +-----------+-----------+
                                    |
          +-------------------------+-------------------------+
          | (Write / Ingestion)                               | (Read / Search / Analytics)
          v                                                   v
+-------------------+                               +-------------------+
|  product-service  |                               |  search-service   |
|     (serverd)     |                               |     (serverd)     |
+---------+---------+                               +---------+---------+
          |                                                   |
          | 1. Insert/Update                                  | 4. Search / Suggest
          v                                                   v
+-------------------+                               +-------------------+
|    PostgreSQL     |                               |    OpenSearch     |
| (schema: product) |                               |     (Indices)     |
+---------+---------+                               +---------+---------+
          |                                                   ^
          | 2. Produce Event                                  | 3. Index & Cache
          v                                                   |
+-------------------+                               +---------+---------+
|   Apache Kafka    | ============================> |  search-service   |
| (Ingestion Topic) |                               |     (workerd)     |
+-------------------+                               +---------+---------+
          |                                                   |
          | Dead Letter Queue                                 | Sync Jobs & Cache
          v                                                   v
+-------------------+                               +-------------------+
|     Kafka DLQ     |                               | PostgreSQL / Redis|
+-------------------+                               +-------------------+
```

### Data Flow

1. **Ingestion (`product-service`)**:
   - `POST /api/v1/products` receives product payload and stores records in PostgreSQL (`product` schema).
   - An event is published to Kafka topic `product-ingestion-events`.

2. **Indexing & Background Processing (`search-service: workerd`)**:
   - Consumes events from `product-ingestion-events` via consumer group `search-indexer-group`.
   - Handles text translation and keyword extraction (via OpenAI).
   - Upserts product documents into OpenSearch and invalidates/updates Redis cache.
   - Writes sync job history to PostgreSQL (`search` schema).
   - If processing fails repeatedly, the event is routed to `product-ingestion-events-dlq`.

3. **Search & Analytics Querying (`search-service: serverd`)**:
   - Serves full-text search with typo tolerance, synonym expansion, and field boost.
   - Provides autocomplete suggestions backed by Redis and OpenSearch prefix matching.
   - Tracks user search clicks and records daily analytics.
   - Exposes admin endpoints for synonym management, spellcheck rules, and conversational search assistant.

---

## Repository Structure

```
.
├── docker-compose.yml           # Infrastructure (Kafka, OpenSearch, Redis, Kafka UI)
├── Makefile                     # Build, migration, and run targets
├── product-service/             # Ingestion service
│   ├── cmd/
│   │   └── serverd/             # API server entrypoint
│   ├── internal/                # Config, handlers, entities, repositories, Kafka publisher
│   └── scripts/migrations/      # Goose SQL migrations for schema 'product'
└── search-service/              # Search & indexing service
    ├── cmd/
    │   ├── serverd/             # Search & Admin REST API entrypoint
    │   └── workerd/             # Kafka consumer worker & cron jobs entrypoint
    ├── internal/                # OpenSearch indexer, Redis cache, AI services, repositories
    └── scripts/migrations/      # Goose SQL migrations for schema 'search'
```

---

## Prerequisites

- **Go**: `1.21` or newer
- **Docker & Docker Compose**: For local infrastructure
- **PostgreSQL Client (`psql`) & Goose**: For running database migrations
  ```bash
  go install github.com/pressly/goose/v3/cmd/goose@latest
  ```

---

## Getting Started

### 1. Start Infrastructure Containers

Start Redis, OpenSearch, Kafka, and Kafka UI:

```bash
make docker-up
```

Verify services:
- OpenSearch: `http://localhost:9200`
- OpenSearch Dashboards: `http://localhost:5601`
- Kafka UI: `http://localhost:8080`
- Redis: `localhost:6379`

### 2. Configure Environment Files

Copy `.env.example` to `.env` in both service directories:

```bash
cp product-service/.env.example product-service/.env
cp search-service/.env.example search-service/.env
```

Review database credentials and API keys in both files.

### 3. Run Database Migrations

Apply schema migrations for both services:

```bash
# Migrate product schema
make migrate-product-up

# Migrate search schema
make migrate-search-up
```

### 4. Run the Services

Open separate terminal windows or processes for each component:

```bash
# Terminal 1: Start Product Ingestion API (Port 8080)
make run-product

# Terminal 2: Start Search Indexer Worker & Background Cron
make run-search-worker

# Terminal 3: Start Search Query & Admin API (Port 8081)
make run-search-api
```

---

## Key API Endpoints

### Product Service (`:8080`)

| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/health` | Service health check |
| `POST` | `/api/v1/products` | Create/ingest new product and trigger Kafka event |

### Search Service (`:8081`)

#### Search & Storefront
| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/health` | Service health check |
| `GET` | `/api/v1/search` | Full-text search with filters, sort, and pagination |
| `GET` | `/api/v1/suggest` | Autocomplete / prefix suggestions |
| `GET` | `/api/v1/search/hot-keywords` | Retrieve top trending search keywords |
| `GET` | `/api/v1/products/:id` | Get cached product details by ID |
| `POST` | `/api/v1/analytics/click` | Track search result click for ranking analytics |
| `POST` | `/api/v1/search/sync` | Trigger full manual index synchronization |

#### Admin & AI Operations
| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/api/v1/admin/analytics/summary` | Query analytics metrics summary |
| `POST` | `/api/v1/admin/analytics/trigger` | Trigger manual analytics aggregation |
| `GET` | `/api/v1/admin/dictionaries/synonyms` | List configured synonym sets |
| `POST` | `/api/v1/admin/dictionaries/synonyms` | Create a new synonym mapping |
| `GET` | `/api/v1/admin/dictionaries/spellcheck` | List spellcheck dictionary rules |
| `POST` | `/api/v1/admin/dictionaries/spellcheck` | Add custom spellcheck rule |
| `GET` | `/api/v1/admin/ai/suggestions` | View AI-generated keyword suggestions |
| `POST` | `/api/v1/admin/ai/suggestions/:id/approve`| Approve and apply AI suggestion |
| `POST` | `/api/v1/admin/assistant/chat` | Chat with AI admin assistant |

---

## Makefile Reference

| Command | Description |
|---|---|
| `make docker-up` | Start infrastructure containers in background |
| `make docker-down` | Stop infrastructure containers |
| `make docker-clean` | Stop infrastructure containers and wipe data volumes |
| `make migrate-product-up` | Apply all pending migrations for `product-service` |
| `make migrate-search-up` | Apply all pending migrations for `search-service` |
| `make run-product` | Start `product-service` HTTP server |
| `make run-search-api` | Start `search-service` HTTP server (`serverd`) |
| `make run-search-worker` | Start `search-service` indexing worker (`workerd`) |
| `make test` | Run all unit & integration tests across services |
| `make clean` | Remove local build caches and temporary files |
