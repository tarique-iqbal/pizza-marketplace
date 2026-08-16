package queries

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	pizzaqueries "restaurant-service/internal/application/pizza/queries"
	resapp "restaurant-service/internal/application/restaurant"
	"restaurant-service/internal/domain/payout"
	"restaurant-service/internal/domain/restaurant"
	apperr "restaurant-service/internal/shared/errors"
)

type GetRestaurant struct {
	restaurantRepo    restaurant.RestaurantRepository
	payoutDetailsRepo payout.PayoutDetailsRepository
	pizzaCatalog      *pizzaqueries.PizzaCatalog
}

func NewGetRestaurant(
	restaurantRepo restaurant.RestaurantRepository,
	payoutDetailsRepo payout.PayoutDetailsRepository,
	pizzaCatalog *pizzaqueries.PizzaCatalog,
) *GetRestaurant {
	return &GetRestaurant{
		restaurantRepo:    restaurantRepo,
		payoutDetailsRepo: payoutDetailsRepo,
		pizzaCatalog:      pizzaCatalog,
	}
}

func (uc *GetRestaurant) Execute(
	ctx context.Context,
	restaurantID uuid.UUID,
	ownerID uuid.UUID,
) (resapp.RestaurantWithPizzasResponse, error) {
	res, err := uc.restaurantRepo.FindByIDAndOwner(ctx, restaurantID, ownerID)
	if err != nil {
		return resapp.RestaurantWithPizzasResponse{}, fmt.Errorf("failed to verify ownership: %w", err)
	}
	if res == nil {
		return resapp.RestaurantWithPizzasResponse{}, fmt.Errorf(
			"access denied: restaurant not owned by user: %w",
			apperr.ErrForbidden,
		)
	}

	pd, err := uc.payoutDetailsRepo.FindActiveByRestaurant(ctx, res.ID)
	if err != nil {
		return resapp.RestaurantWithPizzasResponse{}, fmt.Errorf("failed to fetch payout details: %w", err)
	}

	pizzas, err := uc.pizzaCatalog.Execute(ctx, res.ID)
	if err != nil {
		return resapp.RestaurantWithPizzasResponse{}, fmt.Errorf("failed to load pizza catalog: %w", err)
	}

	return resapp.RestaurantWithPizzasResponse{
		RestaurantResponse: resapp.ToRestaurantResponse(res, pd),
		Pizzas:             pizzas,
	}, nil
}
