package pizza

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	toppingapp "restaurant-service/internal/application/topping"
	"restaurant-service/internal/domain/pizza"
	"restaurant-service/internal/shared/money"
)

type PizzaStatus = pizza.PizzaStatus

type CreatePizzaRequest struct {
	Name         string      `json:"name" binding:"required,max=255"`
	Image        *string     `json:"image" binding:"omitempty,max=255"`
	IsVegetarian *bool       `json:"isVegetarian" binding:"omitempty"`
	Status       *string     `json:"status" binding:"omitempty,oneof=available unavailable archived"`
	SortOrder    int         `json:"sortOrder" binding:"gte=0"`
	ToppingIDs   []uuid.UUID `json:"toppingIds"`
}

type UpdatePizzaRequest = CreatePizzaRequest

type PizzaPriceInput struct {
	SizeID uuid.UUID       `json:"sizeId" binding:"required"`
	Price  decimal.Decimal `json:"price"`
}

type SetPizzaPricesRequest struct {
	Prices []PizzaPriceInput `json:"prices" binding:"required,min=1,dive"`
}

type PizzaResponse struct {
	ID           uuid.UUID                    `json:"id"`
	Name         string                       `json:"name"`
	Image        *string                      `json:"image,omitempty"`
	IsVegetarian bool                         `json:"isVegetarian"`
	Status       PizzaStatus                  `json:"status"`
	SortOrder    int                          `json:"sortOrder"`
	Prices       []PizzaPriceResponse         `json:"prices"`
	Toppings     []toppingapp.ToppingResponse `json:"toppings"`
	CreatedAt    time.Time                    `json:"createdAt"`
	UpdatedAt    *time.Time                   `json:"updatedAt,omitempty"`
}

type PizzaPriceResponse struct {
	SizeID     uuid.UUID   `json:"sizeId"`
	DiameterCm int16       `json:"diameterCm"`
	Price      money.Money `json:"price"`
	IsActive   bool        `json:"isActive"`
}

func (p PizzaResponse) HasActivePrice() bool {
	for _, price := range p.Prices {
		if price.IsActive {
			return true
		}
	}

	return false
}
