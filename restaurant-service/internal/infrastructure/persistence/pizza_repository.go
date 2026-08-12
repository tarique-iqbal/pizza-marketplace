package persistence

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"restaurant-service/internal/domain/pizza"
)

type PizzaRepository struct {
	db *gorm.DB
}

func NewPizzaRepository(db *gorm.DB) pizza.PizzaRepository {
	return &PizzaRepository{db: db}
}

func (repo *PizzaRepository) Create(
	ctx context.Context,
	p *pizza.Pizza,
) error {
	return repo.db.WithContext(ctx).Create(p).Error
}

func (repo *PizzaRepository) Update(
	ctx context.Context,
	p *pizza.Pizza,
) error {
	return repo.db.WithContext(ctx).Save(p).Error
}

func (repo *PizzaRepository) FindByID(
	ctx context.Context,
	pizzaID uuid.UUID,
) (*pizza.Pizza, error) {
	var p pizza.Pizza

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
) (*pizza.Pizza, error) {
	var p pizza.Pizza

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
) ([]pizza.Pizza, error) {
	var pizzas []pizza.Pizza

	err := repo.db.WithContext(ctx).
		Where("restaurant_id = ?", restaurantID).
		Order("sort_order").
		Find(&pizzas).Error

	return pizzas, err
}
