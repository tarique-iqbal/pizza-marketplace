package order

import (
	"time"

	"github.com/google/uuid"

	"order-service/internal/domain/order"
)

type OrderItemPayload struct {
	Name     string `json:"name"`
	Quantity int16  `json:"quantity"`
}

type OrderConfirmedPayload struct {
	OrderID        uuid.UUID          `json:"order_id"`
	RestaurantID   uuid.UUID          `json:"restaurant_id"`
	RestaurantName string             `json:"restaurant_name"`
	CustomerEmail  string             `json:"customer_email"`
	OwnerEmail     string             `json:"owner_email"`
	Items          []OrderItemPayload `json:"items"`
	Total          string             `json:"total"`
	Currency       string             `json:"currency"`
	ConfirmedAt    time.Time          `json:"confirmed_at"`
}

func (OrderConfirmedPayload) GetEventName() string {
	return "order.confirmed"
}

type AddressSavedPayload struct {
	CustomerID uuid.UUID `json:"customer_id"`
	House      string    `json:"house"`
	Street     string    `json:"street"`
	City       string    `json:"city"`
	PostalCode string    `json:"postal_code"`
	EventName  string    `json:"event_name"`
	OccurredAt time.Time `json:"occurred_at"`
}

func (AddressSavedPayload) GetEventName() string {
	return "order.address_saved"
}

func newAddressSavedPayload(e order.AddressSaved) AddressSavedPayload {
	payload := AddressSavedPayload{
		CustomerID: e.CustomerID,
		House:      e.House,
		Street:     e.Street,
		City:       e.City,
		PostalCode: e.PostalCode,
		OccurredAt: e.OccurredAt,
	}
	payload.EventName = payload.GetEventName()

	return payload
}
