# Stage 1: Base (Dependencies & Tools)
FROM golang:1.24 AS base
WORKDIR /app

RUN CGO_ENABLED=0 go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@v4.19.1

COPY identity-service/go.mod identity-service/go.sum ./
RUN go mod download

# Stage 2: Development (Live Reloading)
FROM base AS dev
RUN go install github.com/air-verse/air@v1.61.7
COPY identity-service/ .
CMD ["air", "-c", ".air.toml"]

# Stage 3: Builder (Production Compilation)
FROM base AS builder
COPY identity-service/ .
RUN CGO_ENABLED=0 GOOS=linux go build -o /identity-api ./cmd/api/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -o /identity-worker ./cmd/worker/main.go

# Stage 4a: Final API image
FROM alpine:3.21 AS api
WORKDIR /app
RUN apk add --no-cache postgresql-client
COPY --from=base /go/bin/migrate /usr/local/bin/migrate
COPY --from=builder /app/internal/infrastructure/migrations ./migrations
COPY --from=builder /identity-api .
EXPOSE 8080
CMD ["./identity-api"]

# Stage 4b: Final worker image
FROM alpine:3.21 AS worker
WORKDIR /app
RUN apk add --no-cache postgresql-client
COPY --from=base /go/bin/migrate /usr/local/bin/migrate
COPY --from=builder /app/internal/infrastructure/migrations ./migrations
COPY --from=builder /identity-worker .
CMD ["./identity-worker"]
