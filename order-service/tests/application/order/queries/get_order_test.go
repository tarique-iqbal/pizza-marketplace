package queries_test

import (
	"context"
	"errors"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"order-service/internal/application/order/queries"
	"order-service/internal/domain/order"
	apperr "order-service/internal/shared/errors"
	"order-service/tests/testutil"
)

func TestGetOrder_AsCustomer_Success(t *testing.T) {
	ord := &order.Order{
		ID:         testutil.MustNewID(),
		CustomerID: testutil.MustNewID(),
		Status:     order.StatusPending,
		Subtotal:   decimal.NewFromInt(10),
		Total:      decimal.NewFromInt(10),
		Currency:   "EUR",
	}
	repo := &testutil.MockOrderRepository{FindByIDAndCustomerResult: ord}
	uc := queries.NewGetOrder(repo)

	res, err := uc.Execute(context.Background(), ord.ID, ord.CustomerID, "customer")

	require.NoError(t, err)
	assert.Equal(t, ord.ID, res.OrderID)
}

func TestGetOrder_AsOwner_Success(t *testing.T) {
	ord := &order.Order{
		ID:       testutil.MustNewID(),
		Status:   order.StatusConfirmed,
		Subtotal: decimal.NewFromInt(10),
		Total:    decimal.NewFromInt(10),
		Currency: "EUR",
	}
	repo := &testutil.MockOrderRepository{FindByIDAndRestaurantOwnerResult: ord}
	uc := queries.NewGetOrder(repo)

	res, err := uc.Execute(context.Background(), ord.ID, testutil.MustNewID(), "owner")

	require.NoError(t, err)
	assert.Equal(t, ord.ID, res.OrderID)
}

func TestGetOrder_NotFoundOrWrongCaller_ReturnsForbidden(t *testing.T) {
	repo := &testutil.MockOrderRepository{FindByIDAndCustomerErr: apperr.ErrNotFound}
	uc := queries.NewGetOrder(repo)

	_, err := uc.Execute(context.Background(), testutil.MustNewID(), testutil.MustNewID(), "customer")

	require.Error(t, err)
	assert.ErrorIs(t, err, apperr.ErrForbidden)
}

func TestGetOrder_RepositoryError_Propagates(t *testing.T) {
	repo := &testutil.MockOrderRepository{FindByIDAndCustomerErr: errors.New("db down")}
	uc := queries.NewGetOrder(repo)

	_, err := uc.Execute(context.Background(), testutil.MustNewID(), testutil.MustNewID(), "customer")

	require.Error(t, err)
	assert.NotErrorIs(t, err, apperr.ErrForbidden)
}
