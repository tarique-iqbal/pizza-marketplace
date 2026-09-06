package order_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"order-service/internal/domain/order"
)

func TestOrder_Confirm_TransitionsToConfirmed(t *testing.T) {
	o := order.Order{
		ID:           uuid.New(),
		CustomerID:   uuid.New(),
		RestaurantID: uuid.New(),
		Status:       order.StatusPending,
	}

	err := o.Confirm()
	require.NoError(t, err)

	assert.Equal(t, order.StatusConfirmed, o.Status)
	require.NotNil(t, o.ConfirmedAt)

	events := o.PullEvents()
	require.Len(t, events, 1)

	event, ok := events[0].(order.OrderConfirmed)
	require.True(t, ok)

	assert.Equal(t, o.ID, event.OrderID)
	assert.Equal(t, o.CustomerID, event.CustomerID)
	assert.Equal(t, o.RestaurantID, event.RestaurantID)
	assert.Equal(t, "order.confirmed", event.GetEventName())
	assert.False(t, event.OccurredAt.IsZero())
}

func TestOrder_Confirm_FailsIfNotPendingPayment(t *testing.T) {
	for _, status := range []order.OrderStatus{
		order.StatusConfirmed,
		order.StatusPreparing,
		order.StatusReady,
		order.StatusCompleted,
		order.StatusCancelled,
	} {
		o := order.Order{ID: uuid.New(), Status: status}

		err := o.Confirm()

		require.ErrorIs(t, err, order.ErrInvalidStatusTransition)
		assert.Equal(t, status, o.Status)
		assert.Empty(t, o.PullEvents())
	}
}

func TestOrder_StartPreparing_TransitionsToPreparing(t *testing.T) {
	o := order.Order{ID: uuid.New(), Status: order.StatusConfirmed}

	err := o.StartPreparing()
	require.NoError(t, err)

	assert.Equal(t, order.StatusPreparing, o.Status)
	require.NotNil(t, o.PrepStartedAt)
}

func TestOrder_StartPreparing_FailsIfNotConfirmed(t *testing.T) {
	for _, status := range []order.OrderStatus{
		order.StatusPending,
		order.StatusPreparing,
		order.StatusReady,
		order.StatusCompleted,
		order.StatusCancelled,
	} {
		o := order.Order{ID: uuid.New(), Status: status}

		err := o.StartPreparing()

		require.ErrorIs(t, err, order.ErrInvalidStatusTransition)
		assert.Equal(t, status, o.Status)
	}
}

func TestOrder_MarkReady_TransitionsToReady(t *testing.T) {
	o := order.Order{ID: uuid.New(), Status: order.StatusPreparing}

	err := o.MarkReady()
	require.NoError(t, err)

	assert.Equal(t, order.StatusReady, o.Status)
	require.NotNil(t, o.ReadyAt)
}

func TestOrder_MarkReady_FailsIfNotPreparing(t *testing.T) {
	for _, status := range []order.OrderStatus{
		order.StatusPending,
		order.StatusConfirmed,
		order.StatusReady,
		order.StatusCompleted,
		order.StatusCancelled,
	} {
		o := order.Order{ID: uuid.New(), Status: status}

		err := o.MarkReady()

		require.ErrorIs(t, err, order.ErrInvalidStatusTransition)
		assert.Equal(t, status, o.Status)
	}
}

func TestOrder_Complete_TransitionsToCompleted(t *testing.T) {
	o := order.Order{ID: uuid.New(), Status: order.StatusReady}

	err := o.Complete()
	require.NoError(t, err)

	assert.Equal(t, order.StatusCompleted, o.Status)
	require.NotNil(t, o.CompletedAt)
}

func TestOrder_Complete_FailsIfNotReady(t *testing.T) {
	for _, status := range []order.OrderStatus{
		order.StatusPending,
		order.StatusConfirmed,
		order.StatusPreparing,
		order.StatusCompleted,
		order.StatusCancelled,
	} {
		o := order.Order{ID: uuid.New(), Status: status}

		err := o.Complete()

		require.ErrorIs(t, err, order.ErrInvalidStatusTransition)
		assert.Equal(t, status, o.Status)
	}
}

func TestOrder_Cancel_TransitionsToCancelled(t *testing.T) {
	for _, status := range []order.OrderStatus{
		order.StatusPending,
		order.StatusConfirmed,
		order.StatusPreparing,
	} {
		o := order.Order{ID: uuid.New(), Status: status}

		err := o.Cancel()
		require.NoError(t, err)

		assert.Equal(t, order.StatusCancelled, o.Status)
		require.NotNil(t, o.CancelledAt)
		assert.Empty(t, o.PullEvents(), "cancellation raises no domain event")
	}
}

func TestOrder_Cancel_FailsIfAlreadyTerminal(t *testing.T) {
	for _, status := range []order.OrderStatus{
		order.StatusReady,
		order.StatusCompleted,
		order.StatusCancelled,
	} {
		o := order.Order{ID: uuid.New(), Status: status}

		err := o.Cancel()

		require.ErrorIs(t, err, order.ErrInvalidStatusTransition)
		assert.Equal(t, status, o.Status)
	}
}

func TestOrder_PullEvents_DrainsQueue(t *testing.T) {
	o := order.Order{ID: uuid.New(), Status: order.StatusPending}

	require.NoError(t, o.Confirm())

	require.Len(t, o.PullEvents(), 1)
	assert.Empty(t, o.PullEvents())
}
