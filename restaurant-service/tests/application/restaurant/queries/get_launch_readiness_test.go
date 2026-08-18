package queries_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
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

type getLaunchReadinessSetup struct {
	DB                 *gorm.DB
	GetLaunchReadiness *queries.GetLaunchReadiness
}

func setupGetLaunchReadiness(t *testing.T) getLaunchReadinessSetup {
	db := testutil.DB(t)
	db.TruncateTables(t, testutil.TableRestaurant)

	require.NoError(t, fixtures.LoadRestaurantFixtures(t, db.DB))

	restaurantRepo := persistence.NewRestaurantRepository(db.DB)
	pizzaRepo := persistence.NewPizzaRepository(db.DB)
	pizzaPriceRepo := persistence.NewPizzaPriceRepository(db.DB)
	pizzaSizeRepo := persistence.NewPizzaSizeRepository(db.DB)
	toppingRepo := persistence.NewToppingRepository(db.DB)
	toppingPriceRepo := persistence.NewToppingPriceRepository(db.DB)

	pizzaCatalog := pizzaqry.NewPizzaCatalog(pizzaRepo, pizzaPriceRepo, pizzaSizeRepo, toppingRepo, toppingPriceRepo)

	return getLaunchReadinessSetup{
		DB:                 db.DB,
		GetLaunchReadiness: queries.NewGetLaunchReadiness(restaurantRepo, pizzaCatalog),
	}
}

func setPizzaPrice(t *testing.T, db *gorm.DB, pizzaID uuid.UUID) {
	var size pizza.PizzaSize
	require.NoError(t, db.Order("diameter_cm").First(&size).Error)

	price, err := pizza.NewPizzaPrice(pizzaID, size.ID, decimal.RequireFromString("9.99"))
	require.NoError(t, err)

	pizzaPriceRepo := persistence.NewPizzaPriceRepository(db)
	require.NoError(t, pizzaPriceRepo.ReplacePrices(context.Background(), pizzaID, []pizza.PizzaPrice{*price}))
}

func anatolischeKueche(t *testing.T, db *gorm.DB) restaurant.Restaurant {
	var res restaurant.Restaurant
	require.NoError(t, db.Where("slug = ?", "anatolische-kueche").Take(&res).Error)

	return res
}

func TestGetLaunchReadiness_ReadyToLaunch(t *testing.T) {
	env := setupGetLaunchReadiness(t)
	require.NoError(t, fixtures.LoadPizzaFixtures(t, env.DB))

	res := anatolischeKueche(t, env.DB)
	res.Status = restaurant.StatusApproved
	require.NoError(t, env.DB.Save(&res).Error)

	var pizzas []pizza.Pizza
	require.NoError(t, env.DB.Where("restaurant_id = ?", res.ID).Order("sort_order").Find(&pizzas).Error)
	for _, p := range pizzas {
		setPizzaPrice(t, env.DB, p.ID)
	}

	output, err := env.GetLaunchReadiness.Execute(context.Background(), res.ID, res.OwnerID)
	require.NoError(t, err)

	assert.Equal(t, "Anatolische Küche", output.Name)
	assert.Equal(t, restaurant.StatusApproved, output.Status)
	assert.True(t, output.ReadyToLaunch)
	assert.Equal(t, 2, output.MinPizzasRequired)
	assert.Len(t, output.ReadyPizzas, 2)
	assert.Empty(t, output.IncompletePizzas)
	assert.Equal(t, "Welcome to launch! Your restaurant is ready to go live.", output.Comment)
}

func TestGetLaunchReadiness_NotEnoughPizzas(t *testing.T) {
	env := setupGetLaunchReadiness(t)
	require.NoError(t, fixtures.LoadPizzaFixtures(t, env.DB))

	res := anatolischeKueche(t, env.DB)
	res.Status = restaurant.StatusApproved
	require.NoError(t, env.DB.Save(&res).Error)

	var pizzas []pizza.Pizza
	require.NoError(t, env.DB.Where("restaurant_id = ?", res.ID).Order("sort_order").Find(&pizzas).Error)
	require.Len(t, pizzas, 2, "fixture seeds Margherita and Salami")

	setPizzaPrice(t, env.DB, pizzas[0].ID)

	output, err := env.GetLaunchReadiness.Execute(context.Background(), res.ID, res.OwnerID)
	require.NoError(t, err)

	assert.False(t, output.ReadyToLaunch)
	assert.Len(t, output.ReadyPizzas, 1)
	require.Len(t, output.IncompletePizzas, 1)
	assert.Equal(t, "Salami", output.IncompletePizzas[0].Name)
	assert.Equal(t, "Add 1 more priced pizza(s) to reach the minimum of 2.", output.Comment)
}

func TestGetLaunchReadiness_NotApprovedStatus(t *testing.T) {
	env := setupGetLaunchReadiness(t)
	require.NoError(t, fixtures.LoadPizzaFixtures(t, env.DB))

	res := anatolischeKueche(t, env.DB)
	require.Equal(t, restaurant.StatusDraft, res.Status, "fixture default status")

	var pizzas []pizza.Pizza
	require.NoError(t, env.DB.Where("restaurant_id = ?", res.ID).Order("sort_order").Find(&pizzas).Error)
	for _, p := range pizzas {
		setPizzaPrice(t, env.DB, p.ID)
	}

	output, err := env.GetLaunchReadiness.Execute(context.Background(), res.ID, res.OwnerID)
	require.NoError(t, err)

	assert.Len(t, output.ReadyPizzas, 2, "minimum-pizzas check is independent of status")
	assert.False(t, output.ReadyToLaunch, "not approved yet, so not ready to launch")
	assert.Equal(t, "Complete your onboarding checklist before you can be reviewed for launch.", output.Comment)
}

func TestGetLaunchReadiness_NotApplicableAfterLaunch(t *testing.T) {
	statuses := []restaurant.RestaurantStatus{
		restaurant.StatusActive,
		restaurant.StatusInactive,
		restaurant.StatusRejected,
		restaurant.StatusDisabled,
	}

	for _, status := range statuses {
		t.Run(string(status), func(t *testing.T) {
			env := setupGetLaunchReadiness(t)

			res := anatolischeKueche(t, env.DB)
			res.Status = status
			require.NoError(t, env.DB.Save(&res).Error)

			_, err := env.GetLaunchReadiness.Execute(context.Background(), res.ID, res.OwnerID)

			require.Error(t, err)
			assert.ErrorIs(t, err, restaurant.ErrLaunchReadinessNotApplicable)
			assert.ErrorIs(t, err, apperr.ErrConflict)
		})
	}
}

func TestGetLaunchReadiness_RestaurantNotOwned(t *testing.T) {
	env := setupGetLaunchReadiness(t)

	res := firstRestaurant(t, env.DB)

	_, err := env.GetLaunchReadiness.Execute(context.Background(), res.ID, uuid.New())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "access denied")
	assert.ErrorIs(t, err, apperr.ErrForbidden)
}

func TestGetLaunchReadiness_RestaurantNotFound(t *testing.T) {
	env := setupGetLaunchReadiness(t)

	_, err := env.GetLaunchReadiness.Execute(context.Background(), uuid.New(), uuid.New())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "access denied")
	assert.ErrorIs(t, err, apperr.ErrForbidden)
}
