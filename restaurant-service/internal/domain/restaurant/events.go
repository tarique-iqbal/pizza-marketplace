package restaurant

import (
	"time"

	"github.com/google/uuid"
)

type DomainEvent interface {
	GetEventName() string
}

type RestaurantReadyForReview struct {
	RestaurantID   uuid.UUID
	RestaurantName string
	ReadyAt        time.Time
}

func (RestaurantReadyForReview) GetEventName() string {
	return "restaurant.ready_for_review"
}

type RestaurantApproved struct {
	RestaurantID   uuid.UUID
	RestaurantName string
	ApprovedAt     time.Time
}

func (RestaurantApproved) GetEventName() string {
	return "restaurant.approved"
}

type RestaurantLaunched struct {
	RestaurantID   uuid.UUID
	RestaurantName string
	LaunchedAt     time.Time
}

func (RestaurantLaunched) GetEventName() string {
	return "restaurant.launched"
}
