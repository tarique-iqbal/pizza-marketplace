package persistence

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"customer-service/internal/domain/customer"
)

type CustomerRepository struct {
	db *gorm.DB
}

func NewCustomerRepository(db *gorm.DB) customer.CustomerRepository {
	return &CustomerRepository{db: db}
}

func (r *CustomerRepository) WithTx(tx *gorm.DB) customer.CustomerRepository {
	return &CustomerRepository{db: tx}
}

// Upsert is redelivery-safe only: user.registered fires exactly once per user, ever.
func (r *CustomerRepository) Upsert(ctx context.Context, c customer.Customer) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoNothing: true,
	}).Create(&c).Error
}

func (r *CustomerRepository) Update(ctx context.Context, c *customer.Customer) error {
	return r.db.WithContext(ctx).Save(c).Error
}

func (r *CustomerRepository) FindByID(ctx context.Context, id uuid.UUID) (*customer.Customer, error) {
	var c customer.Customer

	err := r.db.WithContext(ctx).First(&c, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &c, nil
}
