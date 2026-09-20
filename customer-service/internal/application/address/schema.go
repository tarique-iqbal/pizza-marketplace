package address

import (
	"time"

	"github.com/google/uuid"
)

type AddressResponse struct {
	ID         uuid.UUID `json:"id"`
	House      string    `json:"house"`
	Street     string    `json:"street"`
	City       string    `json:"city"`
	PostalCode string    `json:"postalCode"`
	IsDefault  bool      `json:"isDefault"`
	CreatedAt  time.Time `json:"createdAt"`
}
