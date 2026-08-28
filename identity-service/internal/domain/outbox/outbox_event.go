package outbox

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type OutboxStatus string

const (
	StatusPending    OutboxStatus = "pending"
	StatusProcessing OutboxStatus = "processing"
	StatusProcessed  OutboxStatus = "processed"
	StatusFailed     OutboxStatus = "failed"

	EventRestaurantInitiated      = "restaurant.initiated"
	EventUserRegistered           = "user.registered"
	EventEmailVerificationCreated = "email.verification_created"
)

type OutboxEvent struct {
	ID            int64          `gorm:"primaryKey;autoIncrement"`
	AggregateID   uuid.UUID      `gorm:"type:uuid;not null;index"`
	EventName     string         `gorm:"type:varchar(64);not null"`
	Payload       datatypes.JSON `gorm:"type:jsonb;not null"`
	Status        OutboxStatus   `gorm:"type:varchar(16);not null;default:'pending';index"`
	Attempts      int            `gorm:"not null;default:0"`
	LockedUntil   *time.Time     `gorm:"type:timestamptz"`
	NextAttemptAt *time.Time     `gorm:"type:timestamptz;index"`
	LastError     *string        `gorm:"type:text"`
	CreatedAt     time.Time      `gorm:"type:timestamptz;not null"`
	ProcessedAt   *time.Time     `gorm:"type:timestamptz"`
}

func (OutboxEvent) TableName() string {
	return "outbox_events"
}

func NewOutboxEvent(aggregateID uuid.UUID, name string, payload []byte) OutboxEvent {
	return OutboxEvent{
		AggregateID: aggregateID,
		EventName:   name,
		Payload:     datatypes.JSON(payload),
		Status:      StatusPending,
		CreatedAt:   time.Now().UTC(),
	}
}
