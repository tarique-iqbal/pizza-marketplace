# Architecture

This diagram describes the current system architecture, service boundaries, communication patterns, security model, and infrastructure design of the platform.

```mermaid
flowchart TD
  CLIENT["Web / API"]

  subgraph GW["Traefik — API Gateway :80"]
    direction LR
    RP["Reverse proxy"] --> ROUTE["Routing"] --> JWT["JWT forward-auth"] --> MW["Middleware"]
  end

  subgraph BACKEND["Backend network"]
    IS["Identity service\n/auth  /users · API + worker (outbox relay)"]
    RS["Restaurant service\n/restaurants · API · own OpenCage client"]
    OS["Order service\n/cart, /orders · API · own OpenCage client"]
    PS["Payment service\ngRPC :50051 (internal) · webhook, no auth"]
    SS["Search service\n/search, no auth · API · own OpenCage client"]
  end

  BROKER(["RabbitMQ — event broker"])

  subgraph CONSUMERS["Consumers"]
    NOTIF["Notification service\nworker only — channel adapters, email today"]
    RW["Restaurant worker\nrestaurant-service's own cmd/worker\ninbound consumer + outbox relay"]
    SW["Search worker\nsearch-service's own cmd/worker"]
    OW["Order worker\norder-service's own cmd/worker\nread-model consumer + outbox relay"]
    PW["Payment worker\npayment-service's own cmd/worker\noutbox relay only, no inbound consumer"]
  end

  subgraph STORES["Data stores — internal network"]
    direction LR
    PGI[("PostgreSQL\nidentity_db")]
    PGR[("PostgreSQL\nrestaurant_db")]
    PGO[("PostgreSQL\norder_db")]
    PGP[("PostgreSQL\npayment_db")]
    ES[("Elasticsearch\nsearch index + geocode cache")]
    REDIS[("Redis\nrefresh tokens · sessions · OTP rate limit")]
  end

  CLIENT -->|HTTP :80| GW
  GW -->|"/auth, /users"| IS
  GW -->|"/restaurants + verify JWT"| RS
  GW -->|"/search, no auth"| SS
  GW -->|"/cart, /orders + verify JWT"| OS
  GW -->|"/webhooks/mollie, no auth"| PS

  IS -->|owns| PGI
  IS -->|owns| REDIS
  RS -->|owns| PGR
  SS -->|queries| ES
  OS -->|owns| PGO
  PS -->|owns| PGP

  IS -- "restaurant.initiated, user.registered,\nemail.verification_created (outbox)" --> BROKER
  RW -- "restaurant.ready_for_review, restaurant.approved, restaurant.launched,\nrestaurant.updated, restaurant.pizza_updated, restaurant.topping_prices_updated (outbox)" --> BROKER
  PW -- "payment.succeeded, payment.failed (outbox)" --> BROKER

  BROKER -- "email.verification_created\nuser.registered\nrestaurant.ready_for_review\nrestaurant.approved" --> NOTIF
  BROKER -- "restaurant.initiated" --> RW
  BROKER -- "restaurant.launched\nrestaurant.updated\nrestaurant.pizza_updated\nrestaurant.topping_prices_updated" --> SW
  BROKER -- "restaurant.launched\nrestaurant.updated\nrestaurant.pizza_updated\nrestaurant.topping_prices_updated\nuser.registered" --> OW
  BROKER ~~~ PW

  RW -->|creates| PGR
  SW -->|indexes| ES
  OW -->|syncs read-model + clears cart on checkout| PGO
```

- **`identity-service` outboxes every event it raises** (`restaurant.initiated`, `user.registered`, `email.verification_created`). It's also the only service enforcing request-level abuse protection today: a per-email cooldown and a per-code attempt cap on OTP verification, both backed by the same `REDIS` instance used for refresh tokens.
- **`restaurant-service` uses the outbox pattern too**, same full-scope shape as identity-service: it outboxes every event it raises. The relay runs as a second goroutine inside `RW` (`cmd/worker`), alongside the existing inbound `restaurant.initiated` consumer — `RS` (the API) never talks to `BROKER` directly.
- **`search-service` has no Postgres database** — its only store is Elasticsearch, which doubles as the search index and a disposable geocode cache (a second index, unrelated to search, safe to delete anytime since a cache miss just re-populates it) behind a `CachingGeocoder` decorator wrapping its OpenCage client.
- **`notification-service` is a pure event-to-notification pipeline** — one handler per consumed event, dispatching through a channel-agnostic `Sender` interface (email today, via `text/template` + SMTP; a second channel would be a new adapter behind the same interface). It holds no state of its own beyond what's in each event's payload, so it needs no database.
- **`order-service` owns the customer's cart, backed by its own Postgres (`order_db`) and its own OpenCage client for delivery-address geocoding.** Unlike restaurant-service, which calls OpenCage directly per request, its geocoder sits behind a Postgres-backed cache (a `geocode_cache` table in `order_db`, keyed by a hash of the normalized address, no TTL) — a given address is only ever geocoded once. Its worker (`OW`) has two jobs, same dual-goroutine shape as `RW`: it consumes `restaurant.*` and `user.registered` events into a local read-model that cart pricing and availability checks read from (never a live call to another service's database), and it relays `order-service`'s own outbox. Auth for `/cart`/`/orders` is any authenticated user, not role-restricted.
- **`payment-service` processes payments via Mollie, on behalf of any other service in this system** — it's generic on `subject_type`/`subject_id`, never tied to `order-service`'s own concept of an order. It's the only service whose primary surface is **gRPC** (`PaymentService`, internal network only, `:50051`, no Traefik exposure) rather than REST; its one HTTP route, the public `/webhooks/mollie` callback, never trusts the POSTed body — it always re-fetches the true payment status from Mollie's own API first. `PW` (`cmd/worker`) is the only worker in this repo with **no inbound consumer** — payment-service consumes nothing, it only publishes `payment.succeeded`/`payment.failed` once a webhook resolves a payment. Neither `order-service`'s own gRPC call to `CreatePayment`/`CancelPayment` nor its consumption of `payment.events` exist yet — both are order-service's own not-yet-built steps, so those connections aren't drawn above.
