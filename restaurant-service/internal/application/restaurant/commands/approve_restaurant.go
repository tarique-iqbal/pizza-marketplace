package commands

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	resapp "restaurant-service/internal/application/restaurant"
	"restaurant-service/internal/domain/outbox"
	"restaurant-service/internal/domain/payout"
	"restaurant-service/internal/domain/restaurant"
	apperr "restaurant-service/internal/shared/errors"
)

type ApproveRestaurant struct {
	db                *gorm.DB
	restaurantRepo    restaurant.RestaurantRepository
	payoutDetailsRepo payout.PayoutDetailsRepository
	outboxRepo        outbox.OutboxRepository
}

func NewApproveRestaurant(
	db *gorm.DB,
	restaurantRepo restaurant.RestaurantRepository,
	payoutDetailsRepo payout.PayoutDetailsRepository,
	outboxRepo outbox.OutboxRepository,
) *ApproveRestaurant {
	return &ApproveRestaurant{
		db:                db,
		restaurantRepo:    restaurantRepo,
		payoutDetailsRepo: payoutDetailsRepo,
		outboxRepo:        outboxRepo,
	}
}

func (cmd *ApproveRestaurant) Execute(
	ctx context.Context,
	restaurantID uuid.UUID,
) (resapp.RestaurantResponse, error) {
	res, err := cmd.restaurantRepo.FindByID(ctx, restaurantID)
	if err != nil {
		return resapp.RestaurantResponse{}, fmt.Errorf("failed to find restaurant: %w", err)
	}
	if res == nil {
		return resapp.RestaurantResponse{}, apperr.ErrNotFound
	}

	if err := res.Approve(); err != nil {
		return resapp.RestaurantResponse{}, fmt.Errorf("%w: %w", err, apperr.ErrConflict)
	}

	err = cmd.db.Transaction(func(tx *gorm.DB) error {
		if err := cmd.restaurantRepo.WithTx(tx).Update(ctx, res); err != nil {
			return fmt.Errorf("failed to update restaurant: %w", err)
		}

		if err := cmd.payoutDetailsRepo.WithTx(tx).PromoteToActive(ctx, res.ID); err != nil {
			return fmt.Errorf("failed to promote payout details: %w", err)
		}

		return resapp.DispatchEventsTx(ctx, cmd.outboxRepo.WithTx(tx), res)
	})
	if err != nil {
		return resapp.RestaurantResponse{}, err
	}

	pd, err := cmd.payoutDetailsRepo.FindActiveByRestaurant(ctx, res.ID)
	if err != nil {
		return resapp.RestaurantResponse{}, fmt.Errorf("failed to fetch payout details: %w", err)
	}

	return resapp.ToRestaurantResponse(res, pd), nil
}
