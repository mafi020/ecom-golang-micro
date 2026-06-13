package main

import (
	"log"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/mafi020/ecom-golang-micro/config"
	"github.com/mafi020/ecom-golang-micro/internal/gateway/delivery/http/handler"
	"github.com/mafi020/ecom-golang-micro/internal/gateway/delivery/http/middleware"
	"github.com/mafi020/ecom-golang-micro/internal/gateway/router"
	"golang.org/x/time/rate"
)

func main() {
	// Load environment variables
	cfg := config.LoadConfig()

	gatewayPort := cfg.Server.APIGatewayPort
	if gatewayPort == "" {
		gatewayPort = "8000"
	}

	IdentityPort := cfg.Server.IdentityServiceHTTPPort
	if IdentityPort == "" {
		IdentityPort = "8080"
	}

	catalogPort := cfg.Server.CatalogServiceHTTPPort
	if catalogPort == "" {
		catalogPort = "8081"
	}

	cartPort := cfg.Server.CartServiceHTTPPort
	if cartPort == "" {
		cartPort = "8082"
	}

	orderPort := cfg.Server.OrderServiceHTTPPort
	if orderPort == "" {
		orderPort = "8083"
	}

	paymentPort := cfg.Server.PaymentServiceHTTPPort
	if paymentPort == "" {
		paymentPort = "8084"
	}

	catalogURL := cfg.Server.CatalogServiceURL
	log.Printf("GATEWAY IS ROUTING CATALOG TRAFFIC TO: %s", catalogURL)
	if catalogURL == "" {
		catalogURL = "http://localhost:" + catalogPort
	}

	cartURL := cfg.Server.CartServiceURL
	log.Printf("GATEWAY IS ROUTING CART TRAFFIC TO: %s", cartURL)
	if cartURL == "" {
		cartURL = "http://localhost:" + cartPort
	}

	orderURL := cfg.Server.OrderServiceURL
	log.Printf("GATEWAY IS ROUTING ORDER TRAFFIC TO: %s", orderURL)
	if orderURL == "" {
		orderURL = "http://localhost:" + orderPort
	}

	paymentURL := cfg.Server.PaymentServiceURL
	log.Printf("GATEWAY IS ROUTING ORDER TRAFFIC TO: %s", paymentURL)
	if paymentURL == "" {
		paymentURL = "http://localhost:" + paymentPort
	}

	identityURL := cfg.Server.IdentityServiceURL
	log.Printf("GATEWAY IS ROUTING ORDER TRAFFIC TO: %s", identityURL)
	if identityURL == "" {
		identityURL = "http://localhost:" + IdentityPort
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

	// Inside your API Gateway setup file:
	// Explicitly link the target backend with its respective prefix group string
	identityProxy, _ := handler.NewProxyHandler(identityURL, "/api")

	catalogProxy, _ := handler.NewProxyHandler(catalogURL, "/api")

	cartProxy, _ := handler.NewProxyHandler(cartURL, "/api/cart")
	orderProxy, _ := handler.NewProxyHandler(orderURL, "/api/orders")
	paymentProxy, _ := handler.NewProxyHandler(paymentURL, "/api/payments")

	router.SetupGatewayRouter(
		r,
		jwtSecret,
		identityProxy,
		catalogProxy,
		cartProxy,
		orderProxy,
		paymentProxy,
	)

	if err := r.Run(":" + gatewayPort); err != nil {
		log.Fatalf("Failed to start API Gateway: %v", err)
	}
}
