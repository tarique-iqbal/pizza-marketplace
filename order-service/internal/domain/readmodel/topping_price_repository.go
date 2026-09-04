package readmodel

import (
	"context"

	"github.com/google/uuid"
)

type ToppingPriceRepository interface {
	ListByRestaurant(ctx context.Context, restaurantID uuid.UUID) ([]ToppingPrice, error)
	Upsert(ctx context.Context, price ToppingPrice) error
}
