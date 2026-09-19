package readmodel

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"order-service/internal/domain/readmodel"
)

// phoneUpdatedPayload mirrors customer-service's wire shape: a local, independent copy.
type phoneUpdatedPayload struct {
	CustomerID uuid.UUID `json:"customer_id"`
	Phone      string    `json:"phone"`
}

type UpdateCustomerPhone struct {
	customerRepo readmodel.CustomerRepository
}

func NewUpdateCustomerPhone(customerRepo readmodel.CustomerRepository) *UpdateCustomerPhone {
	return &UpdateCustomerPhone{customerRepo: customerRepo}
}

func (h *UpdateCustomerPhone) Handle(event readmodel.EventPayload) error {
	var payload phoneUpdatedPayload
	if err := json.Unmarshal(event.Data, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal %s payload: %w", event.Name, err)
	}

	err := h.customerRepo.UpdatePhone(context.Background(), payload.CustomerID, payload.Phone)
	if err != nil {
		return fmt.Errorf("failed to update customer phone: %w", err)
	}

	return nil
}
