package customer

import (
	"customer-service/internal/domain/customer"
)

func ToProfileResponse(c *customer.Customer) GetProfileResponse {
	return GetProfileResponse{
		ID:        c.ID,
		Email:     c.Email,
		FirstName: c.FirstName,
		LastName:  c.LastName,
		Phone:     c.Phone,
		UpdatedAt: c.UpdatedAt,
	}
}
