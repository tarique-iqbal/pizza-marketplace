package messaging_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"identity-service/internal/infrastructure/messaging"
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
		_ = p.Publish(shortCtx, "test.event", []byte(`{}`))
		cancel()
	}

	// Let any in-flight publishes from the loop above actually land their real,
	// now-unawaited confirmation from the broker.
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
