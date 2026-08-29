package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"product-service/internal/entity"
	"product-service/internal/service"
)

type ProductHandler struct {
	svc service.ProductService
}

func NewProductHandler(svc service.ProductService) *ProductHandler {
	return &ProductHandler{svc: svc}
}

type CreateProductRequest struct {
	Name             string  `json:"name" binding:"required"`
	Description      string  `json:"description"`
	CategoryID       *string `json:"category_id"`
	Brand            string  `json:"brand"`
	Price            float64 `json:"price" binding:"required,gte=0"`
	ImageURL         string  `json:"image_url"`
	Inventory        int     `json:"inventory" binding:"required,gte=0"`
	OriginalLanguage string  `json:"original_language"`
	Featured         bool    `json:"featured"`
}

func (h *ProductHandler) Create(c *gin.Context) {
	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing X-Tenant-ID header"})
		return
	}

	var req CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	productReq := entity.Product{
		Name:             req.Name,
		Description:      req.Description,
		CategoryID:       req.CategoryID,
		Brand:            req.Brand,
		Price:            req.Price,
		ImageURL:         req.ImageURL,
		Inventory:        req.Inventory,
		OriginalLanguage: req.OriginalLanguage,
		Featured:         req.Featured,
	}

	product, err := h.svc.CreateProduct(c.Request.Context(), tenantID, productReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Product created successfully",
		"product": product,
	})
}
