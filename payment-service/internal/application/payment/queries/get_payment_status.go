package queries

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"payment-service/internal/domain/payment"
	apperr "payment-service/internal/shared/errors"
)

type GetPaymentStatus struct {
	paymentRepo payment.PaymentRepository
}

func NewGetPaymentStatus(paymentRepo payment.PaymentRepository) *GetPaymentStatus {
	return &GetPaymentStatus{paymentRepo: paymentRepo}
}

func (uc *GetPaymentStatus) Execute(ctx context.Context, paymentID uuid.UUID) (payment.PaymentStatus, error) {
	p, err := uc.paymentRepo.FindByID(ctx, paymentID)
	if err != nil {
		return "", fmt.Errorf("failed to find payment: %w", err)
	}
	if p == nil {
		return "", apperr.ErrNotFound
	}

	return p.Status, nil
}
