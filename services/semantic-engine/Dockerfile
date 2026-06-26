# Semantic Engine Service
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o semantic-engine ./cmd/main.go

# Final stage
FROM alpine:latest

RUN apk update && apk --no-cache add --allow-untrusted ca-certificates
WORKDIR /root/

COPY --from=builder /app/semantic-engine .

EXPOSE 8083

CMD ["./semantic-engine"]