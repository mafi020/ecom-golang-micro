package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/mafi020/ecom-golang-micro/internal/logger"
)

func ContextLoggerMiddleware() gin.HandlerFunc {
	baseLogger := logger.NewJSONLogger()

	return func(c *gin.Context) {
		// 1. Read tracing headers forwarded by your API Gateway
		userID := c.GetHeader("X-User-ID")
		userRole := c.GetHeader("X-User-Role")

		// 2. Attach tracing attributes to this specific request's log slice
		requestLogger := baseLogger.With(
			"user_id", userID,
			"user_role", userRole,
			"path", c.Request.URL.Path,
			"method", c.Request.Method,
		)

		// 3. Bake the tailored logger straight into the standard http.Request context
		ctxWithLogger := logger.ToContext(c.Request.Context(), requestLogger)
		c.Request = c.Request.WithContext(ctxWithLogger)

		c.Next()
	}
}
