package router

import (
	"github.com/gin-gonic/gin"
	"github.com/mafi020/ecom-golang-micro/internal/gateway/delivery/http/handler"
	"github.com/mafi020/ecom-golang-micro/internal/gateway/delivery/http/middleware"
)

// SetupGatewayRouter wires up your isolated route files into a unified engine configuration
// func SetupGatewayRouter(r *gin.Engine, jwtSecret string, identity, catalog, cart, order, payment *handler.ProxyHandler) {
// 	// Base API Group
// 	baseAPI := r.Group("/api")

// 	// 1. Mount Public Routes File
// 	MapPublicRoutes(baseAPI, catalog, identity)

// 	// 2. Separate and Mount Protected Routes File
// 	protectedAPI := r.Group("/api")
// 	protectedAPI.Use(middleware.AuthMiddleware(jwtSecret))

// 	MapProtectedRoutes(protectedAPI, catalog, cart, order, payment, identity)
// }

func SetupGatewayRouter(r *gin.Engine, jwtSecret string, identity, catalog, cart, order, payment *handler.ProxyHandler) {
	// 1. Force ALL gateway endpoints to flow through the token evaluation engine first
	baseAPI := r.Group("/api")
	baseAPI.Use(middleware.AuthMiddleware(jwtSecret))

	// 2. Mount your routers directly onto the validated base pipeline
	MapPublicRoutes(baseAPI, catalog, identity)
	MapProtectedRoutes(baseAPI, catalog, cart, order, payment, identity)
}
