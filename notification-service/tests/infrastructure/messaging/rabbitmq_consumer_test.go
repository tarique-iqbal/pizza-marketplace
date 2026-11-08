package messaging_test

import (
	"context"
	"errors"
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

	"notification-service/internal/domain/notification"
	"notification-service/internal/infrastructure/messaging"

	"pizza-marketplace/shared/telemetry/otelamqp"
)

type mockDispatcher struct {
	mu             sync.Mutex
	fail           bool
	dispatchCalled bool
	spanContext    trace.SpanContext
	eventReceived  notification.EventPayload
}

func (m *mockDispatcher) Register(eventName string, handler notification.EventHandler) {}

func (m *mockDispatcher) Dispatch(ctx context.Context, event notification.EventPayload) error {
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

func (m *mockDispatcher) wasDispatched() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.dispatchCalled
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

func (f *fakeAcknowledger) isNacked() (nacked, requeue bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.nacked, f.requeue
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

func (f *fakeSource) republishCalls() []republishCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]republishCall(nil), f.republished...)
}

func makeDelivery(
	body []byte,
	routingKey string,
	headers amqp091.Table,
	ack *fakeAcknowledger,
) amqp091.Delivery {
	return amqp091.Delivery{
		Acknowledger: ack,
		Body:         body,
		RoutingKey:   routingKey,
		Headers:      headers,
	}
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

func TestRabbitMQConsumer_Run_DispatchSuccess_AcksMessage(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dispatcher := &mockDispatcher{}
	source := &fakeSource{messages: make(chan amqp091.Delivery, 1)}

	go func() { _ = messaging.Run(ctx, source, dispatcher) }()

	ack := &fakeAcknowledger{}
	source.messages <- makeDelivery([]byte(`{"email":"test@example.com"}`), "user.registered", nil, ack)

	waitFor(t, time.Second, ack.isAcked)

	assert.True(t, dispatcher.wasDispatched())
	assert.Equal(t, "user.registered", dispatcher.eventReceived.Name)
	nacked, _ := ack.isNacked()
	assert.False(t, nacked)
	assert.Empty(t, source.republishCalls())
}

func TestRabbitMQConsumer_Run_DispatchFails_RepublishesWithIncrementedRetryCount(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dispatcher := &mockDispatcher{fail: true}
	source := &fakeSource{messages: make(chan amqp091.Delivery, 1)}

	go func() { _ = messaging.Run(ctx, source, dispatcher) }()

	ack := &fakeAcknowledger{}
	source.messages <- makeDelivery([]byte(`{}`), "user.registered", amqp091.Table{"x-retry-count": int32(1)}, ack)

	waitFor(t, 3*time.Second, func() bool { return len(source.republishCalls()) == 1 })

	assert.True(t, dispatcher.wasDispatched())
	require.Len(t, source.republishCalls(), 1)
	assert.Equal(t, 2, source.republishCalls()[0].retryCount)
	assert.True(t, ack.isAcked(), "original delivery is acked once superseded by the republished copy")
}

func TestRabbitMQConsumer_Run_ExceedsRetryLimit_NacksWithoutRequeue(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dispatcher := &mockDispatcher{fail: true}
	source := &fakeSource{messages: make(chan amqp091.Delivery, 1)}

	go func() { _ = messaging.Run(ctx, source, dispatcher) }()

	ack := &fakeAcknowledger{}
	headers := amqp091.Table{"x-retry-count": int32(messaging.MaxRetryAttempts)}
	source.messages <- makeDelivery([]byte(`{}`), "user.registered", headers, ack)

	waitFor(t, time.Second, func() bool {
		nacked, _ := ack.isNacked()
		return nacked
	})

	assert.True(t, dispatcher.wasDispatched())
	nacked, requeue := ack.isNacked()
	assert.True(t, nacked)
	assert.False(t, requeue, "exhausted retries route to the DLX-bound DLQ, not a plain requeue")
	assert.Empty(t, source.republishCalls())
}

func TestRabbitMQConsumer_Run_RetriedMessage_UsesOriginalRoutingKey(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dispatcher := &mockDispatcher{}
	source := &fakeSource{messages: make(chan amqp091.Delivery, 1)}

	go func() { _ = messaging.Run(ctx, source, dispatcher) }()

	ack := &fakeAcknowledger{}
	headers := amqp091.Table{
		"x-retry-count":          int32(1),
		"x-original-routing-key": "user.registered",
	}
	source.messages <- makeDelivery([]byte(`{}`), messaging.QueueName, headers, ack)

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
