package commands

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	resapp "restaurant-service/internal/application/restaurant"
	"restaurant-service/internal/domain/restaurant"
	apperr "restaurant-service/internal/shared/errors"
)

type UpdatePizza struct {
	restaurantRepo restaurant.RestaurantRepository
	pizzaRepo      restaurant.PizzaRepository
	pizzaPriceRepo restaurant.PizzaPriceRepository
	pizzaSizeRepo  restaurant.PizzaSizeRepository
	toppingRepo    restaurant.ToppingRepository
}

func NewUpdatePizza(
	restaurantRepo restaurant.RestaurantRepository,
	pizzaRepo restaurant.PizzaRepository,
	pizzaPriceRepo restaurant.PizzaPriceRepository,
	pizzaSizeRepo restaurant.PizzaSizeRepository,
	toppingRepo restaurant.ToppingRepository,
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
	input resapp.UpdatePizzaRequest,
) (resapp.PizzaResponse, error) {
	res, err := uc.restaurantRepo.FindByIDAndOwner(ctx, restaurantID, ownerID)
	if err != nil {
		return resapp.PizzaResponse{}, fmt.Errorf("failed to verify ownership: %w", err)
	}
	if res == nil {
		return resapp.PizzaResponse{}, fmt.Errorf(
			"access denied: restaurant not owned by user: %w",
			apperr.ErrForbidden,
		)
	}

	pizza, err := uc.pizzaRepo.FindByIDAndRestaurant(ctx, pizzaID, restaurantID)
	if err != nil {
		return resapp.PizzaResponse{}, fmt.Errorf("failed to find pizza: %w", err)
	}
	if pizza == nil {
		return resapp.PizzaResponse{}, fmt.Errorf("pizza not found: %w", apperr.ErrNotFound)
	}

	isVegetarian := pizza.IsVegetarian
	if input.IsVegetarian != nil {
		isVegetarian = *input.IsVegetarian
	}

	status := pizza.Status
	if input.Status != nil {
		status = restaurant.PizzaStatus(*input.Status)
	}

	pizza.WithDetails(
		input.Name,
		input.Image,
		isVegetarian,
		status,
		input.SortOrder,
	)

	toppings, err := uc.toppingRepo.List(ctx)
	if err != nil {
		return resapp.PizzaResponse{}, fmt.Errorf("failed to list pizza toppings: %w", err)
	}

	toppingByID := make(map[uuid.UUID]restaurant.Topping, len(toppings))
	for _, topping := range toppings {
		toppingByID[topping.ID] = topping
	}

	if input.ToppingIDs != nil {
		if err := resapp.ValidateToppingSelections(input.ToppingIDs, toppingByID); err != nil {
			return resapp.PizzaResponse{}, err
		}

		if err := pizza.SetToppingIDs(input.ToppingIDs); err != nil {
			if errors.Is(err, restaurant.ErrDuplicateTopping) {
				return resapp.PizzaResponse{}, fmt.Errorf("%w: %w", err, apperr.ErrConflict)
			}
			return resapp.PizzaResponse{}, fmt.Errorf("failed to set pizza toppings: %w", err)
		}
	}

	if err := uc.pizzaRepo.Update(ctx, pizza); err != nil {
		return resapp.PizzaResponse{}, fmt.Errorf("failed to update pizza: %w", err)
	}

	toppingIDs, err := pizza.ToppingIDs()
	if err != nil {
		return resapp.PizzaResponse{}, fmt.Errorf("failed to parse pizza toppings: %w", err)
	}

	prices, err := uc.pizzaPriceRepo.ListByPizza(ctx, pizza.ID)
	if err != nil {
		return resapp.PizzaResponse{}, fmt.Errorf("failed to list pizza prices: %w", err)
	}

	sizes, err := uc.pizzaSizeRepo.List(ctx)
	if err != nil {
		return resapp.PizzaResponse{}, fmt.Errorf("failed to list pizza sizes: %w", err)
	}

	sizeByID := make(map[uuid.UUID]restaurant.PizzaSize, len(sizes))
	for _, size := range sizes {
		sizeByID[size.ID] = size
	}

	return resapp.ToPizzaResponse(pizza, prices, sizeByID, toppingIDs, toppingByID, nil), nil
}
