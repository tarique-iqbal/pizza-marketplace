# Identity Service API

Routes served under `/auth`, `/users`, and `/health` via the Traefik gateway.

## Auth — `/auth`

| Method | Path | Auth | Description |
|---|---|---|---|
| `POST` | `/auth/email/verify` | — | Create email verification request |
| `POST` | `/auth/login` | — | Login user and issue JWT tokens |
| `POST` | `/auth/refresh` | — | Refresh access token |
| `GET` | `/auth/verify` | JWT | Internal forward-auth endpoint (used by Traefik) |
| `POST` | `/auth/logout` | JWT | Invalidate refresh token |

## Users — `/users`

| Method | Path | Auth | Description |
|---|---|---|---|
| `POST` | `/users/owners` | — | Register owner account |
| `POST` | `/users/customers` | — | Register customer account |
| `GET` | `/users/{id}` | JWT | Get user by ID |

## Health — `/health`

| Method | Path | Auth | Description |
|---|---|---|---|
| `GET` | `/health/live` | — | Liveness probe |
| `GET` | `/health/ready` | — | Readiness probe |
