package commands

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"customer-service/internal/domain/customer"
)

type Delete struct {
	addressRepo customer.AddressRepository
}

func NewDelete(addressRepo customer.AddressRepository) *Delete {
	return &Delete{addressRepo: addressRepo}
}

func (cmd *Delete) Execute(ctx context.Context, customerID, addressID uuid.UUID) error {
	if err := cmd.addressRepo.Delete(ctx, addressID, customerID); err != nil {
		return fmt.Errorf("failed to delete address: %w", err)
	}

	return nil
}
