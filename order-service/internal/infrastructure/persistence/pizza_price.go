package persistence

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"order-service/internal/domain/readmodel"
)

type PizzaPriceRepository struct {
	db *gorm.DB
}

func NewPizzaPriceRepository(db *gorm.DB) readmodel.PizzaPriceRepository {
	return &PizzaPriceRepository{db: db}
}

func (r *PizzaPriceRepository) ListByPizza(ctx context.Context, pizzaID uuid.UUID) ([]readmodel.PizzaPrice, error) {
	var prices []readmodel.PizzaPrice

	err := r.db.WithContext(ctx).Where("pizza_id = ?", pizzaID).Find(&prices).Error

	return prices, err
}

// Upsert is guarded by updatedAt — a no-op if the stored row is already newer.
func (r *PizzaPriceRepository) Upsert(ctx context.Context, price readmodel.PizzaPrice) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "pizza_id"}, {Name: "size_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"diameter_cm", "price", "is_active", "updated_at"}),
		Where: clause.Where{Exprs: []clause.Expression{
			clause.Expr{SQL: "pizza_prices.updated_at < excluded.updated_at"},
		}},
	}).Create(&price).Error
}
