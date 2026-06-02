package main

import (
	"log"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/mafi020/ecom-golang/config"
	"github.com/mafi020/ecom-golang/internal/gateway/delivery/http/handler"
	"github.com/mafi020/ecom-golang/internal/gateway/delivery/http/middleware"
	"github.com/mafi020/ecom-golang/internal/gateway/router"
	"golang.org/x/time/rate"
)

func main() {
	// Load environment variables
	cfg := config.LoadConfig()

	gatewayPort := cfg.Server.APIGatewayPort
	if gatewayPort == "" {
		gatewayPort = "8000"
	}

	monolithURL := cfg.Server.MonolithURL
	log.Printf("GATEWAY IS ROUTING ALL TRAFFIC TO: %s", monolithURL)
	if monolithURL == "" {
		monolithURL = "http://localhost:8080"
	}

	jwtSecret := cfg.JWT.Secret
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET environment variable is missing")
	}

	// Initialize Gin engine
	r := gin.Default()

	r.SetTrustedProxies(nil)
	r.Use(cors.Default())

	rateLimiter := middleware.NewRateLimiter(rate.Every(time.Minute/100), 20)
	r.Use(rateLimiter.Middleware())

	// Initialize the proxy handler pointing to the monolith target
	monolithProxy, err := handler.NewProxyHandler(monolithURL)
	if err != nil {
		log.Fatalf("Failed to initialize monolith proxy: %v", err)
	}

	router.SetupGatewayRouter(r, jwtSecret, monolithProxy)

	if err := r.Run(":" + gatewayPort); err != nil {
		log.Fatalf("Failed to start API Gateway: %v", err)
	}
}
