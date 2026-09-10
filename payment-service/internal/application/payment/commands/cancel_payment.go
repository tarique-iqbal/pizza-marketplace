package commands

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"payment-service/internal/domain/payment"
	logobs "payment-service/internal/infrastructure/observability/logger"
)

type CancelPayment struct {
	repo    payment.PaymentRepository
	gateway payment.PaymentGateway
}

func NewCancelPayment(repo payment.PaymentRepository, gateway payment.PaymentGateway) *CancelPayment {
	return &CancelPayment{repo: repo, gateway: gateway}
}

func (uc *CancelPayment) Execute(ctx context.Context, paymentID uuid.UUID) error {
	p, err := uc.repo.FindByID(ctx, paymentID)
	if err != nil {
		return fmt.Errorf("failed to look up payment: %w", err)
	}
	if p == nil || p.GatewayPaymentID == nil {
		return nil
	}

	if err := uc.gateway.CancelPayment(ctx, *p.GatewayPaymentID); err != nil {
		logobs.FromContext(ctx).Warn(
			"failed to cancel payment at gateway, ignoring (best-effort)",
			"payment_id", paymentID,
			"error", err,
		)
	}

	return nil
}
