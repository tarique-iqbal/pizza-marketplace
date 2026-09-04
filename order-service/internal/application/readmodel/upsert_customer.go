package readmodel

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"order-service/internal/domain/readmodel"
)

// userRegisteredPayload mirrors identity-service's wire shape — a local, independent copy.
type userRegisteredPayload struct {
	UserID    uuid.UUID `json:"user_id"`
	Email     string    `json:"email"`
	FirstName string    `json:"first_name"`
}

type UpsertCustomer struct {
	customerRepo readmodel.CustomerRepository
}

func NewUpsertCustomer(customerRepo readmodel.CustomerRepository) *UpsertCustomer {
	return &UpsertCustomer{customerRepo: customerRepo}
}

// Handle stores every registered user, regardless of role — an owner can place orders too.
func (h *UpsertCustomer) Handle(event readmodel.EventPayload) error {
	var payload userRegisteredPayload
	if err := json.Unmarshal(event.Data, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal %s payload: %w", event.Name, err)
	}

	customer := readmodel.Customer{
		ID:        payload.UserID,
		Email:     payload.Email,
		FirstName: payload.FirstName,
	}

	if err := h.customerRepo.Upsert(context.Background(), customer); err != nil {
		return fmt.Errorf("failed to upsert customer: %w", err)
	}

	return nil
}
