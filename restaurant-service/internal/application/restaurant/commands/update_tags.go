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

type UpdateTags struct {
	restaurantRepo    restaurant.RestaurantRepository
	payoutDetailsRepo payout.PayoutDetailsRepository
	publisher         event.EventPublisher
}

func NewUpdateTags(
	restaurantRepo restaurant.RestaurantRepository,
	payoutDetailsRepo payout.PayoutDetailsRepository,
	publisher event.EventPublisher,
) *UpdateTags {
	return &UpdateTags{
		restaurantRepo:    restaurantRepo,
		payoutDetailsRepo: payoutDetailsRepo,
		publisher:         publisher,
	}
}

func (uc *UpdateTags) Execute(
	ctx context.Context,
	restaurantID uuid.UUID,
	ownerID uuid.UUID,
	input resapp.UpdateTagsRequest,
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

	res.NotifyUpdated()
	res.WithTags(input.Tags)

	if err := uc.restaurantRepo.Update(ctx, res); err != nil {
		return resapp.RestaurantResponse{}, fmt.Errorf("failed to update restaurant: %w", err)
	}

	resapp.DispatchEvents(ctx, uc.publisher, res, uc.enrichUpdated(res))

	pd, err := uc.payoutDetailsRepo.FindActiveByRestaurant(ctx, res.ID)
	if err != nil {
		return resapp.RestaurantResponse{}, fmt.Errorf("failed to fetch payout details: %w", err)
	}

	return resapp.ToRestaurantResponse(res, pd), nil
}

func (uc *UpdateTags) enrichUpdated(res *restaurant.Restaurant) resapp.Enricher {
	return func(e restaurant.DomainEvent) (event.Event, bool) {
		updated, ok := e.(restaurant.RestaurantUpdated)
		if !ok {
			return nil, false
		}

		return resapp.NewRestaurantUpdatedPayload(updated, res), true
	}
}
