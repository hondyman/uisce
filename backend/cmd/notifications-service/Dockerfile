# Multi-stage build for Notifications Service
FROM golang:1.25.3-alpine AS builder

WORKDIR /app

# Copy full repository so local module replacements resolve during module download
COPY . .

# Download modules (this layer will be cached when go.mod/go.sum don't change)
RUN go mod download

RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo \
    -o notifications-service \
    ./backend/cmd/notifications-service

# Runtime stage
FROM alpine:3.18

WORKDIR /app

RUN apk add --no-cache ca-certificates postgresql-client curl

COPY --from=builder /app/notifications-service .

HEALTHCHECK --interval=10s --timeout=5s --retries=3 \
    CMD curl -f http://localhost:8084/health || exit 1

CMD ["./notifications-service"]
