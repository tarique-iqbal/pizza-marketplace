package pizza

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PizzaRepository interface {
	WithTx(tx *gorm.DB) PizzaRepository
	Create(ctx context.Context, pizza *Pizza) error
	Update(ctx context.Context, pizza *Pizza) error
	FindByID(ctx context.Context, pizzaID uuid.UUID) (*Pizza, error)
	FindByIDAndRestaurant(ctx context.Context, pizzaID uuid.UUID, restaurantID uuid.UUID) (*Pizza, error)
	ListByRestaurant(ctx context.Context, restaurantID uuid.UUID) ([]Pizza, error)
}
