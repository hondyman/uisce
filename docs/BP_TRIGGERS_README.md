# BP Triggers - Local Dev Quickstart

This document shows how to run a minimal local dev stack for Business Process (BP) triggers. It brings up Temporal, Hasura, Postgres (for Hasura and Temporal), RabbitMQ, and the Temporal UI with docker-compose. After starting services, run the Go worker and trigger engine locally with the `bp_versioned` build tag.

Prerequisites
- Docker and docker-compose installed
- Go 1.23+ (the repo uses go.work to scope modules)

Start services

```bash
# from repo root
docker compose -f docker-compose.workflows.local.yml up -d
```

Services exposed locally
- Hasura console: http://localhost:8080 (admin secret: `dev-secret`)
- Temporal gRPC: localhost:7233
- Temporal UI: http://localhost:8088
- RabbitMQ: http://localhost:15672 (guest/guest)

Run backend worker & trigger engine (versioned handler)

Open a terminal and run the worker and triggers runner with the `bp_versioned` build tag so the versioned handler is active.

```bash
# Build and run worker
go run -tags bp_versioned ./backend/cmd/worker

# In another terminal: run trigger engine
go run -tags bp_versioned ./backend/cmd/triggers
```

Notes
- The compose file uses Postgres for both Hasura and Temporal persistence on ports 5433 and 5434 respectively so it won't collide with any local Postgres on 5432. Adjust ports as needed.
- If you want to explore the Hasura GraphQL schema, open the console and add appropriate metadata or migrations.
- If you need a complete end-to-end demo, I can wire the TriggerEngine to call Temporal ExecuteWorkflow and implement a minimal workflow/activity that logs execution and writes to `bp_trigger_executions`.
# Business Process (BP) Triggers - Quick Start

This document provides a quick path to get the BP Triggers feature running locally for development.

1) Apply DB migration

  psql "postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable" -f backend/db/migrations/2025_10_21_create_bp_triggers.sql

2) Start the Trigger Engine (development)

  # from repository root
  cd backend
  go run ./cmd/triggers

3) Start the Temporal worker (development)

  cd backend
  go run ./cmd/worker

4) Start the frontend and use the BP Trigger Builder

  cd frontend
  npm install
  npm start

5) Test a sample trigger via curl (example)

  curl -X POST http://localhost:8080/api/triggers \
    -H 'Content-Type: application/json' \
    -d '{"trigger_name":"VIP Order Escalation","trigger_type":"conditional","target_process_id":"hire-bp","condition_config":{"type":"AND","children":[{"field":"customer.tier","operator":"eq","value":"VIP"}]},"escalation_config":{"levels":[{"level":1,"delay_hours":24,"assignee":"manager"}]}}'

Notes:
- The Go files in `backend/internal/triggers` and `backend/internal/workflows` are skeletons. Add connection strings and wiring to `cmd/triggers` and `cmd/worker` as needed.
- Temporal and RabbitMQ/AMQP clients must be available in your Go modules (go.mod). If missing, run `go get go.temporal.io/sdk` and `go get github.com/streadway/amqp`.
- The BP Trigger metrics materialized view should be refreshed regularly (cron or after executions).
