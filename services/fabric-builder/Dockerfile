FROM golang:1.25.3-alpine AS builder

RUN apk add --no-cache git
WORKDIR /src

# Copy repo and download modules
COPY . .
RUN go mod download

# Build fabric-builder binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/fabric-builder ./services/fabric-builder

FROM alpine:3.18
RUN apk add --no-cache ca-certificates curl
WORKDIR /app

COPY --from=builder /app/fabric-builder /app/fabric-builder

EXPOSE 8081
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
	CMD curl -f http://localhost:8081/health || exit 1

CMD ["/app/fabric-builder"]
