package pizza

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	toppingapp "restaurant-service/internal/application/topping"
	"restaurant-service/internal/domain/pizza"
	"restaurant-service/internal/domain/topping"
	"restaurant-service/internal/shared/money"
)

func ToPizzaResponse(
	p *pizza.Pizza,
	prices []pizza.PizzaPrice,
	sizeByID map[uuid.UUID]pizza.PizzaSize,
	toppingIDs []uuid.UUID,
	toppingByID map[uuid.UUID]topping.Topping,
	priceByToppingID map[uuid.UUID]decimal.Decimal,
) PizzaResponse {
	priceResponses := make([]PizzaPriceResponse, 0, len(prices))
	for _, price := range prices {
		size := sizeByID[price.SizeID]
		priceResponses = append(priceResponses, PizzaPriceResponse{
			SizeID:     price.SizeID,
			DiameterCm: size.DiameterCm,
			Price:      money.Money(price.Price),
			IsActive:   price.IsActive,
		})
	}

	toppingResponses := make([]toppingapp.ToppingResponse, 0, len(toppingIDs))
	for _, toppingID := range toppingIDs {
		t := toppingByID[toppingID]

		var extraPrice *money.Money
		if price, ok := priceByToppingID[toppingID]; ok {
			m := money.Money(price)
			extraPrice = &m
		}

		toppingResponses = append(toppingResponses, toppingapp.ToppingResponse{
			ToppingID:  toppingID,
			Name:       t.Name,
			ExtraPrice: extraPrice,
		})
	}

	return PizzaResponse{
		ID:           p.ID,
		Name:         p.Name,
		Image:        p.Image,
		IsVegetarian: p.IsVegetarian,
		Status:       p.Status,
		SortOrder:    p.SortOrder,
		Prices:       priceResponses,
		Toppings:     toppingResponses,
		CreatedAt:    p.CreatedAt,
		UpdatedAt:    p.UpdatedAt,
	}
}
