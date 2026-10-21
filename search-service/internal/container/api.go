package container

import (
	"context"
	"os"

	"search-service/internal/application/query"
	"search-service/internal/infrastructure/elasticsearch"
	"search-service/internal/infrastructure/geocoder"
	"search-service/internal/interfaces/http/handlers"

	"pizza-marketplace/shared/health"
)

type APIContainer struct {
	SearchHandler        *handlers.SearchHandler
	GetRestaurantHandler *handlers.GetRestaurantHandler
	HealthHandler        *handlers.HealthHandler
}

func NewAPIContainer() (*APIContainer, error) {
	esURL := os.Getenv("ELASTICSEARCH_URL")
	openCageAPIKey := os.Getenv("OPENCAGE_API_KEY")

	es, err := elasticsearch.NewClient(esURL)
	if err != nil {
		return nil, err
	}

	// The API also owns the geocode cache index (via CachingGeocoder below),
	// not just the worker's restaurants index — ensure both exist regardless
	// of which process starts first.
	if err := elasticsearch.EnsureIndex(context.Background(), es); err != nil {
		return nil, err
	}

	searchRepo := elasticsearch.NewSearchRepository(es)
	openCage := geocoder.NewOpenCageGeocoder(openCageAPIKey)
	cachingGeocoder := elasticsearch.NewCachingGeocoder(es, openCage)
	searchRestaurants := query.NewSearchRestaurants(searchRepo, cachingGeocoder)
	searchHandler := handlers.NewSearchHandler(searchRestaurants)

	getRestaurant := query.NewGetRestaurant(searchRepo)
	getRestaurantHandler := handlers.NewGetRestaurantHandler(getRestaurant)

	// health
	readiness := health.NewReadiness(
		health.Check{Name: "elasticsearch", Pinger: elasticsearch.Pinger{ES: es}},
	)
	healthHandler := handlers.NewHealthHandler(readiness)

	return &APIContainer{
		SearchHandler:        searchHandler,
		GetRestaurantHandler: getRestaurantHandler,
		HealthHandler:        healthHandler,
	}, nil
}

func (c *APIContainer) Close() {}
