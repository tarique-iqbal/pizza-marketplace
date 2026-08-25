package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

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
	UpdatedAt    time.Time `json:"updatedAt"`
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
	UpdatedAt    time.Time  `json:"updatedAt"`
}

type esRestaurantFields struct {
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
	UpdatedAt    time.Time  `json:"updatedAt"`
}

type esScript struct {
	Source string         `json:"source"`
	Lang   string         `json:"lang"`
	Params map[string]any `json:"params"`
}

type esUpdateBody struct {
	Script esScript `json:"script"`
	Upsert any      `json:"upsert,omitempty"`
}

// upsertSnapshotScript fully replaces _source (equivalent to the plain
// Index() overwrite this used to be), but only if the incoming event is
// newer than whatever is already indexed — otherwise it's a no-op. This
// guards against a stale, retried restaurant.launched redelivery (see
// rabbitmq_consumer.go's retry-to-back-of-queue behavior) clobbering a
// restaurant.updated write that landed in between.
const upsertSnapshotScript = `if (!ctx._source.containsKey('updatedAt') || ` +
	`params.updatedAtMillis > Instant.parse(ctx._source.updatedAt).toEpochMilli()) { ` +
	`ctx._source = params.doc; ` +
	`} else { ` +
	`ctx.op = 'noop'; ` +
	`}`

const updateFieldsScript = `if (!ctx._source.containsKey('updatedAt') || ` +
	`params.updatedAtMillis > Instant.parse(ctx._source.updatedAt).toEpochMilli()) { ` +
	`ctx._source.name = params.doc.name; ` +
	`ctx._source.slug = params.doc.slug; ` +
	`ctx._source.city = params.doc.city; ` +
	`ctx._source.location = params.doc.location; ` +
	`ctx._source.currency = params.doc.currency; ` +
	`ctx._source.pickup = params.doc.pickup; ` +
	`ctx._source.deliveryType = params.doc.deliveryType; ` +
	`ctx._source.deliveryKm = params.doc.deliveryKm; ` +
	`ctx._source.tags = params.doc.tags; ` +
	`ctx._source.rating = params.doc.rating; ` +
	`ctx._source.totalReviews = params.doc.totalReviews; ` +
	`ctx._source.updatedAt = params.doc.updatedAt; ` +
	`} else { ` +
	`ctx.op = 'noop'; ` +
	`}`

// pizzaUpsertScript replaces-in-place or appends a single pizza within the
// document's pizzas array, guarded per-pizza by its own updatedAt — a
// restaurant-level updatedAt guard can't be reused here since pizza edits
// (e.g. SetPizzaPrices) never touch the restaurants row, so restaurant.
// updated's timestamp has no relation to when a given pizza last changed.
const pizzaUpsertScript = `def idx = -1; ` +
	`for (int i = 0; i < ctx._source.pizzas.size(); i++) { ` +
	`if (ctx._source.pizzas[i].id == params.pizza.id) { idx = i; break; } ` +
	`} ` +
	`if (idx == -1) { ` +
	`ctx._source.pizzas.add(params.pizza); ` +
	`} else if (params.updatedAtMillis > Instant.parse(ctx._source.pizzas[idx].updatedAt).toEpochMilli()) { ` +
	`ctx._source.pizzas[idx] = params.pizza; ` +
	`} else { ` +
	`ctx.op = 'noop'; ` +
	`}`

// pizzaRemoveScript drops a pizza gone archived or unpriced (no longer
// orderable) from the indexed menu. A pizza already absent is a no-op, not
// an error — removal must be idempotent against redelivery.
const pizzaRemoveScript = `def idx = -1; ` +
	`for (int i = 0; i < ctx._source.pizzas.size(); i++) { ` +
	`if (ctx._source.pizzas[i].id == params.pizzaId) { idx = i; break; } ` +
	`} ` +
	`if (idx == -1) { ` +
	`ctx.op = 'noop'; ` +
	`} else if (params.updatedAtMillis > Instant.parse(ctx._source.pizzas[idx].updatedAt).toEpochMilli()) { ` +
	`ctx._source.pizzas.remove(idx); ` +
	`} else { ` +
	`ctx.op = 'noop'; ` +
	`}`

func (r *SearchRepository) UpsertPizza(ctx context.Context, restaurantID uuid.UUID, pizza index.IndexedPizza) error {
	doc := esPizza{
		ID:           pizza.ID,
		Name:         pizza.Name,
		IsVegetarian: pizza.IsVegetarian,
		Toppings:     pizza.Toppings,
		UpdatedAt:    pizza.UpdatedAt,
	}

	body, err := json.Marshal(esUpdateBody{
		Script: esScript{
			Source: pizzaUpsertScript,
			Lang:   "painless",
			Params: map[string]any{
				"pizza":           doc,
				"updatedAtMillis": pizza.UpdatedAt.UnixMilli(),
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to marshal pizza: %w", err)
	}

	res, err := r.es.Update(IndexName, restaurantID.String(), bytes.NewReader(body), r.es.Update.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("failed to upsert pizza: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("failed to upsert pizza: %s", res.String())
	}

	return nil
}

func (r *SearchRepository) RemovePizza(
	ctx context.Context,
	restaurantID, pizzaID uuid.UUID,
	updatedAt time.Time,
) error {
	body, err := json.Marshal(esUpdateBody{
		Script: esScript{
			Source: pizzaRemoveScript,
			Lang:   "painless",
			Params: map[string]any{
				"pizzaId":         pizzaID.String(),
				"updatedAtMillis": updatedAt.UnixMilli(),
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to marshal pizza removal: %w", err)
	}

	res, err := r.es.Update(IndexName, restaurantID.String(), bytes.NewReader(body), r.es.Update.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("failed to remove pizza: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("failed to remove pizza: %s", res.String())
	}

	return nil
}

func (r *SearchRepository) UpsertSnapshot(ctx context.Context, restaurant index.IndexedRestaurant) error {
	doc := toESRestaurant(restaurant)

	body, err := json.Marshal(esUpdateBody{
		Script: esScript{
			Source: upsertSnapshotScript,
			Lang:   "painless",
			Params: map[string]any{
				"doc":             doc,
				"updatedAtMillis": restaurant.UpdatedAt.UnixMilli(),
			},
		},
		Upsert: doc,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal document: %w", err)
	}

	res, err := r.es.Update(
		IndexName,
		restaurant.ID.String(),
		bytes.NewReader(body),
		r.es.Update.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("failed to upsert document: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("failed to upsert document: %s", res.String())
	}

	return nil
}

func (r *SearchRepository) UpdateFields(ctx context.Context, id uuid.UUID, fields index.RestaurantFields) error {
	body, err := json.Marshal(esUpdateBody{
		Script: esScript{
			Source: updateFieldsScript,
			Lang:   "painless",
			Params: map[string]any{
				"doc":             toESRestaurantFields(fields),
				"updatedAtMillis": fields.UpdatedAt.UnixMilli(),
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to marshal fields: %w", err)
	}

	res, err := r.es.Update(IndexName, id.String(), bytes.NewReader(body), r.es.Update.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("failed to update document: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("failed to update document: %s", res.String())
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

func toESRestaurantFields(f index.RestaurantFields) esRestaurantFields {
	return esRestaurantFields{
		Name:         f.Name,
		Slug:         f.Slug,
		City:         f.City,
		Location:     esGeoPoint{Lat: f.Location.Lat, Lon: f.Location.Lon},
		Currency:     f.Currency,
		Pickup:       f.Pickup,
		DeliveryType: f.DeliveryType,
		DeliveryKm:   f.DeliveryKm,
		Tags:         f.Tags,
		Rating:       f.Rating,
		TotalReviews: f.TotalReviews,
		UpdatedAt:    f.UpdatedAt,
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
			UpdatedAt:    p.UpdatedAt,
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
		UpdatedAt:    r.UpdatedAt,
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
			UpdatedAt:    p.UpdatedAt,
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
		UpdatedAt:    doc.UpdatedAt,
	}
}
