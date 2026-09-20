package queries

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	addressapp "customer-service/internal/application/address"
	"customer-service/internal/domain/customer"
)

type List struct {
	addressRepo customer.AddressRepository
}

func NewList(addressRepo customer.AddressRepository) *List {
	return &List{addressRepo: addressRepo}
}

func (uc *List) Execute(
	ctx context.Context,
	customerID uuid.UUID,
) ([]addressapp.AddressResponse, error) {
	addresses, err := uc.addressRepo.ListByCustomer(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("failed to list addresses: %w", err)
	}

	return addressapp.ToAddressResponses(addresses), nil
}
