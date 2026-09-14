package auth

import (
	"time"

	"github.com/google/uuid"
)

type EmailVerification struct {
	ID           uuid.UUID     `gorm:"type:uuid;primaryKey"`
	Email        string        `gorm:"size:255;not null;index"`
	Code         string        `gorm:"type:char(6);not null"`
	IsUsed       bool          `gorm:"default:false"`
	AttemptCount int16         `gorm:"not null"`
	ExpiresAt    time.Time     `gorm:"type:timestamptz;not null"`
	CreatedAt    time.Time     `gorm:"type:timestamptz;autoCreateTime"`
	events       []DomainEvent `gorm:"-"`
}

func (EmailVerification) TableName() string {
	return "email_verifications"
}

func (ev *EmailVerification) MarkCreated() {
	ev.events = append(ev.events, EmailVerificationCreated{
		Email:      ev.Email,
		Code:       ev.Code,
		OccurredAt: time.Now().UTC(),
	})
}

func (ev *EmailVerification) PullEvents() []DomainEvent {
	events := ev.events
	ev.events = nil
	return events
}
