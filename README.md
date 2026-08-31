# Pizza Marketplace – Monorepo

**Online pizza ordering marketplace** built with a **microservices architecture**, enabling restaurants to receive and manage orders. Each service is developed using **Go (Gin)**, with **PostgreSQL** for transactional data and **Elasticsearch** for search. Services communicate asynchronously via **RabbitMQ** and expose **HTTP APIs** through a **Traefik API Gateway** with **JWT-based authentication**.

The platform follows **Domain-Driven Design** and **Clean Architecture** principles. It implements the **Outbox Pattern** for reliable event delivery and data consistency. **Docker** is used for containerization, and **Kubernetes** orchestration is planned for scalable deployments.

## Table of contents

- [Architecture overview](#architecture-overview)
- [Services](#services)
- [Tech stack](#tech-stack)
- [Getting started](#getting-started)
- [Environment variables](#environment-variables)
- [API routes](#api-routes)
- [Event flow](#event-flow)
- [Project structure](#project-structure)
- [Roadmap](#roadmap)


## Architecture overview

```
Client (Web)
        │
        ▼
Traefik API Gateway (:80)
  ├── /auth, /users  ──► Identity Service
  ├── /restaurants   ──► Restaurant Service  (JWT protected)
  └── /search        ──► Search Service      (no auth)

RabbitMQ (async events, publish/consume — see Event flow below)
  ├── Identity Service
  ├── Restaurant Service
  ├── Search Service
  └── Email Service      (worker only — no HTTP route, reached only via RabbitMQ)
```

Each service owns its data store. There is no shared database. See the
[Architecture diagram](docs/architecture.md) for the full picture.


## Services

| Service | Role | Exposure |
|---|---|---|
| `identity-service` | Auth, JWT, user management | mixed (public and JWT-protected) |
| `restaurant-service` | Restaurant & menu CRUD | JWT-protected |
| `search-service` | Search API + Elasticsearch indexing | public (no auth) |
| `email-service` | Email notifications (background worker) | — |

`identity-service`, `restaurant-service`, and `search-service` each also run a `cmd/worker` process (outbox relay / event consumer) alongside their API — not separate services. `identity-service`'s and `search-service`'s workers (`identity-worker`, `search-worker`) each get their own container in `compose.yaml` and run by default in dev — without them, no outbox event ever leaves identity-service and the search index stays permanently empty, respectively. `restaurant-service`'s worker is started manually when working on its outbox/consumer code.

All services are behind Traefik and not directly reachable from outside the Docker network.


## Tech stack

| Layer | Technology |
|---|---|
| Language | Go |
| HTTP framework | Gin |
| Message broker | RabbitMQ 3 |
| Relational DB | PostgreSQL 17 |
| Cache | Redis 7 |
| Search engine | Elasticsearch 8.9 |
| API gateway | Traefik v3 |
| Containerisation | Docker · Docker Compose |


## Getting started

### Prerequisites

- [Docker](https://docs.docker.com/get-docker/) ≥ 29.4
- [Docker Compose](https://docs.docker.com/compose/) ≥ 2.34
- [Go](https://go.dev/dl/) ≥ 1.24

### Run with Docker Compose

```bash
# 1. Clone the repo
git clone https://github.com/tarique-iqbal/pizza-marketplace.git
cd pizza-marketplace

# 2. Copy env files and fill in values
cp identity-service/.env.example   identity-service/.env
cp restaurant-service/.env.example restaurant-service/.env
cp email-service/.env.example      email-service/.env
cp search-service/.env.example     search-service/.env

# 3. Start all services
docker compose up --build
```

The gateway is available at `http://localhost:80`  
The Traefik dashboard is at `http://localhost:8080`  
The RabbitMQ management UI is at `http://localhost:15672`

### Stop services

```bash
docker compose down
```

To also remove volumes (wipes all data):

```bash
docker compose down -v
```


## Environment variables

Each service is configured via its own `.env` file. Copy the `.env.example` in each service directory and update the values.


## API routes

All routes are served through Traefik on port `80`. See each service's API reference for details:

- [Identity service](docs/api/identity-service.md) — `/auth`, `/users`
- [Restaurant service](docs/api/restaurant-service.md) — `/restaurants`
- [Search service](docs/api/search-service.md) — `/search`


## Service documentation

Architecture, domain model, and design decisions for each implemented service:

- [Identity service](docs/services/identity-service.md)
- [Restaurant service](docs/services/restaurant-service.md)
- [Email service](docs/services/email-service.md)
- [Search service](docs/services/search-service.md)


## Event flow

Events are published to RabbitMQ and consumed asynchronously. `identity-service` and `restaurant-service` both use the transactional outbox pattern for at-least-once delivery — each outboxes every event it raises, with no best-effort publish path left in either service.

```
identity-service     ──publishes──► email.verification_created
                                     user.registered
                                     restaurant.initiated

restaurant-service   ──publishes──► restaurant.ready_for_review
                                     restaurant.approved
                                     restaurant.launched
                                     restaurant.updated
                                     restaurant.pizza_updated
                                     restaurant.topping_prices_updated

email-service        ◄──consumes──  email.verification_created
                                     user.registered
                                     restaurant.ready_for_review
                                     restaurant.approved

restaurant-service   ◄──consumes──  restaurant.initiated

search-service       ◄──consumes──  restaurant.launched
                                     restaurant.updated
                                     restaurant.pizza_updated
                                     restaurant.topping_prices_updated
```

`restaurant.approved` and `restaurant.ready_for_review` are the only restaurant-service events with no
`search-service` consumer — they fire pre-launch, before there is anything to index.


## Project structure

```
pizza-marketplace/
├── identity-service/
│   ├── cmd
│   │   ├── api
│   │   └── worker
│   ├── internal
│   │   ├── application
│   │   ├── domain
│   │   ├── infrastructure
│   │   └── interfaces
│   ├── Dockerfile
│   └── .env.example
├── restaurant-service/
├── email-service/
├── search-service/
├── compose.yaml
├── compose.test.yaml
└── README.md
```


## Roadmap

- [ ] Profile service — user profile, address, and payment info management
- [ ] Payment service — payment processing
- [ ] Order service — place and track orders
- [ ] Notification service — SMS/web notification consumer
- [ ] Analytics service — metrics, reporting, and audit logs
- [ ] gRPC inter-service communication
- [ ] Zero-trust networking — trusted proxies, mTLS, and workload identity
- [ ] Observability stack — logs, metrics, traces, and monitoring
- [ ] Web user client — React frontend application (separate repo)
- [ ] Kubernetes manifests
- [ ] CI/CD pipeline
- [ ] Cloud deployment — deploy infrastructure to cloud environment
- [ ] AI-assisted automation — deployment optimization and anomaly detection
