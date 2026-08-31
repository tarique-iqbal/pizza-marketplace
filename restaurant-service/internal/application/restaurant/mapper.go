package restaurant

import (
	"fmt"

	payoutapp "restaurant-service/internal/application/payout"
	payoutdomain "restaurant-service/internal/domain/payout"
	"restaurant-service/internal/domain/restaurant"
	"restaurant-service/internal/shared/money"
)

func ToRestaurantResponse(r *restaurant.Restaurant, pd *payoutdomain.PayoutDetails) RestaurantResponse {
	return RestaurantResponse{
		ID:   r.ID,
		Name: r.Name,
		Slug: r.Slug,
		Contact: ContactResponse{
			Email:   r.Email,
			Phone:   r.Phone,
			Website: r.Website,
		},
		Address:           r.Address,
		DisplayAddress:    formatAddress(r.Address),
		EstimatedDelivery: formatDeliveryTime(r.DeliveryTimeMin, r.DeliveryTimeMax),
		Lat:               r.Lat,
		Lon:               r.Lon,
		Timezone:          r.Timezone,
		Delivery: DeliveryResponse{
			Type:                r.DeliveryType,
			RadiusKm:            r.DeliveryKm,
			EstimatedMinutesMin: r.DeliveryTimeMin,
			EstimatedMinutesMax: r.DeliveryTimeMax,
			Fee:                 money.Money(r.DeliveryFee),
			MinimumOrder:        money.Money(r.MinimumOrder),
		},
		Payout:       payoutapp.ToPayoutResponse(pd),
		Pickup:       r.Pickup,
		Currency:     r.Currency,
		Rating:       r.Rating,
		TotalReviews: r.TotalReviews,
		Tags:         tagsToStrings(r.Tags),
		OpeningHours: r.OpeningHours,
		Status:       r.Status,
		CreatedAt:    r.CreatedAt,
		UpdatedAt:    r.UpdatedAt,
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

func formatDeliveryTime(min, max *int16) string {
	if min == nil || max == nil {
		return ""
	}

	return fmt.Sprintf("%d-%d min", *min, *max)
}

func tagsToStrings(tags []restaurant.RestaurantTag) []string {
	result := make([]string, len(tags))
	for i, tag := range tags {
		result[i] = string(tag)
	}

	return result
}
