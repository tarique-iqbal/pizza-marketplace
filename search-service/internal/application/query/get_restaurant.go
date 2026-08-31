package query

import (
	"context"

	"github.com/google/uuid"

	"search-service/internal/domain/index"
)

type GetRestaurant struct {
	repo index.SearchRepository
}

func NewGetRestaurant(repo index.SearchRepository) *GetRestaurant {
	return &GetRestaurant{repo: repo}
}

func (uc *GetRestaurant) Execute(ctx context.Context, id uuid.UUID) (index.IndexedRestaurant, error) {
	return uc.repo.FindByID(ctx, id)
}
