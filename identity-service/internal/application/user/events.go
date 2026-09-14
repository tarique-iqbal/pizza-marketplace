package user

import (
	"time"

	"github.com/google/uuid"

	"identity-service/internal/domain/user"
)

type UserRegisteredPayload struct {
	UserID     uuid.UUID `json:"user_id"`
	Email      string    `json:"email"`
	FirstName  string    `json:"first_name"`
	Role       string    `json:"role"`
	EventName  string    `json:"event_name"`
	OccurredAt time.Time `json:"occurred_at"`
}

func (UserRegisteredPayload) GetEventName() string {
	return "user.registered"
}

func newUserRegisteredPayload(e user.UserRegistered) UserRegisteredPayload {
	payload := UserRegisteredPayload{
		UserID:     e.UserID,
		Email:      e.Email,
		FirstName:  e.FirstName,
		Role:       e.Role,
		OccurredAt: e.OccurredAt,
	}
	payload.EventName = payload.GetEventName()
	return payload
}

type RestaurantInitiatedPayload struct {
	RestaurantID uuid.UUID `json:"restaurant_id"`
	OwnerID      uuid.UUID `json:"owner_id"`
	BusinessName string    `json:"business_name"`
	VATNumber    string    `json:"vat_number"`
	EventName    string    `json:"event_name"`
	OccurredAt   time.Time `json:"occurred_at"`
}

func (RestaurantInitiatedPayload) GetEventName() string {
	return "restaurant.initiated"
}

func newRestaurantInitiatedPayload(e user.RestaurantInitiated) RestaurantInitiatedPayload {
	payload := RestaurantInitiatedPayload{
		RestaurantID: e.RestaurantID,
		OwnerID:      e.OwnerID,
		BusinessName: e.BusinessName,
		VATNumber:    e.VATNumber,
		OccurredAt:   e.OccurredAt,
	}
	payload.EventName = payload.GetEventName()
	return payload
}
