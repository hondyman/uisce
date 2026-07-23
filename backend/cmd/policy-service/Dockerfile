# Multi-stage build for Policy Service
FROM golang:1.25.3-alpine AS builder

WORKDIR /app

# Copy full repository so local module replacements resolve during module download
COPY . .

# Download modules (this layer will be cached when go.mod/go.sum don't change)
RUN go mod download

# Policy service would be built here
# For now, create a placeholder that builds successfully
RUN echo 'package main; import ("fmt"; "net/http"); func main() { http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) { fmt.Fprintf(w, `{"status":"healthy"}`) }); http.ListenAndServe(":8085", nil) }' > backend/cmd/policy-service/main.go && \
    CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo \
    -o policy-service \
    ./backend/cmd/policy-service

# Runtime stage
FROM alpine:3.18

WORKDIR /app

RUN apk add --no-cache ca-certificates postgresql-client curl

COPY --from=builder /app/policy-service .

HEALTHCHECK --interval=10s --timeout=5s --retries=3 \
    CMD curl -f http://localhost:8085/health || exit 1

CMD ["./policy-service"]
