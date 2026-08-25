package restaurant

import (
	"time"

	"github.com/google/uuid"

	pizzaapp "restaurant-service/internal/application/pizza"
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
	RestaurantID   uuid.UUID                `json:"restaurant_id"`
	RestaurantName string                   `json:"restaurant_name"`
	EventName      string                   `json:"event_name"`
	Slug           string                   `json:"slug"`
	Contact        ContactResponse          `json:"contact"`
	Address        Address                  `json:"address"`
	Lat            float64                  `json:"lat"`
	Lon            float64                  `json:"lon"`
	Delivery       DeliveryResponse         `json:"delivery"`
	Currency       string                   `json:"currency"`
	Rating         float64                  `json:"rating"`
	TotalReviews   int32                    `json:"total_reviews"`
	Pickup         bool                     `json:"pickup"`
	Tags           []string                 `json:"tags"`
	OpeningHours   OpeningHoursResponse     `json:"opening_hours"`
	Pizzas         []pizzaapp.PizzaResponse `json:"pizzas"`
	UpdatedAt      time.Time                `json:"updated_at"`
	OccurredAt     time.Time                `json:"occurred_at"`
}

func (RestaurantLaunchedPayload) GetEventName() string {
	return "restaurant.launched"
}

func NewRestaurantLaunchedPayload(
	e restaurant.RestaurantLaunched,
	r *restaurant.Restaurant,
	pizzas []pizzaapp.PizzaResponse,
) RestaurantLaunchedPayload {
	payload := RestaurantLaunchedPayload{
		RestaurantID:   e.RestaurantID,
		RestaurantName: r.Name,
		Slug:           *r.Slug,
		Contact: ContactResponse{
			Email:   r.Email,
			Phone:   r.Phone,
			Website: r.Website,
		},
		Address: r.Address,
		Lat:     *r.Lat,
		Lon:     *r.Lon,
		Delivery: DeliveryResponse{
			Type:         r.DeliveryType,
			RadiusKm:     r.DeliveryKm,
			Fee:          money.Money(r.DeliveryFee),
			MinimumOrder: money.Money(r.MinimumOrder),
		},
		Currency:     r.Currency,
		Rating:       r.Rating,
		TotalReviews: r.TotalReviews,
		Pickup:       r.Pickup,
		Tags:         parseTags(r.Tags),
		OpeningHours: r.OpeningHours,
		Pizzas:       pizzas,
		UpdatedAt:    *r.UpdatedAt,
		OccurredAt:   e.OccurredAt,
	}
	payload.EventName = payload.GetEventName()

	return payload
}

type RestaurantUpdatedPayload struct {
	RestaurantID   uuid.UUID            `json:"restaurant_id"`
	RestaurantName string               `json:"restaurant_name"`
	EventName      string               `json:"event_name"`
	Slug           string               `json:"slug"`
	Contact        ContactResponse      `json:"contact"`
	Address        Address              `json:"address"`
	Lat            float64              `json:"lat"`
	Lon            float64              `json:"lon"`
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
		RestaurantName: r.Name,
		Slug:           *r.Slug,
		Contact: ContactResponse{
			Email:   r.Email,
			Phone:   r.Phone,
			Website: r.Website,
		},
		Address: r.Address,
		Lat:     *r.Lat,
		Lon:     *r.Lon,
		Delivery: DeliveryResponse{
			Type:         r.DeliveryType,
			RadiusKm:     r.DeliveryKm,
			Fee:          money.Money(r.DeliveryFee),
			MinimumOrder: money.Money(r.MinimumOrder),
		},
		Currency:     r.Currency,
		Rating:       r.Rating,
		TotalReviews: r.TotalReviews,
		Pickup:       r.Pickup,
		Tags:         parseTags(r.Tags),
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
	var updatedAt time.Time
	if pizza.UpdatedAt != nil {
		updatedAt = *pizza.UpdatedAt
	}

	payload := PizzaUpdatedPayload{
		RestaurantID: e.RestaurantID,
		Pizza:        pizza,
		UpdatedAt:    updatedAt,
		OccurredAt:   e.OccurredAt,
	}
	payload.EventName = payload.GetEventName()

	return payload
}
