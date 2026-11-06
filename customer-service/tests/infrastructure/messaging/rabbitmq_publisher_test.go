package messaging_test

import (
	"context"
	"os"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"

	"customer-service/internal/infrastructure/messaging"

	"pizza-marketplace/shared/telemetry/otelamqp"
)

func newTestPublisher(t *testing.T) *messaging.RabbitMQPublisher {
	t.Helper()

	url := os.Getenv("RABBITMQ_URL")
	require.NotEmpty(t, url, "RABBITMQ_URL must be set (see .env.test)")

	p, err := messaging.NewRabbitMQPublisher(url)
	require.NoError(t, err)

	t.Cleanup(p.Close)

	return p
}

func TestRabbitMQPublisher_Publish_Success(t *testing.T) {
	p := newTestPublisher(t)

	err := p.Publish(context.Background(), "test.event", []byte(`{"ok":true}`))
	require.NoError(t, err)
}

func TestRabbitMQPublisher_Publish_SurvivesPriorTimeouts(t *testing.T) {
	p := newTestPublisher(t)

	for range 20 {
		shortCtx, cancel := context.WithTimeout(context.Background(), time.Microsecond)
		_ = p.Publish(shortCtx, "test.event", []byte(`{}`))
		cancel()
	}

	time.Sleep(200 * time.Millisecond)

	done := make(chan error, 1)
	go func() {
		done <- p.Publish(context.Background(), "test.event", []byte(`{"final":true}`))
	}()

	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(3 * time.Second):
		t.Fatal("publish hung: a prior timed-out call likely leaked a confirms listener")
	}
}

func bindTestQueue(t *testing.T, exchange, routingKey string) <-chan amqp.Delivery {
	t.Helper()

	conn, err := amqp.Dial(os.Getenv("RABBITMQ_URL"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	ch, err := conn.Channel()
	require.NoError(t, err)

	q, err := ch.QueueDeclare("", false, true, true, false, nil)
	require.NoError(t, err)
	require.NoError(t, ch.QueueBind(q.Name, routingKey, exchange, false, nil))

	deliveries, err := ch.Consume(q.Name, "", true, true, false, false, nil)
	require.NoError(t, err)

	return deliveries
}

func useTestTracing(t *testing.T) (*sdktrace.TracerProvider, *tracetest.SpanRecorder) {
	t.Helper()

	previousProvider := otel.GetTracerProvider()
	previousPropagator := otel.GetTextMapPropagator()
	t.Cleanup(func() {
		otel.SetTracerProvider(previousProvider)
		otel.SetTextMapPropagator(previousPropagator)
	})

	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	return provider, recorder
}

func TestRabbitMQPublisher_Publish_InjectsProducerTraceContext(t *testing.T) {
	provider, recorder := useTestTracing(t)
	p := newTestPublisher(t)
	deliveries := bindTestQueue(t, "customer.events", "test.traced")

	ctx, parent := provider.Tracer("test").Start(context.Background(), "request")
	require.NoError(t, p.Publish(ctx, "test.traced", []byte(`{}`)))
	parent.End()

	var delivery amqp.Delivery
	select {
	case delivery = <-deliveries:
	case <-time.After(5 * time.Second):
		t.Fatal("message was never delivered")
	}

	require.Equal(t, "test.traced", delivery.Headers["x-event-name"])

	got := trace.SpanContextFromContext(
		propagation.TraceContext{}.Extract(context.Background(), otelamqp.HeaderCarrier(delivery.Headers)),
	)
	require.Equal(t, parent.SpanContext().TraceID(), got.TraceID())

	var producer sdktrace.ReadOnlySpan
	for _, s := range recorder.Ended() {
		if s.Name() == "send customer.events:test.traced" {
			producer = s
		}
	}
	require.NotNil(t, producer, "producer span should be ended")
	require.Equal(t, trace.SpanKindProducer, producer.SpanKind())
	require.Equal(t, parent.SpanContext().SpanID(), producer.Parent().SpanID())
	require.Equal(t, producer.SpanContext().SpanID(), got.SpanID(), "header should carry the producer span")
}
