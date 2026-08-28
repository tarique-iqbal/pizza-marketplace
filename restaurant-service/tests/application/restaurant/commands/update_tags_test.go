package commands_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	resapp "restaurant-service/internal/application/restaurant"
	"restaurant-service/internal/application/restaurant/commands"
	"restaurant-service/internal/domain/outbox"
	"restaurant-service/internal/domain/restaurant"
	"restaurant-service/internal/infrastructure/persistence"
	apperr "restaurant-service/internal/shared/errors"
	"restaurant-service/tests/infrastructure/db/fixtures"
	"restaurant-service/tests/testutil"
)

type updateTagsSetup struct {
	DB         *gorm.DB
	UpdateTags *commands.UpdateTags
}

func setupUpdateTags(t *testing.T) updateTagsSetup {
	db := testutil.DB(t)
	db.TruncateTables(t, testutil.TableRestaurant)

	_ = fixtures.LoadRestaurantFixtures(t, db.DB)

	restaurantRepo := persistence.NewRestaurantRepository(db.DB)
	payoutDetailsRepo := persistence.NewPayoutDetailsRepository(db.DB)
	outboxRepo := persistence.NewOutboxRepository(db.DB)
	updateTags := commands.NewUpdateTags(db.DB, restaurantRepo, payoutDetailsRepo, outboxRepo)

	return updateTagsSetup{
		DB:         db.DB,
		UpdateTags: updateTags,
	}
}

func TestUpdateTags_Success(t *testing.T) {
	env := setupUpdateTags(t)

	res := firstRestaurant(t, env.DB)

	input := resapp.UpdateTagsRequest{
		Tags: []restaurant.RestaurantTag{restaurant.TagVegan, restaurant.TagGlutenFree},
	}

	output, err := env.UpdateTags.Execute(context.Background(), res.ID, res.OwnerID, input)

	require.NoError(t, err)
	assert.Equal(t, []string{"vegan", "glutenfree"}, output.Tags)

	var updated restaurant.Restaurant

	err = env.DB.Take(&updated, "id = ?", res.ID).Error
	require.NoError(t, err)

	assert.Equal(t, []restaurant.RestaurantTag{restaurant.TagVegan, restaurant.TagGlutenFree}, updated.Tags)
	assert.False(t, updated.UpdatedAt.IsZero())
}

func TestUpdateTags_ClearsTags(t *testing.T) {
	env := setupUpdateTags(t)

	var res restaurant.Restaurant
	require.NoError(t, env.DB.Where("slug = ?", "anatolische-kueche").Take(&res).Error)
	require.NotEmpty(t, res.Tags)

	input := resapp.UpdateTagsRequest{Tags: []restaurant.RestaurantTag{}}

	output, err := env.UpdateTags.Execute(context.Background(), res.ID, res.OwnerID, input)

	require.NoError(t, err)
	assert.Empty(t, output.Tags)
}

func TestUpdateTags_PublishesUpdatedEvent_WhenActive(t *testing.T) {
	env := setupUpdateTags(t)

	var res restaurant.Restaurant
	require.NoError(t, env.DB.Where("slug = ?", "anatolische-kueche").Take(&res).Error)

	res.Status = restaurant.StatusActive
	require.NoError(t, env.DB.Save(&res).Error)

	input := resapp.UpdateTagsRequest{
		Tags: []restaurant.RestaurantTag{restaurant.TagHalal},
	}

	_, err := env.UpdateTags.Execute(context.Background(), res.ID, res.OwnerID, input)

	require.NoError(t, err)

	stored := firstOutboxEvent(t, env.DB, res.ID, "restaurant.updated")
	assert.Equal(t, outbox.StatusPending, stored.Status)

	var payload resapp.RestaurantUpdatedPayload
	require.NoError(t, json.Unmarshal(stored.Payload, &payload))

	assert.Equal(t, "restaurant.updated", payload.EventName)
	assert.Equal(t, res.ID, payload.RestaurantID)
	assert.Equal(t, []string{"halal"}, payload.Tags)
}

func TestUpdateTags_RestaurantNotOwned(t *testing.T) {
	env := setupUpdateTags(t)

	res := firstRestaurant(t, env.DB)

	otherOwnerID := uuid.New()

	input := resapp.UpdateTagsRequest{
		Tags: []restaurant.RestaurantTag{restaurant.TagVegan},
	}

	_, err := env.UpdateTags.Execute(context.Background(), res.ID, otherOwnerID, input)

	require.Error(t, err)

	assert.Contains(t, err.Error(), "access denied")
	assert.ErrorIs(t, err, apperr.ErrForbidden)
}

func TestUpdateTags_RestaurantNotFound(t *testing.T) {
	env := setupUpdateTags(t)

	input := resapp.UpdateTagsRequest{
		Tags: []restaurant.RestaurantTag{restaurant.TagVegan},
	}

	_, err := env.UpdateTags.Execute(context.Background(), uuid.New(), uuid.New(), input)

	require.Error(t, err)

	assert.Contains(t, err.Error(), "access denied")
	assert.ErrorIs(t, err, apperr.ErrForbidden)
}
