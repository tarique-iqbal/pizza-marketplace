package restaurant

import (
	"time"

	"github.com/google/uuid"

	pizzaapp "restaurant-service/internal/application/pizza"
	toppingapp "restaurant-service/internal/application/topping"
	"restaurant-service/internal/domain/restaurant"
	"restaurant-service/internal/shared/money"
)

type RestaurantReadyForReviewPayload struct {
	RestaurantID   uuid.UUID `json:"restaurant_id"`
	RestaurantName string    `json:"restaurant_name"`
	EventName      string    `json:"event_name"`
	OccurredAt     time.Time `json:"occurred_at"`
}

func (RestaurantReadyForReviewPayload) GetEventName() string {
	return "restaurant.ready_for_review"
}

func newRestaurantReadyForReviewPayload(
	e restaurant.RestaurantReadyForReview,
) RestaurantReadyForReviewPayload {
	payload := RestaurantReadyForReviewPayload{
		RestaurantID:   e.RestaurantID,
		RestaurantName: e.RestaurantName,
		OccurredAt:     e.OccurredAt,
	}
	payload.EventName = payload.GetEventName()

	return payload
}

type RestaurantApprovedPayload struct {
	RestaurantID   uuid.UUID `json:"restaurant_id"`
	RestaurantName string    `json:"restaurant_name"`
	Email          string    `json:"email"`
	EventName      string    `json:"event_name"`
	OccurredAt     time.Time `json:"occurred_at"`
}

func (RestaurantApprovedPayload) GetEventName() string {
	return "restaurant.approved"
}

func newRestaurantApprovedPayload(e restaurant.RestaurantApproved) RestaurantApprovedPayload {
	payload := RestaurantApprovedPayload{
		RestaurantID:   e.RestaurantID,
		RestaurantName: e.RestaurantName,
		Email:          e.Email,
		OccurredAt:     e.OccurredAt,
	}
	payload.EventName = payload.GetEventName()

	return payload
}

type RestaurantLaunchedPayload struct {
	RestaurantID   uuid.UUID                         `json:"restaurant_id"`
	OwnerID        uuid.UUID                         `json:"owner_id"`
	RestaurantName string                            `json:"restaurant_name"`
	EventName      string                            `json:"event_name"`
	Slug           string                            `json:"slug"`
	Contact        ContactResponse                   `json:"contact"`
	Address        Address                           `json:"address"`
	Lat            float64                           `json:"lat"`
	Lon            float64                           `json:"lon"`
	Timezone       string                            `json:"timezone"`
	Delivery       DeliveryResponse                  `json:"delivery"`
	Currency       string                            `json:"currency"`
	Rating         float64                           `json:"rating"`
	TotalReviews   int32                             `json:"total_reviews"`
	Pickup         bool                              `json:"pickup"`
	Tags           []string                          `json:"tags"`
	OpeningHours   OpeningHoursResponse              `json:"opening_hours"`
	Pizzas         []pizzaapp.PizzaResponse          `json:"pizzas"`
	ToppingPrices  []toppingapp.ToppingPriceResponse `json:"topping_prices"`
	UpdatedAt      time.Time                         `json:"updated_at"`
	OccurredAt     time.Time                         `json:"occurred_at"`
}

func (RestaurantLaunchedPayload) GetEventName() string {
	return "restaurant.launched"
}

func NewRestaurantLaunchedPayload(
	e restaurant.RestaurantLaunched,
	r *restaurant.Restaurant,
	pizzas []pizzaapp.PizzaResponse,
	toppingPrices []toppingapp.ToppingPriceResponse,
) RestaurantLaunchedPayload {
	payload := RestaurantLaunchedPayload{
		RestaurantID:   e.RestaurantID,
		OwnerID:        r.OwnerID,
		RestaurantName: r.Name,
		Slug:           *r.Slug,
		Contact: ContactResponse{
			Email:   r.Email,
			Phone:   r.Phone,
			Website: r.Website,
		},
		Address:  r.Address,
		Lat:      *r.Lat,
		Lon:      *r.Lon,
		Timezone: *r.Timezone,
		Delivery: DeliveryResponse{
			Type:                r.DeliveryType,
			RadiusKm:            r.DeliveryKm,
			EstimatedMinutesMin: r.DeliveryTimeMin,
			EstimatedMinutesMax: r.DeliveryTimeMax,
			Fee:                 money.Money(r.DeliveryFee),
			MinimumOrder:        money.Money(r.MinimumOrder),
		},
		Currency:      r.Currency,
		Rating:        r.Rating,
		TotalReviews:  r.TotalReviews,
		Pickup:        r.Pickup,
		Tags:          tagsToStrings(r.Tags),
		OpeningHours:  r.OpeningHours,
		Pizzas:        pizzas,
		ToppingPrices: toppingPrices,
		UpdatedAt:     *r.UpdatedAt,
		OccurredAt:    e.OccurredAt,
	}
	payload.EventName = payload.GetEventName()

	return payload
}

type RestaurantUpdatedPayload struct {
	RestaurantID   uuid.UUID            `json:"restaurant_id"`
	OwnerID        uuid.UUID            `json:"owner_id"`
	RestaurantName string               `json:"restaurant_name"`
	EventName      string               `json:"event_name"`
	Slug           string               `json:"slug"`
	Contact        ContactResponse      `json:"contact"`
	Address        Address              `json:"address"`
	Lat            float64              `json:"lat"`
	Lon            float64              `json:"lon"`
	Timezone       string               `json:"timezone"`
	Delivery       DeliveryResponse     `json:"delivery"`
	Currency       string               `json:"currency"`
	Rating         float64              `json:"rating"`
	TotalReviews   int32                `json:"total_reviews"`
	Pickup         bool                 `json:"pickup"`
	Tags           []string             `json:"tags"`
	OpeningHours   OpeningHoursResponse `json:"opening_hours"`
	UpdatedAt      time.Time            `json:"updated_at"`
	OccurredAt     time.Time            `json:"occurred_at"`
}

func (RestaurantUpdatedPayload) GetEventName() string {
	return "restaurant.updated"
}

func NewRestaurantUpdatedPayload(
	e restaurant.RestaurantUpdated,
	r *restaurant.Restaurant,
) RestaurantUpdatedPayload {
	payload := RestaurantUpdatedPayload{
		RestaurantID:   e.RestaurantID,
		OwnerID:        r.OwnerID,
		RestaurantName: r.Name,
		Slug:           *r.Slug,
		Contact: ContactResponse{
			Email:   r.Email,
			Phone:   r.Phone,
			Website: r.Website,
		},
		Address:  r.Address,
		Lat:      *r.Lat,
		Lon:      *r.Lon,
		Timezone: *r.Timezone,
		Delivery: DeliveryResponse{
			Type:                r.DeliveryType,
			RadiusKm:            r.DeliveryKm,
			EstimatedMinutesMin: r.DeliveryTimeMin,
			EstimatedMinutesMax: r.DeliveryTimeMax,
			Fee:                 money.Money(r.DeliveryFee),
			MinimumOrder:        money.Money(r.MinimumOrder),
		},
		Currency:     r.Currency,
		Rating:       r.Rating,
		TotalReviews: r.TotalReviews,
		Pickup:       r.Pickup,
		Tags:         tagsToStrings(r.Tags),
		OpeningHours: r.OpeningHours,
		UpdatedAt:    *r.UpdatedAt,
		OccurredAt:   e.OccurredAt,
	}
	payload.EventName = payload.GetEventName()

	return payload
}

type PizzaUpdatedPayload struct {
	RestaurantID uuid.UUID              `json:"restaurant_id"`
	EventName    string                 `json:"event_name"`
	Pizza        pizzaapp.PizzaResponse `json:"pizza"`
	UpdatedAt    time.Time              `json:"updated_at"`
	OccurredAt   time.Time              `json:"occurred_at"`
}

func (PizzaUpdatedPayload) GetEventName() string {
	return "restaurant.pizza_updated"
}

func NewPizzaUpdatedPayload(
	e restaurant.PizzaUpdated,
	pizza pizzaapp.PizzaResponse,
) PizzaUpdatedPayload {
	payload := PizzaUpdatedPayload{
		RestaurantID: e.RestaurantID,
		Pizza:        pizza,
		UpdatedAt:    *pizza.UpdatedAt,
		OccurredAt:   e.OccurredAt,
	}
	payload.EventName = payload.GetEventName()

	return payload
}

type ToppingPricesUpdatedPayload struct {
	RestaurantID  uuid.UUID                         `json:"restaurant_id"`
	EventName     string                            `json:"event_name"`
	ToppingPrices []toppingapp.ToppingPriceResponse `json:"topping_prices"`
	UpdatedAt     time.Time                         `json:"updated_at"`
	OccurredAt    time.Time                         `json:"occurred_at"`
}

func (ToppingPricesUpdatedPayload) GetEventName() string {
	return "restaurant.topping_prices_updated"
}

func NewToppingPricesUpdatedPayload(
	e restaurant.ToppingPricesUpdated,
	toppingPrices []toppingapp.ToppingPriceResponse,
	updatedAt time.Time,
) ToppingPricesUpdatedPayload {
	payload := ToppingPricesUpdatedPayload{
		RestaurantID:  e.RestaurantID,
		ToppingPrices: toppingPrices,
		UpdatedAt:     updatedAt,
		OccurredAt:    e.OccurredAt,
	}
	payload.EventName = payload.GetEventName()

	return payload
}
