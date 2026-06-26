# 🚀 BP Triggers Quick Start Guide

## One-Command Setup (Development)

```bash
# 1. Start Docker services
docker compose -f docker-compose.workflows.local.yml up -d

# 2. Apply database schema
psql -h localhost -p 5435 -U postgres -d northwind < schema/bp_triggers.sql

# 3. Seed test data (optional, for E2E testing)
psql -h localhost -p 5435 -U postgres -d northwind << 'EOF'
INSERT INTO business_processes (id, tenant_id, process_name, description, lifecycle_state, escalation_threshold_mins)
VALUES ('11111111-1111-1111-1111-111111111111', '22222222-2222-2222-2222-222222222222', 'TestHireProcess', 'Test hiring workflow', 'active', 60);

INSERT INTO bp_steps (id, process_id, step_sequence, step_name, step_description, owner, estimated_duration_mins, escalation_threshold_mins, lifecycle_state)
VALUES 
('33333333-3333-3333-3333-333333333333', '11111111-1111-1111-1111-111111111111', 1, 'Draft', 'Initial draft', 'HR', 30, 15, 'active'),
('44444444-4444-4444-4444-444444444444', '11111111-1111-1111-1111-111111111111', 2, 'Review', 'Manager review', 'Manager', 30, 20, 'active'),
('55555555-5555-5555-5555-555555555555', '11111111-1111-1111-1111-111111111111', 3, 'Approve', 'Final approval', 'Director', 30, 25, 'active');

INSERT INTO bp_triggers (id, tenant_id, process_id, event_entity, event_action, conditions, trigger_description, workflow_payload)
VALUES (
  '66666666-6666-6666-6666-666666666666',
  '22222222-2222-2222-2222-222222222222',
  '11111111-1111-1111-1111-111111111111',
  'Employee',
  'created',
  '{"department": {"in": ["Engineering", "HR"]}}',
  'Start hire process for new employees in Engineering/HR',
  '{"escalation_level": 1}'
);
EOF

# 4. Build trigger engine
go build -tags bp_versioned -o ./bin/triggers ./backend/cmd/triggers

# 5. Start trigger engine (Terminal A)
DATABASE_URL="postgres://postgres:postgres@localhost:5435/northwind?sslmode=disable" ./bin/triggers

# 6. Send test event (Terminal B)
PGPASSWORD="postgres" psql -h localhost -p 5435 -U postgres -d northwind -c \
  "SELECT pg_notify('entity_events', json_build_object(
    'tenant_id', '22222222-2222-2222-2222-222222222222',
    'entity', 'Employee',
    'action', 'created',
    'entity_id', '99999999-9999-9999-9999-999999999999',
    'data', json_build_object('name', 'Test Employee', 'department', 'Engineering'),
    'timestamp', NOW()::text
  )::text);"

# 7. Verify execution
PGPASSWORD="postgres" psql -h localhost -p 5435 -U postgres -d northwind -c \
  "SELECT id, execution_status, completed_at FROM bp_trigger_executions ORDER BY executed_at DESC LIMIT 1;"
```

---

## Architecture Overview

### Component: Trigger Engine
- **Port**: 29090 (health checks)
- **Database**: PostgreSQL on 5435
- **Listener**: LISTEN entity_events (PostgreSQL notifications)
- **Workflow Orchestration**: Temporal SDK (mock mode if server unavailable)
- **Language**: Go
- **Build Tag**: `bp_versioned` (enables versioned timeout handler)

### Data Flow

```
Entity Created/Updated
         ↓
  pg_notify('entity_events', {...})
         ↓
PostgreSQL LISTEN channel
         ↓
TriggerEngine.StartEventListener()
         ↓
Load Trigger from DB
         ↓
Evaluate Conditions
         ↓
Match? YES → ExecuteWorkflow() → Record Execution
       NO  → Skip trigger
```

---

## Configuration

### Environment Variables

```bash
# Database connection (default: localhost:5432)
export DATABASE_URL="postgres://postgres:postgres@localhost:5435/northwind?sslmode=disable"

# Temporal server endpoint (default: localhost:7233)
export TEMPORAL_URL="localhost:7233"  # Or skip to run in test mode

# Optional: AMQP for message queue integration
export AMQP_URL="amqp://guest:guest@localhost:5672/"
```

### Docker Services

```bash
# Postgres on 5435
# RabbitMQ on 5672, 15672 (management UI)
# Temporal Server (optional, not in default compose)

# Start all services
docker compose -f docker-compose.workflows.local.yml up -d

# Stop all services
docker compose -f docker-compose.workflows.local.yml down

# View logs
docker compose -f docker-compose.workflows.local.yml logs -f triggers
```

---

## Testing

### Manual Test Event

```bash
# Send a simple test event
PGPASSWORD="postgres" psql -h localhost -p 5435 -U postgres -d northwind -c \
  "SELECT pg_notify('entity_events', json_build_object(
    'tenant_id', '22222222-2222-2222-2222-222222222222',
    'entity', 'Employee',
    'action', 'created',
    'entity_id', 'test-id-123',
    'timestamp', NOW()::text
  )::text);"
```

### Query Executions

```bash
# List all executions
PGPASSWORD="postgres" psql -h localhost -p 5435 -U postgres -d northwind -c \
  "SELECT id, execution_status, completed_at FROM bp_trigger_executions ORDER BY executed_at DESC LIMIT 10;"

# Find failed executions
PGPASSWORD="postgres" psql -h localhost -p 5435 -U postgres -d northwind -c \
  "SELECT id, error_message FROM bp_trigger_executions WHERE execution_status = 'failed';"

# Check trigger statistics
PGPASSWORD="postgres" psql -h localhost -p 5435 -U postgres -d northwind -c \
  "SELECT execution_status, COUNT(*) as count FROM bp_trigger_executions GROUP BY execution_status;"
```

---

## Health Checks

```bash
# Check trigger engine health
curl -s http://localhost:29090/health
# Expected: "ok"

# Check database connectivity
PGPASSWORD="postgres" psql -h localhost -p 5435 -U postgres -d northwind -c "SELECT 1;"

# Check Docker services
docker compose -f docker-compose.workflows.local.yml ps
```

---

## Troubleshooting

### Issue: "failed to create temporal client"
**Solution**: This is expected if Temporal server is not running. Engine enters test mode and logs workflow requests instead of executing them.

### Issue: "failed to connect to postgres"
**Solution**: Verify DATABASE_URL, check that Docker Postgres is running on port 5435.
```bash
docker compose -f docker-compose.workflows.local.yml ps
```

### Issue: "no triggers matching event"
**Solution**: 
1. Check that trigger exists: `SELECT * FROM bp_triggers WHERE event_entity = 'Employee' AND event_action = 'created';`
2. Verify trigger conditions JSON: `SELECT conditions FROM bp_triggers WHERE id = '...';`
3. Check trigger is not deleted: `SELECT lifecycle_state FROM bp_triggers WHERE id = '...';`

### Issue: Event received but not processed
**Solution**: Check engine logs for condition evaluation errors. Example log:
```
triggers: processing event for tenant 22222222-2222-2222-2222-222222222222: map[...]
triggers: executed 0 trigger(s)
```
This means event was received but no triggers matched. Check trigger definitions.

---

## Production Deployment

### With Temporal Server

1. **Set up Temporal** (locally, Docker, or cloud)
   ```bash
   # Point engine to Temporal server
   export TEMPORAL_URL="your-temporal-server:7233"
   ```

2. **Start Worker** (in separate process)
   ```bash
   go run -tags bp_versioned ./backend/cmd/worker
   ```

3. **Start Trigger Engine** (with real Temporal client)
   ```bash
   DATABASE_URL="..." go run -tags bp_versioned ./backend/cmd/triggers
   ```

### With Process Manager

```bash
# systemd service example
[Unit]
Description=BP Trigger Engine
After=network.target postgresql.service

[Service]
Type=simple
User=semlayer
Environment="DATABASE_URL=postgres://..."
Environment="TEMPORAL_URL=temporal-server:7233"
ExecStart=/usr/local/bin/triggers
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

---

## API Endpoints

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/health` | GET | Health check (returns "ok") |
| LISTEN entity_events | Subscribe | PostgreSQL LISTEN for events |

---

## Files Reference

- **Engine**: `backend/cmd/triggers/main.go` → `backend/internal/triggers/engine.go`
- **Workflows**: `backend/internal/workflows/dynamic_bp_workflow.go`
- **Schema**: `backend/db/migrations/2025_10_21_create_bp_triggers.sql`
- **Docker**: `docker-compose.workflows.local.yml`
- **Tests**: `scripts/test_bp_triggers.sh`

---

## Key Concepts

### Business Process (BP)
A sequence of steps (e.g., "Hire Process" = Draft → Review → Approve)

### BP Trigger
A rule that listens for events (e.g., "When Employee.created, start Hire Process")

### Event Matching
Trigger listens for specific entity + action + optional conditions

### Temporal Workflow
Orchestrates execution of BP steps with activities, retries, and timeouts

### Execution Audit
All trigger executions logged in `bp_trigger_executions` table for compliance

---

## Support

For detailed architecture and design decisions, see `docs/BP_TRIGGERS_E2E_TEST_SUCCESS.md`
