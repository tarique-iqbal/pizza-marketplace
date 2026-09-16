package handlers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"customer-service/internal/domain/customer"
)

// userRegisteredPayload mirrors identity-service's wire shape — a local, independent copy.
type userRegisteredPayload struct {
	UserID    uuid.UUID `json:"user_id"`
	Email     string    `json:"email"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
}

type UserRegistered struct {
	customerRepo customer.CustomerRepository
}

func NewUserRegistered(customerRepo customer.CustomerRepository) *UserRegistered {
	return &UserRegistered{customerRepo: customerRepo}
}

// Handle stores every registered user, regardless of role — an owner can place orders too.
func (h *UserRegistered) Handle(event customer.EventPayload) error {
	var payload userRegisteredPayload
	if err := json.Unmarshal(event.Data, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal %s payload: %w", event.Name, err)
	}

	c := customer.Customer{
		ID:        payload.UserID,
		Email:     payload.Email,
		FirstName: payload.FirstName,
		LastName:  payload.LastName,
	}

	if err := h.customerRepo.Upsert(context.Background(), c); err != nil {
		return fmt.Errorf("failed to upsert customer: %w", err)
	}

	return nil
}
