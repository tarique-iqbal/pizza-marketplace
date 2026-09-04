package readmodel

import (
	"context"

	"github.com/google/uuid"
)

type RestaurantRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*Restaurant, error)
	Upsert(ctx context.Context, restaurant Restaurant) error
}
