# Multi-stage build for Go backend
FROM golang:1.21-alpine AS backend-builder

RUN apk add --no-cache git

WORKDIR /app

# Copy go mod files
COPY backend/go.mod backend/go.sum ./
RUN go mod download

# Copy backend source
COPY backend/ ./

# Build backend
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server

# Frontend build stage
FROM node:18-alpine AS frontend-builder

WORKDIR /app

# Copy frontend files
COPY frontend/package*.json ./
RUN npm install

COPY frontend/ ./
RUN npm run build

# Final runtime stage
FROM alpine:latest

RUN apk update && apk --no-cache add --allow-untrusted ca-certificates

WORKDIR /root/

# Copy backend binary
COPY --from=backend-builder /app/server .

# Copy frontend build
COPY --from=frontend-builder /app/dist ./frontend/dist

# Copy migrations
COPY migrations ./migrations

# Expose port
EXPOSE 8080

# Run
CMD ["./server"]
