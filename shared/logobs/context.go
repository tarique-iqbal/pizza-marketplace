package logobs

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel/trace"
)

type contextKey struct{}

func WithContext(
	ctx context.Context,
	logger *slog.Logger,
) context.Context {
	if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
		logger = logger.With(
			"trace_id", sc.TraceID().String(),
			"span_id", sc.SpanID().String(),
		)
	}

	return context.WithValue(ctx, contextKey{}, logger)
}

func FromContext(ctx context.Context) *slog.Logger {
	logger, ok := ctx.Value(contextKey{}).(*slog.Logger)
	if !ok {
		return slog.Default()
	}

	return logger
}
