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
	"restaurant-service/internal/infrastructure/persistence"
	apperr "restaurant-service/internal/shared/errors"
	"restaurant-service/tests/infrastructure/db/fixtures"
	"restaurant-service/tests/testutil"
)

type setToppingPricesSetup struct {
	DB               *gorm.DB
	SetToppingPrices *commands.SetToppingPrices
}

func setupSetToppingPrices(t *testing.T) setToppingPricesSetup {
	db := testutil.DB(t)
	db.TruncateTables(t, testutil.TableRestaurant)

	require.NoError(t, fixtures.LoadRestaurantFixtures(t, db.DB))

	restaurantRepo := persistence.NewRestaurantRepository(db.DB)
	toppingRepo := persistence.NewToppingRepository(db.DB)
	toppingPriceRepo := persistence.NewToppingPriceRepository(db.DB)

	return setToppingPricesSetup{
		DB: db.DB,
		SetToppingPrices: commands.NewSetToppingPrices(
			restaurantRepo, toppingRepo, toppingPriceRepo,
		),
	}
}

func TestSetToppingPrices_Success(t *testing.T) {
	env := setupSetToppingPrices(t)

	var res restaurant.Restaurant
	require.NoError(t, env.DB.Where("slug = ?", "anatolische-kueche").Take(&res).Error)

	var toppings []restaurant.Topping
	require.NoError(t, env.DB.Order("name").Limit(2).Find(&toppings).Error)

	input := resapp.SetToppingPricesRequest{
		Prices: []resapp.ToppingPriceInput{
			{ToppingID: toppings[0].ID, ExtraPrice: decimal.RequireFromString("1.00")},
			{ToppingID: toppings[1].ID, ExtraPrice: decimal.RequireFromString("1.50")},
		},
	}

	output, err := env.SetToppingPrices.Execute(context.Background(), res.ID, res.OwnerID, input)
	require.NoError(t, err)

	require.Len(t, output, 2)
	for _, p := range output {
		assert.NotEmpty(t, p.Name)
	}
}

func TestSetToppingPrices_ToppingDoesNotExist(t *testing.T) {
	env := setupSetToppingPrices(t)

	var res restaurant.Restaurant
	require.NoError(t, env.DB.Where("slug = ?", "anatolische-kueche").Take(&res).Error)

	input := resapp.SetToppingPricesRequest{
		Prices: []resapp.ToppingPriceInput{
			{ToppingID: uuid.New(), ExtraPrice: decimal.RequireFromString("1.00")},
		},
	}

	_, err := env.SetToppingPrices.Execute(context.Background(), res.ID, res.OwnerID, input)
	require.Error(t, err)

	assert.ErrorIs(t, err, apperr.ErrNotFound)
}

func TestSetToppingPrices_DuplicateToppingInRequest(t *testing.T) {
	env := setupSetToppingPrices(t)

	var res restaurant.Restaurant
	require.NoError(t, env.DB.Where("slug = ?", "anatolische-kueche").Take(&res).Error)

	var topping restaurant.Topping
	require.NoError(t, env.DB.Order("name").Take(&topping).Error)

	input := resapp.SetToppingPricesRequest{
		Prices: []resapp.ToppingPriceInput{
			{ToppingID: topping.ID, ExtraPrice: decimal.RequireFromString("1.00")},
			{ToppingID: topping.ID, ExtraPrice: decimal.RequireFromString("2.00")},
		},
	}

	_, err := env.SetToppingPrices.Execute(context.Background(), res.ID, res.OwnerID, input)
	require.Error(t, err)

	assert.ErrorIs(t, err, apperr.ErrConflict)
}

func TestSetToppingPrices_ExtraPriceTooLow(t *testing.T) {
	env := setupSetToppingPrices(t)

	var res restaurant.Restaurant
	require.NoError(t, env.DB.Where("slug = ?", "anatolische-kueche").Take(&res).Error)

	var topping restaurant.Topping
	require.NoError(t, env.DB.Order("name").Take(&topping).Error)

	input := resapp.SetToppingPricesRequest{
		Prices: []resapp.ToppingPriceInput{
			{ToppingID: topping.ID, ExtraPrice: decimal.RequireFromString("0.50")},
		},
	}

	_, err := env.SetToppingPrices.Execute(context.Background(), res.ID, res.OwnerID, input)
	require.Error(t, err)

	assert.ErrorIs(t, err, apperr.ErrInvalid)
}

func TestSetToppingPrices_ExtraPriceTooHigh(t *testing.T) {
	env := setupSetToppingPrices(t)

	var res restaurant.Restaurant
	require.NoError(t, env.DB.Where("slug = ?", "anatolische-kueche").Take(&res).Error)

	var topping restaurant.Topping
	require.NoError(t, env.DB.Order("name").Take(&topping).Error)

	input := resapp.SetToppingPricesRequest{
		Prices: []resapp.ToppingPriceInput{
			{ToppingID: topping.ID, ExtraPrice: decimal.RequireFromString("3.01")},
		},
	}

	_, err := env.SetToppingPrices.Execute(context.Background(), res.ID, res.OwnerID, input)
	require.Error(t, err)

	assert.ErrorIs(t, err, apperr.ErrInvalid)
}

func TestSetToppingPrices_ExtraPriceBoundsInclusive(t *testing.T) {
	env := setupSetToppingPrices(t)

	var res restaurant.Restaurant
	require.NoError(t, env.DB.Where("slug = ?", "anatolische-kueche").Take(&res).Error)

	var toppings []restaurant.Topping
	require.NoError(t, env.DB.Order("name").Limit(2).Find(&toppings).Error)

	input := resapp.SetToppingPricesRequest{
		Prices: []resapp.ToppingPriceInput{
			{ToppingID: toppings[0].ID, ExtraPrice: decimal.RequireFromString("1.00")},
			{ToppingID: toppings[1].ID, ExtraPrice: decimal.RequireFromString("3.00")},
		},
	}

	_, err := env.SetToppingPrices.Execute(context.Background(), res.ID, res.OwnerID, input)
	require.NoError(t, err, "1 and 3 are the inclusive bounds, both must be accepted")
}

func TestSetToppingPrices_RestaurantNotOwned(t *testing.T) {
	env := setupSetToppingPrices(t)

	var res restaurant.Restaurant
	require.NoError(t, env.DB.Where("slug = ?", "anatolische-kueche").Take(&res).Error)

	var topping restaurant.Topping
	require.NoError(t, env.DB.Order("name").Take(&topping).Error)

	input := resapp.SetToppingPricesRequest{
		Prices: []resapp.ToppingPriceInput{
			{ToppingID: topping.ID, ExtraPrice: decimal.RequireFromString("1.00")},
		},
	}

	_, err := env.SetToppingPrices.Execute(context.Background(), res.ID, uuid.New(), input)
	require.Error(t, err)

	assert.ErrorIs(t, err, apperr.ErrForbidden)
}
