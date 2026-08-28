package commands

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	payoutapp "restaurant-service/internal/application/payout"
	resapp "restaurant-service/internal/application/restaurant"
	"restaurant-service/internal/domain/outbox"
	"restaurant-service/internal/domain/payout"
	"restaurant-service/internal/domain/restaurant"
	apperr "restaurant-service/internal/shared/errors"
)

type CreatePayout struct {
	db                *gorm.DB
	restaurantRepo    restaurant.RestaurantRepository
	payoutDetailsRepo payout.PayoutDetailsRepository
	outboxRepo        outbox.OutboxRepository
}

func NewCreatePayout(
	db *gorm.DB,
	restaurantRepo restaurant.RestaurantRepository,
	payoutDetailsRepo payout.PayoutDetailsRepository,
	outboxRepo outbox.OutboxRepository,
) *CreatePayout {
	return &CreatePayout{
		db:                db,
		restaurantRepo:    restaurantRepo,
		payoutDetailsRepo: payoutDetailsRepo,
		outboxRepo:        outboxRepo,
	}
}

func (uc *CreatePayout) Execute(
	ctx context.Context,
	restaurantID uuid.UUID,
	ownerID uuid.UUID,
	input payoutapp.CreatePayoutRequest,
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

	pd, err := payout.NewPayoutDetails(
		res.ID,
		input.AccountHolder,
		input.IBAN,
		input.BIC,
		input.BankName,
	)
	if err != nil {
		return resapp.RestaurantResponse{}, fmt.Errorf("failed to generate payout details id: %w", err)
	}

	res.CompleteChecklistItem(restaurant.ChecklistPayout)

	err = uc.db.Transaction(func(tx *gorm.DB) error {
		if err := uc.payoutDetailsRepo.WithTx(tx).Create(ctx, pd); err != nil {
			return fmt.Errorf("failed to create payout details: %w", err)
		}

		if err := uc.restaurantRepo.WithTx(tx).Update(ctx, res); err != nil {
			return fmt.Errorf("failed to update restaurant: %w", err)
		}

		return resapp.DispatchEventsTx(ctx, uc.outboxRepo.WithTx(tx), res)
	})
	if err != nil {
		return resapp.RestaurantResponse{}, err
	}

	return resapp.ToRestaurantResponse(res, pd), nil
}
