package handlers_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"order-service/internal/application/order/handlers"
	"order-service/internal/domain/order"
	"order-service/internal/infrastructure/persistence"
)

func TestPaymentFailedHandler_Handle_CancelsOrder(t *testing.T) {
	db, _, ord := seedPendingOrder(t)
	h := handlers.NewPaymentFailedHandler(db.DB, persistence.NewOrderRepository(db.DB))

	err := h.Handle(newPaymentEventPayload(t, "payment.failed", "order", ord.ID))
	require.NoError(t, err)

	updated, err := persistence.NewOrderRepository(db.DB).FindByID(context.Background(), ord.ID)
	require.NoError(t, err)
	assert.Equal(t, order.StatusCancelled, updated.Status)
}

func TestPaymentFailedHandler_Handle_NoOpWhenAlreadyCancelled(t *testing.T) {
	db, _, ord := seedPendingOrder(t)
	h := handlers.NewPaymentFailedHandler(db.DB, persistence.NewOrderRepository(db.DB))

	require.NoError(t, h.Handle(newPaymentEventPayload(t, "payment.failed", "order", ord.ID)))
	require.NoError(t, h.Handle(newPaymentEventPayload(t, "payment.failed", "order", ord.ID)))

	updated, err := persistence.NewOrderRepository(db.DB).FindByID(context.Background(), ord.ID)
	require.NoError(t, err)
	assert.Equal(t, order.StatusCancelled, updated.Status)
}

func TestPaymentFailedHandler_Handle_IgnoresNonOrderSubject(t *testing.T) {
	db, _, ord := seedPendingOrder(t)
	h := handlers.NewPaymentFailedHandler(db.DB, persistence.NewOrderRepository(db.DB))

	require.NoError(t, h.Handle(newPaymentEventPayload(t, "payment.failed", "subscription", ord.ID)))

	updated, err := persistence.NewOrderRepository(db.DB).FindByID(context.Background(), ord.ID)
	require.NoError(t, err)
	assert.Equal(t, order.StatusPending, updated.Status)
}
