package logger

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
)

type ctxKey struct{}

var globalLogger *slog.Logger

func init() {
	// 🚀 SINGLETON INITIALIZATION: This runs exactly once when the package is loaded
	handler := &ConditionalSourceHandler{
		infoHandler: slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level:     slog.LevelInfo,
			AddSource: false, // Clean and fast for INFO logs
		}),
		errorHandler: slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level:       slog.LevelInfo,
			AddSource:   true, // Automatically captures file/line/func for ERROR logs
			ReplaceAttr: formatLogOrigin,
		}),
	}

	globalLogger = slog.New(handler)
	slog.SetDefault(globalLogger) // Registers it as the system-wide global Singleton
}

// FromContext extracts the request-scoped logger from context, falling back to the global Singleton
func FromContext(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(ctxKey{}).(*slog.Logger); ok {
		return logger
	}
	return globalLogger
}

// ToContext embeds an existing logger into the provided context
func ToContext(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, logger)
}

// ── OPTIMIZED ALLOCATION-FREE CONDITIONAL HANDLER ────────────────────────────

type ConditionalSourceHandler struct {
	infoHandler  slog.Handler
	errorHandler slog.Handler
}

func (h *ConditionalSourceHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return true
}

func (h *ConditionalSourceHandler) Handle(ctx context.Context, r slog.Record) error {
	// 🚀 ZERO ALLOCATIONS: Routes directly to pre-initialized Singleton handlers
	if r.Level >= slog.LevelError {
		return h.errorHandler.Handle(ctx, r)
	}
	return h.infoHandler.Handle(ctx, r)
}

func (h *ConditionalSourceHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &ConditionalSourceHandler{
		infoHandler:  h.infoHandler.WithAttrs(attrs),
		errorHandler: h.errorHandler.WithAttrs(attrs),
	}
}

func (h *ConditionalSourceHandler) WithGroup(name string) slog.Handler {
	return &ConditionalSourceHandler{
		infoHandler:  h.infoHandler.WithGroup(name),
		errorHandler: h.errorHandler.WithGroup(name),
	}
}

// ── CODE-ORIGIN FORMATTER ───────────────────────────────────────────────────

func formatLogOrigin(groups []string, a slog.Attr) slog.Attr {
	if a.Key == slog.SourceKey {
		source, ok := a.Value.Any().(*slog.Source)
		if !ok {
			return a
		}

		cleanFile := source.File
		if cwd, err := os.Getwd(); err == nil {
			if rel, err := filepath.Rel(cwd, source.File); err == nil {
				cleanFile = rel
			}
		}

		return slog.Attr{
			Key: "origin",
			Value: slog.GroupValue(
				slog.String("file", cleanFile),
				slog.Int("line", source.Line),
				slog.String("func", source.Function),
			),
		}
	}
	return a
}
