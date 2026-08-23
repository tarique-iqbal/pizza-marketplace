package commands_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	pizzaapp "restaurant-service/internal/application/pizza"
	"restaurant-service/internal/application/pizza/commands"
	resapp "restaurant-service/internal/application/restaurant"
	"restaurant-service/internal/domain/pizza"
	"restaurant-service/internal/domain/restaurant"
	"restaurant-service/internal/domain/topping"
	"restaurant-service/internal/infrastructure/persistence"
	apperr "restaurant-service/internal/shared/errors"
	"restaurant-service/internal/shared/event"
	"restaurant-service/tests/infrastructure/db/fixtures"
	"restaurant-service/tests/testutil"
)

type fakePublisher struct {
	events []event.Event
}

func (f *fakePublisher) PublishEvent(ctx context.Context, e event.Event) error {
	f.events = append(f.events, e)
	return nil
}

type createPizzaSetup struct {
	DB          *gorm.DB
	CreatePizza *commands.CreatePizza
	Publisher   *fakePublisher
}

func setupCreatePizza(t *testing.T) createPizzaSetup {
	db := testutil.DB(t)
	db.TruncateTables(t, testutil.TableRestaurant)

	require.NoError(t, fixtures.LoadRestaurantFixtures(t, db.DB))

	restaurantRepo := persistence.NewRestaurantRepository(db.DB)
	pizzaRepo := persistence.NewPizzaRepository(db.DB)
	toppingRepo := persistence.NewToppingRepository(db.DB)
	publisher := &fakePublisher{}

	return createPizzaSetup{
		DB:          db.DB,
		CreatePizza: commands.NewCreatePizza(restaurantRepo, pizzaRepo, toppingRepo, publisher),
		Publisher:   publisher,
	}
}

func restaurantByName(t *testing.T, db *gorm.DB, name string) restaurant.Restaurant {
	var r restaurant.Restaurant

	err := db.Where("name = ?", name).Take(&r).Error
	require.NoError(t, err)

	return r
}

func validCreatePizzaInput() pizzaapp.CreatePizzaRequest {
	return pizzaapp.CreatePizzaRequest{
		Name: "Margherita",
	}
}

func TestCreatePizza_Success(t *testing.T) {
	env := setupCreatePizza(t)

	res := restaurantByName(t, env.DB, "Anatolische Küche")

	output, err := env.CreatePizza.Execute(context.Background(), res.ID, res.OwnerID, validCreatePizzaInput())
	require.NoError(t, err)

	assert.Equal(t, "Margherita", output.Name)
	assert.Equal(t, pizza.PizzaAvailable, output.Status)
	assert.False(t, output.IsVegetarian)
	assert.Empty(t, output.Prices)
	assert.Empty(t, output.Toppings)

	var stored pizza.Pizza
	require.NoError(t, env.DB.Take(&stored, "id = ?", output.ID).Error)
	assert.Equal(t, res.ID, stored.RestaurantID)

	assert.Empty(t, env.Publisher.events, "no pizza.updated event while restaurant is still draft")
}

func TestCreatePizza_PublishesPizzaUpdatedEvent_WhenActive(t *testing.T) {
	env := setupCreatePizza(t)

	res := restaurantByName(t, env.DB, "Anatolische Küche")
	res.Status = restaurant.StatusActive
	require.NoError(t, env.DB.Save(&res).Error)

	output, err := env.CreatePizza.Execute(context.Background(), res.ID, res.OwnerID, validCreatePizzaInput())
	require.NoError(t, err)

	require.Len(t, env.Publisher.events, 1)

	payload, ok := env.Publisher.events[0].(resapp.PizzaUpdatedPayload)
	require.True(t, ok)

	assert.Equal(t, "restaurant.pizza_updated", payload.EventName)
	assert.Equal(t, res.ID, payload.RestaurantID)
	assert.Equal(t, output.ID, payload.Pizza.ID)
	assert.Equal(t, "Margherita", payload.Pizza.Name)
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

	var stored pizza.Pizza
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
	assert.ErrorIs(t, err, pizza.ErrDuplicateTopping)
}
