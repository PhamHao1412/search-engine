package route

import (
	v1 "search-service/internal/handler/rest/v1"

	"github.com/gin-gonic/gin"
)

func V1Router(r *gin.Engine, searchHandler *v1.SearchHandler, adminHandler *v1.AdminHandler) {
	apiV1 := r.Group("/api/v1")

	apiV1.GET("/search", searchHandler.Search)
	apiV1.POST("/search/sync", searchHandler.SyncAll)
	apiV1.POST("/analytics/click", searchHandler.TrackClick)
	apiV1.GET("/suggest", searchHandler.Suggest)
	apiV1.GET("/products/:id", searchHandler.GetProductByID)
	apiV1.GET("/spellcheck", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "Spellcheck API - Under construction"})
	})

	apiV1.GET("/admin/tenants", adminHandler.GetTenants)
	apiV1.GET("/admin/ai/suggestions", adminHandler.GetAISuggestions)
	apiV1.POST("/admin/ai/suggestions/:id/approve", adminHandler.ApproveAISuggestion)
	apiV1.POST("/admin/ai/suggestions/:id/reject", adminHandler.RejectAISuggestion)
	apiV1.POST("/admin/ai/suggestions/generate", adminHandler.GenerateAISuggestions)
	apiV1.GET("/admin/dictionaries/spellcheck", adminHandler.GetSpellcheckRules)
	apiV1.GET("/admin/dictionaries/synonyms", adminHandler.GetSearchSynonyms)
}
