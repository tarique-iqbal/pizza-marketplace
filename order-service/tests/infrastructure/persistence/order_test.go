package persistence_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"order-service/internal/domain/order"
	"order-service/internal/domain/readmodel"
	"order-service/internal/infrastructure/persistence"
	apperr "order-service/internal/shared/errors"
	"order-service/tests/infrastructure/db/fixtures"
	"order-service/tests/testutil"
)

func setupOrderRepo(t *testing.T) (order.OrderRepository, []readmodel.Restaurant, []order.Order) {
	db := testutil.DB(t)
	db.TruncateTables(t, testutil.TableRestaurant)

	restaurants := fixtures.LoadRestaurantFixtures(t, db.DB)
	orders := fixtures.LoadOrderFixtures(t, db.DB, restaurants)

	return persistence.NewOrderRepository(db.DB), restaurants, orders
}

func TestOrderRepository_Create(t *testing.T) {
	repo, restaurants, _ := setupOrderRepo(t)

	id := testutil.MustNewID()
	newOrder := order.NewOrder(
		id, testutil.MustNewID(), restaurants[0].ID,
		order.FulfillmentPickup, "new.customer@example.com", nil,
		nil, nil, nil,
		[]order.OrderItem{},
		decimal.NewFromFloat(10.00), decimal.Zero, decimal.NewFromFloat(10.00),
		"EUR",
	)

	require.NoError(t, repo.Create(context.Background(), newOrder))

	found, err := repo.FindByID(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, "new.customer@example.com", found.ContactEmail)
}

func TestOrderRepository_FindByID_NotFound(t *testing.T) {
	repo, _, _ := setupOrderRepo(t)

	_, err := repo.FindByID(context.Background(), testutil.MustNewID())

	assert.ErrorIs(t, err, apperr.ErrNotFound)
}

func TestOrderRepository_FindByID_PreloadsItems(t *testing.T) {
	repo, _, orders := setupOrderRepo(t)

	found, err := repo.FindByID(context.Background(), orders[0].ID)

	require.NoError(t, err)
	require.Len(t, found.Items, 1)
	assert.Equal(t, "Margherita", found.Items[0].PizzaName)
}

func TestOrderRepository_FindByIDAndCustomer(t *testing.T) {
	repo, _, orders := setupOrderRepo(t)

	found, err := repo.FindByIDAndCustomer(context.Background(), orders[0].ID, orders[0].CustomerID)
	require.NoError(t, err)
	assert.Equal(t, orders[0].ID, found.ID)

	_, err = repo.FindByIDAndCustomer(context.Background(), orders[0].ID, testutil.MustNewID())
	assert.ErrorIs(t, err, apperr.ErrNotFound, "a different customer must not resolve this order")
}

func TestOrderRepository_FindByIDAndRestaurantOwner(t *testing.T) {
	repo, restaurants, orders := setupOrderRepo(t)

	found, err := repo.FindByIDAndRestaurantOwner(context.Background(), orders[0].ID, restaurants[0].OwnerID)
	require.NoError(t, err)
	assert.Equal(t, orders[0].ID, found.ID)

	_, err = repo.FindByIDAndRestaurantOwner(context.Background(), orders[0].ID, restaurants[1].OwnerID)
	assert.ErrorIs(t, err, apperr.ErrNotFound, "a different restaurant's owner must not resolve this order")
}

func TestOrderRepository_ListByCustomer(t *testing.T) {
	repo, _, orders := setupOrderRepo(t)

	found, err := repo.ListByCustomer(context.Background(), orders[0].CustomerID, nil, 10)

	require.NoError(t, err)
	require.Len(t, found, 1)
	assert.Equal(t, orders[0].ID, found[0].ID)
}

func TestOrderRepository_ListByRestaurant(t *testing.T) {
	repo, restaurants, orders := setupOrderRepo(t)

	found, err := repo.ListByRestaurant(context.Background(), restaurants[0].ID, restaurants[0].OwnerID, nil, 10)

	require.NoError(t, err)
	require.Len(t, found, 1)
	assert.Equal(t, orders[0].ID, found[0].ID)
}

func TestOrderRepository_ListByRestaurant_WrongOwner_ReturnsEmpty(t *testing.T) {
	repo, restaurants, _ := setupOrderRepo(t)

	found, err := repo.ListByRestaurant(context.Background(), restaurants[0].ID, restaurants[1].OwnerID, nil, 10)

	require.NoError(t, err)
	assert.Empty(t, found, "a different restaurant's owner must not see this restaurant's orders")
}

func TestOrderRepository_ListByCustomer_KeysetPagination(t *testing.T) {
	repo, restaurants, _ := setupOrderRepo(t)

	customerID := testutil.MustNewID()
	var placed []order.Order
	for i := 0; i < 5; i++ {
		o := order.NewOrder(
			testutil.MustNewID(), customerID, restaurants[0].ID,
			order.FulfillmentPickup, "paging.customer@example.com", nil,
			nil, nil, nil,
			[]order.OrderItem{},
			decimal.NewFromFloat(10.00), decimal.Zero, decimal.NewFromFloat(10.00),
			"EUR",
		)
		require.NoError(t, repo.Create(context.Background(), o))
		placed = append(placed, *o)
	}

	page1, err := repo.ListByCustomer(context.Background(), customerID, nil, 2)
	require.NoError(t, err)
	require.Len(t, page1, 2)

	last := page1[len(page1)-1]
	page2, err := repo.ListByCustomer(context.Background(), customerID, &order.PageCursor{
		PlacedAt: last.PlacedAt, ID: last.ID,
	}, 2)
	require.NoError(t, err)
	require.Len(t, page2, 2)

	seen := map[uuid.UUID]bool{}
	for _, o := range append(page1, page2...) {
		assert.False(t, seen[o.ID], "keyset pages must not repeat a row")
		seen[o.ID] = true
	}

	page3, err := repo.ListByCustomer(context.Background(), customerID, &order.PageCursor{
		PlacedAt: page2[len(page2)-1].PlacedAt, ID: page2[len(page2)-1].ID,
	}, 2)
	require.NoError(t, err)
	require.Len(t, page3, 1, "5 orders paged 2 at a time leaves exactly 1 on the last page")
	assert.False(t, seen[page3[0].ID])
}

func TestOrderRepository_Update(t *testing.T) {
	repo, _, orders := setupOrderRepo(t)
	target := orders[0]

	require.NoError(t, target.Confirm())
	require.NoError(t, repo.Update(context.Background(), &target))

	found, err := repo.FindByID(context.Background(), target.ID)
	require.NoError(t, err)
	assert.Equal(t, order.StatusConfirmed, found.Status)
	require.NotNil(t, found.ConfirmedAt)
}
