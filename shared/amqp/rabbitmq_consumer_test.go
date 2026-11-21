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
	testExchange   = "shared_amqp_test.events"
	testEvent      = "test.happened"
)

type recordingHandler struct {
	mu          sync.Mutex
	failTimes   int
	calls       []string
	spanContext trace.SpanContext
}

func alwaysFailing() *recordingHandler {
	return &recordingHandler{failTimes: -1}
}

func (h *recordingHandler) handle(ctx context.Context, eventName string, body []byte) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.calls = append(h.calls, eventName)
	h.spanContext = trace.SpanContextFromContext(ctx)

	if h.failTimes < 0 || len(h.calls) <= h.failTimes {
		return errors.New("fail dispatch")
	}
	return nil
}

func (h *recordingHandler) received() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]string(nil), h.calls...)
}

func (h *recordingHandler) lastSpanContext() trace.SpanContext {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.spanContext
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

func (f *fakeAcknowledger) settled() bool {
	nacked, _ := f.isNacked()
	return nacked || f.isAcked()
}

type fakeSource struct {
	messages      chan amqp091.Delivery
	deadLetterErr error

	mu           sync.Mutex
	deadLettered int
}

func newFakeSource() *fakeSource {
	return &fakeSource{messages: make(chan amqp091.Delivery, 2)}
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

func (f *fakeSource) DeadLetter(ctx context.Context, msg amqp091.Delivery) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.deadLettered++
	return f.deadLetterErr
}

func (f *fakeSource) deadLetterCalls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.deadLettered
}

func rejectedHeaders(count int64) amqp091.Table {
	return amqp091.Table{
		"x-death": []interface{}{
			amqp091.Table{
				"queue": testQueue + ".wait", "reason": "expired", "count": count,
				"exchange": "test_dlx", "routing-keys": []interface{}{testQueue + ".retry"},
			},
			amqp091.Table{
				"queue": testQueue, "reason": "rejected", "count": count,
				"exchange": testExchange, "routing-keys": []interface{}{testEvent},
			},
		},
	}
}

func redelivery(ack *fakeAcknowledger, count int64) amqp091.Delivery {
	return amqp091.Delivery{
		Acknowledger: ack,
		RoutingKey:   testQueue,
		Headers:      rejectedHeaders(count),
		Body:         []byte(`{}`),
	}
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
	source.messages <- amqp091.Delivery{Acknowledger: ack, RoutingKey: testEvent, Body: []byte(`{}`)}

	waitFor(t, time.Second, ack.isAcked)

	if calls := handler.received(); len(calls) != 1 || calls[0] != testEvent {
		t.Fatalf("handler calls %v, want [%s]", calls, testEvent)
	}
	if nacked, _ := ack.isNacked(); nacked {
		t.Fatal("message should not be nacked")
	}
}

func TestRun_HandlerFailsWithRetriesLeft_NacksToWaitQueue(t *testing.T) {
	source := newFakeSource()
	startRun(t, source, alwaysFailing().handle)

	ack := &fakeAcknowledger{}
	source.messages <- redelivery(ack, testMaxRetries-1)

	waitFor(t, time.Second, ack.settled)

	if nacked, requeue := ack.isNacked(); !nacked || requeue {
		t.Fatalf("nacked=%v requeue=%v, want a nack without requeue", nacked, requeue)
	}
	if ack.isAcked() {
		t.Fatal("message should not be acked")
	}
	if n := source.deadLetterCalls(); n != 0 {
		t.Fatalf("dead-lettered %d times, want 0", n)
	}
}

func TestRun_RetryLimitReached_DeadLettersThenAcks(t *testing.T) {
	source := newFakeSource()
	startRun(t, source, alwaysFailing().handle)

	ack := &fakeAcknowledger{}
	source.messages <- redelivery(ack, testMaxRetries)

	waitFor(t, time.Second, ack.isAcked)

	if n := source.deadLetterCalls(); n != 1 {
		t.Fatalf("dead-lettered %d times, want 1", n)
	}
	if nacked, _ := ack.isNacked(); nacked {
		t.Fatal("message should not be nacked")
	}
}

func TestRun_DeadLetterFails_NacksWithoutAck(t *testing.T) {
	source := newFakeSource()
	source.deadLetterErr = errors.New("broker unavailable")
	startRun(t, source, alwaysFailing().handle)

	ack := &fakeAcknowledger{}
	source.messages <- redelivery(ack, testMaxRetries)

	waitFor(t, time.Second, ack.settled)

	if nacked, requeue := ack.isNacked(); !nacked || requeue {
		t.Fatalf("nacked=%v requeue=%v, want a nack without requeue", nacked, requeue)
	}
	if ack.isAcked() {
		t.Fatal("message must not be acked when the DLQ copy wasn't confirmed")
	}
}

func TestRun_FailingMessage_DoesNotBlockNextMessage(t *testing.T) {
	source := newFakeSource()
	handler := func(ctx context.Context, eventName string, body []byte) error {
		if eventName == "bad.event" {
			return errors.New("fail dispatch")
		}
		return nil
	}
	startRun(t, source, handler)

	bad, good := &fakeAcknowledger{}, &fakeAcknowledger{}
	source.messages <- amqp091.Delivery{Acknowledger: bad, RoutingKey: "bad.event", Body: []byte(`{}`)}
	source.messages <- amqp091.Delivery{Acknowledger: good, RoutingKey: testEvent, Body: []byte(`{}`)}

	waitFor(t, 200*time.Millisecond, func() bool { return bad.settled() && good.isAcked() })
}

func TestRun_RetriedMessage_KeepsOriginalEventName(t *testing.T) {
	handler := &recordingHandler{}
	source := newFakeSource()
	startRun(t, source, handler.handle)

	ack := &fakeAcknowledger{}
	source.messages <- redelivery(ack, 1)

	waitFor(t, time.Second, ack.isAcked)

	if calls := handler.received(); len(calls) != 1 || calls[0] != testEvent {
		t.Fatalf("handler calls %v, want [%s]", calls, testEvent)
	}
}

func TestFirstRejection(t *testing.T) {
	if got := firstRejection(amqp091.Table{}, testQueue); got != nil {
		t.Fatalf("no x-death: got %+v, want nil", got)
	}

	got := firstRejection(rejectedHeaders(2), testQueue)
	if got == nil || got.count != 2 || got.exchange != testExchange || got.routingKey != testEvent {
		t.Fatalf("got %+v, want count 2 from %s with key %s", got, testExchange, testEvent)
	}

	if got := firstRejection(rejectedHeaders(2), "other_queue"); got != nil {
		t.Fatalf("other queue: got %+v, want nil", got)
	}

	int32Count := amqp091.Table{"x-death": []interface{}{
		amqp091.Table{"queue": testQueue, "reason": "rejected", "count": int32(3)},
	}}
	if got := firstRejection(int32Count, testQueue); got == nil || got.count != 3 {
		t.Fatalf("int32 count: got %+v, want count 3", got)
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
		Exchange:     testExchange,
		RoutingKey:   testEvent,
		Headers:      headers,
		Body:         []byte(`{}`),
	}

	waitFor(t, time.Second, ack.isAcked)

	var consumer sdktrace.ReadOnlySpan
	for _, s := range recorder.Ended() {
		if s.Name() == "process "+testExchange+":"+testEvent+":"+testQueue {
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
	if handler.lastSpanContext().SpanID() != consumer.SpanContext().SpanID() {
		t.Fatal("handler should run under the consumer span")
	}
}

type broker struct {
	t  *testing.T
	ch *amqp091.Channel
}

func newBroker(t *testing.T) *broker {
	t.Helper()

	url := os.Getenv("RABBITMQ_URL")
	if url == "" {
		t.Skip("RABBITMQ_URL not set; skipping real-broker test")
	}

	conn, err := amqp091.Dial(url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	ch, err := conn.Channel()
	if err != nil {
		t.Fatal(err)
	}
	return &broker{t: t, ch: ch}
}

func (b *broker) deleteQueues(queue string) {
	for _, name := range []string{queue, queue + ".wait", queue + ".dlq"} {
		if _, err := b.ch.QueueDelete(name, false, false, false); err != nil {
			b.t.Fatal(err)
		}
	}
}

func (b *broker) consumer(queue string, maxRetries int) *RabbitMQConsumer {
	b.t.Helper()

	consumer, err := NewRabbitMQConsumer(os.Getenv("RABBITMQ_URL"), Config{
		Queue:            queue,
		DLX:              queue + "_dlx",
		Bindings:         map[string][]string{testExchange: {testEvent}},
		MaxRetryAttempts: maxRetries,
		RetryDelay:       200 * time.Millisecond,
	})
	if err != nil {
		b.t.Fatal(err)
	}
	b.t.Cleanup(consumer.Close)
	return consumer
}

func (b *broker) publish(headers amqp091.Table) {
	b.t.Helper()

	msg := amqp091.Publishing{ContentType: "application/json", Body: []byte(`{"id":1}`), Headers: headers}
	err := b.ch.PublishWithContext(context.Background(), testExchange, testEvent, false, false, msg)
	if err != nil {
		b.t.Fatal(err)
	}
}

func (b *broker) messages(queue string) int {
	b.t.Helper()

	q, err := b.ch.QueueDeclarePassive(queue, true, false, false, false, nil)
	if err != nil {
		b.t.Fatal(err)
	}
	return q.Messages
}

func TestRabbitMQConsumer_AlwaysFailing_EndsInDLQAfterRetries(t *testing.T) {
	b := newBroker(t)
	const queue = "shared_amqp_always_failing"
	b.deleteQueues(queue)
	t.Cleanup(func() { b.deleteQueues(queue) })

	consumer := b.consumer(queue, testMaxRetries)
	handler := alwaysFailing()
	startRun(t, consumer, handler.handle)

	b.publish(amqp091.Table{"traceparent": "00-0102030405060708090a0b0c0d0e0f10-0102030405060708-01"})

	waitFor(t, 5*time.Second, func() bool { return b.messages(queue+".dlq") == 1 })

	calls := handler.received()
	if len(calls) != testMaxRetries+1 {
		t.Fatalf("handler ran %d times, want %d", len(calls), testMaxRetries+1)
	}
	for i, name := range calls {
		if name != testEvent {
			t.Fatalf("attempt %d saw event %q, want %q", i+1, name, testEvent)
		}
	}

	dead, ok, err := b.ch.Get(queue+".dlq", true)
	if err != nil || !ok {
		t.Fatalf("DLQ get: ok=%v err=%v", ok, err)
	}
	if string(dead.Body) != `{"id":1}` {
		t.Fatalf("DLQ body %q, want the original", dead.Body)
	}
	if dead.Headers["traceparent"] != "00-0102030405060708090a0b0c0d0e0f10-0102030405060708-01" {
		t.Fatalf("DLQ traceparent %v, want the original", dead.Headers["traceparent"])
	}
	if n := b.messages(queue) + b.messages(queue+".wait"); n != 0 {
		t.Fatalf("%d messages left in the main and wait queues, want 0", n)
	}
}

func TestRabbitMQConsumer_FailsOnce_DeliveredOnSecondAttempt(t *testing.T) {
	b := newBroker(t)
	const queue = "shared_amqp_fails_once"
	b.deleteQueues(queue)
	t.Cleanup(func() { b.deleteQueues(queue) })

	consumer := b.consumer(queue, testMaxRetries)
	handler := &recordingHandler{failTimes: 1}
	startRun(t, consumer, handler.handle)

	b.publish(nil)

	waitFor(t, 5*time.Second, func() bool { return len(handler.received()) == 2 })
	time.Sleep(500 * time.Millisecond)

	if calls := handler.received(); len(calls) != 2 || calls[1] != testEvent {
		t.Fatalf("handler calls %v, want 2 attempts of %s", calls, testEvent)
	}
	if n := b.messages(queue + ".dlq"); n != 0 {
		t.Fatalf("%d messages in the DLQ, want 0", n)
	}
}

func TestRabbitMQConsumer_ExistingBroker_RetriesDoNotReachDLQ(t *testing.T) {
	b := newBroker(t)
	const queue = "shared_amqp_upgrade"
	b.deleteQueues(queue)
	t.Cleanup(func() { b.deleteQueues(queue) })

	dlx := queue + "_dlx"
	if err := b.ch.ExchangeDeclare(dlx, "topic", true, false, false, false, nil); err != nil {
		t.Fatal(err)
	}
	args := amqp091.Table{"x-dead-letter-exchange": dlx, "x-dead-letter-routing-key": queue + ".retry"}
	if _, err := b.ch.QueueDeclare(queue, true, false, false, false, args); err != nil {
		t.Fatal(err)
	}
	if _, err := b.ch.QueueDeclare(queue+".dlq", true, false, false, false, nil); err != nil {
		t.Fatal(err)
	}
	if err := b.ch.QueueBind(queue+".dlq", queue+".retry", dlx, false, nil); err != nil {
		t.Fatal(err)
	}

	consumer := b.consumer(queue, 1)
	handler := alwaysFailing()
	startRun(t, consumer, handler.handle)

	b.publish(nil)

	waitFor(t, 5*time.Second, func() bool { return len(handler.received()) == 2 })
	waitFor(t, 2*time.Second, func() bool { return b.messages(queue+".dlq") == 1 })
	time.Sleep(500 * time.Millisecond)

	if n := b.messages(queue + ".dlq"); n != 1 {
		t.Fatalf("%d messages in the DLQ, want 1: the old retry binding must be gone", n)
	}
}

func TestRabbitMQConsumer_Ping_AfterClose_Fails(t *testing.T) {
	b := newBroker(t)
	const queue = "shared_amqp_ping"
	b.deleteQueues(queue)
	t.Cleanup(func() { b.deleteQueues(queue) })

	consumer := b.consumer(queue, testMaxRetries)

	if err := consumer.Ping(context.Background()); err != nil {
		t.Fatalf("ping: %v", err)
	}

	consumer.Close()

	if err := consumer.Ping(context.Background()); err == nil {
		t.Fatal("ping after close should fail")
	}
}
