package utils

import (
	"context"
	"log/slog"
	"strconv"

	"github.com/mafi020/ecom-golang-micro/internal/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// UnaryServerLoggerInterceptor extracts context identity keys over the gRPC wire threshold
func UnaryServerLoggerInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		// 1. Initialize tracking parameters with the execution target endpoint name
		logAttrs := []any{
			slog.String("grpc.method", info.FullMethod),
		}

		// 2. Extract context metadata block transmitted by incoming client connections
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if userIDs := md.Get("x-user-id"); len(userIDs) > 0 && userIDs[0] != "" {
				if userID, err := strconv.ParseInt(userIDs[0], 10, 64); err == nil {
					logAttrs = append(logAttrs, slog.Int64("user_id", userID))
				} else {
					logAttrs = append(logAttrs, slog.String("user_id", userIDs[0]))
				}
			}
			if roles := md.Get("x-user-role"); len(roles) > 0 && roles[0] != "" {
				logAttrs = append(logAttrs, slog.String("user_role", roles[0]))
			}
		}

		// 3. Derive a dedicated request-scoped logger from the global system Singleton
		requestLogger := slog.Default().With(logAttrs...)

		// 4. Inject the scoped logger context down into the handler execution thread
		ctxWithLogger := logger.ToContext(ctx, requestLogger)

		// Execute the underlying gRPC method
		return handler(ctxWithLogger, req)
	}
}
