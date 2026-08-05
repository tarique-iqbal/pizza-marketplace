# Stage 1: Base (Dependencies & Tools)
FROM golang:1.24 AS base
WORKDIR /app

COPY email-service/go.mod email-service/go.sum ./
RUN go mod download

# Stage 2: Development (Live Reloading)
FROM base AS dev
RUN go install github.com/air-verse/air@v1.61.7
COPY email-service/ .
CMD ["air", "-c", ".air.toml"]

# Stage 3: Builder (Production Compilation)
FROM base AS builder
COPY email-service/ .
RUN CGO_ENABLED=0 GOOS=linux go build -o /email-service ./cmd/main.go

# Stage 4: Final (Production Image)
FROM alpine:3.21
WORKDIR /app

RUN apk add --no-cache ca-certificates
COPY --from=builder /email-service .
COPY --from=builder /app/internal/infrastructure/email/templates ./internal/infrastructure/email/templates/

EXPOSE 8080
CMD ["./email-service"]
