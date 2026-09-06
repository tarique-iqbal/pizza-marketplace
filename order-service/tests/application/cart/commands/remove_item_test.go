package commands_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"order-service/internal/application/cart/commands"
	"order-service/internal/domain/cart"
	apperr "order-service/internal/shared/errors"
	"order-service/tests/testutil"
)

func TestRemoveItem_CartNotFound(t *testing.T) {
	cartRepo := &testutil.MockCartRepository{}

	uc := commands.NewRemoveItem(cartRepo)

	err := uc.Execute(context.Background(), testutil.MustNewID(), testutil.MustNewID())

	assert.ErrorIs(t, err, apperr.ErrNotFound)
}

func TestRemoveItem_RemovesScopedToCustomerCart(t *testing.T) {
	customerID := testutil.MustNewID()
	itemID := testutil.MustNewID()
	existingCart := &cart.Cart{ID: testutil.MustNewID(), CustomerID: customerID}

	cartRepo := &testutil.MockCartRepository{FindByCustomerResult: existingCart}

	uc := commands.NewRemoveItem(cartRepo)

	err := uc.Execute(context.Background(), customerID, itemID)

	require.NoError(t, err)
	require.Len(t, cartRepo.RemoveItemCalls, 1)
	call := cartRepo.RemoveItemCalls[0]
	assert.Equal(t, existingCart.ID, call.CartID)
	assert.Equal(t, itemID, call.ItemID)
}

func TestRemoveItem_PropagatesNotFoundFromRepo(t *testing.T) {
	existingCart := &cart.Cart{ID: testutil.MustNewID()}
	cartRepo := &testutil.MockCartRepository{
		FindByCustomerResult: existingCart,
		RemoveItemErr:        apperr.ErrNotFound,
	}

	uc := commands.NewRemoveItem(cartRepo)

	err := uc.Execute(context.Background(), testutil.MustNewID(), testutil.MustNewID())

	assert.ErrorIs(t, err, apperr.ErrNotFound)
}
