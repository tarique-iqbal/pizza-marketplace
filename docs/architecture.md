# Architecture

This diagram describes the current system architecture, service boundaries, communication patterns, security model, and infrastructure design of the platform.

```mermaid
flowchart TD
  CLIENT["Web / API"]

  subgraph GW["Traefik — API Gateway :80"]
    direction LR
    ROUTE["Routing"]
    JWT["JWT forward-auth"]
    MW["Middleware"]
    RP["Reverse proxy"]
  end

  subgraph BACKEND["Backend network"]
    IS["Identity service\n/auth  /users · API + worker (outbox relay)"]
    RS["Restaurant service\n/restaurants · API · own OpenCage client"]
    SS["Search service\n/search, no auth · API · own OpenCage client"]
  end

  BROKER(["RabbitMQ — event broker"])

  subgraph CONSUMERS["Consumers"]
    NOTIF["Notification service\nworker only — channel adapters, email today"]
    RW["Restaurant worker\nrestaurant-service's own cmd/worker\ninbound consumer + outbox relay"]
    SW["Search worker\nsearch-service's own cmd/worker"]
  end

  subgraph STORES["Data stores — internal network"]
    direction LR
    PGI["PostgreSQL\nidentity_db"]
    PGR["PostgreSQL\nrestaurant_db"]
    REDIS["Redis\nrefresh tokens · sessions"]
    ES["Elasticsearch\nsearch index + geocode cache"]
  end

  CLIENT -->|HTTP :80| GW
  GW -->|"/auth, /users"| IS
  GW -->|"/restaurants + verify JWT"| RS
  GW -->|"/search, no auth"| SS

  IS -->|owns| PGI
  IS -->|owns| REDIS
  RS -->|owns| PGR
  SS -->|queries| ES

  IS -- "restaurant.initiated, user.registered,\nemail.verification_created (outbox)" --> BROKER
  RW -- "restaurant.ready_for_review, restaurant.approved, restaurant.launched,\nrestaurant.updated, restaurant.pizza_updated, restaurant.topping_prices_updated (outbox)" --> BROKER

  BROKER -- "email.verification_created\nuser.registered\nrestaurant.ready_for_review\nrestaurant.approved" --> NOTIF
  BROKER -- "restaurant.initiated" --> RW
  BROKER -- "restaurant.launched\nrestaurant.updated\nrestaurant.pizza_updated\nrestaurant.topping_prices_updated" --> SW

  RW -->|creates| PGR
  SW -->|indexes| ES
```

- **`identity-service` outboxes every event it raises** (`restaurant.initiated`, `user.registered`, `email.verification_created`) — no best-effort publish path left in that service.
- **`restaurant-service` uses the outbox pattern too**, same full-scope shape as identity-service: it outboxes every event it raises. The relay runs as a second goroutine inside `RW` (`cmd/worker`), alongside the existing inbound `restaurant.initiated` consumer — `RS` (the API) never talks to `BROKER` directly.
- **`search-service` has no Postgres database** — its only store is Elasticsearch, which doubles as the search index and a disposable geocode cache (a second index, unrelated to search, safe to delete anytime since a cache miss just re-populates it).
- **`notification-service` is a pure event-to-notification pipeline** — one handler per consumed event, dispatching through a channel-agnostic `Sender` interface (email today, via `text/template` + SMTP; a second channel would be a new adapter behind the same interface). It holds no state of its own beyond what's in each event's payload, so it needs no database.
