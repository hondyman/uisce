# Stage 1: Builder
FROM golang:1.24 AS builder

WORKDIR /app

# Install build dependencies for librdkafka
RUN apt-get update && apt-get install -y --no-install-recommends \
    git \
    build-essential \
    pkg-config \
    librdkafka-dev \
    && rm -rf /var/lib/apt/lists/*

# Copy all local module dependencies needed by replace directives
COPY go.mod go.sum ./
COPY backend/go.mod backend/go.sum ./backend/
COPY libs/ ./libs/
COPY calc-engine/ ./calc-engine/

# Download dependencies
WORKDIR /app/backend
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

# Copy backend source code
WORKDIR /app
COPY backend/ ./backend/

# Build the application from backend directory
WORKDIR /app/backend
# Tidy modules and build with GOWORK=off to ignore workspace file
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    GOWORK=off go mod tidy

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=1 GOOS=linux GOWORK=off go build -ldflags="-w -s" -o /app/server ./cmd/server/main.go

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=1 GOOS=linux GOWORK=off go build -ldflags="-w -s" -o /app/worker ./cmd/worker/main.go

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=1 GOOS=linux GOWORK=off go build -ldflags="-w -s" -o /app/audit-consumer ./cmd/audit-consumer/main.go

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=1 GOOS=linux GOWORK=off go build -ldflags="-w -s" -o /app/migrate ./cmd/migrate/migrate.go ./cmd/migrate/migrate_audit.go ./cmd/migrate/migrate_engagement_notifications.go

# Stage 2: Runner
FROM debian:bookworm-slim

# Install runtime dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    librdkafka1 \
    tzdata \
    curl \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy binaries from builder
COPY --from=builder /app/server .
COPY --from=builder /app/worker .
COPY --from=builder /app/audit-consumer .
COPY --from=builder /app/migrate .

# Copy config and migrations
COPY --from=builder /app/backend/config.yaml ./config.yaml
COPY --from=builder /app/backend/migrations ./migrations
COPY --from=builder /app/backend/docker-entrypoint.sh ./docker-entrypoint.sh

# Make entrypoint executable
RUN chmod +x ./docker-entrypoint.sh

# Expose ports
EXPOSE 8080

# Use entrypoint script to run migrations before starting server
ENTRYPOINT ["/app/docker-entrypoint.sh"]
