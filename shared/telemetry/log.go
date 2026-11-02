package telemetry

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel/trace"
)

// spanContextHandler adds trace_id/span_id to every record whose context
// carries an active span. It's meant to wrap the base handler at the point a
// logger is stashed into context (e.g. identity-service's logobs.WithContext
// on the API side, once per request) rather than at every call site.
type spanContextHandler struct {
	slog.Handler
}

// NewSlogHandler wraps base so every record it handles gets trace_id/span_id
// attributes from the active span in ctx, if any.
func NewSlogHandler(base slog.Handler) slog.Handler {
	return &spanContextHandler{Handler: base}
}

func (h *spanContextHandler) Handle(ctx context.Context, record slog.Record) error {
	if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
		record.AddAttrs(
			slog.String("trace_id", sc.TraceID().String()),
			slog.String("span_id", sc.SpanID().String()),
		)
	}

	return h.Handler.Handle(ctx, record)
}

func (h *spanContextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &spanContextHandler{Handler: h.Handler.WithAttrs(attrs)}
}

func (h *spanContextHandler) WithGroup(name string) slog.Handler {
	return &spanContextHandler{Handler: h.Handler.WithGroup(name)}
}
