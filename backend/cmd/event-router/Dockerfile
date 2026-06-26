# Build stage
FROM golang:1.25.3-alpine AS builder

WORKDIR /app

# ✅ Install Git for go mod to work
RUN apk add --no-cache git

# Copy the full repository so local module replace directives (../libs, ./libs) resolve
COPY . .

# Download dependencies
RUN go mod download

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o event-router .

# Runtime stage
FROM alpine:3.18

# Install curl for healthchecks
RUN apk --no-cache add curl

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/event-router .

EXPOSE 8081

CMD ["./event-router"]
