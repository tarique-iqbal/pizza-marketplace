package cart

import (
	"sort"

	"github.com/google/uuid"
)

type CartItem struct {
	ID         uuid.UUID   `gorm:"type:uuid;primaryKey"`
	CartID     uuid.UUID   `gorm:"type:uuid;not null"`
	PizzaID    uuid.UUID   `gorm:"type:uuid;not null"`
	SizeID     uuid.UUID   `gorm:"type:uuid;not null"`
	Quantity   int16       `gorm:"not null;check:quantity > 0"`
	ToppingIDs []uuid.UUID `gorm:"column:toppings;type:jsonb;serializer:json;not null;default:'[]'"`
}

func (CartItem) TableName() string {
	return "cart_items"
}

func NewCartItem(id, pizzaID, sizeID uuid.UUID, quantity int16, toppingIDs []uuid.UUID) CartItem {
	sorted := make([]uuid.UUID, len(toppingIDs))
	copy(sorted, toppingIDs)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].String() < sorted[j].String()
	})

	return CartItem{
		ID:         id,
		PizzaID:    pizzaID,
		SizeID:     sizeID,
		Quantity:   quantity,
		ToppingIDs: sorted,
	}
}
