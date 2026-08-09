package persistence

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"restaurant-service/internal/domain/restaurant"
)

type PizzaRepository struct {
	db *gorm.DB
}

func NewPizzaRepository(db *gorm.DB) restaurant.PizzaRepository {
	return &PizzaRepository{db: db}
}

func (repo *PizzaRepository) Create(
	ctx context.Context,
	pizza *restaurant.Pizza,
) error {
	return repo.db.WithContext(ctx).Create(pizza).Error
}

func (repo *PizzaRepository) Update(
	ctx context.Context,
	pizza *restaurant.Pizza,
) error {
	return repo.db.WithContext(ctx).Save(pizza).Error
}

func (repo *PizzaRepository) FindByID(
	ctx context.Context,
	pizzaID uuid.UUID,
) (*restaurant.Pizza, error) {
	var p restaurant.Pizza

	err := repo.db.WithContext(ctx).
		Where("id = ?", pizzaID).
		Take(&p).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	return &p, err
}

func (repo *PizzaRepository) FindByIDAndRestaurant(
	ctx context.Context,
	pizzaID uuid.UUID,
	restaurantID uuid.UUID,
) (*restaurant.Pizza, error) {
	var p restaurant.Pizza

	err := repo.db.WithContext(ctx).
		Where("id = ? AND restaurant_id = ?", pizzaID, restaurantID).
		Take(&p).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	return &p, err
}

func (repo *PizzaRepository) ListByRestaurant(
	ctx context.Context,
	restaurantID uuid.UUID,
) ([]restaurant.Pizza, error) {
	var pizzas []restaurant.Pizza

	err := repo.db.WithContext(ctx).
		Where("restaurant_id = ?", restaurantID).
		Order("sort_order").
		Find(&pizzas).Error

	return pizzas, err
}
