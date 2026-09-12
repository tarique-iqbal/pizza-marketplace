package order

import (
	"time"

	"github.com/google/uuid"

	"order-service/internal/domain/order"
	"order-service/internal/shared/money"
)

type AddressInput struct {
	House      string `json:"house" binding:"required"`
	Street     string `json:"street" binding:"required"`
	PostalCode string `json:"postalCode" binding:"required"`
	City       string `json:"city" binding:"required"`
}

type CheckoutRequest struct {
	Fulfillment     string        `json:"fulfillment" binding:"required,oneof=delivery pickup"`
	DeliveryAddress *AddressInput `json:"deliveryAddress" binding:"required_if=Fulfillment delivery"`
	ContactPhone    *string       `json:"contactPhone"`
}

type CheckoutResponse struct {
	OrderID     uuid.UUID `json:"orderId"`
	CheckoutURL string    `json:"checkoutUrl"`
}

type OrderItemView struct {
	ItemID          uuid.UUID   `json:"itemId"`
	PizzaID         uuid.UUID   `json:"pizzaId"`
	PizzaName       string      `json:"pizzaName"`
	SizeID          uuid.UUID   `json:"sizeId"`
	DiameterCm      int16       `json:"diameterCm"`
	ExtraToppingIDs []uuid.UUID `json:"extraToppingIds"`
	Quantity        int16       `json:"quantity"`
	UnitPrice       money.Money `json:"unitPrice"`
	LineTotal       money.Money `json:"lineTotal"`
}

type OrderResponse struct {
	OrderID         uuid.UUID         `json:"orderId"`
	Status          order.OrderStatus `json:"status"`
	Fulfillment     order.Fulfillment `json:"fulfillment"`
	ContactEmail    string            `json:"contactEmail"`
	ContactPhone    *string           `json:"contactPhone,omitempty"`
	DeliveryAddress *order.Address    `json:"deliveryAddress,omitempty"`
	Items           []OrderItemView   `json:"items"`
	Subtotal        money.Money       `json:"subtotal"`
	DeliveryFee     money.Money       `json:"deliveryFee"`
	Total           money.Money       `json:"total"`
	Currency        string            `json:"currency"`
	PlacedAt        time.Time         `json:"placedAt"`
	ConfirmedAt     *time.Time        `json:"confirmedAt,omitempty"`
	ReadyAt         *time.Time        `json:"readyAt,omitempty"`
	CompletedAt     *time.Time        `json:"completedAt,omitempty"`
	CancelledAt     *time.Time        `json:"cancelledAt,omitempty"`
}

type ListOrdersResponse struct {
	Orders     []OrderResponse `json:"orders"`
	NextCursor string          `json:"nextCursor,omitempty"`
}
