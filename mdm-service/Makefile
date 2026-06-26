.PHONY: help build test run docker-build docker-run clean migrate

# Variables
BINARY_NAME=mdm-service
IMAGE_NAME=semlayer/mdm-calendar-service
VERSION=$(shell git describe --tags --always 2>/dev/null || echo "latest")
GO=go
GOFLAGS=-v
PORT?=8080
DATABASE_URL?=postgres://postgres:password@localhost:5432/semlayer

help:
	@echo "MDM Calendar Service - Makefile targets:"
	@echo ""
	@echo "Build targets:"
	@echo "  make build              - Build Go binary"
	@echo "  make test               - Run unit tests"
	@echo "  make docker-build       - Build Docker image"
	@echo ""
	@echo "Deploy targets:"
	@echo "  make docker-run         - Run Docker container locally"
	@echo "  make run                - Run service locally"
	@echo ""
	@echo "Database targets:"
	@echo "  make migrate            - Apply database migrations"
	@echo "  make migrate-down       - Rollback database"
	@echo "  make migrate-fresh      - Fresh database setup"
	@echo ""
	@echo "Cleanup targets:"
	@echo "  make clean              - Remove built artifacts"
	@echo "  make docker-clean       - Remove Docker image"

# Build the Go binary
build: generate
	@echo "Building $(BINARY_NAME)..."
	$(GO) build $(GOFLAGS) -o ./bin/$(BINARY_NAME) ./cmd/mdm-service/main.go

# Generate code (gqlgen, mocks, etc.)
generate:
	$(GO) generate ./...

# Run tests
test:
	@echo "Running tests..."
	$(GO) test -v -race -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Run locally
run: build
	@echo "Starting $(BINARY_NAME) on port $(PORT)..."
	DATABASE_URL=$(DATABASE_URL) PORT=$(PORT) ./bin/$(BINARY_NAME)

# Docker build
docker-build:
	@echo "Building Docker image $(IMAGE_NAME):$(VERSION)..."
	docker build \
		--build-arg VERSION=$(VERSION) \
		-t $(IMAGE_NAME):$(VERSION) \
		-t $(IMAGE_NAME):latest \
		.

# Docker run locally
docker-run: docker-build
	@echo "Running Docker container..."
	docker run \
		-p $(PORT):8080 \
		-e DATABASE_URL=$(DATABASE_URL) \
		-e PORT=8080 \
		-e LOG_LEVEL=info \
		$(IMAGE_NAME):latest

# Database migrations
migrate:
	@echo "Applying migrations to $(DATABASE_URL)..."
	psql $(DATABASE_URL) -f ../database/migrations/mdm_calendar_schema.sql
	@echo "Migrations complete!"

migrate-fresh:
	@echo "Dropping and recreating database..."
	psql $(DATABASE_URL) -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"
	$(MAKE) migrate

migrate-down:
	@echo "Rolling back schema (not implemented)"

# Code quality
lint:
	@echo "Running linter..."
	golangci-lint run ./...

fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...

vet:
	@echo "Running go vet..."
	$(GO) vet ./...

# Clean up
clean:
	@echo "Cleaning build artifacts..."
	rm -rf ./bin
	rm -f coverage.out coverage.html
	$(GO) clean

docker-clean:
	@echo "Removing Docker image..."
	docker rmi $(IMAGE_NAME):$(VERSION) $(IMAGE_NAME):latest 2>/dev/null || true

# Benchmarks
bench:
	@echo "Running benchmarks..."
	$(GO) test -bench=. -benchmem ./...

# Integration tests (requires running services)
integration-test:
	@echo "Running integration tests..."
	$(GO) test -v -tags=integration ./...

# Dependencies
deps:
	@echo "Downloading dependencies..."
	$(GO) mod download
	$(GO) mod tidy

# Development setup
dev-setup: deps migrate
	@echo "Development environment ready!"

.DEFAULT_GOAL := help
