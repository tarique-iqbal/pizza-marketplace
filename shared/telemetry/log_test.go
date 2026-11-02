package telemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"go.opentelemetry.io/otel/trace"
)

func TestSlogHandler_AddsTraceAndSpanID(t *testing.T) {
	var buf bytes.Buffer
	handler := NewSlogHandler(slog.NewJSONHandler(&buf, nil))
	logger := slog.New(handler)

	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    trace.TraceID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		SpanID:     trace.SpanID{1, 2, 3, 4, 5, 6, 7, 8},
		TraceFlags: trace.FlagsSampled,
	})
	ctx := trace.ContextWithSpanContext(context.Background(), sc)

	logger.InfoContext(ctx, "test message")

	var record map[string]any
	if err := json.Unmarshal(buf.Bytes(), &record); err != nil {
		t.Fatalf("failed to parse log line: %v", err)
	}

	if record["trace_id"] != sc.TraceID().String() {
		t.Fatalf("expected trace_id %q, got %v", sc.TraceID().String(), record["trace_id"])
	}

	if record["span_id"] != sc.SpanID().String() {
		t.Fatalf("expected span_id %q, got %v", sc.SpanID().String(), record["span_id"])
	}
}

func TestSlogHandler_NoSpanInContext(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(NewSlogHandler(slog.NewJSONHandler(&buf, nil)))

	logger.InfoContext(context.Background(), "no span here")

	var record map[string]any
	if err := json.Unmarshal(buf.Bytes(), &record); err != nil {
		t.Fatalf("failed to parse log line: %v", err)
	}

	if _, ok := record["trace_id"]; ok {
		t.Fatal("expected no trace_id when context has no active span")
	}
}

func TestSlogHandler_WithAttrsPreservesInjection(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(NewSlogHandler(slog.NewJSONHandler(&buf, nil))).With("service", "test")

	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    trace.TraceID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		SpanID:     trace.SpanID{1, 2, 3, 4, 5, 6, 7, 8},
		TraceFlags: trace.FlagsSampled,
	})
	ctx := trace.ContextWithSpanContext(context.Background(), sc)

	logger.InfoContext(ctx, "after With")

	var record map[string]any
	if err := json.Unmarshal(buf.Bytes(), &record); err != nil {
		t.Fatalf("failed to parse log line: %v", err)
	}

	if record["trace_id"] != sc.TraceID().String() {
		t.Fatal("expected trace_id to survive a .With(...) call")
	}

	if record["service"] != "test" {
		t.Fatal("expected the attribute from .With(...) to still be present")
	}
}
