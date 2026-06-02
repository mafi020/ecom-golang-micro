package middleware

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/mafi020/ecom-golang/internal/apperrors"
	"github.com/mafi020/ecom-golang/internal/delivery/http/utils"
)

// Monolith Auth Middleware simply trusts X-User headers coming from the Gateway
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDStr := c.GetHeader("X-User-ID")
		userRole := c.GetHeader("X-User-Role")

		if userIDStr == "" {
			utils.HandleError(c, &apperrors.UnauthorizedError{Message: "missing trust authentication signature"})
			c.Abort()
			return
		}

		userID, err := strconv.ParseInt(userIDStr, 10, 64)
		if err != nil {
			utils.HandleError(c, &apperrors.UnauthorizedError{Message: "malformed trust authentication metadata"})
			c.Abort()
			return
		}

		// Keep context definitions matching your original usecases exactly
		c.Set("user_id", userID)
		c.Set("role", userRole)

		c.Next()
	}
}
