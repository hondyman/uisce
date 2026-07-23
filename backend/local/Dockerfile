# Multi-stage Dockerfile for the local proxy binary

FROM golang:1.25.3-alpine AS builder

# Build inside the `backend` module directory so go module paths resolve correctly
WORKDIR /app

# Copy the whole repository so local replace directives (../calc-engine) resolve
COPY . .

# Switch to backend module directory and download dependencies
WORKDIR /app/backend
# Use explicit GOPROXY and retry loop to make builds resilient to transient network issues
ENV GOPROXY=https://proxy.golang.org,direct
RUN set -eux; \
    for i in 1 2 3 4 5; do \
      echo "Attempting go mod download (try $i)..."; \
      if go mod download; then \
        break; \
      fi; \
      echo "go mod download failed, retrying in 5s..."; \
      sleep 5; \
    done

# Build static binary (disable CGO for portability). Build from backend module root.
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/proxy ./local/cmd/proxy

FROM alpine:3.18
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /app/proxy .
EXPOSE 29080
ENTRYPOINT ["/app/proxy"]
