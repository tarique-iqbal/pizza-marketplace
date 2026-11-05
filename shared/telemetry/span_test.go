package telemetry

import (
	"context"
	"errors"
	"testing"

	"go.opentelemetry.io/otel/codes"
)

func TestEndSpan_WithError_RecordsErrorStatus(t *testing.T) {
	tracer, recorder := newTestTracer()
	_, span := tracer.Start(context.Background(), "publish")

	EndSpan(span, errors.New("broker unavailable"))

	ended := recorder.Ended()
	if len(ended) != 1 {
		t.Fatalf("expected 1 ended span, got %d", len(ended))
	}

	status := ended[0].Status()
	if status.Code != codes.Error || status.Description != "broker unavailable" {
		t.Fatalf("expected error status with the error message, got %v %q", status.Code, status.Description)
	}

	events := ended[0].Events()
	if len(events) != 1 || events[0].Name != "exception" {
		t.Fatalf("expected one exception event, got %v", events)
	}
}

func TestEndSpan_WithoutError_LeavesStatusUnset(t *testing.T) {
	tracer, recorder := newTestTracer()
	_, span := tracer.Start(context.Background(), "publish")

	EndSpan(span, nil)

	ended := recorder.Ended()
	if len(ended) != 1 {
		t.Fatalf("expected 1 ended span, got %d", len(ended))
	}

	if code := ended[0].Status().Code; code != codes.Unset {
		t.Fatalf("expected unset status, got %v", code)
	}

	if events := ended[0].Events(); len(events) != 0 {
		t.Fatalf("expected no events, got %v", events)
	}
}
