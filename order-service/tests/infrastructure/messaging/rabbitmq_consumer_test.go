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

	"order-service/internal/domain/readmodel"
	"order-service/internal/infrastructure/messaging"
)

type mockDispatcher struct {
	mu             sync.Mutex
	fail           bool
	dispatchCalled bool
	eventReceived  readmodel.EventPayload
}

func (m *mockDispatcher) Register(_ string, _ readmodel.EventHandler) {}

func (m *mockDispatcher) Dispatch(event readmodel.EventPayload) error {
	m.mu.Lock()
	m.dispatchCalled = true
	m.eventReceived = event
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

func (f *fakeAcknowledger) Ack(_ uint64, _ bool) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.acked = true

	return nil
}

func (f *fakeAcknowledger) Nack(_ uint64, _, requeue bool) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.nacked = true
	f.requeue = requeue

	return nil
}

func (f *fakeAcknowledger) Reject(_ uint64, _ bool) error {
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

func (f *fakeSource) GetMessages(_ context.Context) (<-chan amqp091.Delivery, error) {
	return f.messages, nil
}

func (f *fakeSource) Republish(_ context.Context, _ amqp091.Delivery, retryCount int) error {
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

func makeDelivery(body []byte, routingKey string, headers amqp091.Table, ack *fakeAcknowledger) amqp091.Delivery {
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
	body := []byte(`{"restaurant_id":"01a06d6e-0000-7000-8000-000000000000"}`)
	source.messages <- makeDelivery(body, "restaurant.launched", nil, ack)

	waitFor(t, time.Second, ack.isAcked)

	assert.True(t, dispatcher.wasDispatched())
	assert.Equal(t, "restaurant.launched", dispatcher.eventReceived.Name)
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
	headers := amqp091.Table{"x-retry-count": int32(1)}
	source.messages <- makeDelivery([]byte(`{}`), "restaurant.launched", headers, ack)

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
	source.messages <- makeDelivery([]byte(`{}`), "restaurant.launched", headers, ack)

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
