package queries_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"order-service/internal/application/cart/queries"
	"order-service/internal/domain/cart"
	"order-service/internal/domain/readmodel"
	apperr "order-service/internal/shared/errors"
	"order-service/tests/testutil"
)

func TestGetCart_NoCart_ReturnsEmptyView(t *testing.T) {
	cartRepo := &testutil.MockCartRepository{}
	pizzaRepo := &testutil.MockPizzaRepository{}
	pizzaPriceRepo := &testutil.MockPizzaPriceRepository{}
	toppingPriceRepo := &testutil.MockToppingPriceRepository{}

	uc := queries.NewGetCart(cartRepo, pizzaRepo, pizzaPriceRepo, toppingPriceRepo)

	res, err := uc.Execute(context.Background(), testutil.MustNewID())

	require.NoError(t, err)
	assert.Empty(t, res.Items)
	assert.True(t, decimal.Decimal(res.Subtotal).IsZero())
}

func TestGetCart_ResolvesAvailableItem(t *testing.T) {
	restaurantID := testutil.MustNewID()
	pizzaID := testutil.MustNewID()
	sizeID := testutil.MustNewID()
	toppingID := testutil.MustNewID()
	itemID := testutil.MustNewID()

	existingCart := &cart.Cart{
		ID:           testutil.MustNewID(),
		RestaurantID: restaurantID,
		Items: []cart.CartItem{
			{
				ID:         itemID,
				PizzaID:    pizzaID,
				SizeID:     sizeID,
				Quantity:   2,
				ToppingIDs: []uuid.UUID{toppingID},
			},
		},
	}

	cartRepo := &testutil.MockCartRepository{FindByCustomerResult: existingCart}
	pizzaRepo := &testutil.MockPizzaRepository{
		FindByIDResult: &readmodel.Pizza{ID: pizzaID, RestaurantID: restaurantID, Name: "Margherita"},
	}
	pizzaPriceRepo := &testutil.MockPizzaPriceRepository{
		ListByPizzaResult: []readmodel.PizzaPrice{
			{PizzaID: pizzaID, SizeID: sizeID, DiameterCm: 26, Price: decimal.NewFromFloat(7.50), IsActive: true},
		},
	}
	toppingPriceRepo := &testutil.MockToppingPriceRepository{
		ListByRestaurantResult: []readmodel.ToppingPrice{
			{RestaurantID: restaurantID, ToppingID: toppingID, Name: "Extra Cheese", ExtraPrice: decimal.NewFromFloat(1.50)},
		},
	}

	uc := queries.NewGetCart(cartRepo, pizzaRepo, pizzaPriceRepo, toppingPriceRepo)

	res, err := uc.Execute(context.Background(), testutil.MustNewID())

	require.NoError(t, err)
	require.Len(t, res.Items, 1)

	item := res.Items[0]
	assert.True(t, item.Available)
	assert.Equal(t, "Margherita", item.PizzaName)
	assert.Equal(t, int16(26), item.DiameterCm)
	require.Len(t, item.Toppings, 1)
	assert.Equal(t, "Extra Cheese", item.Toppings[0].Name)
	require.NotNil(t, item.UnitPrice)
	assert.True(t, decimal.Decimal(*item.UnitPrice).Equal(decimal.NewFromFloat(9.00)))
	require.NotNil(t, item.LineTotal)
	assert.True(t, decimal.Decimal(*item.LineTotal).Equal(decimal.NewFromFloat(18.00)))
	assert.True(t, decimal.Decimal(res.Subtotal).Equal(decimal.NewFromFloat(18.00)))
}

func TestGetCart_FlagsArchivedPizzaAsUnavailable(t *testing.T) {
	restaurantID := testutil.MustNewID()
	pizzaID := testutil.MustNewID()

	existingCart := &cart.Cart{
		ID:           testutil.MustNewID(),
		RestaurantID: restaurantID,
		Items: []cart.CartItem{
			{ID: testutil.MustNewID(), PizzaID: pizzaID, SizeID: testutil.MustNewID(), Quantity: 1},
		},
	}

	cartRepo := &testutil.MockCartRepository{FindByCustomerResult: existingCart}
	pizzaRepo := &testutil.MockPizzaRepository{FindByIDErr: apperr.ErrNotFound}
	pizzaPriceRepo := &testutil.MockPizzaPriceRepository{}
	toppingPriceRepo := &testutil.MockToppingPriceRepository{}

	uc := queries.NewGetCart(cartRepo, pizzaRepo, pizzaPriceRepo, toppingPriceRepo)

	res, err := uc.Execute(context.Background(), testutil.MustNewID())

	require.NoError(t, err)
	require.Len(t, res.Items, 1)
	assert.False(t, res.Items[0].Available)
	assert.Nil(t, res.Items[0].UnitPrice)
	assert.True(t, decimal.Decimal(res.Subtotal).IsZero())
}

func TestGetCart_FlagsDeactivatedSizeAsUnavailable(t *testing.T) {
	restaurantID := testutil.MustNewID()
	pizzaID := testutil.MustNewID()
	sizeID := testutil.MustNewID()

	existingCart := &cart.Cart{
		ID:           testutil.MustNewID(),
		RestaurantID: restaurantID,
		Items: []cart.CartItem{
			{ID: testutil.MustNewID(), PizzaID: pizzaID, SizeID: sizeID, Quantity: 1},
		},
	}

	cartRepo := &testutil.MockCartRepository{FindByCustomerResult: existingCart}
	pizzaRepo := &testutil.MockPizzaRepository{
		FindByIDResult: &readmodel.Pizza{ID: pizzaID, RestaurantID: restaurantID, Name: "Margherita"},
	}
	pizzaPriceRepo := &testutil.MockPizzaPriceRepository{
		ListByPizzaResult: []readmodel.PizzaPrice{
			{PizzaID: pizzaID, SizeID: sizeID, IsActive: false},
		},
	}
	toppingPriceRepo := &testutil.MockToppingPriceRepository{}

	uc := queries.NewGetCart(cartRepo, pizzaRepo, pizzaPriceRepo, toppingPriceRepo)

	res, err := uc.Execute(context.Background(), testutil.MustNewID())

	require.NoError(t, err)
	require.Len(t, res.Items, 1)
	assert.False(t, res.Items[0].Available)
}
