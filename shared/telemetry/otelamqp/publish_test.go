package otelamqp

import (
	"context"
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

func TestStartPublish_ProducerSpanChildOfCallerAndInjected(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	ctx, parent := provider.Tracer("test").Start(context.Background(), "request")

	headers := amqp.Table{"x-event-name": "user.registered"}
	ctx, span := StartPublish(ctx, "identity.events", "user.registered", headers)
	span.End()
	parent.End()

	sc := trace.SpanContextFromContext(ctx)

	restored := trace.SpanContextFromContext(
		propagation.TraceContext{}.Extract(context.Background(), HeaderCarrier(headers)),
	)
	if restored.SpanID() != sc.SpanID() || restored.TraceID() != sc.TraceID() {
		t.Fatalf("headers carry %s/%s, want the producer span %s/%s",
			restored.TraceID(), restored.SpanID(), sc.TraceID(), sc.SpanID())
	}

	if headers["x-event-name"] != "user.registered" {
		t.Fatal("expected existing headers to be kept")
	}

	var producer sdktrace.ReadOnlySpan
	for _, s := range recorder.Ended() {
		if s.Name() == "send identity.events" {
			producer = s
		}
	}
	if producer == nil {
		t.Fatal("expected a span named \"send identity.events\"")
	}

	if producer.SpanKind() != trace.SpanKindProducer {
		t.Fatalf("expected a producer span, got %s", producer.SpanKind())
	}

	if producer.Parent().SpanID() != parent.SpanContext().SpanID() {
		t.Fatal("expected the producer span to be a child of the caller's span")
	}
}

func TestStartPublish_WithoutCallerSpan_StartsRootAndInjects(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	otel.SetTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder)))
	otel.SetTextMapPropagator(propagation.TraceContext{})

	headers := amqp.Table{}
	ctx, span := StartPublish(context.Background(), "identity.events", "user.registered", headers)
	span.End()

	restored := trace.SpanContextFromContext(
		propagation.TraceContext{}.Extract(context.Background(), HeaderCarrier(headers)),
	)
	if restored.SpanID() != trace.SpanContextFromContext(ctx).SpanID() {
		t.Fatal("expected the headers to carry the producer span")
	}

	ended := recorder.Ended()
	if len(ended) != 1 {
		t.Fatalf("expected 1 ended span, got %d", len(ended))
	}

	if ended[0].Parent().IsValid() {
		t.Fatal("expected a root producer span without a caller span")
	}
}
