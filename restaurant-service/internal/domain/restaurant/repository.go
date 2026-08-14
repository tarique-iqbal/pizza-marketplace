package restaurant

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RestaurantRepository interface {
	WithTx(tx *gorm.DB) RestaurantRepository
	Create(ctx context.Context, res *Restaurant) error
	Update(ctx context.Context, res *Restaurant) error
	FindBySlug(ctx context.Context, slug string) (*Restaurant, error)
	FindByID(ctx context.Context, id uuid.UUID) (*Restaurant, error)
	FindByIDAndOwner(ctx context.Context, restaurantID uuid.UUID, ownerID uuid.UUID) (*Restaurant, error)
}
