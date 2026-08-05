# Stage 1: Base (Dependencies & Tools)
FROM golang:1.24 AS base
WORKDIR /app

RUN go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@v4.19.1

COPY restaurant-service/go.mod restaurant-service/go.sum ./
RUN go mod download

# Stage 2: Development (Live Reloading)
FROM base AS dev
RUN go install github.com/air-verse/air@v1.61.7
COPY restaurant-service/ .
CMD ["air", "-c", ".air.toml"]

# Stage 3: Builder (Production Compilation)
FROM base AS builder
COPY restaurant-service/ .
RUN CGO_ENABLED=0 GOOS=linux go build -o /restaurant-service ./cmd/api/main.go

# Stage 4: Final (Production Image)
FROM alpine:3.21
WORKDIR /app

RUN apk add --no-cache ca-certificates postgresql-client
COPY --from=base /go/bin/migrate /usr/local/bin/migrate
COPY --from=builder /restaurant-service .

EXPOSE 8080
CMD ["./restaurant-service"]
