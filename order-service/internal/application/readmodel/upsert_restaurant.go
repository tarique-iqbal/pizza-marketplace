package readmodel

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"order-service/internal/domain/readmodel"
)

// restaurantLaunchedPayload mirrors restaurant-service's wire shape — a local, independent copy.
type restaurantLaunchedPayload struct {
	RestaurantID   uuid.UUID `json:"restaurant_id"`
	OwnerID        uuid.UUID `json:"owner_id"`
	RestaurantName string    `json:"restaurant_name"`
	Contact        struct {
		Email *string `json:"email"`
	} `json:"contact"`
	Lat      float64 `json:"lat"`
	Lon      float64 `json:"lon"`
	Delivery struct {
		Type         string `json:"type"`
		RadiusKm     *int16 `json:"radiusKm"`
		Fee          string `json:"fee"`
		MinimumOrder string `json:"minimumOrder"`
	} `json:"delivery"`
	Currency      string                     `json:"currency"`
	Pickup        bool                       `json:"pickup"`
	Pizzas        []launchedPizzaWire        `json:"pizzas"`
	ToppingPrices []launchedToppingPriceWire `json:"topping_prices"`
	UpdatedAt     time.Time                  `json:"updated_at"`
}

type launchedPizzaWire struct {
	ID        uuid.UUID                `json:"id"`
	Name      string                   `json:"name"`
	Status    string                   `json:"status"`
	Prices    []launchedPizzaPriceWire `json:"prices"`
	UpdatedAt *time.Time               `json:"updatedAt"`
}

func (p launchedPizzaWire) hasActivePrice() bool {
	for _, price := range p.Prices {
		if price.IsActive {
			return true
		}
	}

	return false
}

type launchedPizzaPriceWire struct {
	SizeID     uuid.UUID `json:"sizeId"`
	DiameterCm int16     `json:"diameterCm"`
	Price      string    `json:"price"`
	IsActive   bool      `json:"isActive"`
}

type launchedToppingPriceWire struct {
	ToppingID  uuid.UUID `json:"toppingId"`
	Name       string    `json:"name"`
	ExtraPrice string    `json:"extraPrice"`
}

type UpsertRestaurant struct {
	restaurantRepo   readmodel.RestaurantRepository
	pizzaRepo        readmodel.PizzaRepository
	pizzaPriceRepo   readmodel.PizzaPriceRepository
	toppingPriceRepo readmodel.ToppingPriceRepository
}

func NewUpsertRestaurant(
	restaurantRepo readmodel.RestaurantRepository,
	pizzaRepo readmodel.PizzaRepository,
	pizzaPriceRepo readmodel.PizzaPriceRepository,
	toppingPriceRepo readmodel.ToppingPriceRepository,
) *UpsertRestaurant {
	return &UpsertRestaurant{
		restaurantRepo:   restaurantRepo,
		pizzaRepo:        pizzaRepo,
		pizzaPriceRepo:   pizzaPriceRepo,
		toppingPriceRepo: toppingPriceRepo,
	}
}

func (h *UpsertRestaurant) Handle(event readmodel.EventPayload) error {
	var payload restaurantLaunchedPayload
	if err := json.Unmarshal(event.Data, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal %s payload: %w", event.Name, err)
	}

	if payload.Contact.Email == nil {
		return fmt.Errorf("%s payload missing contact email for restaurant %s", event.Name, payload.RestaurantID)
	}

	deliveryFee, err := decimal.NewFromString(payload.Delivery.Fee)
	if err != nil {
		return fmt.Errorf("failed to parse delivery fee: %w", err)
	}

	minimumOrder, err := decimal.NewFromString(payload.Delivery.MinimumOrder)
	if err != nil {
		return fmt.Errorf("failed to parse minimum order: %w", err)
	}

	ctx := context.Background()

	restaurant := readmodel.Restaurant{
		ID:           payload.RestaurantID,
		OwnerID:      payload.OwnerID,
		Name:         payload.RestaurantName,
		OwnerEmail:   *payload.Contact.Email,
		Lat:          payload.Lat,
		Lon:          payload.Lon,
		DeliveryKm:   payload.Delivery.RadiusKm,
		DeliveryFee:  deliveryFee,
		MinimumOrder: minimumOrder,
		Pickup:       payload.Pickup,
		DeliveryType: readmodel.DeliveryType(payload.Delivery.Type),
		Currency:     payload.Currency,
		UpdatedAt:    payload.UpdatedAt,
	}

	if err := h.restaurantRepo.Upsert(ctx, restaurant); err != nil {
		return fmt.Errorf("failed to upsert restaurant: %w", err)
	}

	for _, pz := range payload.Pizzas {
		if pz.Status == "archived" || !pz.hasActivePrice() {
			continue
		}

		if pz.UpdatedAt == nil {
			return fmt.Errorf("%s payload missing updatedAt for pizza %s", event.Name, pz.ID)
		}

		pizza := readmodel.Pizza{
			ID:           pz.ID,
			RestaurantID: payload.RestaurantID,
			Name:         pz.Name,
			Status:       readmodel.PizzaStatus(pz.Status),
			UpdatedAt:    *pz.UpdatedAt,
		}

		if err := h.pizzaRepo.Upsert(ctx, pizza); err != nil {
			return fmt.Errorf("failed to upsert pizza: %w", err)
		}

		for _, price := range pz.Prices {
			amount, err := decimal.NewFromString(price.Price)
			if err != nil {
				return fmt.Errorf("failed to parse pizza price: %w", err)
			}

			pizzaPrice := readmodel.PizzaPrice{
				PizzaID:    pz.ID,
				SizeID:     price.SizeID,
				DiameterCm: price.DiameterCm,
				Price:      amount,
				IsActive:   price.IsActive,
				UpdatedAt:  *pz.UpdatedAt,
			}

			if err := h.pizzaPriceRepo.Upsert(ctx, pizzaPrice); err != nil {
				return fmt.Errorf("failed to upsert pizza price: %w", err)
			}
		}
	}

	for _, tp := range payload.ToppingPrices {
		extraPrice, err := decimal.NewFromString(tp.ExtraPrice)
		if err != nil {
			return fmt.Errorf("failed to parse topping extra price: %w", err)
		}

		toppingPrice := readmodel.ToppingPrice{
			RestaurantID: payload.RestaurantID,
			ToppingID:    tp.ToppingID,
			Name:         tp.Name,
			ExtraPrice:   extraPrice,
			UpdatedAt:    payload.UpdatedAt,
		}

		if err := h.toppingPriceRepo.Upsert(ctx, toppingPrice); err != nil {
			return fmt.Errorf("failed to upsert topping price: %w", err)
		}
	}

	return nil
}
