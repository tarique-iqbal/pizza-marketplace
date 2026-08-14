package restaurant

import (
	"time"

	"github.com/google/uuid"

	"restaurant-service/internal/domain/restaurant"
)

type RestaurantReadyForReviewPayload struct {
	RestaurantID   uuid.UUID `json:"restaurant_id"`
	RestaurantName string    `json:"restaurant_name"`
	EventName      string    `json:"event_name"`
	ReadyAt        time.Time `json:"ready_at"`
}

func (RestaurantReadyForReviewPayload) GetEventName() string {
	return "restaurant.ready_for_review"
}

func newRestaurantReadyForReviewPayload(
	e restaurant.RestaurantReadyForReview,
) RestaurantReadyForReviewPayload {
	payload := RestaurantReadyForReviewPayload{
		RestaurantID:   e.RestaurantID,
		RestaurantName: e.RestaurantName,
		ReadyAt:        e.ReadyAt,
	}
	payload.EventName = payload.GetEventName()

	return payload
}

type RestaurantApprovedPayload struct {
	RestaurantID   uuid.UUID `json:"restaurant_id"`
	RestaurantName string    `json:"restaurant_name"`
	EventName      string    `json:"event_name"`
	ApprovedAt     time.Time `json:"approved_at"`
}

func (RestaurantApprovedPayload) GetEventName() string {
	return "restaurant.approved"
}

func newRestaurantApprovedPayload(e restaurant.RestaurantApproved) RestaurantApprovedPayload {
	payload := RestaurantApprovedPayload{
		RestaurantID:   e.RestaurantID,
		RestaurantName: e.RestaurantName,
		ApprovedAt:     e.ApprovedAt,
	}
	payload.EventName = payload.GetEventName()

	return payload
}

type RestaurantLaunchedPayload struct {
	RestaurantID   uuid.UUID `json:"restaurant_id"`
	RestaurantName string    `json:"restaurant_name"`
	EventName      string    `json:"event_name"`
	LaunchedAt     time.Time `json:"launched_at"`
}

func (RestaurantLaunchedPayload) GetEventName() string {
	return "restaurant.launched"
}

func newRestaurantLaunchedPayload(e restaurant.RestaurantLaunched) RestaurantLaunchedPayload {
	payload := RestaurantLaunchedPayload{
		RestaurantID:   e.RestaurantID,
		RestaurantName: e.RestaurantName,
		LaunchedAt:     e.LaunchedAt,
	}
	payload.EventName = payload.GetEventName()

	return payload
}
