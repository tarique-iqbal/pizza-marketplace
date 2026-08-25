package commands_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
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
	"restaurant-service/tests/infrastructure/db/fixtures"
	"restaurant-service/tests/testutil"
)

type setPizzaPricesSetup struct {
	DB             *gorm.DB
	SetPizzaPrices *commands.SetPizzaPrices
	Publisher      *fakePublisher
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
	publisher := &fakePublisher{}

	return setPizzaPricesSetup{
		DB: db.DB,
		SetPizzaPrices: commands.NewSetPizzaPrices(
			restaurantRepo, pizzaRepo, pizzaPriceRepo, pizzaSizeRepo, toppingRepo, publisher,
		),
		Publisher: publisher,
	}
}

func TestSetPizzaPrices_Success(t *testing.T) {
	env := setupSetPizzaPrices(t)

	pz := firstPizza(t, env.DB)
	owner := restaurantByID(t, env.DB, pz.RestaurantID)

	var sizes []pizza.PizzaSize
	require.NoError(t, env.DB.Order("diameter_cm").Limit(2).Find(&sizes).Error)

	input := pizzaapp.SetPizzaPricesRequest{
		Prices: []pizzaapp.PizzaPriceInput{
			{SizeID: sizes[0].ID, Price: decimal.RequireFromString("9.50")},
			{SizeID: sizes[1].ID, Price: decimal.RequireFromString("12.00")},
		},
	}

	output, err := env.SetPizzaPrices.Execute(context.Background(), pz.RestaurantID, pz.ID, owner.OwnerID, input)
	require.NoError(t, err)

	require.Len(t, output.Prices, 2)

	byPrice := make(map[uuid.UUID]pizzaapp.PizzaPriceResponse, len(output.Prices))
	for _, p := range output.Prices {
		assert.True(t, p.IsActive)
		assert.NotZero(t, p.DiameterCm)
		byPrice[p.SizeID] = p
	}

	assert.True(t, decimal.RequireFromString("9.50").Equal(decimal.Decimal(byPrice[sizes[0].ID].Price)))
	assert.True(t, decimal.RequireFromString("12.00").Equal(decimal.Decimal(byPrice[sizes[1].ID].Price)))

	assert.Empty(t, env.Publisher.events, "no pizza.updated event while restaurant is still draft")
}

func TestSetPizzaPrices_PublishesPizzaUpdatedEvent_WhenActive(t *testing.T) {
	env := setupSetPizzaPrices(t)

	pz := firstPizza(t, env.DB)
	owner := restaurantByID(t, env.DB, pz.RestaurantID)
	owner.Status = restaurant.StatusActive
	require.NoError(t, env.DB.Save(&owner).Error)

	var size pizza.PizzaSize
	require.NoError(t, env.DB.Order("diameter_cm").Take(&size).Error)

	input := pizzaapp.SetPizzaPricesRequest{
		Prices: []pizzaapp.PizzaPriceInput{{SizeID: size.ID, Price: decimal.RequireFromString("9.50")}},
	}

	output, err := env.SetPizzaPrices.Execute(context.Background(), pz.RestaurantID, pz.ID, owner.OwnerID, input)
	require.NoError(t, err)

	require.Len(t, env.Publisher.events, 1)

	payload, ok := env.Publisher.events[0].(resapp.PizzaUpdatedPayload)
	require.True(t, ok)

	assert.Equal(t, "restaurant.pizza_updated", payload.EventName)
	assert.Equal(t, owner.ID, payload.RestaurantID)
	assert.Equal(t, output.ID, payload.Pizza.ID)
	require.Len(t, payload.Pizza.Prices, 1)

	require.NotNil(t, output.UpdatedAt)
	assert.Equal(t, *output.UpdatedAt, payload.UpdatedAt)

	var stored pizza.Pizza
	require.NoError(t, env.DB.Take(&stored, "id = ?", pz.ID).Error)
	require.NotNil(t, stored.UpdatedAt)
	assert.WithinDuration(t, time.Now(), *stored.UpdatedAt, 5*time.Second)
}

func TestSetPizzaPrices_ReportsExistingToppings(t *testing.T) {
	env := setupSetPizzaPrices(t)

	pz := firstPizza(t, env.DB)
	owner := restaurantByID(t, env.DB, pz.RestaurantID)

	var t1 topping.Topping
	require.NoError(t, env.DB.Order("name").Take(&t1).Error)
	require.NoError(t, pz.SetToppingIDs([]uuid.UUID{t1.ID}))
	require.NoError(t, env.DB.Save(&pz).Error)

	var size pizza.PizzaSize
	require.NoError(t, env.DB.Order("diameter_cm").Take(&size).Error)

	input := pizzaapp.SetPizzaPricesRequest{
		Prices: []pizzaapp.PizzaPriceInput{{SizeID: size.ID, Price: decimal.RequireFromString("9.50")}},
	}

	output, err := env.SetPizzaPrices.Execute(context.Background(), pz.RestaurantID, pz.ID, owner.OwnerID, input)
	require.NoError(t, err)

	require.Len(t, output.Toppings, 1, "must report the pizza's real toppings, not fake them as empty")
	assert.Equal(t, t1.ID, output.Toppings[0].ToppingID)
}

func TestSetPizzaPrices_DuplicateSizeInRequest(t *testing.T) {
	env := setupSetPizzaPrices(t)

	pz := firstPizza(t, env.DB)
	owner := restaurantByID(t, env.DB, pz.RestaurantID)

	var size pizza.PizzaSize
	require.NoError(t, env.DB.Order("diameter_cm").Take(&size).Error)

	input := pizzaapp.SetPizzaPricesRequest{
		Prices: []pizzaapp.PizzaPriceInput{
			{SizeID: size.ID, Price: decimal.RequireFromString("9.50")},
			{SizeID: size.ID, Price: decimal.RequireFromString("10.00")},
		},
	}

	_, err := env.SetPizzaPrices.Execute(context.Background(), pz.RestaurantID, pz.ID, owner.OwnerID, input)
	require.Error(t, err)

	assert.ErrorIs(t, err, apperr.ErrConflict)
}

func TestSetPizzaPrices_SizeDoesNotExist(t *testing.T) {
	env := setupSetPizzaPrices(t)

	pz := firstPizza(t, env.DB)
	owner := restaurantByID(t, env.DB, pz.RestaurantID)

	input := pizzaapp.SetPizzaPricesRequest{
		Prices: []pizzaapp.PizzaPriceInput{
			{SizeID: uuid.New(), Price: decimal.RequireFromString("9.50")},
		},
	}

	_, err := env.SetPizzaPrices.Execute(context.Background(), pz.RestaurantID, pz.ID, owner.OwnerID, input)
	require.Error(t, err)

	assert.ErrorIs(t, err, apperr.ErrNotFound)
}

func TestSetPizzaPrices_PizzaNotFound(t *testing.T) {
	env := setupSetPizzaPrices(t)

	pz := firstPizza(t, env.DB)
	owner := restaurantByID(t, env.DB, pz.RestaurantID)

	var size pizza.PizzaSize
	require.NoError(t, env.DB.Order("diameter_cm").Take(&size).Error)

	input := pizzaapp.SetPizzaPricesRequest{
		Prices: []pizzaapp.PizzaPriceInput{{SizeID: size.ID, Price: decimal.RequireFromString("9.50")}},
	}

	_, err := env.SetPizzaPrices.Execute(context.Background(), pz.RestaurantID, uuid.New(), owner.OwnerID, input)
	require.Error(t, err)

	assert.ErrorIs(t, err, apperr.ErrNotFound)
}
