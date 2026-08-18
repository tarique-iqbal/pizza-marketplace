package restaurant

import (
	"fmt"

	pizzaapp "restaurant-service/internal/application/pizza"
	"restaurant-service/internal/domain/restaurant"
)

const MinPizzasToLaunch = 2

type LaunchReadiness struct {
	ReadyPizzas      []pizzaapp.PizzaResponse
	IncompletePizzas []pizzaapp.PizzaResponse
}

func (r LaunchReadiness) MeetsMinimum() bool {
	return len(r.ReadyPizzas) >= MinPizzasToLaunch
}

func EvaluateLaunchReadiness(pizzas []pizzaapp.PizzaResponse) LaunchReadiness {
	readiness := LaunchReadiness{
		ReadyPizzas:      make([]pizzaapp.PizzaResponse, 0, len(pizzas)),
		IncompletePizzas: make([]pizzaapp.PizzaResponse, 0),
	}

	for _, p := range pizzas {
		if p.HasActivePrice() {
			readiness.ReadyPizzas = append(readiness.ReadyPizzas, p)
		} else {
			readiness.IncompletePizzas = append(readiness.IncompletePizzas, p)
		}
	}

	return readiness
}

type LaunchReadinessResponse struct {
	Name              string                      `json:"name"`
	Status            restaurant.RestaurantStatus `json:"status"`
	ReadyToLaunch     bool                        `json:"readyToLaunch"`
	MinPizzasRequired int                         `json:"minPizzasRequired"`
	ReadyPizzas       []pizzaapp.PizzaResponse    `json:"readyPizzas"`
	IncompletePizzas  []pizzaapp.PizzaResponse    `json:"incompletePizzas"`
	Comment           string                      `json:"comment"`
}

func ToLaunchReadinessResponse(r *restaurant.Restaurant, readiness LaunchReadiness) LaunchReadinessResponse {
	readyToLaunch := r.Status == restaurant.StatusApproved && readiness.MeetsMinimum()

	return LaunchReadinessResponse{
		Name:              r.Name,
		Status:            r.Status,
		ReadyToLaunch:     readyToLaunch,
		MinPizzasRequired: MinPizzasToLaunch,
		ReadyPizzas:       readiness.ReadyPizzas,
		IncompletePizzas:  readiness.IncompletePizzas,
		Comment:           launchReadinessComment(r.Status, readyToLaunch, readiness),
	}
}

func launchReadinessComment(status restaurant.RestaurantStatus, readyToLaunch bool, readiness LaunchReadiness) string {
	if readyToLaunch {
		return "Welcome to launch! Your restaurant is ready to go live."
	}

	switch status {
	case restaurant.StatusDraft:
		return "Complete your onboarding checklist before you can be reviewed for launch."
	case restaurant.StatusReview:
		return "Waiting for admin approval before you can launch."
	}

	missing := MinPizzasToLaunch - len(readiness.ReadyPizzas)

	return fmt.Sprintf("Add %d more priced pizza(s) to reach the minimum of %d.", missing, MinPizzasToLaunch)
}
