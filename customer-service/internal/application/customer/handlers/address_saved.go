package handlers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"customer-service/internal/domain/customer"
	logobs "customer-service/internal/infrastructure/observability/logger"
)

type addressSavedPayload struct {
	CustomerID uuid.UUID `json:"customer_id"`
	House      string    `json:"house"`
	Street     string    `json:"street"`
	City       string    `json:"city"`
	PostalCode string    `json:"postal_code"`
}

type AddressSaved struct {
	addressRepo customer.AddressRepository
}

func NewAddressSaved(addressRepo customer.AddressRepository) *AddressSaved {
	return &AddressSaved{addressRepo: addressRepo}
}

func (h *AddressSaved) Handle(event customer.EventPayload) error {
	var payload addressSavedPayload
	if err := json.Unmarshal(event.Data, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal %s payload: %w", event.Name, err)
	}

	ctx := context.Background()

	exists, err := h.addressRepo.ExistsForCustomer(
		ctx, payload.CustomerID, payload.House, payload.Street, payload.City, payload.PostalCode,
	)
	if err != nil {
		return fmt.Errorf("failed to check existing address: %w", err)
	}
	if exists {
		logobs.FromContext(ctx).Warn(
			"order.address_saved no-op, address already saved",
			"customer_id", payload.CustomerID,
		)
		return nil
	}

	existing, err := h.addressRepo.ListByCustomer(ctx, payload.CustomerID)
	if err != nil {
		return fmt.Errorf("failed to list customer addresses: %w", err)
	}

	a, err := customer.NewAddress(
		payload.CustomerID, payload.House, payload.Street, payload.City, payload.PostalCode,
		len(existing) == 0,
	)
	if err != nil {
		return fmt.Errorf("failed to build address: %w", err)
	}

	if err := h.addressRepo.Create(ctx, a); err != nil {
		return fmt.Errorf("failed to create address: %w", err)
	}

	return nil
}
