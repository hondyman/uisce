## Makefile -- small helper targets

.PHONY: health
health:
	@./scripts/health_check.sh

.PHONY: ci-health
ci-health:
	@echo "Run health_check.sh in CI (expects docker compose to be available)"

.PHONY: migrate
migrate:
	@echo "Running backend migration runner"
	@cd backend && go run ./cmd/migrate

.PHONY: start-proxy
start-proxy:
	@echo "Starting local proxy on :29080 (ctrl-c to stop)"
	@cd backend && go run ./local/cmd/proxy

.PHONY: local-up
local-up:
	@echo "Starting local minimal stack (hasura + rule-engine + proxy) via docker-compose.local.yml"
	@docker compose -f docker-compose.local.yml up -d --build

.PHONY: local-down
local-down:
	@echo "Stopping local minimal stack"
	@docker compose -f docker-compose.local.yml down

.PHONY: local-logs
local-logs:
	@docker compose -f docker-compose.local.yml logs -f

.PHONY: backup-pgapp
backup-pgapp:
	@echo "Dumping pgapp databases to ./backups/pgapp"
	@mkdir -p backups/pgapp
	@/Applications/Postgres.app/Contents/Versions/17/bin/pg_dump -Fc -f backups/pgapp/alpha.dump -U postgres -h localhost -p 5433 alpha || true
	@/Applications/Postgres.app/Contents/Versions/17/bin/pg_dump -Fc -f backups/pgapp/alpha_dwh.dump -U postgres -h localhost -p 5433 alpha_dwh || true
	@/Applications/Postgres.app/Contents/Versions/17/bin/pg_dump -Fc -f backups/pgapp/northwinds.dump -U postgres -h localhost -p 5433 northwinds || true
	@/Applications/Postgres.app/Contents/Versions/17/bin/pg_dump -Fc -f backups/pgapp/northwinds_gold.dump -U postgres -h localhost -p 5433 northwinds_gold || true

.PHONY: migrate-sql
migrate-sql:
	@echo "Run a specific SQL migration file against local POSTGRES_URL environment"
	@test -n "$$POSTGRES_URL" || (echo "Please set POSTGRES_URL, e.g. postgres://user:pass@localhost:5432/dbname?sslmode=disable" && exit 1)
	@psql "$$POSTGRES_URL" -f backend/migrations/000023_seed_private_markets_bundles.sql

.PHONY: sync-cube-tenants
sync-cube-tenants:
	@echo "Syncing Cube tenant scopes from Postgres"
	@go run ./scripts/sync_cube_tenants.go


## Docker-compose helper targets (backend)
.PHONY: up up-minimal down logs migrate-runner shell

COMPOSE_FILE := docker-compose.local.yml

up:
	@echo "Starting full backend stack (may build several images)"
	@docker compose -f $(COMPOSE_FILE) up -d

up-minimal:
	@echo "Starting minimal backend services (hasura, rabbitmq, backend)"
	@./scripts/docker-start.sh up-minimal

down:
	@docker compose -f $(COMPOSE_FILE) down

logs:
	@./scripts/docker-start.sh logs

migrate-runner:
	@./scripts/migrate.sh

shell:
	@docker compose -f $(COMPOSE_FILE) run --rm runner "sh"
