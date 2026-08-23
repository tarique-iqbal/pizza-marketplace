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

type UpdatePizza struct {
	restaurantRepo restaurant.RestaurantRepository
	pizzaRepo      pizza.PizzaRepository
	pizzaPriceRepo pizza.PizzaPriceRepository
	pizzaSizeRepo  pizza.PizzaSizeRepository
	toppingRepo    topping.ToppingRepository
	publisher      event.EventPublisher
}

func NewUpdatePizza(
	restaurantRepo restaurant.RestaurantRepository,
	pizzaRepo pizza.PizzaRepository,
	pizzaPriceRepo pizza.PizzaPriceRepository,
	pizzaSizeRepo pizza.PizzaSizeRepository,
	toppingRepo topping.ToppingRepository,
	publisher event.EventPublisher,
) *UpdatePizza {
	return &UpdatePizza{
		restaurantRepo: restaurantRepo,
		pizzaRepo:      pizzaRepo,
		pizzaPriceRepo: pizzaPriceRepo,
		pizzaSizeRepo:  pizzaSizeRepo,
		toppingRepo:    toppingRepo,
		publisher:      publisher,
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

	output := pizzaapp.ToPizzaResponse(p, prices, sizeByID, toppingIDs, toppingByID, nil)

	res.NotifyPizzaUpdated()
	resapp.DispatchEvents(ctx, uc.publisher, res, uc.enrichPizzaUpdated(output))

	return output, nil
}

func (uc *UpdatePizza) enrichPizzaUpdated(output pizzaapp.PizzaResponse) resapp.Enricher {
	return func(e restaurant.DomainEvent) (event.Event, bool) {
		updated, ok := e.(restaurant.PizzaUpdated)
		if !ok {
			return nil, false
		}

		return resapp.NewPizzaUpdatedPayload(updated, output), true
	}
}
