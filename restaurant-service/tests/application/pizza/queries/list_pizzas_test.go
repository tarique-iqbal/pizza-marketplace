package queries_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	pizzaapp "restaurant-service/internal/application/pizza"
	"restaurant-service/internal/application/pizza/queries"
	toppingapp "restaurant-service/internal/application/topping"
	"restaurant-service/internal/domain/pizza"
	"restaurant-service/internal/domain/restaurant"
	"restaurant-service/internal/domain/topping"
	"restaurant-service/internal/infrastructure/persistence"
	apperr "restaurant-service/internal/shared/errors"
	"restaurant-service/tests/infrastructure/db/fixtures"
	"restaurant-service/tests/testutil"
)

type listPizzasSetup struct {
	DB         *gorm.DB
	ListPizzas *queries.ListPizzas
}

func setupListPizzas(t *testing.T) listPizzasSetup {
	db := testutil.DB(t)
	db.TruncateTables(t, testutil.TableRestaurant)

	require.NoError(t, fixtures.LoadRestaurantFixtures(t, db.DB))
	require.NoError(t, fixtures.LoadPizzaFixtures(t, db.DB))

	restaurantRepo := persistence.NewRestaurantRepository(db.DB)
	pizzaRepo := persistence.NewPizzaRepository(db.DB)
	pizzaPriceRepo := persistence.NewPizzaPriceRepository(db.DB)
	pizzaSizeRepo := persistence.NewPizzaSizeRepository(db.DB)
	toppingRepo := persistence.NewToppingRepository(db.DB)
	toppingPriceRepo := persistence.NewToppingPriceRepository(db.DB)

	return listPizzasSetup{
		DB: db.DB,
		ListPizzas: queries.NewListPizzas(
			restaurantRepo, pizzaRepo, pizzaPriceRepo, pizzaSizeRepo, toppingRepo, toppingPriceRepo,
		),
	}
}

func firstPizza(t *testing.T, db *gorm.DB) pizza.Pizza {
	var p pizza.Pizza

	err := db.Order("sort_order").First(&p).Error
	require.NoError(t, err)

	return p
}

func restaurantByID(t *testing.T, db *gorm.DB, id uuid.UUID) restaurant.Restaurant {
	var r restaurant.Restaurant

	err := db.Take(&r, "id = ?", id).Error
	require.NoError(t, err)

	return r
}

func TestListPizzas_Success(t *testing.T) {
	env := setupListPizzas(t)

	pz := firstPizza(t, env.DB)
	owner := restaurantByID(t, env.DB, pz.RestaurantID)

	output, err := env.ListPizzas.Execute(context.Background(), pz.RestaurantID, owner.OwnerID)
	require.NoError(t, err)

	assert.Len(t, output, 2, "both fixture pizzas are `available`")
}

func TestListPizzas_ExcludesArchived(t *testing.T) {
	env := setupListPizzas(t)

	pz := firstPizza(t, env.DB)
	owner := restaurantByID(t, env.DB, pz.RestaurantID)

	require.NoError(t, env.DB.Model(&pizza.Pizza{}).
		Where("id = ?", pz.ID).Update("status", pizza.PizzaArchived).Error)

	output, err := env.ListPizzas.Execute(context.Background(), pz.RestaurantID, owner.OwnerID)
	require.NoError(t, err)

	assert.Len(t, output, 1, "archived pizza must be filtered out")
	for _, p := range output {
		assert.NotEqual(t, pz.ID, p.ID)
	}
}

func TestListPizzas_ToppingExtraPrice_OmittedWhenUnset_SetWhenPriced(t *testing.T) {
	env := setupListPizzas(t)

	pz := firstPizza(t, env.DB)
	owner := restaurantByID(t, env.DB, pz.RestaurantID)

	var toppings []topping.Topping
	require.NoError(t, env.DB.Order("name").Limit(2).Find(&toppings).Error)

	require.NoError(t, pz.SetToppingIDs([]uuid.UUID{toppings[0].ID, toppings[1].ID}))
	require.NoError(t, env.DB.Save(&pz).Error)

	toppingPriceRepo := persistence.NewToppingPriceRepository(env.DB)
	price, err := topping.NewToppingPrice(pz.RestaurantID, toppings[0].ID, decimal.RequireFromString("1.50"))
	require.NoError(t, err)
	require.NoError(t, toppingPriceRepo.UpsertPrices(
		context.Background(), pz.RestaurantID, []topping.ToppingPrice{*price},
	))

	output, err := env.ListPizzas.Execute(context.Background(), pz.RestaurantID, owner.OwnerID)
	require.NoError(t, err)

	var found pizzaapp.PizzaResponse
	for _, p := range output {
		if p.ID == pz.ID {
			found = p
		}
	}
	require.Len(t, found.Toppings, 2)

	byID := make(map[uuid.UUID]toppingapp.ToppingResponse, len(found.Toppings))
	for _, tr := range found.Toppings {
		byID[tr.ToppingID] = tr
	}

	require.NotNil(t, byID[toppings[0].ID].ExtraPrice, "priced topping must show extraPrice")
	assert.True(t, decimal.RequireFromString("1.50").Equal(decimal.Decimal(*byID[toppings[0].ID].ExtraPrice)))
	assert.Nil(t, byID[toppings[1].ID].ExtraPrice, "unpriced topping must omit extraPrice, not show 0")
}

func TestListPizzas_RestaurantNotOwned(t *testing.T) {
	env := setupListPizzas(t)

	pz := firstPizza(t, env.DB)

	_, err := env.ListPizzas.Execute(context.Background(), pz.RestaurantID, uuid.New())
	require.Error(t, err)

	assert.ErrorIs(t, err, apperr.ErrForbidden)
}
