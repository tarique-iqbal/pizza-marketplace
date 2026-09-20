package address

import (
	"customer-service/internal/domain/customer"
)

func ToAddressResponse(a customer.Address) AddressResponse {
	return AddressResponse{
		ID:         a.ID,
		House:      a.House,
		Street:     a.Street,
		City:       a.City,
		PostalCode: a.PostalCode,
		IsDefault:  a.IsDefault,
		CreatedAt:  a.CreatedAt,
	}
}

func ToAddressResponses(addresses []customer.Address) []AddressResponse {
	res := make([]AddressResponse, len(addresses))
	for i, a := range addresses {
		res[i] = ToAddressResponse(a)
	}

	return res
}
