package testutil

import (
	"context"

	"search-service/internal/domain/index"
)

type MockSearchRepository struct {
	Upserted     []index.IndexedRestaurant
	UpsertErr    error
	SearchResult []index.IndexedRestaurant
	SearchErr    error
	LastQuery    index.SearchQuery
}

var _ index.SearchRepository = (*MockSearchRepository)(nil)

func (m *MockSearchRepository) UpsertSnapshot(_ context.Context, r index.IndexedRestaurant) error {
	m.Upserted = append(m.Upserted, r)
	return m.UpsertErr
}

func (m *MockSearchRepository) Search(_ context.Context, q index.SearchQuery) ([]index.IndexedRestaurant, error) {
	m.LastQuery = q
	return m.SearchResult, m.SearchErr
}
