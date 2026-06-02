package router

import (
	"github.com/gin-gonic/gin"
	"github.com/mafi020/ecom-golang/internal/gateway/delivery/http/handler"
)

// MapPublicRoutes attaches all open, unauthenticated endpoints to the gateway
func MapPublicRoutes(api *gin.RouterGroup, monolith *handler.ProxyHandler) {
	auth := api.Group("/auth")
	{
		auth.POST("/login", monolith.ProxyRequest())
		auth.POST("/register", monolith.ProxyRequest())
		auth.POST("/refresh", monolith.ProxyRequest())
	}

	// Catalog open routes
	api.GET("/products", monolith.ProxyRequest())
	api.GET("/products/:id", monolith.ProxyRequest())
	api.GET("/categories", monolith.ProxyRequest())
	api.GET("/categories/:id", monolith.ProxyRequest())
	api.GET("/categories/:id/products", monolith.ProxyRequest())
}
