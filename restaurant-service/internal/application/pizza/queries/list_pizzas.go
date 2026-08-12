package queries

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	pizzaapp "restaurant-service/internal/application/pizza"
	"restaurant-service/internal/domain/pizza"
	"restaurant-service/internal/domain/restaurant"
	"restaurant-service/internal/domain/topping"
	apperr "restaurant-service/internal/shared/errors"
)

type ListPizzas struct {
	restaurantRepo   restaurant.RestaurantRepository
	pizzaRepo        pizza.PizzaRepository
	pizzaPriceRepo   pizza.PizzaPriceRepository
	pizzaSizeRepo    pizza.PizzaSizeRepository
	toppingRepo      topping.ToppingRepository
	toppingPriceRepo topping.ToppingPriceRepository
}

func NewListPizzas(
	restaurantRepo restaurant.RestaurantRepository,
	pizzaRepo pizza.PizzaRepository,
	pizzaPriceRepo pizza.PizzaPriceRepository,
	pizzaSizeRepo pizza.PizzaSizeRepository,
	toppingRepo topping.ToppingRepository,
	toppingPriceRepo topping.ToppingPriceRepository,
) *ListPizzas {
	return &ListPizzas{
		restaurantRepo:   restaurantRepo,
		pizzaRepo:        pizzaRepo,
		pizzaPriceRepo:   pizzaPriceRepo,
		pizzaSizeRepo:    pizzaSizeRepo,
		toppingRepo:      toppingRepo,
		toppingPriceRepo: toppingPriceRepo,
	}
}

func (uc *ListPizzas) Execute(
	ctx context.Context,
	restaurantID uuid.UUID,
	ownerID uuid.UUID,
) ([]pizzaapp.PizzaResponse, error) {
	res, err := uc.restaurantRepo.FindByIDAndOwner(ctx, restaurantID, ownerID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify ownership: %w", err)
	}
	if res == nil {
		return nil, fmt.Errorf(
			"access denied: restaurant not owned by user: %w",
			apperr.ErrForbidden,
		)
	}

	pizzas, err := uc.pizzaRepo.ListByRestaurant(ctx, restaurantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list pizzas: %w", err)
	}

	sizes, err := uc.pizzaSizeRepo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list pizza sizes: %w", err)
	}

	sizeByID := make(map[uuid.UUID]pizza.PizzaSize, len(sizes))
	for _, size := range sizes {
		sizeByID[size.ID] = size
	}

	toppings, err := uc.toppingRepo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list pizza toppings: %w", err)
	}

	toppingByID := make(map[uuid.UUID]topping.Topping, len(toppings))
	for _, t := range toppings {
		toppingByID[t.ID] = t
	}

	toppingPrices, err := uc.toppingPriceRepo.ListByRestaurant(ctx, restaurantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list topping prices: %w", err)
	}

	priceByToppingID := make(map[uuid.UUID]decimal.Decimal, len(toppingPrices))
	for _, price := range toppingPrices {
		priceByToppingID[price.ToppingID] = price.ExtraPrice
	}

	responses := make([]pizzaapp.PizzaResponse, 0, len(pizzas))

	for i := range pizzas {
		p := pizzas[i]

		if p.Status == pizza.PizzaArchived {
			continue
		}

		prices, err := uc.pizzaPriceRepo.ListByPizza(ctx, p.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to list prices for pizza %s: %w", p.ID, err)
		}

		toppingIDs, err := p.ToppingIDs()
		if err != nil {
			return nil, fmt.Errorf("failed to parse toppings for pizza %s: %w", p.ID, err)
		}

		responses = append(responses, pizzaapp.ToPizzaResponse(
			&p, prices, sizeByID, toppingIDs, toppingByID, priceByToppingID,
		))
	}

	return responses, nil
}
