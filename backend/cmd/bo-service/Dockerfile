# Build Stage
FROM golang:1.25.3-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy full repository so local replace directives resolve during module download
COPY . .

# Download dependencies
RUN go mod download

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o /bo-service ./cmd/bo-service/main.go

# Runtime Stage
FROM alpine:latest

# Install runtime dependencies
RUN apk update && apk --no-cache add --allow-untrusted ca-certificates tzdata

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /bo-service .

# Create non-root user for security
RUN addgroup -g 1000 appgroup && \
    adduser -D -u 1000 -G appgroup appuser && \
    chown -R appuser:appgroup /app

USER appuser

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ps aux | grep "[b]o-service" || exit 1

# Run the service
CMD ["./bo-service"]
