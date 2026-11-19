# Logs

How services write logs, what each log line carries, how logs reach Loki, and how to correlate them with
traces. Companion docs: [traces.md](traces.md), [metrics.md](metrics.md) and [alerts.md](alerts.md).

## Pipeline

```
logobs logger ──┬──> stdout (every service)
                └──> otelslog bridge ──OTLP/gRPC──> OTel Collector ──OTLP/HTTP──> Loki ──> Grafana
                     (only when telemetry.Init ran)                 /otlp
```

Every service writes to stdout. Every binary calls `telemetry.Init`, so each also ships the same records to
Loki. Loki is on `http://localhost:3100`; Grafana on
`http://localhost:3000` (Explore → Loki). Start both stacks as described in
[traces.md](traces.md#pipeline).

## Writing a log line

Every service uses `shared/logobs`:

```go
logger := logobs.NewLogger("identity-api")         // once per binary, in main.go
ctx = logobs.WithContext(ctx, logger)              // stored in the context (middleware or worker bootstrap)

logobs.FromContext(ctx).WarnContext(ctx, "failed to update user's last login", "error", err)
```

The rule: **when a `context.Context` is in scope, log with the `...Context(ctx, ...)` method and pass that
`ctx`.** The trace and span IDs come from the context passed to the call, not from the logger, so plain
`Info`/`Warn`/`Error`/`Debug` produces a line with no trace ID. Plain calls are only for code with no
context at all: `main()`, shutdown paths, panic recovery.

- **Pass the closest context.** A log written inside a child span carries that child's `span_id` if the
  child's context is the one passed.
- **In a Gin handler, pass `c.Request.Context()`, never the `*gin.Context`.** `*gin.Context` satisfies
  `context.Context`, but its `Value` doesn't reach the request's context unless Gin's
  `ContextWithFallback` is turned on, and it isn't here, so the span isn't found.
- `FromContext` returns `slog.Default()` when the context holds no logger, so a missing `WithContext` loses
  the service fields, not the line.

## What a log line carries

| Field | Source | Present on |
|---|---|---|
| `service`, `env` | `NewLogger` (`APP_ENV`) | every line |
| `request_id`, `method`, `path` | `logobs.Middleware`, per request | lines logged during an HTTP request |
| `trace_id`, `span_id` | the span in the context passed to the call | lines logged with a context that carries a valid span |
| `status`, `duration` | `logobs.Middleware` | the `http request completed` line, one per request |

`request_id` is a UUIDv7, falling back to `fallback-<unix nanoseconds>` if generation fails. Example
stdout line (dev, text format, wrapped for width):

```
time=2026-11-03T00:23:50.063Z level=INFO msg="http request completed" service=identity-api env=dev
request_id=01a19eca-721a-76d9-9fe4-5f166633b563 method=POST path=/auth/email/verify status=204
duration=20.678468ms trace_id=af6150c1942c2fb25d5870210663a4dd span_id=378d81e3b837c443
```

### Configuration

| Variable | Values | Effect |
|---|---|---|
| `APP_ENV` | `prod` / anything else | JSON on stdout for `prod`, text otherwise; also logged as `env` |
| `LOG_LEVEL` | `debug` / `info` / `warn` / `error` | minimum level for **both** stdout and OTLP; default `info` |

## How it works

`logobs.NewLogger` builds one `slog.Logger` over `slog.NewMultiHandler` with two branches:

- **stdout:** the text or JSON handler, wrapped in `logobs`'s trace handler. On each record it reads the
  span from the context passed to the call and adds `trace_id`/`span_id` as attributes.
- **OTLP:** the `otelslog` bridge (scope `pizza-marketplace/shared/logobs`), wrapped in a level gate,
  because the bridge has no minimum-level option of its own. The bridge hands records to the global
  `LoggerProvider` from `telemetry.Init`, and the SDK sets the record's native OTLP trace and span IDs from
  the context passed to the call. Without `telemetry.Init` the global provider is a no-op and only stdout
  receives the line.

`NewLogger` may run before `telemetry.Init`: the global provider forwards to the real one once it is set.
`logobs.Middleware` must be registered after `otelgin.Middleware`, so the request's span exists when
the middleware stores the logger and writes the completion line.

**Workers** store the logger once, in `bootstrap.App.Run`, and log with the worker loop's context, which
carries no span. Per-unit work gets its own context:

- **Consumers:** the shared consumer's `runOnce` (`shared/amqp`) derives a context per message holding the
  consumer span and a logger with `event_name` and `message_id`, and passes it to the handler. The handler's
  lines and the consumer's own retry warnings carry the trace ID.
- **Outbox relay:** each event is handled under its restored trace context, so its lines carry the trace
  ID of the request that raised it.

Lines from the loop itself (startup, reconnects, shutdown) have no trace ID.

## In Loki

The Collector sends logs to Loki's native OTLP endpoint (`http://loki:3100/otlp`).

- **One index label: `service_name`**, from the resource's `service.name`. Keep it that way: every extra
  label multiplies the number of streams.
- **Everything else is structured metadata**, queryable but not indexed: `trace_id`, `span_id` (from the
  native OTLP fields), `request_id`, `method`, `path`, `status`, `duration` (integer nanoseconds),
  `service`, `env`, `severity_text`, `scope_name`, plus `detected_level`, which Loki derives.

```logql
{service_name="identity-api"} | trace_id="af6150c1942c2fb25d5870210663a4dd"
{service_name="identity-api"} | request_id="01a19eca-721a-76d9-9fe4-5f166633b563"
{service_name=~"identity-.*"} | detected_level="error"
```

Loki `3.2.1` stores data on the `loki_data` volume with no retention configured, so logs are kept until
the volume is removed.

### Correlation with traces

- **Log → trace:** the Loki datasource's derived field matches the `trace_id` structured-metadata field
  (`matcherType: label`) and links it to Tempo.
- **Trace → logs:** the Tempo datasource runs `{service_name="<service.name>"} | trace_id="<trace ID>"`.

Both are provisioned in `observability/grafana/provisioning/datasources/datasources.yaml`, where template
placeholders are written `$${...}` so Grafana's provisioning doesn't replace them with empty environment
variables.

## Known gaps

- **No redaction.** Nothing strips emails, tokens or payment data from attributes. No log call passes
  such fields, but nothing prevents one.
- **GORM's own logger is separate.** It writes slow and failed queries to stdout with bound values (for
  example `WHERE email = '...'`), bypassing `logobs`, so those lines carry no service fields or trace ID
  and don't reach Loki.
- **Worker loop lines have no trace ID:** startup, reconnect and shutdown lines run outside any message.
- **Field names aren't standardised** across services beyond the ones `logobs` sets.
- **Lost lines on backend outages and in dev.** While `compose.observability.yaml` is down the SDK logs OTLP
  connection errors and drops records. Under air (dev) the API is killed rather than signalled, so its last
  buffered records aren't flushed; the prod image flushes on SIGTERM.
