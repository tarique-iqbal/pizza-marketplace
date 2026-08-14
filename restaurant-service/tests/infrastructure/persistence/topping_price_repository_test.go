package persistence_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"restaurant-service/internal/domain/restaurant"
	"restaurant-service/internal/domain/topping"
	"restaurant-service/internal/infrastructure/persistence"
	"restaurant-service/tests/infrastructure/db/fixtures"
	"restaurant-service/tests/testutil"
)

func setupToppingPriceRepo(
	t *testing.T,
) (*gorm.DB, topping.ToppingPriceRepository, restaurant.Restaurant, []topping.Topping) {
	db := testutil.DB(t)
	db.TruncateTables(t, testutil.TableRestaurant)

	require.NoError(t, fixtures.LoadRestaurantFixtures(t, db.DB))

	var res restaurant.Restaurant
	require.NoError(t, db.DB.Where("slug = ?", "anatolische-kueche").Take(&res).Error)

	var toppings []topping.Topping
	require.NoError(t, db.DB.Order("name").Limit(2).Find(&toppings).Error)
	require.Len(t, toppings, 2)

	return db.DB, persistence.NewToppingPriceRepository(db.DB), res, toppings
}

func mustNewToppingPrice(t *testing.T, restaurantID, toppingID uuid.UUID, price string) topping.ToppingPrice {
	t.Helper()

	p, err := topping.NewToppingPrice(restaurantID, toppingID, decimal.RequireFromString(price))
	require.NoError(t, err)

	return *p
}

func TestToppingPriceRepository_UpsertPrices_InsertsAndUpdates(t *testing.T) {
	_, repo, res, toppings := setupToppingPriceRepo(t)

	first := []topping.ToppingPrice{
		mustNewToppingPrice(t, res.ID, toppings[0].ID, "1.00"),
		mustNewToppingPrice(t, res.ID, toppings[1].ID, "1.50"),
	}
	require.NoError(t, repo.UpsertPrices(context.Background(), res.ID, first))

	all, err := repo.ListByRestaurant(context.Background(), res.ID)
	require.NoError(t, err)
	require.Len(t, all, 2)
	for _, p := range all {
		assert.NotNil(t, p.UpdatedAt, "UpdatedAt must not be nil after UpsertPrices")
	}

	// re-upsert only toppings[0] at a new price — toppings[1] must be untouched (additive, not a replace)
	second := []topping.ToppingPrice{
		mustNewToppingPrice(t, res.ID, toppings[0].ID, "2.00"),
	}
	require.NoError(t, repo.UpsertPrices(context.Background(), res.ID, second))

	all, err = repo.ListByRestaurant(context.Background(), res.ID)
	require.NoError(t, err)
	require.Len(t, all, 2, "upsert-only: nothing gets removed")

	byToppingID := make(map[uuid.UUID]topping.ToppingPrice, len(all))
	for _, p := range all {
		byToppingID[p.ToppingID] = p
	}

	assert.True(t, decimal.RequireFromString("2.00").Equal(byToppingID[toppings[0].ID].ExtraPrice))
	assert.True(t, decimal.RequireFromString("1.50").Equal(byToppingID[toppings[1].ID].ExtraPrice))
}

func TestToppingPriceRepository_UpsertPrices_EmptyIsNoop(t *testing.T) {
	_, repo, res, toppings := setupToppingPriceRepo(t)

	initial := []topping.ToppingPrice{
		mustNewToppingPrice(t, res.ID, toppings[0].ID, "1.00"),
	}
	require.NoError(t, repo.UpsertPrices(context.Background(), res.ID, initial))

	require.NoError(t, repo.UpsertPrices(context.Background(), res.ID, nil))

	all, err := repo.ListByRestaurant(context.Background(), res.ID)
	require.NoError(t, err)
	require.Len(t, all, 1)
}

func TestToppingPriceRepository_ListByRestaurant_ScopedPerRestaurant(t *testing.T) {
	db, repo, res, toppings := setupToppingPriceRepo(t)

	var other restaurant.Restaurant
	require.NoError(t, db.Where("name = ?", "Pizza Paradise").Take(&other).Error)

	require.NoError(t, repo.UpsertPrices(context.Background(), res.ID, []topping.ToppingPrice{
		mustNewToppingPrice(t, res.ID, toppings[0].ID, "1.00"),
	}))
	require.NoError(t, repo.UpsertPrices(context.Background(), other.ID, []topping.ToppingPrice{
		mustNewToppingPrice(t, other.ID, toppings[0].ID, "9.00"),
	}))

	all, err := repo.ListByRestaurant(context.Background(), res.ID)
	require.NoError(t, err)
	require.Len(t, all, 1)
	assert.True(t, decimal.RequireFromString("1.00").Equal(all[0].ExtraPrice))
}
