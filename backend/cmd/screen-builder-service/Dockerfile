FROM golang:1.25.3-alpine AS builder

WORKDIR /src

# Copy the entire repository so module dependencies and internal packages
# resolve during module download.
COPY . .

RUN cd backend/cmd/screen-builder-service && go mod download

RUN cd backend/cmd/screen-builder-service && CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /screen-builder-service .

FROM alpine:latest

RUN apk update && apk --no-cache add --allow-untrusted ca-certificates

WORKDIR /root/

COPY --from=builder /screen-builder-service ./

EXPOSE 8083

CMD ["./screen-builder-service"]
