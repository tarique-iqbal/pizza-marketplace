package persistence

import (
	"context"

	"gorm.io/gorm"

	"restaurant-service/internal/domain/restaurant"
)

type PizzaSizeRepository struct {
	db *gorm.DB
}

func NewPizzaSizeRepository(db *gorm.DB) restaurant.PizzaSizeRepository {
	return &PizzaSizeRepository{db: db}
}

func (repo *PizzaSizeRepository) List(ctx context.Context) ([]restaurant.PizzaSize, error) {
	var sizes []restaurant.PizzaSize

	err := repo.db.WithContext(ctx).
		Order("diameter_cm").
		Find(&sizes).Error

	return sizes, err
}
