package cart

import (
	"github.com/gin-gonic/gin"
	"github.com/mafi020/ecom-golang-micro/config"
	"github.com/mafi020/ecom-golang-micro/internal/delivery/http/handler"
	"github.com/mafi020/ecom-golang-micro/internal/delivery/http/middleware"
)

func RegisterHTTPHandlers(r *gin.Engine, uc *Usecases, cfg *config.Config) {
	cartHandler := handler.NewCartHandler(uc.CartUC)

	// Gateway strips /api/cart, service receives /, /items, /items/:product_id
	cart := r.Group("")
	cart.Use(middleware.AuthMiddleware())
	{
		cart.GET("", cartHandler.GetCart)
		cart.DELETE("", cartHandler.ClearCart)
		cart.POST("/items", cartHandler.AddItem)
		cart.PUT("/items/:product_id", cartHandler.UpdateItem)
		cart.DELETE("/items/:product_id", cartHandler.RemoveItem)
	}
}
