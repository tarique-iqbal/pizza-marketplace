package elasticsearch

import (
	"bytes"
	"context"
	"fmt"

	"github.com/elastic/go-elasticsearch/v8"
)

const IndexName = "restaurants"

// pizzas is a plain object array (not "nested"), so its fields flatten into
// multi-valued arrays (pizzas.name, pizzas.toppings) that a top-level
// multi_match can search directly — no nested query needed. This trades away
// precise per-pizza cross-field matching for simplicity, acceptable at this
// index's current scope.
const indexMapping = `{
  "mappings": {
    "properties": {
      "name": {"type": "text"},
      "slug": {"type": "keyword"},
      "city": {"type": "keyword"},
      "location": {"type": "geo_point"},
      "currency": {"type": "keyword"},
      "pickup": {"type": "boolean"},
      "deliveryType": {"type": "keyword"},
      "deliveryKm": {"type": "short"},
      "tags": {"type": "keyword"},
      "rating": {"type": "float"},
      "totalReviews": {"type": "integer"},
      "pizzas": {
        "properties": {
          "id": {"type": "keyword"},
          "name": {"type": "text"},
          "isVegetarian": {"type": "boolean"},
          "toppings": {"type": "keyword"}
        }
      },
      "updatedAt": {"type": "date"}
    }
  }
}`

// geocodeIndexMapping backs CachingGeocoder — a disposable key/value cache
// (address hash -> resolved coordinates), not search-facing, so it needs no
// geo_point/text mapping subtleties.
const geocodeIndexMapping = `{
  "mappings": {
    "properties": {
      "lat": {"type": "float"},
      "lon": {"type": "float"}
    }
  }
}`

// EnsureIndex is idempotent: it no-ops if either index already exists, so
// it's safe to call on every worker startup.
func EnsureIndex(ctx context.Context, es *elasticsearch.Client) error {
	if err := ensureIndex(ctx, es, IndexName, indexMapping); err != nil {
		return err
	}

	return ensureIndex(ctx, es, GeocodeIndexName, geocodeIndexMapping)
}

func ensureIndex(ctx context.Context, es *elasticsearch.Client, name, mapping string) error {
	exists, err := es.Indices.Exists([]string{name}, es.Indices.Exists.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("failed to check index existence: %w", err)
	}
	defer exists.Body.Close()

	if exists.StatusCode == 200 {
		return nil
	}

	res, err := es.Indices.Create(
		name,
		es.Indices.Create.WithContext(ctx),
		es.Indices.Create.WithBody(bytes.NewReader([]byte(mapping))),
	)
	if err != nil {
		return fmt.Errorf("failed to create index %s: %w", name, err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("failed to create index %s: %s", name, res.String())
	}

	return nil
}
