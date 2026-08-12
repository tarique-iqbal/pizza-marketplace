package commands_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
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

type createPizzaSetup struct {
	DB          *gorm.DB
	CreatePizza *commands.CreatePizza
}

func setupCreatePizza(t *testing.T) createPizzaSetup {
	db := testutil.DB(t)
	db.TruncateTables(t, testutil.TableRestaurant)

	require.NoError(t, fixtures.LoadRestaurantFixtures(t, db.DB))

	restaurantRepo := persistence.NewRestaurantRepository(db.DB)
	pizzaRepo := persistence.NewPizzaRepository(db.DB)
	toppingRepo := persistence.NewToppingRepository(db.DB)

	return createPizzaSetup{
		DB:          db.DB,
		CreatePizza: commands.NewCreatePizza(restaurantRepo, pizzaRepo, toppingRepo),
	}
}

func restaurantByName(t *testing.T, db *gorm.DB, name string) restaurant.Restaurant {
	var r restaurant.Restaurant

	err := db.Where("name = ?", name).Take(&r).Error
	require.NoError(t, err)

	return r
}

func validCreatePizzaInput() resapp.CreatePizzaRequest {
	return resapp.CreatePizzaRequest{
		Name: "Margherita",
	}
}

func TestCreatePizza_Success(t *testing.T) {
	env := setupCreatePizza(t)

	res := restaurantByName(t, env.DB, "Anatolische Küche")

	output, err := env.CreatePizza.Execute(context.Background(), res.ID, res.OwnerID, validCreatePizzaInput())
	require.NoError(t, err)

	assert.Equal(t, "Margherita", output.Name)
	assert.Equal(t, restaurant.PizzaAvailable, output.Status)
	assert.False(t, output.IsVegetarian)
	assert.Empty(t, output.Prices)
	assert.Empty(t, output.Toppings)

	var stored restaurant.Pizza
	require.NoError(t, env.DB.Take(&stored, "id = ?", output.ID).Error)
	assert.Equal(t, res.ID, stored.RestaurantID)
}

func TestCreatePizza_ChecklistIncomplete(t *testing.T) {
	env := setupCreatePizza(t)

	res := restaurantByName(t, env.DB, "Pizza Paradise") // only `basic` complete

	_, err := env.CreatePizza.Execute(context.Background(), res.ID, res.OwnerID, validCreatePizzaInput())
	require.Error(t, err)

	assert.ErrorIs(t, err, restaurant.ErrChecklistIncomplete)
	assert.ErrorIs(t, err, apperr.ErrForbidden)
}

func TestCreatePizza_RestaurantNotOwned(t *testing.T) {
	env := setupCreatePizza(t)

	res := restaurantByName(t, env.DB, "Anatolische Küche")

	_, err := env.CreatePizza.Execute(context.Background(), res.ID, uuid.New(), validCreatePizzaInput())
	require.Error(t, err)

	assert.ErrorIs(t, err, apperr.ErrForbidden)
}

func TestCreatePizza_RestaurantNotFound(t *testing.T) {
	env := setupCreatePizza(t)

	_, err := env.CreatePizza.Execute(context.Background(), uuid.New(), uuid.New(), validCreatePizzaInput())
	require.Error(t, err)

	assert.ErrorIs(t, err, apperr.ErrForbidden)
}

func TestCreatePizza_WithToppings_NoPriceRequired_Success(t *testing.T) {
	env := setupCreatePizza(t)

	res := restaurantByName(t, env.DB, "Anatolische Küche")

	var t1 topping.Topping
	require.NoError(t, env.DB.Order("name").Take(&t1).Error)

	input := validCreatePizzaInput()
	input.ToppingIDs = []uuid.UUID{t1.ID}

	output, err := env.CreatePizza.Execute(context.Background(), res.ID, res.OwnerID, input)
	require.NoError(t, err, "a default topping must not require topping_prices to already exist")

	require.Len(t, output.Toppings, 1)
	assert.Equal(t, t1.ID, output.Toppings[0].ToppingID)

	var stored restaurant.Pizza
	require.NoError(t, env.DB.Take(&stored, "id = ?", output.ID).Error)
	storedIDs, err := stored.ToppingIDs()
	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{t1.ID}, storedIDs)
}

func TestCreatePizza_ToppingDoesNotExist(t *testing.T) {
	env := setupCreatePizza(t)

	res := restaurantByName(t, env.DB, "Anatolische Küche")

	input := validCreatePizzaInput()
	input.ToppingIDs = []uuid.UUID{uuid.New()}

	_, err := env.CreatePizza.Execute(context.Background(), res.ID, res.OwnerID, input)
	require.Error(t, err)

	assert.ErrorIs(t, err, apperr.ErrNotFound)
}

func TestCreatePizza_DuplicateTopping(t *testing.T) {
	env := setupCreatePizza(t)

	res := restaurantByName(t, env.DB, "Anatolische Küche")

	var t1 topping.Topping
	require.NoError(t, env.DB.Order("name").Take(&t1).Error)

	input := validCreatePizzaInput()
	input.ToppingIDs = []uuid.UUID{t1.ID, t1.ID}

	_, err := env.CreatePizza.Execute(context.Background(), res.ID, res.OwnerID, input)
	require.Error(t, err)

	assert.ErrorIs(t, err, apperr.ErrConflict)
	assert.ErrorIs(t, err, restaurant.ErrDuplicateTopping)
}
