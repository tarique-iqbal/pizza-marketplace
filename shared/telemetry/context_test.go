package telemetry

import (
	"context"
	"testing"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

func newTestTracer() (trace.Tracer, *tracetest.SpanRecorder) {
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.AlwaysSample())),
		sdktrace.WithSpanProcessor(recorder),
	)

	return provider.Tracer("test"), recorder
}

func TestTraceParent_RoundTripsSpanContext(t *testing.T) {
	tracer, _ := newTestTracer()
	ctx, span := tracer.Start(context.Background(), "request")
	defer span.End()

	traceParent := TraceParent(ctx)
	if traceParent == "" {
		t.Fatal("expected a traceparent for a context carrying a span")
	}

	restored := trace.SpanContextFromContext(WithTraceParent(context.Background(), traceParent))
	original := span.SpanContext()

	if restored.TraceID() != original.TraceID() || restored.SpanID() != original.SpanID() {
		t.Fatalf("restored %s/%s, want %s/%s",
			restored.TraceID(), restored.SpanID(), original.TraceID(), original.SpanID())
	}

	if !restored.IsRemote() {
		t.Fatal("expected the restored span context to be marked remote")
	}
}

func TestTraceParent_EmptyWithoutSpan(t *testing.T) {
	if got := TraceParent(context.Background()); got != "" {
		t.Fatalf("expected empty traceparent, got %q", got)
	}
}

func TestWithTraceParent_EmptyLeavesContextUnchanged(t *testing.T) {
	ctx := context.Background()

	if WithTraceParent(ctx, "") != ctx {
		t.Fatal("expected the same context back for an empty traceparent")
	}
}

func TestWithTraceParent_MalformedLeavesNoSpanContext(t *testing.T) {
	ctx := WithTraceParent(context.Background(), "not-a-traceparent")

	if trace.SpanContextFromContext(ctx).IsValid() {
		t.Fatal("expected no valid span context for a malformed traceparent")
	}
}

func TestTraceParent_UnsampledStaysUnsampled(t *testing.T) {
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: trace.TraceID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		SpanID:  trace.SpanID{1, 2, 3, 4, 5, 6, 7, 8},
	})

	traceParent := TraceParent(trace.ContextWithSpanContext(context.Background(), sc))
	if traceParent != "00-"+sc.TraceID().String()+"-"+sc.SpanID().String()+"-00" {
		t.Fatalf("expected an unsampled traceparent, got %q", traceParent)
	}

	tracer, recorder := newTestTracer()
	_, span := tracer.Start(WithTraceParent(context.Background(), traceParent), "publish")
	span.End()

	if span.SpanContext().IsSampled() {
		t.Fatal("expected a child of an unsampled traceparent to be unsampled")
	}

	if n := len(recorder.Ended()); n != 0 {
		t.Fatalf("expected no recorded spans, got %d", n)
	}
}

func TestWithoutTracing_ChildSpansAreNotRecorded(t *testing.T) {
	tracer, recorder := newTestTracer()

	_, span := tracer.Start(WithoutTracing(context.Background()), "poll")
	span.End()

	if span.SpanContext().IsSampled() {
		t.Fatal("expected a span under WithoutTracing to be unsampled")
	}

	if n := len(recorder.Ended()); n != 0 {
		t.Fatalf("expected no recorded spans, got %d", n)
	}
}

func TestLinkTraceParent_AddsLinkToCurrentSpan(t *testing.T) {
	tracer, recorder := newTestTracer()

	_, origin := tracer.Start(context.Background(), "checkout")
	origin.End()
	traceParent := TraceParent(trace.ContextWithSpan(context.Background(), origin))

	ctx, span := tracer.Start(context.Background(), "webhook")
	LinkTraceParent(ctx, traceParent)
	span.End()

	webhook := recorder.Ended()[1]
	if len(webhook.Links()) != 1 {
		t.Fatalf("expected 1 link, got %d", len(webhook.Links()))
	}

	linked, want := webhook.Links()[0].SpanContext, origin.SpanContext()
	if linked.TraceID() != want.TraceID() || linked.SpanID() != want.SpanID() {
		t.Fatalf("linked %s/%s, want %s/%s", linked.TraceID(), linked.SpanID(), want.TraceID(), want.SpanID())
	}

	if webhook.SpanContext().TraceID() == want.TraceID() {
		t.Fatal("expected the linking span to stay in its own trace")
	}
}

func TestLinkTraceParent_InvalidAddsNoLink(t *testing.T) {
	for _, traceParent := range []string{"", "not-a-traceparent"} {
		tracer, recorder := newTestTracer()

		ctx, span := tracer.Start(context.Background(), "webhook")
		LinkTraceParent(ctx, traceParent)
		span.End()

		if n := len(recorder.Ended()[0].Links()); n != 0 {
			t.Errorf("traceparent %q: expected no links, got %d", traceParent, n)
		}
	}
}
