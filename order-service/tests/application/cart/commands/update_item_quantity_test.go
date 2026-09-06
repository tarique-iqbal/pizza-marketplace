package commands_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	cartapp "order-service/internal/application/cart"
	"order-service/internal/application/cart/commands"
	"order-service/internal/domain/cart"
	apperr "order-service/internal/shared/errors"
	"order-service/tests/testutil"
)

func TestUpdateItemQuantity_CartNotFound(t *testing.T) {
	cartRepo := &testutil.MockCartRepository{}

	uc := commands.NewUpdateItemQuantity(cartRepo)

	_, err := uc.Execute(context.Background(), testutil.MustNewID(), testutil.MustNewID(),
		cartapp.UpdateItemQuantityRequest{Quantity: 3})

	assert.ErrorIs(t, err, apperr.ErrNotFound)
}

func TestUpdateItemQuantity_UpdatesScopedToCustomerCart(t *testing.T) {
	customerID := testutil.MustNewID()
	itemID := testutil.MustNewID()
	existingCart := &cart.Cart{ID: testutil.MustNewID(), CustomerID: customerID}

	cartRepo := &testutil.MockCartRepository{FindByCustomerResult: existingCart}

	uc := commands.NewUpdateItemQuantity(cartRepo)

	res, err := uc.Execute(context.Background(), customerID, itemID, cartapp.UpdateItemQuantityRequest{Quantity: 5})

	require.NoError(t, err)
	assert.Equal(t, itemID, res.ItemID)
	assert.Equal(t, int16(5), res.Quantity)

	require.Len(t, cartRepo.UpdateItemQuantityCalls, 1)
	call := cartRepo.UpdateItemQuantityCalls[0]
	assert.Equal(t, existingCart.ID, call.CartID)
	assert.Equal(t, itemID, call.ItemID)
	assert.Equal(t, int16(5), call.Quantity)
}

func TestUpdateItemQuantity_PropagatesNotFoundFromRepo(t *testing.T) {
	existingCart := &cart.Cart{ID: testutil.MustNewID()}
	cartRepo := &testutil.MockCartRepository{
		FindByCustomerResult:  existingCart,
		UpdateItemQuantityErr: apperr.ErrNotFound,
	}

	uc := commands.NewUpdateItemQuantity(cartRepo)

	_, err := uc.Execute(context.Background(), testutil.MustNewID(), testutil.MustNewID(),
		cartapp.UpdateItemQuantityRequest{Quantity: 1})

	assert.ErrorIs(t, err, apperr.ErrNotFound)
}
