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
