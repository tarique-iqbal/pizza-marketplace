# Stage 1: Base (Dependencies & Tools)
FROM golang:1.26 AS base
WORKDIR /app

RUN go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@v4.19.1

COPY customer-service/go.mod customer-service/go.sum ./
RUN go mod download

# Stage 2: Development (Live Reloading)
FROM base AS dev
RUN go install github.com/air-verse/air@v1.61.7
COPY customer-service/ .
CMD ["air", "-c", ".air.toml"]

# Stage 3: Builder (Production Compilation)
FROM base AS builder
COPY customer-service/ .
RUN CGO_ENABLED=0 GOOS=linux go build -o /customer-api ./cmd/api/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -o /customer-worker ./cmd/worker/main.go

# Stage 4a: Final API image
FROM alpine:3.21 AS api
WORKDIR /app
RUN apk add --no-cache ca-certificates postgresql-client
COPY --from=base /go/bin/migrate /usr/local/bin/migrate
COPY --from=builder /customer-api .
EXPOSE 8080
CMD ["./customer-api"]

# Stage 4b: Final worker image
FROM alpine:3.21 AS worker
WORKDIR /app
RUN apk add --no-cache ca-certificates postgresql-client
COPY --from=base /go/bin/migrate /usr/local/bin/migrate
COPY --from=builder /customer-worker .
CMD ["./customer-worker"]
