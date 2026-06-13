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
	cfg := config.LoadConfig()

	if cfg.JWT.Secret == "" {
		log.Fatal("JWT_SECRET environment variable is missing")
	}

	gatewayPort := getOrDefault(cfg.Server.APIGatewayPort, "8000")

	// 1. Centralized Routing Registry Map
	services := map[string]struct {
		url, port, defaultPort, prefix string
	}{
		"identity": {cfg.Server.IdentityServiceURL, cfg.Server.IdentityServiceHTTPPort, "8080", "/api"},
		"catalog":  {cfg.Server.CatalogServiceURL, cfg.Server.CatalogServiceHTTPPort, "8081", "/api"},
		"cart":     {cfg.Server.CartServiceURL, cfg.Server.CartServiceHTTPPort, "8082", "/api/cart"},
		"order":    {cfg.Server.OrderServiceURL, cfg.Server.OrderServiceHTTPPort, "8083", "/api/orders"},
		"payment":  {cfg.Server.PaymentServiceURL, cfg.Server.PaymentServiceHTTPPort, "8084", "/api/payments"},
	}

	// 2. Loop Initialization Strategy for Proxies
	proxies := make(map[string]*handler.ProxyHandler)
	for name, svc := range services {
		// Uses actual port configuration value first, falling back to the assigned service standard port fallback
		resolvedPort := getOrDefault(svc.port, svc.defaultPort)
		targetURL := getOrDefault(svc.url, "http://localhost:"+resolvedPort)

		log.Printf("[GATEWAY] Routing %s traffic -> %s (Prefix: %s)", name, targetURL, svc.prefix)

		// 🚀 FIXED: Pass target URL and the correct dynamic path matching prefix to the proxy constructor
		proxy, err := handler.NewProxyHandler(targetURL, svc.prefix)
		if err != nil {
			log.Fatalf("Failed to initialize proxy for %s service: %v", name, err)
		}
		proxies[name] = proxy
	}

	// 3. Mount Engine Global Middlewares
	r := gin.Default()
	r.SetTrustedProxies(nil)
	r.Use(cors.Default())

	rateLimiter := middleware.NewRateLimiter(rate.Every(time.Minute/100), 20)
	r.Use(rateLimiter.Middleware())

	// 4. Clean Router Target Descriptors passing
	router.SetupGatewayRouter(
		r,
		cfg.JWT.Secret,
		proxies["identity"],
		proxies["catalog"],
		proxies["cart"],
		proxies["order"],
		proxies["payment"],
	)

	log.Printf("[GATEWAY] Started engine router tracking block listening securely on port :%s", gatewayPort)
	if err := r.Run(":" + gatewayPort); err != nil {
		log.Fatalf("Failed to start API Gateway: %v", err)
	}
}

// ── UTILITY REFACTOR TOOLS ───────────────────────────────────────────────────

func getOrDefault(val, defaultVal string) string {
	if val == "" {
		return defaultVal
	}
	return val
}
