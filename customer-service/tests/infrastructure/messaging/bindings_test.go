package messaging_test

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"customer-service/internal/domain/customer"
	"customer-service/internal/infrastructure/messaging"

	"pizza-marketplace/shared/amqp"
)

type recordingDispatcher struct {
	mu     sync.Mutex
	events []customer.EventPayload
}

func (d *recordingDispatcher) Register(eventName string, handler customer.EventHandler) {}

func (d *recordingDispatcher) Dispatch(ctx context.Context, event customer.EventPayload) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.events = append(d.events, event)
	return nil
}

func (d *recordingDispatcher) received() []customer.EventPayload {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]customer.EventPayload(nil), d.events...)
}

func (d *recordingDispatcher) receivedNames() []string {
	var names []string
	for _, e := range d.received() {
		names = append(names, e.Name)
	}
	return names
}

func TestHandler_DispatchesEventPayload(t *testing.T) {
	dispatcher := &recordingDispatcher{}

	err := messaging.Handler(dispatcher)(context.Background(), "order.address_saved", []byte(`{"id":1}`))

	require.NoError(t, err)
	require.Len(t, dispatcher.received(), 1)
	assert.Equal(t, "order.address_saved", dispatcher.received()[0].Name)
	assert.Equal(t, []byte(`{"id":1}`), dispatcher.received()[0].Data)
}

func TestConsumerConfig_RoutesBoundEventsToDispatcher(t *testing.T) {
	url := os.Getenv("RABBITMQ_URL")
	require.NotEmpty(t, url, "RABBITMQ_URL must be set (see .env.test)")

	consumer, err := amqp.NewRabbitMQConsumer(url, messaging.ConsumerConfig())
	require.NoError(t, err)
	t.Cleanup(consumer.Close)

	conn, err := amqp091.Dial(url)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	ch, err := conn.Channel()
	require.NoError(t, err)

	_, err = ch.QueuePurge(messaging.QueueName, false)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dispatcher := &recordingDispatcher{}
	go func() { _ = amqp.Run(ctx, consumer, messaging.Handler(dispatcher)) }()

	var published []string
	for exchange, routingKeys := range messaging.Exchanges {
		for _, key := range routingKeys {
			err := ch.PublishWithContext(ctx, exchange, key, false, false, amqp091.Publishing{
				ContentType: "application/json",
				Body:        []byte(`{}`),
			})
			require.NoError(t, err)
			published = append(published, key)
		}
	}

	require.Eventually(t, func() bool {
		return len(dispatcher.receivedNames()) == len(published)
	}, 5*time.Second, 10*time.Millisecond)

	assert.ElementsMatch(t, published, dispatcher.receivedNames())
}
