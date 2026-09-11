package order

import (
	"time"

	"github.com/google/uuid"
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
