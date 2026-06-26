FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy main.go
COPY cmd/aiserver/main.go .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -o server main.go

FROM alpine:latest

RUN apk update && apk --no-cache add --allow-untrusted ca-certificates curl

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/server .

EXPOSE 8088

CMD ["./server"]
