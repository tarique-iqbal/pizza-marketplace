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
- [Contributing](#contributing)
- [License](#license)


## Architecture overview

```
Client (Web)
        │
        ▼
Traefik API Gateway (:80)
  ├── /auth, /users  ──► Identity Service
  ├── /restaurants   ──► Restaurant Service  (JWT protected)
  └── /search        ──► Search Service      (JWT protected)

Async event flow via RabbitMQ:
  Identity Service  ──► email.verification_created
                    ──► user.registered
                    ──► restaurant.initiated
  Restaurant Service──► restaurant.launched

Consumers:
  Email Service       ◄── email.verification_created, user.registered
  Restaurant Worker   ◄── restaurant.initiated
  Search Service      ◄── restaurant.launched ──► Elasticsearch
```

Each service owns its data store. There is no shared database.

See the system architecture diagram: [Architecture diagram](docs/architecture.md)


## Services

| Service | Language | Role | Exposure |
|---|---|---|---|
| `identity-service` | Go · Gin | Auth, JWT, user management | private |
| `restaurant-service` | Go · Gin | Restaurant & menu CRUD | private |
| `search-service` | Go · Gin | Search API + Elasticsearch indexing | private |
| `email-service` | Go | Email notifications (background worker) | — |
| `identity-worker` | Go | Outbox event publisher (AMQP) | — |
| `restaurant-worker` | Go | Consumes identity events to initialize restaurants | — |

All services are behind Traefik and not directly reachable from outside the Docker network.


## Tech stack

| Layer | Technology |
|---|---|
| Language | Go |
| API gateway | Traefik v3 |
| HTTP framework | Gin |
| Message broker | RabbitMQ 3 |
| Relational DB | PostgreSQL 17 |
| Cache | Redis 7 |
| Search engine | Elasticsearch 8.9 |
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

All routes are served through Traefik on port `80`.

### Identity service — `/auth`, `/users`

| Method | Path | Auth | Description |
|---|---|---|---|
| `POST` | `/auth/email/verify` | — | Create email verification request |
| `POST` | `/auth/login` | — | Login user and issue JWT tokens |
| `POST` | `/auth/refresh` | — | Refresh access token |
| `GET` | `/auth/verify` | JWT | Internal forward-auth endpoint (used by Traefik) |
| `POST` | `/auth/logout` | JWT | Invalidate refresh token |
| `POST` | `/users/owners` | — | Register owner account |
| `POST` | `/users/customers` | — | Register customer account |
| `GET` | `/users/{id}` | JWT | Get user by ID |

### Restaurant service — `/restaurants`

| Method | Path | Auth | Description |
|---|---|---|---|
| `PATCH` | `/restaurants/{id}/address` | JWT | Update address |

### Search service — `/search`

| Method | Path | Auth | Description |
|---|---|---|---|
| `GET` | `/search/restaurants` | — | Search restaurants (geo, keyword) |


## Event flow

Events are published to RabbitMQ and consumed asynchronously. The outbox pattern is used in producer services for at-least-once delivery.

```
identity-service  ──publishes──► email.verification_created
                                 user.registered
                                 restaurant.initiated

restaurant-service──publishes──► restaurant.launched

email-service     ◄──consumes── email.verification_created
                                user.registered

restaurant-worker ◄──consumes── restaurant.initiated

search-service    ◄──consumes── restaurant.launched
                  ──indexes───► Elasticsearch
```


## Project structure

```
pizza-marketplace/
├── docker/
│   ├── identity-service/Dockerfile
│   ├── restaurant-service/Dockerfile
│   ├── email-service/Dockerfile
│   └── search-service/Dockerfile
│
├── identity-service/
│   ├── cmd
│   │   ├── api
│   │   └── worker
│   ├── internal
│   │   ├── application
│   │   ├── domain
│   │   ├── infrastructure
│   │   └── interfaces
│   └── .env.example
│
├── restaurant-service/
├── email-service/
├── search-service/
├── web-user/
│
├── docker-compose.yml
└── README.md
```


## Roadmap

- [ ] Search service — Elasticsearch-based search and indexing
- [ ] Profile service — user profile, address, and payment info management
- [ ] Order service — place and track orders
- [ ] Payment service — payment processing
- [ ] Web user client — React frontend application
- [ ] Notification service — SMS/web notification consumer
- [ ] Analytics service — metrics, reporting, and audit logs
- [ ] gRPC inter-service communication
- [ ] Zero-trust networking — trusted proxies, mTLS, and workload identity
- [ ] Observability stack — logs, metrics, traces, and monitoring
- [ ] Kubernetes manifests
- [ ] CI/CD pipeline
- [ ] Cloud deployment — deploy infrastructure to cloud environment
- [ ] AI-assisted automation — deployment optimization and anomaly detection


## Contributing

⚠️ This project is in early development. Pull requests are not currently accepted and will open after v1.0.0. The project has known issues, but you are welcome to explore it and provide feedback via issues.


## License

This project is licensed under the [GNU Affero General Public License v3.0](LICENSE).
