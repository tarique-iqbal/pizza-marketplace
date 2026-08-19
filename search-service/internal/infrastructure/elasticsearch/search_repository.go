package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/google/uuid"

	"search-service/internal/domain/index"
)

type SearchRepository struct {
	es *elasticsearch.Client
}

func NewSearchRepository(es *elasticsearch.Client) *SearchRepository {
	return &SearchRepository{es: es}
}

type esPizza struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	IsVegetarian bool      `json:"isVegetarian"`
	Toppings     []string  `json:"toppings"`
}

type esGeoPoint struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

type esRestaurant struct {
	Name         string     `json:"name"`
	Slug         string     `json:"slug"`
	City         string     `json:"city"`
	Location     esGeoPoint `json:"location"`
	Currency     string     `json:"currency"`
	Pickup       bool       `json:"pickup"`
	DeliveryType string     `json:"deliveryType"`
	DeliveryKm   *int16     `json:"deliveryKm,omitempty"`
	Tags         []string   `json:"tags"`
	Rating       float64    `json:"rating"`
	TotalReviews int32      `json:"totalReviews"`
	Pizzas       []esPizza  `json:"pizzas"`
}

func (r *SearchRepository) UpsertSnapshot(ctx context.Context, restaurant index.IndexedRestaurant) error {
	body, err := json.Marshal(toESRestaurant(restaurant))
	if err != nil {
		return fmt.Errorf("failed to marshal document: %w", err)
	}

	res, err := r.es.Index(
		IndexName,
		bytes.NewReader(body),
		r.es.Index.WithDocumentID(restaurant.ID.String()),
		r.es.Index.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("failed to index document: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("failed to index document: %s", res.String())
	}

	return nil
}

func (r *SearchRepository) Search(ctx context.Context, q index.SearchQuery) ([]index.IndexedRestaurant, error) {
	body, err := json.Marshal(buildSearchQuery(q))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal search query: %w", err)
	}

	res, err := r.es.Search(
		r.es.Search.WithContext(ctx),
		r.es.Search.WithIndex(IndexName),
		r.es.Search.WithBody(bytes.NewReader(body)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to execute search: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("search failed: %s", res.String())
	}

	var result struct {
		Hits struct {
			Hits []struct {
				ID     string       `json:"_id"`
				Source esRestaurant `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode search response: %w", err)
	}

	restaurants := make([]index.IndexedRestaurant, 0, len(result.Hits.Hits))
	for _, hit := range result.Hits.Hits {
		id, err := uuid.Parse(hit.ID)
		if err != nil {
			continue
		}
		restaurants = append(restaurants, fromESRestaurant(id, hit.Source))
	}

	return restaurants, nil
}

// deliveryRangeScript excludes a restaurant with no configured delivery
// radius (deliveryKm unset — DeliveryType "none", or delivery-config not
// otherwise reachable) rather than treating "no radius" as "unlimited
// range." arcDistance returns meters; deliveryKm is kilometers, hence *1000.
const deliveryRangeScript = `doc['deliveryKm'].size() != 0 && ` +
	`doc['location'].arcDistance(params.lat, params.lon) <= doc['deliveryKm'].value * 1000`

func buildSearchQuery(q index.SearchQuery) map[string]any {
	var must []map[string]any
	if q.Text == "" {
		// No text means "browse what's deliverable to me" — a customer
		// opening the app expects to see nearby restaurants immediately,
		// not only after typing something.
		must = append(must, map[string]any{"match_all": map[string]any{}})
	} else {
		must = append(must, map[string]any{
			"multi_match": map[string]any{
				"query":     q.Text,
				"fields":    []string{"name^2", "pizzas.name", "pizzas.toppings"},
				"fuzziness": "AUTO",
			},
		})
	}

	filter := []map[string]any{
		{
			"script": map[string]any{
				"script": map[string]any{
					"source": deliveryRangeScript,
					"params": map[string]any{
						"lat": q.Location.Lat,
						"lon": q.Location.Lon,
					},
				},
			},
		},
	}

	return map[string]any{
		"query": map[string]any{
			// field_value_factor nudges higher-rated restaurants up on top of
			// text relevance (boost_mode "sum", not "multiply" or "replace")
			// — log1p keeps a 5.0 vs 4.8 restaurant from swamping actual
			// text-match quality, and "missing": 0 means an unrated
			// restaurant (rating 0) gets no boost rather than being
			// penalized relative to text-only scoring. Ranking stays
			// relevance+rating even once the delivery-range filter narrows
			// the candidate set — every remaining hit can already reach the
			// customer, so which one is a few hundred meters closer matters
			// less than which one actually matches what they're looking for.
			"function_score": map[string]any{
				"query": map[string]any{
					"bool": map[string]any{
						"must":   must,
						"filter": filter,
					},
				},
				"field_value_factor": map[string]any{
					"field":    "rating",
					"modifier": "log1p",
					"factor":   1,
					"missing":  0,
				},
				"boost_mode": "sum",
			},
		},
	}
}

func toESRestaurant(r index.IndexedRestaurant) esRestaurant {
	location := esGeoPoint{Lat: r.Location.Lat, Lon: r.Location.Lon}

	pizzas := make([]esPizza, 0, len(r.Pizzas))
	for _, p := range r.Pizzas {
		pizzas = append(pizzas, esPizza{
			ID:           p.ID,
			Name:         p.Name,
			IsVegetarian: p.IsVegetarian,
			Toppings:     p.Toppings,
		})
	}

	return esRestaurant{
		Name:         r.Name,
		Slug:         r.Slug,
		City:         r.City,
		Location:     location,
		Currency:     r.Currency,
		Pickup:       r.Pickup,
		DeliveryType: r.DeliveryType,
		DeliveryKm:   r.DeliveryKm,
		Tags:         r.Tags,
		Rating:       r.Rating,
		TotalReviews: r.TotalReviews,
		Pizzas:       pizzas,
	}
}

func fromESRestaurant(id uuid.UUID, doc esRestaurant) index.IndexedRestaurant {
	location := index.GeoPoint{Lat: doc.Location.Lat, Lon: doc.Location.Lon}

	pizzas := make([]index.IndexedPizza, 0, len(doc.Pizzas))
	for _, p := range doc.Pizzas {
		pizzas = append(pizzas, index.IndexedPizza{
			ID:           p.ID,
			Name:         p.Name,
			IsVegetarian: p.IsVegetarian,
			Toppings:     p.Toppings,
		})
	}

	return index.IndexedRestaurant{
		ID:           id,
		Name:         doc.Name,
		Slug:         doc.Slug,
		City:         doc.City,
		Location:     location,
		Currency:     doc.Currency,
		Pickup:       doc.Pickup,
		DeliveryType: doc.DeliveryType,
		DeliveryKm:   doc.DeliveryKm,
		Tags:         doc.Tags,
		Rating:       doc.Rating,
		TotalReviews: doc.TotalReviews,
		Pizzas:       pizzas,
	}
}
