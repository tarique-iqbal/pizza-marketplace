package cart

import "github.com/google/uuid"

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
