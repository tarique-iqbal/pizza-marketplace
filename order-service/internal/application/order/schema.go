package order

import "github.com/google/uuid"

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
