package restaurant

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"restaurant-service/internal/domain/restaurant"
)

type Address = restaurant.Address
type DeliveryType = restaurant.DeliveryType
type RestaurantStatus = restaurant.RestaurantStatus
type PayoutStatus = restaurant.PayoutStatus
type OpeningHoursResponse = restaurant.OpeningHours
type PizzaStatus = restaurant.PizzaStatus

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

type CreatePayoutRequest struct {
	AccountHolder string `json:"accountHolder" binding:"required,max=100"`
	IBAN          string `json:"iban" binding:"required,iban"`
	BIC           string `json:"bic" binding:"required,bic"`
	BankName      string `json:"bankName" binding:"required,max=100"`
}

type UpdatePayoutRequest = CreatePayoutRequest

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

type CreatePizzaRequest struct {
	Name         string      `json:"name" binding:"required,max=255"`
	Image        *string     `json:"image" binding:"omitempty,max=255"`
	IsVegetarian *bool       `json:"isVegetarian" binding:"omitempty"`
	Status       *string     `json:"status" binding:"omitempty,oneof=available unavailable archived"`
	SortOrder    int         `json:"sortOrder" binding:"gte=0"`
	ToppingIDs   []uuid.UUID `json:"toppingIds"`
}

type UpdatePizzaRequest = CreatePizzaRequest

type PizzaPriceInput struct {
	SizeID uuid.UUID       `json:"sizeId" binding:"required"`
	Price  decimal.Decimal `json:"price"`
}

type SetPizzaPricesRequest struct {
	Prices []PizzaPriceInput `json:"prices" binding:"required,min=1,dive"`
}

type ToppingPriceInput struct {
	ToppingID  uuid.UUID       `json:"toppingId" binding:"required"`
	ExtraPrice decimal.Decimal `json:"extraPrice"`
}

type SetToppingPricesRequest struct {
	Prices []ToppingPriceInput `json:"prices" binding:"required,min=1,dive"`
}

type RestaurantResponse struct {
	ID             uuid.UUID            `json:"id"`
	Name           string               `json:"name"`
	Slug           *string              `json:"slug,omitempty"`
	Contact        ContactResponse      `json:"contact"`
	Address        Address              `json:"address"`
	DisplayAddress string               `json:"displayAddress"`
	Lat            *float64             `json:"lat,omitempty"`
	Lon            *float64             `json:"lon,omitempty"`
	Delivery       DeliveryResponse     `json:"delivery"`
	Payout         PayoutResponse       `json:"payout"`
	Currency       string               `json:"currency"`
	Rating         float64              `json:"rating"`
	TotalReviews   int32                `json:"totalReviews"`
	Pickup         bool                 `json:"pickup"`
	Tags           []string             `json:"tags"`
	OpeningHours   OpeningHoursResponse `json:"openingHours"`
	Status         RestaurantStatus     `json:"status"`
	CreatedAt      time.Time            `json:"createdAt"`
	UpdatedAt      *time.Time           `json:"updatedAt,omitempty"`
}

type ContactResponse struct {
	Email   *string `json:"email,omitempty"`
	Phone   *string `json:"phone,omitempty"`
	Website *string `json:"website,omitempty"`
}

type DeliveryResponse struct {
	Type         DeliveryType `json:"type"`
	RadiusKm     *int16       `json:"radiusKm,omitempty"`
	Fee          Money        `json:"fee"`
	MinimumOrder Money        `json:"minimumOrder"`
}

type PayoutResponse struct {
	AccountHolder string       `json:"accountHolder"`
	IBAN          string       `json:"iban"`
	BIC           string       `json:"bic"`
	BankName      string       `json:"bankName"`
	Status        PayoutStatus `json:"status,omitempty"`
}

type PizzaResponse struct {
	ID           uuid.UUID            `json:"id"`
	Name         string               `json:"name"`
	Image        *string              `json:"image,omitempty"`
	IsVegetarian bool                 `json:"isVegetarian"`
	Status       PizzaStatus          `json:"status"`
	SortOrder    int                  `json:"sortOrder"`
	Prices       []PizzaPriceResponse `json:"prices"`
	Toppings     []ToppingResponse    `json:"toppings"`
	CreatedAt    time.Time            `json:"createdAt"`
	UpdatedAt    *time.Time           `json:"updatedAt,omitempty"`
}

type PizzaPriceResponse struct {
	SizeID     uuid.UUID `json:"sizeId"`
	DiameterCm int16     `json:"diameterCm"`
	Price      Money     `json:"price"`
	IsActive   bool      `json:"isActive"`
}

type ToppingResponse struct {
	ToppingID  uuid.UUID `json:"toppingId"`
	Name       string    `json:"name"`
	ExtraPrice *Money    `json:"extraPrice,omitempty"`
}

type ToppingPriceResponse struct {
	ToppingID  uuid.UUID `json:"toppingId"`
	Name       string    `json:"name"`
	ExtraPrice Money     `json:"extraPrice"`
}
