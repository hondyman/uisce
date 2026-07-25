FROM golang:1.25.3-alpine AS builder

RUN apk add --no-cache git
WORKDIR /src

COPY . .
RUN go mod download

# Build wealth-management binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/wealth-management ./services/wealth-management

FROM alpine:3.18
RUN apk add --no-cache ca-certificates curl
WORKDIR /app

COPY --from=builder /app/wealth-management /app/wealth-management

EXPOSE 8081
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
	CMD curl -f http://localhost:8081/health || exit 1

CMD ["/app/wealth-management"]
