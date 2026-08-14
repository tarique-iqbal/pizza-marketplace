package persistence

import (
	"context"
	"errors"
	"restaurant-service/internal/domain/restaurant"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type RestaurantRepository struct {
	db *gorm.DB
}

func NewRestaurantRepository(db *gorm.DB) restaurant.RestaurantRepository {
	return &RestaurantRepository{db: db}
}

func (r *RestaurantRepository) WithTx(tx *gorm.DB) restaurant.RestaurantRepository {
	return &RestaurantRepository{db: tx}
}

func (repo *RestaurantRepository) Create(
	ctx context.Context,
	res *restaurant.Restaurant,
) error {
	return repo.db.WithContext(ctx).Create(res).Error
}

func (repo *RestaurantRepository) Update(
	ctx context.Context,
	res *restaurant.Restaurant,
) error {
	return repo.db.WithContext(ctx).Omit(clause.Associations).Save(res).Error
}

func (repo *RestaurantRepository) FindBySlug(
	ctx context.Context,
	slug string,
) (*restaurant.Restaurant, error) {
	var r restaurant.Restaurant

	err := repo.db.WithContext(ctx).
		Where("slug = ?", slug).
		Take(&r).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	return &r, err
}

func (repo *RestaurantRepository) FindByID(
	ctx context.Context,
	id uuid.UUID,
) (*restaurant.Restaurant, error) {
	var r restaurant.Restaurant

	err := repo.db.WithContext(ctx).
		Where("id = ?", id).
		Take(&r).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	return &r, err
}

func (repo *RestaurantRepository) FindByIDAndOwner(
	ctx context.Context,
	restaurantID uuid.UUID,
	ownerID uuid.UUID,
) (*restaurant.Restaurant, error) {
	var r restaurant.Restaurant

	err := repo.db.WithContext(ctx).
		Where("id = ? AND owner_id = ?", restaurantID, ownerID).
		Take(&r).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	return &r, err
}
