package testutil

import (
	"context"

	"github.com/google/uuid"

	"order-service/internal/domain/readmodel"
)

type MockRestaurantRepository struct {
	FindByIDResult *readmodel.Restaurant
	FindByIDErr    error
	Upserted       []readmodel.Restaurant
	UpsertErr      error
}

var _ readmodel.RestaurantRepository = (*MockRestaurantRepository)(nil)

func (m *MockRestaurantRepository) FindByID(_ context.Context, _ uuid.UUID) (*readmodel.Restaurant, error) {
	return m.FindByIDResult, m.FindByIDErr
}

func (m *MockRestaurantRepository) Upsert(_ context.Context, restaurant readmodel.Restaurant) error {
	m.Upserted = append(m.Upserted, restaurant)
	return m.UpsertErr
}
