package route

import (
	v1 "search-service/internal/handler/rest/v1"

	"github.com/gin-gonic/gin"
)

func V1Router(r *gin.Engine, searchHandler *v1.SearchHandler) {
	apiV1 := r.Group("/api/v1")

	apiV1.GET("/search", searchHandler.Search)
	apiV1.POST("/search/sync", searchHandler.SyncAll)
	apiV1.GET("/suggest", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "Suggest API - Under construction"})
	})
	apiV1.GET("/spellcheck", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "Spellcheck API - Under construction"})
	})
}
