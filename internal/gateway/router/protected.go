package router

import (
	"github.com/gin-gonic/gin"
	"github.com/mafi020/ecom-golang/internal/gateway/delivery/http/handler"
)

// MapProtectedRoutes attaches endpoints that require a valid signature header
func MapProtectedRoutes(api *gin.RouterGroup, monolith *handler.ProxyHandler) {
	api.POST("/auth/logout", monolith.ProxyRequest())

	users := api.Group("/users")
	{
		users.GET("/:id", monolith.ProxyRequest())
		users.GET("", monolith.ProxyRequest())
	}

	categories := api.Group("/categories")
	{
		categories.POST("", monolith.ProxyRequest())
		categories.PUT("/:id", monolith.ProxyRequest())
		categories.DELETE("/:id", monolith.ProxyRequest())
	}

	products := api.Group("/products")
	{
		products.POST("", monolith.ProxyRequest())
		products.PUT("/:id", monolith.ProxyRequest())
		products.DELETE("/:id", monolith.ProxyRequest())
	}

	cartGroup := api.Group("/cart")
	{
		cartGroup.GET("", monolith.ProxyRequest())
		cartGroup.DELETE("", monolith.ProxyRequest())
		cartGroup.POST("/items", monolith.ProxyRequest())
		cartGroup.PUT("/items/:product_id", monolith.ProxyRequest())
		cartGroup.DELETE("/items/:product_id", monolith.ProxyRequest())
	}

	orderGroup := api.Group("/orders")
	{
		orderGroup.POST("", monolith.ProxyRequest())
		orderGroup.GET("", monolith.ProxyRequest())
		orderGroup.GET("/:id", monolith.ProxyRequest())
	}

	paymentGroup := api.Group("/payments")
	{
		paymentGroup.POST("/online", monolith.ProxyRequest())
		paymentGroup.POST("/cod", monolith.ProxyRequest())
		paymentGroup.GET("/order/:order_id", monolith.ProxyRequest())
	}

	// ADMIN ROUTES (Can easily read from X-User-Role header)
	// adminGroup := api.Group("/admin")
	// adminGroup.Use(middleware.RequireRole("admin"))
	// {
	// 	adminGroup.PUT("/payments/order/:order_id/collect", monolith.ProxyRequest())
	// 	adminGroup.PUT("/orders/:id/status", monolith.ProxyRequest())
	// }
}
