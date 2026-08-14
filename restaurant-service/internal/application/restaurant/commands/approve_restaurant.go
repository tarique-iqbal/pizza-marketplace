package commands

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	resapp "restaurant-service/internal/application/restaurant"
	"restaurant-service/internal/domain/payout"
	"restaurant-service/internal/domain/restaurant"
	apperr "restaurant-service/internal/shared/errors"
	"restaurant-service/internal/shared/event"
)

type ApproveRestaurant struct {
	restaurantRepo    restaurant.RestaurantRepository
	payoutDetailsRepo payout.PayoutDetailsRepository
	publisher         event.EventPublisher
}

func NewApproveRestaurant(
	restaurantRepo restaurant.RestaurantRepository,
	payoutDetailsRepo payout.PayoutDetailsRepository,
	publisher event.EventPublisher,
) *ApproveRestaurant {
	return &ApproveRestaurant{
		restaurantRepo:    restaurantRepo,
		payoutDetailsRepo: payoutDetailsRepo,
		publisher:         publisher,
	}
}

func (uc *ApproveRestaurant) Execute(
	ctx context.Context,
	restaurantID uuid.UUID,
) (resapp.RestaurantResponse, error) {
	res, err := uc.restaurantRepo.FindByID(ctx, restaurantID)
	if err != nil {
		return resapp.RestaurantResponse{}, fmt.Errorf("failed to find restaurant: %w", err)
	}
	if res == nil {
		return resapp.RestaurantResponse{}, apperr.ErrNotFound
	}

	if err := res.Approve(); err != nil {
		return resapp.RestaurantResponse{}, fmt.Errorf("%w: %w", err, apperr.ErrConflict)
	}

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
