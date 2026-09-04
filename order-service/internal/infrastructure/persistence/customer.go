package persistence

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"order-service/internal/domain/readmodel"
	apperr "order-service/internal/shared/errors"
)

type CustomerRepository struct {
	db *gorm.DB
}

func NewCustomerRepository(db *gorm.DB) readmodel.CustomerRepository {
	return &CustomerRepository{db: db}
}

func (r *CustomerRepository) FindByID(ctx context.Context, id uuid.UUID) (*readmodel.Customer, error) {
	var customer readmodel.Customer

	err := r.db.WithContext(ctx).First(&customer, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperr.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return &customer, nil
}

// Upsert is redelivery-safe only — no updatedAt guard.
func (r *CustomerRepository) Upsert(ctx context.Context, customer readmodel.Customer) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoNothing: true,
	}).Create(&customer).Error
}
