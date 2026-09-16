package customer

import (
	"time"

	"github.com/google/uuid"
)

type Address struct {
	ID         uuid.UUID  `gorm:"type:uuid;primaryKey"`
	CustomerID uuid.UUID  `gorm:"type:uuid;not null;index"`
	House      string     `gorm:"size:20;not null"`
	Street     string     `gorm:"size:255;not null"`
	City       string     `gorm:"size:100;not null"`
	PostalCode string     `gorm:"size:20;not null"`
	IsDefault  bool       `gorm:"not null"`
	CreatedAt  time.Time  `gorm:"type:timestamptz;not null;autoCreateTime"`
	UpdatedAt  *time.Time `gorm:"type:timestamptz;autoUpdateTime;default:null"`
}

func (Address) TableName() string {
	return "customer_addresses"
}

func NewAddress(
	customerID uuid.UUID,
	house, street, city, postalCode string,
	isDefault bool,
) (*Address, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}

	return &Address{
		ID:         id,
		CustomerID: customerID,
		House:      house,
		Street:     street,
		City:       city,
		PostalCode: postalCode,
		IsDefault:  isDefault,
	}, nil
}
