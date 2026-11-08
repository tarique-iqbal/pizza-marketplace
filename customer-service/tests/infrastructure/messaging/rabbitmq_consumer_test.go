package messaging_test

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"

	"customer-service/internal/domain/customer"
	"customer-service/internal/infrastructure/messaging"

	"pizza-marketplace/shared/telemetry/otelamqp"
)

type mockDispatcher struct {
	mu             sync.Mutex
	fail           bool
	dispatchCalled bool
	spanContext    trace.SpanContext
	eventReceived  customer.EventPayload
}

func (m *mockDispatcher) Register(eventName string, handler customer.EventHandler) {}

func (m *mockDispatcher) Dispatch(ctx context.Context, event customer.EventPayload) error {
	m.mu.Lock()
	m.dispatchCalled = true
	m.eventReceived = event
	m.spanContext = trace.SpanContextFromContext(ctx)
	m.mu.Unlock()

	if m.fail {
		return errors.New("fail dispatch")
	}
	return nil
}

type fakeAcknowledger struct {
	mu      sync.Mutex
	acked   bool
	nacked  bool
	requeue bool
}

func (f *fakeAcknowledger) Ack(tag uint64, multiple bool) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.acked = true
	return nil
}

func (f *fakeAcknowledger) Nack(tag uint64, multiple, requeue bool) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.nacked = true
	f.requeue = requeue
	return nil
}

func (f *fakeAcknowledger) Reject(tag uint64, requeue bool) error {
	return nil
}

func (f *fakeAcknowledger) isAcked() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.acked
}

type republishCall struct {
	retryCount int
}

type fakeSource struct {
	messages chan amqp091.Delivery

	mu          sync.Mutex
	republished []republishCall
}

func (f *fakeSource) GetMessages(ctx context.Context) (<-chan amqp091.Delivery, error) {
	return f.messages, nil
}

func (f *fakeSource) Republish(ctx context.Context, msg amqp091.Delivery, retryCount int) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.republished = append(f.republished, republishCall{retryCount: retryCount})
	return nil
}

func waitFor(t *testing.T, timeout time.Duration, cond func() bool) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}

	t.Fatal("condition not met within timeout")
}

func purgeQueue(t *testing.T, url, queue string) {
	t.Helper()

	conn, err := amqp091.Dial(url)
	require.NoError(t, err)
	defer conn.Close()

	ch, err := conn.Channel()
	require.NoError(t, err)

	_, err = ch.QueuePurge(queue, false)
	require.NoError(t, err)
}

func TestRabbitMQConsumer_Republish_RetryKeepsEventName(t *testing.T) {
	url := os.Getenv("RABBITMQ_URL")
	require.NotEmpty(t, url, "RABBITMQ_URL must be set (see .env.test)")

	consumer, err := messaging.NewRabbitMQConsumer(url)
	require.NoError(t, err)
	t.Cleanup(consumer.Close)

	purgeQueue(t, url, messaging.QueueName)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	original := amqp091.Delivery{
		Exchange:   "identity.events",
		RoutingKey: "user.registered",
		Body:       []byte(`{}`),
		Headers:    amqp091.Table{},
		MessageId:  "retry-keeps-event-name",
	}
	require.NoError(t, consumer.Republish(ctx, original, 1))

	msgs, err := consumer.GetMessages(ctx)
	require.NoError(t, err)

	var retried amqp091.Delivery
	select {
	case retried = <-msgs:
	case <-ctx.Done():
		t.Fatal("republished message was never delivered")
	}
	require.NoError(t, retried.Ack(false))

	ack := &fakeAcknowledger{}
	retried.Acknowledger = ack

	dispatcher := &mockDispatcher{}
	source := &fakeSource{messages: make(chan amqp091.Delivery, 1)}

	go func() { _ = messaging.Run(ctx, source, dispatcher) }()

	source.messages <- retried

	waitFor(t, time.Second, ack.isAcked)

	assert.Equal(t, "user.registered", dispatcher.eventReceived.Name)
}

func (m *mockDispatcher) receivedSpanContext() trace.SpanContext {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.spanContext
}

func TestRabbitMQConsumer_Run_DispatchesUnderConsumerSpan(t *testing.T) {
	previousProvider := otel.GetTracerProvider()
	previousPropagator := otel.GetTextMapPropagator()
	t.Cleanup(func() {
		otel.SetTracerProvider(previousProvider)
		otel.SetTextMapPropagator(previousPropagator)
	})

	recorder := tracetest.NewSpanRecorder()
	otel.SetTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder)))
	otel.SetTextMapPropagator(propagation.TraceContext{})

	producer := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    trace.TraceID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		SpanID:     trace.SpanID{1, 2, 3, 4, 5, 6, 7, 8},
		TraceFlags: trace.FlagsSampled,
	})
	headers := amqp091.Table{}
	propagation.TraceContext{}.Inject(
		trace.ContextWithSpanContext(context.Background(), producer),
		otelamqp.HeaderCarrier(headers),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dispatcher := &mockDispatcher{}
	source := &fakeSource{messages: make(chan amqp091.Delivery, 1)}

	go func() { _ = messaging.Run(ctx, source, dispatcher) }()

	ack := &fakeAcknowledger{}
	source.messages <- amqp091.Delivery{
		Acknowledger: ack,
		Exchange:     "identity.events",
		RoutingKey:   "user.registered",
		Headers:      headers,
		Body:         []byte(`{}`),
	}

	waitFor(t, time.Second, ack.isAcked)

	var consumer sdktrace.ReadOnlySpan
	for _, s := range recorder.Ended() {
		if s.Name() == "process identity.events:user.registered:"+messaging.QueueName {
			consumer = s
		}
	}
	require.NotNil(t, consumer, "consumer span should be ended")
	require.Equal(t, trace.SpanKindConsumer, consumer.SpanKind())
	require.Equal(t, producer.TraceID(), consumer.SpanContext().TraceID())
	require.Equal(t, producer.SpanID(), consumer.Parent().SpanID())

	got := dispatcher.receivedSpanContext()
	require.Equal(t, consumer.SpanContext().SpanID(), got.SpanID(), "handler runs under the consumer span")
}
