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
	Email          string
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

type RestaurantUpdated struct {
	RestaurantID uuid.UUID
	UpdatedAt    time.Time
}

func (RestaurantUpdated) GetEventName() string {
	return "restaurant.updated"
}

type PizzaUpdated struct {
	RestaurantID uuid.UUID
	UpdatedAt    time.Time
}

func (PizzaUpdated) GetEventName() string {
	return "restaurant.pizza_updated"
}
