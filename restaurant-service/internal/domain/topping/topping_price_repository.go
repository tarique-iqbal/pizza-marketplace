package topping

import (
	"context"

	"github.com/google/uuid"
)

type ToppingPriceRepository interface {
	UpsertPrices(ctx context.Context, restaurantID uuid.UUID, prices []ToppingPrice) error
	ListByRestaurant(ctx context.Context, restaurantID uuid.UUID) ([]ToppingPrice, error)
}
