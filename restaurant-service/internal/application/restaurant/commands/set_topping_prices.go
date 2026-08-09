package commands

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	resapp "restaurant-service/internal/application/restaurant"
	"restaurant-service/internal/domain/restaurant"
	apperr "restaurant-service/internal/shared/errors"
)

var (
	minToppingExtraPrice = decimal.NewFromInt(1)
	maxToppingExtraPrice = decimal.NewFromInt(3)
)

type SetToppingPrices struct {
	restaurantRepo   restaurant.RestaurantRepository
	toppingRepo      restaurant.ToppingRepository
	toppingPriceRepo restaurant.ToppingPriceRepository
}

func NewSetToppingPrices(
	restaurantRepo restaurant.RestaurantRepository,
	toppingRepo restaurant.ToppingRepository,
	toppingPriceRepo restaurant.ToppingPriceRepository,
) *SetToppingPrices {
	return &SetToppingPrices{
		restaurantRepo:   restaurantRepo,
		toppingRepo:      toppingRepo,
		toppingPriceRepo: toppingPriceRepo,
	}
}

func (uc *SetToppingPrices) Execute(
	ctx context.Context,
	restaurantID uuid.UUID,
	ownerID uuid.UUID,
	input resapp.SetToppingPricesRequest,
) ([]resapp.ToppingPriceResponse, error) {
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

	toppingByID := make(map[uuid.UUID]restaurant.Topping, len(toppings))
	for _, topping := range toppings {
		toppingByID[topping.ID] = topping
	}

	seen := make(map[uuid.UUID]bool, len(input.Prices))
	prices := make([]restaurant.ToppingPrice, 0, len(input.Prices))

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

		if priceInput.ExtraPrice.LessThan(minToppingExtraPrice) || priceInput.ExtraPrice.GreaterThan(maxToppingExtraPrice) {
			return nil, fmt.Errorf(
				"topping %s extra price must be between %s and %s: %w",
				priceInput.ToppingID,
				minToppingExtraPrice.String(),
				maxToppingExtraPrice.String(),
				apperr.ErrInvalid,
			)
		}

		price, err := restaurant.NewToppingPrice(restaurantID, priceInput.ToppingID, priceInput.ExtraPrice)
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

	responses := make([]resapp.ToppingPriceResponse, 0, len(updated))
	for _, price := range updated {
		responses = append(responses, resapp.ToToppingPriceResponse(price, toppingByID[price.ToppingID].Name))
	}

	return responses, nil
}
