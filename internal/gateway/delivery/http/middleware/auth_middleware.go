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
		if publicEndpoints[c.Request.URL.Path] {
			c.Next()
			return
		}
		// Proceed with JWT validation for all other protected routes
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing or invalid token"})
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

		userID := int64(claims["sub"].(float64))
		userRole := claims["role"].(string)

		c.Set("user_id", userID)
		c.Set("role", userRole)

		c.Next()
	}
}
