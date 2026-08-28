package topping

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ToppingPriceRepository interface {
	WithTx(tx *gorm.DB) ToppingPriceRepository
	UpsertPrices(ctx context.Context, restaurantID uuid.UUID, prices []ToppingPrice) error
	ListByRestaurant(ctx context.Context, restaurantID uuid.UUID) ([]ToppingPrice, error)
}
