# Traces

How distributed traces are produced, carried between components, stored, and read in this repo. Companion
docs: [logs.md](logs.md), [metrics.md](metrics.md) and [alerts.md](alerts.md).

## Pipeline

```
Traefik ──OTLP/HTTP :4318──┐
                           ├──> OTel Collector ──OTLP/gRPC──> Tempo ──> Grafana (Explore → Tempo)
service ──OTLP/gRPC :4317──┘                                    │
                                                                └─ metrics-generator ──> Prometheus
```

Services only ever talk to the Collector (`otel-collector`), never to Tempo directly. The backend runs as
its own Compose project:

```bash
docker compose up -d                                   # app stack (creates the shared networks)
docker compose -f compose.observability.yaml up -d     # Collector, Tempo, Prometheus, Loki, Grafana
```

Grafana is on `http://localhost:3000` (anonymous admin), Tempo's API on `http://localhost:3200`.

## What a trace looks like

One request through the gateway is one trace, including the events it raises. `POST /auth/email/verify`
produces:

```
traefik              SERVER    POST                                         (root)
traefik              CLIENT    ReverseProxy
identity-api         SERVER    POST /auth/email/verify
identity-api         CLIENT    select users
identity-api         CLIENT    select email_verifications
identity-api         CLIENT    insert email_verifications
identity-api         CLIENT    insert outbox_events
identity-worker      PRODUCER  send identity.events:email.verification_created
notification-worker  CONSUMER  process identity.events:email.verification_created:email_queue
identity-worker      CLIENT    update outbox_events
```

Every span's parent is in the same trace. The server span is named after the Gin route template, so
`/users/:id` stays one span name regardless of the ID. The producer span and the worker's
`update outbox_events` are children of the request span; the consumer span is a child of the producer
span. The gap between `insert outbox_events` and the producer span is the outbox poll delay.

## Coverage

| Producer | Spans | Status |
|---|---|---|
| Traefik | entrypoint `SERVER` + `ReverseProxy` `CLIENT`, every routed request | wired (`compose/base.yaml`) |
| `<svc>-api` (identity, restaurant, order, payment, customer, search) | Gin server span per request, one span per GORM query | wired |
| `<svc>-worker` (identity, restaurant, order, payment, customer, search) | `PRODUCER` span per relayed outbox event (all but search, which has no outbox) and `CONSUMER` span per consumed message (all but payment, which consumes nothing), each with its queries, inside the originating request's trace | wired |
| notification-worker | `CONSUMER` span per consumed message, inside the publisher's trace | wired |
| order → payment gRPC | `CLIENT` span in order, `SERVER` span in payment-api, same trace | wired |
| search | one `CLIENT` span per Elasticsearch request (`db.system=elasticsearch`, index, operation), request bodies not captured | wired |

## How it works

### `telemetry.Init`

`shared/telemetry.Init(ctx, telemetry.Config{ServiceName: ...})` runs once per process, from `cmd/api/main.go`
and `cmd/worker/main.go`, with a distinct name per binary (`<svc>-api`, `<svc>-worker`). It:

- builds the OTel resource from `OTEL_SERVICE_NAME`/`OTEL_RESOURCE_ATTRIBUTES`, host and SDK detectors,
  then `service.name` from `Config.ServiceName`, which wins over the env var;
- creates three OTLP/gRPC exporters: traces behind a batch span processor, metrics behind a periodic reader
  (every 60 s), logs behind a batch processor;
- installs the global `TracerProvider`, `MeterProvider` and `LoggerProvider`, and the W3C `TraceContext`
  propagator;
- returns a `Shutdown` that flushes traces, metrics and logs.

Exporter configuration comes from standard env vars, read by the SDK itself:

| Variable | Value in every service | Effect |
|---|---|---|
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `http://otel-collector:4317` | Collector address; the `http` scheme means plaintext |
| `OTEL_TRACES_SAMPLER` / `OTEL_TRACES_SAMPLER_ARG` | unset | unset = `ParentBased(AlwaysSample)` |

Because the sampler is parent-based, the decision Traefik makes for the root span is honoured by every
downstream service. Set `OTEL_TRACES_SAMPLER=parentbased_traceidratio` and `OTEL_TRACES_SAMPLER_ARG=0.1` to
keep 10% of new traces.

### Propagation

Traefik injects a `traceparent` header into the proxied request. `otelgin.Middleware` extracts it
with the global propagator, so identity-api's server span becomes a child of Traefik's `ReverseProxy`
span. Anything that receives a `context.Context` from the request carries that span forward.

Events cross two asynchronous hops, and the trace context crosses both:

1. **Outbox row.** The outbox repository's `Create` stores `telemetry.TraceParent(ctx)` in
   `outbox_events.trace_context` (`VARCHAR(128)`, nullable), in the same transaction as the business write.
   Call sites and the domain layer don't change.
2. **Relay.** The worker restores that value per event (`telemetry.WithTraceParent`), so the publish and the
   worker's `update outbox_events` join the request's trace. The poll query runs under
   `telemetry.WithoutTracing`, an unsampled parent the `ParentBased` sampler drops, so polling emits no
   spans.
3. **Publish.** `otelamqp.StartPublish` starts a `PRODUCER` span and injects its context into the AMQP
   headers as `traceparent`.
4. **Consume.** Each consumer's `runOnce` calls `otelamqp.StartConsume`, which extracts `traceparent` and
   starts a `CONSUMER` span as its child. The handler runs with that context, through
   `Dispatch(ctx, event)` → `Handle(ctx, event)`, so its queries and any events it raises into its own
   outbox join the same trace. The span ends when the handler returns, with error status on failure.

Span names and attributes follow the OpenTelemetry messaging conventions (semconv v1.43.0):
`messaging.destination.name` is `{exchange}:{routing key}` on the producer and
`{exchange}:{routing key}:{queue}` on the consumer, and the span name is `{operation} {destination}`. Routing
keys are fixed strings in code, so span names stay low-cardinality. The conventions default to span links
between producer and consumer and allow a parent link for single-message processing, which is the case here.

A retry is republished through the default exchange, so its delivery has no exchange and the queue name as
routing key. `Republish` keeps the original values in `x-original-exchange`/`x-original-routing-key`, and
each attempt's consumer span carries the original destination, as a sibling of the first attempt under the
same producer span. A message that exhausts its retries ends in the queue's DLQ with its headers, including
`traceparent`.

### The Mollie webhook: a span link, not a parent

Mollie reports a payment's outcome by calling `POST /webhooks/mollie`, a new inbound request with no trace
context. payment-service's repository stores `telemetry.TraceParent(ctx)` in `payments.trace_context`
(`VARCHAR(128)`) when the payment is created, inside order's checkout trace. `HandleMollieWebhook` reads it back and
calls `telemetry.LinkTraceParent`, so the webhook's server span carries a **link** to the checkout. It stays its own
trace, because it is a separate request that can arrive much later or more than once. The `payment.succeeded`/
`payment.failed` outbox row is written under the webhook's context, so everything downstream (order's consumer,
`order.confirmed`, notification) continues the webhook's trace.

### Instrumentation helpers (`shared/telemetry/...`)

Each instrumentation is its own subpackage, so a service imports only what it uses. `shared/telemetry`
itself holds `Init`/`Shutdown` and depends on none of Gin, GORM or the AMQP client.

| Package | Helper | Wraps | Use |
|---|---|---|---|
| `telemetry/otelgin` | `Middleware(serviceName)` | contrib `otelgin` | `router.Use(...)`, registered **before** `logobs.Middleware` |
| `telemetry/otelgrpc` | `ClientOption()` / `ServerOption()` | contrib `otelgrpc` stats handlers | `grpc.NewClient(..., opt)` / `grpc.NewServer(opt)` |
| `telemetry/otelgorm` | `Plugin()` | `gorm.io/plugin/opentelemetry/tracing`, `db.system=postgresql` | `db.Use(otelgorm.Plugin())` after `gorm.Open` |
| `telemetry/otelelasticsearch` | `Instrumentation()` | go-elasticsearch's built-in OTel instrumentation, search bodies not captured | `elasticsearch.Config{Instrumentation: ...}` |
| `telemetry/otelamqp` | `HeaderCarrier` | `propagation.TextMapCarrier` over `amqp.Table` | injecting/extracting trace context in message headers |
| `telemetry/otelamqp` | `StartPublish(ctx, exchange, routingKey, headers)` | producer span + header injection | before `PublishWithContext`, ended with `telemetry.EndSpan` |
| `telemetry/otelamqp` | `StartConsume(ctx, queue, exchange, routingKey, msg)` | header extraction + consumer span | per delivered message, ended with `telemetry.EndSpan` |
| `telemetry` | `TraceParent` / `WithTraceParent` | W3C `traceparent` string ↔ context | storing and restoring context on outbox rows |
| `telemetry` | `WithoutTracing(ctx)` | unsampled parent context | background queries that shouldn't produce traces |
| `telemetry` | `LinkTraceParent(ctx, traceParent)` | adds a span link from the current span to a stored context | joining a separate request to the trace that caused it |
| `telemetry` | `EndSpan(span, err)` | records the error and status, then ends the span | `defer func() { telemetry.EndSpan(span, err) }()` |

GORM spans attach to whatever context the query runs with. A repository that skips
`db.WithContext(ctx)` produces a span with no parent, which shows up as its own root trace instead of
inside the request.

### Collector and Tempo

- **Collector** (`observability/otel-collector-config.yaml`): OTLP receiver on `:4317`/`:4318`,
  `memory_limiter` (200 MiB, 50 MiB spike), `batch` (5 s or 1024 items), exported to Tempo over plaintext
  gRPC.
- **Tempo** (`observability/tempo-config.yaml`): local filesystem storage on the `tempo_data` volume, blocks
  cut every 5 minutes, **24-hour retention**. The metrics-generator derives `traces_spanmetrics_*` (calls,
  latency, size per span name) and `traces_service_graph_*` (edges between services) and remote-writes
  them to Prometheus. See [metrics.md](metrics.md).

### Grafana

The Tempo datasource (`uid: tempo`) links each span to its logs with the LogQL query
`{service_name="<span's service.name>"} | trace_id="<trace ID>"`, and uses Prometheus for the service map.
In `observability/grafana/provisioning/datasources/datasources.yaml` the template placeholders are
written `$${...}`, because Grafana's provisioning replaces a bare `${...}` with an environment variable,
which is empty.

## Wiring a service

1. `cmd/api/main.go` and `cmd/worker/main.go`: call `telemetry.Init` with a per-binary service name and
   flush with a bounded context on exit. In a worker, run the flush before `os.Exit`, which skips defers. An API
   must stop on SIGTERM through `http.Server.Shutdown`, not `router.Run`, or the deferred flush never runs.
2. API router: `router.Use(gin.Recovery(), otelgin.Middleware(name), logobs.Middleware(logger))`.
3. Database: `db.Use(otelgorm.Plugin())` where the `*gorm.DB` is opened, and check that every
   repository method calls `db.WithContext(ctx)`.
4. gRPC: add `otelgrpc.ClientOption()` on the client and `otelgrpc.ServerOption()` on the server.
   Elasticsearch: set `Instrumentation: otelelasticsearch.Instrumentation()` in the client config.
5. Env: add `OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector:4317` to `.env` and `.env.example`.


## Known behaviour and gaps

- **Traefik traces its own Docker API polling.** Its Docker provider calls show up as root traces named
  `GET /v1.54/containers/<id>/json`.
- **Dev containers don't flush on stop.** air kills the API process rather than signalling it, so the flush on
  SIGTERM only runs where the binary is the container's process (the prod image).
- **Mollie API calls aren't traced.** payment-service's Mollie gateway uses a plain `http.Client`, so creating,
  cancelling or fetching a payment shows no `CLIENT` span; its time and errors appear only on the gRPC server
  span or the webhook span around it.
- **Readiness probes are traced.** Each probe is a Gin span, plus an Elasticsearch `ping` span in search; under
  Kubernetes probing this adds a trace every few seconds per pod.
- **Exporter errors when the backend is down.** With `compose.observability.yaml` stopped, the SDK logs
  OTLP connection errors; spans are dropped, the service keeps working.
