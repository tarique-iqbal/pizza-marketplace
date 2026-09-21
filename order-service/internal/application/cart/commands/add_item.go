package commands

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	cartapp "order-service/internal/application/cart"
	"order-service/internal/domain/cart"
	"order-service/internal/domain/readmodel"
	apperr "order-service/internal/shared/errors"
)

type AddItem struct {
	cartRepo         cart.CartRepository
	pizzaRepo        readmodel.PizzaRepository
	pizzaPriceRepo   readmodel.PizzaPriceRepository
	toppingPriceRepo readmodel.ToppingPriceRepository
}

func NewAddItem(
	cartRepo cart.CartRepository,
	pizzaRepo readmodel.PizzaRepository,
	pizzaPriceRepo readmodel.PizzaPriceRepository,
	toppingPriceRepo readmodel.ToppingPriceRepository,
) *AddItem {
	return &AddItem{
		cartRepo:         cartRepo,
		pizzaRepo:        pizzaRepo,
		pizzaPriceRepo:   pizzaPriceRepo,
		toppingPriceRepo: toppingPriceRepo,
	}
}

func (cmd *AddItem) Execute(
	ctx context.Context,
	customerID uuid.UUID,
	input cartapp.AddItemRequest,
) (cartapp.AddItemResponse, error) {
	pizza, err := cmd.pizzaRepo.FindByID(ctx, input.PizzaID)
	if err != nil {
		return cartapp.AddItemResponse{}, fmt.Errorf("pizza not found: %w", err)
	}

	prices, err := cmd.pizzaPriceRepo.ListByPizza(ctx, input.PizzaID)
	if err != nil {
		return cartapp.AddItemResponse{}, fmt.Errorf("failed to look up pizza prices: %w", err)
	}

	priceFound := false
	for _, p := range prices {
		if p.SizeID == input.SizeID && p.IsActive {
			priceFound = true
			break
		}
	}
	if !priceFound {
		return cartapp.AddItemResponse{}, fmt.Errorf("pizza size not available: %w", apperr.ErrNotFound)
	}

	if len(input.ExtraToppingIDs) > 0 {
		toppingPrices, err := cmd.toppingPriceRepo.ListByRestaurant(ctx, pizza.RestaurantID)
		if err != nil {
			return cartapp.AddItemResponse{}, fmt.Errorf("failed to look up topping prices: %w", err)
		}

		valid := make(map[uuid.UUID]bool, len(toppingPrices))
		for _, tp := range toppingPrices {
			valid[tp.ToppingID] = true
		}

		for _, toppingID := range input.ExtraToppingIDs {
			if !valid[toppingID] {
				return cartapp.AddItemResponse{}, fmt.Errorf("topping not found: %w", apperr.ErrNotFound)
			}
		}
	}

	existingCart, err := cmd.cartRepo.FindByCustomer(ctx, customerID)
	if err != nil {
		return cartapp.AddItemResponse{}, fmt.Errorf("failed to look up cart: %w", err)
	}

	if existingCart == nil {
		id, err := uuid.NewV7()
		if err != nil {
			return cartapp.AddItemResponse{}, fmt.Errorf("failed to generate cart id: %w", err)
		}

		newCart := cart.NewCart(id, customerID, pizza.RestaurantID)
		if err := cmd.cartRepo.Create(ctx, newCart); err != nil {
			return cartapp.AddItemResponse{}, fmt.Errorf("failed to create cart: %w", err)
		}

		existingCart = newCart
	} else if err := existingCart.EnsureRestaurant(pizza.RestaurantID); err != nil {
		return cartapp.AddItemResponse{}, err
	}

	itemID, err := uuid.NewV7()
	if err != nil {
		return cartapp.AddItemResponse{}, fmt.Errorf("failed to generate cart item id: %w", err)
	}

	item := cart.NewCartItem(itemID, input.PizzaID, input.SizeID, input.Quantity, input.ExtraToppingIDs)

	if err := cmd.cartRepo.AddOrMergeItem(ctx, existingCart.ID, item); err != nil {
		return cartapp.AddItemResponse{}, fmt.Errorf("failed to add cart item: %w", err)
	}

	return cartapp.AddItemResponse{
		PizzaID:         item.PizzaID,
		SizeID:          item.SizeID,
		Quantity:        item.Quantity,
		ExtraToppingIDs: item.ExtraToppingIDs,
	}, nil
}
