package handlers_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"

	"order-service/internal/application/order/handlers"
	"order-service/internal/domain/order"
	"order-service/internal/infrastructure/persistence"
	"order-service/tests/testutil"
)

var paymentFailedSource = attribute.String("source", "payment_failed")

func TestPaymentFailedHandler_Handle_CancelsOrder(t *testing.T) {
	db, _, ord := seedPendingOrder(t)
	h := handlers.NewPaymentFailedHandler(db.DB, persistence.NewOrderRepository(db.DB))
	cancelledBefore := testutil.CounterValue(t, "orders.cancelled", paymentFailedSource)

	err := h.Handle(context.Background(), newPaymentEventPayload(t, "payment.failed", "order", ord.ID))
	require.NoError(t, err)

	updated, err := persistence.NewOrderRepository(db.DB).FindByID(context.Background(), ord.ID)
	require.NoError(t, err)
	assert.Equal(t, order.StatusCancelled, updated.Status)
	assert.Equal(t, cancelledBefore+1, testutil.CounterValue(t, "orders.cancelled", paymentFailedSource))
}

func TestPaymentFailedHandler_Handle_NoOpWhenAlreadyCancelled(t *testing.T) {
	db, _, ord := seedPendingOrder(t)
	h := handlers.NewPaymentFailedHandler(db.DB, persistence.NewOrderRepository(db.DB))
	cancelledBefore := testutil.CounterValue(t, "orders.cancelled", paymentFailedSource)

	require.NoError(t, h.Handle(
		context.Background(),
		newPaymentEventPayload(t, "payment.failed", "order", ord.ID),
	))
	require.NoError(t, h.Handle(
		context.Background(),
		newPaymentEventPayload(t, "payment.failed", "order", ord.ID),
	))

	updated, err := persistence.NewOrderRepository(db.DB).FindByID(context.Background(), ord.ID)
	require.NoError(t, err)
	assert.Equal(t, order.StatusCancelled, updated.Status)
	assert.Equal(t, cancelledBefore+1, testutil.CounterValue(t, "orders.cancelled", paymentFailedSource),
		"redelivery must not count a second cancellation")
}

func TestPaymentFailedHandler_Handle_IgnoresNonOrderSubject(t *testing.T) {
	db, _, ord := seedPendingOrder(t)
	h := handlers.NewPaymentFailedHandler(db.DB, persistence.NewOrderRepository(db.DB))

	require.NoError(t, h.Handle(
		context.Background(),
		newPaymentEventPayload(t, "payment.failed", "subscription", ord.ID),
	))

	updated, err := persistence.NewOrderRepository(db.DB).FindByID(context.Background(), ord.ID)
	require.NoError(t, err)
	assert.Equal(t, order.StatusPending, updated.Status)
}
