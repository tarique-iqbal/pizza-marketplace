package restaurant

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	payoutapp "restaurant-service/internal/application/payout"
	pizzaapp "restaurant-service/internal/application/pizza"
	"restaurant-service/internal/domain/restaurant"
	"restaurant-service/internal/shared/money"
)

type Address = restaurant.Address
type DeliveryType = restaurant.DeliveryType
type RestaurantStatus = restaurant.RestaurantStatus
type RestaurantTag = restaurant.RestaurantTag
type OpeningHoursResponse = restaurant.OpeningHours

type UpdateAddressRequest struct {
	House      string `json:"house" binding:"required,max=64"`
	Street     string `json:"street" binding:"required,max=128"`
	City       string `json:"city" binding:"required,alphaunicode,max=64"`
	PostalCode string `json:"postalCode" binding:"required"`
}

type UpdateContactRequest struct {
	Email   *string `json:"email" binding:"required,email,max=255"`
	Phone   *string `json:"phone" binding:"required,phone"`
	Website *string `json:"website" binding:"omitempty,url,max=255"`
}

type UpdateDeliveryRequest struct {
	Pickup       bool            `json:"pickup"`
	DeliveryType DeliveryType    `json:"deliveryType" binding:"required,oneof=own external none"`
	DeliveryKm   *int16          `json:"deliveryKm" binding:"omitempty,gte=1,lte=25"`
	DeliveryFee  decimal.Decimal `json:"deliveryFee"`
	MinimumOrder decimal.Decimal `json:"minimumOrder"`
}

type UpdateTagsRequest struct {
	Tags []RestaurantTag `json:"tags" binding:"unique,dive,oneof=vegetarian vegan glutenfree halal"`
}

type DayRangeRequest struct {
	Open  string `json:"open" binding:"required,hhmm"`
	Close string `json:"close" binding:"required,hhmm"`
}

type UpdateOpeningHoursRequest struct {
	Monday    []DayRangeRequest `json:"monday" binding:"dive"`
	Tuesday   []DayRangeRequest `json:"tuesday" binding:"dive"`
	Wednesday []DayRangeRequest `json:"wednesday" binding:"dive"`
	Thursday  []DayRangeRequest `json:"thursday" binding:"dive"`
	Friday    []DayRangeRequest `json:"friday" binding:"dive"`
	Saturday  []DayRangeRequest `json:"saturday" binding:"dive"`
	Sunday    []DayRangeRequest `json:"sunday" binding:"dive"`
}

type RestaurantResponse struct {
	ID             uuid.UUID                `json:"id"`
	Name           string                   `json:"name"`
	Slug           *string                  `json:"slug,omitempty"`
	Contact        ContactResponse          `json:"contact"`
	Address        Address                  `json:"address"`
	DisplayAddress string                   `json:"displayAddress"`
	Lat            *float64                 `json:"lat,omitempty"`
	Lon            *float64                 `json:"lon,omitempty"`
	Delivery       DeliveryResponse         `json:"delivery"`
	Payout         payoutapp.PayoutResponse `json:"payout"`
	Currency       string                   `json:"currency"`
	Rating         float64                  `json:"rating"`
	TotalReviews   int32                    `json:"totalReviews"`
	Pickup         bool                     `json:"pickup"`
	Tags           []string                 `json:"tags"`
	OpeningHours   OpeningHoursResponse     `json:"openingHours"`
	Status         RestaurantStatus         `json:"status"`
	CreatedAt      time.Time                `json:"createdAt"`
	UpdatedAt      *time.Time               `json:"updatedAt,omitempty"`
}

type RestaurantWithPizzasResponse struct {
	RestaurantResponse
	Pizzas []pizzaapp.PizzaResponse `json:"pizzas"`
}

type ContactResponse struct {
	Email   *string `json:"email,omitempty"`
	Phone   *string `json:"phone,omitempty"`
	Website *string `json:"website,omitempty"`
}

type DeliveryResponse struct {
	Type         DeliveryType `json:"type"`
	RadiusKm     *int16       `json:"radiusKm,omitempty"`
	Fee          money.Money  `json:"fee"`
	MinimumOrder money.Money  `json:"minimumOrder"`
}
