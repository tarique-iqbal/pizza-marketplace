package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	customerapp "customer-service/internal/application/customer"
	"customer-service/internal/domain/customer"
	"customer-service/internal/domain/outbox"
	apperr "customer-service/internal/shared/errors"
)

type UpdatePhone struct {
	db           *gorm.DB
	customerRepo customer.CustomerRepository
	outboxRepo   outbox.OutboxRepository
}

func NewUpdatePhone(
	db *gorm.DB,
	customerRepo customer.CustomerRepository,
	outboxRepo outbox.OutboxRepository,
) *UpdatePhone {
	return &UpdatePhone{
		db:           db,
		customerRepo: customerRepo,
		outboxRepo:   outboxRepo,
	}
}

func (uc *UpdatePhone) Execute(
	ctx context.Context,
	customerID uuid.UUID,
	input customerapp.UpdatePhoneRequest,
) (customerapp.GetProfileResponse, error) {
	c, err := uc.customerRepo.FindByID(ctx, customerID)
	if err != nil {
		return customerapp.GetProfileResponse{}, fmt.Errorf("failed to find customer: %w", err)
	}
	if c == nil {
		return customerapp.GetProfileResponse{}, apperr.ErrNotFound
	}

	c.SetPhone(input.Phone)

	updated := customerapp.PhoneUpdatedPayload{
		CustomerID: c.ID,
		Phone:      input.Phone,
		OccurredAt: time.Now().UTC(),
	}
	updated.EventName = updated.GetEventName()

	payload, err := json.Marshal(updated)
	if err != nil {
		return customerapp.GetProfileResponse{}, fmt.Errorf("failed to encode event payload: %w", err)
	}

	err = uc.db.Transaction(func(tx *gorm.DB) error {
		if err := uc.customerRepo.WithTx(tx).Update(ctx, c); err != nil {
			return fmt.Errorf("failed to update customer: %w", err)
		}

		event := outbox.NewOutboxEvent(c.ID, updated.EventName, payload)
		if err := uc.outboxRepo.WithTx(tx).Create(ctx, &event); err != nil {
			return fmt.Errorf("failed to create outbox event: %w", err)
		}

		return nil
	})
	if err != nil {
		return customerapp.GetProfileResponse{}, err
	}

	return customerapp.ToProfileResponse(c), nil
}
