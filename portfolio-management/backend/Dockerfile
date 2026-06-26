# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy full repository so local module replacements resolve during module download
COPY . .

# Download dependencies
RUN go mod download

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o notification-service ./cmd/main.go

# Final stage
FROM alpine:latest

RUN apk update && apk --no-cache add --allow-untrusted ca-certificates curl

WORKDIR /root/

# Copy binary from builder
COPY --from=builder /app/notification-service .

EXPOSE 8081

HEALTHCHECK --interval=10s --timeout=5s --retries=5 \
  CMD curl -f http://localhost:8081/health || exit 1

CMD ["./notification-service"]
