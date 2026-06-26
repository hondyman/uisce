FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum files
# In a real monorepo/polyrepo setup, you might need to adjust this
# For this structure, we assume we are building from the root context
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY . .

# Build argument to specify which service to build
ARG SERVICE_PATH
RUN go build -o /app/service ./apps/${SERVICE_PATH}

# Final stage
FROM alpine:latest

WORKDIR /app
COPY --from=builder /app/service /app/service

# Expose port (can be overridden)
EXPOSE 8080

CMD ["/app/service"]
