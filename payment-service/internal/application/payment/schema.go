package payment

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type CreatePaymentRequest struct {
	SubjectType  string
	SubjectID    uuid.UUID
	RestaurantID uuid.UUID
	CustomerID   uuid.UUID
	Amount       decimal.Decimal
	Currency     string
	RedirectURL  string
}

type CreatePaymentResponse struct {
	PaymentID   uuid.UUID
	CheckoutURL string
	Status      string
}
