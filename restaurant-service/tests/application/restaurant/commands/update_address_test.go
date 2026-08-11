package commands_test

import (
	"context"
	"errors"
	"fmt"
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

type mockGeocoder struct {
	lat float64
	lon float64
	err error
}

func (m *mockGeocoder) GeocodeAddress(
	ctx context.Context,
	addr restaurant.Address,
) (float64, float64, error) {
	return m.lat, m.lon, m.err
}

type updateAddressSetup struct {
	DB            *gorm.DB
	UpdateAddress *commands.UpdateAddress
}

func setupUpdateAddress(
	t *testing.T,
	lat float64,
	lon float64,
	errGeo error,
) updateAddressSetup {
	db := testutil.DB(t)
	db.TruncateTables(t, testutil.TableRestaurant)

	_ = fixtures.LoadRestaurantFixtures(t, db.DB)

	mockGeo := &mockGeocoder{
		lat: lat,
		lon: lon,
		err: errGeo,
	}

	restaurantRepo := persistence.NewRestaurantRepository(db.DB)
	updateAddress := commands.NewUpdateAddress(mockGeo, restaurantRepo, testutil.NoopPublisher{})

	return updateAddressSetup{
		DB:            db.DB,
		UpdateAddress: updateAddress,
	}
}

func validAddressInput() resapp.UpdateAddressRequest {
	return resapp.UpdateAddressRequest{
		House:      "1",
		Street:     "Main Str.",
		City:       "Cityville",
		PostalCode: "12345",
	}
}

func firstRestaurant(t *testing.T, db *gorm.DB) restaurant.Restaurant {
	var res restaurant.Restaurant

	err := db.First(&res).Error
	require.NoError(t, err)

	return res
}

func TestUpdateAddress_Success(t *testing.T) {
	updateAddr := setupUpdateAddress(t, 52.52, 13.405, nil)

	res := firstRestaurant(t, updateAddr.DB)

	assert.Empty(t, res.Slug)

	output, err := updateAddr.UpdateAddress.Execute(
		context.Background(),
		res.ID,
		res.OwnerID,
		validAddressInput(),
	)

	require.NoError(t, err)

	assert.Equal(t, "1", output.Address.House)
	assert.Equal(t, "Main Str.", output.Address.Street)
	assert.Equal(t, "Cityville", output.Address.City)
	assert.Equal(t, "12345", output.Address.PostalCode)

	var updated restaurant.Restaurant

	err = updateAddr.DB.Take(&updated, "id = ?", res.ID).Error
	require.NoError(t, err)

	assert.Equal(t, "1", updated.Address.House)
	assert.Equal(t, "Main Str.", updated.Address.Street)
	assert.Equal(t, "Cityville", updated.Address.City)
	assert.Equal(t, "12345", updated.Address.PostalCode)

	assert.Equal(t, 52.52, *updated.Lat)
	assert.Equal(t, 13.405, *updated.Lon)

	assert.NotEmpty(t, updated.Slug)
	assert.Contains(t, *updated.Slug, "cityville")
	assert.Contains(t, *updated.Slug, "main-str")
	assert.False(t, updated.UpdatedAt.IsZero())
}

func TestUpdateAddress_RestaurantNotOwned(t *testing.T) {
	updateAddr := setupUpdateAddress(t, 52.52, 13.405, nil)

	res := firstRestaurant(t, updateAddr.DB)

	otherOwnerID := uuid.New()

	_, err := updateAddr.UpdateAddress.Execute(
		context.Background(),
		res.ID,
		otherOwnerID,
		validAddressInput(),
	)

	require.Error(t, err)

	assert.Contains(t, err.Error(), "access denied")
	assert.ErrorIs(t, err, apperr.ErrForbidden)
}

func TestUpdateAddress_RestaurantNotFound(t *testing.T) {
	updateAddr := setupUpdateAddress(t, 52.52, 13.405, nil)

	_, err := updateAddr.UpdateAddress.Execute(
		context.Background(),
		uuid.New(),
		uuid.New(),
		validAddressInput(),
	)

	require.Error(t, err)

	assert.Contains(t, err.Error(), "access denied")
	assert.ErrorIs(t, err, apperr.ErrForbidden)
}

func TestUpdateAddress_GeocoderFails(t *testing.T) {
	geoErr := errors.New("geocoder failed")

	env := setupUpdateAddress(t, 0, 0, geoErr)

	res := firstRestaurant(t, env.DB)

	_, err := env.UpdateAddress.Execute(
		context.Background(),
		res.ID,
		res.OwnerID,
		validAddressInput(),
	)

	require.Error(t, err)

	assert.Contains(t, err.Error(), "failed to geocode address")
	assert.ErrorIs(t, err, geoErr)

	var unchanged restaurant.Restaurant

	dbErr := env.DB.Take(&unchanged, "id = ?", res.ID).Error
	require.NoError(t, dbErr)

	assert.Empty(t, unchanged.Slug)

	assert.Nil(t, unchanged.Lat)
	assert.Nil(t, unchanged.Lon)

	assert.Empty(t, unchanged.Address.Street)
	assert.Empty(t, unchanged.Address.City)
}

func TestUpdateAddress_GeneratesUniqueSlug_WhenSlugAlreadyExists(t *testing.T) {
	updateAddr := setupUpdateAddress(t, 52.52, 13.405, nil)

	restaurants := []restaurant.Restaurant{}

	err := updateAddr.DB.Find(&restaurants).Error
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(restaurants), 2)

	target := restaurants[0]
	conflict := restaurants[1]

	conflict.WithSlug("test-restaurant-cityville-main-str")
	err = updateAddr.DB.Save(&conflict).Error
	require.NoError(t, err)

	target.Name = "Test Restaurant"
	err = updateAddr.DB.Save(&target).Error
	require.NoError(t, err)

	input := validAddressInput()

	_, err = updateAddr.UpdateAddress.Execute(
		context.Background(),
		target.ID,
		target.OwnerID,
		input,
	)

	require.NoError(t, err)

	var updated restaurant.Restaurant

	err = updateAddr.DB.Take(&updated, "id = ?", target.ID).Error
	require.NoError(t, err)

	assert.Equal(t, "test-restaurant-cityville-main-str-2", *updated.Slug)
}

func TestUpdateAddress_UsesBaseSlug_WhenOwnedRestaurantAlreadyHasSlug(t *testing.T) {
	updateAddr := setupUpdateAddress(t, 52.52, 13.405, nil)

	res := firstRestaurant(t, updateAddr.DB)

	res.Name = "Pizza Place"
	res.WithSlug("pizza-place-cityville-main-str")

	err := updateAddr.DB.Save(&res).Error
	require.NoError(t, err)

	_, err = updateAddr.UpdateAddress.Execute(
		context.Background(),
		res.ID,
		res.OwnerID,
		validAddressInput(),
	)

	require.NoError(t, err)

	var updated restaurant.Restaurant

	err = updateAddr.DB.Take(&updated, "id = ?", res.ID).Error
	require.NoError(t, err)

	assert.Equal(t, "pizza-place-cityville-main-str", *updated.Slug)
}

func TestUpdateAddress_Fails_WhenAllSlugVariationsTaken(t *testing.T) {
	updateAddr := setupUpdateAddress(t, 52.52, 13.405, nil)

	target := firstRestaurant(t, updateAddr.DB)

	target.Name = "Pizza Place"

	err := updateAddr.DB.Save(&target).Error
	require.NoError(t, err)

	base := "pizza-place-cityville-main-str"

	for i := 0; i <= 9; i++ {
		clone := restaurant.Restaurant{
			ID:      uuid.New(),
			OwnerID: uuid.New(),
			Name:    fmt.Sprintf("Clone %d", i),
		}

		if i == 0 {
			clone.WithSlug(base)
		} else {
			clone.WithSlug(fmt.Sprintf("%s-%d", base, i))
		}

		err = updateAddr.DB.Create(&clone).Error
		require.NoError(t, err)
	}

	_, err = updateAddr.UpdateAddress.Execute(
		context.Background(),
		target.ID,
		target.OwnerID,
		validAddressInput(),
	)

	require.Error(t, err)

	assert.Contains(t, err.Error(), "failed to generate unique slug")
}

func TestUpdateAddress_SlugIsSlugified(t *testing.T) {
	updateAddr := setupUpdateAddress(t, 52.52, 13.405, nil)

	res := firstRestaurant(t, updateAddr.DB)

	res.Name = "My Fancy Restaurant!!!"

	err := updateAddr.DB.Save(&res).Error
	require.NoError(t, err)

	input := resapp.UpdateAddressRequest{
		House:      "10A",
		Street:     "Äußere Straße",
		City:       "München",
		PostalCode: "80331",
	}

	_, err = updateAddr.UpdateAddress.Execute(
		context.Background(),
		res.ID,
		res.OwnerID,
		input,
	)

	require.NoError(t, err)

	var updated restaurant.Restaurant

	err = updateAddr.DB.Take(&updated, "id = ?", res.ID).Error
	require.NoError(t, err)

	assert.Equal(t, "my-fancy-restaurant-munchen-aussere-strasse", *updated.Slug)
}

func TestUpdateAddress_UpdatesExistingAddress(t *testing.T) {
	updateAddr := setupUpdateAddress(t, 11.11, 22.22, nil)

	res := firstRestaurant(t, updateAddr.DB)

	res.WithAddress(restaurant.Address{
		House:      "OLD",
		Street:     "OLD",
		City:       "OLD",
		PostalCode: "OLD",
	})

	err := updateAddr.DB.Save(&res).Error
	require.NoError(t, err)

	input := validAddressInput()

	_, err = updateAddr.UpdateAddress.Execute(
		context.Background(),
		res.ID,
		res.OwnerID,
		input,
	)

	require.NoError(t, err)

	var updated restaurant.Restaurant

	err = updateAddr.DB.Take(&updated, "id = ?", res.ID).Error
	require.NoError(t, err)

	assert.Equal(t, "1", updated.Address.House)
	assert.Equal(t, "Main Str.", updated.Address.Street)
	assert.Equal(t, "Cityville", updated.Address.City)
	assert.Equal(t, "12345", updated.Address.PostalCode)

	assert.Equal(t, 11.11, *updated.Lat)
	assert.Equal(t, 22.22, *updated.Lon)
}

func TestUpdateAddress_ResponseContainsUpdatedData(t *testing.T) {
	updateAddr := setupUpdateAddress(t, 40.7128, -74.0060, nil)

	res := firstRestaurant(t, updateAddr.DB)

	output, err := updateAddr.UpdateAddress.Execute(
		context.Background(),
		res.ID,
		res.OwnerID,
		validAddressInput(),
	)

	require.NoError(t, err)

	assert.Equal(t, res.ID, output.ID)
	assert.Equal(t, "1", output.Address.House)
	assert.Equal(t, "Main Str.", output.Address.Street)
	assert.Equal(t, "Cityville", output.Address.City)
	assert.Equal(t, "12345", output.Address.PostalCode)

	assert.NotEmpty(t, output.Slug)
}
