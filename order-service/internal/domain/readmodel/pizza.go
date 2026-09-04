package readmodel

import (
	"time"

	"github.com/google/uuid"
)

type PizzaStatus string

const (
	PizzaAvailable   PizzaStatus = "available"
	PizzaUnavailable PizzaStatus = "unavailable"
	PizzaArchived    PizzaStatus = "archived"
)

type Pizza struct {
	ID           uuid.UUID   `gorm:"type:uuid;primaryKey"`
	RestaurantID uuid.UUID   `gorm:"type:uuid;not null"`
	Name         string      `gorm:"size:255;not null"`
	Status       PizzaStatus `gorm:"type:pizza_status_enum;not null;default:'available'"`
	UpdatedAt    time.Time   `gorm:"type:timestamptz;not null"`
}

func (Pizza) TableName() string {
	return "pizzas"
}
