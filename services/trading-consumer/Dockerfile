FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git gcc musl-dev

# Copy go.mod and go.sum if they exist
COPY go.mod go.sum* ./

# Download dependencies
RUN go mod download || true

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o consumer main.go

# Final image
FROM alpine:3.18

RUN apk add --no-cache ca-certificates

COPY --from=builder /app/consumer /usr/local/bin/consumer

ENV REDPANDA_BROKERS=redpanda:9092
ENV CONSUMER_GROUP_ID=trading-platform-consumer

HEALTH CHECK --interval=10s --timeout=3s --start-period=5s --retries=3 \
    CMD nc -z localhost 8080 || exit 1

ENTRYPOINT ["consumer"]
