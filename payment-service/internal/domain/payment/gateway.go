package payment

import (
	"context"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type CreatePaymentRequest struct {
	RestaurantID uuid.UUID
	Amount       decimal.Decimal
	Currency     string
	RedirectURL  string
	WebhookURL   string
}

type CreatePaymentResult struct {
	GatewayPaymentID string
	CheckoutURL      string
}

type PaymentStatusResult struct {
	Status PaymentStatus
	Reason string
}

type PaymentGateway interface {
	CreatePayment(ctx context.Context, req CreatePaymentRequest) (CreatePaymentResult, error)
	CancelPayment(ctx context.Context, gatewayPaymentID string) error
	GetStatus(ctx context.Context, gatewayPaymentID string) (PaymentStatusResult, error)
}
