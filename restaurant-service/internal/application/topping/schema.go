package topping

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"restaurant-service/internal/shared/money"
)

type ToppingPriceInput struct {
	ToppingID  uuid.UUID       `json:"toppingId" binding:"required"`
	ExtraPrice decimal.Decimal `json:"extraPrice"`
}

type SetToppingPricesRequest struct {
	Prices []ToppingPriceInput `json:"prices" binding:"required,min=1,dive"`
}

type ToppingResponse struct {
	ToppingID  uuid.UUID    `json:"toppingId"`
	Name       string       `json:"name"`
	ExtraPrice *money.Money `json:"extraPrice,omitempty"`
}

type ToppingPriceResponse struct {
	ToppingID  uuid.UUID   `json:"toppingId"`
	Name       string      `json:"name"`
	ExtraPrice money.Money `json:"extraPrice"`
}
