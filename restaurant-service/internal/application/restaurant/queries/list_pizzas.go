package queries

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	resapp "restaurant-service/internal/application/restaurant"
	"restaurant-service/internal/domain/restaurant"
	apperr "restaurant-service/internal/shared/errors"
)

type ListPizzas struct {
	restaurantRepo   restaurant.RestaurantRepository
	pizzaRepo        restaurant.PizzaRepository
	pizzaPriceRepo   restaurant.PizzaPriceRepository
	pizzaSizeRepo    restaurant.PizzaSizeRepository
	toppingRepo      restaurant.ToppingRepository
	toppingPriceRepo restaurant.ToppingPriceRepository
}

func NewListPizzas(
	restaurantRepo restaurant.RestaurantRepository,
	pizzaRepo restaurant.PizzaRepository,
	pizzaPriceRepo restaurant.PizzaPriceRepository,
	pizzaSizeRepo restaurant.PizzaSizeRepository,
	toppingRepo restaurant.ToppingRepository,
	toppingPriceRepo restaurant.ToppingPriceRepository,
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
) ([]resapp.PizzaResponse, error) {
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

	sizeByID := make(map[uuid.UUID]restaurant.PizzaSize, len(sizes))
	for _, size := range sizes {
		sizeByID[size.ID] = size
	}

	toppings, err := uc.toppingRepo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list pizza toppings: %w", err)
	}

	toppingByID := make(map[uuid.UUID]restaurant.Topping, len(toppings))
	for _, topping := range toppings {
		toppingByID[topping.ID] = topping
	}

	toppingPrices, err := uc.toppingPriceRepo.ListByRestaurant(ctx, restaurantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list topping prices: %w", err)
	}

	priceByToppingID := make(map[uuid.UUID]decimal.Decimal, len(toppingPrices))
	for _, price := range toppingPrices {
		priceByToppingID[price.ToppingID] = price.ExtraPrice
	}

	responses := make([]resapp.PizzaResponse, 0, len(pizzas))

	for i := range pizzas {
		pizza := pizzas[i]

		if pizza.Status == restaurant.PizzaArchived {
			continue
		}

		prices, err := uc.pizzaPriceRepo.ListByPizza(ctx, pizza.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to list prices for pizza %s: %w", pizza.ID, err)
		}

		toppingIDs, err := pizza.ToppingIDs()
		if err != nil {
			return nil, fmt.Errorf("failed to parse toppings for pizza %s: %w", pizza.ID, err)
		}

		responses = append(responses, resapp.ToPizzaResponse(&pizza, prices, sizeByID, toppingIDs, toppingByID, priceByToppingID))
	}

	return responses, nil
}
