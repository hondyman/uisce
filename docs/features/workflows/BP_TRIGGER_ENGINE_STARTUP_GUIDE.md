# 🚀 BP Trigger Engine - Complete Startup Guide

**Status**: ✅ READY TO START  
**All Components**: Temporal, PostgreSQL, Hasura, RabbitMQ, Worker, Trigger Engine

---

## 📋 Quick Reference

| Component | URL | Port | Purpose |
|-----------|-----|------|---------|
| **Temporal** | http://localhost:8080 | 8080 | Web UI for workflows |
| **Temporal gRPC** | localhost:7233 | 7233 | Worker connection |
| **Hasura** | http://localhost:8083 | 8083 | GraphQL API |
| **RabbitMQ** | http://localhost:15672 | 15672 | Message broker |
| **PostgreSQL** | localhost:5432 | 5432 | Main database |
| **Worker** | - | - | Your terminal |
| **Trigger Engine** | - | - | Your terminal |

---

## 🚀 ONE-COMMAND SETUP (Recommended)

### Run the automated setup script:

```bash
cd /Users/eganpj/GitHub/semlayer

# Make it executable (if needed)
chmod +x setup_trigger_engine.sh

# Run it
./setup_trigger_engine.sh
```

This script:
- ✅ Checks all prerequisites (Docker, Docker Compose, PostgreSQL, Go)
- ✅ Pulls latest Docker images
- ✅ Starts Temporal, PostgreSQL, Temporal UI, Hasura, RabbitMQ
- ✅ Waits for services to be healthy
- ✅ Builds Go services (worker, trigger engine)
- ✅ Displays service URLs and next steps

**Expected output:**
```
✅ Docker installed
✅ Docker Compose installed
✅ PostgreSQL client installed
✅ Go installed (go1.21.x)
✅ Temporal is running
✅ Hasura is running at http://localhost:8083
✅ RabbitMQ is running at http://localhost:15672
✅ Temporal UI is running at http://localhost:8080
✅ Worker built
✅ Trigger engine built

🎉 SETUP COMPLETE!
```

---

## ⚙️ MANUAL SETUP (If you prefer step-by-step)

### Step 1: Start Temporal Server (Locally)

Temporal is already installed via Homebrew. Start it in a separate terminal:

```bash
cd /Users/eganpj/GitHub/semlayer

# Run Temporal Server locally (development mode)
./start_temporal_locally.sh
```

**Expected output:**
```
🚀 Starting Temporal Server...
2025/10/21 22:30:00 Starting temporal server...
2025/10/21 22:30:01 Temporal is running at localhost:7233
```

This starts Temporal with SQLite backend (fine for development).

### Step 2: Start Temporal UI (Docker)

```bash
cd /Users/eganpj/GitHub/semlayer

# Start Temporal UI container
docker compose up -d temporal-ui

# Verify it's running
docker ps | grep temporal-ui
```

**Expected output:**
```
semlayer-temporal-ui    temporalio/ui:latest    Up ...    0.0.0.0:8081->8080/tcp
```

### Step 3: Start Other Services

```bash
# Start Hasura and RabbitMQ (optional but recommended)
docker compose up -d graphql-engine rabbitmq

# Verify services are running
docker ps | grep -E "graphql|rabbitmq"
```

```bash
# Check BP tables exist
psql postgresql://postgres:postgres@localhost/alpha -c "
SELECT tablename FROM pg_tables 
WHERE schemaname='public' AND tablename LIKE 'bp_%'
ORDER BY tablename;"
```

**Expected output:**
```
    tablename
─────────────────────────
 bp_activity_logs
 bp_steps
 bp_trigger_executions
 bp_triggers
(4 rows)
```

If tables don't exist, run:
```bash
psql postgresql://postgres:postgres@localhost/alpha < backend/db/migrations/2025_10_21_create_bp_triggers.sql
```

### Step 3: Build and Start Worker

**Terminal 1: Start Temporal Worker**

```bash
cd /Users/eganpj/GitHub/semlayer/backend/cmd/worker

# Build
go build -o ./worker main.go

# Run
./worker
```

**Expected output:**
```
✅ Connected to Temporal at localhost:7233
✅ Connected to PostgreSQL database
✅ Worker created for task queue: bp_queue
✅ Registered workflow: DynamicBPWorkflow
✅ Registered 9 activities
🚀 Starting Temporal worker...
✅ Worker started and listening for workflows on bp_queue
```

### Step 4: Start Trigger Engine

**Terminal 2: Start Trigger Engine**

```bash
cd /Users/eganpj/GitHub/semlayer/backend/cmd/triggers

# Build
go build -o ./triggers main.go

# Run
./triggers
```

**Expected output:**
```
✅ PostgreSQL connected
✅ Event listener started for entity_events
🚀 Starting escalation monitor (5 min interval)
✅ Trigger engine running on port 29090
📡 Listening for PostgreSQL events...
```

### Step 5: Verify Everything is Working

**Terminal 3: Run verification**

```bash
# Check all services are running
docker ps | grep semlayer

# Check Temporal is responding
curl http://localhost:8080/api/v1/namespaces

# Check Hasura is responding
curl http://localhost:8083/v1/version

# Check database connection
psql postgresql://postgres:postgres@localhost/alpha -c "SELECT version();"
```

---

## 🧪 Test the Complete Workflow

### Create a Test Trigger and BP

```bash
# 1. First, get your tenant ID
TENANT_ID=$(psql postgresql://postgres:postgres@localhost/alpha -t -c "
SELECT id FROM tenants LIMIT 1;" | head -1 | xargs)

echo "Using tenant: $TENANT_ID"

# 2. Create a business process
PROCESS_ID=$(psql postgresql://postgres:postgres@localhost/alpha -t -c "
INSERT INTO business_processes (id, tenant_id, process_name)
VALUES (gen_random_uuid(), '$TENANT_ID', 'test-hire-process')
RETURNING id;" | xargs)

echo "Created process: $PROCESS_ID"

# 3. Add steps to the process
psql postgresql://postgres:postgres@localhost/alpha << EOF
INSERT INTO bp_steps (process_id, step_order, step_name, step_type, duration_hours)
VALUES 
  ('$PROCESS_ID', 1, 'Data Entry', 'data_entry', 0),
  ('$PROCESS_ID', 2, 'Validation', 'validate', 0),
  ('$PROCESS_ID', 3, 'Approval', 'approve', 2),
  ('$PROCESS_ID', 4, 'Notify', 'notify_email', 0);
EOF

echo "✅ Added 4 steps to process"

# 4. Create a trigger
TRIGGER_ID=$(psql postgresql://postgres:postgres@localhost/alpha -t -c "
INSERT INTO bp_triggers (
  id, tenant_id, trigger_name, trigger_type, enabled,
  event_config, target_process_id, priority
) VALUES (
  gen_random_uuid(),
  '$TENANT_ID',
  'EmployeeHiredTrigger',
  'event',
  true,
  '{\"entity\":\"Employee\",\"action\":\"created\"}',
  '$PROCESS_ID',
  1
)
RETURNING id;" | xargs)

echo "Created trigger: $TRIGGER_ID"

# 5. Fire a test event
echo "🔥 Firing test event..."
psql postgresql://postgres:postgres@localhost/alpha << EOF
SELECT pg_notify('entity_events', json_build_object(
  'tenant_id', '$TENANT_ID',
  'entity', 'Employee',
  'action', 'created',
  'entity_id', 'emp-12345',
  'data', json_build_object(
    'name', 'John Doe',
    'department', 'Engineering',
    'salary', 80000
  ),
  'timestamp', NOW()
)::text);
EOF

echo "✅ Event fired!"
echo ""
echo "📊 Monitor in Temporal UI:"
echo "   Open http://localhost:8080"
echo "   Look for workflows in the 'bp_queue' task queue"
```

---

## 📊 Monitor Your Workflows

### Temporal Web UI

Open http://localhost:8080 in your browser:

1. Click "Workflows" in sidebar
2. Select "bp_queue" task queue
3. View running/completed workflows
4. Click workflow ID to see:
   - Execution timeline
   - Activity details
   - Step results
   - Any errors

### Command Line Monitoring

```bash
# List all workflows
docker exec semlayer-temporal temporal workflow list --address localhost:7233 --limit 10

# Get workflow details
docker exec semlayer-temporal temporal workflow describe \
  --workflow-id <WORKFLOW_ID>

# View activity details
docker exec semlayer-temporal temporal workflow show \
  --workflow-id <WORKFLOW_ID>
```

### Database Monitoring

```bash
# Check execution history
psql postgresql://postgres:postgres@localhost/alpha -c "
SELECT workflow_id, execution_status, executed_at, completed_at
FROM bp_trigger_executions
ORDER BY executed_at DESC
LIMIT 10;"

# Check activity logs
psql postgresql://postgres:postgres@localhost/alpha -c "
SELECT process_id, activity_type, status, logged_at
FROM bp_activity_logs
ORDER BY logged_at DESC
LIMIT 20;"

# Check escalations
psql postgresql://postgres:postgres@localhost/alpha -c "
SELECT workflow_id, escalation_time
FROM bp_trigger_executions
WHERE execution_status = 'escalated'
ORDER BY escalation_time DESC;"
```

---

## 🔧 Configuration & Tuning

### Environment Variables

```bash
# Set in .env or when running commands

# Temporal connection
export TEMPORAL_HOST_PORT=localhost:7233

# Database connection
export DATABASE_URL=postgresql://postgres:postgres@localhost:5432/alpha?sslmode=disable

# Hasura connection
export HASURA_URL=http://localhost:8083/v1/graphql
export HASURA_ADMIN_SECRET=your-secret-here

# Worker settings
export TEMPORAL_TASK_QUEUE=bp_queue
```

### Performance Tuning

```go
// In worker configuration (backend/cmd/worker/main.go):
w := worker.New(temporalClient, "bp_queue", worker.Options{
    MaxConcurrentActivityExecutionSize:     100,  // Parallel activities
    MaxConcurrentLocalActivityExecutionSize: 100,  // Parallel local activities
    MaxConcurrentWorkflowTaskExecutionSize: 10,   // Workflow tasks
})
```

---

## 🚨 Troubleshooting

### Temporal not connecting

```bash
# Check if Temporal is running
docker ps | grep temporal

# Check Temporal logs
docker logs semlayer-temporal

# Restart Temporal
docker-compose restart temporal
```

### Database connection errors

```bash
# Verify PostgreSQL is running
psql postgresql://postgres:postgres@localhost/alpha -c "SELECT 1;"

# Check BP tables exist
psql postgresql://postgres:postgres@localhost/alpha -c "\dt bp_*"

# Create tables if missing
psql postgresql://postgres:postgres@localhost/alpha < backend/db/migrations/2025_10_21_create_bp_triggers.sql
```

### Worker not receiving workflows

```bash
# Check worker is running and listening
# Look for: "Worker started and listening for workflows on bp_queue"

# Verify task queue name matches
# Both trigger engine and worker must use "bp_queue"

# Check logs in worker terminal

# Restart worker if needed
# Kill current process and run again
```

### Events not triggering workflows

```bash
# Check trigger engine is running and listening
# Look for: "Listening for PostgreSQL events..."

# Verify PostgreSQL LISTEN permissions
psql postgresql://postgres:postgres@localhost/alpha -c "
SELECT 1;" # Should return 1

# Test event manually
psql postgresql://postgres:postgres@localhost/alpha -c "
SELECT pg_notify('entity_events', '{\"test\":\"data\"}');"

# Check trigger engine logs for event processing
```

### Hasura not connecting

```bash
# Check Hasura is running
docker ps | grep graphql-engine

# Check logs
docker logs semlayer-graphql-engine-1

# Verify PostgreSQL connection
docker logs semlayer-graphql-engine-1 | grep -i "postgres"

# Restart Hasura
docker-compose restart graphql-engine
```

---

## 📚 Documentation References

- **Complete Guide**: `BP_TRIGGER_ENGINE_COMPLETE.md`
- **Quick Reference**: `BP_TRIGGER_ENGINE_QUICK_REFERENCE.md`
- **Architecture**: `BP_BUILDER_ENTERPRISE_INTEGRATION.md`
- **Deployment**: `BP_BUILDER_NEXT_STEPS.md`

---

## ✅ Deployment Checklist

Before going to production:

- [ ] Temporal running and healthy
- [ ] PostgreSQL backed up
- [ ] All BP tables created
- [ ] Worker running successfully
- [ ] Trigger engine receiving events
- [ ] Test workflow executed successfully
- [ ] Monitoring and alerts set up
- [ ] Temporal retention policies configured
- [ ] Database indexes optimized
- [ ] Resource limits set for services

---

## 🎯 Next Steps

1. **Run the setup script**: `./setup_trigger_engine.sh`
2. **Start the worker**: `cd backend/cmd/worker && ./worker`
3. **Start trigger engine**: `cd backend/cmd/triggers && ./triggers`
4. **View Temporal UI**: Open http://localhost:8080
5. **Create test trigger**: Follow "Test the Complete Workflow" section
6. **Monitor workflows**: Use Temporal UI or database queries

---

## 📞 Support

If you encounter issues:

1. Check troubleshooting section above
2. Review service logs (docker logs)
3. Verify all services are running (docker ps)
4. Check database connections (psql)
5. Read complete documentation files

**Status**: ✅ All systems ready for deployment!
