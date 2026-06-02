package bootstrap

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mafi020/ecom-golang/config"
	"github.com/mafi020/ecom-golang/internal/delivery/http/handler"
	"github.com/mafi020/ecom-golang/internal/delivery/http/middleware"
)

func RegisterHTTPHandlers(r *gin.Engine, uc *Usecases, cfg *config.Config) {
	// Health Check - Still useful for internal service health checks
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})

	authHandler := handler.NewAuthHandler(uc.AuthUC)
	userHandler := handler.NewUserHandler(uc.UserUC)
	categoryHandler := handler.NewCategoryHandler(uc.CategoryUC)
	productHandler := handler.NewProductHandler(uc.ProductUC)
	orderHandler := handler.NewOrderHandler(uc.OrderUC)
	cartHandler := handler.NewCartHandler(uc.CartUC)
	paymentHandler := handler.NewPaymentHandler(uc.PaymentUC)

	api := r.Group("/api") // Ensure this matches your gateway's URL prefix matching
	{
		// 1. PUBLIC ROUTES GROUP (Bypasses Auth)
		authGroup := api.Group("/auth")
		{
			authGroup.POST("/register", authHandler.Register)
			authGroup.POST("/login", authHandler.Login)
			authGroup.POST("/refresh", authHandler.RefreshToken)
		}

		api.GET("/products", productHandler.GetProducts)
		api.GET("/products/:id", productHandler.GetProductByID)
		api.GET("/categories", categoryHandler.GetAllCategories)
		api.GET("/categories/:id", categoryHandler.GetCategoryByID)
		api.GET("/categories/:id/products", categoryHandler.GetCategoryByIDWithProducts)

		// 2. PROTECTED ROUTES GROUP (Requires Auth Verification)
		// This now executes our lightweight header-checking middleware
		protected := api.Group("")
		protected.Use(middleware.AuthMiddleware())
		{
			protected.POST("/auth/logout", authHandler.Logout)

			userGroup := protected.Group("/users")
			{
				userGroup.GET("/:id", userHandler.GetUserByID)
				userGroup.GET("", userHandler.GetUsers)
			}

			categoryGroup := protected.Group("/categories")
			{
				categoryGroup.POST("", categoryHandler.CreateCategory)
				categoryGroup.PUT("/:id", categoryHandler.UpdateCategory)
				categoryGroup.DELETE("/:id", categoryHandler.DeleteCategory)
			}

			productGroup := protected.Group("/products")
			{
				productGroup.POST("", productHandler.CreateProduct)
				productGroup.PUT("/:id", productHandler.UpdateProduct)
				productGroup.DELETE("/:id", productHandler.DeleteProduct)
			}

			cartGroup := protected.Group("/cart")
			{
				cartGroup.GET("", cartHandler.GetCart)
				cartGroup.DELETE("", cartHandler.ClearCart)
				cartGroup.POST("/items", cartHandler.AddItem)
				cartGroup.PUT("/items/:product_id", cartHandler.UpdateItem)
				cartGroup.DELETE("/items/:product_id", cartHandler.RemoveItem)
			}

			orderGroup := protected.Group("/orders")
			{
				orderGroup.POST("", orderHandler.PlaceOrder)
				orderGroup.GET("", orderHandler.GetOrdersByUserID)
				orderGroup.GET("/:id", orderHandler.GetOrderByID)
			}

			paymentGroup := protected.Group("/payments")
			{
				paymentGroup.POST("/online", paymentHandler.PayOnline)
				paymentGroup.POST("/cod", paymentHandler.PayCOD)
				paymentGroup.GET("/order/:order_id", paymentHandler.GetPaymentByOrderID)
			}

			// ADMIN ROUTES (Can easily read from X-User-Role header)
			// adminGroup := protected.Group("/admin")
			// adminGroup.Use(middleware.RequireRole("admin"))
			// {
			// 	adminGroup.PUT("/payments/order/:order_id/collect", paymentHandler.CollectCOD)
			// 	adminGroup.PUT("/orders/:id/status", orderHandler.UpdateStatus)
			// }
		}
	}
}
