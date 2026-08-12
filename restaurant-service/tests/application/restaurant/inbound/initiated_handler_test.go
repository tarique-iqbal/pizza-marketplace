package inbound_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"restaurant-service/internal/application/restaurant/inbound"
	"restaurant-service/internal/domain/restaurant"
	"restaurant-service/internal/infrastructure/persistence"
	"restaurant-service/tests/testutil"
)

func setupRestaurantInitiatedHandler(t *testing.T) *inbound.RestaurantInitiated {
	db := testutil.DB(t)
	db.TruncateTables(t, testutil.TableRestaurant)

	repo := persistence.NewRestaurantRepository(db.DB)

	return inbound.NewRestaurantInitiated(repo)
}

func TestRestaurantInitiated_Handle_Success(t *testing.T) {
	db := testutil.DB(t)
	handler := setupRestaurantInitiatedHandler(t)

	restaurantID := testutil.MustNewID()
	ownerID := testutil.MustNewID()

	eventPayload := restaurant.EventPayload{
		Data: []byte(`{
			"restaurant_id": "` + restaurantID.String() + `",
			"owner_id": "` + ownerID.String() + `",
			"business_name": "Pizza Palace",
			"vat_number": "DE123456789"
		}`),
	}

	err := handler.Handle(context.Background(), eventPayload)

	require.NoError(t, err)

	var res restaurant.Restaurant

	err = db.DB.
		Where("id = ?", restaurantID).
		First(&res).
		Error

	require.NoError(t, err)

	assert.Equal(t, restaurantID, res.ID)
	assert.Equal(t, ownerID, res.OwnerID)
	assert.Equal(t, "Pizza Palace", res.Name)
	assert.Equal(t, "DE123456789", res.VATNumber)
	assert.True(t, res.Checklist[restaurant.ChecklistBasic])
}

func TestRestaurantInitiated_Handle_InvalidJSON(t *testing.T) {
	db := testutil.DB(t)
	handler := setupRestaurantInitiatedHandler(t)

	eventPayload := restaurant.EventPayload{
		Data: []byte(`{invalid-json}`),
	}

	err := handler.Handle(context.Background(), eventPayload)

	require.Error(t, err)

	var count int64

	err = db.DB.
		Model(&restaurant.Restaurant{}).
		Count(&count).
		Error

	require.NoError(t, err)
	assert.Equal(t, int64(0), count)
}

func TestRestaurantInitiated_Handle_DuplicateRestaurant(t *testing.T) {
	db := testutil.DB(t)
	handler := setupRestaurantInitiatedHandler(t)

	restaurantID := testutil.MustNewID()
	ownerID := testutil.MustNewID()

	payload := []byte(`{
		"restaurant_id": "` + restaurantID.String() + `",
		"owner_id": "` + ownerID.String() + `",
		"business_name": "Pizza Palace",
		"vat_number": "DE123456789"
	}`)

	eventPayload := restaurant.EventPayload{
		Data: payload,
	}

	err := handler.Handle(context.Background(), eventPayload)
	require.NoError(t, err)

	// second insert fail
	err = handler.Handle(context.Background(), eventPayload)
	require.Error(t, err)

	var count int64

	err = db.DB.
		Model(&restaurant.Restaurant{}).
		Where("id = ?", restaurantID).
		Count(&count).
		Error

	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}
