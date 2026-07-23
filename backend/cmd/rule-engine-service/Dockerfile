# Multi-stage build for Rule Engine Service
FROM golang:1.25.3-alpine AS builder

WORKDIR /app

# Copy the whole repository into the build context so local replace directives
# (for example ./libs/temporal-client) resolve correctly during `go mod download`.
COPY . .

# Download modules (cache layer when go.mod/go.sum unchanged)
RUN go mod download

# Ensure C toolchain is available for CGO builds on Alpine
RUN apk add --no-cache build-base musl-dev gcc

RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo \
    -o rule-engine-service \
    ./backend/cmd/rule-engine-service

# Runtime stage
FROM alpine:3.18

WORKDIR /app

RUN apk add --no-cache ca-certificates postgresql-client curl

COPY --from=builder /app/rule-engine-service .

HEALTHCHECK --interval=10s --timeout=5s --retries=3 \
    CMD curl -f http://localhost:8083/health || exit 1

CMD ["./rule-engine-service"]
