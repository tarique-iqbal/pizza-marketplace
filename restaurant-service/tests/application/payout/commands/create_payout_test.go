package commands_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	payoutapp "restaurant-service/internal/application/payout"
	"restaurant-service/internal/application/payout/commands"
	"restaurant-service/internal/domain/payout"
	"restaurant-service/internal/domain/restaurant"
	"restaurant-service/internal/infrastructure/persistence"
	apperr "restaurant-service/internal/shared/errors"
	"restaurant-service/tests/infrastructure/db/fixtures"
	"restaurant-service/tests/testutil"
)

type createPayoutSetup struct {
	DB           *gorm.DB
	CreatePayout *commands.CreatePayout
}

func setupCreatePayout(t *testing.T) createPayoutSetup {
	db := testutil.DB(t)
	db.TruncateTables(t, testutil.TableRestaurant)

	_ = fixtures.LoadRestaurantFixtures(t, db.DB)

	restaurantRepo := persistence.NewRestaurantRepository(db.DB)
	payoutDetailsRepo := persistence.NewPayoutDetailsRepository(db.DB)
	createPayout := commands.NewCreatePayout(restaurantRepo, payoutDetailsRepo, testutil.NoopPublisher{})

	return createPayoutSetup{
		DB:           db.DB,
		CreatePayout: createPayout,
	}
}

func firstRestaurant(t *testing.T, db *gorm.DB) restaurant.Restaurant {
	var res restaurant.Restaurant

	err := db.First(&res).Error
	require.NoError(t, err)

	return res
}

func validPayoutInput() payoutapp.CreatePayoutRequest {
	return payoutapp.CreatePayoutRequest{
		AccountHolder: "Mehmet Yilmaz",
		IBAN:          "DE89370400440532013000",
		BIC:           "DEUTDEFF",
		BankName:      "Deutsche Bank",
	}
}

func findPayoutDetailsByStatus(
	t *testing.T,
	db *gorm.DB,
	restaurantID uuid.UUID,
	status payout.PayoutStatus,
) payout.PayoutDetails {
	var pd payout.PayoutDetails

	err := db.Take(&pd, "restaurant_id = ? AND status = ?", restaurantID, status).Error
	require.NoError(t, err)

	return pd
}

func countPayoutDetails(t *testing.T, db *gorm.DB, restaurantID uuid.UUID) int64 {
	var count int64

	err := db.Model(&payout.PayoutDetails{}).
		Where("restaurant_id = ?", restaurantID).
		Count(&count).Error
	require.NoError(t, err)

	return count
}

func TestCreatePayout_Success(t *testing.T) {
	env := setupCreatePayout(t)

	res := firstRestaurant(t, env.DB)

	assert.False(t, res.Checklist[restaurant.ChecklistPayout])

	output, err := env.CreatePayout.Execute(
		context.Background(),
		res.ID,
		res.OwnerID,
		validPayoutInput(),
	)

	require.NoError(t, err)

	assert.Equal(t, "Mehmet Yilmaz", output.Payout.AccountHolder)
	assert.Equal(t, "DE89370400440532013000", output.Payout.IBAN)
	assert.Equal(t, "DEUTDEFF", output.Payout.BIC)
	assert.Equal(t, "Deutsche Bank", output.Payout.BankName)
	assert.Equal(t, payout.PayoutPending, output.Payout.Status)

	var updated restaurant.Restaurant

	err = env.DB.Take(&updated, "id = ?", res.ID).Error
	require.NoError(t, err)

	assert.True(t, updated.Checklist[restaurant.ChecklistPayout])
	assert.False(t, updated.UpdatedAt.IsZero())

	pd := findPayoutDetailsByStatus(t, env.DB, res.ID, payout.PayoutPending)

	assert.Equal(t, "Mehmet Yilmaz", pd.AccountHolder)
	assert.Equal(t, "DE89370400440532013000", pd.IBAN)
	assert.Equal(t, "DEUTDEFF", pd.BIC)
	assert.Equal(t, "Deutsche Bank", pd.BankName)
}

func TestCreatePayout_RestaurantNotOwned(t *testing.T) {
	env := setupCreatePayout(t)

	res := firstRestaurant(t, env.DB)

	otherOwnerID := uuid.New()

	_, err := env.CreatePayout.Execute(
		context.Background(),
		res.ID,
		otherOwnerID,
		validPayoutInput(),
	)

	require.Error(t, err)

	assert.Contains(t, err.Error(), "access denied")
	assert.ErrorIs(t, err, apperr.ErrForbidden)
}

func TestCreatePayout_RestaurantNotFound(t *testing.T) {
	env := setupCreatePayout(t)

	_, err := env.CreatePayout.Execute(
		context.Background(),
		uuid.New(),
		uuid.New(),
		validPayoutInput(),
	)

	require.Error(t, err)

	assert.Contains(t, err.Error(), "access denied")
	assert.ErrorIs(t, err, apperr.ErrForbidden)
}

func TestCreatePayout_RejectsWhenPendingAlreadyExists(t *testing.T) {
	env := setupCreatePayout(t)

	res := firstRestaurant(t, env.DB)

	_, err := env.CreatePayout.Execute(
		context.Background(),
		res.ID,
		res.OwnerID,
		validPayoutInput(),
	)
	require.NoError(t, err)

	secondInput := payoutapp.CreatePayoutRequest{
		AccountHolder: "Ayse Yilmaz",
		IBAN:          "GB29NWBK60161331926819",
		BIC:           "NWBKGB2L",
		BankName:      "NatWest",
	}

	_, err = env.CreatePayout.Execute(
		context.Background(),
		res.ID,
		res.OwnerID,
		secondInput,
	)

	require.Error(t, err)
	assert.ErrorIs(t, err, payout.ErrPendingPayoutExists)

	assert.Equal(t, int64(1), countPayoutDetails(t, env.DB, res.ID))

	pd := findPayoutDetailsByStatus(t, env.DB, res.ID, payout.PayoutPending)
	assert.Equal(t, "Mehmet Yilmaz", pd.AccountHolder)
}

func TestCreatePayout_DoesNotTouchExistingActiveRecord(t *testing.T) {
	env := setupCreatePayout(t)

	restaurants := []restaurant.Restaurant{}
	err := env.DB.Find(&restaurants).Error
	require.NoError(t, err)

	var target restaurant.Restaurant
	for _, r := range restaurants {
		if r.Checklist[restaurant.ChecklistPayout] {
			target = r
			break
		}
	}
	require.NotEmpty(t, target.ID, "expected a fixture restaurant with an active payout record")

	activeBefore := findPayoutDetailsByStatus(t, env.DB, target.ID, payout.PayoutActive)

	newSubmission := payoutapp.CreatePayoutRequest{
		AccountHolder: "Ayse Yilmaz",
		IBAN:          "GB29NWBK60161331926819",
		BIC:           "NWBKGB2L",
		BankName:      "NatWest",
	}

	_, err = env.CreatePayout.Execute(
		context.Background(),
		target.ID,
		target.OwnerID,
		newSubmission,
	)
	require.NoError(t, err)

	activeAfter := findPayoutDetailsByStatus(t, env.DB, target.ID, payout.PayoutActive)
	assert.Equal(t, activeBefore.ID, activeAfter.ID)
	assert.Equal(t, activeBefore.IBAN, activeAfter.IBAN)

	pending := findPayoutDetailsByStatus(t, env.DB, target.ID, payout.PayoutPending)
	assert.Equal(t, "Ayse Yilmaz", pending.AccountHolder)
	assert.Equal(t, "GB29NWBK60161331926819", pending.IBAN)

	assert.Equal(t, int64(2), countPayoutDetails(t, env.DB, target.ID))
}
