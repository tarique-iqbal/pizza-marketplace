package persistence

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"payment-service/internal/domain/payment"
)

type PaymentRepository struct {
	db *gorm.DB
}

func NewPaymentRepository(db *gorm.DB) payment.PaymentRepository {
	return &PaymentRepository{db: db}
}

func (r *PaymentRepository) WithTx(tx *gorm.DB) payment.PaymentRepository {
	return &PaymentRepository{db: tx}
}

func (r *PaymentRepository) Create(ctx context.Context, p *payment.Payment) error {
	return r.db.WithContext(ctx).Create(p).Error
}

func (r *PaymentRepository) Update(ctx context.Context, p *payment.Payment) error {
	return r.db.WithContext(ctx).Save(p).Error
}

func (r *PaymentRepository) FindByID(ctx context.Context, id uuid.UUID) (*payment.Payment, error) {
	var p payment.Payment

	err := r.db.WithContext(ctx).First(&p, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &p, nil
}

func (r *PaymentRepository) FindBySubject(
	ctx context.Context,
	subjectType string,
	subjectID uuid.UUID,
) (*payment.Payment, error) {
	var p payment.Payment

	err := r.db.WithContext(ctx).
		First(&p, "subject_type = ? AND subject_id = ?", subjectType, subjectID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &p, nil
}

func (r *PaymentRepository) FindByGatewayPaymentID(
	ctx context.Context,
	gatewayPaymentID string,
) (*payment.Payment, error) {
	var p payment.Payment

	err := r.db.WithContext(ctx).
		First(&p, "gateway_payment_id = ?", gatewayPaymentID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &p, nil
}
