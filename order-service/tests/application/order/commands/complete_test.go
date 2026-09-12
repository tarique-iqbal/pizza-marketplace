package commands_test

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"order-service/internal/application/order/commands"
	"order-service/internal/domain/order"
	apperr "order-service/internal/shared/errors"
	"order-service/tests/testutil"
)

func TestComplete_Success(t *testing.T) {
	ord := &order.Order{
		ID: testutil.MustNewID(), Status: order.StatusReady,
		Subtotal: decimal.NewFromInt(10), Total: decimal.NewFromInt(10), Currency: "EUR",
	}
	repo := &testutil.MockOrderRepository{FindByIDAndRestaurantOwnerResult: ord}
	uc := commands.NewComplete(repo)

	res, err := uc.Execute(context.Background(), ord.ID, testutil.MustNewID())

	require.NoError(t, err)
	assert.Equal(t, order.StatusCompleted, res.Status)
	require.Len(t, repo.Updated, 1)
	assert.Equal(t, order.StatusCompleted, repo.Updated[0].Status)
}

func TestComplete_NotOwner_ReturnsForbidden(t *testing.T) {
	repo := &testutil.MockOrderRepository{FindByIDAndRestaurantOwnerErr: apperr.ErrNotFound}
	uc := commands.NewComplete(repo)

	_, err := uc.Execute(context.Background(), testutil.MustNewID(), testutil.MustNewID())

	require.Error(t, err)
	assert.ErrorIs(t, err, apperr.ErrForbidden)
}

func TestComplete_FromConfirmed_Success(t *testing.T) {
	ord := &order.Order{
		ID: testutil.MustNewID(), Status: order.StatusConfirmed,
		Subtotal: decimal.NewFromInt(10), Total: decimal.NewFromInt(10), Currency: "EUR",
	}
	repo := &testutil.MockOrderRepository{FindByIDAndRestaurantOwnerResult: ord}
	uc := commands.NewComplete(repo)

	res, err := uc.Execute(context.Background(), ord.ID, testutil.MustNewID())

	require.NoError(t, err)
	assert.Equal(t, order.StatusCompleted, res.Status)
}

func TestComplete_InvalidTransition_ReturnsConflict(t *testing.T) {
	ord := &order.Order{
		ID: testutil.MustNewID(), Status: order.StatusPending,
		Subtotal: decimal.NewFromInt(10), Total: decimal.NewFromInt(10), Currency: "EUR",
	}
	repo := &testutil.MockOrderRepository{FindByIDAndRestaurantOwnerResult: ord}
	uc := commands.NewComplete(repo)

	_, err := uc.Execute(context.Background(), ord.ID, testutil.MustNewID())

	require.Error(t, err)
	assert.ErrorIs(t, err, apperr.ErrConflict)
	assert.Empty(t, repo.Updated)
}
