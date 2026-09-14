package auth

import "time"

type DomainEvent interface {
	GetEventName() string
}

type EmailVerificationCreated struct {
	Email      string
	Code       string
	OccurredAt time.Time
}

func (EmailVerificationCreated) GetEventName() string {
	return "email.verification_created"
}
