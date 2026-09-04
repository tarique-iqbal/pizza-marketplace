package readmodel

import (
	"context"

	"github.com/google/uuid"
)

type PizzaPriceRepository interface {
	ListByPizza(ctx context.Context, pizzaID uuid.UUID) ([]PizzaPrice, error)
	Upsert(ctx context.Context, price PizzaPrice) error
}
