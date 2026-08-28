package commands_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"restaurant-service/internal/application/pizza/queries"
	resapp "restaurant-service/internal/application/restaurant"
	"restaurant-service/internal/application/restaurant/commands"
	"restaurant-service/internal/domain/pizza"
	"restaurant-service/internal/domain/restaurant"
	"restaurant-service/internal/domain/topping"
	"restaurant-service/internal/infrastructure/persistence"
	apperr "restaurant-service/internal/shared/errors"
	"restaurant-service/internal/shared/event"
	"restaurant-service/tests/infrastructure/db/fixtures"
	"restaurant-service/tests/testutil"
)

type launchRestaurantSetup struct {
	DB               *gorm.DB
	LaunchRestaurant *commands.LaunchRestaurant
	Publisher        *fakePublisher
}

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

func setupLaunchRestaurant(t *testing.T) launchRestaurantSetup {
	db := testutil.DB(t)
	db.TruncateTables(t, testutil.TableRestaurant)

	_ = fixtures.LoadRestaurantFixtures(t, db.DB)

	restaurantRepo := persistence.NewRestaurantRepository(db.DB)
	payoutDetailsRepo := persistence.NewPayoutDetailsRepository(db.DB)
	pizzaRepo := persistence.NewPizzaRepository(db.DB)
	pizzaPriceRepo := persistence.NewPizzaPriceRepository(db.DB)
	pizzaSizeRepo := persistence.NewPizzaSizeRepository(db.DB)
	toppingRepo := persistence.NewToppingRepository(db.DB)
	toppingPriceRepo := persistence.NewToppingPriceRepository(db.DB)

	pizzaCatalog := queries.NewPizzaCatalog(pizzaRepo, pizzaPriceRepo, pizzaSizeRepo, toppingRepo, toppingPriceRepo)
	publisher := &fakePublisher{}
	launchRestaurant := commands.NewLaunchRestaurant(
		restaurantRepo, payoutDetailsRepo, pizzaCatalog, toppingRepo, toppingPriceRepo, publisher,
	)

	return launchRestaurantSetup{
		DB:               db.DB,
		LaunchRestaurant: launchRestaurant,
		Publisher:        publisher,
	}
}

func priceMinimumPizzas(t *testing.T, db *gorm.DB, restaurantID uuid.UUID) {
	pizzaRepo := persistence.NewPizzaRepository(db)

	for i := range resapp.MinPizzasToLaunch {
		p := pizza.NewPizza(uuid.New(), restaurantID).WithDetails(
			fmt.Sprintf("Test Pizza %d", i+1), nil, false, pizza.PizzaAvailable, i,
		)
		require.NoError(t, pizzaRepo.Create(context.Background(), p))
		setPizzaPrice(t, db, p.ID)
	}
}

func TestLaunchRestaurant_Success(t *testing.T) {
	env := setupLaunchRestaurant(t)

	var res restaurant.Restaurant
	require.NoError(t, env.DB.Where("slug = ?", "anatolische-kueche").Take(&res).Error)

	res.Status = restaurant.StatusApproved
	require.NoError(t, env.DB.Save(&res).Error)
	priceMinimumPizzas(t, env.DB, res.ID)

	output, err := env.LaunchRestaurant.Execute(context.Background(), res.ID, res.OwnerID)
	require.NoError(t, err)

	assert.Equal(t, restaurant.StatusActive, output.Status)

	var updated restaurant.Restaurant

	err = env.DB.Take(&updated, "id = ?", res.ID).Error
	require.NoError(t, err)

	assert.Equal(t, restaurant.StatusActive, updated.Status)
}

func TestLaunchRestaurant_RestaurantNotOwned(t *testing.T) {
	env := setupLaunchRestaurant(t)

	res := firstRestaurant(t, env.DB)
	res.Status = restaurant.StatusApproved
	require.NoError(t, env.DB.Save(&res).Error)

	otherOwnerID := uuid.New()

	_, err := env.LaunchRestaurant.Execute(context.Background(), res.ID, otherOwnerID)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "access denied")
	assert.ErrorIs(t, err, apperr.ErrForbidden)
}

func TestLaunchRestaurant_RestaurantNotFound(t *testing.T) {
	env := setupLaunchRestaurant(t)

	_, err := env.LaunchRestaurant.Execute(context.Background(), uuid.New(), uuid.New())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "access denied")
	assert.ErrorIs(t, err, apperr.ErrForbidden)
}

func TestLaunchRestaurant_FailsIfNotApproved(t *testing.T) {
	env := setupLaunchRestaurant(t)

	res := firstRestaurant(t, env.DB)
	res.Status = restaurant.StatusReview
	require.NoError(t, env.DB.Save(&res).Error)
	priceMinimumPizzas(t, env.DB, res.ID)

	_, err := env.LaunchRestaurant.Execute(context.Background(), res.ID, res.OwnerID)

	require.Error(t, err)
	assert.ErrorIs(t, err, restaurant.ErrNotReadyToLaunch)
	assert.ErrorIs(t, err, apperr.ErrConflict)

	var unchanged restaurant.Restaurant

	err = env.DB.Take(&unchanged, "id = ?", res.ID).Error
	require.NoError(t, err)

	assert.Equal(t, restaurant.StatusReview, unchanged.Status)
}

func setPizzaPrice(t *testing.T, db *gorm.DB, pizzaID uuid.UUID) {
	var size pizza.PizzaSize
	require.NoError(t, db.Order("diameter_cm").First(&size).Error)

	price, err := pizza.NewPizzaPrice(pizzaID, size.ID, decimal.RequireFromString("9.99"))
	require.NoError(t, err)

	pizzaPriceRepo := persistence.NewPizzaPriceRepository(db)
	require.NoError(t, pizzaPriceRepo.ReplacePrices(context.Background(), pizzaID, []pizza.PizzaPrice{*price}))
}

func TestLaunchRestaurant_PublishesLaunchedEventWithMenu(t *testing.T) {
	env := setupLaunchRestaurant(t)
	require.NoError(t, fixtures.LoadPizzaFixtures(t, env.DB))

	var res restaurant.Restaurant
	require.NoError(t, env.DB.Where("slug = ?", "anatolische-kueche").Take(&res).Error)

	res.Status = restaurant.StatusApproved
	require.NoError(t, env.DB.Save(&res).Error)

	var pizzas []pizza.Pizza
	require.NoError(t, env.DB.Where("restaurant_id = ?", res.ID).Order("sort_order").Find(&pizzas).Error)
	for _, p := range pizzas {
		setPizzaPrice(t, env.DB, p.ID)
	}

	_, err := env.LaunchRestaurant.Execute(context.Background(), res.ID, res.OwnerID)
	require.NoError(t, err)

	require.Len(t, env.Publisher.events, 1)

	payload, ok := env.Publisher.events[0].(resapp.RestaurantLaunchedPayload)
	require.True(t, ok)

	assert.Equal(t, "restaurant.launched", payload.EventName)
	require.Len(t, payload.Pizzas, 2)
	assert.Equal(t, "Margherita", payload.Pizzas[0].Name)
	assert.Equal(t, "Salami", payload.Pizzas[1].Name)
	assert.Equal(t, "Hamburg", payload.Address.City)
	assert.Equal(t, []string{"vegetarian", "vegan", "halal"}, payload.Tags)
	assert.Equal(t, restaurant.DeliveryOwn, payload.Delivery.Type)
	assert.InDelta(t, 53.5511, payload.Lat, 0.0001)
	assert.False(t, payload.UpdatedAt.IsZero(), "must carry the restaurant row's real write timestamp")
}

func TestLaunchRestaurant_PublishesLaunchedEventWithToppingPrices(t *testing.T) {
	env := setupLaunchRestaurant(t)
	require.NoError(t, fixtures.LoadPizzaFixtures(t, env.DB))

	var res restaurant.Restaurant
	require.NoError(t, env.DB.Where("slug = ?", "anatolische-kueche").Take(&res).Error)

	res.Status = restaurant.StatusApproved
	require.NoError(t, env.DB.Save(&res).Error)

	var pizzas []pizza.Pizza
	require.NoError(t, env.DB.Where("restaurant_id = ?", res.ID).Order("sort_order").Find(&pizzas).Error)
	for _, p := range pizzas {
		setPizzaPrice(t, env.DB, p.ID)
	}

	var t1 topping.Topping
	require.NoError(t, env.DB.Order("name").Take(&t1).Error)

	price, err := topping.NewToppingPrice(res.ID, t1.ID, decimal.RequireFromString("1.50"))
	require.NoError(t, err)
	require.NoError(t, persistence.NewToppingPriceRepository(env.DB).
		UpsertPrices(context.Background(), res.ID, []topping.ToppingPrice{*price}))

	_, err = env.LaunchRestaurant.Execute(context.Background(), res.ID, res.OwnerID)
	require.NoError(t, err)

	require.Len(t, env.Publisher.events, 1)

	payload, ok := env.Publisher.events[0].(resapp.RestaurantLaunchedPayload)
	require.True(t, ok)

	require.Len(t, payload.ToppingPrices, 1, "topping prices set before launch must be seeded into the snapshot")
	assert.Equal(t, t1.ID, payload.ToppingPrices[0].ToppingID)
	assert.True(t, decimal.RequireFromString("1.50").Equal(decimal.Decimal(payload.ToppingPrices[0].ExtraPrice)))
}

func TestLaunchRestaurant_ExcludesUnpricedPizzasFromMenu(t *testing.T) {
	env := setupLaunchRestaurant(t)
	require.NoError(t, fixtures.LoadPizzaFixtures(t, env.DB))

	var res restaurant.Restaurant
	require.NoError(t, env.DB.Where("slug = ?", "anatolische-kueche").Take(&res).Error)

	res.Status = restaurant.StatusApproved
	require.NoError(t, env.DB.Save(&res).Error)

	var pizzas []pizza.Pizza
	require.NoError(t, env.DB.Where("restaurant_id = ?", res.ID).Order("sort_order").Find(&pizzas).Error)
	require.Len(t, pizzas, 2, "fixture seeds Margherita and Salami")

	setPizzaPrice(t, env.DB, pizzas[0].ID)

	extra := pizza.NewPizza(uuid.New(), res.ID).WithDetails("Funghi", nil, false, pizza.PizzaAvailable, 3)
	require.NoError(t, persistence.NewPizzaRepository(env.DB).Create(context.Background(), extra))
	setPizzaPrice(t, env.DB, extra.ID)

	_, err := env.LaunchRestaurant.Execute(context.Background(), res.ID, res.OwnerID)
	require.NoError(t, err)

	require.Len(t, env.Publisher.events, 1)

	payload, ok := env.Publisher.events[0].(resapp.RestaurantLaunchedPayload)
	require.True(t, ok)

	require.Len(t, payload.Pizzas, 2, "unpriced pizza must be excluded from the launched snapshot")

	names := []string{payload.Pizzas[0].Name, payload.Pizzas[1].Name}
	assert.Contains(t, names, "Margherita")
	assert.Contains(t, names, "Funghi")
	assert.NotContains(t, names, "Salami")
}

func TestLaunchRestaurant_FailsIfNotEnoughPizzas(t *testing.T) {
	env := setupLaunchRestaurant(t)
	require.NoError(t, fixtures.LoadPizzaFixtures(t, env.DB))

	var res restaurant.Restaurant
	require.NoError(t, env.DB.Where("slug = ?", "anatolische-kueche").Take(&res).Error)

	res.Status = restaurant.StatusApproved
	require.NoError(t, env.DB.Save(&res).Error)

	var pizzas []pizza.Pizza
	require.NoError(t, env.DB.Where("restaurant_id = ?", res.ID).Order("sort_order").Find(&pizzas).Error)
	require.Len(t, pizzas, 2, "fixture seeds Margherita and Salami")

	setPizzaPrice(t, env.DB, pizzas[0].ID)

	_, err := env.LaunchRestaurant.Execute(context.Background(), res.ID, res.OwnerID)

	require.Error(t, err)
	assert.ErrorIs(t, err, restaurant.ErrNotEnoughPizzas)
	assert.ErrorIs(t, err, apperr.ErrConflict)
	assert.Empty(t, env.Publisher.events, "no event should be published when launch is rejected")

	var unchanged restaurant.Restaurant

	err = env.DB.Take(&unchanged, "id = ?", res.ID).Error
	require.NoError(t, err)
	assert.Equal(t, restaurant.StatusApproved, unchanged.Status)
}
