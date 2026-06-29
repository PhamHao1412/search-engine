package v1

import (
	"fmt"
	"log"
	"net/http"
	"search-service/internal/service"
	"strconv"

	"github.com/gin-gonic/gin"
)

type AdminHandler struct {
	repo  service.SearchRepository
	cache service.ProductCache
	aiSvc service.AISuggestionService
}

func NewAdminHandler(repo service.SearchRepository, cache service.ProductCache, aiSvc service.AISuggestionService) *AdminHandler {
	return &AdminHandler{
		repo:  repo,
		cache: cache,
		aiSvc: aiSvc,
	}
}

func (h *AdminHandler) GetAISuggestions(c *gin.Context) {
	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-Tenant-ID header is required"})
		return
	}

	status := c.Query("status")
	suggestionType := c.Query("type")
	search := c.Query("search")

	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "5")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil || pageSize < 1 {
		pageSize = 5
	}

	suggestions, total, err := h.repo.GetAISuggestions(c.Request.Context(), service.GetAISuggestionsParams{
		TenantID:       tenantID,
		Status:         status,
		SuggestionType: suggestionType,
		Search:         search,
		Page:           page,
		PageSize:       pageSize,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to get AI suggestions: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"suggestions": suggestions,
		"total":       total,
	})
}

func (h *AdminHandler) GetTenants(c *gin.Context) {
	list, err := h.repo.GetAllTenants(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to get tenants: %v", err)})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"tenants": list,
	})
}

func (h *AdminHandler) ApproveAISuggestion(c *gin.Context) {
	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-Tenant-ID header is required"})
		return
	}

	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "suggestion ID is required"})
		return
	}

	sugg, err := h.repo.ApproveAISuggestion(c.Request.Context(), tenantID, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to approve suggestion: %v", err)})
		return
	}

	log.Printf("[Admin] Approved AI suggestion: %s -> %s (Type: %s, Tenant: %s)", sugg.SourceValue, sugg.SuggestedValue, sugg.SuggestionType, tenantID)

	// Invalidate all tenant search, suggest, and dictionary caches when approved
	_ = h.cache.DeleteTenantCache(c.Request.Context(), tenantID)

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Suggestion approved and applied successfully",
	})
}

func (h *AdminHandler) RejectAISuggestion(c *gin.Context) {
	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-Tenant-ID header is required"})
		return
	}

	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "suggestion ID is required"})
		return
	}

	ctx := c.Request.Context()
	sugg, err := h.repo.GetAISuggestionByID(ctx, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "suggestion not found"})
		return
	}

	if sugg.TenantID != tenantID {
		c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized for this tenant"})
		return
	}

	if sugg.Status != "pending" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "suggestion is not pending"})
		return
	}

	if err := h.repo.UpdateAISuggestionStatus(ctx, id, "rejected"); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to reject suggestion: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Suggestion rejected successfully",
	})
}

func (h *AdminHandler) GetSpellcheckRules(c *gin.Context) {
	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-Tenant-ID header is required"})
		return
	}

	rules, err := h.repo.GetSpellcheckRules(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to get spellcheck rules: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"rules": rules,
	})
}

func (h *AdminHandler) GetSearchSynonyms(c *gin.Context) {
	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-Tenant-ID header is required"})
		return
	}

	rules, err := h.repo.GetSearchSynonyms(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to get synonyms: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"rules": rules,
	})
}

func (h *AdminHandler) GenerateAISuggestions(c *gin.Context) {
	err := h.aiSvc.GenerateAISuggestions(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to generate suggestions: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "AI Suggestions generated successfully",
	})
}
