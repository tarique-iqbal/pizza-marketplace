package user

import "time"

type UserRegistered struct {
	Email      string    `json:"email"`
	FirstName  string    `json:"first_name"`
	Role       string    `json:"role"`
	EventName  string    `json:"event_name"`
	OccurredAt time.Time `json:"occurred_at"`
}

func (e UserRegistered) GetEventName() string {
	return "user.registered"
}
