package commands_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	resapp "restaurant-service/internal/application/restaurant"
	"restaurant-service/internal/application/restaurant/commands"
	"restaurant-service/internal/domain/restaurant"
	"restaurant-service/internal/domain/topping"
	"restaurant-service/internal/infrastructure/persistence"
	apperr "restaurant-service/internal/shared/errors"
	"restaurant-service/tests/infrastructure/db/fixtures"
	"restaurant-service/tests/testutil"
)

type setPizzaPricesSetup struct {
	DB             *gorm.DB
	SetPizzaPrices *commands.SetPizzaPrices
}

func setupSetPizzaPrices(t *testing.T) setPizzaPricesSetup {
	db := testutil.DB(t)
	db.TruncateTables(t, testutil.TableRestaurant)

	require.NoError(t, fixtures.LoadRestaurantFixtures(t, db.DB))
	require.NoError(t, fixtures.LoadPizzaFixtures(t, db.DB))

	restaurantRepo := persistence.NewRestaurantRepository(db.DB)
	pizzaRepo := persistence.NewPizzaRepository(db.DB)
	pizzaPriceRepo := persistence.NewPizzaPriceRepository(db.DB)
	pizzaSizeRepo := persistence.NewPizzaSizeRepository(db.DB)
	toppingRepo := persistence.NewToppingRepository(db.DB)

	return setPizzaPricesSetup{
		DB:             db.DB,
		SetPizzaPrices: commands.NewSetPizzaPrices(restaurantRepo, pizzaRepo, pizzaPriceRepo, pizzaSizeRepo, toppingRepo),
	}
}

func TestSetPizzaPrices_Success(t *testing.T) {
	env := setupSetPizzaPrices(t)

	pizza := firstPizza(t, env.DB)
	owner := restaurantByID(t, env.DB, pizza.RestaurantID)

	var sizes []restaurant.PizzaSize
	require.NoError(t, env.DB.Order("diameter_cm").Limit(2).Find(&sizes).Error)

	input := resapp.SetPizzaPricesRequest{
		Prices: []resapp.PizzaPriceInput{
			{SizeID: sizes[0].ID, Price: decimal.RequireFromString("9.50")},
			{SizeID: sizes[1].ID, Price: decimal.RequireFromString("12.00")},
		},
	}

	output, err := env.SetPizzaPrices.Execute(context.Background(), pizza.RestaurantID, pizza.ID, owner.OwnerID, input)
	require.NoError(t, err)

	require.Len(t, output.Prices, 2)
	for _, p := range output.Prices {
		assert.True(t, p.IsActive)
		assert.NotZero(t, p.DiameterCm)
	}
}

func TestSetPizzaPrices_ReportsExistingToppings(t *testing.T) {
	env := setupSetPizzaPrices(t)

	pizza := firstPizza(t, env.DB)
	owner := restaurantByID(t, env.DB, pizza.RestaurantID)

	var t1 topping.Topping
	require.NoError(t, env.DB.Order("name").Take(&t1).Error)
	require.NoError(t, pizza.SetToppingIDs([]uuid.UUID{t1.ID}))
	require.NoError(t, env.DB.Save(&pizza).Error)

	var size restaurant.PizzaSize
	require.NoError(t, env.DB.Order("diameter_cm").Take(&size).Error)

	input := resapp.SetPizzaPricesRequest{
		Prices: []resapp.PizzaPriceInput{{SizeID: size.ID, Price: decimal.RequireFromString("9.50")}},
	}

	output, err := env.SetPizzaPrices.Execute(context.Background(), pizza.RestaurantID, pizza.ID, owner.OwnerID, input)
	require.NoError(t, err)

	require.Len(t, output.Toppings, 1, "must report the pizza's real toppings, not fake them as empty")
	assert.Equal(t, t1.ID, output.Toppings[0].ToppingID)
}

func TestSetPizzaPrices_DuplicateSizeInRequest(t *testing.T) {
	env := setupSetPizzaPrices(t)

	pizza := firstPizza(t, env.DB)
	owner := restaurantByID(t, env.DB, pizza.RestaurantID)

	var size restaurant.PizzaSize
	require.NoError(t, env.DB.Order("diameter_cm").Take(&size).Error)

	input := resapp.SetPizzaPricesRequest{
		Prices: []resapp.PizzaPriceInput{
			{SizeID: size.ID, Price: decimal.RequireFromString("9.50")},
			{SizeID: size.ID, Price: decimal.RequireFromString("10.00")},
		},
	}

	_, err := env.SetPizzaPrices.Execute(context.Background(), pizza.RestaurantID, pizza.ID, owner.OwnerID, input)
	require.Error(t, err)

	assert.ErrorIs(t, err, apperr.ErrConflict)
}

func TestSetPizzaPrices_SizeDoesNotExist(t *testing.T) {
	env := setupSetPizzaPrices(t)

	pizza := firstPizza(t, env.DB)
	owner := restaurantByID(t, env.DB, pizza.RestaurantID)

	input := resapp.SetPizzaPricesRequest{
		Prices: []resapp.PizzaPriceInput{
			{SizeID: uuid.New(), Price: decimal.RequireFromString("9.50")},
		},
	}

	_, err := env.SetPizzaPrices.Execute(context.Background(), pizza.RestaurantID, pizza.ID, owner.OwnerID, input)
	require.Error(t, err)

	assert.ErrorIs(t, err, apperr.ErrNotFound)
}

func TestSetPizzaPrices_PizzaNotFound(t *testing.T) {
	env := setupSetPizzaPrices(t)

	pizza := firstPizza(t, env.DB)
	owner := restaurantByID(t, env.DB, pizza.RestaurantID)

	var size restaurant.PizzaSize
	require.NoError(t, env.DB.Order("diameter_cm").Take(&size).Error)

	input := resapp.SetPizzaPricesRequest{
		Prices: []resapp.PizzaPriceInput{{SizeID: size.ID, Price: decimal.RequireFromString("9.50")}},
	}

	_, err := env.SetPizzaPrices.Execute(context.Background(), pizza.RestaurantID, uuid.New(), owner.OwnerID, input)
	require.Error(t, err)

	assert.ErrorIs(t, err, apperr.ErrNotFound)
}
