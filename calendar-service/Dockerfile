# Stage 1: Builder
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build the application
# CGO_ENABLED=0 for static binary
# GOOS=linux GOARCH=amd64 for cross-compilation
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -a -installsuffix cgo \
    -ldflags="-w -s -X main.version=$(git describe --tags --always --dirty) \
    -X main.buildTime=$(date -u '+%Y-%m-%dT%H:%M:%SZ') \
    -X main.gitCommit=$(git rev-parse HEAD)" \
    -o bin/calendar-service ./cmd/server

# Stage 2: Security Scanner (optional but recommended)
FROM aquasec/trivy:latest AS scanner

COPY --from=builder /app/bin/calendar-service /app/calendar-service

# Scan for vulnerabilities
RUN trivy fs --exit-code 0 --severity HIGH,CRITICAL /app

# Stage 3: Runtime
FROM alpine:3.18

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata curl

# Create non-root user
RUN addgroup -S -g 10001 calendar && adduser -S -u 10001 -G calendar calendar

# Set working directory
WORKDIR /home/calendar

# Copy CA certificates from builder for HTTPS
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy timezone data
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy binary from builder
COPY --from=builder /app/bin/calendar-service .

# Change ownership
RUN chown -R calendar:calendar /home/calendar

# Switch to non-root user
USER calendar

# Expose ports
EXPOSE 8080 9090

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
  CMD curl -f http://localhost:8080/health || exit 1

# Run application
ENTRYPOINT ["./calendar-service"]
CMD []
