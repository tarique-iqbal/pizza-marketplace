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
	apperr "search-service/internal/shared/errors"
)

type SearchRepository struct {
	es *elasticsearch.Client
}

func NewSearchRepository(es *elasticsearch.Client) *SearchRepository {
	return &SearchRepository{es: es}
}

type esPizzaPrice struct {
	SizeID     uuid.UUID `json:"sizeId"`
	DiameterCm int16     `json:"diameterCm"`
	Price      string    `json:"price"`
}

type esPizza struct {
	ID           uuid.UUID      `json:"id"`
	Name         string         `json:"name"`
	IsVegetarian bool           `json:"isVegetarian"`
	Toppings     []string       `json:"toppings"`
	Prices       []esPizzaPrice `json:"prices"`
	UpdatedAt    time.Time      `json:"updatedAt"`
}

type esGeoPoint struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

type esToppingPrice struct {
	ToppingID  uuid.UUID `json:"toppingId"`
	Name       string    `json:"name"`
	ExtraPrice string    `json:"extraPrice"`
}

type esOpeningHours struct {
	Weekday string `json:"weekday"`
	Open    string `json:"open"`
	Close   string `json:"close"`
}

type esRestaurant struct {
	Name                   string           `json:"name"`
	Slug                   string           `json:"slug"`
	City                   string           `json:"city"`
	Location               esGeoPoint       `json:"location"`
	Timezone               string           `json:"timezone"`
	Currency               string           `json:"currency"`
	Pickup                 bool             `json:"pickup"`
	DeliveryType           string           `json:"deliveryType"`
	DeliveryKm             *int16           `json:"deliveryKm,omitempty"`
	DeliveryTimeMin        *int16           `json:"deliveryTimeMin,omitempty"`
	DeliveryTimeMax        *int16           `json:"deliveryTimeMax,omitempty"`
	MinimumOrder           float64          `json:"minimumOrder"`
	Tags                   []string         `json:"tags"`
	OpeningHours           []esOpeningHours `json:"openingHours"`
	Rating                 float64          `json:"rating"`
	TotalReviews           int32            `json:"totalReviews"`
	Pizzas                 []esPizza        `json:"pizzas"`
	ToppingPrices          []esToppingPrice `json:"toppingPrices"`
	UpdatedAt              time.Time        `json:"updatedAt"`
	ToppingPricesUpdatedAt time.Time        `json:"toppingPricesUpdatedAt"`
}

type esRestaurantFields struct {
	Name            string           `json:"name"`
	Slug            string           `json:"slug"`
	City            string           `json:"city"`
	Location        esGeoPoint       `json:"location"`
	Timezone        string           `json:"timezone"`
	Currency        string           `json:"currency"`
	Pickup          bool             `json:"pickup"`
	DeliveryType    string           `json:"deliveryType"`
	DeliveryKm      *int16           `json:"deliveryKm,omitempty"`
	DeliveryTimeMin *int16           `json:"deliveryTimeMin,omitempty"`
	DeliveryTimeMax *int16           `json:"deliveryTimeMax,omitempty"`
	MinimumOrder    float64          `json:"minimumOrder"`
	Tags            []string         `json:"tags"`
	OpeningHours    []esOpeningHours `json:"openingHours"`
	Rating          float64          `json:"rating"`
	TotalReviews    int32            `json:"totalReviews"`
	UpdatedAt       time.Time        `json:"updatedAt"`
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
	`ctx._source.timezone = params.doc.timezone; ` +
	`ctx._source.currency = params.doc.currency; ` +
	`ctx._source.pickup = params.doc.pickup; ` +
	`ctx._source.deliveryType = params.doc.deliveryType; ` +
	`ctx._source.deliveryKm = params.doc.deliveryKm; ` +
	`ctx._source.deliveryTimeMin = params.doc.deliveryTimeMin; ` +
	`ctx._source.deliveryTimeMax = params.doc.deliveryTimeMax; ` +
	`ctx._source.minimumOrder = params.doc.minimumOrder; ` +
	`ctx._source.tags = params.doc.tags; ` +
	`ctx._source.openingHours = params.doc.openingHours; ` +
	`ctx._source.rating = params.doc.rating; ` +
	`ctx._source.totalReviews = params.doc.totalReviews; ` +
	`ctx._source.updatedAt = params.doc.updatedAt; ` +
	`} else { ` +
	`ctx.op = 'noop'; ` +
	`}`

// toppingPricesUpdateScript is guarded by its own toppingPricesUpdatedAt
// field, not the restaurant-level updatedAt — SetToppingPrices never touches
// the restaurants row (same reason pizzas need their own per-pizza guard
// instead of sharing updatedAt).
const toppingPricesUpdateScript = `if (!ctx._source.containsKey('toppingPricesUpdatedAt') || ` +
	`params.updatedAtMillis > Instant.parse(ctx._source.toppingPricesUpdatedAt).toEpochMilli()) { ` +
	`ctx._source.toppingPrices = params.toppingPrices; ` +
	`ctx._source.toppingPricesUpdatedAt = params.updatedAt; ` +
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
		Prices:       toESPizzaPrices(pizza.Prices),
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

func (r *SearchRepository) UpdateToppingPrices(
	ctx context.Context,
	restaurantID uuid.UUID,
	prices []index.IndexedToppingPrice,
	updatedAt time.Time,
) error {
	body, err := json.Marshal(esUpdateBody{
		Script: esScript{
			Source: toppingPricesUpdateScript,
			Lang:   "painless",
			Params: map[string]any{
				"toppingPrices":   toESToppingPrices(prices),
				"updatedAt":       updatedAt,
				"updatedAtMillis": updatedAt.UnixMilli(),
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to marshal topping prices: %w", err)
	}

	res, err := r.es.Update(IndexName, restaurantID.String(), bytes.NewReader(body), r.es.Update.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("failed to update topping prices: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("failed to update topping prices: %s", res.String())
	}

	return nil
}

func (r *SearchRepository) FindByID(ctx context.Context, id uuid.UUID) (index.IndexedRestaurant, error) {
	res, err := r.es.Get(IndexName, id.String(), r.es.Get.WithContext(ctx))
	if err != nil {
		return index.IndexedRestaurant{}, fmt.Errorf("failed to execute get: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode == 404 {
		return index.IndexedRestaurant{}, fmt.Errorf("restaurant %s: %w", id, apperr.ErrNotFound)
	}

	if res.IsError() {
		return index.IndexedRestaurant{}, fmt.Errorf("get failed: %s", res.String())
	}

	var result struct {
		Found  bool         `json:"found"`
		Source esRestaurant `json:"_source"`
	}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return index.IndexedRestaurant{}, fmt.Errorf("failed to decode get response: %w", err)
	}

	if !result.Found {
		return index.IndexedRestaurant{}, fmt.Errorf("restaurant %s: %w", id, apperr.ErrNotFound)
	}

	return fromESRestaurant(id, result.Source), nil
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

// openNowScript reads the document's own timezone (not a query param) so
// "now" is localized per-restaurant, not per-search. A document with no
// stored timezone is excluded rather than erroring — same "unknown state
// excluded" convention deliveryRangeScript uses for a missing deliveryKm.
// openingHours is a plain object array, not "nested" (same tradeoff as
// pizzas), so its sub-fields flatten into parallel same-order doc-value
// arrays — index i of weekdays/opens/closes all describe the same range.
const openNowScript = `if (!doc.containsKey('timezone') || doc['timezone'].size() == 0 || doc['timezone'].value.isEmpty()) { return false; } ` +
	`def zdt = ZonedDateTime.ofInstant(Instant.ofEpochMilli(params.nowMillis), ZoneId.of(doc['timezone'].value)); ` +
	`def weekday = zdt.getDayOfWeek().toString().toLowerCase(); ` +
	`def hh = zdt.getHour(); def mm = zdt.getMinute(); ` +
	`def now = (hh < 10 ? "0" + hh : "" + hh) + ":" + (mm < 10 ? "0" + mm : "" + mm); ` +
	`def weekdays = doc['openingHours.weekday']; ` +
	`def opens = doc['openingHours.open']; ` +
	`def closes = doc['openingHours.close']; ` +
	`for (int i = 0; i < weekdays.size(); i++) { ` +
	`if (weekdays.get(i).equals(weekday) && opens.get(i).compareTo(now) <= 0 && closes.get(i).compareTo(now) > 0) { return true; } ` +
	`} ` +
	`return false;`

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

	deliveryClause := map[string]any{
		"script": map[string]any{
			"script": map[string]any{
				"source": deliveryRangeScript,
				"params": map[string]any{
					"lat": q.Location.Lat,
					"lon": q.Location.Lon,
				},
			},
		},
	}
	pickupClause := map[string]any{"term": map[string]any{"pickup": true}}

	var fulfillmentClause map[string]any
	switch q.Fulfillment {
	case "pickup":
		fulfillmentClause = pickupClause
	default:
		// "delivery", or unset — a plain, filter-less search defaults to
		// delivery-only; a customer must explicitly ask fulfillment=pickup
		// to see pickup-only restaurants.
		fulfillmentClause = deliveryClause
	}

	filter := []map[string]any{fulfillmentClause}

	for _, tag := range q.Tags {
		filter = append(filter, map[string]any{"term": map[string]any{"tags": tag}})
	}

	if q.OpenNow {
		filter = append(filter, map[string]any{
			"script": map[string]any{
				"script": map[string]any{
					"source": openNowScript,
					"params": map[string]any{
						"nowMillis": time.Now().UTC().UnixMilli(),
					},
				},
			},
		})
	}

	query := map[string]any{
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
	}

	body := map[string]any{"query": query}

	// An explicit sort replaces relevance/rating ordering rather than
	// blending with it — a customer who asks to sort by distance wants
	// distance order, not distance-nudged-by-text-relevance. The
	// underlying bool query (text match + every filter above) is
	// unchanged either way.
	switch q.Sort {
	case "distance":
		body["sort"] = []map[string]any{
			{
				"_geo_distance": map[string]any{
					"location": map[string]any{"lat": q.Location.Lat, "lon": q.Location.Lon},
					"order":    "asc",
					"unit":     "km",
				},
			},
		}
	case "minimumOrder":
		body["sort"] = []map[string]any{
			{"minimumOrder": map[string]any{"order": "asc"}},
		}
	case "deliveryTime":
		body["sort"] = []map[string]any{
			{"deliveryTimeMin": map[string]any{"order": "asc"}},
		}
	}

	return body
}

func toESRestaurantFields(f index.RestaurantFields) esRestaurantFields {
	return esRestaurantFields{
		Name:            f.Name,
		Slug:            f.Slug,
		City:            f.City,
		Location:        esGeoPoint{Lat: f.Location.Lat, Lon: f.Location.Lon},
		Timezone:        f.Timezone,
		Currency:        f.Currency,
		Pickup:          f.Pickup,
		DeliveryType:    f.DeliveryType,
		DeliveryKm:      f.DeliveryKm,
		DeliveryTimeMin: f.DeliveryTimeMin,
		DeliveryTimeMax: f.DeliveryTimeMax,
		MinimumOrder:    f.MinimumOrder,
		Tags:            f.Tags,
		OpeningHours:    toESOpeningHours(f.OpeningHours),
		Rating:          f.Rating,
		TotalReviews:    f.TotalReviews,
		UpdatedAt:       f.UpdatedAt,
	}
}

func toESOpeningHours(hours []index.IndexedOpeningHours) []esOpeningHours {
	out := make([]esOpeningHours, 0, len(hours))
	for _, h := range hours {
		out = append(out, esOpeningHours{Weekday: h.Weekday, Open: h.Open, Close: h.Close})
	}

	return out
}

func toIndexedOpeningHours(hours []esOpeningHours) []index.IndexedOpeningHours {
	out := make([]index.IndexedOpeningHours, 0, len(hours))
	for _, h := range hours {
		out = append(out, index.IndexedOpeningHours{Weekday: h.Weekday, Open: h.Open, Close: h.Close})
	}

	return out
}

func toESPizzaPrices(prices []index.IndexedPizzaPrice) []esPizzaPrice {
	out := make([]esPizzaPrice, 0, len(prices))
	for _, p := range prices {
		out = append(out, esPizzaPrice{SizeID: p.SizeID, DiameterCm: p.DiameterCm, Price: p.Price})
	}

	return out
}

func toIndexedPizzaPrices(prices []esPizzaPrice) []index.IndexedPizzaPrice {
	out := make([]index.IndexedPizzaPrice, 0, len(prices))
	for _, p := range prices {
		out = append(out, index.IndexedPizzaPrice{SizeID: p.SizeID, DiameterCm: p.DiameterCm, Price: p.Price})
	}

	return out
}

func toESToppingPrices(prices []index.IndexedToppingPrice) []esToppingPrice {
	out := make([]esToppingPrice, 0, len(prices))
	for _, p := range prices {
		out = append(out, esToppingPrice{ToppingID: p.ToppingID, Name: p.Name, ExtraPrice: p.ExtraPrice})
	}

	return out
}

func toIndexedToppingPrices(prices []esToppingPrice) []index.IndexedToppingPrice {
	out := make([]index.IndexedToppingPrice, 0, len(prices))
	for _, p := range prices {
		out = append(out, index.IndexedToppingPrice{ToppingID: p.ToppingID, Name: p.Name, ExtraPrice: p.ExtraPrice})
	}

	return out
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
			Prices:       toESPizzaPrices(p.Prices),
			UpdatedAt:    p.UpdatedAt,
		})
	}

	return esRestaurant{
		Name:                   r.Name,
		Slug:                   r.Slug,
		City:                   r.City,
		Location:               location,
		Timezone:               r.Timezone,
		Currency:               r.Currency,
		Pickup:                 r.Pickup,
		DeliveryType:           r.DeliveryType,
		DeliveryKm:             r.DeliveryKm,
		DeliveryTimeMin:        r.DeliveryTimeMin,
		DeliveryTimeMax:        r.DeliveryTimeMax,
		MinimumOrder:           r.MinimumOrder,
		Tags:                   r.Tags,
		OpeningHours:           toESOpeningHours(r.OpeningHours),
		Rating:                 r.Rating,
		TotalReviews:           r.TotalReviews,
		Pizzas:                 pizzas,
		ToppingPrices:          toESToppingPrices(r.ToppingPrices),
		UpdatedAt:              r.UpdatedAt,
		ToppingPricesUpdatedAt: r.ToppingPricesUpdatedAt,
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
			Prices:       toIndexedPizzaPrices(p.Prices),
			UpdatedAt:    p.UpdatedAt,
		})
	}

	return index.IndexedRestaurant{
		ID:                     id,
		Name:                   doc.Name,
		Slug:                   doc.Slug,
		City:                   doc.City,
		Location:               location,
		Timezone:               doc.Timezone,
		Currency:               doc.Currency,
		Pickup:                 doc.Pickup,
		DeliveryType:           doc.DeliveryType,
		DeliveryKm:             doc.DeliveryKm,
		DeliveryTimeMin:        doc.DeliveryTimeMin,
		DeliveryTimeMax:        doc.DeliveryTimeMax,
		MinimumOrder:           doc.MinimumOrder,
		Tags:                   doc.Tags,
		OpeningHours:           toIndexedOpeningHours(doc.OpeningHours),
		Rating:                 doc.Rating,
		TotalReviews:           doc.TotalReviews,
		Pizzas:                 pizzas,
		ToppingPrices:          toIndexedToppingPrices(doc.ToppingPrices),
		UpdatedAt:              doc.UpdatedAt,
		ToppingPricesUpdatedAt: doc.ToppingPricesUpdatedAt,
	}
}
