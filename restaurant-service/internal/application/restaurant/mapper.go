package restaurant

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/datatypes"

	payoutapp "restaurant-service/internal/application/payout"
	toppingapp "restaurant-service/internal/application/topping"
	payoutdomain "restaurant-service/internal/domain/payout"
	"restaurant-service/internal/domain/restaurant"
	toppingdomain "restaurant-service/internal/domain/topping"
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
		Address:        r.Address,
		DisplayAddress: formatAddress(r.Address),
		Lat:            r.Lat,
		Lon:            r.Lon,
		Delivery: DeliveryResponse{
			Type:         r.DeliveryType,
			RadiusKm:     r.DeliveryKm,
			Fee:          money.Money(r.DeliveryFee),
			MinimumOrder: money.Money(r.MinimumOrder),
		},
		Payout:       payoutapp.ToPayoutResponse(pd),
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

func ToPizzaResponse(
	pizza *restaurant.Pizza,
	prices []restaurant.PizzaPrice,
	sizeByID map[uuid.UUID]restaurant.PizzaSize,
	toppingIDs []uuid.UUID,
	toppingByID map[uuid.UUID]toppingdomain.Topping,
	priceByToppingID map[uuid.UUID]decimal.Decimal,
) PizzaResponse {
	priceResponses := make([]PizzaPriceResponse, 0, len(prices))
	for _, p := range prices {
		size := sizeByID[p.SizeID]
		priceResponses = append(priceResponses, PizzaPriceResponse{
			SizeID:     p.SizeID,
			DiameterCm: size.DiameterCm,
			Price:      money.Money(p.Price),
			IsActive:   p.IsActive,
		})
	}

	toppingResponses := make([]toppingapp.ToppingResponse, 0, len(toppingIDs))
	for _, toppingID := range toppingIDs {
		t := toppingByID[toppingID]

		var extraPrice *money.Money
		if price, ok := priceByToppingID[toppingID]; ok {
			m := money.Money(price)
			extraPrice = &m
		}

		toppingResponses = append(toppingResponses, toppingapp.ToppingResponse{
			ToppingID:  toppingID,
			Name:       t.Name,
			ExtraPrice: extraPrice,
		})
	}

	return PizzaResponse{
		ID:           pizza.ID,
		Name:         pizza.Name,
		Image:        pizza.Image,
		IsVegetarian: pizza.IsVegetarian,
		Status:       pizza.Status,
		SortOrder:    pizza.SortOrder,
		Prices:       priceResponses,
		Toppings:     toppingResponses,
		CreatedAt:    pizza.CreatedAt,
		UpdatedAt:    pizza.UpdatedAt,
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
