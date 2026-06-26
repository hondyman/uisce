.PHONY: build run clean validate tidy

run:
	go run ./

validate:
	go run scripts/validate_datasource.go

build:
	go build -o gonja-service ./

tidy:
	go mod tidy
