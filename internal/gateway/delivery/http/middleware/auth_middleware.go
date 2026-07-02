package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

var publicEndpoints = map[string]bool{
	"/api/auth/login":    true,
	"/api/auth/register": true,
	"/api/auth/refresh":  true,
	"/api/products":      true, // Allows users to view products without logging in
	"/api/categories":    true, // Allows users to view categories without logging in
}

func AuthMiddleware(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if the request path is in the public endpoints list
		// if publicEndpoints[c.Request.URL.Path] {
		// 	c.Next()
		// 	return
		// }

		isPublic := publicEndpoints[c.Request.URL.Path]

		// Proceed with JWT validation for all other protected routes
		authHeader := c.GetHeader("Authorization")

		// 1. If it's a public route and there is no token, treat them as an anonymous guest cleanly
		if isPublic && (authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ")) {
			c.Next()
			return
		}

		// 2. If it's a private route and token is missing, block them instantly
		if !isPublic && (authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ")) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing or invalid token signature"})
			c.Abort()
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

		token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (any, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method")
			}
			return []byte(jwtSecret), nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			c.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token claims"})
			c.Abort()
			return
		}

		// Set credentials on context
		if sub, ok := claims["sub"].(float64); ok {
			c.Set("user_id", int64(sub))
		}
		if role, ok := claims["role"].(string); ok {
			c.Set("role", role)
		}

		c.Next()
	}
}
