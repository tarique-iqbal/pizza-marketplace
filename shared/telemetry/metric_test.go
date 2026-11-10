package telemetry

import (
	"context"
	"slices"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric/noop"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func newTestMeterReader(t *testing.T) *sdkmetric.ManualReader {
	reader := sdkmetric.NewManualReader()
	otel.SetMeterProvider(sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader)))
	t.Cleanup(func() { otel.SetMeterProvider(noop.NewMeterProvider()) })

	return reader
}

func collectMetric(t *testing.T, reader *sdkmetric.ManualReader, name string) metricdata.Metrics {
	var rm metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &rm); err != nil {
		t.Fatalf("collect: %v", err)
	}

	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == name {
				return m
			}
		}
	}

	t.Fatalf("metric %q not collected", name)
	return metricdata.Metrics{}
}

func TestCounter_Add_CountsPerLabelSet(t *testing.T) {
	reader := newTestMeterReader(t)
	counter := NewCounter("test", "orders.placed", "Orders placed.")

	ctx := context.Background()
	counter.Add(ctx, Label{Key: "fulfillment", Value: "delivery"})
	counter.Add(ctx, Label{Key: "fulfillment", Value: "delivery"})
	counter.Add(ctx, Label{Key: "fulfillment", Value: "pickup"})

	m := collectMetric(t, reader, "orders.placed")
	sum, ok := m.Data.(metricdata.Sum[int64])
	if !ok {
		t.Fatalf("expected Sum[int64], got %T", m.Data)
	}
	if !sum.IsMonotonic {
		t.Fatal("expected a monotonic sum")
	}

	got := map[string]int64{}
	for _, dp := range sum.DataPoints {
		value, _ := dp.Attributes.Value(attribute.Key("fulfillment"))
		got[value.AsString()] = dp.Value
	}

	if len(got) != 2 || got["delivery"] != 2 || got["pickup"] != 1 {
		t.Fatalf("expected delivery=2 pickup=1, got %v", got)
	}
}

func TestHistogram_Record_UsesUnitAndBuckets(t *testing.T) {
	reader := newTestMeterReader(t)
	histogram := NewHistogram("test", "order.completion.duration", "s", "Completion time.", 300, 900)

	histogram.Record(context.Background(), 600, Label{Key: "fulfillment", Value: "pickup"})

	m := collectMetric(t, reader, "order.completion.duration")
	if m.Unit != "s" {
		t.Fatalf("expected unit s, got %q", m.Unit)
	}

	hist, ok := m.Data.(metricdata.Histogram[float64])
	if !ok {
		t.Fatalf("expected Histogram[float64], got %T", m.Data)
	}
	if len(hist.DataPoints) != 1 {
		t.Fatalf("expected 1 data point, got %d", len(hist.DataPoints))
	}

	dp := hist.DataPoints[0]
	if !slices.Equal(dp.Bounds, []float64{300, 900}) {
		t.Fatalf("expected bounds [300 900], got %v", dp.Bounds)
	}
	if !slices.Equal(dp.BucketCounts, []uint64{0, 1, 0}) || dp.Count != 1 || dp.Sum != 600 {
		t.Fatalf("expected one value in the middle bucket, got counts %v count %d sum %v",
			dp.BucketCounts, dp.Count, dp.Sum)
	}

	value, _ := dp.Attributes.Value(attribute.Key("fulfillment"))
	if value.AsString() != "pickup" {
		t.Fatalf("expected fulfillment=pickup, got %q", value.AsString())
	}
}
