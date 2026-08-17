# Pizza Marketplace – Monorepo

**Online pizza ordering marketplace** built with a **microservices architecture**, enabling restaurants to receive and manage orders. Each service is developed using **Go (Gin)**, with **PostgreSQL** for transactional data and planned **Elasticsearch** integration for high-performance search. Services communicate asynchronously via **RabbitMQ** and expose **HTTP APIs** through a **Traefik API Gateway** with **JWT-based authentication**.

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
  └── /search        ──► Search Service      (planned)

Async event flow via RabbitMQ:
  Identity Service  ──► email.verification_created
                    ──► user.registered
                    ──► restaurant.initiated

Consumers:
  Email Service       ◄── email.verification_created, user.registered
  Restaurant Service   ◄── restaurant.initiated (worker)
```

Each service owns its data store. There is no shared database.

See the system architecture diagram: [Architecture diagram](docs/architecture.md)


## Services

| Service | Role | Exposure |
|---|---|---|
| `identity-service` | Auth, JWT, user management | mixed (public and JWT-protected) |
| `restaurant-service` | Restaurant & menu CRUD | JWT-protected |
| `search-service` | Search API + Elasticsearch indexing | **not implemented** |
| `email-service` | Email notifications (background worker) | — |

`identity-service` and `restaurant-service` each also run a `cmd/worker` process (outbox relay / event consumer) alongside their API — not separate services.

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

# 3. Start all services
docker compose up --build
```

The gateway is available at `http://localhost:80`.  
The Traefik dashboard is at `http://localhost:8080`.  
The RabbitMQ management UI is at `http://localhost:15672`.

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


## Event flow

Events are published to RabbitMQ and consumed asynchronously. The outbox pattern is used in producer services for at-least-once delivery.

```
identity-service   ──publishes──► email.verification_created
                                   user.registered
                                   restaurant.initiated

email-service       ◄──consumes── email.verification_created
                                   user.registered

restaurant-service   ◄──consumes── restaurant.initiated
```

Planned, not wired up yet: once `search-service` is implemented, `restaurant-service` will publish `restaurant.launched` for it to consume and index into Elasticsearch.


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
- [ ] Search service — Elasticsearch-based search and indexing
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
