package commands

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	addressapp "customer-service/internal/application/address"
	"customer-service/internal/domain/customer"
	apperr "customer-service/internal/shared/errors"
)

type SetDefault struct {
	db          *gorm.DB
	addressRepo customer.AddressRepository
}

func NewSetDefault(db *gorm.DB, addressRepo customer.AddressRepository) *SetDefault {
	return &SetDefault{db: db, addressRepo: addressRepo}
}

func (uc *SetDefault) Execute(
	ctx context.Context,
	customerID, addressID uuid.UUID,
) (addressapp.AddressResponse, error) {
	a, err := uc.addressRepo.FindByID(ctx, addressID, customerID)
	if err != nil {
		return addressapp.AddressResponse{}, fmt.Errorf("failed to find address: %w", err)
	}
	if a == nil {
		return addressapp.AddressResponse{}, apperr.ErrNotFound
	}

	err = uc.db.Transaction(func(tx *gorm.DB) error {
		txRepo := uc.addressRepo.WithTx(tx)

		if err := txRepo.UnsetDefault(ctx, customerID); err != nil {
			return fmt.Errorf("failed to unset current default: %w", err)
		}

		if err := txRepo.SetDefault(ctx, addressID); err != nil {
			return fmt.Errorf("failed to set default: %w", err)
		}

		return nil
	})
	if err != nil {
		return addressapp.AddressResponse{}, err
	}

	a.IsDefault = true

	return addressapp.ToAddressResponse(*a), nil
}
