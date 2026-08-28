package outbox_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	outboxapp "restaurant-service/internal/application/outbox"
	"restaurant-service/internal/domain/outbox"
	"restaurant-service/internal/shared/event"
)

type PublishedRawMessage struct {
	Topic   string
	Payload []byte
}

type MockEventPublisher struct {
	PublishedEvent []event.Event
	PublishedRaw   []PublishedRawMessage
	ShouldFail     bool
}

func (m *MockEventPublisher) PublishEvent(
	ctx context.Context,
	e event.Event,
) error {
	m.PublishedEvent = append(m.PublishedEvent, e)

	if m.ShouldFail {
		return errors.New("mock event publish failure")
	}

	return nil
}

func (m *MockEventPublisher) PublishRaw(
	ctx context.Context,
	topic string,
	jsonData []byte,
) error {
	m.PublishedRaw = append(m.PublishedRaw, PublishedRawMessage{
		Topic:   topic,
		Payload: jsonData,
	})

	if m.ShouldFail {
		return errors.New("mock raw publish failure")
	}

	return nil
}

func TestRelay_Process_Success(t *testing.T) {
	mockPublisher := &MockEventPublisher{}

	relay := outboxapp.NewRelay(mockPublisher)

	e := outbox.OutboxEvent{
		EventName: "restaurant.updated",
		Payload:   []byte(`{"restaurant_name":"L'Osteria"}`),
	}

	err := relay.Process(context.Background(), e)

	require.NoError(t, err)

	require.Len(t, mockPublisher.PublishedRaw, 1)

	msg := mockPublisher.PublishedRaw[0]

	assert.Equal(t, "restaurant.updated", msg.Topic)

	assert.JSONEq(
		t,
		`{"restaurant_name":"L'Osteria"}`,
		string(msg.Payload),
	)
}

func TestRelay_Process_PublishError(t *testing.T) {
	mockPublisher := &MockEventPublisher{
		ShouldFail: true,
	}

	relay := outboxapp.NewRelay(mockPublisher)

	e := outbox.OutboxEvent{
		EventName: "restaurant.updated",
		Payload:   []byte(`{"restaurant_name":"L'Osteria"}`),
	}

	err := relay.Process(context.Background(), e)

	require.Error(t, err)

	assert.Equal(t, "mock raw publish failure", err.Error())

	// publish still attempted
	require.Len(t, mockPublisher.PublishedRaw, 1)
}
