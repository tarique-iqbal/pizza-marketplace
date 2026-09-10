package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"

	paymentapp "payment-service/internal/application/payment"
	"payment-service/internal/domain/outbox"
	"payment-service/internal/domain/payment"
)

type HandleMollieWebhook struct {
	db         *gorm.DB
	repo       payment.PaymentRepository
	gateway    payment.PaymentGateway
	outboxRepo outbox.OutboxRepository
}

func NewHandleMollieWebhook(
	db *gorm.DB,
	repo payment.PaymentRepository,
	gateway payment.PaymentGateway,
	outboxRepo outbox.OutboxRepository,
) *HandleMollieWebhook {
	return &HandleMollieWebhook{
		db:         db,
		repo:       repo,
		gateway:    gateway,
		outboxRepo: outboxRepo,
	}
}

func (uc *HandleMollieWebhook) Execute(ctx context.Context, gatewayPaymentID string) error {
	p, err := uc.repo.FindByGatewayPaymentID(ctx, gatewayPaymentID)
	if err != nil {
		return fmt.Errorf("failed to look up payment: %w", err)
	}
	if p == nil {
		return nil
	}
	if p.Status != payment.StatusPending {
		return nil
	}

	result, err := uc.gateway.GetStatus(ctx, gatewayPaymentID)
	if err != nil {
		return fmt.Errorf("failed to check payment status: %w", err)
	}

	var eventName string
	var payload []byte

	switch result.Status {
	case payment.StatusSucceeded:
		if err := p.MarkSucceeded(); err != nil {
			return fmt.Errorf("failed to mark payment succeeded: %w", err)
		}

		succeeded := paymentapp.PaymentSucceededPayload{
			PaymentID:    p.ID,
			SubjectType:  p.SubjectType,
			SubjectID:    p.SubjectID,
			RestaurantID: p.RestaurantID,
			Amount:       p.Amount.StringFixed(2),
			Currency:     p.Currency,
			OccurredAt:   time.Now().UTC(),
		}
		succeeded.EventName = succeeded.GetEventName()
		eventName = succeeded.EventName

		payload, err = json.Marshal(succeeded)
		if err != nil {
			return fmt.Errorf("failed to encode event payload: %w", err)
		}

	case payment.StatusFailed:
		if err := p.MarkFailed(result.Reason); err != nil {
			return fmt.Errorf("failed to mark payment failed: %w", err)
		}

		failed := paymentapp.PaymentFailedPayload{
			PaymentID:    p.ID,
			SubjectType:  p.SubjectType,
			SubjectID:    p.SubjectID,
			RestaurantID: p.RestaurantID,
			Amount:       p.Amount.StringFixed(2),
			Currency:     p.Currency,
			Reason:       result.Reason,
			OccurredAt:   time.Now().UTC(),
		}
		failed.EventName = failed.GetEventName()
		eventName = failed.EventName

		payload, err = json.Marshal(failed)
		if err != nil {
			return fmt.Errorf("failed to encode event payload: %w", err)
		}

	default:
		return nil
	}

	return uc.db.Transaction(func(tx *gorm.DB) error {
		if err := uc.repo.WithTx(tx).Update(ctx, p); err != nil {
			return fmt.Errorf("failed to update payment: %w", err)
		}

		event := outbox.NewOutboxEvent(p.ID, eventName, payload)
		if err := uc.outboxRepo.WithTx(tx).Create(ctx, &event); err != nil {
			return fmt.Errorf("failed to create outbox event: %w", err)
		}

		return nil
	})
}
