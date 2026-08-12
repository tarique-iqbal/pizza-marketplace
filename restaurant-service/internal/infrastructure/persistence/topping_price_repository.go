package persistence

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"restaurant-service/internal/domain/topping"
)

type ToppingPriceRepository struct {
	db *gorm.DB
}

func NewToppingPriceRepository(db *gorm.DB) topping.ToppingPriceRepository {
	return &ToppingPriceRepository{db: db}
}

func (repo *ToppingPriceRepository) UpsertPrices(
	ctx context.Context,
	restaurantID uuid.UUID,
	prices []topping.ToppingPrice,
) error {
	if len(prices) == 0 {
		return nil
	}

	now := time.Now().UTC()
	for i := range prices {
		prices[i].UpdatedAt = &now
	}

	return repo.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "restaurant_id"}, {Name: "topping_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"extra_price", "updated_at"}),
	}).Create(&prices).Error
}

func (repo *ToppingPriceRepository) ListByRestaurant(
	ctx context.Context,
	restaurantID uuid.UUID,
) ([]topping.ToppingPrice, error) {
	var prices []topping.ToppingPrice

	err := repo.db.WithContext(ctx).
		Where("restaurant_id = ?", restaurantID).
		Find(&prices).Error

	return prices, err
}
