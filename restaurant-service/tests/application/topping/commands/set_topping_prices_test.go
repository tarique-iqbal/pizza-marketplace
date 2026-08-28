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
	toppingapp "restaurant-service/internal/application/topping"
	"restaurant-service/internal/application/topping/commands"
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

func (f *fakePublisher) PublishRaw(ctx context.Context, topic string, jsonData []byte) error {
	return nil
}

type setToppingPricesSetup struct {
	DB               *gorm.DB
	SetToppingPrices *commands.SetToppingPrices
	Publisher        *fakePublisher
}

func setupSetToppingPrices(t *testing.T) setToppingPricesSetup {
	db := testutil.DB(t)
	db.TruncateTables(t, testutil.TableRestaurant)

	require.NoError(t, fixtures.LoadRestaurantFixtures(t, db.DB))

	restaurantRepo := persistence.NewRestaurantRepository(db.DB)
	toppingRepo := persistence.NewToppingRepository(db.DB)
	toppingPriceRepo := persistence.NewToppingPriceRepository(db.DB)
	publisher := &fakePublisher{}

	return setToppingPricesSetup{
		DB: db.DB,
		SetToppingPrices: commands.NewSetToppingPrices(
			restaurantRepo, toppingRepo, toppingPriceRepo, publisher,
		),
		Publisher: publisher,
	}
}

func TestSetToppingPrices_Success(t *testing.T) {
	env := setupSetToppingPrices(t)

	var res restaurant.Restaurant
	require.NoError(t, env.DB.Where("slug = ?", "anatolische-kueche").Take(&res).Error)

	var toppings []topping.Topping
	require.NoError(t, env.DB.Order("name").Limit(2).Find(&toppings).Error)

	input := toppingapp.SetToppingPricesRequest{
		Prices: []toppingapp.ToppingPriceInput{
			{ToppingID: toppings[0].ID, ExtraPrice: decimal.RequireFromString("1.00")},
			{ToppingID: toppings[1].ID, ExtraPrice: decimal.RequireFromString("1.50")},
		},
	}

	output, err := env.SetToppingPrices.Execute(context.Background(), res.ID, res.OwnerID, input)
	require.NoError(t, err)

	require.Len(t, output, 2)

	byToppingID := make(map[uuid.UUID]toppingapp.ToppingPriceResponse, len(output))
	for _, p := range output {
		assert.NotEmpty(t, p.Name)
		byToppingID[p.ToppingID] = p
	}

	assert.True(t, decimal.RequireFromString("1.00").Equal(decimal.Decimal(byToppingID[toppings[0].ID].ExtraPrice)))
	assert.True(t, decimal.RequireFromString("1.50").Equal(decimal.Decimal(byToppingID[toppings[1].ID].ExtraPrice)))

	assert.Empty(t, env.Publisher.events, "no topping_prices_updated event while restaurant is still draft")
}

func TestSetToppingPrices_PublishesToppingPricesUpdatedEvent_WhenActive(t *testing.T) {
	env := setupSetToppingPrices(t)

	var res restaurant.Restaurant
	require.NoError(t, env.DB.Where("slug = ?", "anatolische-kueche").Take(&res).Error)
	res.Status = restaurant.StatusActive
	require.NoError(t, env.DB.Save(&res).Error)

	var toppings []topping.Topping
	require.NoError(t, env.DB.Order("name").Limit(2).Find(&toppings).Error)

	input := toppingapp.SetToppingPricesRequest{
		Prices: []toppingapp.ToppingPriceInput{
			{ToppingID: toppings[0].ID, ExtraPrice: decimal.RequireFromString("1.00")},
			{ToppingID: toppings[1].ID, ExtraPrice: decimal.RequireFromString("1.50")},
		},
	}

	output, err := env.SetToppingPrices.Execute(context.Background(), res.ID, res.OwnerID, input)
	require.NoError(t, err)

	require.Len(t, env.Publisher.events, 1)

	payload, ok := env.Publisher.events[0].(resapp.ToppingPricesUpdatedPayload)
	require.True(t, ok)

	assert.Equal(t, "restaurant.topping_prices_updated", payload.EventName)
	assert.Equal(t, res.ID, payload.RestaurantID)
	assert.Equal(t, output, payload.ToppingPrices)
	assert.False(t, payload.UpdatedAt.IsZero(), "must carry the topping_prices row's real write timestamp")
}

func TestSetToppingPrices_ToppingDoesNotExist(t *testing.T) {
	env := setupSetToppingPrices(t)

	var res restaurant.Restaurant
	require.NoError(t, env.DB.Where("slug = ?", "anatolische-kueche").Take(&res).Error)

	input := toppingapp.SetToppingPricesRequest{
		Prices: []toppingapp.ToppingPriceInput{
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

	var t1 topping.Topping
	require.NoError(t, env.DB.Order("name").Take(&t1).Error)

	input := toppingapp.SetToppingPricesRequest{
		Prices: []toppingapp.ToppingPriceInput{
			{ToppingID: t1.ID, ExtraPrice: decimal.RequireFromString("1.00")},
			{ToppingID: t1.ID, ExtraPrice: decimal.RequireFromString("2.00")},
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

	var t1 topping.Topping
	require.NoError(t, env.DB.Order("name").Take(&t1).Error)

	input := toppingapp.SetToppingPricesRequest{
		Prices: []toppingapp.ToppingPriceInput{
			{ToppingID: t1.ID, ExtraPrice: decimal.RequireFromString("0.50")},
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

	var t1 topping.Topping
	require.NoError(t, env.DB.Order("name").Take(&t1).Error)

	input := toppingapp.SetToppingPricesRequest{
		Prices: []toppingapp.ToppingPriceInput{
			{ToppingID: t1.ID, ExtraPrice: decimal.RequireFromString("3.01")},
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

	var toppings []topping.Topping
	require.NoError(t, env.DB.Order("name").Limit(2).Find(&toppings).Error)

	input := toppingapp.SetToppingPricesRequest{
		Prices: []toppingapp.ToppingPriceInput{
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

	var t1 topping.Topping
	require.NoError(t, env.DB.Order("name").Take(&t1).Error)

	input := toppingapp.SetToppingPricesRequest{
		Prices: []toppingapp.ToppingPriceInput{
			{ToppingID: t1.ID, ExtraPrice: decimal.RequireFromString("1.00")},
		},
	}

	_, err := env.SetToppingPrices.Execute(context.Background(), res.ID, uuid.New(), input)
	require.Error(t, err)

	assert.ErrorIs(t, err, apperr.ErrForbidden)
}
