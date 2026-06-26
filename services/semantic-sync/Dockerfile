FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go mod files (they don't exist yet, so we'll create them in the container)
RUN go mod init semantic-sync || true

# Get dependencies
RUN go get github.com/lib/pq

# Copy source
COPY services/semantic-sync/main.go .

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -o semantic-sync .

# Runtime stage
FROM alpine:latest

RUN apk update && apk --no-cache add --allow-untrusted ca-certificates curl

WORKDIR /app

COPY --from=builder /app/semantic-sync .

# Create cube-schemas directory
RUN mkdir -p cube-schemas

EXPOSE 8000

HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:8000/health || exit 1

CMD ["./semantic-sync"]
