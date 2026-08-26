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
    RS["Restaurant service\n/restaurants · API"]
    SS["Search service\n/search, no auth · API"]
  end

  BROKER(["RabbitMQ — event broker"])

  subgraph CONSUMERS["Consumers"]
    EMAIL["Email service\nworker only — sends emails / notifications"]
    RW["Restaurant worker\nrestaurant-service's own cmd/worker"]
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

  IS -- "restaurant.initiated (outbox)\nuser.registered, email.verification_created (best-effort)" --> BROKER
  RS -- "restaurant.ready_for_review, restaurant.approved, restaurant.launched,\nrestaurant.updated, restaurant.pizza_updated, restaurant.topping_prices_updated" --> BROKER

  BROKER -- "email.verification_created\nuser.registered\nrestaurant.ready_for_review\nrestaurant.approved" --> EMAIL
  BROKER -- "restaurant.initiated" --> RW
  BROKER -- "restaurant.launched\nrestaurant.updated\nrestaurant.pizza_updated\nrestaurant.topping_prices_updated" --> SW

  RW -->|creates| PGR
  SW -->|indexes| ES
```

- **`restaurant-service` does not use the outbox pattern** — unlike `identity-service`, its RabbitMQ publisher is called directly/best-effort from the same request that mutates the `restaurants` row, for every event it raises.
- **`restaurant.reactivated`/`restaurant.deactivated`** are defined nowhere yet — `Restaurant.Deactivate`/`Reactivate` don't exist as domain methods, so nothing publishes them and `search-service` has no corresponding delete/de-index path. `restaurant.rejected` doesn't exist either — only `Approve`/`Launch` are implemented on the status workflow so far, no `Reject`.
- **Both `restaurant-service` and `search-service` call OpenCage geocoding independently** — separate API clients, separate quotas, not shared. Not shown above since the diagram scopes to internal services/stores/broker, not external APIs.
- **`search-service` has no Postgres database** — its only store is Elasticsearch, which doubles as the search index and a disposable geocode cache (a second index, unrelated to search, safe to delete anytime since a cache miss just re-populates it).
