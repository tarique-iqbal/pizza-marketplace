package commands

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	pizzaapp "restaurant-service/internal/application/pizza"
	pizzaqry "restaurant-service/internal/application/pizza/queries"
	resapp "restaurant-service/internal/application/restaurant"
	toppingapp "restaurant-service/internal/application/topping"
	"restaurant-service/internal/domain/payout"
	"restaurant-service/internal/domain/restaurant"
	"restaurant-service/internal/domain/topping"
	apperr "restaurant-service/internal/shared/errors"
	"restaurant-service/internal/shared/event"
)

type LaunchRestaurant struct {
	restaurantRepo    restaurant.RestaurantRepository
	payoutDetailsRepo payout.PayoutDetailsRepository
	pizzaCatalog      *pizzaqry.PizzaCatalog
	toppingRepo       topping.ToppingRepository
	toppingPriceRepo  topping.ToppingPriceRepository
	publisher         event.EventPublisher
}

func NewLaunchRestaurant(
	restaurantRepo restaurant.RestaurantRepository,
	payoutDetailsRepo payout.PayoutDetailsRepository,
	pizzaCatalog *pizzaqry.PizzaCatalog,
	toppingRepo topping.ToppingRepository,
	toppingPriceRepo topping.ToppingPriceRepository,
	publisher event.EventPublisher,
) *LaunchRestaurant {
	return &LaunchRestaurant{
		restaurantRepo:    restaurantRepo,
		payoutDetailsRepo: payoutDetailsRepo,
		pizzaCatalog:      pizzaCatalog,
		toppingRepo:       toppingRepo,
		toppingPriceRepo:  toppingPriceRepo,
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

	pizzas, err := uc.pizzaCatalog.Execute(ctx, res.ID)
	if err != nil {
		return resapp.RestaurantResponse{}, fmt.Errorf("failed to load pizza catalog: %w", err)
	}

	readiness := resapp.EvaluateLaunchReadiness(pizzas)
	if !readiness.MeetsMinimum() {
		return resapp.RestaurantResponse{}, fmt.Errorf("%w: %w", restaurant.ErrNotEnoughPizzas, apperr.ErrConflict)
	}

	toppingPrices, err := uc.buildToppingPriceResponses(ctx, res.ID)
	if err != nil {
		return resapp.RestaurantResponse{}, fmt.Errorf("failed to load topping prices: %w", err)
	}

	if err := res.Launch(); err != nil {
		return resapp.RestaurantResponse{}, fmt.Errorf("%w: %w", err, apperr.ErrConflict)
	}

	if err := uc.restaurantRepo.Update(ctx, res); err != nil {
		return resapp.RestaurantResponse{}, fmt.Errorf("failed to update restaurant: %w", err)
	}

	resapp.DispatchEvents(ctx, uc.publisher, res, uc.enrichLaunched(res, readiness.ReadyPizzas, toppingPrices))

	pd, err := uc.payoutDetailsRepo.FindActiveByRestaurant(ctx, res.ID)
	if err != nil {
		return resapp.RestaurantResponse{}, fmt.Errorf("failed to fetch payout details: %w", err)
	}

	return resapp.ToRestaurantResponse(res, pd), nil
}

func (uc *LaunchRestaurant) enrichLaunched(
	res *restaurant.Restaurant,
	pizzas []pizzaapp.PizzaResponse,
	toppingPrices []toppingapp.ToppingPriceResponse,
) resapp.Enricher {
	return func(e restaurant.DomainEvent) (event.Event, bool) {
		launched, ok := e.(restaurant.RestaurantLaunched)
		if !ok {
			return nil, false
		}

		return resapp.NewRestaurantLaunchedPayload(launched, res, pizzas, toppingPrices), true
	}
}

func (uc *LaunchRestaurant) buildToppingPriceResponses(
	ctx context.Context,
	restaurantID uuid.UUID,
) ([]toppingapp.ToppingPriceResponse, error) {
	toppings, err := uc.toppingRepo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list toppings: %w", err)
	}

	toppingByID := make(map[uuid.UUID]topping.Topping, len(toppings))
	for _, t := range toppings {
		toppingByID[t.ID] = t
	}

	prices, err := uc.toppingPriceRepo.ListByRestaurant(ctx, restaurantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list topping prices: %w", err)
	}

	responses := make([]toppingapp.ToppingPriceResponse, 0, len(prices))
	for _, price := range prices {
		responses = append(responses, toppingapp.ToToppingPriceResponse(price, toppingByID[price.ToppingID].Name))
	}

	return responses, nil
}
