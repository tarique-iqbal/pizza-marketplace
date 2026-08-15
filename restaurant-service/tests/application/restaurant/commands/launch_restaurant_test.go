package commands_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"restaurant-service/internal/application/pizza/queries"
	resapp "restaurant-service/internal/application/restaurant"
	"restaurant-service/internal/application/restaurant/commands"
	"restaurant-service/internal/domain/restaurant"
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
	launchRestaurant := commands.NewLaunchRestaurant(restaurantRepo, payoutDetailsRepo, pizzaCatalog, publisher)

	return launchRestaurantSetup{
		DB:               db.DB,
		LaunchRestaurant: launchRestaurant,
		Publisher:        publisher,
	}
}

func TestLaunchRestaurant_Success(t *testing.T) {
	env := setupLaunchRestaurant(t)

	res := firstRestaurant(t, env.DB)
	res.Status = restaurant.StatusApproved
	require.NoError(t, env.DB.Save(&res).Error)

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

	_, err := env.LaunchRestaurant.Execute(context.Background(), res.ID, res.OwnerID)

	require.Error(t, err)
	assert.ErrorIs(t, err, restaurant.ErrNotReadyToLaunch)
	assert.ErrorIs(t, err, apperr.ErrConflict)

	var unchanged restaurant.Restaurant

	err = env.DB.Take(&unchanged, "id = ?", res.ID).Error
	require.NoError(t, err)

	assert.Equal(t, restaurant.StatusReview, unchanged.Status)
}

func TestLaunchRestaurant_PublishesLaunchedEventWithMenu(t *testing.T) {
	env := setupLaunchRestaurant(t)
	require.NoError(t, fixtures.LoadPizzaFixtures(t, env.DB))

	var res restaurant.Restaurant
	require.NoError(t, env.DB.Where("slug = ?", "anatolische-kueche").Take(&res).Error)

	res.Status = restaurant.StatusApproved
	require.NoError(t, env.DB.Save(&res).Error)

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
	assert.InDelta(t, 53.5511, *payload.Lat, 0.0001)
}
