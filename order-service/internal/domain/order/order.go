package order

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type OrderStatus string

const (
	StatusPending   OrderStatus = "pending"
	StatusConfirmed OrderStatus = "confirmed"
	StatusPreparing OrderStatus = "preparing"
	StatusReady     OrderStatus = "ready"
	StatusCompleted OrderStatus = "completed"
	StatusCancelled OrderStatus = "cancelled"
)

type Fulfillment string

const (
	FulfillmentDelivery Fulfillment = "delivery"
	FulfillmentPickup   Fulfillment = "pickup"
)

type Address struct {
	House      string `json:"house"`
	Street     string `json:"street"`
	PostalCode string `json:"postalCode"`
	City       string `json:"city"`
}

type Order struct {
	ID              uuid.UUID       `gorm:"type:uuid;primaryKey"`
	CustomerID      uuid.UUID       `gorm:"type:uuid;not null;index"`
	RestaurantID    uuid.UUID       `gorm:"type:uuid;not null;index"`
	Status          OrderStatus     `gorm:"type:order_status_enum;not null;default:'pending'"`
	Fulfillment     Fulfillment     `gorm:"type:order_fulfillment_enum;not null"`
	ContactEmail    string          `gorm:"size:255;not null"`
	ContactPhone    *string         `gorm:"size:32"`
	DeliveryAddress *Address        `gorm:"type:jsonb;serializer:json"`
	DeliveryLat     *float64        `gorm:"type:double precision;check:delivery_lat BETWEEN -90 AND 90"`
	DeliveryLon     *float64        `gorm:"type:double precision;check:delivery_lon BETWEEN -180 AND 180"`
	Items           []OrderItem     `gorm:"foreignKey:OrderID"`
	Subtotal        decimal.Decimal `gorm:"type:numeric(8,2);not null"`
	DeliveryFee     decimal.Decimal `gorm:"type:numeric(5,2);not null;default:0"`
	Total           decimal.Decimal `gorm:"type:numeric(8,2);not null"`
	Currency        string          `gorm:"type:char(3);not null;default:'EUR';size:3"`
	PaymentID       *string         `gorm:"size:64"`
	PlacedAt        time.Time       `gorm:"type:timestamptz;not null;autoCreateTime"`
	ConfirmedAt     *time.Time      `gorm:"type:timestamptz"`
	PrepStartedAt   *time.Time      `gorm:"type:timestamptz"`
	ReadyAt         *time.Time      `gorm:"type:timestamptz"`
	CompletedAt     *time.Time      `gorm:"type:timestamptz"`
	CancelledAt     *time.Time      `gorm:"type:timestamptz"`
	events          []DomainEvent   `gorm:"-"`
}

func (Order) TableName() string {
	return "orders"
}

func NewOrder(
	id, customerID, restaurantID uuid.UUID,
	fulfillment Fulfillment,
	contactEmail string,
	contactPhone *string,
	deliveryAddress *Address,
	deliveryLat, deliveryLon *float64,
	items []OrderItem,
	subtotal, deliveryFee, total decimal.Decimal,
	currency string,
) *Order {
	return &Order{
		ID:              id,
		CustomerID:      customerID,
		RestaurantID:    restaurantID,
		Status:          StatusPending,
		Fulfillment:     fulfillment,
		ContactEmail:    contactEmail,
		ContactPhone:    contactPhone,
		DeliveryAddress: deliveryAddress,
		DeliveryLat:     deliveryLat,
		DeliveryLon:     deliveryLon,
		Items:           items,
		Subtotal:        subtotal,
		DeliveryFee:     deliveryFee,
		Total:           total,
		Currency:        currency,
	}
}

func (o *Order) Confirm() error {
	if o.Status != StatusPending {
		return ErrInvalidStatusTransition
	}

	now := time.Now().UTC()
	o.Status = StatusConfirmed
	o.ConfirmedAt = &now

	o.events = append(o.events, OrderConfirmed{
		OrderID:      o.ID,
		CustomerID:   o.CustomerID,
		RestaurantID: o.RestaurantID,
		OccurredAt:   now,
	})

	return nil
}

func (o *Order) StartPreparing() error {
	if o.Status != StatusConfirmed {
		return ErrInvalidStatusTransition
	}

	now := time.Now().UTC()
	o.Status = StatusPreparing
	o.PrepStartedAt = &now

	return nil
}

func (o *Order) MarkReady() error {
	if o.Status != StatusPreparing {
		return ErrInvalidStatusTransition
	}

	now := time.Now().UTC()
	o.Status = StatusReady
	o.ReadyAt = &now

	return nil
}

func (o *Order) Complete() error {
	if o.Status != StatusReady {
		return ErrInvalidStatusTransition
	}

	now := time.Now().UTC()
	o.Status = StatusCompleted
	o.CompletedAt = &now

	return nil
}

func (o *Order) Cancel() error {
	switch o.Status {
	case StatusPending, StatusConfirmed, StatusPreparing:
	default:
		return ErrInvalidStatusTransition
	}

	now := time.Now().UTC()
	o.Status = StatusCancelled
	o.CancelledAt = &now

	return nil
}

func (o *Order) PullEvents() []DomainEvent {
	events := o.events
	o.events = nil
	return events
}
