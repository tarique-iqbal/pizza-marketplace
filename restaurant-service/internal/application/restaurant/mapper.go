package restaurant

import (
	"encoding/json"
	"fmt"

	"gorm.io/datatypes"

	"restaurant-service/internal/domain/restaurant"
)

func ToRestaurantResponse(r *restaurant.Restaurant) RestaurantResponse {
	return RestaurantResponse{
		ID:   r.ID,
		Name: r.Name,
		Slug: r.Slug,
		Contact: ContactResponse{
			Email:   r.Email,
			Phone:   r.Phone,
			Website: r.Website,
		},
		Address:        r.Address,
		DisplayAddress: formatAddress(r.Address),
		Lat:            r.Lat,
		Lon:            r.Lon,
		Delivery: DeliveryResponse{
			Type:         r.DeliveryType,
			RadiusKm:     r.DeliveryKm,
			Fee:          r.DeliveryFee,
			MinimumOrder: r.MinimumOrder,
		},
		Payout:       toPayoutResponse(r.PayoutDetails),
		Pickup:       r.Pickup,
		Currency:     r.Currency,
		Rating:       r.Rating,
		TotalReviews: r.TotalReviews,
		Tags:         parseTags(r.Tags),
		OpeningHours: r.OpeningHours,
		Status:       r.Status,
		CreatedAt:    r.CreatedAt,
		UpdatedAt:    r.UpdatedAt,
	}
}

func toPayoutResponse(pd *restaurant.PayoutDetails) PayoutResponse {
	if pd == nil {
		return PayoutResponse{}
	}

	return PayoutResponse{
		AccountHolder: pd.AccountHolder,
		IBAN:          pd.IBAN,
		BIC:           pd.BIC,
		BankName:      pd.BankName,
		Status:        pd.Status,
	}
}

func formatAddress(a Address) string {
	return fmt.Sprintf(
		"%s %s, %s %s",
		a.Street,
		a.House,
		a.PostalCode,
		a.City,
	)
}

func parseTags(data datatypes.JSON) []string {
	if len(data) == 0 {
		return []string{}
	}

	var tags []string

	if err := json.Unmarshal(data, &tags); err != nil {
		return []string{}
	}

	return tags
}
