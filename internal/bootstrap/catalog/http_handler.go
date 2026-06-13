package catalog

import (
	"github.com/gin-gonic/gin"
	"github.com/mafi020/ecom-golang-micro/config"
	"github.com/mafi020/ecom-golang-micro/internal/delivery/http/handler"
	"github.com/mafi020/ecom-golang-micro/internal/delivery/http/middleware"
)

func RegisterHTTPHandlers(r *gin.Engine, uc *Usecases, cfg *config.Config) {
	categoryHandler := handler.NewCategoryHandler(uc.CategoryUC)
	productHandler := handler.NewProductHandler(uc.ProductUC)

	protected := r.Group("")
	protected.Use(middleware.AuthMiddleware())

	// Products — proxy strips /api/products, service receives / and /:id
	r.GET("/products", productHandler.GetProducts)
	r.GET("/products/:id", productHandler.GetProductByID)
	protected.POST("/products", productHandler.CreateProduct)
	protected.PUT("/products/:id", productHandler.UpdateProduct)
	protected.DELETE("/products/:id", productHandler.DeleteProduct)

	// Categories — proxy strips /api/categories, service receives / and /:id
	r.GET("/categories", categoryHandler.GetAllCategories)
	r.GET("/categories/:id", categoryHandler.GetCategoryByID)
	r.GET("/categories/:id/products", categoryHandler.GetCategoryByIDWithProducts)
	protected.POST("/categories", categoryHandler.CreateCategory)
	protected.PUT("/categories/:id", categoryHandler.UpdateCategory)
	protected.DELETE("/categories/:id", categoryHandler.DeleteCategory)
}
