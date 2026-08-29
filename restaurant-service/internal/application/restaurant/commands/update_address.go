package commands

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	goslug "github.com/gosimple/slug"
	"gorm.io/gorm"

	resapp "restaurant-service/internal/application/restaurant"
	"restaurant-service/internal/domain/outbox"
	"restaurant-service/internal/domain/payout"
	"restaurant-service/internal/domain/restaurant"
	apperr "restaurant-service/internal/shared/errors"
	"restaurant-service/internal/shared/event"
)

type UpdateAddress struct {
	db                *gorm.DB
	geocoder          restaurant.Geocoder
	restaurantRepo    restaurant.RestaurantRepository
	payoutDetailsRepo payout.PayoutDetailsRepository
	outboxRepo        outbox.OutboxRepository
}

func NewUpdateAddress(
	db *gorm.DB,
	geocoder restaurant.Geocoder,
	restaurantRepo restaurant.RestaurantRepository,
	payoutDetailsRepo payout.PayoutDetailsRepository,
	outboxRepo outbox.OutboxRepository,
) *UpdateAddress {
	return &UpdateAddress{
		db:                db,
		geocoder:          geocoder,
		restaurantRepo:    restaurantRepo,
		payoutDetailsRepo: payoutDetailsRepo,
		outboxRepo:        outboxRepo,
	}
}

func (uc *UpdateAddress) Execute(
	ctx context.Context,
	restaurantID uuid.UUID,
	ownerID uuid.UUID,
	input resapp.UpdateAddressRequest,
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

	addr := restaurant.Address{
		House:      input.House,
		Street:     input.Street,
		PostalCode: input.PostalCode,
		City:       input.City,
	}

	var lat, lon float64
	var timezone string

	if res.Address == addr {
		lat, lon, timezone = *res.Lat, *res.Lon, *res.Timezone
	} else {
		lat, lon, timezone, err = uc.geocoder.GeocodeAddress(ctx, addr)
		if err != nil {
			return resapp.RestaurantResponse{}, fmt.Errorf("failed to geocode address: %w", err)
		}
	}

	slug, err := uc.generateUniqueSlug(ctx, res.ID, res.Name, input.City, input.Street)
	if err != nil {
		return resapp.RestaurantResponse{}, fmt.Errorf("failed to generate slug: %w", err)
	}

	res.CompleteChecklistItem(restaurant.ChecklistAddress)
	res.NotifyUpdated()

	res.WithSlug(slug).
		WithAddress(addr).
		WithCoordinates(lat, lon, timezone)

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

func (uc *UpdateAddress) generateUniqueSlug(
	ctx context.Context,
	restaurantID uuid.UUID,
	name, city, street string,
) (string, error) {
	base := goslug.Make(fmt.Sprintf("%s-%s-%s", name, city, street))

	res, err := uc.restaurantRepo.FindBySlug(ctx, base)
	if err != nil {
		return "", fmt.Errorf("failed to find restaurant by slug: %w", err)
	}

	if res == nil || res.ID == restaurantID {
		return base, nil
	}

	for i := 2; i <= 9; i++ {
		extended := fmt.Sprintf("%s-%d", base, i)

		res, err := uc.restaurantRepo.FindBySlug(ctx, extended)
		if err != nil {
			return "", fmt.Errorf("failed to find restaurant by slug: %w", err)
		}

		if res == nil || res.ID == restaurantID {
			return extended, nil
		}
	}

	return "", fmt.Errorf("failed to generate unique slug")
}

func (uc *UpdateAddress) enrichUpdated(res *restaurant.Restaurant) resapp.Enricher {
	return func(e restaurant.DomainEvent) (event.Event, bool) {
		updated, ok := e.(restaurant.RestaurantUpdated)
		if !ok {
			return nil, false
		}

		return resapp.NewRestaurantUpdatedPayload(updated, res), true
	}
}
