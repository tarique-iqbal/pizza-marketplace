package persistence

import (
	"context"
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"order-service/internal/domain/order"
)

type GeocodeRepository struct {
	db *gorm.DB
}

func NewGeocodeRepository(db *gorm.DB) order.GeocodeRepository {
	return &GeocodeRepository{db: db}
}

func (r *GeocodeRepository) FindByHash(ctx context.Context, hash string) (*order.GeocodeEntry, error) {
	var entry order.GeocodeEntry

	err := r.db.WithContext(ctx).First(&entry, "address_hash = ?", hash).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &entry, nil
}

func (r *GeocodeRepository) Create(ctx context.Context, entry order.GeocodeEntry) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "address_hash"}},
		DoNothing: true,
	}).Create(&entry).Error
}
