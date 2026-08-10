package restaurant

import (
	"context"

	"github.com/google/uuid"
)

type PizzaPriceRepository interface {
	ReplacePrices(ctx context.Context, pizzaID uuid.UUID, prices []PizzaPrice) error
	ListByPizza(ctx context.Context, pizzaID uuid.UUID) ([]PizzaPrice, error)
}
