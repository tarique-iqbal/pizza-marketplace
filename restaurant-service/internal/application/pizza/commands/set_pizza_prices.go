package commands

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	pizzaapp "restaurant-service/internal/application/pizza"
	"restaurant-service/internal/domain/pizza"
	"restaurant-service/internal/domain/restaurant"
	"restaurant-service/internal/domain/topping"
	apperr "restaurant-service/internal/shared/errors"
)

type SetPizzaPrices struct {
	restaurantRepo restaurant.RestaurantRepository
	pizzaRepo      pizza.PizzaRepository
	pizzaPriceRepo pizza.PizzaPriceRepository
	pizzaSizeRepo  pizza.PizzaSizeRepository
	toppingRepo    topping.ToppingRepository
}

func NewSetPizzaPrices(
	restaurantRepo restaurant.RestaurantRepository,
	pizzaRepo pizza.PizzaRepository,
	pizzaPriceRepo pizza.PizzaPriceRepository,
	pizzaSizeRepo pizza.PizzaSizeRepository,
	toppingRepo topping.ToppingRepository,
) *SetPizzaPrices {
	return &SetPizzaPrices{
		restaurantRepo: restaurantRepo,
		pizzaRepo:      pizzaRepo,
		pizzaPriceRepo: pizzaPriceRepo,
		pizzaSizeRepo:  pizzaSizeRepo,
		toppingRepo:    toppingRepo,
	}
}

func (uc *SetPizzaPrices) Execute(
	ctx context.Context,
	restaurantID uuid.UUID,
	pizzaID uuid.UUID,
	ownerID uuid.UUID,
	input pizzaapp.SetPizzaPricesRequest,
) (pizzaapp.PizzaResponse, error) {
	res, err := uc.restaurantRepo.FindByIDAndOwner(ctx, restaurantID, ownerID)
	if err != nil {
		return pizzaapp.PizzaResponse{}, fmt.Errorf("failed to verify ownership: %w", err)
	}
	if res == nil {
		return pizzaapp.PizzaResponse{}, fmt.Errorf(
			"access denied: restaurant not owned by user: %w",
			apperr.ErrForbidden,
		)
	}

	p, err := uc.pizzaRepo.FindByIDAndRestaurant(ctx, pizzaID, restaurantID)
	if err != nil {
		return pizzaapp.PizzaResponse{}, fmt.Errorf("failed to find pizza: %w", err)
	}
	if p == nil {
		return pizzaapp.PizzaResponse{}, fmt.Errorf("pizza not found: %w", apperr.ErrNotFound)
	}

	sizes, err := uc.pizzaSizeRepo.List(ctx)
	if err != nil {
		return pizzaapp.PizzaResponse{}, fmt.Errorf("failed to list pizza sizes: %w", err)
	}

	sizeByID := make(map[uuid.UUID]pizza.PizzaSize, len(sizes))
	for _, size := range sizes {
		sizeByID[size.ID] = size
	}

	seen := make(map[uuid.UUID]bool, len(input.Prices))
	prices := make([]pizza.PizzaPrice, 0, len(input.Prices))

	for _, priceInput := range input.Prices {
		if _, ok := sizeByID[priceInput.SizeID]; !ok {
			return pizzaapp.PizzaResponse{}, fmt.Errorf(
				"size %s does not exist: %w",
				priceInput.SizeID,
				apperr.ErrNotFound,
			)
		}

		if seen[priceInput.SizeID] {
			return pizzaapp.PizzaResponse{}, fmt.Errorf(
				"duplicate size %s in request: %w",
				priceInput.SizeID,
				apperr.ErrConflict,
			)
		}
		seen[priceInput.SizeID] = true

		price, err := pizza.NewPizzaPrice(pizzaID, priceInput.SizeID, priceInput.Price)
		if err != nil {
			return pizzaapp.PizzaResponse{}, fmt.Errorf("failed to generate price id: %w", err)
		}

		prices = append(prices, *price)
	}

	if err := uc.pizzaPriceRepo.ReplacePrices(ctx, pizzaID, prices); err != nil {
		return pizzaapp.PizzaResponse{}, fmt.Errorf("failed to set pizza prices: %w", err)
	}

	updated, err := uc.pizzaPriceRepo.ListByPizza(ctx, pizzaID)
	if err != nil {
		return pizzaapp.PizzaResponse{}, fmt.Errorf("failed to list pizza prices: %w", err)
	}

	toppingIDs, err := p.ToppingIDs()
	if err != nil {
		return pizzaapp.PizzaResponse{}, fmt.Errorf("failed to parse pizza toppings: %w", err)
	}

	toppings, err := uc.toppingRepo.List(ctx)
	if err != nil {
		return pizzaapp.PizzaResponse{}, fmt.Errorf("failed to list pizza toppings: %w", err)
	}

	toppingByID := make(map[uuid.UUID]topping.Topping, len(toppings))
	for _, t := range toppings {
		toppingByID[t.ID] = t
	}

	return pizzaapp.ToPizzaResponse(p, updated, sizeByID, toppingIDs, toppingByID, nil), nil
}
