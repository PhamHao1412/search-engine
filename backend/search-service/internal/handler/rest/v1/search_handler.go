package v1

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

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

	products, total, err := h.searchSvc.Search(c.Request.Context(), tenantID, query, page, pageSize)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": fmt.Sprintf("Search Service Unavailable: %v", err)})
		return
	}

	totalPages := 0
	if total > 0 {
		totalPages = (total + pageSize - 1) / pageSize
	}

	c.JSON(http.StatusOK, gin.H{
		"products":    products,
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": totalPages,
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
