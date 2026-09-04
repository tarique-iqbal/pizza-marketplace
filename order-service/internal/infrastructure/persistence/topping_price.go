package persistence

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"order-service/internal/domain/readmodel"
)

type ToppingPriceRepository struct {
	db *gorm.DB
}

func NewToppingPriceRepository(db *gorm.DB) readmodel.ToppingPriceRepository {
	return &ToppingPriceRepository{db: db}
}

func (r *ToppingPriceRepository) ListByRestaurant(
	ctx context.Context,
	restaurantID uuid.UUID,
) ([]readmodel.ToppingPrice, error) {
	var prices []readmodel.ToppingPrice

	err := r.db.WithContext(ctx).Where("restaurant_id = ?", restaurantID).Find(&prices).Error

	return prices, err
}

// Upsert is guarded by updatedAt — a no-op if the stored row is already newer.
func (r *ToppingPriceRepository) Upsert(ctx context.Context, price readmodel.ToppingPrice) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "restaurant_id"}, {Name: "topping_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"name", "extra_price", "updated_at"}),
		Where: clause.Where{Exprs: []clause.Expression{
			clause.Expr{SQL: "topping_prices.updated_at < excluded.updated_at"},
		}},
	}).Create(&price).Error
}
