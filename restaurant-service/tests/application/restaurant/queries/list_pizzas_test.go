package queries_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	resapp "restaurant-service/internal/application/restaurant"
	"restaurant-service/internal/application/restaurant/queries"
	"restaurant-service/internal/domain/restaurant"
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

func firstPizza(t *testing.T, db *gorm.DB) restaurant.Pizza {
	var p restaurant.Pizza

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

	pizza := firstPizza(t, env.DB)
	owner := restaurantByID(t, env.DB, pizza.RestaurantID)

	output, err := env.ListPizzas.Execute(context.Background(), pizza.RestaurantID, owner.OwnerID)
	require.NoError(t, err)

	assert.Len(t, output, 2, "both fixture pizzas are `available`")
}

func TestListPizzas_ExcludesArchived(t *testing.T) {
	env := setupListPizzas(t)

	pizza := firstPizza(t, env.DB)
	owner := restaurantByID(t, env.DB, pizza.RestaurantID)

	require.NoError(t, env.DB.Model(&restaurant.Pizza{}).
		Where("id = ?", pizza.ID).Update("status", restaurant.PizzaArchived).Error)

	output, err := env.ListPizzas.Execute(context.Background(), pizza.RestaurantID, owner.OwnerID)
	require.NoError(t, err)

	assert.Len(t, output, 1, "archived pizza must be filtered out")
	for _, p := range output {
		assert.NotEqual(t, pizza.ID, p.ID)
	}
}

func TestListPizzas_ToppingExtraPrice_OmittedWhenUnset_SetWhenPriced(t *testing.T) {
	env := setupListPizzas(t)

	pizza := firstPizza(t, env.DB)
	owner := restaurantByID(t, env.DB, pizza.RestaurantID)

	var toppings []restaurant.Topping
	require.NoError(t, env.DB.Order("name").Limit(2).Find(&toppings).Error)

	require.NoError(t, pizza.SetToppingIDs([]uuid.UUID{toppings[0].ID, toppings[1].ID}))
	require.NoError(t, env.DB.Save(&pizza).Error)

	toppingPriceRepo := persistence.NewToppingPriceRepository(env.DB)
	price, err := restaurant.NewToppingPrice(pizza.RestaurantID, toppings[0].ID, decimal.RequireFromString("1.50"))
	require.NoError(t, err)
	require.NoError(t, toppingPriceRepo.UpsertPrices(context.Background(), pizza.RestaurantID, []restaurant.ToppingPrice{*price}))

	output, err := env.ListPizzas.Execute(context.Background(), pizza.RestaurantID, owner.OwnerID)
	require.NoError(t, err)

	var found resapp.PizzaResponse
	for _, p := range output {
		if p.ID == pizza.ID {
			found = p
		}
	}
	require.Len(t, found.Toppings, 2)

	byID := make(map[uuid.UUID]resapp.ToppingResponse, len(found.Toppings))
	for _, tr := range found.Toppings {
		byID[tr.ToppingID] = tr
	}

	require.NotNil(t, byID[toppings[0].ID].ExtraPrice, "priced topping must show extraPrice")
	assert.True(t, decimal.RequireFromString("1.50").Equal(decimal.Decimal(*byID[toppings[0].ID].ExtraPrice)))
	assert.Nil(t, byID[toppings[1].ID].ExtraPrice, "unpriced topping must omit extraPrice, not show 0")
}

func TestListPizzas_RestaurantNotOwned(t *testing.T) {
	env := setupListPizzas(t)

	pizza := firstPizza(t, env.DB)

	_, err := env.ListPizzas.Execute(context.Background(), pizza.RestaurantID, uuid.New())
	require.Error(t, err)

	assert.ErrorIs(t, err, apperr.ErrForbidden)
}
