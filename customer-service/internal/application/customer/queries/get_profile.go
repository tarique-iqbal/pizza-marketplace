package queries

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	customerapp "customer-service/internal/application/customer"
	"customer-service/internal/domain/customer"
	apperr "customer-service/internal/shared/errors"
)

type GetProfile struct {
	customerRepo customer.CustomerRepository
}

func NewGetProfile(customerRepo customer.CustomerRepository) *GetProfile {
	return &GetProfile{customerRepo: customerRepo}
}

func (uc *GetProfile) Execute(
	ctx context.Context,
	customerID uuid.UUID,
) (customerapp.GetProfileResponse, error) {
	c, err := uc.customerRepo.FindByID(ctx, customerID)
	if err != nil {
		return customerapp.GetProfileResponse{}, fmt.Errorf("failed to find customer: %w", err)
	}
	if c == nil {
		return customerapp.GetProfileResponse{}, apperr.ErrNotFound
	}

	return customerapp.ToProfileResponse(c), nil
}
