package persistence

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"restaurant-service/internal/domain/restaurant"
)

type ToppingRepository struct {
	db *gorm.DB
}

func NewToppingRepository(db *gorm.DB) restaurant.ToppingRepository {
	return &ToppingRepository{db: db}
}

func (repo *ToppingRepository) List(ctx context.Context) ([]restaurant.Topping, error) {
	var toppings []restaurant.Topping

	err := repo.db.WithContext(ctx).
		Order("name").
		Find(&toppings).Error

	return toppings, err
}

func (repo *ToppingRepository) FindByID(
	ctx context.Context,
	toppingID uuid.UUID,
) (*restaurant.Topping, error) {
	var t restaurant.Topping

	err := repo.db.WithContext(ctx).
		Where("id = ?", toppingID).
		Take(&t).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	return &t, err
}
