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
(`restaurant_dlx`, `search_dlx`, `order_dlx`, `customer_dlx`, `notification_dlx`) is declared the same way:
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

## 3. Consumer side: three queues + one DLX per consuming service

The consumer itself is shared code (`shared/amqp/rabbitmq_consumer.go`). Each consuming service only
describes its topology in `internal/infrastructure/messaging/bindings.go` (`QueueName`, `DLXName`,
`Exchanges`, returned by `ConsumerConfig()`), and the shared `connect()` declares:

- **Its own DLX**, named `<service>_dlx` (e.g. `restaurant_dlx`) — `topic`, durable.
- **Its own main queue**, named `<service>_queue`, durable, with `x-dead-letter-exchange: <service>_dlx` and
  `x-dead-letter-routing-key: <service>_queue.retry` as queue arguments.
- **An `Exchanges` map** (`map[string][]string`, exchange name → routing keys) listing every upstream
  exchange this service binds to and exactly which event names it wants from each. `connect()` loops this
  map to `ExchangeDeclare` each referenced exchange (so a consumer never depends on its publisher having
  started first) and `QueueBind`s the main queue to it once per routing key.
- **One wait queue**, named `<service>_queue.wait`, bound to the DLX with routing key `<service>_queue.retry`,
  with `x-message-ttl` = `Config.RetryDelay` (5 s by default) and `x-dead-letter-exchange: ""` +
  `x-dead-letter-routing-key: <service>_queue`, so an expired message goes straight back to the main queue.
  It never has consumers. Changing the delay means deleting this queue first: the TTL is a queue argument.
- **One DLQ**, named `<service>_queue.dlq`, bound to the DLX with routing key `<service>_queue.dlq` — where
  a message lands after exhausting `MaxRetryAttempts` (always `3` across every service), held for manual
  inspection/replay, never auto-reprocessed. `connect()` also removes any older binding of the DLQ to
  `<service>_queue.retry`, which would otherwise copy every retry into it.

**Adding a new inbound event to an existing consumer**: add one entry to its `Exchanges` map (a new exchange
key if it's a new upstream publisher, or a new routing key under an existing one), plus a handler registered
against that exact routing-key string in `internal/container/worker.go` — a literal string, not a constant
imported from the publishing service (`shared/` holds infrastructure code only, never event names or other
domain constants).

## 4. Retries go through the DLX and the wait queue, never the topic exchange

The broker does the retrying; the consumer never republishes a retry itself (`handle` in `shared/amqp`):

- **Retries left:** `Nack(false, false)`. The main queue's dead-letter arguments route the message through
  the DLX (key `<service>_queue.retry`) into `.wait`; after the TTL it returns via the default exchange (`""`)
  to the main queue. No sleep in the consumer, so one failing message never blocks the queue.
- **Retries used up:** the consumer publishes the message to the DLX with key `<service>_queue.dlq` and waits
  for the broker's confirm (`DeadLetter`) before acking the original. If that fails it nacks again, so the
  message takes one more pass through `.wait` instead of being lost.

A redelivered message arrives with exchange `""` and routing key `<service>_queue`. The retry count and the
original exchange and routing key come from the broker's `x-death` entry for the main queue with reason
`rejected` (`firstRejection`), which keeps the original values on every pass. Don't add custom retry
headers, and keep the two DLX routing keys distinct: `.retry` means "try again later", `.dlq` means "given up".

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
