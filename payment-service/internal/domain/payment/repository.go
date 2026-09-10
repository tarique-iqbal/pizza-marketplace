package payment

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PaymentRepository interface {
	WithTx(tx *gorm.DB) PaymentRepository
	Create(ctx context.Context, p *Payment) error
	Update(ctx context.Context, p *Payment) error
	FindBySubject(ctx context.Context, subjectType string, subjectID uuid.UUID) (*Payment, error)
	FindByGatewayPaymentID(ctx context.Context, gatewayPaymentID string) (*Payment, error)
}
