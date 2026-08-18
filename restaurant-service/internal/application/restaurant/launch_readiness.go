package restaurant

import (
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
	Status            restaurant.RestaurantStatus `json:"status"`
	ReadyToLaunch     bool                        `json:"readyToLaunch"`
	MinPizzasRequired int                         `json:"minPizzasRequired"`
	ReadyPizzas       []pizzaapp.PizzaResponse    `json:"readyPizzas"`
	IncompletePizzas  []pizzaapp.PizzaResponse    `json:"incompletePizzas"`
}

func ToLaunchReadinessResponse(r *restaurant.Restaurant, readiness LaunchReadiness) LaunchReadinessResponse {
	return LaunchReadinessResponse{
		Status:            r.Status,
		ReadyToLaunch:     r.Status == restaurant.StatusApproved && readiness.MeetsMinimum(),
		MinPizzasRequired: MinPizzasToLaunch,
		ReadyPizzas:       readiness.ReadyPizzas,
		IncompletePizzas:  readiness.IncompletePizzas,
	}
}
