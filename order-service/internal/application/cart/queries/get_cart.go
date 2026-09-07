package queries

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	cartapp "order-service/internal/application/cart"
	"order-service/internal/domain/cart"
	"order-service/internal/domain/readmodel"
	apperr "order-service/internal/shared/errors"
	"order-service/internal/shared/money"
)

type GetCart struct {
	cartRepo         cart.CartRepository
	pizzaRepo        readmodel.PizzaRepository
	pizzaPriceRepo   readmodel.PizzaPriceRepository
	toppingPriceRepo readmodel.ToppingPriceRepository
}

func NewGetCart(
	cartRepo cart.CartRepository,
	pizzaRepo readmodel.PizzaRepository,
	pizzaPriceRepo readmodel.PizzaPriceRepository,
	toppingPriceRepo readmodel.ToppingPriceRepository,
) *GetCart {
	return &GetCart{
		cartRepo:         cartRepo,
		pizzaRepo:        pizzaRepo,
		pizzaPriceRepo:   pizzaPriceRepo,
		toppingPriceRepo: toppingPriceRepo,
	}
}

func (uc *GetCart) Execute(ctx context.Context, customerID uuid.UUID) (cartapp.GetCartResponse, error) {
	c, err := uc.cartRepo.FindByCustomer(ctx, customerID)
	if err != nil {
		return cartapp.GetCartResponse{}, fmt.Errorf("failed to look up cart: %w", err)
	}
	if c == nil {
		return cartapp.GetCartResponse{Items: []cartapp.CartItemView{}}, nil
	}

	toppingPrices, err := uc.toppingPriceRepo.ListByRestaurant(ctx, c.RestaurantID)
	if err != nil {
		return cartapp.GetCartResponse{}, fmt.Errorf("failed to look up topping prices: %w", err)
	}

	toppingByID := make(map[uuid.UUID]readmodel.ToppingPrice, len(toppingPrices))
	for _, tp := range toppingPrices {
		toppingByID[tp.ToppingID] = tp
	}

	subtotal := decimal.Zero
	views := make([]cartapp.CartItemView, 0, len(c.Items))

	for _, item := range c.Items {
		view, lineTotal, err := uc.resolveItemView(ctx, item, toppingByID)
		if err != nil {
			return cartapp.GetCartResponse{}, err
		}

		subtotal = subtotal.Add(lineTotal)
		views = append(views, view)
	}

	return cartapp.GetCartResponse{
		CartID:       c.ID,
		RestaurantID: c.RestaurantID,
		Items:        views,
		Subtotal:     money.Money(subtotal),
	}, nil
}

func (uc *GetCart) resolveItemView(
	ctx context.Context,
	item cart.CartItem,
	toppingByID map[uuid.UUID]readmodel.ToppingPrice,
) (cartapp.CartItemView, decimal.Decimal, error) {
	view := cartapp.CartItemView{
		ItemID:   item.ID,
		PizzaID:  item.PizzaID,
		SizeID:   item.SizeID,
		Quantity: item.Quantity,
	}

	pizza, err := uc.pizzaRepo.FindByID(ctx, item.PizzaID)
	if err != nil && !errors.Is(err, apperr.ErrNotFound) {
		return cartapp.CartItemView{}, decimal.Zero, fmt.Errorf("failed to look up pizza: %w", err)
	}

	unitPrice := decimal.Zero
	available := err == nil

	if available {
		view.PizzaName = pizza.Name

		prices, err := uc.pizzaPriceRepo.ListByPizza(ctx, item.PizzaID)
		if err != nil {
			return cartapp.CartItemView{}, decimal.Zero, fmt.Errorf("failed to look up pizza prices: %w", err)
		}

		priceFound := false
		for _, p := range prices {
			if p.SizeID == item.SizeID && p.IsActive {
				unitPrice = p.Price
				view.DiameterCm = p.DiameterCm
				priceFound = true
				break
			}
		}

		available = priceFound
	}

	toppingViews := make([]cartapp.CartToppingView, 0, len(item.ExtraToppingIDs))
	for _, toppingID := range item.ExtraToppingIDs {
		toppingView := cartapp.CartToppingView{ToppingID: toppingID}

		tp, ok := toppingByID[toppingID]
		if !ok {
			available = false
			toppingViews = append(toppingViews, toppingView)
			continue
		}

		toppingView.Name = tp.Name
		extra := money.Money(tp.ExtraPrice)
		toppingView.ExtraPrice = &extra
		unitPrice = unitPrice.Add(tp.ExtraPrice)

		toppingViews = append(toppingViews, toppingView)
	}

	view.ExtraToppings = toppingViews
	view.Available = available

	if !available {
		return view, decimal.Zero, nil
	}

	unitPriceMoney := money.Money(unitPrice)
	view.UnitPrice = &unitPriceMoney

	lineTotal := unitPrice.Mul(decimal.NewFromInt(int64(item.Quantity)))
	lineTotalMoney := money.Money(lineTotal)
	view.LineTotal = &lineTotalMoney

	return view, lineTotal, nil
}
