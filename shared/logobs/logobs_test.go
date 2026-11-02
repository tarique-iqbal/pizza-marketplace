package logobs

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"go.opentelemetry.io/otel/trace"
)

func spanContext() trace.SpanContext {
	return trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    trace.TraceID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		SpanID:     trace.SpanID{1, 2, 3, 4, 5, 6, 7, 8},
		TraceFlags: trace.FlagsSampled,
	})
}

func decode(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()

	var record map[string]any
	if err := json.Unmarshal(buf.Bytes(), &record); err != nil {
		t.Fatalf("failed to parse log line: %v", err)
	}

	return record
}

func TestWithContext_AddsTraceAndSpanID(t *testing.T) {
	var buf bytes.Buffer
	base := slog.New(slog.NewJSONHandler(&buf, nil))

	sc := spanContext()
	ctx := trace.ContextWithSpanContext(context.Background(), sc)

	FromContext(WithContext(ctx, base)).Info("with span")

	record := decode(t, &buf)

	if record["trace_id"] != sc.TraceID().String() {
		t.Fatalf("expected trace_id %q, got %v", sc.TraceID().String(), record["trace_id"])
	}

	if record["span_id"] != sc.SpanID().String() {
		t.Fatalf("expected span_id %q, got %v", sc.SpanID().String(), record["span_id"])
	}
}

func TestWithContext_NoSpanInContext(t *testing.T) {
	var buf bytes.Buffer
	base := slog.New(slog.NewJSONHandler(&buf, nil))

	FromContext(WithContext(context.Background(), base)).Info("no span")

	if _, ok := decode(t, &buf)["trace_id"]; ok {
		t.Fatal("expected no trace_id when context has no active span")
	}
}

func TestLevelHandler_DropsRecordsBelowLevel(t *testing.T) {
	var buf bytes.Buffer
	inner := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	logger := slog.New(&levelHandler{Handler: inner, level: slog.LevelInfo})

	logger.Debug("dropped")

	if buf.Len() != 0 {
		t.Fatalf("expected debug record to be dropped, got %q", buf.String())
	}

	logger.With("k", "v").Info("kept")

	if decode(t, &buf)["k"] != "v" {
		t.Fatal("expected info record with attrs to pass through after With")
	}
}
