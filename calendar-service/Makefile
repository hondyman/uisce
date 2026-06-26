.PHONY: help dev migrate test test-unit test-integration test-e2e clean build docker-up docker-down logs lint fmt deps health seed docker-restart

# Color output
BLUE := \033[0;34m
GREEN := \033[0;32m
YELLOW := \033[0;33m
NC := \033[0m # No Color

help:
	@echo "$(BLUE)📖 Epic 31 Calendar Service - Makefile Commands$(NC)"
	@echo ""
	@echo "$(YELLOW)Development$(NC):"
	@echo "  $(GREEN)make dev$(NC)              - Start all services (Docker + Go)"
	@echo "  $(GREEN)make migrate$(NC)          - Run database migrations + seed"
	@echo "  $(GREEN)make logs$(NC)             - View Docker Compose logs"
	@echo "  $(GREEN)make docker-restart$(NC)   - Restart Docker services"
	@echo ""
	@echo "$(YELLOW)Testing$(NC):"
	@echo "  $(GREEN)make test$(NC)             - Run all tests (unit + integration)"
	@echo "  $(GREEN)make test-unit$(NC)        - Run unit tests only"
	@echo "  $(GREEN)make test-integration$(NC) - Run integration tests"
	@echo "  $(GREEN)make test-e2e$(NC)         - Run E2E tests"
	@echo ""
	@echo "$(YELLOW)Build & Deploy$(NC):"
	@echo "  $(GREEN)make build$(NC)            - Build binary"
	@echo "  $(GREEN)make clean$(NC)            - Clean build artifacts"
	@echo "  $(GREEN)make docker-up$(NC)        - Start Docker Compose"
	@echo "  $(GREEN)make docker-down$(NC)      - Stop Docker Compose"
	@echo ""
	@echo "$(YELLOW)Utility$(NC):"
	@echo "  $(GREEN)make lint$(NC)             - Run linter (golangci-lint)"
	@echo "  $(GREEN)make fmt$(NC)              - Format code"
	@echo "  $(GREEN)make deps$(NC)             - Download dependencies"
	@echo "  $(GREEN)make health$(NC)           - Check service health"
	@echo "  $(GREEN)make seed$(NC)             - Run database seed only"
	@echo "  $(GREEN)make help$(NC)             - Show this help"

# ===========================
# DEVELOPMENT
# ===========================

dev: docker-up
	@echo "$(BLUE)🚀 Starting Calendar Service...$(NC)"
	@if [ -f .env ]; then \
		export $$(cat .env | grep -v '^#' | xargs); \
	fi
	@go run ./cmd/server/main.go

migrate:
	@echo "$(BLUE)📦 Running database migrations...$(NC)"
	@if [ ! -f ./scripts/migrate.sh ]; then \
		echo "$(YELLOW)⚠️  migrate.sh not found$(NC)"; \
		exit 1; \
	fi
	@chmod +x ./scripts/migrate.sh
	@./scripts/migrate.sh

seed:
	@echo "$(BLUE)🌱 Seeding test data...$(NC)"
	@chmod +x ./scripts/migrate.sh
	@./scripts/migrate.sh | tail -20

# ===========================
# TESTING
# ===========================

test: test-unit test-integration
	@echo "$(GREEN)✅ All tests passed!$(NC)"

test-unit:
	@echo "$(BLUE)🧪 Running unit tests...$(NC)"
	@go test -v ./internal/... -short -race -cover -coverprofile=coverage.out

test-integration:
	@echo "$(BLUE)🔗 Running integration tests...$(NC)"
	@go test -v ./internal/... -tags=integration -race -cover -timeout 10m

test-e2e:
	@echo "$(BLUE)🎯 Running E2E tests...$(NC)"
	@if [ -f ./tests/e2e/main.go ]; then \
		go run ./tests/e2e/main.go; \
	else \
		echo "$(YELLOW)⚠️  E2E tests not found at ./tests/e2e/main.go$(NC)"; \
	fi

# ===========================
# DOCKER COMPOSE
# ===========================

docker-up:
	@echo "$(BLUE)🐳 Starting Docker Compose services...$(NC)"
	@if [ ! -f docker-compose.yml ] && [ ! -f docker-compose.local.yml ]; then \
		echo "$(YELLOW)⚠️  docker-compose.yml not found$(NC)"; \
		exit 1; \
	fi
	@if [ -f docker-compose.yml ]; then \
		docker-compose -f docker-compose.yml up -d; \
	fi
	@if [ -f docker-compose.local.yml ]; then \
		docker-compose -f docker-compose.local.yml up -d; \
	fi
	@echo "$(YELLOW)⏳ Waiting for services to be ready...$(NC)"
	@sleep 10
	@echo "$(GREEN)✅ Services started!$(NC)"

docker-down:
	@echo "$(BLUE)🛑 Stopping Docker Compose services...$(NC)"
	@if [ -f docker-compose.yml ]; then \
		docker-compose -f docker-compose.yml down; \
	fi
	@if [ -f docker-compose.local.yml ]; then \
		docker-compose -f docker-compose.local.yml down; \
	fi

docker-restart: docker-down docker-up
	@echo "$(GREEN)✅ Docker services restarted!$(NC)"

# ===========================
# BUILD & CLEAN
# ===========================

build:
	@echo "$(BLUE)🔨 Building Calendar Service...$(NC)"
	@mkdir -p bin
	@go build -v -o bin/calendar-service ./cmd/server/main.go
	@echo "$(GREEN)✅ Build complete: bin/calendar-service$(NC)"

clean:
	@echo "$(BLUE)🧹 Cleaning up...$(NC)"
	@rm -rf bin/
	@rm -f coverage.out coverage.html
	@go clean -cache -testcache
	@echo "$(GREEN)✅ Clean complete!$(NC)"

# ===========================
# CODE QUALITY
# ===========================

lint:
	@echo "$(BLUE)🔍 Running linter...$(NC)"
	@if command -v golangci-lint >/dev/null; then \
		golangci-lint run ./...; \
	else \
		echo "$(YELLOW)⚠️  golangci-lint not installed$(NC)"; \
		echo "Install: brew install golangci-lint"; \
	fi

fmt:
	@echo "$(BLUE)📝 Formatting code...$(NC)"
	@go fmt ./...
	@if command -v goimports >/dev/null; then \
		goimports -w .; \
	fi

deps:
	@echo "$(BLUE)📦 Installing dependencies...$(NC)"
	@go mod download
	@go mod tidy
	@echo "$(GREEN)✅ Dependencies updated!$(NC)"

# ===========================
# MONITORING & UTILITIES
# ===========================

health:
	@echo "$(BLUE)🏥 Checking service health...$(NC)"
	@curl -s http://localhost:8081/health | jq . 2>/dev/null || curl -s http://localhost:8081/health
	@echo ""

logs:
	@echo "$(BLUE)📋 Showing Docker Compose logs (Ctrl+C to exit)...$(NC)"
	@if [ -f docker-compose.local.yml ]; then \
		docker-compose -f docker-compose.local.yml logs -f; \
	elif [ -f docker-compose.yml ]; then \
		docker-compose -f docker-compose.yml logs -f; \
	else \
		echo "$(YELLOW)⚠️  docker-compose files not found$(NC)"; \
	fi

version:
	@echo "$(BLUE)📦 Calendar Service Build Info$(NC)"
	@echo "Go Version: $$(go version)"
	@echo "Module Path: $$(go list -m)"

# ===========================
# QUICK COMMANDS
# ===========================

ready: docker-up migrate
	@echo ""
	@echo "$(GREEN)🚀 System Ready!$(NC)"
	@echo ""
	@echo "Start service:"
	@echo "  $(BLUE)make dev$(NC)"
	@echo ""
	@echo "Or test endpoints immediately:"
	@echo "  $(BLUE)make health$(NC)"
	@echo ""

.DEFAULT_GOAL := help
	docker-compose -f docker-compose.local.yml up -d
	@echo "✓ Calendar Service started"
	@echo "Check health:"
	@echo "  - Calendar Service: curl http://localhost:8081/health"
	@echo "Note: Remote services configured in .env.local"

build:
	@echo "Building Calendar Service..."
	docker-compose -f docker-compose.local.yml build calendar-service
	@echo "✓ Build complete"

test:
	@echo "Running tests..."
	go test -v ./...

clean:
	@echo "Stopping Calendar Service..."
	docker-compose -f docker-compose.local.yml down -v
	@echo "✓ Calendar Service stopped"

logs:
	docker-compose -f docker-compose.local.yml logs -f

logs-service:
	docker-compose -f docker-compose.local.yml logs -f calendar-service

connector:
	@echo "Registering Debezium PostgreSQL connector..."
	@echo "Debezium runs as remote service."
	@echo "See docs/DEPLOYMENT.md for CDC setup instructions"

service-bash:
	docker-compose -f docker-compose.local.yml exec calendar-service sh

psql:
	@echo "Note: Run this against your remote Postgres:"
	@echo "psql postgresql://postgres:password@YOUR_REMOTE_HOST:5432/postgres"
