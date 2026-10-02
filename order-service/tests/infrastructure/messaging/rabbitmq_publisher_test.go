package messaging_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"order-service/internal/infrastructure/messaging"
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
