package queries

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	pizzaapp "restaurant-service/internal/application/pizza"
	"restaurant-service/internal/domain/restaurant"
	apperr "restaurant-service/internal/shared/errors"
)

type ListPizzas struct {
	restaurantRepo restaurant.RestaurantRepository
	pizzaCatalog   *PizzaCatalog
}

func NewListPizzas(
	restaurantRepo restaurant.RestaurantRepository,
	pizzaCatalog *PizzaCatalog,
) *ListPizzas {
	return &ListPizzas{
		restaurantRepo: restaurantRepo,
		pizzaCatalog:   pizzaCatalog,
	}
}

func (uc *ListPizzas) Execute(
	ctx context.Context,
	restaurantID uuid.UUID,
	ownerID uuid.UUID,
) ([]pizzaapp.PizzaResponse, error) {
	res, err := uc.restaurantRepo.FindByIDAndOwner(ctx, restaurantID, ownerID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify ownership: %w", err)
	}
	if res == nil {
		return nil, fmt.Errorf(
			"access denied: restaurant not owned by user: %w",
			apperr.ErrForbidden,
		)
	}

	return uc.pizzaCatalog.Execute(ctx, restaurantID)
}
