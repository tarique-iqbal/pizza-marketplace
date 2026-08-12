package commands

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	resapp "restaurant-service/internal/application/restaurant"
	"restaurant-service/internal/domain/restaurant"
	apperr "restaurant-service/internal/shared/errors"
	"restaurant-service/internal/shared/event"
)

type UpdateDelivery struct {
	restaurantRepo    restaurant.RestaurantRepository
	payoutDetailsRepo restaurant.PayoutDetailsRepository
	publisher         event.EventPublisher
}

func NewUpdateDelivery(
	restaurantRepo restaurant.RestaurantRepository,
	payoutDetailsRepo restaurant.PayoutDetailsRepository,
	publisher event.EventPublisher,
) *UpdateDelivery {
	return &UpdateDelivery{
		restaurantRepo:    restaurantRepo,
		payoutDetailsRepo: payoutDetailsRepo,
		publisher:         publisher,
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

	res.CompleteChecklistItem(restaurant.ChecklistDelivery)

	res.WithDelivery(
		input.Pickup,
		input.DeliveryType,
		input.DeliveryKm,
		input.DeliveryFee,
		input.MinimumOrder,
	)

	if err := uc.restaurantRepo.Update(ctx, res); err != nil {
		return resapp.RestaurantResponse{}, fmt.Errorf("failed to update restaurant: %w", err)
	}

	resapp.DispatchEvents(ctx, uc.publisher, res)

	pd, err := uc.payoutDetailsRepo.FindActiveByRestaurant(ctx, res.ID)
	if err != nil {
		return resapp.RestaurantResponse{}, fmt.Errorf("failed to fetch payout details: %w", err)
	}

	return resapp.ToRestaurantResponse(res, pd), nil
}
