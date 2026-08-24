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
	occurredAt := time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC)
	restaurantID := uuid.New()

	payload := resapp.RestaurantReadyForReviewPayload{
		RestaurantID:   restaurantID,
		RestaurantName: "Pizza Paradise",
		EventName:      "restaurant.ready_for_review",
		OccurredAt:     occurredAt,
	}

	out, err := json.Marshal(payload)
	require.NoError(t, err)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(out, &decoded))

	assert.Equal(t, restaurantID.String(), decoded["restaurant_id"])
	assert.Equal(t, "Pizza Paradise", decoded["restaurant_name"])
	assert.Equal(t, "restaurant.ready_for_review", decoded["event_name"])
	assert.Equal(t, occurredAt.Format(time.RFC3339), decoded["occurred_at"])
}

func TestRestaurantApprovedPayload_GetEventName(t *testing.T) {
	payload := resapp.RestaurantApprovedPayload{}

	assert.Equal(t, "restaurant.approved", payload.GetEventName())
}

func TestRestaurantApprovedPayload_MarshalJSON(t *testing.T) {
	occurredAt := time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC)
	restaurantID := uuid.New()

	payload := resapp.RestaurantApprovedPayload{
		RestaurantID:   restaurantID,
		RestaurantName: "Pizza Paradise",
		Email:          "kontakt@pizzaparadise.de",
		EventName:      "restaurant.approved",
		OccurredAt:     occurredAt,
	}

	out, err := json.Marshal(payload)
	require.NoError(t, err)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(out, &decoded))

	assert.Equal(t, restaurantID.String(), decoded["restaurant_id"])
	assert.Equal(t, "Pizza Paradise", decoded["restaurant_name"])
	assert.Equal(t, "kontakt@pizzaparadise.de", decoded["email"])
	assert.Equal(t, "restaurant.approved", decoded["event_name"])
	assert.Equal(t, occurredAt.Format(time.RFC3339), decoded["occurred_at"])
}

func TestRestaurantLaunchedPayload_GetEventName(t *testing.T) {
	payload := resapp.RestaurantLaunchedPayload{}

	assert.Equal(t, "restaurant.launched", payload.GetEventName())
}

func TestRestaurantLaunchedPayload_MarshalJSON(t *testing.T) {
	occurredAt := time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2026, 8, 11, 12, 0, 5, 0, time.UTC)
	restaurantID := uuid.New()

	payload := resapp.RestaurantLaunchedPayload{
		RestaurantID:   restaurantID,
		RestaurantName: "Pizza Paradise",
		EventName:      "restaurant.launched",
		OccurredAt:     occurredAt,
		UpdatedAt:      updatedAt,
	}

	out, err := json.Marshal(payload)
	require.NoError(t, err)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(out, &decoded))

	assert.Equal(t, restaurantID.String(), decoded["restaurant_id"])
	assert.Equal(t, "Pizza Paradise", decoded["restaurant_name"])
	assert.Equal(t, "restaurant.launched", decoded["event_name"])
	assert.Equal(t, occurredAt.Format(time.RFC3339), decoded["occurred_at"])
	assert.Equal(t, updatedAt.Format(time.RFC3339), decoded["updated_at"])
}
