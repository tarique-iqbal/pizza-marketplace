# Alerts

What fires, why each rule is shaped the way it is, and what to check when one fires. The queries build on
[metrics.md](metrics.md); [traces.md](traces.md) and [logs.md](logs.md) are where the investigation continues.

## Setup

Grafana-managed rules, provisioned from `observability/grafana/provisioning/alerting/`, one rule group per file,
into the "Pizza Marketplace" folder. Grafana loads them at startup, so a rule change needs a Grafana restart.
Every rule is evaluated once a minute: an instant Prometheus query plus a threshold, one alert instance per job,
method, queue or database. Firing alerts are listed in Grafana under Alerting → Alert rules
(`http://localhost:3000`).

| File | Rule | Fires when | For | No data | Severity |
|---|---|---|---|---|---|
| `api.yaml` | [API 5xx share](#api-5xx-share) | 5xx / all per API job, `[5m]`, > 1% | 5 m | OK | critical |
| `api.yaml` | [API p95 latency](#api-p95-latency) | p95 per API job, `[5m]`, > 500 ms | 5 m | OK | warning |
| `api.yaml` | [gRPC server errors](#grpc-server-errors) | server-side codes / all per payment-api method, `[5m]`, > 1% | 5 m | OK | critical |
| `rabbitmq.yaml` | [Queue with messages but no consumers](#queue-with-messages-but-no-consumers) | messages > 0 and consumers == 0, per queue, not `.dlq`/`.wait` | 5 m | OK | warning |
| `rabbitmq.yaml` | [Messages in a DLQ](#messages-in-a-dlq) | `.dlq` queue messages > 0, per queue | 1 m | OK | warning |
| `business.yaml` | [Payment failure share](#payment-failure-share) | failed / attempts, `[15m]`, at least 3 attempts, > 20% | 5 m | OK | critical |
| `postgres.yaml` | [Database unreachable](#database-unreachable) | `pg_up == 0`, per database | 2 m | Alerting | critical |

Rules shared by all of them:

- **No traffic isn't a fault.** Traffic-based rules treat no data as OK, and their ratios guard the denominator
  with `> 0`, so an idle API returns no series instead of `NaN`.
- **Summaries print the value only when there is one** (`{{ with $values.A }}…{{ end }}`); without the guard the
  template fails while the rule has no data.
- **Rates use `[5m]` or longer**, never `$__rate_interval`: pushed series get one sample a minute (see
  [Freshness](metrics.md#freshness)).

## Rules

### API 5xx share

Share of 5xx responses per API (`job=~".+-api"`), `/healthz/*` excluded, above 1% for 5 minutes. Errors fall back
to `total * 0`, so an API without errors reads 0.

When it fires:
- Service RED → the API's routes table shows which route returns the 5xx.
- Loki: `{service_name="<job>"} | detected_level="error"`, then follow the `trace_id` to Tempo for the failing
  span (often a database or gRPC call).
- If [Database unreachable](#database-unreachable) fires for the same service, start there: a stopped database
  makes every route that touches it return 500.

### API p95 latency

p95 request duration per API, `/healthz/*` excluded, above 500 ms for 5 minutes. A p95 of no requests is `NaN`
and is filtered out (`>= 0`).

When it fires:
- Service RED → routes table: which route is slow.
- Tempo: search the service for spans over 500 ms; the slow child span (a database query, a gRPC call, the
  geocoder) shows where the time goes.
- Postgres Overview → active backends: a lock or a long transaction holds queries back.

### gRPC server errors

Share of server-side status codes (`UNKNOWN`, `DEADLINE_EXCEEDED`, `UNIMPLEMENTED`, `INTERNAL`, `UNAVAILABLE`,
`DATA_LOSS`) per payment-api method, health checks excluded, above 1% for 5 minutes. `NOT_FOUND` and
`INVALID_ARGUMENT` are answers, not faults.

It exists alongside the 5xx rule because not every failed payment call reaches the HTTP side: order-service
returns 503 when checkout or the pre-cancel status check fails, but a failed `CancelPayment` is only logged and
the cancel still returns 200.

When it fires:
- Service RED → gRPC row: which method.
- Loki: `{service_name="payment-api"} | detected_level="error"`. `INTERNAL` from `CreatePayment` usually means
  Mollie rejected the call (key, webhook URL); `INTERNAL` on every method points at payment's database.

### Queue with messages but no consumers

A queue holding messages with zero consumers for 5 minutes. The 5 minutes cover a worker restart. DLQs and
`<service>_queue.wait` queues are left out: neither has consumers by design.

When it fires:
- The queue's worker isn't running or can't connect: check its container (`docker ps`, `docker logs
  pizza-<svc>-worker`). In dev it usually means the service's Compose profile isn't started.
- The messages wait in the queue and are processed once the worker is back.

### Messages in a DLQ

Any message in a `<service>_queue.dlq` for 1 minute. A consumer retries a failing message 3 times, 5 s apart
through `<service>_queue.wait`, then moves it into the DLQ, so one message there already means an event was not
handled.

When it fires:
- Service RED → workers row → failed messages by event: which event failed.
- Loki: `{service_name="<svc>-worker"} |= "message exceeded retry limit"` gives the `event_name` and the
  `trace_id` of the last attempt; the trace shows the error.
- RabbitMQ UI → Queues → the DLQ → Get messages shows the payload and headers; `x-death` records each pass
  through the wait queue.
- Fix the cause; the message stays in the DLQ until it is moved back or purged by hand. There's no replay tool.

### Payment failure share

```promql
F / (S + F) and round(S + F) >= 3

F = (sum(increase(payments_failed_total{reason!="canceled"}[15m])) or vector(0))
S = (sum(increase(payments_succeeded_total[15m])) or vector(0))
```

Above 20% for 5 minutes, one alert in total (summed across `job`).

- **`canceled` isn't a failure**: a customer abandoning the Mollie checkout. Same definition as the Business
  dashboard.
- **At least 3 attempts.** Fewer is no data, so one failure out of one can't fire.
- **Only the minimum is rounded.** `round()` stops `increase()`'s extrapolation (3 attempts reading 2.9) from
  skipping the minimum; the share itself stays unrounded, so numerator and denominator share the same
  extrapolation and it cancels. Rounding the denominator would read 3 failures as 107%.
- **First-sample effect:** the first failure with a new `reason` is missed by `increase()`, so the share reads low
  right after payment-service starts.

When it fires:
- Business Overview → Payment outcomes per hour, by reason: `expired` points at customers not finishing checkout, `failed`
  at Mollie or the payment method.
- Check the [gRPC rule](#grpc-server-errors) and payment-api logs for Mollie errors.

### Database unreachable

`pg_up == 0` per database for 2 minutes. `pg_up`, not `pg_exporter_last_scrape_error`: it means exactly "the
exporter couldn't connect", while a scrape error can come from elsewhere in the scrape.

No data is Alerting here: if postgres-exporter stops, every `pg_up` goes stale and all five databases show
"Alerting (NoData)", still labelled per database. A silent exporter means nothing watches the databases.

When it fires:
- One database: its container (`docker ps`, `docker logs pizza-pg-<svc>`). In dev, its profile isn't started.
- All five at once, as NoData: postgres-exporter is down (`docker logs pizza-postgres-exporter`).

## In dev

Some rules fire for stacks that aren't running, and that's correct:

- **Database unreachable** for every database whose Compose profile isn't started; every target is scraped
  regardless.
- **Queue with messages but no consumers** for a queue whose worker isn't running, once a message arrives.
- **Messages in a DLQ** until the message is handled, even after the cause is gone.

Start the profile (or handle the DLQ message) to clear them. When stopping a single service's stack, stop its
containers rather than the profile: `docker compose --profile <svc> stop` also stops the shared RabbitMQ and
Traefik.

## Notifications

No contact point is configured. The default policy routes to `grafana-default-email` and SMTP isn't set up, so
each notification fails and Grafana logs `SMTP not configured, check your grafana.ini config file's [smtp]
section`. Alerts are visible in Grafana only.

## Known gaps

- **Alerts aren't delivered anywhere.** A contact point (email or chat) and a notification policy are still to
  be added.
- **The payment failure rule hasn't fired for real.** Dev has no live Mollie payments; its logic was checked with
  `promtool test rules` on the rule's own expression.
- **No alerts on the observability stack itself.** If Prometheus or the Collector stops, the rules see no data,
  and most treat that as OK.
- **No DLQ replay.** Moving a message back to its queue is manual.
