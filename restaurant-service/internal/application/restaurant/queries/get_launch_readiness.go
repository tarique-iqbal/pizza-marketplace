package queries

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	pizzaqry "restaurant-service/internal/application/pizza/queries"
	resapp "restaurant-service/internal/application/restaurant"
	"restaurant-service/internal/domain/restaurant"
	apperr "restaurant-service/internal/shared/errors"
)

type GetLaunchReadiness struct {
	restaurantRepo restaurant.RestaurantRepository
	pizzaCatalog   *pizzaqry.PizzaCatalog
}

func NewGetLaunchReadiness(
	restaurantRepo restaurant.RestaurantRepository,
	pizzaCatalog *pizzaqry.PizzaCatalog,
) *GetLaunchReadiness {
	return &GetLaunchReadiness{
		restaurantRepo: restaurantRepo,
		pizzaCatalog:   pizzaCatalog,
	}
}

func (uc *GetLaunchReadiness) Execute(
	ctx context.Context,
	restaurantID uuid.UUID,
	ownerID uuid.UUID,
) (resapp.LaunchReadinessResponse, error) {
	res, err := uc.restaurantRepo.FindByIDAndOwner(ctx, restaurantID, ownerID)
	if err != nil {
		return resapp.LaunchReadinessResponse{}, fmt.Errorf("failed to verify ownership: %w", err)
	}
	if res == nil {
		return resapp.LaunchReadinessResponse{}, fmt.Errorf(
			"access denied: restaurant not owned by user: %w",
			apperr.ErrForbidden,
		)
	}

	switch res.Status {
	case restaurant.StatusDraft, restaurant.StatusReview, restaurant.StatusApproved:
	default:
		return resapp.LaunchReadinessResponse{}, fmt.Errorf(
			"%w: %w",
			restaurant.ErrLaunchReadinessNotApplicable,
			apperr.ErrConflict,
		)
	}

	pizzas, err := uc.pizzaCatalog.Execute(ctx, res.ID)
	if err != nil {
		return resapp.LaunchReadinessResponse{}, fmt.Errorf("failed to load pizza catalog: %w", err)
	}

	readiness := resapp.EvaluateLaunchReadiness(pizzas)

	return resapp.ToLaunchReadinessResponse(res, readiness), nil
}
