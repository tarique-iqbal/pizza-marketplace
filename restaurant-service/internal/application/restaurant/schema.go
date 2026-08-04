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

type UpdateAddressRequest struct {
	House      string `json:"house" binding:"required,max=64"`
	Street     string `json:"street" binding:"required,max=128"`
	City       string `json:"city" binding:"required,alphaunicode,max=64"`
	PostalCode string `json:"postalCode" binding:"required"`
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

type RestaurantResponse struct {
	ID             uuid.UUID        `json:"id"`
	Name           string           `json:"name"`
	Slug           *string          `json:"slug,omitempty"`
	Contact        ContactResponse  `json:"contact"`
	Address        Address          `json:"address"`
	DisplayAddress string           `json:"displayAddress"`
	Lat            *float64         `json:"lat,omitempty"`
	Lon            *float64         `json:"lon,omitempty"`
	Delivery       DeliveryResponse `json:"delivery"`
	Payout         PayoutResponse   `json:"payout"`
	Currency       string           `json:"currency"`
	Rating         float64          `json:"rating"`
	TotalReviews   int32            `json:"totalReviews"`
	Pickup         bool             `json:"pickup"`
	Tags           []string         `json:"tags"`
	OpeningHours   any              `json:"openingHours"`
	Status         RestaurantStatus `json:"status"`
	CreatedAt      time.Time        `json:"createdAt"`
	UpdatedAt      *time.Time       `json:"updatedAt,omitempty"`
}

type ContactResponse struct {
	Email   *string `json:"email,omitempty"`
	Phone   *string `json:"phone,omitempty"`
	Website *string `json:"website,omitempty"`
}

type DeliveryResponse struct {
	Type         DeliveryType    `json:"type"`
	RadiusKm     *int16          `json:"radiusKm,omitempty"`
	Fee          decimal.Decimal `json:"fee"`
	MinimumOrder decimal.Decimal `json:"minimumOrder"`
}

type PayoutResponse struct {
	AccountHolder string       `json:"accountHolder"`
	IBAN          string       `json:"iban"`
	BIC           string       `json:"bic"`
	BankName      string       `json:"bankName"`
	Status        PayoutStatus `json:"status,omitempty"`
}
