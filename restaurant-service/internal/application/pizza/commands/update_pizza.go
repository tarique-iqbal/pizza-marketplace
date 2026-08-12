package commands

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	pizzaapp "restaurant-service/internal/application/pizza"
	"restaurant-service/internal/domain/pizza"
	"restaurant-service/internal/domain/restaurant"
	"restaurant-service/internal/domain/topping"
	apperr "restaurant-service/internal/shared/errors"
)

type UpdatePizza struct {
	restaurantRepo restaurant.RestaurantRepository
	pizzaRepo      pizza.PizzaRepository
	pizzaPriceRepo pizza.PizzaPriceRepository
	pizzaSizeRepo  pizza.PizzaSizeRepository
	toppingRepo    topping.ToppingRepository
}

func NewUpdatePizza(
	restaurantRepo restaurant.RestaurantRepository,
	pizzaRepo pizza.PizzaRepository,
	pizzaPriceRepo pizza.PizzaPriceRepository,
	pizzaSizeRepo pizza.PizzaSizeRepository,
	toppingRepo topping.ToppingRepository,
) *UpdatePizza {
	return &UpdatePizza{
		restaurantRepo: restaurantRepo,
		pizzaRepo:      pizzaRepo,
		pizzaPriceRepo: pizzaPriceRepo,
		pizzaSizeRepo:  pizzaSizeRepo,
		toppingRepo:    toppingRepo,
	}
}

func (uc *UpdatePizza) Execute(
	ctx context.Context,
	restaurantID uuid.UUID,
	pizzaID uuid.UUID,
	ownerID uuid.UUID,
	input pizzaapp.UpdatePizzaRequest,
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

	isVegetarian := p.IsVegetarian
	if input.IsVegetarian != nil {
		isVegetarian = *input.IsVegetarian
	}

	status := p.Status
	if input.Status != nil {
		status = pizza.PizzaStatus(*input.Status)
	}

	p.WithDetails(
		input.Name,
		input.Image,
		isVegetarian,
		status,
		input.SortOrder,
	)

	toppings, err := uc.toppingRepo.List(ctx)
	if err != nil {
		return pizzaapp.PizzaResponse{}, fmt.Errorf("failed to list pizza toppings: %w", err)
	}

	toppingByID := make(map[uuid.UUID]topping.Topping, len(toppings))
	for _, t := range toppings {
		toppingByID[t.ID] = t
	}

	if input.ToppingIDs != nil {
		if err := pizzaapp.ValidateToppingSelections(input.ToppingIDs, toppingByID); err != nil {
			return pizzaapp.PizzaResponse{}, err
		}

		if err := p.SetToppingIDs(input.ToppingIDs); err != nil {
			if errors.Is(err, pizza.ErrDuplicateTopping) {
				return pizzaapp.PizzaResponse{}, fmt.Errorf("%w: %w", err, apperr.ErrConflict)
			}
			return pizzaapp.PizzaResponse{}, fmt.Errorf("failed to set pizza toppings: %w", err)
		}
	}

	if err := uc.pizzaRepo.Update(ctx, p); err != nil {
		return pizzaapp.PizzaResponse{}, fmt.Errorf("failed to update pizza: %w", err)
	}

	toppingIDs, err := p.ToppingIDs()
	if err != nil {
		return pizzaapp.PizzaResponse{}, fmt.Errorf("failed to parse pizza toppings: %w", err)
	}

	prices, err := uc.pizzaPriceRepo.ListByPizza(ctx, p.ID)
	if err != nil {
		return pizzaapp.PizzaResponse{}, fmt.Errorf("failed to list pizza prices: %w", err)
	}

	sizes, err := uc.pizzaSizeRepo.List(ctx)
	if err != nil {
		return pizzaapp.PizzaResponse{}, fmt.Errorf("failed to list pizza sizes: %w", err)
	}

	sizeByID := make(map[uuid.UUID]pizza.PizzaSize, len(sizes))
	for _, size := range sizes {
		sizeByID[size.ID] = size
	}

	return pizzaapp.ToPizzaResponse(p, prices, sizeByID, toppingIDs, toppingByID, nil), nil
}
