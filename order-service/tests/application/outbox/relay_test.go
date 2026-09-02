package outbox_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	outboxapp "order-service/internal/application/outbox"
	"order-service/internal/domain/outbox"
)

type PublishedMessage struct {
	RoutingKey string
	Payload    []byte
}

type MockEventPublisher struct {
	Published  []PublishedMessage
	ShouldFail bool
}

func (m *MockEventPublisher) Publish(
	ctx context.Context,
	routingKey string,
	payload []byte,
) error {
	m.Published = append(m.Published, PublishedMessage{
		RoutingKey: routingKey,
		Payload:    payload,
	})

	if m.ShouldFail {
		return errors.New("mock publish failure")
	}

	return nil
}

func TestRelay_Process_Success(t *testing.T) {
	mockPublisher := &MockEventPublisher{}

	relay := outboxapp.NewRelay(mockPublisher)

	e := outbox.OutboxEvent{
		EventName: "order.confirmed",
		Payload:   []byte(`{"order_id":"c9b1f7e0-8f7e-4b7c-9b3e-1a2b3c4d5e6f"}`),
	}

	err := relay.Process(context.Background(), e)

	require.NoError(t, err)

	require.Len(t, mockPublisher.Published, 1)

	msg := mockPublisher.Published[0]

	assert.Equal(t, "order.confirmed", msg.RoutingKey)

	assert.JSONEq(
		t,
		`{"order_id":"c9b1f7e0-8f7e-4b7c-9b3e-1a2b3c4d5e6f"}`,
		string(msg.Payload),
	)
}

func TestRelay_Process_PublishError(t *testing.T) {
	mockPublisher := &MockEventPublisher{
		ShouldFail: true,
	}

	relay := outboxapp.NewRelay(mockPublisher)

	e := outbox.OutboxEvent{
		EventName: "order.confirmed",
		Payload:   []byte(`{"order_id":"c9b1f7e0-8f7e-4b7c-9b3e-1a2b3c4d5e6f"}`),
	}

	err := relay.Process(context.Background(), e)

	require.Error(t, err)

	assert.Equal(t, "mock publish failure", err.Error())

	// publish still attempted
	require.Len(t, mockPublisher.Published, 1)
}
