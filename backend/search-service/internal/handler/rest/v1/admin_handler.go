package v1

import (
	"fmt"
	"net/http"
	"search-service/internal/service"

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

	suggestions, err := h.repo.GetAISuggestions(c.Request.Context(), tenantID, status, suggestionType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to get AI suggestions: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"suggestions": suggestions,
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

	// Invalidate spellcheck cache when approved
	if sugg.SuggestionType == "typo" {
		_ = h.cache.DeleteSpellcheckCache(c.Request.Context(), tenantID, sugg.SourceValue)
	}

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
