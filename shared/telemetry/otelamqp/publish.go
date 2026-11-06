package otelamqp

import (
	"context"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	"go.opentelemetry.io/otel/trace"
)

const scopeName = "pizza-marketplace/shared/telemetry/otelamqp"

func StartPublish(
	ctx context.Context,
	exchange string,
	routingKey string,
	headers amqp.Table,
) (context.Context, trace.Span) {
	destination := publishDestination(exchange, routingKey)

	ctx, span := otel.Tracer(scopeName).Start(ctx, "send "+destination,
		trace.WithSpanKind(trace.SpanKindProducer),
		trace.WithAttributes(
			semconv.MessagingSystemRabbitMQ,
			semconv.MessagingOperationTypeSend,
			semconv.MessagingOperationName("send"),
			semconv.MessagingDestinationName(destination),
			semconv.MessagingRabbitMQDestinationRoutingKey(routingKey),
		),
	)

	otel.GetTextMapPropagator().Inject(ctx, HeaderCarrier(headers))

	return ctx, span
}

func publishDestination(exchange, routingKey string) string {
	switch {
	case exchange != "" && routingKey != "":
		return exchange + ":" + routingKey
	case exchange != "":
		return exchange
	case routingKey != "":
		return routingKey
	default:
		return "amq.default"
	}
}
