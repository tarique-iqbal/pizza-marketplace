package user

import (
	"time"

	"github.com/google/uuid"
)

type DomainEvent interface {
	GetEventName() string
}

type UserRegistered struct {
	UserID     uuid.UUID
	Email      string
	FirstName  string
	LastName   string
	Role       string
	OccurredAt time.Time
}

func (UserRegistered) GetEventName() string {
	return "user.registered"
}

type RestaurantInitiated struct {
	RestaurantID uuid.UUID
	OwnerID      uuid.UUID
	BusinessName string
	VATNumber    string
	OccurredAt   time.Time
}

func (RestaurantInitiated) GetEventName() string {
	return "restaurant.initiated"
}
