package logger

import (
	"context"
	"log/slog"
)

type contextKey struct{}

func WithContext(
	ctx context.Context,
	logger *slog.Logger,
) context.Context {
	return context.WithValue(ctx, contextKey{}, logger)
}

func FromContext(ctx context.Context) *slog.Logger {
	logger, ok := ctx.Value(contextKey{}).(*slog.Logger)
	if !ok {
		return slog.Default()
	}

	return logger
}
