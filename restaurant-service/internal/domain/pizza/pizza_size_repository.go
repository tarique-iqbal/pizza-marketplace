package pizza

import "context"

type PizzaSizeRepository interface {
	List(ctx context.Context) ([]PizzaSize, error)
}
