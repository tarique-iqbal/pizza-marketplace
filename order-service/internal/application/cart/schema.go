package cart

import (
	"github.com/google/uuid"

	"order-service/internal/shared/money"
)

type AddItemRequest struct {
	PizzaID    uuid.UUID   `json:"pizzaId" binding:"required,uuid"`
	SizeID     uuid.UUID   `json:"sizeId" binding:"required,uuid"`
	Quantity   int16       `json:"quantity" binding:"required,min=1"`
	ToppingIDs []uuid.UUID `json:"toppingIds" binding:"dive,uuid"`
}

type AddItemResponse struct {
	PizzaID    uuid.UUID   `json:"pizzaId"`
	SizeID     uuid.UUID   `json:"sizeId"`
	Quantity   int16       `json:"quantity"`
	ToppingIDs []uuid.UUID `json:"toppingIds"`
}

type UpdateItemQuantityRequest struct {
	Quantity int16 `json:"quantity" binding:"required,min=1"`
}

type UpdateItemQuantityResponse struct {
	ItemID   uuid.UUID `json:"itemId"`
	Quantity int16     `json:"quantity"`
}

type CartToppingView struct {
	ToppingID  uuid.UUID    `json:"toppingId"`
	Name       string       `json:"name,omitempty"`
	ExtraPrice *money.Money `json:"extraPrice,omitempty"`
}

type CartItemView struct {
	ItemID     uuid.UUID         `json:"itemId"`
	PizzaID    uuid.UUID         `json:"pizzaId"`
	PizzaName  string            `json:"pizzaName,omitempty"`
	SizeID     uuid.UUID         `json:"sizeId"`
	DiameterCm int16             `json:"diameterCm,omitempty"`
	Quantity   int16             `json:"quantity"`
	Toppings   []CartToppingView `json:"toppings"`
	UnitPrice  *money.Money      `json:"unitPrice,omitempty"`
	LineTotal  *money.Money      `json:"lineTotal,omitempty"`
	Available  bool              `json:"available"`
}

type GetCartResponse struct {
	CartID       uuid.UUID      `json:"cartId"`
	RestaurantID uuid.UUID      `json:"restaurantId"`
	Items        []CartItemView `json:"items"`
	Subtotal     money.Money    `json:"subtotal"`
}
