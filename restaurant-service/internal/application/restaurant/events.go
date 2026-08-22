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
	ReadyAt        time.Time `json:"ready_at"`
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
		ReadyAt:        e.ReadyAt,
	}
	payload.EventName = payload.GetEventName()

	return payload
}

type RestaurantApprovedPayload struct {
	RestaurantID   uuid.UUID `json:"restaurant_id"`
	RestaurantName string    `json:"restaurant_name"`
	Email          string    `json:"email"`
	EventName      string    `json:"event_name"`
	ApprovedAt     time.Time `json:"approved_at"`
}

func (RestaurantApprovedPayload) GetEventName() string {
	return "restaurant.approved"
}

func newRestaurantApprovedPayload(e restaurant.RestaurantApproved) RestaurantApprovedPayload {
	payload := RestaurantApprovedPayload{
		RestaurantID:   e.RestaurantID,
		RestaurantName: e.RestaurantName,
		Email:          e.Email,
		ApprovedAt:     e.ApprovedAt,
	}
	payload.EventName = payload.GetEventName()

	return payload
}

type RestaurantLaunchedPayload struct {
	RestaurantID   uuid.UUID                `json:"restaurant_id"`
	RestaurantName string                   `json:"restaurant_name"`
	EventName      string                   `json:"event_name"`
	LaunchedAt     time.Time                `json:"launched_at"`
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
		RestaurantName: e.RestaurantName,
		LaunchedAt:     e.LaunchedAt,
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
	}
	payload.EventName = payload.GetEventName()

	return payload
}

type RestaurantUpdatedPayload struct {
	RestaurantID   uuid.UUID            `json:"restaurant_id"`
	RestaurantName string               `json:"restaurant_name"`
	EventName      string               `json:"event_name"`
	UpdatedAt      time.Time            `json:"updated_at"`
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
		UpdatedAt:      e.UpdatedAt,
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
	}
	payload.EventName = payload.GetEventName()

	return payload
}
