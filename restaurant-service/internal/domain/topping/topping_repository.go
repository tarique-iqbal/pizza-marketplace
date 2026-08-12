package topping

import (
	"context"

	"github.com/google/uuid"
)

type ToppingRepository interface {
	List(ctx context.Context) ([]Topping, error)
	FindByID(ctx context.Context, toppingID uuid.UUID) (*Topping, error)
}
