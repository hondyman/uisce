# /Users/eganpj/GitHub/semlayer/api-gateway/Dockerfile

# Stage 1: Build the Go binary in a dedicated build environment
FROM golang:1.25-alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum to leverage Docker's layer caching.
# This step is only re-run if these files change.
COPY go.mod go.sum ./
COPY libs ./libs
COPY api-gateway/go.mod api-gateway/go.sum ./api-gateway/
RUN cd api-gateway && go mod download

# Copy the api-gateway source code
COPY api-gateway/ ./api-gateway/

# Build the Go application, creating a statically linked binary.
# CGO_ENABLED=0 is important for building for a minimal Alpine base image.
RUN cd api-gateway && CGO_ENABLED=0 go build -o /api-gateway .

# Stage 2: Create the final, minimal production image
FROM alpine:latest

# Copy the compiled binary from the 'builder' stage
COPY --from=builder /api-gateway /api-gateway

# Copy static assets that the application serves
COPY --from=builder /app/api-gateway/openapi.yaml /openapi.yaml
COPY --from=builder /app/api-gateway/docs /docs

EXPOSE 8001

# The command to run when the container starts
CMD ["/api-gateway"]
