package middleware

import (
	"log/slog"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/mafi020/ecom-golang-micro/internal/logger"
)

func ContextLoggerMiddleware() gin.HandlerFunc {
	// 🚀 FIXED: Removed custom instantiation. We now leverage the global system-wide Singleton.
	return func(c *gin.Context) {
		userIDStr := c.GetHeader("X-User-ID")
		userRole := c.GetHeader("X-User-Role")

		// Prepare structured context attributes
		logAttrs := []any{
			slog.String("path", c.Request.URL.Path),
			slog.String("method", c.Request.Method),
		}

		// Type-safe header extraction
		if userIDStr != "" {
			if userID, err := strconv.ParseInt(userIDStr, 10, 64); err == nil {
				logAttrs = append(logAttrs, slog.Int64("user_id", userID))
			} else {
				logAttrs = append(logAttrs, slog.String("user_id", userIDStr))
			}
		}

		if userRole != "" {
			logAttrs = append(logAttrs, slog.String("user_role", userRole))
		}

		// 🚀 FIXED: Derive request-scoped log logger from the active pre-configured Singleton engine instance
		requestLogger := slog.Default().With(logAttrs...)

		// Bake the tracking logger context directly into the request pipeline
		ctxWithLogger := logger.ToContext(c.Request.Context(), requestLogger)
		c.Request = c.Request.WithContext(ctxWithLogger)

		c.Next()
	}
}
