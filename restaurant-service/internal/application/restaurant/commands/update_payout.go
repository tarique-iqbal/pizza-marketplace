package commands

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	resapp "restaurant-service/internal/application/restaurant"
	"restaurant-service/internal/domain/restaurant"
	apperr "restaurant-service/internal/shared/errors"
)

type UpdatePayout struct {
	restaurantRepo    restaurant.RestaurantRepository
	payoutDetailsRepo restaurant.PayoutDetailsRepository
}

func NewUpdatePayout(
	restaurantRepo restaurant.RestaurantRepository,
	payoutDetailsRepo restaurant.PayoutDetailsRepository,
) *UpdatePayout {
	return &UpdatePayout{
		restaurantRepo:    restaurantRepo,
		payoutDetailsRepo: payoutDetailsRepo,
	}
}

func (uc *UpdatePayout) Execute(
	ctx context.Context,
	restaurantID uuid.UUID,
	ownerID uuid.UUID,
	input resapp.UpdatePayoutRequest,
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

	if err := uc.payoutDetailsRepo.UpdatePending(
		ctx,
		res.ID,
		input.AccountHolder,
		input.IBAN,
		input.BIC,
		input.BankName,
	); err != nil {
		return resapp.RestaurantResponse{}, fmt.Errorf("failed to update payout details: %w", err)
	}

	res.PayoutDetails = &restaurant.PayoutDetails{
		AccountHolder: input.AccountHolder,
		IBAN:          input.IBAN,
		BIC:           input.BIC,
		BankName:      input.BankName,
		Status:        restaurant.PayoutPending,
	}

	return resapp.ToRestaurantResponse(res), nil
}
