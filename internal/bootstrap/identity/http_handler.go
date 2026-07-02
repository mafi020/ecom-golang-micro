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

	auth := r.Group("/auth")
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
		auth.POST("/refresh", authHandler.RefreshToken)
	}

	// Protected — gateway strips /api, forwards /auth/logout and /users/*
	// Gateway already verified the JWT, so AuthMiddleware here just reads headers
	protected := r.Group("")
	protected.Use(middleware.AuthMiddleware())
	{
		protected.POST("/auth/logout", authHandler.Logout)

		users := protected.Group("/users")
		{
			users.GET("/:id", userHandler.GetUserByID)
			users.GET("", userHandler.GetUsers)
		}
	}
}
