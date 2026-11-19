# Metrics

Where metrics come from, how they reach Prometheus, what they're named, and how to query them. Companion
docs: [traces.md](traces.md), [logs.md](logs.md) and [alerts.md](alerts.md).

## Pipeline

Prometheus receives metrics two ways:

```
pull (scrape, every 15 s)                             push (Prometheus remote write)
  rabbitmq:15692 ─────────┐                ┌─ OTel Collector <── OTLP ── Traefik, every <svc>-api and <svc>-worker
  postgres-exporter:9187 ─┼─> Prometheus <─┤
  prometheus (self) ──────┘                └─ Tempo metrics-generator (derived from spans)
```

Prometheus is on `http://localhost:9090`; Grafana on `http://localhost:3000` (Explore → Prometheus). Start
both stacks as described in [traces.md](traces.md#pipeline).

## Sources

| Source | Path | Metric families |
|---|---|---|
| RabbitMQ | scrape `rabbitmq:15692` (`rabbitmq_prometheus` plugin, enabled in `compose/rabbitmq-enabled-plugins`) | `rabbitmq_*`, totals across all queues |
| RabbitMQ, per queue | scrape `rabbitmq:15692/metrics/detailed`, job `rabbitmq-queues`, families `queue_coarse_metrics` and `queue_consumer_count` | `rabbitmq_detailed_queue_{messages,messages_ready,messages_unacked,consumers}` by `queue` |
| Postgres | scrape `postgres-exporter:9187/probe`, one target per database (`prometheuscommunity/postgres-exporter`, multi-target) | `pg_*`, job `postgres`, `instance` = `identity`, `restaurant`, `order`, `payment`, `customer` |
| Prometheus | scrape `localhost:9090` | `prometheus_*` |
| Traefik | OTLP → Collector → remote write | `traefik_entrypoint_*` (request count, duration, bytes), `traefik_config_*`, `http_server_*` with `job="traefik"` |
| `<svc>-api` | OTLP → Collector → remote write | `http_server_request_duration_seconds`, `http_server_{request,response}_body_size_bytes` (Gin), `go_sql_connections_*` (GORM pool, not search) |
| payment-api / order-api | OTLP → Collector → remote write | `rpc_server_call_duration_seconds` (payment, gRPC server), `rpc_client_*` (order, gRPC client) |
| `<svc>-worker` | OTLP → Collector → remote write | `go_sql_connections_*` (not search or notification) |
| order-api / order-worker, payment-api | OTLP → Collector → remote write | business metrics, see [below](#business-metrics) |
| Tempo metrics-generator | remote write | `traces_spanmetrics_*`, `traces_service_graph_*` |

Each binary is its own `job`. search-worker and notification-worker call `telemetry.Init` but have no
instrumented library that records metrics, so they appear only through the span-derived metrics below.

Span-derived metrics are grouped by span name, and messaging span names include the destination, so
they break down per event type, for example
`traces_spanmetrics_calls_total{service="notification-worker", span_name="process identity.events:email.verification_created:notification_queue", status_code="STATUS_CODE_ERROR"}`.

## Names and labels

OTel metric names are converted on the way into Prometheus: dots become underscores, the unit becomes a
suffix and counters get `_total`. So `http.server.request.duration` (seconds) is stored as
`http_server_request_duration_seconds`.

The service is identified by the **`job`** label. The Collector's remote-write exporter maps
`service.name` to `job`, unlike Loki, where the same attribute becomes `service_name`. Other resource
attributes land on a separate `target_info` series per service.

Gin's HTTP metrics carry `http_route` (the route template), `http_request_method` and
`http_response_status_code`. They never carry user or restaurant IDs, so the series count stays bounded
by the number of routes.

## Freshness

OTLP metrics are exported by the SDK's periodic reader every 60 seconds, so a request shows up in
Prometheus up to about a minute later. Scraped metrics update every 15 s.

Because pushed series have one sample a minute, every `rate()`/`increase()` on them uses at least a `[5m]` window;
Grafana's `$__rate_interval` would often hold a single sample and show no data.

A series appears only when its first value is exported, so `rate()` and `increase()` have no earlier sample to
compare it with and miss whatever it counted first. A route, event or business counter seen for the first time
since its service started reads low (or 0) until the next export. `increase()` also extrapolates to the window
edges, so counts over a range are approximate.

## Querying

```promql
# p95 latency per route, identity-api
histogram_quantile(0.95,
  sum by (le, http_route) (rate(http_server_request_duration_seconds_bucket{job="identity-api"}[5m])))

# 5xx share of requests through the gateway
sum(rate(traefik_entrypoint_requests_total{code=~"5.."}[5m]))
  / sum(rate(traefik_entrypoint_requests_total[5m]))

# requests per second per span name, derived from traces
sum by (service, span_name) (rate(traces_spanmetrics_calls_total[5m]))

# DB connections currently in use
go_sql_connections_in_use{job=~"identity-.*"}
```

## Dashboards

Provisioned from files in `observability/grafana/provisioning/dashboards/`, picked up without a Grafana
restart:

- **Service RED** (`service-red.json`): an `API` selector, then requests/s, 5xx share and p95 per API with
  1% and 500 ms threshold lines, and a per-route table over the dashboard range. A gRPC row for payment-api
  (calls/s, server-error share, p95 per method) and a worker row from span metrics (messages/s, error share, p95
  per worker, failed messages by event).
- **Traefik Overview** (`traefik-overview.json`): gateway requests/s, 5xx share, p95 and open connections for the
  `web` entrypoint, requests by status class, p50/p95/p99, and requests/s and p95 per Traefik service.
- **Business Overview** (`business-overview.json`): the [business metrics](#business-metrics): order and payment
  counts over the range, cancellations by source, placed by fulfillment, p50/p90 completion time, payment
  failures by reason and the failure share.
- **RabbitMQ Overview** (`rabbitmq-overview.json`): connections, channels, queues, consumers, messages
  ready vs. total, then per queue: DLQ messages, queues with messages but no consumers, and a ready / unacked /
  consumers table.
- **Postgres Overview** (`postgres-overview.json`): a `Database` selector, then up, max connections, active
  backends, database size and commit rate per database.

Dashboard queries follow these rules:

- **HTTP panels exclude `/healthz/*`.** Container health probes hit those routes and would otherwise dominate
  the request rate.
- **A gRPC error is a server-side code**: `UNKNOWN`, `DEADLINE_EXCEEDED`, `UNIMPLEMENTED`, `INTERNAL`,
  `UNAVAILABLE`, `DATA_LOSS`. `NOT_FOUND` and `INVALID_ARGUMENT` are answers, the gRPC counterpart of a 4xx.
- **Error shares fall back to 0.** `errors or total * 0`, divided by the total, so a service without errors
  reads 0 rather than no data.
- **Payment failure share leaves out `reason="canceled"`**, a customer abandoning the Mollie checkout. It shows
  no data while there are no payments.

## Alerts

Grafana alert rules on these metrics (API 5xx and latency, gRPC errors, queues, DLQs, payment failures,
database reachability) are described in [alerts.md](alerts.md).

## Postgres: one exporter, five databases

postgres-exporter runs in multi-target mode. Prometheus's `postgres` job lists each database as a target
(`postgres-order:5432/order_db`), passes it as `?target=` to `/probe` with `auth_module=dev`, and sends every
probe to the one exporter. The `dev` auth module in `observability/postgres-exporter-config.yaml` holds the shared
dev credentials. Relabelling sets `instance` to the service name. The exporter has no default database, so it
logs `empty dsn` once at startup; that's expected.

**Up/down.** Every probe emits `pg_up` (`1` when the exporter connected to the database, `0` when it didn't) and
`pg_exporter_last_scrape_error` (`0` / `1`, the inverse). The dashboard's Up panel shows
`1 - pg_exporter_last_scrape_error`; the [alert](alerts.md#database-unreachable) uses `pg_up`, which means
exactly "couldn't connect" rather than "something in the scrape failed". Every target is scraped whether or not
its service's Compose profile is running, so a database that isn't started shows as down.

```promql
# databases the exporter can't reach
pg_up{job="postgres"} == 0
```

## RabbitMQ: totals and per queue

The plugin's default `/metrics` endpoint aggregates across queues: `rabbitmq_queue_messages_ready` is one series
with no `queue` label. Per-queue values come from `/metrics/detailed`, scraped as the `rabbitmq-queues` job with
only the two families the dashboards need, which keeps the series count at a few per queue.

```promql
# messages sitting in dead-letter queues
sum(rabbitmq_detailed_queue_messages{queue=~".+[.]dlq"})

# queues holding messages that nothing consumes (DLQs never have consumers)
(rabbitmq_detailed_queue_messages{queue!~".+[.]dlq"} > 0)
  and on (queue) (rabbitmq_detailed_queue_consumers == 0)
```

A queue whose worker isn't running (its Compose profile not started) shows up as having no consumers.

## Business metrics

Recorded in the application layer through `telemetry.NewCounter`/`NewHistogram`, only after the state change has
committed. A duplicate delivery (a redelivered RabbitMQ event, a Mollie webhook retry) is a no-op and counts
nothing.

| Prometheus name | Recorded in | Labels |
|---|---|---|
| `orders_placed_total` | order-api, `Checkout` after its transaction | `fulfillment` |
| `orders_confirmed_total` | order-worker, `payment.succeeded` handler | `fulfillment` |
| `orders_cancelled_total` | order-api (`Cancel`), order-worker (`payment.failed` handler) | `source`: `customer`, `owner`, `payment_failed` |
| `orders_completed_total` | order-api, `Complete` | `fulfillment` |
| `order_completion_duration_seconds_*` | order-api, `Complete`: completed minus placed | `fulfillment`; buckets 5, 10, 15, 20, 30, 45, 60, 90, 120 min |
| `payments_succeeded_total` | payment-api, Mollie webhook | none |
| `payments_failed_total` | payment-api, Mollie webhook | `reason`: `canceled`, `expired`, `failed`, `other` |

`fulfillment` is `delivery` or `pickup`. No label carries an order, user or restaurant ID.

The same counter can come from two binaries (`orders_cancelled_total` from order-api and order-worker), so `job`
differs per source. Aggregate away `job` when charting totals:

```promql
# cancellations per minute by source
sum by (source) (rate(orders_cancelled_total[5m])) * 60

# median time from placed to completed
histogram_quantile(0.5, sum by (le) (rate(order_completion_duration_seconds_bucket[1h])))

# payment failure share, abandoned checkouts excluded
sum(rate(payments_failed_total{reason!="canceled"}[15m]))
  / (sum(rate(payments_succeeded_total[15m])) + sum(rate(payments_failed_total{reason!="canceled"}[15m])))
```

A counter appears in Prometheus only after it is first incremented, so a freshly started service has no series
for it yet.

## Storage

Prometheus `v2.55.1`, started with `--web.enable-remote-write-receiver` so the Collector and Tempo can push.
No retention flag is set, so Prometheus's default of 15 days applies, on the `prometheus_data` volume.

## Known gaps

- **Exemplars are configured but inert.** Tempo sends exemplars with its remote write, and the Grafana
  Prometheus datasource maps `trace_id` exemplars to Tempo, but Prometheus runs without
  `--enable-feature=exemplar-storage`, so none are stored; `/api/v1/query_exemplars` returns nothing.
- **`stat_bgwriter` fails on Postgres 17.** Exporter `v0.15.0` queries checkpoint columns that Postgres 17
  moved to `pg_stat_checkpointer`, so that one collector fails on every probe (`pg_scrape_collector_success
  {collector="stat_bgwriter"} 0`, an error line in the exporter log). The other collectors and
  `pg_exporter_last_scrape_error` are unaffected.
- **The observability stack doesn't monitor itself.** Collector, Tempo and Loki internal metrics aren't
  scraped.
