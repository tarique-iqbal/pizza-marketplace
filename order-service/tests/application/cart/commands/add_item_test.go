package commands_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	cartapp "order-service/internal/application/cart"
	"order-service/internal/application/cart/commands"
	"order-service/internal/domain/cart"
	"order-service/internal/domain/readmodel"
	apperr "order-service/internal/shared/errors"
	"order-service/tests/testutil"
)

func TestAddItem_PizzaNotFound(t *testing.T) {
	cartRepo := &testutil.MockCartRepository{}
	pizzaRepo := &testutil.MockPizzaRepository{}
	pizzaPriceRepo := &testutil.MockPizzaPriceRepository{}
	toppingPriceRepo := &testutil.MockToppingPriceRepository{}

	uc := commands.NewAddItem(cartRepo, pizzaRepo, pizzaPriceRepo, toppingPriceRepo)

	_, err := uc.Execute(context.Background(), testutil.MustNewID(), cartapp.AddItemRequest{
		PizzaID: testutil.MustNewID(), SizeID: testutil.MustNewID(), Quantity: 1,
	})

	assert.ErrorIs(t, err, apperr.ErrNotFound)
}

func TestAddItem_SizeNotActive(t *testing.T) {
	restaurantID := testutil.MustNewID()
	pizzaID := testutil.MustNewID()
	sizeID := testutil.MustNewID()

	cartRepo := &testutil.MockCartRepository{}
	pizzaRepo := &testutil.MockPizzaRepository{
		FindByIDResult: &readmodel.Pizza{ID: pizzaID, RestaurantID: restaurantID},
	}
	pizzaPriceRepo := &testutil.MockPizzaPriceRepository{
		ListByPizzaResult: []readmodel.PizzaPrice{{PizzaID: pizzaID, SizeID: sizeID, IsActive: false}},
	}
	toppingPriceRepo := &testutil.MockToppingPriceRepository{}

	uc := commands.NewAddItem(cartRepo, pizzaRepo, pizzaPriceRepo, toppingPriceRepo)

	_, err := uc.Execute(context.Background(), testutil.MustNewID(), cartapp.AddItemRequest{
		PizzaID: pizzaID, SizeID: sizeID, Quantity: 1,
	})

	assert.ErrorIs(t, err, apperr.ErrNotFound)
}

func TestAddItem_ToppingNotFound(t *testing.T) {
	restaurantID := testutil.MustNewID()
	pizzaID := testutil.MustNewID()
	sizeID := testutil.MustNewID()

	cartRepo := &testutil.MockCartRepository{}
	pizzaRepo := &testutil.MockPizzaRepository{
		FindByIDResult: &readmodel.Pizza{ID: pizzaID, RestaurantID: restaurantID},
	}
	pizzaPriceRepo := &testutil.MockPizzaPriceRepository{
		ListByPizzaResult: []readmodel.PizzaPrice{{PizzaID: pizzaID, SizeID: sizeID, IsActive: true}},
	}
	toppingPriceRepo := &testutil.MockToppingPriceRepository{}

	uc := commands.NewAddItem(cartRepo, pizzaRepo, pizzaPriceRepo, toppingPriceRepo)

	_, err := uc.Execute(context.Background(), testutil.MustNewID(), cartapp.AddItemRequest{
		PizzaID: pizzaID, SizeID: sizeID, Quantity: 1, ToppingIDs: []uuid.UUID{testutil.MustNewID()},
	})

	assert.ErrorIs(t, err, apperr.ErrNotFound)
}

func TestAddItem_CreatesCartOnFirstAdd(t *testing.T) {
	restaurantID := testutil.MustNewID()
	pizzaID := testutil.MustNewID()
	sizeID := testutil.MustNewID()
	customerID := testutil.MustNewID()

	cartRepo := &testutil.MockCartRepository{}
	pizzaRepo := &testutil.MockPizzaRepository{
		FindByIDResult: &readmodel.Pizza{ID: pizzaID, RestaurantID: restaurantID},
	}
	pizzaPriceRepo := &testutil.MockPizzaPriceRepository{
		ListByPizzaResult: []readmodel.PizzaPrice{{PizzaID: pizzaID, SizeID: sizeID, IsActive: true}},
	}
	toppingPriceRepo := &testutil.MockToppingPriceRepository{}

	uc := commands.NewAddItem(cartRepo, pizzaRepo, pizzaPriceRepo, toppingPriceRepo)

	res, err := uc.Execute(context.Background(), customerID, cartapp.AddItemRequest{
		PizzaID: pizzaID, SizeID: sizeID, Quantity: 2,
	})

	require.NoError(t, err)
	assert.Equal(t, pizzaID, res.PizzaID)
	assert.Equal(t, int16(2), res.Quantity)

	require.Len(t, cartRepo.Created, 1)
	assert.Equal(t, customerID, cartRepo.Created[0].CustomerID)
	assert.Equal(t, restaurantID, cartRepo.Created[0].RestaurantID)

	require.Len(t, cartRepo.AddOrMergeItemCalls, 1)
	call := cartRepo.AddOrMergeItemCalls[0]
	assert.Equal(t, cartRepo.Created[0].ID, call.CartID)
	assert.Equal(t, pizzaID, call.Item.PizzaID)
	assert.Equal(t, int16(2), call.Item.Quantity)
}

func TestAddItem_RestaurantMismatch(t *testing.T) {
	existingRestaurantID := testutil.MustNewID()
	newRestaurantID := testutil.MustNewID()
	pizzaID := testutil.MustNewID()
	sizeID := testutil.MustNewID()

	cartRepo := &testutil.MockCartRepository{
		FindByCustomerResult: &cart.Cart{ID: testutil.MustNewID(), RestaurantID: existingRestaurantID},
	}
	pizzaRepo := &testutil.MockPizzaRepository{
		FindByIDResult: &readmodel.Pizza{ID: pizzaID, RestaurantID: newRestaurantID},
	}
	pizzaPriceRepo := &testutil.MockPizzaPriceRepository{
		ListByPizzaResult: []readmodel.PizzaPrice{{PizzaID: pizzaID, SizeID: sizeID, IsActive: true}},
	}
	toppingPriceRepo := &testutil.MockToppingPriceRepository{}

	uc := commands.NewAddItem(cartRepo, pizzaRepo, pizzaPriceRepo, toppingPriceRepo)

	_, err := uc.Execute(context.Background(), testutil.MustNewID(), cartapp.AddItemRequest{
		PizzaID: pizzaID, SizeID: sizeID, Quantity: 1,
	})

	assert.ErrorIs(t, err, cart.ErrCartRestaurantMismatch)
	assert.Empty(t, cartRepo.AddOrMergeItemCalls)
}

func TestAddItem_SortsToppingIDs(t *testing.T) {
	restaurantID := testutil.MustNewID()
	pizzaID := testutil.MustNewID()
	sizeID := testutil.MustNewID()
	toppingA := testutil.MustNewID()
	toppingB := testutil.MustNewID()

	cartRepo := &testutil.MockCartRepository{
		FindByCustomerResult: &cart.Cart{ID: testutil.MustNewID(), RestaurantID: restaurantID},
	}
	pizzaRepo := &testutil.MockPizzaRepository{
		FindByIDResult: &readmodel.Pizza{ID: pizzaID, RestaurantID: restaurantID},
	}
	pizzaPriceRepo := &testutil.MockPizzaPriceRepository{
		ListByPizzaResult: []readmodel.PizzaPrice{{PizzaID: pizzaID, SizeID: sizeID, IsActive: true}},
	}
	toppingPriceRepo := &testutil.MockToppingPriceRepository{
		ListByRestaurantResult: []readmodel.ToppingPrice{
			{RestaurantID: restaurantID, ToppingID: toppingA},
			{RestaurantID: restaurantID, ToppingID: toppingB},
		},
	}

	uc := commands.NewAddItem(cartRepo, pizzaRepo, pizzaPriceRepo, toppingPriceRepo)

	res, err := uc.Execute(context.Background(), testutil.MustNewID(), cartapp.AddItemRequest{
		PizzaID: pizzaID, SizeID: sizeID, Quantity: 1, ToppingIDs: []uuid.UUID{toppingB, toppingA},
	})

	require.NoError(t, err)
	require.Len(t, res.ToppingIDs, 2)
	assert.True(t, res.ToppingIDs[0].String() < res.ToppingIDs[1].String(), "topping ids must be sorted")

	require.Len(t, cartRepo.AddOrMergeItemCalls, 1)

	got := cartRepo.AddOrMergeItemCalls[0].Item.ToppingIDs
	require.Len(t, got, 2)
	assert.True(t, got[0].String() < got[1].String(), "topping ids must be stored sorted")
}
