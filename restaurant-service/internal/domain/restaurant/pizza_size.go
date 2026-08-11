package restaurant

import (
	"time"

	"github.com/google/uuid"
)

type PizzaSize struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey"`
	DiameterCm int16     `gorm:"not null;unique;check:diameter_cm BETWEEN 20 AND 45"`
	CreatedAt  time.Time `gorm:"type:timestamptz;autoCreateTime"`
}

func (PizzaSize) TableName() string {
	return "pizza_sizes"
}
