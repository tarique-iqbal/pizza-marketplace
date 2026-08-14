package commands_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"restaurant-service/internal/application/restaurant/commands"
	"restaurant-service/internal/domain/restaurant"
	"restaurant-service/internal/infrastructure/persistence"
	apperr "restaurant-service/internal/shared/errors"
	"restaurant-service/tests/infrastructure/db/fixtures"
	"restaurant-service/tests/testutil"
)

type launchRestaurantSetup struct {
	DB               *gorm.DB
	LaunchRestaurant *commands.LaunchRestaurant
}

func setupLaunchRestaurant(t *testing.T) launchRestaurantSetup {
	db := testutil.DB(t)
	db.TruncateTables(t, testutil.TableRestaurant)

	_ = fixtures.LoadRestaurantFixtures(t, db.DB)

	restaurantRepo := persistence.NewRestaurantRepository(db.DB)
	payoutDetailsRepo := persistence.NewPayoutDetailsRepository(db.DB)
	launchRestaurant := commands.NewLaunchRestaurant(restaurantRepo, payoutDetailsRepo, testutil.NoopPublisher{})

	return launchRestaurantSetup{
		DB:               db.DB,
		LaunchRestaurant: launchRestaurant,
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
