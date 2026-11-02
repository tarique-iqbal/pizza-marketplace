package logobs

import (
	"context"
	"log/slog"
	"os"

	"go.opentelemetry.io/contrib/bridges/otelslog"
)

const scopeName = "pizza-marketplace/shared/logobs"

func NewLogger(serviceName string) *slog.Logger {
	env := os.Getenv("APP_ENV")
	level := parseLevel()

	opts := &slog.HandlerOptions{
		Level: level,
	}

	var stdout slog.Handler

	switch env {
	case "prod":
		stdout = slog.NewJSONHandler(os.Stdout, opts)

	default:
		stdout = slog.NewTextHandler(os.Stdout, opts)
	}

	handler := slog.NewMultiHandler(
		stdout,
		&levelHandler{Handler: otelslog.NewHandler(scopeName), level: level},
	)

	return slog.New(handler).With(
		"service", serviceName,
		"env", env,
	)
}

func parseLevel() slog.Level {
	switch os.Getenv("LOG_LEVEL") {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

type levelHandler struct {
	slog.Handler
	level slog.Level
}

func (h *levelHandler) Enabled(ctx context.Context, l slog.Level) bool {
	return l >= h.level && h.Handler.Enabled(ctx, l)
}

func (h *levelHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &levelHandler{Handler: h.Handler.WithAttrs(attrs), level: h.level}
}

func (h *levelHandler) WithGroup(name string) slog.Handler {
	return &levelHandler{Handler: h.Handler.WithGroup(name), level: h.level}
}
