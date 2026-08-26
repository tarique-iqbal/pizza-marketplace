package testutil

import (
	"context"
	"time"

	"github.com/google/uuid"

	"search-service/internal/domain/index"
)

type UpdatedFields struct {
	ID     uuid.UUID
	Fields index.RestaurantFields
}

type UpsertedPizza struct {
	RestaurantID uuid.UUID
	Pizza        index.IndexedPizza
}

type RemovedPizza struct {
	RestaurantID uuid.UUID
	PizzaID      uuid.UUID
	UpdatedAt    time.Time
}

type UpdatedToppingPrices struct {
	RestaurantID uuid.UUID
	Prices       []index.IndexedToppingPrice
	UpdatedAt    time.Time
}

type MockSearchRepository struct {
	Upserted               []index.IndexedRestaurant
	UpsertErr              error
	UpdatedFields          []UpdatedFields
	UpdateErr              error
	UpsertedPizzas         []UpsertedPizza
	UpsertPizzaErr         error
	RemovedPizzas          []RemovedPizza
	RemovePizzaErr         error
	UpdatedToppingPrices   []UpdatedToppingPrices
	UpdateToppingPricesErr error
	SearchResult           []index.IndexedRestaurant
	SearchErr              error
	LastQuery              index.SearchQuery
}

var _ index.SearchRepository = (*MockSearchRepository)(nil)

func (m *MockSearchRepository) UpsertSnapshot(_ context.Context, r index.IndexedRestaurant) error {
	m.Upserted = append(m.Upserted, r)
	return m.UpsertErr
}

func (m *MockSearchRepository) UpdateFields(_ context.Context, id uuid.UUID, fields index.RestaurantFields) error {
	m.UpdatedFields = append(m.UpdatedFields, UpdatedFields{ID: id, Fields: fields})
	return m.UpdateErr
}

func (m *MockSearchRepository) UpsertPizza(_ context.Context, restaurantID uuid.UUID, pizza index.IndexedPizza) error {
	m.UpsertedPizzas = append(m.UpsertedPizzas, UpsertedPizza{RestaurantID: restaurantID, Pizza: pizza})
	return m.UpsertPizzaErr
}

func (m *MockSearchRepository) RemovePizza(
	_ context.Context,
	restaurantID, pizzaID uuid.UUID,
	updatedAt time.Time,
) error {
	m.RemovedPizzas = append(m.RemovedPizzas, RemovedPizza{
		RestaurantID: restaurantID,
		PizzaID:      pizzaID,
		UpdatedAt:    updatedAt,
	})
	return m.RemovePizzaErr
}

func (m *MockSearchRepository) UpdateToppingPrices(
	_ context.Context,
	restaurantID uuid.UUID,
	prices []index.IndexedToppingPrice,
	updatedAt time.Time,
) error {
	m.UpdatedToppingPrices = append(m.UpdatedToppingPrices, UpdatedToppingPrices{
		RestaurantID: restaurantID,
		Prices:       prices,
		UpdatedAt:    updatedAt,
	})
	return m.UpdateToppingPricesErr
}

func (m *MockSearchRepository) Search(_ context.Context, q index.SearchQuery) ([]index.IndexedRestaurant, error) {
	m.LastQuery = q
	return m.SearchResult, m.SearchErr
}
