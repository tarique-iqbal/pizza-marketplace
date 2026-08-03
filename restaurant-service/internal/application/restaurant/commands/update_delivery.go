package commands

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	resapp "restaurant-service/internal/application/restaurant"
	"restaurant-service/internal/domain/restaurant"
	apperr "restaurant-service/internal/shared/errors"
)

type UpdateDelivery struct {
	restaurantRepo restaurant.RestaurantRepository
}

func NewUpdateDelivery(
	restaurantRepo restaurant.RestaurantRepository,
) *UpdateDelivery {
	return &UpdateDelivery{
		restaurantRepo: restaurantRepo,
	}
}

func (uc *UpdateDelivery) Execute(
	ctx context.Context,
	restaurantID uuid.UUID,
	ownerID uuid.UUID,
	input resapp.UpdateDeliveryRequest,
) (resapp.RestaurantResponse, error) {
	res, err := uc.restaurantRepo.FindByIDAndOwner(ctx, restaurantID, ownerID)
	if err != nil {
		return resapp.RestaurantResponse{}, fmt.Errorf("failed to verify ownership: %w", err)
	}
	if res == nil {
		return resapp.RestaurantResponse{}, fmt.Errorf(
			"access denied: restaurant not owned by user: %w",
			apperr.ErrForbidden,
		)
	}

	res.Checklist.Complete(restaurant.ChecklistDelivery)

	res.WithDelivery(
		input.Pickup,
		input.DeliveryType,
		input.DeliveryKm,
		input.DeliveryFee,
		input.MinimumOrder,
	).WithUpdated()

	if err := uc.restaurantRepo.Update(ctx, res); err != nil {
		return resapp.RestaurantResponse{}, fmt.Errorf("failed to update restaurant: %w", err)
	}

	return resapp.ToRestaurantResponse(res), nil
}
