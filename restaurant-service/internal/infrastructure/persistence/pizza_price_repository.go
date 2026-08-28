package persistence

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"restaurant-service/internal/domain/pizza"
)

type PizzaPriceRepository struct {
	db *gorm.DB
}

func NewPizzaPriceRepository(db *gorm.DB) pizza.PizzaPriceRepository {
	return &PizzaPriceRepository{db: db}
}

func (repo *PizzaPriceRepository) WithTx(tx *gorm.DB) pizza.PizzaPriceRepository {
	return &PizzaPriceRepository{db: tx}
}

func (repo *PizzaPriceRepository) ReplacePrices(
	ctx context.Context,
	pizzaID uuid.UUID,
	prices []pizza.PizzaPrice,
) error {
	return repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// GORM's autoUpdateTime doesn't populate this on bulk upserts, so set it explicitly.
		now := time.Now().UTC()
		sizeIDs := make([]uuid.UUID, 0, len(prices))
		for i := range prices {
			prices[i].UpdatedAt = &now
			sizeIDs = append(sizeIDs, prices[i].SizeID)
		}

		if len(prices) > 0 {
			err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "pizza_id"}, {Name: "size_id"}},
				DoUpdates: clause.AssignmentColumns([]string{"price", "is_active", "updated_at"}),
			}).Create(&prices).Error
			if err != nil {
				return err
			}
		}

		deactivate := tx.Model(&pizza.PizzaPrice{}).
			Where("pizza_id = ?", pizzaID)

		if len(sizeIDs) > 0 {
			deactivate = deactivate.Where("size_id NOT IN ?", sizeIDs)
		}

		return deactivate.Update("is_active", false).Error
	})
}

func (repo *PizzaPriceRepository) ListByPizza(
	ctx context.Context,
	pizzaID uuid.UUID,
) ([]pizza.PizzaPrice, error) {
	var prices []pizza.PizzaPrice

	err := repo.db.WithContext(ctx).
		Where("pizza_id = ?", pizzaID).
		Find(&prices).Error

	return prices, err
}
