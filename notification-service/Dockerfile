# Stage 1: Base (Dependencies & Tools)
FROM golang:1.24 AS base
WORKDIR /app

COPY notification-service/go.mod notification-service/go.sum ./
RUN go mod download

# Stage 2: Development (Live Reloading)
FROM base AS dev
RUN go install github.com/air-verse/air@v1.61.7
COPY notification-service/ .
CMD ["air", "-c", ".air.toml"]

# Stage 3: Builder (Production Compilation)
FROM base AS builder
COPY notification-service/ .
RUN CGO_ENABLED=0 GOOS=linux go build -o /notification-service ./cmd/main.go

# Stage 4: Final (Production Image)
FROM alpine:3.21
WORKDIR /app

RUN apk add --no-cache ca-certificates
COPY --from=builder /notification-service .
COPY --from=builder /app/internal/infrastructure/notification/email/templates ./internal/infrastructure/notification/email/templates/

EXPOSE 8080
CMD ["./notification-service"]
