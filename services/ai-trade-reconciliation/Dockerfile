# Multi-stage build for AI Trade Reconciliation Backend
# Stage 1: Builder

ARG GO_VERSION=1.24
FROM golang:${GO_VERSION}-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git gcc musl-dev

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -o atr-service \
    ./backend/cmd/main.go

# Stage 2: Runtime
FROM alpine:latest

# Install runtime dependencies
RUN apk update && apk --no-cache add --allow-untrusted ca-certificates curl postgresql-client

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/atr-service .

# Create directories for logs and data
RUN mkdir -p /app/logs /app/data

# Health check
HEALTHCHECK --interval=10s --timeout=5s --retries=5 \
    CMD curl -f http://localhost:8080/health || exit 1

# Expose port
EXPOSE 8080

# Run the service
CMD ["./atr-service"]
