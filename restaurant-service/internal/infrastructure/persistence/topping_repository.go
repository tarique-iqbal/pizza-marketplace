package persistence

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"restaurant-service/internal/domain/topping"
)

type ToppingRepository struct {
	db *gorm.DB
}

func NewToppingRepository(db *gorm.DB) topping.ToppingRepository {
	return &ToppingRepository{db: db}
}

func (repo *ToppingRepository) List(ctx context.Context) ([]topping.Topping, error) {
	var toppings []topping.Topping

	err := repo.db.WithContext(ctx).
		Order("name").
		Find(&toppings).Error

	return toppings, err
}

func (repo *ToppingRepository) FindByID(
	ctx context.Context,
	toppingID uuid.UUID,
) (*topping.Topping, error) {
	var t topping.Topping

	err := repo.db.WithContext(ctx).
		Where("id = ?", toppingID).
		Take(&t).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	return &t, err
}
