package customer

import (
	"time"

	"github.com/google/uuid"
)

type Customer struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey"`
	Email     string     `gorm:"size:255;not null"`
	FirstName string     `gorm:"size:128;not null"`
	LastName  string     `gorm:"size:128;not null"`
	Phone     *string    `gorm:"size:32"`
	UpdatedAt *time.Time `gorm:"type:timestamptz;autoUpdateTime;default:null"`
}

func (Customer) TableName() string {
	return "customers"
}

func (c *Customer) SetPhone(phone string) {
	c.Phone = &phone
}
