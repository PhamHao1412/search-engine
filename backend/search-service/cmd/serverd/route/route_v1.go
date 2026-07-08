package route

import (
	v1 "search-service/internal/handler/rest/v1"

	"github.com/gin-gonic/gin"
)

func V1Router(r *gin.Engine, searchHandler *v1.SearchHandler, adminHandler *v1.AdminHandler) {
	apiV1 := r.Group("/api/v1")

	apiV1.GET("/search", searchHandler.Search)
	apiV1.GET("/search/hot-keywords", searchHandler.GetHotKeywords)
	apiV1.POST("/search/sync", searchHandler.SyncAll)
	apiV1.POST("/analytics/click", searchHandler.TrackClick)
	apiV1.GET("/suggest", searchHandler.Suggest)
	apiV1.GET("/products/:id", searchHandler.GetProductByID)
	apiV1.GET("/spellcheck", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "Spellcheck API - Under construction"})
	})

	apiV1.GET("/admin/tenants", adminHandler.GetTenants)
	apiV1.GET("/admin/analytics/summary", adminHandler.GetAnalyticsSummary)
	apiV1.POST("/admin/analytics/trigger", adminHandler.TriggerAnalyticsAggregation)
	apiV1.GET("/admin/ai/suggestions", adminHandler.GetAISuggestions)
	apiV1.POST("/admin/ai/suggestions/:id/approve", adminHandler.ApproveAISuggestion)
	apiV1.POST("/admin/ai/suggestions/:id/reject", adminHandler.RejectAISuggestion)
	apiV1.POST("/admin/ai/suggestions/generate", adminHandler.GenerateAISuggestions)
	apiV1.POST("/admin/assistant/chat", adminHandler.ChatWithAssistant)
	apiV1.GET("/admin/assistant/conversations", adminHandler.ListConversations)
	apiV1.POST("/admin/assistant/conversations", adminHandler.CreateConversation)
	apiV1.GET("/admin/assistant/conversations/:id/messages", adminHandler.GetConversationMessages)
	apiV1.DELETE("/admin/assistant/conversations/:id", adminHandler.DeleteConversation)
	apiV1.POST("/admin/assistant/messages/:id/action", adminHandler.UpdateActionState)
	apiV1.GET("/admin/dictionaries/spellcheck", adminHandler.GetSpellcheckRules)
	apiV1.POST("/admin/dictionaries/spellcheck", adminHandler.CreateSpellcheck)
	apiV1.DELETE("/admin/dictionaries/spellcheck/:id", adminHandler.DeleteSpellcheck)
	apiV1.GET("/admin/dictionaries/synonyms", adminHandler.GetSearchSynonyms)
	apiV1.POST("/admin/dictionaries/synonyms", adminHandler.CreateSynonym)
	apiV1.DELETE("/admin/dictionaries/synonyms/:id", adminHandler.DeleteSynonym)
}
