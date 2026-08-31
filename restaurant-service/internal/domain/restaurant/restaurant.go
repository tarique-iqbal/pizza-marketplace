package restaurant

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type DeliveryType string
type RestaurantStatus string
type RestaurantTag string

const (
	DeliveryOwn      DeliveryType = "own"
	DeliveryExternal DeliveryType = "external"
	DeliveryNone     DeliveryType = "none"
)

const (
	TagVegetarian RestaurantTag = "vegetarian"
	TagVegan      RestaurantTag = "vegan"
	TagGlutenFree RestaurantTag = "glutenfree"
	TagHalal      RestaurantTag = "halal"
)

const (
	StatusDraft    RestaurantStatus = "draft"
	StatusReview   RestaurantStatus = "review"
	StatusApproved RestaurantStatus = "approved"
	StatusActive   RestaurantStatus = "active"
	StatusInactive RestaurantStatus = "inactive"
	StatusDisabled RestaurantStatus = "disabled"
	StatusRejected RestaurantStatus = "rejected"
)

type Address struct {
	House      string `json:"house"`
	Street     string `json:"street"`
	PostalCode string `json:"postalCode"`
	City       string `json:"city"`
}

type Restaurant struct {
	ID              uuid.UUID        `gorm:"type:uuid;primaryKey"`
	OwnerID         uuid.UUID        `gorm:"type:uuid;not null;index"`
	Name            string           `gorm:"size:128;not null"`
	VATNumber       string           `gorm:"column:vat_number;size:16;not null"`
	Slug            *string          `gorm:"size:255"`
	Email           *string          `gorm:"size:255"`
	Phone           *string          `gorm:"size:32"`
	Website         *string          `gorm:"size:255"`
	Checklist       Checklist        `gorm:"type:jsonb;serializer:json;not null;default:'{}'"`
	Status          RestaurantStatus `gorm:"type:restaurant_status_enum;not null;default:'draft'"`
	Address         Address          `gorm:"type:jsonb;serializer:json;not null;default:'{}'"`
	Lat             *float64         `gorm:"type:double precision;check:lat BETWEEN -90 AND 90"`
	Lon             *float64         `gorm:"type:double precision;check:lon BETWEEN -180 AND 180"`
	Timezone        *string          `gorm:"size:64"`
	OpeningHours    OpeningHours     `gorm:"type:jsonb;serializer:json;not null;default:'{}'"`
	Tags            []RestaurantTag  `gorm:"type:jsonb;serializer:json;not null;default:'[]'"`
	Pickup          bool             `gorm:"not null;default:true"`
	Currency        string           `gorm:"type:char(3);not null;default:'EUR';size:3"`
	DeliveryKm      *int16           `gorm:"check:delivery_km BETWEEN 1 AND 25"`
	DeliveryTimeMin *int16           `gorm:"check:delivery_time_min BETWEEN 5 AND 120"`
	DeliveryTimeMax *int16           `gorm:"check:delivery_time_max BETWEEN 5 AND 120"`
	DeliveryType    DeliveryType     `gorm:"type:restaurant_delivery_type_enum;not null;default:'none'"`
	DeliveryFee     decimal.Decimal  `gorm:"type:numeric(5,2);not null;default:0"`
	MinimumOrder    decimal.Decimal  `gorm:"type:numeric(6,2);not null;default:0"`
	Rating          float64          `gorm:"type:numeric(2,1);not null;default:0;check:rating BETWEEN 0 AND 5"`
	TotalReviews    int32            `gorm:"not null;default:0;check:total_reviews >= 0"`
	CreatedAt       time.Time        `gorm:"type:timestamptz;autoCreateTime"`
	UpdatedAt       *time.Time       `gorm:"type:timestamptz;autoUpdateTime;default:null"`
	events          []DomainEvent    `gorm:"-"`
}

func (Restaurant) TableName() string {
	return "restaurants"
}

func NewRestaurant(
	id uuid.UUID,
	ownerID uuid.UUID,
	name string,
	vatNumber string,
	checklist Checklist,
) *Restaurant {
	return &Restaurant{
		ID:        id,
		OwnerID:   ownerID,
		Name:      name,
		VATNumber: vatNumber,
		Checklist: checklist,
	}
}

func (r *Restaurant) WithSlug(slug string) *Restaurant {
	r.Slug = &slug
	return r
}

func (r *Restaurant) WithContact(email, phone, website *string) *Restaurant {
	r.Email = email
	r.Phone = phone
	r.Website = website
	return r
}

func (r *Restaurant) WithAddress(address Address) *Restaurant {
	r.Address = address
	return r
}

func (r *Restaurant) WithCoordinates(lat, lon float64, timezone string) *Restaurant {
	r.Lat = &lat
	r.Lon = &lon
	r.Timezone = &timezone
	return r
}

func (r *Restaurant) WithDelivery(
	pickup bool,
	deliveryType DeliveryType,
	deliveryKm *int16,
	deliveryTimeMin *int16,
	deliveryTimeMax *int16,
	deliveryFee decimal.Decimal,
	minimumOrder decimal.Decimal,
) *Restaurant {
	r.Pickup = pickup
	r.DeliveryType = deliveryType
	r.DeliveryKm = deliveryKm
	r.DeliveryTimeMin = deliveryTimeMin
	r.DeliveryTimeMax = deliveryTimeMax
	r.DeliveryFee = deliveryFee
	r.MinimumOrder = minimumOrder
	return r
}

func (r *Restaurant) WithOpeningHours(openingHours OpeningHours) *Restaurant {
	r.OpeningHours = openingHours
	return r
}

func (r *Restaurant) WithTags(tags []RestaurantTag) *Restaurant {
	r.Tags = tags
	return r
}

func (r *Restaurant) CompleteChecklistItem(item ChecklistItem) {
	r.Checklist.Complete(item)

	if r.Status == StatusDraft && r.Checklist.IsCompleted() {
		r.Status = StatusReview

		r.events = append(r.events, RestaurantReadyForReview{
			RestaurantID:   r.ID,
			RestaurantName: r.Name,
			OccurredAt:     time.Now().UTC(),
		})
	}
}

func (r *Restaurant) Approve() error {
	if r.Status != StatusReview {
		return ErrNotPendingReview
	}

	r.Status = StatusApproved

	r.events = append(r.events, RestaurantApproved{
		RestaurantID:   r.ID,
		RestaurantName: r.Name,
		Email:          *r.Email,
		OccurredAt:     time.Now().UTC(),
	})

	return nil
}

func (r *Restaurant) Launch() error {
	if r.Status != StatusApproved {
		return ErrNotReadyToLaunch
	}

	r.Status = StatusActive

	r.events = append(r.events, RestaurantLaunched{
		RestaurantID: r.ID,
		OccurredAt:   time.Now().UTC(),
	})

	return nil
}

func (r *Restaurant) NotifyUpdated() {
	if r.Status != StatusActive && r.Status != StatusInactive {
		return
	}

	r.events = append(r.events, RestaurantUpdated{
		RestaurantID: r.ID,
		OccurredAt:   time.Now().UTC(),
	})
}

func (r *Restaurant) NotifyPizzaUpdated() {
	if r.Status != StatusActive && r.Status != StatusInactive {
		return
	}

	r.events = append(r.events, PizzaUpdated{
		RestaurantID: r.ID,
		OccurredAt:   time.Now().UTC(),
	})
}

func (r *Restaurant) NotifyToppingPricesUpdated() {
	if r.Status != StatusActive && r.Status != StatusInactive {
		return
	}

	r.events = append(r.events, ToppingPricesUpdated{
		RestaurantID: r.ID,
		OccurredAt:   time.Now().UTC(),
	})
}

func (r *Restaurant) PullEvents() []DomainEvent {
	events := r.events
	r.events = nil
	return events
}
