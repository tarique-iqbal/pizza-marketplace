package commands

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	pizzaapp "restaurant-service/internal/application/pizza"
	resapp "restaurant-service/internal/application/restaurant"
	"restaurant-service/internal/domain/outbox"
	"restaurant-service/internal/domain/pizza"
	"restaurant-service/internal/domain/restaurant"
	"restaurant-service/internal/domain/topping"
	apperr "restaurant-service/internal/shared/errors"
	"restaurant-service/internal/shared/event"
)

type UpdatePizza struct {
	db             *gorm.DB
	restaurantRepo restaurant.RestaurantRepository
	pizzaRepo      pizza.PizzaRepository
	pizzaPriceRepo pizza.PizzaPriceRepository
	pizzaSizeRepo  pizza.PizzaSizeRepository
	toppingRepo    topping.ToppingRepository
	outboxRepo     outbox.OutboxRepository
}

func NewUpdatePizza(
	db *gorm.DB,
	restaurantRepo restaurant.RestaurantRepository,
	pizzaRepo pizza.PizzaRepository,
	pizzaPriceRepo pizza.PizzaPriceRepository,
	pizzaSizeRepo pizza.PizzaSizeRepository,
	toppingRepo topping.ToppingRepository,
	outboxRepo outbox.OutboxRepository,
) *UpdatePizza {
	return &UpdatePizza{
		db:             db,
		restaurantRepo: restaurantRepo,
		pizzaRepo:      pizzaRepo,
		pizzaPriceRepo: pizzaPriceRepo,
		pizzaSizeRepo:  pizzaSizeRepo,
		toppingRepo:    toppingRepo,
		outboxRepo:     outboxRepo,
	}
}

func (cmd *UpdatePizza) Execute(
	ctx context.Context,
	restaurantID uuid.UUID,
	pizzaID uuid.UUID,
	ownerID uuid.UUID,
	input pizzaapp.UpdatePizzaRequest,
) (pizzaapp.PizzaResponse, error) {
	res, err := cmd.restaurantRepo.FindByIDAndOwner(ctx, restaurantID, ownerID)
	if err != nil {
		return pizzaapp.PizzaResponse{}, fmt.Errorf("failed to verify ownership: %w", err)
	}
	if res == nil {
		return pizzaapp.PizzaResponse{}, fmt.Errorf(
			"access denied: restaurant not owned by user: %w",
			apperr.ErrForbidden,
		)
	}

	p, err := cmd.pizzaRepo.FindByIDAndRestaurant(ctx, pizzaID, restaurantID)
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

	toppings, err := cmd.toppingRepo.List(ctx)
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

	toppingIDs, err := p.ToppingIDs()
	if err != nil {
		return pizzaapp.PizzaResponse{}, fmt.Errorf("failed to parse pizza toppings: %w", err)
	}

	prices, err := cmd.pizzaPriceRepo.ListByPizza(ctx, p.ID)
	if err != nil {
		return pizzaapp.PizzaResponse{}, fmt.Errorf("failed to list pizza prices: %w", err)
	}

	sizes, err := cmd.pizzaSizeRepo.List(ctx)
	if err != nil {
		return pizzaapp.PizzaResponse{}, fmt.Errorf("failed to list pizza sizes: %w", err)
	}

	sizeByID := make(map[uuid.UUID]pizza.PizzaSize, len(sizes))
	for _, size := range sizes {
		sizeByID[size.ID] = size
	}

	res.NotifyPizzaUpdated()

	var output pizzaapp.PizzaResponse

	err = cmd.db.Transaction(func(tx *gorm.DB) error {
		if err := cmd.pizzaRepo.WithTx(tx).Update(ctx, p); err != nil {
			return fmt.Errorf("failed to update pizza: %w", err)
		}

		// p.UpdatedAt is only set once Update runs, so build the response after it.
		output = pizzaapp.ToPizzaResponse(p, prices, sizeByID, toppingIDs, toppingByID, nil)

		return resapp.DispatchEventsTx(ctx, cmd.outboxRepo.WithTx(tx), res, cmd.enrichPizzaUpdated(output))
	})
	if err != nil {
		return pizzaapp.PizzaResponse{}, err
	}

	return output, nil
}

func (cmd *UpdatePizza) enrichPizzaUpdated(output pizzaapp.PizzaResponse) resapp.Enricher {
	return func(e restaurant.DomainEvent) (event.Event, bool) {
		updated, ok := e.(restaurant.PizzaUpdated)
		if !ok {
			return nil, false
		}

		return resapp.NewPizzaUpdatedPayload(updated, output), true
	}
}
