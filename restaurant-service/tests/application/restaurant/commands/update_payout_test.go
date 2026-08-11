package commands_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
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

type updatePayoutSetup struct {
	DB           *gorm.DB
	CreatePayout *commands.CreatePayout
	UpdatePayout *commands.UpdatePayout
}

func setupUpdatePayout(t *testing.T) updatePayoutSetup {
	db := testutil.DB(t)
	db.TruncateTables(t, testutil.TableRestaurant)

	_ = fixtures.LoadRestaurantFixtures(t, db.DB)

	restaurantRepo := persistence.NewRestaurantRepository(db.DB)
	payoutDetailsRepo := persistence.NewPayoutDetailsRepository(db.DB)
	createPayout := commands.NewCreatePayout(restaurantRepo, payoutDetailsRepo, testutil.NoopPublisher{})
	updatePayout := commands.NewUpdatePayout(restaurantRepo, payoutDetailsRepo)

	return updatePayoutSetup{
		DB:           db.DB,
		CreatePayout: createPayout,
		UpdatePayout: updatePayout,
	}
}

func TestUpdatePayout_Success(t *testing.T) {
	env := setupUpdatePayout(t)

	res := firstRestaurant(t, env.DB)

	_, err := env.CreatePayout.Execute(
		context.Background(),
		res.ID,
		res.OwnerID,
		validPayoutInput(),
	)
	require.NoError(t, err)

	edited := resapp.UpdatePayoutRequest{
		AccountHolder: "Ayse Yilmaz",
		IBAN:          "GB29NWBK60161331926819",
		BIC:           "NWBKGB2L",
		BankName:      "NatWest",
	}

	output, err := env.UpdatePayout.Execute(
		context.Background(),
		res.ID,
		res.OwnerID,
		edited,
	)

	require.NoError(t, err)

	assert.Equal(t, "Ayse Yilmaz", output.Payout.AccountHolder)
	assert.Equal(t, "GB29NWBK60161331926819", output.Payout.IBAN)
	assert.Equal(t, "NWBKGB2L", output.Payout.BIC)
	assert.Equal(t, "NatWest", output.Payout.BankName)
	assert.Equal(t, restaurant.PayoutPending, output.Payout.Status)

	pd := findPayoutDetailsByStatus(t, env.DB, res.ID, restaurant.PayoutPending)
	assert.Equal(t, "Ayse Yilmaz", pd.AccountHolder)
	assert.Equal(t, "GB29NWBK60161331926819", pd.IBAN)

	assert.Equal(t, int64(1), countPayoutDetails(t, env.DB, res.ID))
}

func TestUpdatePayout_DoesNotTouchRestaurantRow(t *testing.T) {
	env := setupUpdatePayout(t)

	res := firstRestaurant(t, env.DB)

	_, err := env.CreatePayout.Execute(
		context.Background(),
		res.ID,
		res.OwnerID,
		validPayoutInput(),
	)
	require.NoError(t, err)

	var afterCreate restaurant.Restaurant
	err = env.DB.Take(&afterCreate, "id = ?", res.ID).Error
	require.NoError(t, err)

	_, err = env.UpdatePayout.Execute(
		context.Background(),
		res.ID,
		res.OwnerID,
		resapp.UpdatePayoutRequest{
			AccountHolder: "Ayse Yilmaz",
			IBAN:          "GB29NWBK60161331926819",
			BIC:           "NWBKGB2L",
			BankName:      "NatWest",
		},
	)
	require.NoError(t, err)

	var afterUpdate restaurant.Restaurant
	err = env.DB.Take(&afterUpdate, "id = ?", res.ID).Error
	require.NoError(t, err)

	assert.Equal(t, afterCreate.UpdatedAt, afterUpdate.UpdatedAt)
}

func TestUpdatePayout_Failure_NothingPending(t *testing.T) {
	env := setupUpdatePayout(t)

	res := firstRestaurant(t, env.DB)

	_, err := env.UpdatePayout.Execute(
		context.Background(),
		res.ID,
		res.OwnerID,
		validPayoutInput(),
	)

	require.Error(t, err)
	assert.ErrorIs(t, err, restaurant.ErrNoPendingPayout)
}

func TestUpdatePayout_Failure_OnlyActiveRecordExists(t *testing.T) {
	env := setupUpdatePayout(t)

	restaurants := []restaurant.Restaurant{}
	err := env.DB.Find(&restaurants).Error
	require.NoError(t, err)

	var target restaurant.Restaurant
	for _, r := range restaurants {
		if r.Checklist[restaurant.ChecklistPayment] {
			target = r
			break
		}
	}
	require.NotEmpty(t, target.ID, "expected a fixture restaurant with an active payout record")

	_, err = env.UpdatePayout.Execute(
		context.Background(),
		target.ID,
		target.OwnerID,
		validPayoutInput(),
	)

	require.Error(t, err)
	assert.ErrorIs(t, err, restaurant.ErrNoPendingPayout)
}

func TestUpdatePayout_RestaurantNotOwned(t *testing.T) {
	env := setupUpdatePayout(t)

	res := firstRestaurant(t, env.DB)

	otherOwnerID := uuid.New()

	_, err := env.UpdatePayout.Execute(
		context.Background(),
		res.ID,
		otherOwnerID,
		validPayoutInput(),
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "access denied")
	assert.ErrorIs(t, err, apperr.ErrForbidden)
}
