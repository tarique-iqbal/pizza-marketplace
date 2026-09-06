package cart

import "github.com/google/uuid"

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
