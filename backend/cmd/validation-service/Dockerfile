# Multi-stage build for Validation Service
FROM golang:1.25.3-alpine AS builder

WORKDIR /app

# Copy full repository so local replace directives are available during module download
COPY . .

# Download dependencies
RUN go mod download

# Build the validation service
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo \
    -o validation-service \
    ./backend/cmd/validation-service

# Runtime stage
FROM alpine:3.18

WORKDIR /app

# Install ca-certificates for HTTPS and postgresql client
RUN apk add --no-cache ca-certificates postgresql-client curl

# Copy binary from builder
COPY --from=builder /app/validation-service .

# Health check
HEALTHCHECK --interval=10s --timeout=5s --retries=3 \
    CMD curl -f http://localhost:8082/health || exit 1

# Run the service
CMD ["./validation-service"]
