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

type UpdateContact struct {
	restaurantRepo restaurant.RestaurantRepository
	publisher      event.EventPublisher
}

func NewUpdateContact(
	restaurantRepo restaurant.RestaurantRepository,
	publisher event.EventPublisher,
) *UpdateContact {
	return &UpdateContact{
		restaurantRepo: restaurantRepo,
		publisher:      publisher,
	}
}

func (uc *UpdateContact) Execute(
	ctx context.Context,
	restaurantID uuid.UUID,
	ownerID uuid.UUID,
	input resapp.UpdateContactRequest,
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

	res.CompleteChecklistItem(restaurant.ChecklistContact)

	res.WithContact(input.Email, input.Phone, input.Website)

	if err := uc.restaurantRepo.Update(ctx, res); err != nil {
		return resapp.RestaurantResponse{}, fmt.Errorf("failed to update restaurant: %w", err)
	}

	resapp.DispatchEvents(ctx, uc.publisher, res)

	return resapp.ToRestaurantResponse(res), nil
}
