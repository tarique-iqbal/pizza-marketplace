---
name: rabbitmq-exchange
description: Naming and binding conventions for RabbitMQ exchanges, queues, and DLXs in this repo — read before adding a new exchange, queue, or event binding to any service
---

# RabbitMQ exchange conventions

Use this when adding a new event, a new exchange, or a new consumer binding to any service's messaging
layer. Read the target service's own `CLAUDE.md` first for which events it actually raises/consumes — this
skill only covers *how exchanges, queues, and bindings are named and wired*, not which events exist.

## 1. Every exchange in this repo today is `topic`, including DLXs

No `direct`, `fanout`, or `headers` exchange exists anywhere in this repo. Every publish exchange
(`identity.events`, `restaurant.events`, `order.events`, `payment.events`, `customer.events`) and every DLX
(`restaurant_dlx`, `search_dlx`, `order_dlx`, `customer_dlx`, `email_dlx`) is declared the same way:
`ExchangeDeclare(name, "topic", true, false, false, false, nil)` — durable, not auto-deleted, no extra
arguments. Routing keys are always exact event names (`user.registered`, `restaurant.launched`, ...), never
wildcard patterns (`restaurant.*`) — `topic` is used here purely for its exact-match routing, not its
wildcard capability. See §6 before introducing a different exchange type.

## 2. Publisher side: one `<service>.events` exchange per publishing service

A service that raises domain events owns exactly **one** exchange, named `<service>.events` — not one
exchange per event type. Declared in `internal/infrastructure/messaging/rabbitmq_publisher.go`'s `connect()`:

```go
const exchangeName = "<service>.events"
// ...
ch.ExchangeDeclare(exchangeName, "topic", true, false, false, false, nil)
```

Routing key on publish is the event's own name (`payload.GetEventName()`), nothing else. A new event type
from an existing publisher needs no new exchange and no new `ExchangeDeclare` call — it publishes to the
same `<service>.events` exchange with its own routing key.

A service only gets its own `.events` exchange once it actually raises an event. `search-service` and
`notification-service` have none — both are pure consumers (sinks), never publishers.

## 3. Consumer side: one queue + one DLX per consuming service

Each consuming service declares, in its own `rabbitmq_consumer.go`'s `connect()`:

- **Its own DLX**, named `<service>_dlx` (e.g. `restaurant_dlx`) — `topic`, durable. Note:
  `notification-service`'s is still `email_dlx`/`email_queue` (its main queue, below), left over from before
  its rename from `email-service` — don't silently "fix" this to `notification_*` without checking whether
  it's worth the breaking rename first.
- **Its own main queue**, named `<service>_queue`, durable, with `x-dead-letter-exchange: <service>_dlx` and
  `x-dead-letter-routing-key: <service>_queue.retry` as queue arguments.
- **An `Exchanges` map** (`map[string][]string`, exchange name → routing keys) listing every upstream
  exchange this service binds to and exactly which event names it wants from each. `connect()` loops this
  map to `ExchangeDeclare` each referenced exchange (so a consumer never depends on its publisher having
  started first) and `QueueBind`s the main queue to it once per routing key.
- **One DLQ**, named `<service>_queue.dlq`, bound to the DLX with routing key `<service>_queue.retry` — where
  a message lands after exhausting `MaxRetryAttempts` (always `3` across every service), held for manual
  inspection/replay, never auto-reprocessed.

**Adding a new inbound event to an existing consumer**: add one entry to its `Exchanges` map (a new exchange
key if it's a new upstream publisher, or a new routing key under an existing one), plus a handler registered
against that exact routing-key string in `internal/container/worker.go` — a literal string, not a constant
imported from the publishing service (no shared Go module in this repo; see root `CLAUDE.md`).

## 4. Retries don't go back through the topic exchange

A failed delivery is retried via the **default exchange** (`""`), published directly to the queue by name:

```go
channel.PublishWithContext(ctx, "", QueueName, false, false, amqp.Publishing{
    Headers: headers, // x-retry-count incremented here
    // ...
})
```

Not republished through the `.events` topic exchange or the DLX. Only final exhaustion (`Nack`, no requeue)
routes through the queue's dead-letter arguments into the DLX → DLQ. Keep this distinction if touching retry
logic: the DLX only ever sees messages being given up on, never an in-progress retry.

## 5. `payment-service` is publisher-only, by design

It declares an `.events` exchange (publishes `payment.succeeded`/`payment.failed`) but has no consumer, no
queue, no DLX, and no `Exchanges` map at all — the one service in this repo with no inbound events. Don't
add consumer scaffolding to it speculatively; it would have no publisher ever targeting it until a real
upstream need exists.

## 6. When would a type other than `topic` actually fit?

No event in this repo needs `fanout`'s blast-to-every-subscriber semantics (every consumer wants a specific,
named subset of events), `direct` would just duplicate `topic`'s existing exact-match routing, and `headers`
only pays off once routing depends on several independent attributes rather than one event name. Default to
`topic` for any new exchange; only reach for another type when a real routing requirement actually needs
what it uniquely offers, not on a hunch.
