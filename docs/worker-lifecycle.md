# Worker lifecycle — `cmd/worker/`

This document covers the design shared by every service's worker-shaped process in this repo:
`identity-worker`, `restaurant-worker`, `order-worker`, `payment-worker`, `customer-worker`,
`search-worker`, and `notification-service` (a worker-only service with no `cmd/api` sibling, but the
same internal shape). All seven follow the identical `cmd/worker/bootstrap/` pattern. This doc explains
how that pattern actually behaves, including several non-obvious edge cases, so a future change to any
one service's worker can be made with the same assumptions the others already rely on.

## Scope

Applies to every `cmd/worker/main.go` + `cmd/worker/bootstrap/{app.go,runner.go,health_server.go}` in the
repo. It does **not** cover each service's `cmd/api` (a separate, much simpler Gin HTTP server with no
goroutine orchestration of its own), and it does not cover the *content* of what each worker's goroutines
do (consuming RabbitMQ events, relaying the outbox) — see each service's own `docs/services/<name>.md`
and `CLAUDE.md` for that.

## The three files and what each owns

```
cmd/worker/main.go              → thin entrypoint: build a logger, call bootstrap.NewApp(logger).Run(),
                                   os.Exit(1) on error. No orchestration logic lives here.
cmd/worker/bootstrap/app.go      → App: process-level concerns — constructs the WorkerContainer, wires up
                                   OS signal handling, decides how long to wait for a clean shutdown.
cmd/worker/bootstrap/runner.go   → runner: goroutine-level concerns — starts each of the worker's long-
                                   running goroutines, tracks them with a WaitGroup, recovers panics.
cmd/worker/bootstrap/health_server.go → the one piece of actual HTTP serving logic: builds the gin router,
                                   runs it, and ties its own shutdown to the same ctx everything else uses.
```

This three-way split exists so each file answers one question only: *should the process still be
running* (`app.go`), *what is it doing while it runs* (`runner.go`), and *how does the health check
actually get served* (`health_server.go`). Nothing in `runner.go` knows how gin or `http.Server` work;
nothing in `health_server.go` knows about signals or the other goroutines.

## Startup sequence

1. `main()` builds a `*slog.Logger` tagged with the service's own name (e.g. `"restaurant-worker"`) and
   calls `bootstrap.NewApp(logger).Run()`.
2. `App.Run()` calls `container.NewWorkerContainer(logger)` (or `NewContainer()` for notification-service).
   This is where every real dependency gets dialed: Postgres, RabbitMQ (publisher and/or consumer),
   Elasticsearch for search-service. **If this fails for any reason — e.g. RabbitMQ unreachable — `Run()`
   returns the error immediately, before a single goroutine has started.** `main()` logs it and calls
   `os.Exit(1)`. No health server is ever listening in this failure mode; see "Startup failure is invisible
   to the health probe" below.
3. On success, `defer app.Close()` is registered (runs when `Run()` itself returns, not before).
4. `signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)` builds a `ctx` that
   becomes `Done()` the moment either signal arrives, plus a `stop()` function that cancels it manually.
   `stop()` is also how a failing goroutine triggers a *coordinated* shutdown of every other goroutine —
   see "Panic and error handling" below.
5. `ctx` is wrapped with the logger (`logobs.WithContext`) so every goroutine's log lines carry the same
   structured fields without having to thread the logger through every function call.
6. `runner.start(ctx, stop)` is called. **This call does not block** — it only launches goroutines and
   returns immediately. Everything from here on runs concurrently.
7. `App.Run()` blocks on `<-ctx.Done()`.

## Steady-state: what `runner.start` actually launches

Every goroutine `runner.start` launches is wrapped identically:

```go
r.wg.Add(1)
go func() {
    defer r.wg.Done()
    defer r.recoverPanic(stop)

    // the actual work
}()
```

Which goroutines exist depends on the service — see the table further down — but every service has at
least one "business" goroutine plus the health-server goroutine:

- **Consumer goroutine** (`messaging.Run(ctx, consumer, dispatcher)`) — present for every service with
  inbound events (restaurant, order, customer, search, notification; *not* identity or payment, which
  consume nothing). Its result is checked:
  ```go
  if err := messaging.Run(ctx, r.app.Consumer, r.app.Dispatcher); err != nil &&
      !errors.Is(err, context.Canceled) {
      r.logger.Error("consumer stopped unexpectedly", "error", err)
      stop()
  }
  ```
  **This branch is unreachable in every service, by design.** `messaging.Run`'s own implementation
  (independent copies in each service, not shared code) never returns a non-nil, non-cancellation error:
  any transient broker failure inside its loop is logged and retried after a 1-second backoff, forever,
  until `ctx` is cancelled. The branch exists for a failure mode the current implementation doesn't have —
  a future change to `Run()` that gives up after N retries would need exactly this branch to shut the
  worker down cleanly, but as the code stands, it cannot execute.
  **One inconsistency**: restaurant-service's copy of `Run` returns `ctx.Err()` on both exit paths
  instead of `nil` like the other four. This happens to be harmless — `ctx.Err()` is guaranteed to equal
  `context.Canceled` at that point, so `errors.Is(err, context.Canceled)` is still true and the branch above
  is still skipped — but it's a real, if cosmetic, drift between "independent copies, not shared code."
- **Outbox worker goroutine** (`outbox.Worker.Start(ctx)`) — present for every service with an outbox
  (identity, restaurant, order, payment, customer; *not* search or notification, which never publish).
  Returns nothing — there's no error to check. Internally it runs its own ticker loop and, on each tick,
  spawns a *further* batch-processing goroutine tracked by the `Worker`'s own private `WaitGroup`
  (`activeWg`), entirely invisible to `runner.go`. This nested concurrency matters for shutdown — see
  below.
- **Health server goroutine** (`runHealthServer(ctx, handler)`) — present in every service. Builds a gin
  router with `/healthz/liveness`/`/healthz/readiness`, runs `http.Server.ListenAndServe` in its own
  sub-goroutine so it can simultaneously watch for `ctx.Done()`, and on cancellation calls `Shutdown` with
  its own 5-second timeout before returning `ctx.Err()`. Unlike the consumer branch above, **this error
  path is genuinely reachable**: if `ListenAndServe` fails immediately (e.g. port `:8081` already bound),
  that real error comes back, gets logged, and `stop()` is called — **a health-server bind failure takes
  down the entire worker process**, not just the probe endpoint. This is intentional fail-fast design (a
  worker that can't be probed shouldn't limp along unprobable), but it means the health check's own
  plumbing is itself a single point of failure for the thing it's meant to be monitoring.

## Shutdown sequence, step by step

1. SIGINT or SIGTERM arrives (from `docker stop`, `kubectl delete pod`, Ctrl+C, etc.).
2. `signal.NotifyContext`'s internal goroutine cancels `ctx`.
3. Every goroutine notices independently, on its own schedule:
   - The consumer's `messaging.Run` loop checks `ctx.Err()` right after its current `runOnce` call returns
     (which itself selects on `<-ctx.Done()` while waiting for the next message) and exits.
   - The outbox worker's ticker `select` sees `<-ctx.Done()`, logs `"stopping worker..."`, and calls its own
     internal `shutdown()` — which waits for its *own* in-flight batches (`activeWg`) to finish, capped at
     its own `StopTimeout` (**30 seconds** by `DefaultConfig()`).
   - The health server's `select` sees `<-ctx.Done()` and calls `server.Shutdown()` with its own 5-second
     timeout.
4. As each goroutine's function body returns, its `defer r.wg.Done()` fires.
5. `App.Run()` is still blocked on `<-ctx.Done()`, which already fired in step 2 — so it immediately moves on
   to `a.awaitShutdown(runner)`.
6. `awaitShutdown` races `<-r.done()` (a channel closed once a *separate* background goroutine's
   `r.wg.Wait()` returns) against a timeout:
   ```go
   select {
   case <-r.done():
       a.logger.Info("shutdown complete")
   case <-time.After(shutdownTimeout): // 10 seconds, every service
       a.logger.Warn("forced shutdown after timeout")
   }
   ```
7. Either way, `awaitShutdown` returns `nil`, `Run()` returns, `defer app.Close()` fires (closing Postgres,
   RabbitMQ connections — exactly which, per service, varies; see the table), and `main()` exits 0.

### The nested-timeout mismatch

Step 6's outer timeout is **10 seconds**. Step 3's outbox-worker inner timeout is **30 seconds**. These are
not coordinated. If the outbox worker is genuinely mid-batch when shutdown begins — plausible, since each
batch can involve real network calls to RabbitMQ with up to a 30-second `EventTimeout` of their own — the
*outer* bootstrap layer can give up and log `"forced shutdown after timeout"` a full 20 seconds before the
*inner* layer's own designed grace period would have elapsed. When that happens, `Run()` returns, `defer
app.Close()` fires immediately, and the Postgres connection pool / RabbitMQ connection the still-running
inner goroutine depends on get closed out from under it — not a memory-safety issue, but the abandoned
goroutine's next DB or AMQP call will simply start failing. In the worst case this means events the
30-second inner timeout was explicitly built to protect ("all in-flight events completed") never get that
protection, because the outer layer never waits long enough to let it matter. This applies to every service
with an outbox worker (identity, restaurant, order, payment, customer) — search-worker and notification have
no such nested timeout, since their only non-health goroutine is the consumer, which has no comparable
internal drain phase.

### Startup failure is invisible to the health probe

If `container.NewWorkerContainer` fails, the process exits before the health server ever starts listening.
From the outside (a `kubectl` probe, a `docker` healthcheck), this looks identical to "still starting up" —
connection refused on `:8081` — right up until the container actually exits and Kubernetes sees a non-zero
exit code instead of a failed probe. The health endpoints this repo built cannot help distinguish "slow to
start" from "failed to start" in this specific window; that distinction is made by the container's own exit
code, not by `/healthz`.

### You cannot Ctrl+C your way to a faster shutdown

`signal.NotifyContext` intercepts SIGINT/SIGTERM for as long as the process is running and `stop()` hasn't
been called to release them. A context can only be cancelled once — a *second* SIGINT while shutdown is
already in progress does nothing further; it doesn't shorten either the 10-second outer or 30-second inner
timeout, and it doesn't force an earlier exit. The only way to actually force-kill a worker stuck past both
timeouts (e.g. `app.Close()` itself hanging on a dead connection) is `SIGKILL` (`kill -9`), which bypasses
the Go runtime's signal handling entirely and is never caught by application code, by design.

## Panic and error handling

`recoverPanic` is deferred in every goroutine, *after* `wg.Done()` in source order — which means it runs
*before* `wg.Done()` during a panic unwind, since deferred calls run LIFO. The sequence on a panic:

1. The panic unwinds the goroutine's stack.
2. The deferred `recoverPanic(stop)` call (registered second, so it runs first) catches it via `recover()`,
   logs the panic value and a full stack trace (`runtime/debug.Stack()`), and calls `stop()` — cancelling
   the shared `ctx`, which is what turns "one goroutine panicked" into "every goroutine begins shutting
   down," rather than one goroutine silently dying while the rest of the process keeps running in a
   half-broken state.
3. Because `recover()` stops the panic from propagating further, the function then returns normally — so
   the deferred `wg.Done()` (registered first, so it runs second) still fires. The WaitGroup is guaranteed to
   reach zero even after a panic; nothing gets permanently stuck waiting on a goroutine that panicked.

## Per-service variation

| Service | Consumer goroutine | Outbox goroutine | Readiness checks | `Close()` tears down |
|---|---|---|---|---|
| identity-worker | — | ✓ | postgres, rabbitmq | Postgres, Publisher (via embedded `Shared.Close()`) |
| restaurant-worker | ✓ | ✓ | postgres, rabbitmq | DB, Consumer, Publisher |
| order-worker | ✓ | ✓ | postgres, rabbitmq | DB, Consumer, Publisher |
| payment-worker | — | ✓ | postgres, rabbitmq | DB, Publisher (no consumer — this service never consumes) |
| customer-worker | ✓ | ✓ | postgres, rabbitmq | DB, Consumer, Publisher |
| search-worker | ✓ | — | elasticsearch, rabbitmq | Consumer only (no DB, no outbox) |
| notification | ✓ | — | rabbitmq | Consumer only (no DB, no outbox) |

Every service's readiness check only covers what *that process* genuinely holds a live connection to — not
a copy of its own API container's checks. search-service's and notification-service's `RabbitMQConsumer`
types are mutex-protected around `conn`/`channel`, because their `rabbitmq` readiness check calls `Ping`
from the health-server goroutine while the consumer loop's own `connect()`/`ensureConnected()` can be
reassigning those same fields from a different goroutine — without the mutex, that would be a genuine data
race, not just a theoretical one.

## Why this exists: health probes

`/healthz/liveness` always returns `200` unconditionally — it only proves the process can still respond to
HTTP, nothing about whether its *actual* job (consuming, relaying) is healthy. `/healthz/readiness` is the
one that calls `Checker.Check(ctx)` and reflects the real dependency pings from the table above, returning
`503` if any fail. Neither endpoint is reachable through Traefik in any service — they're deliberately
container-internal only, reachable only on the pod/container's own `:8081` directly, which is the actual
motivation: this is preparation for the EKS deployment work, where a Kubernetes
`livenessProbe`/`readinessProbe` would point `httpGet` checks at these exact paths and ports. No Kubernetes
manifests exist yet to actually wire that up.
