package otelamqp

import (
	"context"
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	"go.opentelemetry.io/otel/trace"
)

func useTestTracing() *tracetest.SpanRecorder {
	recorder := tracetest.NewSpanRecorder()
	otel.SetTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder)))
	otel.SetTextMapPropagator(propagation.TraceContext{})

	return recorder
}

func findSpan(t *testing.T, recorder *tracetest.SpanRecorder, name string) sdktrace.ReadOnlySpan {
	t.Helper()

	for _, s := range recorder.Ended() {
		if s.Name() == name {
			return s
		}
	}

	t.Fatalf("expected a span named %q", name)

	return nil
}

func attributeValue(span sdktrace.ReadOnlySpan, key attribute.Key) (attribute.Value, bool) {
	for _, attr := range span.Attributes() {
		if attr.Key == key {
			return attr.Value, true
		}
	}

	return attribute.Value{}, false
}

func TestStartConsume_ChildOfProducerSpan(t *testing.T) {
	recorder := useTestTracing()

	headers := amqp.Table{}
	_, producer := StartPublish(context.Background(), "identity.events", "user.registered", headers)
	producer.End()

	msg := amqp.Delivery{Headers: headers, DeliveryTag: 7, MessageId: "msg-1"}
	ctx, span := StartConsume(context.Background(), "email_queue", "identity.events", "user.registered", msg)
	span.End()

	consumer := findSpan(t, recorder, "process identity.events:user.registered:email_queue")

	if consumer.SpanKind() != trace.SpanKindConsumer {
		t.Fatalf("expected a consumer span, got %s", consumer.SpanKind())
	}

	if consumer.Parent().SpanID() != producer.SpanContext().SpanID() {
		t.Fatal("expected the consumer span to be a child of the producer span")
	}

	if consumer.SpanContext().TraceID() != producer.SpanContext().TraceID() {
		t.Fatal("expected the consumer span to join the producer's trace")
	}

	if trace.SpanContextFromContext(ctx).SpanID() != consumer.SpanContext().SpanID() {
		t.Fatal("expected the returned context to carry the consumer span")
	}

	want := map[attribute.Key]attribute.Value{
		semconv.MessagingDestinationNameKey: attribute.StringValue(
			"identity.events:user.registered:email_queue",
		),
		semconv.MessagingOperationTypeKey:                 attribute.StringValue("process"),
		semconv.MessagingRabbitMQDestinationRoutingKeyKey: attribute.StringValue("user.registered"),
		semconv.MessagingRabbitMQMessageDeliveryTagKey:    attribute.IntValue(7),
		semconv.MessagingMessageIDKey:                     attribute.StringValue("msg-1"),
	}
	for key, value := range want {
		got, ok := attributeValue(consumer, key)
		if !ok || got != value {
			t.Errorf("attribute %s = %v, want %v", key, got.Emit(), value.Emit())
		}
	}
}

func TestStartConsume_WithoutTraceParent_StartsRoot(t *testing.T) {
	recorder := useTestTracing()

	msg := amqp.Delivery{Headers: amqp.Table{}}
	_, span := StartConsume(context.Background(), "email_queue", "identity.events", "user.registered", msg)
	span.End()

	consumer := findSpan(t, recorder, "process identity.events:user.registered:email_queue")

	if consumer.Parent().IsValid() {
		t.Fatal("expected a root consumer span without a traceparent header")
	}

	if _, ok := attributeValue(consumer, semconv.MessagingMessageIDKey); ok {
		t.Fatal("expected no message id attribute for a message without one")
	}
}

func TestConsumeDestination(t *testing.T) {
	tests := []struct {
		exchange, routingKey, queue, want string
	}{
		{"identity.events", "user.registered", "email_queue", "identity.events:user.registered:email_queue"},
		{"identity.events", "email_queue", "email_queue", "identity.events:email_queue"},
		{"", "user.registered", "email_queue", "user.registered:email_queue"},
		{"", "email_queue", "email_queue", "email_queue"},
		{"orders.broadcast", "", "email_queue", "orders.broadcast:email_queue"},
	}

	for _, tt := range tests {
		got := consumeDestination(tt.exchange, tt.routingKey, tt.queue)
		if got != tt.want {
			t.Errorf("consumeDestination(%q, %q, %q) = %q, want %q",
				tt.exchange, tt.routingKey, tt.queue, got, tt.want)
		}
	}
}
