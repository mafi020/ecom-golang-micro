package logger

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
)

type ctxKey struct{}

// 🚀 FIXED: Default global engine boots up with the conditional source filtering engine wrapper
var defaultLogger = slog.New(NewConditionalSourceHandler(slog.LevelInfo))

// ToContext embeds an existing logger into the provided context
func ToContext(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, logger)
}

// FromContext extracts the request-scoped logger from context.
func FromContext(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(ctxKey{}).(*slog.Logger); ok {
		return logger
	}
	return defaultLogger
}

// NewJSONLogger initializes a base production-ready structured slog engine
func NewJSONLogger() *slog.Logger {
	return slog.New(NewConditionalSourceHandler(slog.LevelInfo))
}

// ── CONDITIONAL SOURCE LOG HANDLER WRAPPER ───────────────────────────────────

type ConditionalSourceHandler struct {
	baseLevel slog.Level
}

// NewConditionalSourceHandler provisions an isolated JSON runtime stream with variable schema layouts
func NewConditionalSourceHandler(level slog.Level) slog.Handler {
	return &ConditionalSourceHandler{baseLevel: level}
}

func (h *ConditionalSourceHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.baseLevel
}

func (h *ConditionalSourceHandler) Handle(ctx context.Context, r slog.Record) error {
	// 🚀 DYNAMIC TOGGLE MECHANISM: Enforce structural parameters exclusively on LevelError execution
	addSource := r.Level >= slog.LevelError

	jsonHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:       h.baseLevel,
		AddSource:   addSource, // Automatically turns true for ERROR, false for INFO
		ReplaceAttr: formatLogOrigin,
	})

	return jsonHandler.Handle(ctx, r)
}

func (h *ConditionalSourceHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h // Returns handler chain to support streaming properties
}

func (h *ConditionalSourceHandler) WithGroup(name string) slog.Handler {
	return h
}

// ── CODE-ORIGIN FORMATTER ───────────────────────────────────────────────────

func formatLogOrigin(groups []string, a slog.Attr) slog.Attr {
	if a.Key == slog.SourceKey {
		source, ok := a.Value.Any().(*slog.Source)
		if !ok {
			return a
		}

		// Clean up absolute directory paths down to crisp, project-relative landmarks
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
				// slog.String("func", source.Function),
			),
		}
	}
	return a
}
