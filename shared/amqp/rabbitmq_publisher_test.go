package amqp

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"

	"pizza-marketplace/shared/telemetry/otelamqp"
)

func newTestPublisher(t *testing.T) *RabbitMQPublisher {
	t.Helper()

	url := os.Getenv("RABBITMQ_URL")
	if url == "" {
		t.Skip("RABBITMQ_URL not set; skipping real-broker test")
	}

	p, err := NewRabbitMQPublisher(url, testExchange)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(p.Close)

	return p
}

func TestRabbitMQPublisher_Publish_Success(t *testing.T) {
	p := newTestPublisher(t)

	if err := p.Publish(context.Background(), "test:1", "test.event", []byte(`{"ok":true}`)); err != nil {
		t.Fatal(err)
	}
}

// Publish must keep serving future calls even when a prior call gives up waiting for its
// own confirmation: amqp091-go broadcasts every confirmation to every listener ever
// registered on a channel via NotifyPublish, with no way to unregister one, so an
// abandoned, unread listener left full would block that broadcast, and therefore every
// future confirmation, forever. A deadline this short usually loses the race against a
// real confirm round trip, though not every iteration is guaranteed to (some return via
// the ctx-already-done guard before doing any real work), so this repeats it to make a
// genuine race loss highly likely.
func TestRabbitMQPublisher_Publish_SurvivesPriorTimeouts(t *testing.T) {
	p := newTestPublisher(t)

	for range 20 {
		shortCtx, cancel := context.WithTimeout(context.Background(), time.Microsecond)
		_ = p.Publish(shortCtx, "test:1", "test.event", []byte(`{}`))
		cancel()
	}

	time.Sleep(200 * time.Millisecond)

	done := make(chan error, 1)
	go func() {
		done <- p.Publish(context.Background(), "test:1", "test.event", []byte(`{"final":true}`))
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("publish hung: a prior timed-out call likely leaked a confirms listener")
	}
}

func bindTestQueue(t *testing.T, exchange, routingKey string) <-chan amqp091.Delivery {
	t.Helper()

	conn, err := amqp091.Dial(os.Getenv("RABBITMQ_URL"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	ch, err := conn.Channel()
	if err != nil {
		t.Fatal(err)
	}

	q, err := ch.QueueDeclare("", false, true, true, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := ch.QueueBind(q.Name, routingKey, exchange, false, nil); err != nil {
		t.Fatal(err)
	}

	deliveries, err := ch.Consume(q.Name, "", true, true, false, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	return deliveries
}

func TestRabbitMQPublisher_Publish_InjectsProducerTraceContext(t *testing.T) {
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

	p := newTestPublisher(t)
	deliveries := bindTestQueue(t, testExchange, "test.traced")

	ctx, parent := provider.Tracer("test").Start(context.Background(), "request")
	if err := p.Publish(ctx, "test:1", "test.traced", []byte(`{}`)); err != nil {
		t.Fatal(err)
	}
	parent.End()

	var delivery amqp091.Delivery
	select {
	case delivery = <-deliveries:
	case <-time.After(5 * time.Second):
		t.Fatal("message was never delivered")
	}

	if delivery.Headers["x-event-name"] != "test.traced" {
		t.Fatalf("x-event-name %v, want test.traced", delivery.Headers["x-event-name"])
	}

	got := trace.SpanContextFromContext(
		propagation.TraceContext{}.Extract(context.Background(), otelamqp.HeaderCarrier(delivery.Headers)),
	)
	if got.TraceID() != parent.SpanContext().TraceID() {
		t.Fatal("message should carry the request's trace")
	}

	var producer sdktrace.ReadOnlySpan
	for _, s := range recorder.Ended() {
		if s.Name() == "send "+testExchange+":test.traced" {
			producer = s
		}
	}
	if producer == nil {
		t.Fatal("producer span should be ended")
	}
	if producer.SpanKind() != trace.SpanKindProducer {
		t.Fatalf("span kind %v, want producer", producer.SpanKind())
	}
	if producer.Parent().SpanID() != parent.SpanContext().SpanID() {
		t.Fatal("producer span should be a child of the request span")
	}
	if got.SpanID() != producer.SpanContext().SpanID() {
		t.Fatal("header should carry the producer span")
	}
}

func TestRabbitMQPublisher_Publish_SetsMessageID(t *testing.T) {
	p := newTestPublisher(t)
	deliveries := bindTestQueue(t, testExchange, "test.message_id")

	if err := p.Publish(context.Background(), "test:42", "test.message_id", []byte(`{}`)); err != nil {
		t.Fatal(err)
	}

	select {
	case delivery := <-deliveries:
		if delivery.MessageId != "test:42" {
			t.Fatalf("message id %q, want test:42", delivery.MessageId)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("message was never delivered")
	}
}

func TestRabbitMQPublisher_Ping_ReconnectsAfterConnectionLoss(t *testing.T) {
	p := newTestPublisher(t)

	if err := p.conn.Close(); err != nil {
		t.Fatal(err)
	}

	if err := p.Ping(context.Background()); err != nil {
		t.Fatalf("ping after connection loss: %v, want a reconnect", err)
	}
	if p.conn.IsClosed() {
		t.Fatal("ping should leave an open connection")
	}
	if err := p.Publish(context.Background(), "test:1", "test.event", []byte(`{}`)); err != nil {
		t.Fatalf("publish after reconnect: %v", err)
	}
}

func TestRabbitMQPublisher_ChannelFailure_ClosesOldConnection(t *testing.T) {
	p := newTestPublisher(t)
	old := p.conn

	if err := p.channel.Close(); err != nil {
		t.Fatal(err)
	}

	if err := p.Publish(context.Background(), "test:1", "test.event", []byte(`{}`)); err != nil {
		t.Fatalf("publish after channel failure: %v", err)
	}
	if p.conn == old {
		t.Fatal("publish should have reconnected")
	}
	if !old.IsClosed() {
		t.Fatal("the old connection should be closed, not leaked")
	}
}
