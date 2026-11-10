package telemetry

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
)

type Label struct {
	Key   string
	Value string
}

type Counter struct {
	counter metric.Int64Counter
}

func NewCounter(scope, name, description string) Counter {
	counter, err := otel.Meter(scope).Int64Counter(name, metric.WithDescription(description))
	if err != nil {
		otel.Handle(err)
		return Counter{counter: noop.Int64Counter{}}
	}

	return Counter{counter: counter}
}

func (c Counter) Add(ctx context.Context, labels ...Label) {
	c.counter.Add(ctx, 1, metric.WithAttributes(attributes(labels)...))
}

type Histogram struct {
	histogram metric.Float64Histogram
}

func NewHistogram(scope, name, unit, description string, buckets ...float64) Histogram {
	options := []metric.Float64HistogramOption{metric.WithUnit(unit), metric.WithDescription(description)}
	if len(buckets) > 0 {
		options = append(options, metric.WithExplicitBucketBoundaries(buckets...))
	}

	histogram, err := otel.Meter(scope).Float64Histogram(name, options...)
	if err != nil {
		otel.Handle(err)
		return Histogram{histogram: noop.Float64Histogram{}}
	}

	return Histogram{histogram: histogram}
}

func (h Histogram) Record(ctx context.Context, value float64, labels ...Label) {
	h.histogram.Record(ctx, value, metric.WithAttributes(attributes(labels)...))
}

func attributes(labels []Label) []attribute.KeyValue {
	kvs := make([]attribute.KeyValue, len(labels))
	for i, label := range labels {
		kvs[i] = attribute.String(label.Key, label.Value)
	}

	return kvs
}
