package otelelasticsearch

import (
	"github.com/elastic/elastic-transport-go/v8/elastictransport"
	"github.com/elastic/go-elasticsearch/v8"
	"go.opentelemetry.io/otel"
)

func Instrumentation() elastictransport.Instrumentation {
	return elasticsearch.NewOpenTelemetryInstrumentation(otel.GetTracerProvider(), false)
}
