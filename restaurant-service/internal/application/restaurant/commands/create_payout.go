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

type CreatePayout struct {
	restaurantRepo    restaurant.RestaurantRepository
	payoutDetailsRepo restaurant.PayoutDetailsRepository
	publisher         event.EventPublisher
}

func NewCreatePayout(
	restaurantRepo restaurant.RestaurantRepository,
	payoutDetailsRepo restaurant.PayoutDetailsRepository,
	publisher event.EventPublisher,
) *CreatePayout {
	return &CreatePayout{
		restaurantRepo:    restaurantRepo,
		payoutDetailsRepo: payoutDetailsRepo,
		publisher:         publisher,
	}
}

func (uc *CreatePayout) Execute(
	ctx context.Context,
	restaurantID uuid.UUID,
	ownerID uuid.UUID,
	input resapp.CreatePayoutRequest,
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

	pd, err := restaurant.NewPayoutDetails(
		res.ID,
		input.AccountHolder,
		input.IBAN,
		input.BIC,
		input.BankName,
	)
	if err != nil {
		return resapp.RestaurantResponse{}, fmt.Errorf("failed to generate payout details id: %w", err)
	}

	if err := uc.payoutDetailsRepo.Create(ctx, pd); err != nil {
		return resapp.RestaurantResponse{}, fmt.Errorf("failed to create payout details: %w", err)
	}

	res.CompleteChecklistItem(restaurant.ChecklistPayment)
	res.PayoutDetails = pd

	if err := uc.restaurantRepo.Update(ctx, res); err != nil {
		return resapp.RestaurantResponse{}, fmt.Errorf("failed to update restaurant: %w", err)
	}

	resapp.DispatchEvents(ctx, uc.publisher, res)

	return resapp.ToRestaurantResponse(res), nil
}
