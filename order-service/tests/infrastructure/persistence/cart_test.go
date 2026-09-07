package persistence_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"order-service/internal/domain/cart"
	"order-service/internal/infrastructure/persistence"
	apperr "order-service/internal/shared/errors"
	"order-service/tests/infrastructure/db/fixtures"
	"order-service/tests/testutil"
)

func setupCartRepo(t *testing.T) (cart.CartRepository, []cart.Cart) {
	db := testutil.DB(t)
	db.TruncateTables(t, testutil.TableCart)

	seeded := fixtures.LoadCartFixtures(t, db.DB)

	return persistence.NewCartRepository(db.DB), seeded
}

func TestCartRepository_FindByCustomer(t *testing.T) {
	repo, seeded := setupCartRepo(t)

	found, err := repo.FindByCustomer(context.Background(), seeded[0].CustomerID)

	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, seeded[0].ID, found.ID)
	require.Len(t, found.Items, 1)
	assert.Equal(t, int16(2), found.Items[0].Quantity)
}

func TestCartRepository_FindByCustomer_NoneReturnsNil(t *testing.T) {
	repo, _ := setupCartRepo(t)

	found, err := repo.FindByCustomer(context.Background(), testutil.MustNewID())

	require.NoError(t, err)
	assert.Nil(t, found)
}

func TestCartRepository_Create(t *testing.T) {
	repo, _ := setupCartRepo(t)

	newCart := cart.NewCart(testutil.MustNewID(), testutil.MustNewID(), testutil.MustNewID())

	err := repo.Create(context.Background(), newCart)
	require.NoError(t, err)

	found, err := repo.FindByCustomer(context.Background(), newCart.CustomerID)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, newCart.ID, found.ID)
}

func TestCartRepository_AddOrMergeItem_InsertsNewLine(t *testing.T) {
	repo, seeded := setupCartRepo(t)
	target := seeded[1]

	item := cart.CartItem{
		ID:              testutil.MustNewID(),
		PizzaID:         testutil.MustNewID(),
		SizeID:          testutil.MustNewID(),
		Quantity:        1,
		ExtraToppingIDs: []uuid.UUID{},
	}

	err := repo.AddOrMergeItem(context.Background(), target.ID, item)
	require.NoError(t, err)

	found, err := repo.FindByCustomer(context.Background(), target.CustomerID)
	require.NoError(t, err)
	require.Len(t, found.Items, 1)
	assert.Equal(t, int16(1), found.Items[0].Quantity)
}

func TestCartRepository_AddOrMergeItem_MergesQuantityOnSameCombo(t *testing.T) {
	repo, seeded := setupCartRepo(t)
	target := seeded[0]
	existing := target.Items[0]

	duplicate := cart.CartItem{
		ID:              testutil.MustNewID(),
		PizzaID:         existing.PizzaID,
		SizeID:          existing.SizeID,
		Quantity:        3,
		ExtraToppingIDs: existing.ExtraToppingIDs,
	}

	err := repo.AddOrMergeItem(context.Background(), target.ID, duplicate)
	require.NoError(t, err)

	found, err := repo.FindByCustomer(context.Background(), target.CustomerID)
	require.NoError(t, err)
	require.Len(t, found.Items, 1, "same pizza+size+toppings must merge into one line")
	assert.Equal(t, int16(5), found.Items[0].Quantity)
}

func TestCartRepository_AddOrMergeItem_DifferentToppingsSeparateLine(t *testing.T) {
	repo, seeded := setupCartRepo(t)
	target := seeded[0]
	existing := target.Items[0]

	differentToppings := cart.CartItem{
		ID:              testutil.MustNewID(),
		PizzaID:         existing.PizzaID,
		SizeID:          existing.SizeID,
		Quantity:        1,
		ExtraToppingIDs: []uuid.UUID{testutil.MustNewID()},
	}

	err := repo.AddOrMergeItem(context.Background(), target.ID, differentToppings)
	require.NoError(t, err)

	found, err := repo.FindByCustomer(context.Background(), target.CustomerID)
	require.NoError(t, err)
	assert.Len(t, found.Items, 2, "same pizza+size but different toppings must stay separate lines")
}

func TestCartRepository_UpdateItemQuantity(t *testing.T) {
	repo, seeded := setupCartRepo(t)
	target := seeded[0]
	item := target.Items[0]

	err := repo.UpdateItemQuantity(context.Background(), target.ID, item.ID, 9)
	require.NoError(t, err)

	found, err := repo.FindByCustomer(context.Background(), target.CustomerID)
	require.NoError(t, err)
	assert.Equal(t, int16(9), found.Items[0].Quantity)
}

func TestCartRepository_UpdateItemQuantity_NotFound(t *testing.T) {
	repo, seeded := setupCartRepo(t)
	target := seeded[0]

	err := repo.UpdateItemQuantity(context.Background(), target.ID, testutil.MustNewID(), 9)

	assert.ErrorIs(t, err, apperr.ErrNotFound)
}

func TestCartRepository_RemoveItem_DeletesCartWhenLastItemRemoved(t *testing.T) {
	repo, seeded := setupCartRepo(t)
	target := seeded[0]
	item := target.Items[0]

	err := repo.RemoveItem(context.Background(), target.ID, item.ID)
	require.NoError(t, err)

	found, err := repo.FindByCustomer(context.Background(), target.CustomerID)
	require.NoError(t, err)
	assert.Nil(t, found, "an emptied cart must not linger as an orphan row")
}

func TestCartRepository_RemoveItem_CartSurvivesWhenItemsRemain(t *testing.T) {
	repo, seeded := setupCartRepo(t)
	target := seeded[0]
	original := target.Items[0]

	extra := cart.CartItem{
		ID:              testutil.MustNewID(),
		PizzaID:         testutil.MustNewID(),
		SizeID:          testutil.MustNewID(),
		Quantity:        1,
		ExtraToppingIDs: []uuid.UUID{},
	}
	require.NoError(t, repo.AddOrMergeItem(context.Background(), target.ID, extra))

	err := repo.RemoveItem(context.Background(), target.ID, original.ID)
	require.NoError(t, err)

	found, err := repo.FindByCustomer(context.Background(), target.CustomerID)
	require.NoError(t, err)
	require.NotNil(t, found)
	require.Len(t, found.Items, 1)
	assert.Equal(t, extra.PizzaID, found.Items[0].PizzaID)
}

func TestCartRepository_RemoveItem_NotFound(t *testing.T) {
	repo, seeded := setupCartRepo(t)
	target := seeded[0]

	err := repo.RemoveItem(context.Background(), target.ID, testutil.MustNewID())

	assert.ErrorIs(t, err, apperr.ErrNotFound)
}

func TestCartRepository_Clear_DeletesCartAndItems(t *testing.T) {
	repo, seeded := setupCartRepo(t)
	target := seeded[0]

	err := repo.Clear(context.Background(), target.ID)
	require.NoError(t, err)

	found, err := repo.FindByCustomer(context.Background(), target.CustomerID)
	require.NoError(t, err)
	assert.Nil(t, found)
}
