package payment

import (
	"time"

	"github.com/google/uuid"
)

type PaymentSucceededPayload struct {
	PaymentID    uuid.UUID `json:"payment_id"`
	SubjectType  string    `json:"subject_type"`
	SubjectID    uuid.UUID `json:"subject_id"`
	RestaurantID uuid.UUID `json:"restaurant_id"`
	Amount       string    `json:"amount"`
	Currency     string    `json:"currency"`
	EventName    string    `json:"event_name"`
	OccurredAt   time.Time `json:"occurred_at"`
}

func (PaymentSucceededPayload) GetEventName() string {
	return "payment.succeeded"
}

type PaymentFailedPayload struct {
	PaymentID    uuid.UUID `json:"payment_id"`
	SubjectType  string    `json:"subject_type"`
	SubjectID    uuid.UUID `json:"subject_id"`
	RestaurantID uuid.UUID `json:"restaurant_id"`
	Amount       string    `json:"amount"`
	Currency     string    `json:"currency"`
	Reason       string    `json:"reason"`
	EventName    string    `json:"event_name"`
	OccurredAt   time.Time `json:"occurred_at"`
}

func (PaymentFailedPayload) GetEventName() string {
	return "payment.failed"
}
