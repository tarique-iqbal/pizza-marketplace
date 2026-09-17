package customer

import (
	"time"

	"github.com/google/uuid"
)

type GetProfileResponse struct {
	ID        uuid.UUID  `json:"id"`
	Email     string     `json:"email"`
	FirstName string     `json:"firstName"`
	LastName  string     `json:"lastName"`
	Phone     *string    `json:"phone,omitempty"`
	UpdatedAt *time.Time `json:"updatedAt,omitempty"`
}
