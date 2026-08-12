package topping

import (
	"restaurant-service/internal/domain/topping"
	"restaurant-service/internal/shared/money"
)

func ToToppingPriceResponse(price topping.ToppingPrice, toppingName string) ToppingPriceResponse {
	return ToppingPriceResponse{
		ToppingID:  price.ToppingID,
		Name:       toppingName,
		ExtraPrice: money.Money(price.ExtraPrice),
	}
}
