# Module Design - US-008 (Click Tracking)

This document describes the module design, database mappings, validation rules, and asynchronous concurrency execution patterns implemented in `search-service` for **US-008**.

---

## 1. Overview
Under US-008, `search-service` exposes an analytical POST route to track buyer click actions on the storefront. Clicks are logged directly to PostgreSQL using GORM. To prevent blocking the customer's browser redirection, requests are processed asynchronously in memory using Go's goroutines.

---

## 2. Directory & Structure
* `cmd/serverd/route/route_v1.go`: Binds the route `POST /api/v1/analytics/click` to `SearchHandler.TrackClick`.
* `internal/handler/rest/v1/search_handler.go`: Handles the HTTP request parsing, header extraction (`X-Tenant-ID`), and validation.
* `internal/service/search_service.go`: Encapsulates business validation (ensuring `position > 0`).
* `internal/repository/analytics_repository.go`: Creates and saves GORM entities to `click_logs` in PostgreSQL.

---

## 3. Asynchronous Concurrency Design (Goroutines)

To maintain a response time under 5ms (P99), we run GORM database inserts in a separate execution thread (goroutine) and respond `200 OK` to the buyer immediately:

```go
// search_handler.go
go func() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := h.searchSvc.TrackClick(ctx, tenantID, req.SearchLogID, req.ProductID, req.Query, req.Position); err != nil {
		log.Printf("Warning: failed to track click: %v", err)
	}
}()
```

* **Rationale**: Buyers click on products to transition to the detail page. If the click API takes too long (e.g. 50ms+), it causes perceived storefront lag.
* **Context Propagation**: We decouple the goroutine from the main HTTP request context (`c.Request.Context()`) by creating a fresh `context.Background()` with a 5-second timeout. This prevents GORM from canceling the database transaction when the original HTTP connection is closed.

---

## 4. Database Foreign Key Validation & Error Mitigation

* **Foreign Key**: `click_logs.search_log_id` references `search_logs.id`.
* **Constraint Protection (BR-002)**: If an invalid or non-existent `search_log_id` is supplied by the frontend (which would cause a PostgreSQL foreign key violation), the GORM database transaction fails inside the background goroutine.
* **Resilience**: The system logs a Warning level event to the console but does not throw any API error. The client still receives a successful `200 OK` code to guarantee that storefront navigation is never interrupted due to tracking/analytics errors.
