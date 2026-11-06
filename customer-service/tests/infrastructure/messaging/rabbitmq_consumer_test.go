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

	"customer-service/internal/domain/customer"
	"customer-service/internal/infrastructure/messaging"
)

type mockDispatcher struct {
	mu             sync.Mutex
	fail           bool
	dispatchCalled bool
	eventReceived  customer.EventPayload
}

func (m *mockDispatcher) Register(eventName string, handler customer.EventHandler) {}

func (m *mockDispatcher) Dispatch(ctx context.Context, event customer.EventPayload) error {
	m.mu.Lock()
	m.dispatchCalled = true
	m.eventReceived = event
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
