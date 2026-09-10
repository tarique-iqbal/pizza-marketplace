package payment

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type PaymentStatus string

const (
	StatusPending   PaymentStatus = "pending"
	StatusSucceeded PaymentStatus = "succeeded"
	StatusFailed    PaymentStatus = "failed"
)

type Payment struct {
	ID               uuid.UUID       `gorm:"type:uuid;primaryKey"`
	SubjectType      string          `gorm:"size:32;not null"`
	SubjectID        uuid.UUID       `gorm:"type:uuid;not null"`
	RestaurantID     uuid.UUID       `gorm:"type:uuid;not null"`
	CustomerID       uuid.UUID       `gorm:"type:uuid;not null"`
	Amount           decimal.Decimal `gorm:"type:numeric(10,2);not null"`
	Currency         string          `gorm:"size:3;not null"`
	PlatformFee      decimal.Decimal `gorm:"type:numeric(10,2);not null;default:0"`
	Status           PaymentStatus   `gorm:"type:payment_status_enum;not null;default:'pending'"`
	Gateway          string          `gorm:"size:32;not null"`
	GatewayPaymentID *string         `gorm:"size:64"`
	FailureReason    *string         `gorm:"size:500"`
	CreatedAt        time.Time       `gorm:"type:timestamptz;not null;autoCreateTime"`
	UpdatedAt        *time.Time      `gorm:"type:timestamptz;autoUpdateTime;default:null"`
}

func (Payment) TableName() string {
	return "payments"
}

func NewPayment(
	id, subjectID, restaurantID, customerID uuid.UUID,
	subjectType string,
	amount decimal.Decimal,
	currency string,
	platformFee decimal.Decimal,
	gateway string,
) *Payment {
	return &Payment{
		ID:           id,
		SubjectType:  subjectType,
		SubjectID:    subjectID,
		RestaurantID: restaurantID,
		CustomerID:   customerID,
		Amount:       amount,
		Currency:     currency,
		PlatformFee:  platformFee,
		Status:       StatusPending,
		Gateway:      gateway,
	}
}

func (p *Payment) AttachGatewayReference(gatewayPaymentID string) {
	p.GatewayPaymentID = &gatewayPaymentID
}

func (p *Payment) MarkSucceeded() error {
	if p.Status != StatusPending {
		return ErrInvalidStatusTransition
	}

	p.Status = StatusSucceeded

	return nil
}

func (p *Payment) MarkFailed(reason string) error {
	if p.Status != StatusPending {
		return ErrInvalidStatusTransition
	}

	p.Status = StatusFailed
	p.FailureReason = &reason

	return nil
}
