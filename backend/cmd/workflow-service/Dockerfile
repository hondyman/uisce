FROM golang:1.25.3-alpine AS builder

WORKDIR /src

# Copy the entire repository into the build context so module paths and
# replace directives resolve correctly during `go mod download`.
COPY . .

# Download modules (cache layer when go.mod/go.sum unchanged)
RUN cd backend/cmd/workflow-service && go mod download

# Build the workflow-service binary
RUN cd backend/cmd/workflow-service && CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /workflow-service .

FROM alpine:latest

RUN apk update && apk --no-cache add --allow-untrusted ca-certificates

WORKDIR /root/

COPY --from=builder /workflow-service .

EXPOSE 8082

CMD ["./workflow-service"]
