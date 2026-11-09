package elasticsearch

import (
	"context"
	"fmt"
	"time"

	"github.com/elastic/go-elasticsearch/v8"

	"pizza-marketplace/shared/telemetry/otelelasticsearch"
)

type Pinger struct {
	ES *elasticsearch.Client
}

func NewClient(addr string) (*elasticsearch.Client, error) {
	return elasticsearch.NewClient(elasticsearch.Config{
		Addresses:       []string{addr},
		Instrumentation: otelelasticsearch.Instrumentation(),
	})
}

func (p Pinger) Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	res, err := p.ES.Ping(p.ES.Ping.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("elasticsearch: ping: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("elasticsearch: ping: %s", res.Status())
	}

	return nil
}
