package telemetry

import (
	"context"
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

func TestAMQPHeaderCarrier_RoundTripsTraceContext(t *testing.T) {
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    trace.TraceID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		SpanID:     trace.SpanID{1, 2, 3, 4, 5, 6, 7, 8},
		TraceFlags: trace.FlagsSampled,
	})
	ctx := trace.ContextWithSpanContext(context.Background(), sc)

	propagator := propagation.TraceContext{}

	headers := AMQPHeaderCarrier(amqp.Table{})
	propagator.Inject(ctx, headers)

	if headers.Get("traceparent") == "" {
		t.Fatal("expected traceparent header to be set after Inject")
	}

	restored := propagator.Extract(context.Background(), headers)
	restoredSC := trace.SpanContextFromContext(restored)

	if restoredSC.TraceID() != sc.TraceID() {
		t.Fatalf("trace ID mismatch: got %s, want %s", restoredSC.TraceID(), sc.TraceID())
	}

	if restoredSC.SpanID() != sc.SpanID() {
		t.Fatalf("span ID mismatch: got %s, want %s", restoredSC.SpanID(), sc.SpanID())
	}
}

func TestAMQPHeaderCarrier_Keys(t *testing.T) {
	headers := AMQPHeaderCarrier(amqp.Table{"a": "1", "b": "2"})

	keys := headers.Keys()
	if len(keys) != 2 {
		t.Fatalf("expected 2 keys, got %d", len(keys))
	}
}

func TestAMQPHeaderCarrier_GetMissingKey(t *testing.T) {
	headers := AMQPHeaderCarrier(amqp.Table{})

	if got := headers.Get("missing"); got != "" {
		t.Fatalf("expected empty string for missing key, got %q", got)
	}
}
