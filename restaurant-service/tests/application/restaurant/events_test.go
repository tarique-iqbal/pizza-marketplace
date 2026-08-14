package restaurant_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	resapp "restaurant-service/internal/application/restaurant"
)

func TestRestaurantReadyForReviewPayload_GetEventName(t *testing.T) {
	payload := resapp.RestaurantReadyForReviewPayload{}

	assert.Equal(t, "restaurant.ready_for_review", payload.GetEventName())
}

func TestRestaurantReadyForReviewPayload_MarshalJSON(t *testing.T) {
	readyAt := time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC)
	restaurantID := uuid.New()

	payload := resapp.RestaurantReadyForReviewPayload{
		RestaurantID:   restaurantID,
		RestaurantName: "Pizza Paradise",
		EventName:      "restaurant.ready_for_review",
		ReadyAt:        readyAt,
	}

	out, err := json.Marshal(payload)
	require.NoError(t, err)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(out, &decoded))

	assert.Equal(t, restaurantID.String(), decoded["restaurant_id"])
	assert.Equal(t, "Pizza Paradise", decoded["restaurant_name"])
	assert.Equal(t, "restaurant.ready_for_review", decoded["event_name"])
	assert.Equal(t, readyAt.Format(time.RFC3339), decoded["ready_at"])
}

func TestRestaurantApprovedPayload_GetEventName(t *testing.T) {
	payload := resapp.RestaurantApprovedPayload{}

	assert.Equal(t, "restaurant.approved", payload.GetEventName())
}

func TestRestaurantApprovedPayload_MarshalJSON(t *testing.T) {
	approvedAt := time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC)
	restaurantID := uuid.New()

	payload := resapp.RestaurantApprovedPayload{
		RestaurantID:   restaurantID,
		RestaurantName: "Pizza Paradise",
		EventName:      "restaurant.approved",
		ApprovedAt:     approvedAt,
	}

	out, err := json.Marshal(payload)
	require.NoError(t, err)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(out, &decoded))

	assert.Equal(t, restaurantID.String(), decoded["restaurant_id"])
	assert.Equal(t, "Pizza Paradise", decoded["restaurant_name"])
	assert.Equal(t, "restaurant.approved", decoded["event_name"])
	assert.Equal(t, approvedAt.Format(time.RFC3339), decoded["approved_at"])
}

