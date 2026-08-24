package testutil

import (
	"context"

	"github.com/google/uuid"

	"search-service/internal/domain/index"
)

type UpdatedFields struct {
	ID     uuid.UUID
	Fields index.RestaurantFields
}

type MockSearchRepository struct {
	Upserted      []index.IndexedRestaurant
	UpsertErr     error
	UpdatedFields []UpdatedFields
	UpdateErr     error
	SearchResult  []index.IndexedRestaurant
	SearchErr     error
	LastQuery     index.SearchQuery
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

func (m *MockSearchRepository) Search(_ context.Context, q index.SearchQuery) ([]index.IndexedRestaurant, error) {
	m.LastQuery = q
	return m.SearchResult, m.SearchErr
}
