package router

import (
	"github.com/gin-gonic/gin"
	"github.com/mafi020/ecom-golang-micro/internal/gateway/delivery/http/handler"
)

// MapProtectedRoutes attaches endpoints that require a valid signature header
func MapProtectedRoutes(api *gin.RouterGroup, catalog, cart, order, payment, identity *handler.ProxyHandler) {
	api.POST("/auth/logout", identity.ProxyRequest())

	userGroup := api.Group("/users")
	{
		userGroup.GET("/:id", identity.ProxyRequest())
		userGroup.GET("", identity.ProxyRequest())
	}

	categoryGroup := api.Group("/categories")
	{
		categoryGroup.POST("", catalog.ProxyRequest())
		categoryGroup.PUT("/:id", catalog.ProxyRequest())
		categoryGroup.DELETE("/:id", catalog.ProxyRequest())
	}

	productGroup := api.Group("/products")
	{
		productGroup.POST("", catalog.ProxyRequest())
		productGroup.PUT("/:id", catalog.ProxyRequest())
		productGroup.DELETE("/:id", catalog.ProxyRequest())
	}

	cartGroup := api.Group("/cart")
	{
		cartGroup.GET("", cart.ProxyRequest())
		cartGroup.DELETE("", cart.ProxyRequest())
		cartGroup.POST("/items", cart.ProxyRequest())
		cartGroup.PUT("/items/:product_id", cart.ProxyRequest())
		cartGroup.DELETE("/items/:product_id", cart.ProxyRequest())
	}

	orderGroup := api.Group("/orders")
	{
		orderGroup.POST("", order.ProxyRequest())
		orderGroup.GET("", order.ProxyRequest())
		orderGroup.GET("/:id", order.ProxyRequest())
	}

	paymentGroup := api.Group("/payments")
	{
		paymentGroup.POST("/online", payment.ProxyRequest())
		paymentGroup.POST("/cod", payment.ProxyRequest())
		paymentGroup.GET("/order/:order_id", payment.ProxyRequest())
	}
}
