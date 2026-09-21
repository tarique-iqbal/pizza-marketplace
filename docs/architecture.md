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
    CS["Customer service\n/customers · API"]
    SS["Search service\n/search, no auth · API · own OpenCage client"]
  end

  BROKER(["RabbitMQ — event broker"])

  subgraph CONSUMERS["Consumers"]
    NOTIF["Notification service\nworker only — channel adapters, email today"]
    RW["Restaurant worker\nrestaurant-service's own cmd/worker\ninbound consumer + outbox relay"]
    SW["Search worker\nsearch-service's own cmd/worker"]
    OW["Order worker\norder-service's own cmd/worker\nread-model + payment-event consumer, outbox relay"]
    PW["Payment worker\npayment-service's own cmd/worker\noutbox relay only, no inbound consumer"]
    CW["Customer worker\ncustomer-service's own cmd/worker\ninbound consumer + outbox relay"]
  end

  subgraph STORES["Data stores — internal network"]
    direction LR
    PGI[("PostgreSQL\nidentity_db")]
    PGR[("PostgreSQL\nrestaurant_db")]
    PGO[("PostgreSQL\norder_db")]
    PGP[("PostgreSQL\npayment_db")]
    PGC[("PostgreSQL\ncustomer_db")]
    ES[("Elasticsearch\nsearch index + geocode cache")]
    REDIS[("Redis\nrefresh tokens · sessions · OTP rate limit")]
  end

  CLIENT -->|HTTP :80| GW
  GW -->|"/auth, /users"| IS
  GW -->|"/restaurants + verify JWT"| RS
  GW -->|"/search, no auth"| SS
  GW -->|"/cart, /orders + verify JWT"| OS
  GW -->|"/webhooks/mollie, no auth"| PS
  GW -->|"/customers + verify JWT"| CS

  IS -->|owns| PGI
  IS -->|owns| REDIS
  RS -->|owns| PGR
  SS -->|queries| ES
  OS -->|owns| PGO
  PS -->|owns| PGP
  CS -->|owns| PGC

  IS -- "restaurant.initiated, user.registered,\nemail.verification_created (outbox)" --> BROKER
  RW -- "restaurant.ready_for_review, restaurant.approved, restaurant.launched,\nrestaurant.updated, restaurant.pizza_updated, restaurant.topping_prices_updated (outbox)" --> BROKER
  PW -- "payment.succeeded, payment.failed (outbox)" --> BROKER
  OW -- "order.confirmed, order.address_saved (outbox)" --> BROKER
  CW -- "customer.phone_updated (outbox)" --> BROKER

  BROKER -- "email.verification_created\nuser.registered\nrestaurant.ready_for_review\nrestaurant.approved\norder.confirmed" --> NOTIF
  BROKER -- "restaurant.initiated" --> RW
  BROKER -- "restaurant.launched\nrestaurant.updated\nrestaurant.pizza_updated\nrestaurant.topping_prices_updated" --> SW
  BROKER -- "restaurant.launched\nrestaurant.updated\nrestaurant.pizza_updated\nrestaurant.topping_prices_updated\nuser.registered\ncustomer.phone_updated\npayment.succeeded\npayment.failed" --> OW
  BROKER -- "user.registered\norder.address_saved" --> CW
  BROKER ~~~ PW

  RW -->|creates| PGR
  CW -->|"creates profiles + saved addresses"| PGC
  SW -->|indexes| ES
  OW -->|"syncs read-model + clears cart on checkout\nconfirms/cancels order on payment outcome"| PGO
```

- **`identity-service` outboxes every event it raises** (`restaurant.initiated`, `user.registered`, `email.verification_created`). It's also the only service enforcing request-level abuse protection today: a per-email cooldown and a per-code attempt cap on OTP verification, both backed by the same Redis instance used for refresh tokens.
- **`restaurant-service` uses the outbox pattern too**, same full-scope shape as identity-service: it outboxes every event it raises. The relay runs as a goroutine inside the restaurant worker (`cmd/worker`), next to the inbound `restaurant.initiated` consumer — the restaurant API never talks to RabbitMQ directly.
- **`search-service` has no Postgres database** — its only store is Elasticsearch, which doubles as the search index and a disposable geocode cache (a second index, unrelated to search, safe to delete anytime since a cache miss just re-populates it) behind a `CachingGeocoder` decorator wrapping its OpenCage client.
- **`notification-service` is a pure event-to-notification pipeline** — one handler per consumed event, dispatching through a channel-agnostic `Sender` interface (email today, via `text/template` + SMTP; a second channel would be a new adapter behind the same interface). It holds no state of its own beyond what's in each event's payload, so it needs no database.
- **`order-service` owns the customer's cart and order lifecycle**, backed by its own Postgres (`order_db`) and a Postgres-backed geocode cache for delivery-address checks. Checkout/cancel call payment-service over gRPC (`CreatePayment`/`CancelPayment`/`GetPaymentStatus`, behind a circuit breaker); its worker confirms/cancels orders on `payment.succeeded`/`payment.failed` and relays its own `order.confirmed` and `order.address_saved` outbox events.
- **`payment-service` processes payments via Mollie**, generic on `subject_type`/`subject_id`, not tied to order-service's concept of an order. Its primary surface is **gRPC** (internal only, `:50051`); its one HTTP route, `/webhooks/mollie`, never trusts the POSTed body — it always re-fetches true status from Mollie. The payment worker is the only worker with no inbound consumer — it only publishes `payment.succeeded`/`payment.failed`.
- **`customer-service` owns a customer's phone and saved delivery addresses**, backed by its own Postgres (`customer_db`). Name and email are a read-only mirror of identity-service, created by consuming `user.registered`. Addresses have no create endpoint: the customer worker creates them from order-service's `order.address_saved`, so nothing that didn't pass a real checkout can be saved. `PATCH /customers/me/phone` publishes `customer.phone_updated` through its outbox, which order-service consumes to use as the delivery contact number. The customer worker runs an inbound consumer and an outbox relay side by side.
