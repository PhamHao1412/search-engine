package route

import (
	v1 "product-service/internal/handler/rest/v1"

	"github.com/gin-gonic/gin"
)

func V1Router(
	r *gin.Engine,
	productHandler *v1.ProductHandler,
) {
	apiV1 := r.Group("/api/v1")

	apiV1.POST("/products", productHandler.Create)
}
