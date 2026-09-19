package persistence

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"customer-service/internal/domain/customer"
	apperr "customer-service/internal/shared/errors"
)

type AddressRepository struct {
	db *gorm.DB
}

func NewAddressRepository(db *gorm.DB) customer.AddressRepository {
	return &AddressRepository{db: db}
}

func (r *AddressRepository) WithTx(tx *gorm.DB) customer.AddressRepository {
	return &AddressRepository{db: tx}
}

func (r *AddressRepository) Create(ctx context.Context, a *customer.Address) error {
	return r.db.WithContext(ctx).Create(a).Error
}

func (r *AddressRepository) Delete(ctx context.Context, id, customerID uuid.UUID) error {
	result := r.db.WithContext(ctx).
		Where("id = ? AND customer_id = ?", id, customerID).
		Delete(&customer.Address{})

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return apperr.ErrNotFound
	}

	return nil
}

func (r *AddressRepository) FindByID(
	ctx context.Context,
	id, customerID uuid.UUID,
) (*customer.Address, error) {
	var a customer.Address

	err := r.db.WithContext(ctx).
		First(&a, "id = ? AND customer_id = ?", id, customerID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &a, nil
}

func (r *AddressRepository) ListByCustomer(
	ctx context.Context,
	customerID uuid.UUID,
) ([]customer.Address, error) {
	var addresses []customer.Address

	err := r.db.WithContext(ctx).
		Where("customer_id = ?", customerID).
		Order("is_default DESC, created_at ASC").
		Find(&addresses).Error

	return addresses, err
}

func (r *AddressRepository) UnsetDefault(ctx context.Context, customerID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&customer.Address{}).
		Where("customer_id = ? AND is_default = ?", customerID, true).
		Update("is_default", false).Error
}

func (r *AddressRepository) SetDefault(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).
		Model(&customer.Address{}).
		Where("id = ?", id).
		Update("is_default", true)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return apperr.ErrNotFound
	}

	return nil
}

func (r *AddressRepository) ExistsForCustomer(
	ctx context.Context,
	customerID uuid.UUID,
	house, street, city, postalCode string,
) (bool, error) {
	var count int64

	err := r.db.WithContext(ctx).
		Model(&customer.Address{}).
		Where(
			"customer_id = ? AND house = ? AND street = ? AND city = ? AND postal_code = ?",
			customerID, house, street, city, postalCode,
		).
		Count(&count).Error

	return count > 0, err
}
