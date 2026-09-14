package auth

import (
	"time"

	"identity-service/internal/domain/auth"
)

type EmailVerificationCreatedPayload struct {
	Email      string    `json:"email"`
	Code       string    `json:"code"`
	EventName  string    `json:"event_name"`
	OccurredAt time.Time `json:"occurred_at"`
}

func (EmailVerificationCreatedPayload) GetEventName() string {
	return "email.verification_created"
}

func newEmailVerificationCreatedPayload(e auth.EmailVerificationCreated) EmailVerificationCreatedPayload {
	payload := EmailVerificationCreatedPayload{
		Email:      e.Email,
		Code:       e.Code,
		OccurredAt: e.OccurredAt,
	}
	payload.EventName = payload.GetEventName()
	return payload
}
