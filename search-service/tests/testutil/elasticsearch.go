package testutil

import (
	"context"
	"os"
	"testing"

	"github.com/elastic/go-elasticsearch/v8"

	esinfra "search-service/internal/infrastructure/elasticsearch"
)

func ES(t *testing.T) *elasticsearch.Client {
	t.Helper()

	client, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses: []string{os.Getenv("ELASTICSEARCH_URL")},
	})
	if err != nil {
		t.Fatalf("failed to create elasticsearch client: %v", err)
	}

	res, err := client.Indices.Delete(
		[]string{esinfra.IndexName, esinfra.GeocodeIndexName},
		client.Indices.Delete.WithIgnoreUnavailable(true),
	)
	if err != nil {
		t.Fatalf("failed to reset elasticsearch indices: %v", err)
	}
	res.Body.Close()

	if err := esinfra.EnsureIndex(context.Background(), client); err != nil {
		t.Fatalf("failed to ensure elasticsearch indices: %v", err)
	}

	return client
}

func RefreshIndex(t *testing.T, client *elasticsearch.Client, index string) {
	t.Helper()

	res, err := client.Indices.Refresh(client.Indices.Refresh.WithIndex(index))
	if err != nil {
		t.Fatalf("failed to refresh index %s: %v", index, err)
	}
	res.Body.Close()
}
