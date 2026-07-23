FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
COPY libs ./libs
RUN go mod download

COPY . .

RUN go build -o cdc_service ./backend/cmd/cdc_service/main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/cdc_service .

CMD ["./cdc_service"]
