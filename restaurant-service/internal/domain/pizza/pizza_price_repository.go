package pizza

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PizzaPriceRepository interface {
	WithTx(tx *gorm.DB) PizzaPriceRepository
	ReplacePrices(ctx context.Context, pizzaID uuid.UUID, prices []PizzaPrice) error
	ListByPizza(ctx context.Context, pizzaID uuid.UUID) ([]PizzaPrice, error)
}
