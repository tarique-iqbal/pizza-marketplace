package otelamqp

import (
	"context"
	"strings"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	"go.opentelemetry.io/otel/trace"
)

func StartConsume(
	ctx context.Context,
	queue, exchange, routingKey string,
	msg amqp.Delivery,
) (context.Context, trace.Span) {
	ctx = otel.GetTextMapPropagator().Extract(ctx, HeaderCarrier(msg.Headers))

	destination := consumeDestination(exchange, routingKey, queue)

	attrs := []attribute.KeyValue{
		semconv.MessagingSystemRabbitMQ,
		semconv.MessagingOperationTypeProcess,
		semconv.MessagingOperationName("process"),
		semconv.MessagingDestinationName(destination),
		semconv.MessagingRabbitMQMessageDeliveryTag(int(msg.DeliveryTag)),
	}
	if routingKey != "" {
		attrs = append(attrs, semconv.MessagingRabbitMQDestinationRoutingKey(routingKey))
	}
	if msg.MessageId != "" {
		attrs = append(attrs, semconv.MessagingMessageID(msg.MessageId))
	}

	return otel.Tracer(scopeName).Start(ctx, "process "+destination,
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(attrs...),
	)
}

func consumeDestination(exchange, routingKey, queue string) string {
	if routingKey == queue {
		queue = ""
	}

	parts := make([]string, 0, 3)
	for _, part := range []string{exchange, routingKey, queue} {
		if part != "" {
			parts = append(parts, part)
		}
	}

	return strings.Join(parts, ":")
}
