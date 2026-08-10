package restaurant

import "context"

type PizzaSizeRepository interface {
	List(ctx context.Context) ([]PizzaSize, error)
}
