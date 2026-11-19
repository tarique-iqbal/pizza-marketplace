package amqp

import (
	"context"
	"errors"
	"os"
	"sync"
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

const (
	testQueue      = "shared_amqp_test_queue"
	testMaxRetries = 3
)

type recordingHandler struct {
	mu          sync.Mutex
	fail        bool
	called      bool
	eventName   string
	body        []byte
	spanContext trace.SpanContext
}

func (h *recordingHandler) handle(ctx context.Context, eventName string, body []byte) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.called = true
	h.eventName = eventName
	h.body = body
	h.spanContext = trace.SpanContextFromContext(ctx)

	if h.fail {
		return errors.New("fail dispatch")
	}
	return nil
}

func (h *recordingHandler) received() (called bool, eventName string, spanContext trace.SpanContext) {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.called, h.eventName, h.spanContext
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

type fakeSource struct {
	messages chan amqp091.Delivery

	mu          sync.Mutex
	republished []int
}

func newFakeSource() *fakeSource {
	return &fakeSource{messages: make(chan amqp091.Delivery, 1)}
}

func (f *fakeSource) Queue() string {
	return testQueue
}

func (f *fakeSource) MaxRetryAttempts() int {
	return testMaxRetries
}

func (f *fakeSource) GetMessages(ctx context.Context) (<-chan amqp091.Delivery, error) {
	return f.messages, nil
}

func (f *fakeSource) Republish(ctx context.Context, msg amqp091.Delivery, retryCount int) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.republished = append(f.republished, retryCount)
	return nil
}

func (f *fakeSource) republishCalls() []int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]int(nil), f.republished...)
}

func startRun(t *testing.T, source MessageSource, handler Handler) {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	go func() { _ = Run(ctx, source, handler) }()
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

func TestRun_HandlerSucceeds_AcksMessage(t *testing.T) {
	handler := &recordingHandler{}
	source := newFakeSource()
	startRun(t, source, handler.handle)

	ack := &fakeAcknowledger{}
	source.messages <- amqp091.Delivery{
		Acknowledger: ack,
		RoutingKey:   "restaurant.initiated",
		Body:         []byte(`{"business_name":"Pizza Palace"}`),
	}

	waitFor(t, time.Second, ack.isAcked)

	called, eventName, _ := handler.received()
	if !called || eventName != "restaurant.initiated" {
		t.Fatalf("handler called=%v with %q, want restaurant.initiated", called, eventName)
	}
	if nacked, _ := ack.isNacked(); nacked {
		t.Fatal("message should not be nacked")
	}
	if calls := source.republishCalls(); len(calls) != 0 {
		t.Fatalf("republished %v, want none", calls)
	}
}

func TestRun_HandlerFails_RepublishesWithIncrementedRetryCount(t *testing.T) {
	handler := &recordingHandler{fail: true}
	source := newFakeSource()
	startRun(t, source, handler.handle)

	ack := &fakeAcknowledger{}
	source.messages <- amqp091.Delivery{
		Acknowledger: ack,
		RoutingKey:   "restaurant.initiated",
		Headers:      amqp091.Table{"x-retry-count": int32(1)},
		Body:         []byte(`{}`),
	}

	waitFor(t, 3*time.Second, func() bool { return len(source.republishCalls()) == 1 })

	if calls := source.republishCalls(); calls[0] != 2 {
		t.Fatalf("republished with retry count %d, want 2", calls[0])
	}
	waitFor(t, time.Second, ack.isAcked)
}

func TestRun_RetryLimitReached_NacksWithoutRequeue(t *testing.T) {
	handler := &recordingHandler{fail: true}
	source := newFakeSource()
	startRun(t, source, handler.handle)

	ack := &fakeAcknowledger{}
	source.messages <- amqp091.Delivery{
		Acknowledger: ack,
		RoutingKey:   "restaurant.initiated",
		Headers:      amqp091.Table{"x-retry-count": int32(testMaxRetries)},
		Body:         []byte(`{}`),
	}

	waitFor(t, time.Second, func() bool {
		nacked, _ := ack.isNacked()
		return nacked
	})

	if _, requeue := ack.isNacked(); requeue {
		t.Fatal("exhausted retries must go to the DLQ, not back to the queue")
	}
	if calls := source.republishCalls(); len(calls) != 0 {
		t.Fatalf("republished %v, want none", calls)
	}
}

func TestRun_RetriedMessage_KeepsOriginalEventName(t *testing.T) {
	handler := &recordingHandler{}
	source := newFakeSource()
	startRun(t, source, handler.handle)

	ack := &fakeAcknowledger{}
	source.messages <- amqp091.Delivery{
		Acknowledger: ack,
		RoutingKey:   testQueue,
		Headers: amqp091.Table{
			"x-retry-count":          int32(1),
			"x-original-routing-key": "restaurant.initiated",
			"x-original-exchange":    "identity.events",
		},
		Body: []byte(`{}`),
	}

	waitFor(t, time.Second, ack.isAcked)

	if _, eventName, _ := handler.received(); eventName != "restaurant.initiated" {
		t.Fatalf("event name %q, want restaurant.initiated", eventName)
	}
}

func TestRun_HandlerRunsUnderConsumerSpan(t *testing.T) {
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

	handler := &recordingHandler{}
	source := newFakeSource()
	startRun(t, source, handler.handle)

	ack := &fakeAcknowledger{}
	source.messages <- amqp091.Delivery{
		Acknowledger: ack,
		Exchange:     "identity.events",
		RoutingKey:   "restaurant.initiated",
		Headers:      headers,
		Body:         []byte(`{}`),
	}

	waitFor(t, time.Second, ack.isAcked)

	var consumer sdktrace.ReadOnlySpan
	for _, s := range recorder.Ended() {
		if s.Name() == "process identity.events:restaurant.initiated:"+testQueue {
			consumer = s
		}
	}
	if consumer == nil {
		t.Fatal("consumer span should be ended")
	}
	if consumer.SpanKind() != trace.SpanKindConsumer {
		t.Fatalf("span kind %v, want consumer", consumer.SpanKind())
	}
	if consumer.SpanContext().TraceID() != producer.TraceID() {
		t.Fatal("consumer span should be in the producer's trace")
	}
	if consumer.Parent().SpanID() != producer.SpanID() {
		t.Fatal("consumer span should be a child of the producer span")
	}
	if _, _, got := handler.received(); got.SpanID() != consumer.SpanContext().SpanID() {
		t.Fatal("handler should run under the consumer span")
	}
}

func brokerURL(t *testing.T) string {
	t.Helper()

	url := os.Getenv("RABBITMQ_URL")
	if url == "" {
		t.Skip("RABBITMQ_URL not set; skipping real-broker test")
	}
	return url
}

func TestRabbitMQConsumer_Republish_RetryKeepsEventName(t *testing.T) {
	url := brokerURL(t)

	consumer, err := NewRabbitMQConsumer(url, Config{
		Queue:            testQueue,
		DLX:              "shared_amqp_test_dlx",
		Bindings:         map[string][]string{"shared_amqp_test.events": {"test.happened"}},
		MaxRetryAttempts: testMaxRetries,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(consumer.Close)

	if _, err := consumer.channel.QueuePurge(testQueue, false); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := consumer.Ping(ctx); err != nil {
		t.Fatalf("ping: %v", err)
	}

	original := amqp091.Delivery{
		Exchange:   "shared_amqp_test.events",
		RoutingKey: "test.happened",
		Headers:    amqp091.Table{},
		Body:       []byte(`{}`),
		MessageId:  "retry-keeps-event-name",
	}
	if err := consumer.Republish(ctx, original, 1); err != nil {
		t.Fatal(err)
	}

	msgs, err := consumer.GetMessages(ctx)
	if err != nil {
		t.Fatal(err)
	}

	var retried amqp091.Delivery
	select {
	case retried = <-msgs:
	case <-ctx.Done():
		t.Fatal("republished message was never delivered")
	}
	if err := retried.Ack(false); err != nil {
		t.Fatal(err)
	}

	if got := getRetryCount(retried.Headers); got != 1 {
		t.Fatalf("retry count %d, want 1", got)
	}
	if got := getEventName(retried); got != "test.happened" {
		t.Fatalf("event name %q, want test.happened", got)
	}
	if got := getExchange(retried); got != "shared_amqp_test.events" {
		t.Fatalf("exchange %q, want shared_amqp_test.events", got)
	}
}

func TestRabbitMQConsumer_Ping_AfterClose_Fails(t *testing.T) {
	url := brokerURL(t)

	consumer, err := NewRabbitMQConsumer(url, Config{
		Queue:            testQueue,
		DLX:              "shared_amqp_test_dlx",
		Bindings:         map[string][]string{"shared_amqp_test.events": {"test.happened"}},
		MaxRetryAttempts: testMaxRetries,
	})
	if err != nil {
		t.Fatal(err)
	}

	consumer.Close()

	if err := consumer.Ping(context.Background()); err == nil {
		t.Fatal("ping after close should fail")
	}
}
