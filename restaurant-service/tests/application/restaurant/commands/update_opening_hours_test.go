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

type updateOpeningHoursSetup struct {
	DB                 *gorm.DB
	UpdateOpeningHours *commands.UpdateOpeningHours
}

func setupUpdateOpeningHours(t *testing.T) updateOpeningHoursSetup {
	db := testutil.DB(t)
	db.TruncateTables(t, testutil.TableRestaurant)

	_ = fixtures.LoadRestaurantFixtures(t, db.DB)

	restaurantRepo := persistence.NewRestaurantRepository(db.DB)
	payoutDetailsRepo := persistence.NewPayoutDetailsRepository(db.DB)
	outboxRepo := persistence.NewOutboxRepository(db.DB)
	updateOpeningHours := commands.NewUpdateOpeningHours(db.DB, restaurantRepo, payoutDetailsRepo, outboxRepo)

	return updateOpeningHoursSetup{
		DB:                 db.DB,
		UpdateOpeningHours: updateOpeningHours,
	}
}

func validOpeningHoursInput() resapp.UpdateOpeningHoursRequest {
	weekday := []resapp.DayRangeRequest{{Open: "11:00", Close: "22:00"}}

	return resapp.UpdateOpeningHoursRequest{
		Monday:    weekday,
		Tuesday:   weekday,
		Wednesday: weekday,
		Thursday:  weekday,
		Friday:    weekday,
		Saturday:  weekday,
		Sunday:    nil,
	}
}

func TestUpdateOpeningHours_Success(t *testing.T) {
	env := setupUpdateOpeningHours(t)

	res := firstRestaurant(t, env.DB)

	assert.False(t, res.Checklist[restaurant.ChecklistOpeningHours])

	output, err := env.UpdateOpeningHours.Execute(
		context.Background(),
		res.ID,
		res.OwnerID,
		validOpeningHoursInput(),
	)

	require.NoError(t, err)

	assert.Equal(t, "11:00", output.OpeningHours.Monday[0].Open)
	assert.Equal(t, "22:00", output.OpeningHours.Monday[0].Close)
	assert.Empty(t, output.OpeningHours.Sunday)

	var updated restaurant.Restaurant

	err = env.DB.Take(&updated, "id = ?", res.ID).Error
	require.NoError(t, err)

	assert.True(t, updated.Checklist[restaurant.ChecklistOpeningHours])
	assert.Equal(t, "11:00", updated.OpeningHours.Friday[0].Open)
	assert.Empty(t, updated.OpeningHours.Sunday)
	assert.False(t, updated.UpdatedAt.IsZero())
}

func TestUpdateOpeningHours_PublishesUpdatedEvent_WhenActive(t *testing.T) {
	env := setupUpdateOpeningHours(t)

	var res restaurant.Restaurant
	require.NoError(t, env.DB.Where("slug = ?", "anatolische-kueche").Take(&res).Error)

	res.Status = restaurant.StatusActive
	require.NoError(t, env.DB.Save(&res).Error)

	_, err := env.UpdateOpeningHours.Execute(
		context.Background(),
		res.ID,
		res.OwnerID,
		validOpeningHoursInput(),
	)

	require.NoError(t, err)

	stored := firstOutboxEvent(t, env.DB, res.ID, "restaurant.updated")
	assert.Equal(t, outbox.StatusPending, stored.Status)

	var payload resapp.RestaurantUpdatedPayload
	require.NoError(t, json.Unmarshal(stored.Payload, &payload))

	assert.Equal(t, "restaurant.updated", payload.EventName)
	assert.Equal(t, res.ID, payload.RestaurantID)
	assert.Equal(t, "11:00", payload.OpeningHours.Monday[0].Open)
}

func TestUpdateOpeningHours_SupportsMultipleRangesPerDay(t *testing.T) {
	env := setupUpdateOpeningHours(t)

	res := firstRestaurant(t, env.DB)

	input := resapp.UpdateOpeningHoursRequest{
		Monday: []resapp.DayRangeRequest{
			{Open: "00:00", Close: "03:00"},
			{Open: "15:00", Close: "23:59"},
		},
	}

	output, err := env.UpdateOpeningHours.Execute(
		context.Background(),
		res.ID,
		res.OwnerID,
		input,
	)

	require.NoError(t, err)
	require.Len(t, output.OpeningHours.Monday, 2)

	assert.Equal(t, "00:00", output.OpeningHours.Monday[0].Open)
	assert.Equal(t, "03:00", output.OpeningHours.Monday[0].Close)
	assert.Equal(t, "15:00", output.OpeningHours.Monday[1].Open)
	assert.Equal(t, "23:59", output.OpeningHours.Monday[1].Close)
}

func TestUpdateOpeningHours_RestaurantNotOwned(t *testing.T) {
	env := setupUpdateOpeningHours(t)

	res := firstRestaurant(t, env.DB)

	otherOwnerID := uuid.New()

	_, err := env.UpdateOpeningHours.Execute(
		context.Background(),
		res.ID,
		otherOwnerID,
		validOpeningHoursInput(),
	)

	require.Error(t, err)

	assert.Contains(t, err.Error(), "access denied")
	assert.ErrorIs(t, err, apperr.ErrForbidden)
}

func TestUpdateOpeningHours_RestaurantNotFound(t *testing.T) {
	env := setupUpdateOpeningHours(t)

	_, err := env.UpdateOpeningHours.Execute(
		context.Background(),
		uuid.New(),
		uuid.New(),
		validOpeningHoursInput(),
	)

	require.Error(t, err)

	assert.Contains(t, err.Error(), "access denied")
	assert.ErrorIs(t, err, apperr.ErrForbidden)
}
