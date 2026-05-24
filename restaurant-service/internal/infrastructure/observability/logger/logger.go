package logger

import (
	"log/slog"
	"os"
)

func NewLogger(serviceName string) *slog.Logger {
	env := os.Getenv("APP_ENV")

	opts := &slog.HandlerOptions{
		Level: parseLevel(),
	}

	var handler slog.Handler

	switch env {
	case "prod":
		handler = slog.NewJSONHandler(os.Stdout, opts)

	default:
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

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
