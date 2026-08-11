package commands

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	resapp "restaurant-service/internal/application/restaurant"
	"restaurant-service/internal/domain/restaurant"
	apperr "restaurant-service/internal/shared/errors"
)

type UpdateOpeningHours struct {
	restaurantRepo restaurant.RestaurantRepository
}

func NewUpdateOpeningHours(
	restaurantRepo restaurant.RestaurantRepository,
) *UpdateOpeningHours {
	return &UpdateOpeningHours{
		restaurantRepo: restaurantRepo,
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

	res.Checklist.Complete(restaurant.ChecklistOpeningHours)

	res.WithOpeningHours(toDomainOpeningHours(input))

	if err := uc.restaurantRepo.Update(ctx, res); err != nil {
		return resapp.RestaurantResponse{}, fmt.Errorf("failed to update restaurant: %w", err)
	}

	return resapp.ToRestaurantResponse(res), nil
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
