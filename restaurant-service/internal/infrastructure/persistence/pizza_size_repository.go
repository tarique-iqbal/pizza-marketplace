package persistence

import (
	"context"

	"gorm.io/gorm"

	"restaurant-service/internal/domain/pizza"
)

type PizzaSizeRepository struct {
	db *gorm.DB
}

func NewPizzaSizeRepository(db *gorm.DB) pizza.PizzaSizeRepository {
	return &PizzaSizeRepository{db: db}
}

func (repo *PizzaSizeRepository) List(ctx context.Context) ([]pizza.PizzaSize, error) {
	var sizes []pizza.PizzaSize

	err := repo.db.WithContext(ctx).
		Order("diameter_cm").
		Find(&sizes).Error

	return sizes, err
}
