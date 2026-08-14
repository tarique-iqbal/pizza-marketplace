package commands_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	pizzaapp "restaurant-service/internal/application/pizza"
	"restaurant-service/internal/application/pizza/commands"
	"restaurant-service/internal/domain/pizza"
	"restaurant-service/internal/domain/restaurant"
	"restaurant-service/internal/domain/topping"
	"restaurant-service/internal/infrastructure/persistence"
	apperr "restaurant-service/internal/shared/errors"
	"restaurant-service/tests/infrastructure/db/fixtures"
	"restaurant-service/tests/testutil"
)

type updatePizzaSetup struct {
	DB          *gorm.DB
	UpdatePizza *commands.UpdatePizza
}

func setupUpdatePizza(t *testing.T) updatePizzaSetup {
	db := testutil.DB(t)
	db.TruncateTables(t, testutil.TableRestaurant)

	require.NoError(t, fixtures.LoadRestaurantFixtures(t, db.DB))
	require.NoError(t, fixtures.LoadPizzaFixtures(t, db.DB))

	restaurantRepo := persistence.NewRestaurantRepository(db.DB)
	pizzaRepo := persistence.NewPizzaRepository(db.DB)
	pizzaPriceRepo := persistence.NewPizzaPriceRepository(db.DB)
	pizzaSizeRepo := persistence.NewPizzaSizeRepository(db.DB)
	toppingRepo := persistence.NewToppingRepository(db.DB)

	return updatePizzaSetup{
		DB:          db.DB,
		UpdatePizza: commands.NewUpdatePizza(restaurantRepo, pizzaRepo, pizzaPriceRepo, pizzaSizeRepo, toppingRepo),
	}
}

func TestUpdatePizza_Success(t *testing.T) {
	env := setupUpdatePizza(t)

	pz := firstPizza(t, env.DB)

	status := "unavailable"
	input := pizzaapp.UpdatePizzaRequest{
		Name:      "Margherita Deluxe",
		Status:    &status,
		SortOrder: 5,
	}

	output, err := env.UpdatePizza.Execute(
		context.Background(),
		pz.RestaurantID,
		pz.ID,
		restaurantByID(t, env.DB, pz.RestaurantID).OwnerID,
		input,
	)
	require.NoError(t, err)

	assert.Equal(t, "Margherita Deluxe", output.Name)
	assert.Equal(t, pizza.PizzaUnavailable, output.Status)
	assert.Equal(t, 5, output.SortOrder)

	var updated pizza.Pizza
	require.NoError(t, env.DB.Take(&updated, "id = ?", pz.ID).Error)
	assert.NotNil(t, updated.UpdatedAt)
}

func TestUpdatePizza_NilFields_KeepExistingValues(t *testing.T) {
	env := setupUpdatePizza(t)

	pz := firstPizza(t, env.DB)
	require.True(t, pz.IsVegetarian, "fixture assumption")

	input := pizzaapp.UpdatePizzaRequest{
		Name: pz.Name,
		// IsVegetarian and Status omitted (nil)
	}

	output, err := env.UpdatePizza.Execute(
		context.Background(),
		pz.RestaurantID,
		pz.ID,
		restaurantByID(t, env.DB, pz.RestaurantID).OwnerID,
		input,
	)
	require.NoError(t, err)

	assert.True(t, output.IsVegetarian, "nil IsVegetarian must preserve the existing value, not reset to false")
	assert.Equal(t, pizza.PizzaAvailable, output.Status)
}

func TestUpdatePizza_PizzaNotFound(t *testing.T) {
	env := setupUpdatePizza(t)

	pz := firstPizza(t, env.DB)
	owner := restaurantByID(t, env.DB, pz.RestaurantID)

	_, err := env.UpdatePizza.Execute(
		context.Background(),
		pz.RestaurantID,
		uuid.New(),
		owner.OwnerID,
		pizzaapp.UpdatePizzaRequest{Name: "x"},
	)
	require.Error(t, err)

	assert.ErrorIs(t, err, apperr.ErrNotFound)
}

func TestUpdatePizza_RestaurantNotOwned(t *testing.T) {
	env := setupUpdatePizza(t)

	pz := firstPizza(t, env.DB)

	_, err := env.UpdatePizza.Execute(
		context.Background(),
		pz.RestaurantID,
		pz.ID,
		uuid.New(),
		pizzaapp.UpdatePizzaRequest{Name: "x"},
	)
	require.Error(t, err)

	assert.ErrorIs(t, err, apperr.ErrForbidden)
}

func TestUpdatePizza_ToppingIDsNil_LeavesToppingsUntouched(t *testing.T) {
	env := setupUpdatePizza(t)

	pz := firstPizza(t, env.DB)
	owner := restaurantByID(t, env.DB, pz.RestaurantID)

	var t1 topping.Topping
	require.NoError(t, env.DB.Order("name").Take(&t1).Error)
	require.NoError(t, pz.SetToppingIDs([]uuid.UUID{t1.ID}))
	require.NoError(t, env.DB.Save(&pz).Error)

	output, err := env.UpdatePizza.Execute(
		context.Background(),
		pz.RestaurantID,
		pz.ID,
		owner.OwnerID,
		pizzaapp.UpdatePizzaRequest{Name: "Renamed"},
	)
	require.NoError(t, err)

	require.Len(t, output.Toppings, 1, "response must report the pizza's real toppings, not fake them as empty")
	assert.Equal(t, t1.ID, output.Toppings[0].ToppingID)

	var stored pizza.Pizza
	require.NoError(t, env.DB.Take(&stored, "id = ?", pz.ID).Error)
	storedIDs, err := stored.ToppingIDs()
	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{t1.ID}, storedIDs, "omitted toppingIds must not touch existing toppings")
}

func TestUpdatePizza_ReportsExistingPrices(t *testing.T) {
	env := setupUpdatePizza(t)

	pz := firstPizza(t, env.DB)
	owner := restaurantByID(t, env.DB, pz.RestaurantID)

	var size pizza.PizzaSize
	require.NoError(t, env.DB.Order("diameter_cm").Take(&size).Error)

	pizzaPriceRepo := persistence.NewPizzaPriceRepository(env.DB)
	price, err := pizza.NewPizzaPrice(pz.ID, size.ID, decimal.RequireFromString("9.50"))
	require.NoError(t, err)
	require.NoError(t, pizzaPriceRepo.ReplacePrices(context.Background(), pz.ID, []pizza.PizzaPrice{*price}))

	output, err := env.UpdatePizza.Execute(
		context.Background(),
		pz.RestaurantID,
		pz.ID,
		owner.OwnerID,
		pizzaapp.UpdatePizzaRequest{Name: "Renamed"},
	)
	require.NoError(t, err)

	require.Len(t, output.Prices, 1, "response must report the pizza's real prices, not fake them as empty")
	assert.Equal(t, size.ID, output.Prices[0].SizeID)
}

func TestUpdatePizza_ToppingIDsEmpty_ClearsToppings(t *testing.T) {
	env := setupUpdatePizza(t)

	pz := firstPizza(t, env.DB)
	owner := restaurantByID(t, env.DB, pz.RestaurantID)

	var t1 topping.Topping
	require.NoError(t, env.DB.Order("name").Take(&t1).Error)
	require.NoError(t, pz.SetToppingIDs([]uuid.UUID{t1.ID}))
	require.NoError(t, env.DB.Save(&pz).Error)

	output, err := env.UpdatePizza.Execute(
		context.Background(),
		pz.RestaurantID,
		pz.ID,
		owner.OwnerID,
		pizzaapp.UpdatePizzaRequest{
			Name:       "Renamed",
			ToppingIDs: []uuid.UUID{},
		},
	)
	require.NoError(t, err)

	assert.Empty(t, output.Toppings)
}

func TestUpdatePizza_ToppingIDs_SetsDefaults_NoPriceRequired(t *testing.T) {
	env := setupUpdatePizza(t)

	pz := firstPizza(t, env.DB)
	owner := restaurantByID(t, env.DB, pz.RestaurantID)

	var t1 topping.Topping
	require.NoError(t, env.DB.Order("name").Take(&t1).Error)

	output, err := env.UpdatePizza.Execute(
		context.Background(),
		pz.RestaurantID,
		pz.ID,
		owner.OwnerID,
		pizzaapp.UpdatePizzaRequest{
			Name:       "Renamed",
			ToppingIDs: []uuid.UUID{t1.ID},
		},
	)
	require.NoError(t, err, "a default topping must not require topping_prices to already exist")

	require.Len(t, output.Toppings, 1)
	assert.Equal(t, t1.ID, output.Toppings[0].ToppingID)
}

func TestUpdatePizza_ToppingIDs_ToppingDoesNotExist(t *testing.T) {
	env := setupUpdatePizza(t)

	pz := firstPizza(t, env.DB)
	owner := restaurantByID(t, env.DB, pz.RestaurantID)

	_, err := env.UpdatePizza.Execute(
		context.Background(),
		pz.RestaurantID,
		pz.ID,
		owner.OwnerID,
		pizzaapp.UpdatePizzaRequest{
			Name:       "Renamed",
			ToppingIDs: []uuid.UUID{uuid.New()},
		},
	)
	require.Error(t, err)

	assert.ErrorIs(t, err, apperr.ErrNotFound)
}

func TestUpdatePizza_ToppingIDs_Duplicate(t *testing.T) {
	env := setupUpdatePizza(t)

	pz := firstPizza(t, env.DB)
	owner := restaurantByID(t, env.DB, pz.RestaurantID)

	var t1 topping.Topping
	require.NoError(t, env.DB.Order("name").Take(&t1).Error)

	_, err := env.UpdatePizza.Execute(
		context.Background(),
		pz.RestaurantID,
		pz.ID,
		owner.OwnerID,
		pizzaapp.UpdatePizzaRequest{
			Name:       "Renamed",
			ToppingIDs: []uuid.UUID{t1.ID, t1.ID},
		},
	)
	require.Error(t, err)

	assert.ErrorIs(t, err, apperr.ErrConflict)
	assert.ErrorIs(t, err, pizza.ErrDuplicateTopping)
}

func restaurantByID(t *testing.T, db *gorm.DB, id uuid.UUID) restaurant.Restaurant {
	var r restaurant.Restaurant

	err := db.Take(&r, "id = ?", id).Error
	require.NoError(t, err)

	return r
}

func firstPizza(t *testing.T, db *gorm.DB) pizza.Pizza {
	var p pizza.Pizza

	err := db.Order("sort_order").First(&p).Error
	require.NoError(t, err)

	return p
}
