package commands_test

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"

	"order-service/internal/application/order/commands"
	"order-service/internal/domain/order"
	apperr "order-service/internal/shared/errors"
	"order-service/tests/testutil"
)

func TestComplete_Success(t *testing.T) {
	ord := &order.Order{
		ID: testutil.MustNewID(), Status: order.StatusReady, Fulfillment: order.FulfillmentDelivery,
		Subtotal: decimal.NewFromInt(10), Total: decimal.NewFromInt(10), Currency: "EUR",
		PlacedAt: time.Now().UTC().Add(-25 * time.Minute),
	}
	repo := &testutil.MockOrderRepository{FindByIDAndRestaurantOwnerResult: ord}
	cmd := commands.NewComplete(repo)
	delivery := attribute.String("fulfillment", "delivery")
	completedBefore := testutil.CounterValue(t, "orders.completed", delivery)
	durationsBefore, secondsBefore := testutil.HistogramCountAndSum(t, "order.completion.duration", delivery)

	res, err := cmd.Execute(context.Background(), ord.ID, testutil.MustNewID())

	require.NoError(t, err)
	assert.Equal(t, order.StatusCompleted, res.Status)
	require.Len(t, repo.Updated, 1)
	assert.Equal(t, order.StatusCompleted, repo.Updated[0].Status)
	assert.Equal(t, completedBefore+1, testutil.CounterValue(t, "orders.completed", delivery))

	durations, seconds := testutil.HistogramCountAndSum(t, "order.completion.duration", delivery)
	assert.Equal(t, durationsBefore+1, durations)
	assert.InDelta(t, 25*60, seconds-secondsBefore, 5)
}

func TestComplete_NotOwner_ReturnsForbidden(t *testing.T) {
	repo := &testutil.MockOrderRepository{FindByIDAndRestaurantOwnerErr: apperr.ErrNotFound}
	cmd := commands.NewComplete(repo)

	_, err := cmd.Execute(context.Background(), testutil.MustNewID(), testutil.MustNewID())

	require.Error(t, err)
	assert.ErrorIs(t, err, apperr.ErrForbidden)
}

func TestComplete_FromConfirmed_Success(t *testing.T) {
	ord := &order.Order{
		ID: testutil.MustNewID(), Status: order.StatusConfirmed,
		Subtotal: decimal.NewFromInt(10), Total: decimal.NewFromInt(10), Currency: "EUR",
	}
	repo := &testutil.MockOrderRepository{FindByIDAndRestaurantOwnerResult: ord}
	cmd := commands.NewComplete(repo)

	res, err := cmd.Execute(context.Background(), ord.ID, testutil.MustNewID())

	require.NoError(t, err)
	assert.Equal(t, order.StatusCompleted, res.Status)
}

func TestComplete_InvalidTransition_ReturnsConflict(t *testing.T) {
	ord := &order.Order{
		ID: testutil.MustNewID(), Status: order.StatusPending,
		Subtotal: decimal.NewFromInt(10), Total: decimal.NewFromInt(10), Currency: "EUR",
	}
	repo := &testutil.MockOrderRepository{FindByIDAndRestaurantOwnerResult: ord}
	cmd := commands.NewComplete(repo)
	completedBefore := testutil.CounterValue(t, "orders.completed")
	durationsBefore, _ := testutil.HistogramCountAndSum(t, "order.completion.duration")

	_, err := cmd.Execute(context.Background(), ord.ID, testutil.MustNewID())

	require.Error(t, err)
	assert.ErrorIs(t, err, apperr.ErrConflict)
	assert.Empty(t, repo.Updated)
	assert.Equal(t, completedBefore, testutil.CounterValue(t, "orders.completed"))

	durations, _ := testutil.HistogramCountAndSum(t, "order.completion.duration")
	assert.Equal(t, durationsBefore, durations)
}
