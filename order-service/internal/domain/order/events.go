package order

import (
	"time"

	"github.com/google/uuid"
)

type DomainEvent interface {
	GetEventName() string
}

type OrderConfirmed struct {
	OrderID      uuid.UUID
	CustomerID   uuid.UUID
	RestaurantID uuid.UUID
	OccurredAt   time.Time
}

func (OrderConfirmed) GetEventName() string {
	return "order.confirmed"
}

type AddressSaved struct {
	CustomerID uuid.UUID
	House      string
	Street     string
	City       string
	PostalCode string
	OccurredAt time.Time
}

func (AddressSaved) GetEventName() string {
	return "order.address_saved"
}
