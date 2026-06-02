package router

import (
	"github.com/gin-gonic/gin"
	"github.com/mafi020/ecom-golang/internal/gateway/delivery/http/handler"
	"github.com/mafi020/ecom-golang/internal/gateway/delivery/http/middleware"
)

// SetupGatewayRouter wires up your isolated route files into a unified engine configuration
func SetupGatewayRouter(r *gin.Engine, jwtSecret string, monolith *handler.ProxyHandler) {
	// Base API Group
	baseAPI := r.Group("/api")

	// 1. Mount Public Routes File
	MapPublicRoutes(baseAPI, monolith)

	// 2. Separate and Mount Protected Routes File
	protectedAPI := r.Group("/api")
	protectedAPI.Use(middleware.AuthMiddleware(jwtSecret))

	MapProtectedRoutes(protectedAPI, monolith)
}
