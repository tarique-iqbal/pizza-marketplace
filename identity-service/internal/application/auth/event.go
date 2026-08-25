package auth

import "time"

type EmailVerificationCreated struct {
	Email      string    `json:"email"`
	Code       string    `json:"code"`
	EventName  string    `json:"event_name"`
	OccurredAt time.Time `json:"occurred_at"`
}

func (e EmailVerificationCreated) GetEventName() string {
	return "email.verification_created"
}
