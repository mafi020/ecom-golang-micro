package router

import (
	"github.com/gin-gonic/gin"
	"github.com/mafi020/ecom-golang-micro/internal/gateway/delivery/http/handler"
)

// MapPublicRoutes attaches all open, unauthenticated endpoints to the gateway
func MapPublicRoutes(api *gin.RouterGroup, catalog, identity *handler.ProxyHandler) {
	authGroup := api.Group("/auth")
	{
		authGroup.POST("/login", identity.ProxyRequest())
		authGroup.POST("/register", identity.ProxyRequest())
		authGroup.POST("/refresh", identity.ProxyRequest())
	}

	api.GET("/products", catalog.ProxyRequest())
	api.GET("/products/:id", catalog.ProxyRequest())
	api.GET("/categories", catalog.ProxyRequest())
	api.GET("/categories/:id", catalog.ProxyRequest())
	api.GET("/categories/:id/products", catalog.ProxyRequest())
}
