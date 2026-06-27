package v1

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"search-service/internal/service"
)

type SearchHandler struct {
	searchSvc service.SearchService
	syncSvc   service.SyncService
}

func NewSearchHandler(searchSvc service.SearchService, syncSvc service.SyncService) *SearchHandler {
	return &SearchHandler{
		searchSvc: searchSvc,
		syncSvc:   syncSvc,
	}
}

func (h *SearchHandler) Search(c *gin.Context) {
	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-Tenant-ID header is required"})
		return
	}

	query := c.Query("q")
	// Normalize query to validate its length correctly
	normalized := strings.ToLower(strings.Join(strings.Fields(query), " "))
	if len(normalized) > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "search query exceeds 100 characters"})
		return
	}

	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "20")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page <= 0 {
		page = 1
	}

	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil || pageSize <= 0 {
		pageSize = 20
	}

	products, total, searchLogID, err := h.searchSvc.Search(c.Request.Context(), tenantID, query, page, pageSize)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": fmt.Sprintf("Search Service Unavailable: %v", err)})
		return
	}

	totalPages := 0
	if total > 0 {
		totalPages = (total + pageSize - 1) / pageSize
	}

	c.JSON(http.StatusOK, gin.H{
		"search_log_id": searchLogID,
		"products":      products,
		"total":         total,
		"page":          page,
		"page_size":     pageSize,
		"total_pages":   totalPages,
	})
}

func (h *SearchHandler) SyncAll(c *gin.Context) {
	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-Tenant-ID header is required"})
		return
	}

	count, err := h.syncSvc.SyncAllProductsByTenant(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to sync products: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "Products sync completed successfully",
		"tenant_id":    tenantID,
		"synced_count": count,
	})
}

func (h *SearchHandler) Suggest(c *gin.Context) {
	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-Tenant-ID header is required"})
		return
	}

	query := c.Query("q")
	suggestions, err := h.searchSvc.Suggest(c.Request.Context(), tenantID, query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to get suggestions: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"suggestions": suggestions,
	})
}

type ClickTrackRequest struct {
	SearchLogID string `json:"search_log_id" binding:"required"`
	ProductID   string `json:"product_id" binding:"required"`
	Query       string `json:"query" binding:"required"`
	Position    int    `json:"position" binding:"required"`
}

func (h *SearchHandler) TrackClick(c *gin.Context) {
	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-Tenant-ID header is required"})
		return
	}

	var req ClickTrackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid payload: %v", err)})
		return
	}

	if req.Position <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "position must be greater than 0"})
		return
	}

	// Process click logging in background (non-blocking)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := h.searchSvc.TrackClick(ctx, tenantID, req.SearchLogID, req.ProductID, req.Query, req.Position); err != nil {
			log.Printf("Warning: failed to track click: %v", err)
		}
	}()

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

func (h *SearchHandler) GetProductByID(c *gin.Context) {
	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-Tenant-ID header is required"})
		return
	}

	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "product ID is required"})
		return
	}

	product, err := h.searchSvc.GetProductByID(c.Request.Context(), tenantID, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("product not found: %v", err)})
		return
	}

	c.JSON(http.StatusOK, product)
}
