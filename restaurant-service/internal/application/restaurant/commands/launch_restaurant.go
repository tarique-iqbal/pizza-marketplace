package commands

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	pizzaqry "restaurant-service/internal/application/pizza/queries"
	resapp "restaurant-service/internal/application/restaurant"
	"restaurant-service/internal/domain/payout"
	"restaurant-service/internal/domain/restaurant"
	logobs "restaurant-service/internal/infrastructure/observability/logger"
	apperr "restaurant-service/internal/shared/errors"
	"restaurant-service/internal/shared/event"
)

type LaunchRestaurant struct {
	restaurantRepo    restaurant.RestaurantRepository
	payoutDetailsRepo payout.PayoutDetailsRepository
	pizzaCatalog      *pizzaqry.PizzaCatalog
	publisher         event.EventPublisher
}

func NewLaunchRestaurant(
	restaurantRepo restaurant.RestaurantRepository,
	payoutDetailsRepo payout.PayoutDetailsRepository,
	pizzaCatalog *pizzaqry.PizzaCatalog,
	publisher event.EventPublisher,
) *LaunchRestaurant {
	return &LaunchRestaurant{
		restaurantRepo:    restaurantRepo,
		payoutDetailsRepo: payoutDetailsRepo,
		pizzaCatalog:      pizzaCatalog,
		publisher:         publisher,
	}
}

func (uc *LaunchRestaurant) Execute(
	ctx context.Context,
	restaurantID uuid.UUID,
	ownerID uuid.UUID,
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

	if err := res.Launch(); err != nil {
		return resapp.RestaurantResponse{}, fmt.Errorf("%w: %w", err, apperr.ErrConflict)
	}

	if err := uc.restaurantRepo.Update(ctx, res); err != nil {
		return resapp.RestaurantResponse{}, fmt.Errorf("failed to update restaurant: %w", err)
	}

	resapp.DispatchEvents(ctx, uc.publisher, res, uc.enrichLaunched(ctx, res))

	pd, err := uc.payoutDetailsRepo.FindActiveByRestaurant(ctx, res.ID)
	if err != nil {
		return resapp.RestaurantResponse{}, fmt.Errorf("failed to fetch payout details: %w", err)
	}

	return resapp.ToRestaurantResponse(res, pd), nil
}

func (uc *LaunchRestaurant) enrichLaunched(ctx context.Context, res *restaurant.Restaurant) resapp.Enricher {
	return func(e restaurant.DomainEvent) (event.Event, bool) {
		launched, ok := e.(restaurant.RestaurantLaunched)
		if !ok {
			return nil, false
		}

		pizzas, err := uc.pizzaCatalog.Execute(ctx, res.ID)
		if err != nil {
			logobs.FromContext(ctx).Warn("failed to load pizza catalog for restaurant.launched", "error", err)
			return nil, false
		}

		priced := pizzas[:0]
		for _, p := range pizzas {
			if p.HasActivePrice() {
				priced = append(priced, p)
			}
		}

		return resapp.NewRestaurantLaunchedPayload(launched, res, priced), true
	}
}
