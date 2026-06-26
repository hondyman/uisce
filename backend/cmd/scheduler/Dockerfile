FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build scheduler binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /scheduler ./cmd/scheduler

# Final image
FROM alpine:3.19

RUN apk update && apk --no-cache add --allow-untrusted ca-certificates tzdata

WORKDIR /app

# Copy binary from builder
COPY --from=builder /scheduler /app/scheduler

# Create non-root user
RUN adduser -D -g '' appuser
USER appuser

# Default environment variables
ENV TEMPORAL_HOST=temporal:7233 \
    TEMPORAL_NAMESPACE=default \
    DB_PRIMARY_HOST=localhost \
    DB_PRIMARY_PORT=5432 \
    DB_NAME=semlayer \
    DB_SSL_MODE=require \
    REGION=us-east \
    LOG_LEVEL=info

ENTRYPOINT ["/app/scheduler"]
