package order

import (
	"github.com/gin-gonic/gin"
	"github.com/mafi020/ecom-golang-micro/config"
	"github.com/mafi020/ecom-golang-micro/internal/delivery/http/handler"
	"github.com/mafi020/ecom-golang-micro/internal/delivery/http/middleware"
)

func RegisterHTTPHandlers(r *gin.Engine, uc *Usecases, cfg *config.Config) {
	orderHandler := handler.NewOrderHandler(uc.OrderUC)

	// Gateway strips /api/orders, service receives /, /:id
	order := r.Group("")
	order.Use(middleware.AuthMiddleware())
	{
		order.POST("", orderHandler.PlaceOrder)
		order.GET("", orderHandler.GetOrdersByUserID)
		order.GET("/:id", orderHandler.GetOrderByID)
	}
}
