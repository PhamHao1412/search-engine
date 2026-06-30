package v1

import (
	"crypto/rand"
	"fmt"
	"log"
	"net/http"
	"search-service/internal/entity"
	"search-service/internal/service"
	"strconv"
	"strings"
	"time"

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

func (h *AdminHandler) CreateSynonym(c *gin.Context) {
	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-Tenant-ID header is required"})
		return
	}

	type CreateSynonymReq struct {
		Keyword         string `json:"keyword" binding:"required"`
		Synonym         string `json:"synonym" binding:"required"`
		IsBidirectional bool   `json:"is_bidirectional"`
	}

	var req CreateSynonymReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	keywordClean := strings.TrimSpace(req.Keyword)
	synonymClean := strings.TrimSpace(req.Synonym)
	if keywordClean == "" || synonymClean == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Keyword and Synonym cannot be empty"})
		return
	}

	ctx := c.Request.Context()

	// 1. Create main synonym A -> B
	syn1 := &entity.SearchSynonym{
		ID:        h.newUUID(),
		TenantID:  tenantID,
		Keyword:   keywordClean,
		Synonym:   synonymClean,
		Status:    "active",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := h.repo.SaveSearchSynonym(ctx, syn1); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to save synonym: %v", err)})
		return
	}

	// 2. Create reverse synonym B -> A if bidirectional
	if req.IsBidirectional {
		syn2 := &entity.SearchSynonym{
			ID:        h.newUUID(),
			TenantID:  tenantID,
			Keyword:   synonymClean,
			Synonym:   keywordClean,
			Status:    "active",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if err := h.repo.SaveSearchSynonym(ctx, syn2); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to save bidirectional synonym: %v", err)})
			return
		}
	}

	// 3. Invalidate caches
	_ = h.cache.DeleteTenantCache(ctx, tenantID)

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Synonym created successfully",
	})
}

func (h *AdminHandler) DeleteSynonym(c *gin.Context) {
	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-Tenant-ID header is required"})
		return
	}

	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Synonym ID is required"})
		return
	}

	ctx := c.Request.Context()
	if err := h.repo.DeleteSearchSynonym(ctx, tenantID, id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to delete synonym: %v", err)})
		return
	}

	// Invalidate caches
	_ = h.cache.DeleteTenantCache(ctx, tenantID)

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Synonym deleted successfully",
	})
}

func (h *AdminHandler) CreateSpellcheck(c *gin.Context) {
	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-Tenant-ID header is required"})
		return
	}

	type CreateSpellcheckReq struct {
		TypoWord    string `json:"typo_word" binding:"required"`
		CorrectWord string `json:"correct_word" binding:"required"`
	}

	var req CreateSpellcheckReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	typoClean := strings.TrimSpace(req.TypoWord)
	correctClean := strings.TrimSpace(req.CorrectWord)
	if typoClean == "" || correctClean == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "TypoWord and CorrectWord cannot be empty"})
		return
	}

	ctx := c.Request.Context()
	entry := &entity.SpellcheckDictionary{
		ID:          h.newUUID(),
		TenantID:    tenantID,
		TypoWord:    typoClean,
		CorrectWord: correctClean,
		Status:      "active",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := h.repo.SaveSpellcheckDictionary(ctx, entry); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to save spellcheck rule: %v", err)})
		return
	}

	// Invalidate caches
	_ = h.cache.DeleteTenantCache(ctx, tenantID)

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Spellcheck rule created successfully",
	})
}

func (h *AdminHandler) DeleteSpellcheck(c *gin.Context) {
	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-Tenant-ID header is required"})
		return
	}

	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Spellcheck ID is required"})
		return
	}

	ctx := c.Request.Context()
	if err := h.repo.DeleteSpellcheckRule(ctx, tenantID, id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to delete spellcheck rule: %v", err)})
		return
	}

	// Invalidate caches
	_ = h.cache.DeleteTenantCache(ctx, tenantID)

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Spellcheck rule deleted successfully",
	})
}

func (h *AdminHandler) newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
