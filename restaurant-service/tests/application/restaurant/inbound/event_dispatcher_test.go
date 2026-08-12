package inbound_test

import (
	"context"
	"errors"
	"testing"

	"restaurant-service/internal/application/restaurant/inbound"
	"restaurant-service/internal/domain/restaurant"

	"github.com/stretchr/testify/assert"
)

type mockHandler struct {
	Called bool
	Passed restaurant.EventPayload
	Err    error
}

func (m *mockHandler) Handle(ctx context.Context, event restaurant.EventPayload) error {
	m.Called = true
	m.Passed = event
	return m.Err
}

func TestDispatch_CallsRegisteredHandler(t *testing.T) {
	dispatcher := inbound.NewEventDispatcher()
	mock := &mockHandler{}

	dispatcher.Register("restaurant.initiated", mock)

	event := restaurant.EventPayload{
		Name: "restaurant.initiated",
		Data: []byte(`{}`),
	}

	err := dispatcher.Dispatch(context.Background(), event)

	assert.NoError(t, err)
	assert.True(t, mock.Called)
	assert.Equal(t, event, mock.Passed)
}

func TestDispatch_NoHandler(t *testing.T) {
	dispatcher := inbound.NewEventDispatcher()

	event := restaurant.EventPayload{
		Name: "restaurant.unknown",
		Data: []byte(`{}`),
	}

	err := dispatcher.Dispatch(context.Background(), event)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no handler for event: restaurant.unknown")
}

func TestDispatch_HandlerReturnsError(t *testing.T) {
	dispatcher := inbound.NewEventDispatcher()
	mock := &mockHandler{Err: errors.New("handler error")}

	dispatcher.Register("restaurant.initiated", mock)

	event := restaurant.EventPayload{
		Name: "restaurant.initiated",
		Data: []byte(`{}`),
	}

	err := dispatcher.Dispatch(context.Background(), event)

	assert.Error(t, err)
	assert.Equal(t, "handler error", err.Error())
}

func TestRegister_OverridesExistingHandler(t *testing.T) {
	dispatcher := inbound.NewEventDispatcher()

	first := &mockHandler{}
	second := &mockHandler{}

	dispatcher.Register("restaurant.initiated", first)
	dispatcher.Register("restaurant.initiated", second)

	event := restaurant.EventPayload{
		Name: "restaurant.initiated",
		Data: []byte(`{}`),
	}

	err := dispatcher.Dispatch(context.Background(), event)

	assert.NoError(t, err)

	assert.False(t, first.Called)
	assert.True(t, second.Called)
	assert.Equal(t, event, second.Passed)
}
