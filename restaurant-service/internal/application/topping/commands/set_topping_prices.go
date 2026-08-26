package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	resapp "restaurant-service/internal/application/restaurant"
	toppingapp "restaurant-service/internal/application/topping"
	"restaurant-service/internal/domain/restaurant"
	"restaurant-service/internal/domain/topping"
	apperr "restaurant-service/internal/shared/errors"
	"restaurant-service/internal/shared/event"
)

var (
	minToppingExtraPrice = decimal.NewFromInt(1)
	maxToppingExtraPrice = decimal.NewFromInt(3)
)

type SetToppingPrices struct {
	restaurantRepo   restaurant.RestaurantRepository
	toppingRepo      topping.ToppingRepository
	toppingPriceRepo topping.ToppingPriceRepository
	publisher        event.EventPublisher
}

func NewSetToppingPrices(
	restaurantRepo restaurant.RestaurantRepository,
	toppingRepo topping.ToppingRepository,
	toppingPriceRepo topping.ToppingPriceRepository,
	publisher event.EventPublisher,
) *SetToppingPrices {
	return &SetToppingPrices{
		restaurantRepo:   restaurantRepo,
		toppingRepo:      toppingRepo,
		toppingPriceRepo: toppingPriceRepo,
		publisher:        publisher,
	}
}

func (uc *SetToppingPrices) Execute(
	ctx context.Context,
	restaurantID uuid.UUID,
	ownerID uuid.UUID,
	input toppingapp.SetToppingPricesRequest,
) ([]toppingapp.ToppingPriceResponse, error) {
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

	toppings, err := uc.toppingRepo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list pizza toppings: %w", err)
	}

	toppingByID := make(map[uuid.UUID]topping.Topping, len(toppings))
	for _, t := range toppings {
		toppingByID[t.ID] = t
	}

	seen := make(map[uuid.UUID]bool, len(input.Prices))
	prices := make([]topping.ToppingPrice, 0, len(input.Prices))

	for _, priceInput := range input.Prices {
		if _, ok := toppingByID[priceInput.ToppingID]; !ok {
			return nil, fmt.Errorf(
				"topping %s does not exist: %w",
				priceInput.ToppingID,
				apperr.ErrNotFound,
			)
		}

		if seen[priceInput.ToppingID] {
			return nil, fmt.Errorf(
				"duplicate topping %s in request: %w",
				priceInput.ToppingID,
				apperr.ErrConflict,
			)
		}
		seen[priceInput.ToppingID] = true

		if priceInput.ExtraPrice.LessThan(minToppingExtraPrice) ||
			priceInput.ExtraPrice.GreaterThan(maxToppingExtraPrice) {
			return nil, fmt.Errorf(
				"topping %s extra price must be between %s and %s: %w",
				priceInput.ToppingID,
				minToppingExtraPrice.String(),
				maxToppingExtraPrice.String(),
				apperr.ErrInvalid,
			)
		}

		price, err := topping.NewToppingPrice(restaurantID, priceInput.ToppingID, priceInput.ExtraPrice)
		if err != nil {
			return nil, fmt.Errorf("failed to generate topping price id: %w", err)
		}

		prices = append(prices, *price)
	}

	if err := uc.toppingPriceRepo.UpsertPrices(ctx, restaurantID, prices); err != nil {
		return nil, fmt.Errorf("failed to set topping prices: %w", err)
	}

	updated, err := uc.toppingPriceRepo.ListByRestaurant(ctx, restaurantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list topping prices: %w", err)
	}

	responses := make([]toppingapp.ToppingPriceResponse, 0, len(updated))
	for _, price := range updated {
		responses = append(
			responses,
			toppingapp.ToToppingPriceResponse(price, toppingByID[price.ToppingID].Name),
		)
	}

	res.NotifyToppingPricesUpdated()
	resapp.DispatchEvents(ctx, uc.publisher, res, uc.enrichToppingPricesUpdated(responses, *prices[0].UpdatedAt))

	return responses, nil
}

func (uc *SetToppingPrices) enrichToppingPricesUpdated(
	toppingPrices []toppingapp.ToppingPriceResponse,
	updatedAt time.Time,
) resapp.Enricher {
	return func(e restaurant.DomainEvent) (event.Event, bool) {
		updated, ok := e.(restaurant.ToppingPricesUpdated)
		if !ok {
			return nil, false
		}

		return resapp.NewToppingPricesUpdatedPayload(updated, toppingPrices, updatedAt), true
	}
}
