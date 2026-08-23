package commands

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	pizzaapp "restaurant-service/internal/application/pizza"
	resapp "restaurant-service/internal/application/restaurant"
	"restaurant-service/internal/domain/pizza"
	"restaurant-service/internal/domain/restaurant"
	"restaurant-service/internal/domain/topping"
	apperr "restaurant-service/internal/shared/errors"
	"restaurant-service/internal/shared/event"
)

type CreatePizza struct {
	restaurantRepo restaurant.RestaurantRepository
	pizzaRepo      pizza.PizzaRepository
	toppingRepo    topping.ToppingRepository
	publisher      event.EventPublisher
}

func NewCreatePizza(
	restaurantRepo restaurant.RestaurantRepository,
	pizzaRepo pizza.PizzaRepository,
	toppingRepo topping.ToppingRepository,
	publisher event.EventPublisher,
) *CreatePizza {
	return &CreatePizza{
		restaurantRepo: restaurantRepo,
		pizzaRepo:      pizzaRepo,
		toppingRepo:    toppingRepo,
		publisher:      publisher,
	}
}

func (uc *CreatePizza) Execute(
	ctx context.Context,
	restaurantID uuid.UUID,
	ownerID uuid.UUID,
	input pizzaapp.CreatePizzaRequest,
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

	if !res.Checklist.IsCompleted() {
		return pizzaapp.PizzaResponse{}, fmt.Errorf(
			"%w: %w",
			restaurant.ErrChecklistIncomplete,
			apperr.ErrForbidden,
		)
	}

	isVegetarian := false
	if input.IsVegetarian != nil {
		isVegetarian = *input.IsVegetarian
	}

	status := pizza.PizzaAvailable
	if input.Status != nil {
		status = pizza.PizzaStatus(*input.Status)
	}

	id, err := uuid.NewV7()
	if err != nil {
		return pizzaapp.PizzaResponse{}, fmt.Errorf("failed to generate pizza id: %w", err)
	}

	p := pizza.NewPizza(id, restaurantID).WithDetails(
		input.Name,
		input.Image,
		isVegetarian,
		status,
		input.SortOrder,
	)

	var toppingIDs []uuid.UUID
	var toppingByID map[uuid.UUID]topping.Topping

	if len(input.ToppingIDs) > 0 {
		toppings, err := uc.toppingRepo.List(ctx)
		if err != nil {
			return pizzaapp.PizzaResponse{}, fmt.Errorf("failed to list pizza toppings: %w", err)
		}

		toppingByID = make(map[uuid.UUID]topping.Topping, len(toppings))
		for _, t := range toppings {
			toppingByID[t.ID] = t
		}

		if err := pizzaapp.ValidateToppingSelections(input.ToppingIDs, toppingByID); err != nil {
			return pizzaapp.PizzaResponse{}, err
		}

		if err := p.SetToppingIDs(input.ToppingIDs); err != nil {
			if errors.Is(err, pizza.ErrDuplicateTopping) {
				return pizzaapp.PizzaResponse{}, fmt.Errorf("%w: %w", err, apperr.ErrConflict)
			}
			return pizzaapp.PizzaResponse{}, fmt.Errorf("failed to set pizza toppings: %w", err)
		}

		toppingIDs = input.ToppingIDs
	}

	if err := uc.pizzaRepo.Create(ctx, p); err != nil {
		return pizzaapp.PizzaResponse{}, fmt.Errorf("failed to create pizza: %w", err)
	}

	output := pizzaapp.ToPizzaResponse(p, nil, nil, toppingIDs, toppingByID, nil)

	res.NotifyPizzaUpdated()
	resapp.DispatchEvents(ctx, uc.publisher, res, uc.enrichPizzaUpdated(output))

	return output, nil
}

func (uc *CreatePizza) enrichPizzaUpdated(output pizzaapp.PizzaResponse) resapp.Enricher {
	return func(e restaurant.DomainEvent) (event.Event, bool) {
		updated, ok := e.(restaurant.PizzaUpdated)
		if !ok {
			return nil, false
		}

		return resapp.NewPizzaUpdatedPayload(updated, output), true
	}
}
