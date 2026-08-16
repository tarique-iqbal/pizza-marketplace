package queries_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	pizzaqry "restaurant-service/internal/application/pizza/queries"
	"restaurant-service/internal/application/restaurant/queries"
	"restaurant-service/internal/domain/pizza"
	"restaurant-service/internal/domain/restaurant"
	"restaurant-service/internal/infrastructure/persistence"
	apperr "restaurant-service/internal/shared/errors"
	"restaurant-service/tests/infrastructure/db/fixtures"
	"restaurant-service/tests/testutil"
)

type getRestaurantSetup struct {
	DB            *gorm.DB
	GetRestaurant *queries.GetRestaurant
}

func setupGetRestaurant(t *testing.T) getRestaurantSetup {
	db := testutil.DB(t)
	db.TruncateTables(t, testutil.TableRestaurant)

	require.NoError(t, fixtures.LoadRestaurantFixtures(t, db.DB))

	restaurantRepo := persistence.NewRestaurantRepository(db.DB)
	payoutDetailsRepo := persistence.NewPayoutDetailsRepository(db.DB)
	pizzaRepo := persistence.NewPizzaRepository(db.DB)
	pizzaPriceRepo := persistence.NewPizzaPriceRepository(db.DB)
	pizzaSizeRepo := persistence.NewPizzaSizeRepository(db.DB)
	toppingRepo := persistence.NewToppingRepository(db.DB)
	toppingPriceRepo := persistence.NewToppingPriceRepository(db.DB)

	pizzaCatalog := pizzaqry.NewPizzaCatalog(pizzaRepo, pizzaPriceRepo, pizzaSizeRepo, toppingRepo, toppingPriceRepo)

	return getRestaurantSetup{
		DB:            db.DB,
		GetRestaurant: queries.NewGetRestaurant(restaurantRepo, payoutDetailsRepo, pizzaCatalog),
	}
}

func firstRestaurant(t *testing.T, db *gorm.DB) restaurant.Restaurant {
	var res restaurant.Restaurant

	err := db.Order("name").First(&res).Error
	require.NoError(t, err)

	return res
}

func TestGetRestaurant_Success(t *testing.T) {
	env := setupGetRestaurant(t)

	res := firstRestaurant(t, env.DB)

	output, err := env.GetRestaurant.Execute(context.Background(), res.ID, res.OwnerID)
	require.NoError(t, err)

	assert.Equal(t, res.ID, output.ID)
	assert.Equal(t, res.Name, output.Name)
}

func TestGetRestaurant_IncludesPizzas(t *testing.T) {
	env := setupGetRestaurant(t)
	require.NoError(t, fixtures.LoadPizzaFixtures(t, env.DB))

	var res restaurant.Restaurant
	require.NoError(t, env.DB.Where("slug = ?", "anatolische-kueche").Take(&res).Error)

	var margherita pizza.Pizza
	require.NoError(t, env.DB.Where("restaurant_id = ? AND name = ?", res.ID, "Margherita").Take(&margherita).Error)
	require.NoError(t, env.DB.Model(&margherita).Update("status", pizza.PizzaUnavailable).Error)

	output, err := env.GetRestaurant.Execute(context.Background(), res.ID, res.OwnerID)
	require.NoError(t, err)

	require.Len(t, output.Pizzas, 2, "unavailable pizzas still show, only archived is excluded")
	assert.Equal(t, "Margherita", output.Pizzas[0].Name)
	assert.Equal(t, "Salami", output.Pizzas[1].Name)
}

func TestGetRestaurant_RestaurantNotOwned(t *testing.T) {
	env := setupGetRestaurant(t)

	res := firstRestaurant(t, env.DB)

	_, err := env.GetRestaurant.Execute(context.Background(), res.ID, uuid.New())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "access denied")
	assert.ErrorIs(t, err, apperr.ErrForbidden)
}

func TestGetRestaurant_RestaurantNotFound(t *testing.T) {
	env := setupGetRestaurant(t)

	_, err := env.GetRestaurant.Execute(context.Background(), uuid.New(), uuid.New())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "access denied")
	assert.ErrorIs(t, err, apperr.ErrForbidden)
}
