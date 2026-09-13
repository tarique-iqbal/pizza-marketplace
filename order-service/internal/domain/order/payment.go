package order

import (
	"context"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type CreatePaymentRequest struct {
	OrderID      uuid.UUID
	RestaurantID uuid.UUID
	CustomerID   uuid.UUID
	Amount       decimal.Decimal
	Currency     string
	RedirectURL  string
}

type CreatePaymentResult struct {
	PaymentID   string
	CheckoutURL string
}

type PaymentStatus string

const (
	PaymentStatusPending   PaymentStatus = "pending"
	PaymentStatusSucceeded PaymentStatus = "succeeded"
	PaymentStatusFailed    PaymentStatus = "failed"
)

type PaymentProvider interface {
	CreatePayment(ctx context.Context, req CreatePaymentRequest) (CreatePaymentResult, error)
	CancelPayment(ctx context.Context, paymentID string) error
	GetPaymentStatus(ctx context.Context, paymentID string) (PaymentStatus, error)
}
