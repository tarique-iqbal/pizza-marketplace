package queries_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	orderapp "order-service/internal/application/order"
	"order-service/internal/application/order/queries"
	"order-service/internal/domain/order"
	apperr "order-service/internal/shared/errors"
	"order-service/tests/testutil"
)

func newOrderAt(customerID uuid.UUID, placedAt time.Time) order.Order {
	return order.Order{
		ID: testutil.MustNewID(), CustomerID: customerID, Status: order.StatusPending,
		Subtotal: decimal.NewFromInt(10), Total: decimal.NewFromInt(10), Currency: "EUR",
		PlacedAt: placedAt,
	}
}

func TestListMyOrders_FewerThanLimit_NoNextCursor(t *testing.T) {
	customerID := testutil.MustNewID()
	orders := []order.Order{newOrderAt(customerID, time.Now())}
	repo := &testutil.MockOrderRepository{ListByCustomerResult: orders}
	uc := queries.NewListMyOrders(repo)

	res, err := uc.Execute(context.Background(), customerID, "", 20)

	require.NoError(t, err)
	require.Len(t, res.Orders, 1)
	assert.Empty(t, res.NextCursor)
	require.Len(t, repo.ListByCustomerCalls, 1)
	assert.Equal(t, 21, repo.ListByCustomerCalls[0].Limit, "must request limit+1 to detect a next page")
}

func TestListMyOrders_MoreThanLimit_TrimsAndSetsNextCursor(t *testing.T) {
	customerID := testutil.MustNewID()
	now := time.Now().UTC()
	orders := []order.Order{
		newOrderAt(customerID, now),
		newOrderAt(customerID, now.Add(-time.Minute)),
		newOrderAt(customerID, now.Add(-2*time.Minute)),
	}
	repo := &testutil.MockOrderRepository{ListByCustomerResult: orders}
	uc := queries.NewListMyOrders(repo)

	res, err := uc.Execute(context.Background(), customerID, "", 2)

	require.NoError(t, err)
	require.Len(t, res.Orders, 2, "trimmed to the requested limit, the extra row only signals hasMore")
	require.NotEmpty(t, res.NextCursor)

	cursor, err := orderapp.DecodeCursor(res.NextCursor)
	require.NoError(t, err)
	assert.Equal(t, orders[1].ID, cursor.ID, "next cursor seeks from the last KEPT row, not the trimmed one")
}

func TestListMyOrders_InvalidCursor_ReturnsInvalid(t *testing.T) {
	repo := &testutil.MockOrderRepository{}
	uc := queries.NewListMyOrders(repo)

	_, err := uc.Execute(context.Background(), testutil.MustNewID(), "not-a-cursor!!!", 20)

	require.Error(t, err)
	assert.ErrorIs(t, err, apperr.ErrInvalid)
	assert.Empty(t, repo.ListByCustomerCalls, "must not call the repo with an undecodable cursor")
}

func TestListMyOrders_RepositoryError_Propagates(t *testing.T) {
	repo := &testutil.MockOrderRepository{ListByCustomerErr: errors.New("db down")}
	uc := queries.NewListMyOrders(repo)

	_, err := uc.Execute(context.Background(), testutil.MustNewID(), "", 20)

	require.Error(t, err)
}
