package customer

import (
	"time"

	"github.com/google/uuid"
)

type PhoneUpdatedPayload struct {
	CustomerID uuid.UUID `json:"customer_id"`
	Phone      string    `json:"phone"`
	EventName  string    `json:"event_name"`
	OccurredAt time.Time `json:"occurred_at"`
}

func (PhoneUpdatedPayload) GetEventName() string {
	return "customer.phone_updated"
}
