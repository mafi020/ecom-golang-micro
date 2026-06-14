package middleware

import (
	"log/slog"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/mafi020/ecom-golang-micro/internal/logger"
)

func ContextLoggerMiddleware() gin.HandlerFunc {
	baseLogger := logger.NewJSONLogger()

	return func(c *gin.Context) {
		userIDStr := c.GetHeader("X-User-ID")
		userRole := c.GetHeader("X-User-Role")

		// Prepare fields slice array safely
		logAttrs := []any{
			slog.String("path", c.Request.URL.Path),
			slog.String("method", c.Request.Method),
		}

		// If user_id header is passed down from Gateway, convert it to a type-safe integer representation
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

		// Attach tracing attributes directly to this request log instance
		requestLogger := baseLogger.With(logAttrs...)

		// Bake the tailored logger straight into the standard http.Request context
		ctxWithLogger := logger.ToContext(c.Request.Context(), requestLogger)
		c.Request = c.Request.WithContext(ctxWithLogger)

		c.Next()
	}
}
