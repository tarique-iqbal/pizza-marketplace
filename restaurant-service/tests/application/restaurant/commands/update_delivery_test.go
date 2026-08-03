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

type updateDeliverySetup struct {
	DB             *gorm.DB
	UpdateDelivery *commands.UpdateDelivery
}

func setupUpdateDelivery(t *testing.T) updateDeliverySetup {
	db := testutil.DB(t)
	db.TruncateTables(t, testutil.TableRestaurant)

	_ = fixtures.LoadRestaurantFixtures(t, db.DB)

	restaurantRepo := persistence.NewRestaurantRepository(db.DB)
	updateDelivery := commands.NewUpdateDelivery(restaurantRepo)

	return updateDeliverySetup{
		DB:             db.DB,
		UpdateDelivery: updateDelivery,
	}
}

func validDeliveryInput() resapp.UpdateDeliveryRequest {
	return resapp.UpdateDeliveryRequest{
		Pickup:       true,
		DeliveryType: restaurant.DeliveryOwn,
		DeliveryKm:   testutil.Int16Ptr(5),
		DeliveryFee:  decimal.NewFromFloat(2.50),
		MinimumOrder: decimal.NewFromFloat(15.00),
	}
}

func TestUpdateDelivery_Success(t *testing.T) {
	env := setupUpdateDelivery(t)

	res := firstRestaurant(t, env.DB)

	assert.False(t, res.Checklist[restaurant.ChecklistDelivery])

	output, err := env.UpdateDelivery.Execute(
		context.Background(),
		res.ID,
		res.OwnerID,
		validDeliveryInput(),
	)

	require.NoError(t, err)

	assert.True(t, output.Pickup)
	assert.Equal(t, restaurant.DeliveryOwn, output.Delivery.Type)
	assert.Equal(t, int16(5), *output.Delivery.RadiusKm)
	assert.True(t, decimal.NewFromFloat(2.50).Equal(output.Delivery.Fee))
	assert.True(t, decimal.NewFromFloat(15.00).Equal(output.Delivery.MinimumOrder))

	var updated restaurant.Restaurant

	err = env.DB.Take(&updated, "id = ?", res.ID).Error
	require.NoError(t, err)

	assert.True(t, updated.Checklist[restaurant.ChecklistDelivery])
	assert.True(t, updated.Pickup)
	assert.Equal(t, restaurant.DeliveryOwn, updated.DeliveryType)
	assert.Equal(t, int16(5), *updated.DeliveryKm)
	assert.True(t, decimal.NewFromFloat(2.50).Equal(updated.DeliveryFee))
	assert.True(t, decimal.NewFromFloat(15.00).Equal(updated.MinimumOrder))
	assert.False(t, updated.UpdatedAt.IsZero())
}

func TestUpdateDelivery_RestaurantNotOwned(t *testing.T) {
	env := setupUpdateDelivery(t)

	res := firstRestaurant(t, env.DB)

	otherOwnerID := uuid.New()

	_, err := env.UpdateDelivery.Execute(
		context.Background(),
		res.ID,
		otherOwnerID,
		validDeliveryInput(),
	)

	require.Error(t, err)

	assert.Contains(t, err.Error(), "access denied")
	assert.ErrorIs(t, err, apperr.ErrForbidden)
}

func TestUpdateDelivery_RestaurantNotFound(t *testing.T) {
	env := setupUpdateDelivery(t)

	_, err := env.UpdateDelivery.Execute(
		context.Background(),
		uuid.New(),
		uuid.New(),
		validDeliveryInput(),
	)

	require.Error(t, err)

	assert.Contains(t, err.Error(), "access denied")
	assert.ErrorIs(t, err, apperr.ErrForbidden)
}

func TestUpdateDelivery_PickupOnly_ClearsDeliveryKm(t *testing.T) {
	env := setupUpdateDelivery(t)

	res := firstRestaurant(t, env.DB)

	input := resapp.UpdateDeliveryRequest{
		Pickup:       true,
		DeliveryType: restaurant.DeliveryNone,
		DeliveryKm:   nil,
		DeliveryFee:  decimal.Zero,
		MinimumOrder: decimal.Zero,
	}

	output, err := env.UpdateDelivery.Execute(
		context.Background(),
		res.ID,
		res.OwnerID,
		input,
	)

	require.NoError(t, err)

	assert.Equal(t, restaurant.DeliveryNone, output.Delivery.Type)
	assert.Nil(t, output.Delivery.RadiusKm)

	var updated restaurant.Restaurant

	err = env.DB.Take(&updated, "id = ?", res.ID).Error
	require.NoError(t, err)

	assert.Nil(t, updated.DeliveryKm)
	assert.Equal(t, restaurant.DeliveryNone, updated.DeliveryType)
}
