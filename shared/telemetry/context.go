package telemetry

import (
	"context"
	"crypto/rand"

	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

const traceParentKey = "traceparent"

func TraceParent(ctx context.Context) string {
	carrier := propagation.MapCarrier{}
	propagation.TraceContext{}.Inject(ctx, carrier)

	return carrier[traceParentKey]
}

func WithTraceParent(ctx context.Context, traceParent string) context.Context {
	if traceParent == "" {
		return ctx
	}

	return propagation.TraceContext{}.Extract(ctx, propagation.MapCarrier{traceParentKey: traceParent})
}

func LinkTraceParent(ctx context.Context, traceParent string) {
	linked := trace.SpanContextFromContext(WithTraceParent(context.Background(), traceParent))
	if !linked.IsValid() {
		return
	}

	trace.SpanFromContext(ctx).AddLink(trace.Link{SpanContext: linked})
}

func WithoutTracing(ctx context.Context) context.Context {
	var traceID trace.TraceID
	var spanID trace.SpanID
	_, _ = rand.Read(traceID[:])
	_, _ = rand.Read(spanID[:])

	return trace.ContextWithSpanContext(ctx, trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: traceID,
		SpanID:  spanID,
	}))
}
