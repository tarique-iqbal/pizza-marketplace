package order

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type OrderItem struct {
	ID           uuid.UUID       `gorm:"type:uuid;primaryKey"`
	OrderID      uuid.UUID       `gorm:"type:uuid;not null;index"`
	PizzaID      uuid.UUID       `gorm:"type:uuid;not null"`
	SizeID       uuid.UUID       `gorm:"type:uuid;not null"`
	PizzaName    string          `gorm:"size:128;not null"`
	SizeDiameter int16           `gorm:"not null"`
	ToppingIDs   []uuid.UUID     `gorm:"column:toppings;type:jsonb;serializer:json;not null;default:'[]'"`
	Quantity     int16           `gorm:"not null;check:quantity > 0"`
	UnitPrice    decimal.Decimal `gorm:"type:numeric(6,2);not null"`
	TotalPrice   decimal.Decimal `gorm:"type:numeric(7,2);not null"`
}

func (OrderItem) TableName() string {
	return "order_items"
}
