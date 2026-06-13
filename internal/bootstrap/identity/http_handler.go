package identity

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mafi020/ecom-golang-micro/config"
	"github.com/mafi020/ecom-golang-micro/internal/delivery/http/handler"
	"github.com/mafi020/ecom-golang-micro/internal/delivery/http/middleware"
)

func RegisterHTTPHandlers(r *gin.Engine, uc *Usecases, cfg *config.Config) {
	// Health Check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	authHandler := handler.NewAuthHandler(uc.AuthUC)
	userHandler := handler.NewUserHandler(uc.UserUC)

	// ── 1. PUBLIC AUTH ENGINES ───────────────────────────────────────────────
	// Gateway forwards "/api/auth/login" -> trimmed to "/login"
	// Gateway forwards "/api/auth/register" -> trimmed to "/register"
	// Gateway forwards "/api/auth/refresh" -> trimmed to "/refresh"
	r.POST("/register", authHandler.Register)
	r.POST("/login", authHandler.Login)
	r.POST("/refresh", authHandler.RefreshToken)

	// ── 2. PROTECTED AUTH ENGINES ────────────────────────────────────────────
	// Gateway forwards "/api/auth/logout" -> trimmed to "/logout"
	r.POST("/logout", authHandler.Logout)

	// ── 3. PROTECTED USER ENGINES ────────────────────────────────────────────
	// Gateway forwards "/api/users/:id" -> trimmed to "/:id"
	// Gateway forwards "/api/users" -> trimmed to "/"
	users := r.Group("/")
	users.Use(middleware.AuthMiddleware())
	{
		users.GET("/:id", userHandler.GetUserByID)
		users.GET("/", userHandler.GetUsers) // 🚀 FIXED: Changed from "" to "/" to catch root requests cleanly
	}
}
