package commands_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"restaurant-service/internal/application/restaurant/commands"
	"restaurant-service/internal/domain/payout"
	"restaurant-service/internal/domain/restaurant"
	"restaurant-service/internal/infrastructure/persistence"
	apperr "restaurant-service/internal/shared/errors"
	"restaurant-service/tests/infrastructure/db/fixtures"
	"restaurant-service/tests/testutil"
)

type approveRestaurantSetup struct {
	DB                *gorm.DB
	ApproveRestaurant *commands.ApproveRestaurant
}

func setupApproveRestaurant(t *testing.T) approveRestaurantSetup {
	db := testutil.DB(t)
	db.TruncateTables(t, testutil.TableRestaurant)

	_ = fixtures.LoadRestaurantFixtures(t, db.DB)

	restaurantRepo := persistence.NewRestaurantRepository(db.DB)
	payoutDetailsRepo := persistence.NewPayoutDetailsRepository(db.DB)
	approveRestaurant := commands.NewApproveRestaurant(restaurantRepo, payoutDetailsRepo, testutil.NoopPublisher{})

	return approveRestaurantSetup{
		DB:                db.DB,
		ApproveRestaurant: approveRestaurant,
	}
}

func createPendingPayout(t *testing.T, db *gorm.DB, restaurantID uuid.UUID) *payout.PayoutDetails {
	pd, err := payout.NewPayoutDetails(restaurantID, "Jane Doe", "DE89370400440532013000", "COBADEFFXXX", "Bank")
	require.NoError(t, err)
	require.NoError(t, persistence.NewPayoutDetailsRepository(db).Create(context.Background(), pd))

	return pd
}

func TestApproveRestaurant_Success(t *testing.T) {
	env := setupApproveRestaurant(t)

	res := firstRestaurant(t, env.DB)
	res.Status = restaurant.StatusReview
	res.Email = testutil.StringPtr("kontakt@pizzaparadise.de")
	require.NoError(t, env.DB.Save(&res).Error)
	createPendingPayout(t, env.DB, res.ID)

	output, err := env.ApproveRestaurant.Execute(context.Background(), res.ID)
	require.NoError(t, err)

	assert.Equal(t, restaurant.StatusApproved, output.Status)
	assert.Equal(t, payout.PayoutActive, output.Payout.Status)
	assert.Equal(t, "Jane Doe", output.Payout.AccountHolder)

	var updated restaurant.Restaurant

	err = env.DB.Take(&updated, "id = ?", res.ID).Error
	require.NoError(t, err)

	assert.Equal(t, restaurant.StatusApproved, updated.Status)

	var promoted payout.PayoutDetails

	err = env.DB.Take(&promoted, "restaurant_id = ?", res.ID).Error
	require.NoError(t, err)

	assert.Equal(t, payout.PayoutActive, promoted.Status)
}

func TestApproveRestaurant_FailsIfNoPendingPayout(t *testing.T) {
	env := setupApproveRestaurant(t)

	res := firstRestaurant(t, env.DB)
	res.Status = restaurant.StatusReview
	res.Email = testutil.StringPtr("kontakt@pizzaparadise.de")
	require.NoError(t, env.DB.Save(&res).Error)

	_, err := env.ApproveRestaurant.Execute(context.Background(), res.ID)

	require.Error(t, err)
	assert.ErrorIs(t, err, payout.ErrNoPendingPayout)

	var unchanged restaurant.Restaurant

	err = env.DB.Take(&unchanged, "id = ?", res.ID).Error
	require.NoError(t, err)

	assert.Equal(
		t, restaurant.StatusApproved, unchanged.Status,
		"restaurant status is already saved before payout promotion is attempted",
	)
}

func TestApproveRestaurant_NotFound(t *testing.T) {
	env := setupApproveRestaurant(t)

	_, err := env.ApproveRestaurant.Execute(context.Background(), uuid.New())

	require.Error(t, err)
	assert.ErrorIs(t, err, apperr.ErrNotFound)
}

func TestApproveRestaurant_FailsIfNotPendingReview(t *testing.T) {
	env := setupApproveRestaurant(t)

	res := firstRestaurant(t, env.DB)
	res.Status = restaurant.StatusDraft
	require.NoError(t, env.DB.Save(&res).Error)

	_, err := env.ApproveRestaurant.Execute(context.Background(), res.ID)

	require.Error(t, err)
	assert.ErrorIs(t, err, restaurant.ErrNotPendingReview)
	assert.ErrorIs(t, err, apperr.ErrConflict)

	var unchanged restaurant.Restaurant

	err = env.DB.Take(&unchanged, "id = ?", res.ID).Error
	require.NoError(t, err)

	assert.Equal(t, restaurant.StatusDraft, unchanged.Status)
}
