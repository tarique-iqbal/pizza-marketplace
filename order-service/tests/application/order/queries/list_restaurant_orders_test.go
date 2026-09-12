package queries_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	orderapp "order-service/internal/application/order"
	"order-service/internal/application/order/queries"
	"order-service/internal/domain/order"
	apperr "order-service/internal/shared/errors"
	"order-service/tests/testutil"
)

func TestListRestaurantOrders_FewerThanLimit_NoNextCursor(t *testing.T) {
	orders := []order.Order{newOrderAt(testutil.MustNewID(), time.Now())}
	repo := &testutil.MockOrderRepository{ListByRestaurantResult: orders}
	uc := queries.NewListRestaurantOrders(repo)

	res, err := uc.Execute(context.Background(), testutil.MustNewID(), testutil.MustNewID(), "", 20)

	require.NoError(t, err)
	require.Len(t, res.Orders, 1)
	assert.Empty(t, res.NextCursor)
	require.Len(t, repo.ListByRestaurantCalls, 1)
	assert.Equal(t, 21, repo.ListByRestaurantCalls[0].Limit)
}

func TestListRestaurantOrders_MoreThanLimit_TrimsAndSetsNextCursor(t *testing.T) {
	now := time.Now().UTC()
	orders := []order.Order{
		newOrderAt(testutil.MustNewID(), now),
		newOrderAt(testutil.MustNewID(), now.Add(-time.Minute)),
		newOrderAt(testutil.MustNewID(), now.Add(-2*time.Minute)),
	}
	repo := &testutil.MockOrderRepository{ListByRestaurantResult: orders}
	uc := queries.NewListRestaurantOrders(repo)

	res, err := uc.Execute(context.Background(), testutil.MustNewID(), testutil.MustNewID(), "", 2)

	require.NoError(t, err)
	require.Len(t, res.Orders, 2)
	require.NotEmpty(t, res.NextCursor)

	cursor, err := orderapp.DecodeCursor(res.NextCursor)
	require.NoError(t, err)
	assert.Equal(t, orders[1].ID, cursor.ID)
}

func TestListRestaurantOrders_PassesRestaurantAndOwnerID(t *testing.T) {
	restaurantID := testutil.MustNewID()
	ownerID := testutil.MustNewID()
	repo := &testutil.MockOrderRepository{}
	uc := queries.NewListRestaurantOrders(repo)

	_, err := uc.Execute(context.Background(), restaurantID, ownerID, "", 20)

	require.NoError(t, err)
	require.Len(t, repo.ListByRestaurantCalls, 1)
	assert.Equal(t, restaurantID, repo.ListByRestaurantCalls[0].RestaurantID)
	assert.Equal(t, ownerID, repo.ListByRestaurantCalls[0].OwnerID)
}

func TestListRestaurantOrders_InvalidCursor_ReturnsInvalid(t *testing.T) {
	repo := &testutil.MockOrderRepository{}
	uc := queries.NewListRestaurantOrders(repo)

	_, err := uc.Execute(context.Background(), testutil.MustNewID(), testutil.MustNewID(), "not-a-cursor!!!", 20)

	require.Error(t, err)
	assert.ErrorIs(t, err, apperr.ErrInvalid)
	assert.Empty(t, repo.ListByRestaurantCalls)
}

func TestListRestaurantOrders_RepositoryError_Propagates(t *testing.T) {
	repo := &testutil.MockOrderRepository{ListByRestaurantErr: errors.New("db down")}
	uc := queries.NewListRestaurantOrders(repo)

	_, err := uc.Execute(context.Background(), testutil.MustNewID(), testutil.MustNewID(), "", 20)

	require.Error(t, err)
}
