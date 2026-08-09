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

type CreatePizza struct {
	restaurantRepo restaurant.RestaurantRepository
	pizzaRepo      restaurant.PizzaRepository
	toppingRepo    restaurant.ToppingRepository
}

func NewCreatePizza(
	restaurantRepo restaurant.RestaurantRepository,
	pizzaRepo restaurant.PizzaRepository,
	toppingRepo restaurant.ToppingRepository,
) *CreatePizza {
	return &CreatePizza{
		restaurantRepo: restaurantRepo,
		pizzaRepo:      pizzaRepo,
		toppingRepo:    toppingRepo,
	}
}

func (uc *CreatePizza) Execute(
	ctx context.Context,
	restaurantID uuid.UUID,
	ownerID uuid.UUID,
	input resapp.CreatePizzaRequest,
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

	if !res.Checklist.IsCompleted() {
		return resapp.PizzaResponse{}, fmt.Errorf(
			"%w: %w",
			restaurant.ErrChecklistIncomplete,
			apperr.ErrForbidden,
		)
	}

	isVegetarian := false
	if input.IsVegetarian != nil {
		isVegetarian = *input.IsVegetarian
	}

	status := restaurant.PizzaAvailable
	if input.Status != nil {
		status = restaurant.PizzaStatus(*input.Status)
	}

	id, err := uuid.NewV7()
	if err != nil {
		return resapp.PizzaResponse{}, fmt.Errorf("failed to generate pizza id: %w", err)
	}

	pizza := restaurant.NewPizza(id, restaurantID).WithDetails(
		input.Name,
		input.Image,
		isVegetarian,
		status,
		input.SortOrder,
	)

	var toppingIDs []uuid.UUID
	var toppingByID map[uuid.UUID]restaurant.Topping

	if len(input.ToppingIDs) > 0 {
		toppings, err := uc.toppingRepo.List(ctx)
		if err != nil {
			return resapp.PizzaResponse{}, fmt.Errorf("failed to list pizza toppings: %w", err)
		}

		toppingByID = make(map[uuid.UUID]restaurant.Topping, len(toppings))
		for _, topping := range toppings {
			toppingByID[topping.ID] = topping
		}

		if err := resapp.ValidateToppingSelections(input.ToppingIDs, toppingByID); err != nil {
			return resapp.PizzaResponse{}, err
		}

		if err := pizza.SetToppingIDs(input.ToppingIDs); err != nil {
			if errors.Is(err, restaurant.ErrDuplicateTopping) {
				return resapp.PizzaResponse{}, fmt.Errorf("%w: %w", err, apperr.ErrConflict)
			}
			return resapp.PizzaResponse{}, fmt.Errorf("failed to set pizza toppings: %w", err)
		}

		toppingIDs = input.ToppingIDs
	}

	if err := uc.pizzaRepo.Create(ctx, pizza); err != nil {
		return resapp.PizzaResponse{}, fmt.Errorf("failed to create pizza: %w", err)
	}

	return resapp.ToPizzaResponse(pizza, nil, nil, toppingIDs, toppingByID, nil), nil
}
