package testutil

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

var (
	meterReaderOnce sync.Once
	meterReader     *sdkmetric.ManualReader
)

func installMeterReader() *sdkmetric.ManualReader {
	meterReaderOnce.Do(func() {
		meterReader = sdkmetric.NewManualReader()
		otel.SetMeterProvider(sdkmetric.NewMeterProvider(sdkmetric.WithReader(meterReader)))
	})

	return meterReader
}

func collectMetric(t *testing.T, name string) (metricdata.Metrics, bool) {
	t.Helper()

	var rm metricdata.ResourceMetrics
	require.NoError(t, installMeterReader().Collect(context.Background(), &rm))

	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == name {
				return m, true
			}
		}
	}

	return metricdata.Metrics{}, false
}

func CounterValue(t *testing.T, name string, labels ...attribute.KeyValue) int64 {
	t.Helper()

	m, ok := collectMetric(t, name)
	if !ok {
		return 0
	}

	sum, ok := m.Data.(metricdata.Sum[int64])
	require.True(t, ok, "metric %s is %T, not an int64 sum", name, m.Data)

	var total int64
	for _, dp := range sum.DataPoints {
		if hasLabels(dp.Attributes, labels) {
			total += dp.Value
		}
	}

	return total
}

func hasLabels(set attribute.Set, labels []attribute.KeyValue) bool {
	for _, label := range labels {
		value, ok := set.Value(label.Key)
		if !ok || value != label.Value {
			return false
		}
	}

	return true
}
