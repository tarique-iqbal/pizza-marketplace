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
	"restaurant-service/internal/shared/event"
)

type UpdateOpeningHours struct {
	db                *gorm.DB
	restaurantRepo    restaurant.RestaurantRepository
	payoutDetailsRepo payout.PayoutDetailsRepository
	outboxRepo        outbox.OutboxRepository
}

func NewUpdateOpeningHours(
	db *gorm.DB,
	restaurantRepo restaurant.RestaurantRepository,
	payoutDetailsRepo payout.PayoutDetailsRepository,
	outboxRepo outbox.OutboxRepository,
) *UpdateOpeningHours {
	return &UpdateOpeningHours{
		db:                db,
		restaurantRepo:    restaurantRepo,
		payoutDetailsRepo: payoutDetailsRepo,
		outboxRepo:        outboxRepo,
	}
}

func (uc *UpdateOpeningHours) Execute(
	ctx context.Context,
	restaurantID uuid.UUID,
	ownerID uuid.UUID,
	input resapp.UpdateOpeningHoursRequest,
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

	res.CompleteChecklistItem(restaurant.ChecklistOpeningHours)
	res.NotifyUpdated()

	res.WithOpeningHours(toDomainOpeningHours(input))

	err = uc.db.Transaction(func(tx *gorm.DB) error {
		if err := uc.restaurantRepo.WithTx(tx).Update(ctx, res); err != nil {
			return fmt.Errorf("failed to update restaurant: %w", err)
		}

		return resapp.DispatchEventsTx(ctx, uc.outboxRepo.WithTx(tx), res, uc.enrichUpdated(res))
	})
	if err != nil {
		return resapp.RestaurantResponse{}, err
	}

	pd, err := uc.payoutDetailsRepo.FindActiveByRestaurant(ctx, res.ID)
	if err != nil {
		return resapp.RestaurantResponse{}, fmt.Errorf("failed to fetch payout details: %w", err)
	}

	return resapp.ToRestaurantResponse(res, pd), nil
}

func (uc *UpdateOpeningHours) enrichUpdated(res *restaurant.Restaurant) resapp.Enricher {
	return func(e restaurant.DomainEvent) (event.Event, bool) {
		updated, ok := e.(restaurant.RestaurantUpdated)
		if !ok {
			return nil, false
		}

		return resapp.NewRestaurantUpdatedPayload(updated, res), true
	}
}

func toDomainOpeningHours(input resapp.UpdateOpeningHoursRequest) restaurant.OpeningHours {
	return restaurant.OpeningHours{
		Monday:    toDomainDayRanges(input.Monday),
		Tuesday:   toDomainDayRanges(input.Tuesday),
		Wednesday: toDomainDayRanges(input.Wednesday),
		Thursday:  toDomainDayRanges(input.Thursday),
		Friday:    toDomainDayRanges(input.Friday),
		Saturday:  toDomainDayRanges(input.Saturday),
		Sunday:    toDomainDayRanges(input.Sunday),
	}
}

func toDomainDayRanges(ranges []resapp.DayRangeRequest) []restaurant.DayRange {
	out := make([]restaurant.DayRange, len(ranges))
	for i, r := range ranges {
		out[i] = restaurant.DayRange{Open: r.Open, Close: r.Close}
	}
	return out
}
