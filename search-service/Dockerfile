# Stage 1: Base (Dependencies & Tools)
FROM golang:1.24 AS base
WORKDIR /app

COPY search-service/go.mod search-service/go.sum ./
RUN go mod download

# Stage 2: Development (Live Reloading)
FROM base AS dev
RUN go install github.com/air-verse/air@v1.61.7
COPY search-service/ .
CMD ["air", "-c", ".air.toml"]

# Stage 3: Builder (Production Compilation)
FROM base AS builder
COPY search-service/ .
RUN CGO_ENABLED=0 GOOS=linux go build -o /search-api ./cmd/api/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -o /search-worker ./cmd/worker/main.go

# Stage 4a: Final API image
FROM alpine:3.21 AS api
WORKDIR /app
RUN apk add --no-cache ca-certificates
COPY --from=builder /search-api .
EXPOSE 8080
CMD ["./search-api"]

# Stage 4b: Final worker image
FROM alpine:3.21 AS worker
WORKDIR /app
RUN apk add --no-cache ca-certificates
COPY --from=builder /search-worker .
CMD ["./search-worker"]
