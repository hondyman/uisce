# 🚀 BP Trigger Engine - Complete Startup Guide (UPDATED)

**Status**: ✅ READY TO START  
**Setup Time**: 10 minutes total  
**Temporal**: Running locally (NOT Docker)  
**UI**: Temporal UI in Docker at port 8081

---

## 🎯 Quick Start (Fastest Path)

### Terminal 1: Start Temporal Server

```bash
cd /Users/eganpj/GitHub/semlayer
./start_temporal_locally.sh
```

Wait for message: `Temporal is running at localhost:7233`

### Terminal 2: Start Temporal UI

```bash
cd /Users/eganpj/GitHub/semlayer
docker compose up -d temporal-ui

# Verify
curl -s http://localhost:8081 | head -5
```

### Terminal 3: Start Worker

```bash
cd /Users/eganpj/GitHub/semlayer/backend/cmd/worker
go build -o ./worker main.go
./worker
```

Wait for output: `✅ Worker started and listening for workflows on bp_queue`

### Terminal 4: Start Trigger Engine

```bash
cd /Users/eganpj/GitHub/semlayer/backend/cmd/triggers
go build -o ./triggers main.go
./triggers
```

Wait for output: `✅ Trigger engine running on port 29090`

---

## 📊 Service URLs

Once everything is running:

| Service | URL | Port |
|---------|-----|------|
| **Temporal UI** | http://localhost:8081 | 8081 |
| **Temporal gRPC** | localhost:7233 | 7233 |
| **Trigger Engine Health** | http://localhost:29090/health | 29090 |

---

## ✅ Verify Everything Works

### Check Temporal is Running

```bash
# From any terminal
temporal workflow list

# Should show empty list (no workflows yet)
```

### Check Worker is Connected

```bash
# From Temporal CLI
temporal worker describe --task-queue bp_queue

# Should show worker info
```

### Check Trigger Engine is Running

```bash
# From any terminal
curl http://localhost:29090/health

# Should return: ok
```

### Check Database Connections

```bash
# Check BP tables exist
psql postgresql://postgres:postgres@localhost/alpha -c "
SELECT COUNT(*) FROM bp_triggers;
SELECT COUNT(*) FROM bp_steps;
SELECT COUNT(*) FROM bp_activity_logs;"

# All should return 0 rows initially
```

---

## 🧪 Test End-to-End

### Create a Test Process

```bash
# Get a tenant ID
TENANT_ID=$(psql postgresql://postgres:postgres@localhost/alpha -t -c "
SELECT id FROM tenants LIMIT 1;" | xargs)

# Create a process
PROCESS_ID=$(psql postgresql://postgres:postgres@localhost/alpha -t -c "
INSERT INTO business_processes (id, tenant_id, process_name)
VALUES (gen_random_uuid(), '$TENANT_ID', 'test-process')
RETURNING id;" | xargs)

# Add a step
psql postgresql://postgres:postgres@localhost/alpha << EOF
INSERT INTO bp_steps (process_id, step_order, step_name, step_type, duration_hours)
VALUES ('$PROCESS_ID', 1, 'Test Step', 'data_entry', 0);
EOF

echo "✅ Created process: $PROCESS_ID"
```

### Create a Test Trigger

```bash
# Create trigger
TRIGGER_ID=$(psql postgresql://postgres:postgres@localhost/alpha -t -c "
INSERT INTO bp_triggers (
  id, tenant_id, trigger_name, trigger_type, enabled,
  event_config, target_process_id, priority
) VALUES (
  gen_random_uuid(),
  '$TENANT_ID',
  'TestTrigger',
  'event',
  true,
  '{\"entity\":\"Test\",\"action\":\"created\"}',
  '$PROCESS_ID',
  1
)
RETURNING id;" | xargs)

echo "✅ Created trigger: $TRIGGER_ID"
```

### Fire a Test Event

```bash
# Fire event
psql postgresql://postgres:postgres@localhost/alpha << EOF
SELECT pg_notify('entity_events', json_build_object(
  'tenant_id', '$TENANT_ID',
  'entity', 'Test',
  'action', 'created',
  'entity_id', 'test-001',
  'data', json_build_object('test', 'data'),
  'timestamp', NOW()
)::text);
EOF

echo "🔥 Event fired!"
```

### Monitor in Temporal UI

1. Open http://localhost:8081
2. Click "Workflows"
3. You should see a new workflow in "Pending" or "Running" status
4. Click it to see execution details

---

## 📂 What Gets Created/Used

```
Temporal Data: /tmp/temporal_dev.db (local file, auto-created)
Main Database: localhost:5432/alpha (PostgreSQL)
Temporal UI:   Docker container on port 8081
Worker:        Your current terminal session
Trigger Engine: Your current terminal session
```

---

## 🛑 Stopping Everything

```bash
# Terminal 1: Stop Temporal
# Press Ctrl+C

# Terminal 2: Stop Temporal UI
docker compose down temporal-ui

# Terminal 3: Stop Worker
# Press Ctrl+C

# Terminal 4: Stop Trigger Engine
# Press Ctrl+C

# Optional: Clean up
rm /tmp/temporal_dev.db
```

---

## 🚨 Troubleshooting

### Temporal CLI not found

```bash
# Install Temporal CLI
brew install temporalite

# Or if you have it already
which temporal
```

### Port 7233 already in use

```bash
# Kill existing process
lsof -i :7233
kill -9 <PID>

# Or use different port
temporal server start-dev --headless --port 7234
```

### Worker not connecting to Temporal

```bash
# Make sure Temporal server is running first
# Check logs in Temporal terminal for errors

# Verify Temporal is listening
nc -zv localhost 7233

# If it fails, restart Temporal server
```

### Trigger engine not receiving events

```bash
# Check PostgreSQL LISTEN is working
psql postgresql://postgres:postgres@localhost/alpha -c "
SELECT pg_notify('test', 'message');
"

# Check trigger engine logs for "Listening for PostgreSQL events"

# Verify BP tables exist
psql postgresql://postgres:postgres@localhost/alpha -c "\dt bp_*"
```

### Can't connect to PostgreSQL

```bash
# Test connection
psql postgresql://postgres:postgres@localhost/alpha -c "SELECT 1;"

# If fails, make sure PostgreSQL is running:
# It should be running on localhost:5432 (not in Docker)
```

---

## 📚 Architecture Flow

```
1. Event Fired (PostgreSQL NOTIFY)
   ↓
2. Trigger Engine Receives (ListenForNotifications)
   ↓
3. Process Event Triggers (Query bp_triggers, match config)
   ↓
4. Execute Trigger (Start Temporal Workflow)
   ↓
5. Worker Receives (gRPC from :7233)
   ↓
6. Run Workflow (DynamicBPWorkflow)
   ↓
7. Execute Steps (Call Activities)
   ↓
8. Log Results (bp_activity_logs)
   ↓
9. Complete (View in Temporal UI at :8081)
```

---

## ✨ Key Commands Reference

```bash
# Temporal CLI
temporal workflow list
temporal workflow describe --workflow-id <ID>
temporal workflow show --workflow-id <ID>
temporal activity list

# PostgreSQL
psql postgresql://postgres:postgres@localhost/alpha
\dt bp_*  # List BP tables
SELECT COUNT(*) FROM bp_triggers;

# Docker
docker ps
docker compose up -d temporal-ui
docker compose down

# Kill ports
lsof -i :7233   # Temporal
lsof -i :8081   # Temporal UI
lsof -i :29090  # Trigger Engine
```

---

## 🎉 Success Indicators

When everything is working:

✅ Terminal 1: "Temporal is running at localhost:7233"  
✅ Terminal 2: Docker shows "Up" for temporal-ui  
✅ Terminal 3: "Worker started and listening for workflows on bp_queue"  
✅ Terminal 4: "Trigger engine running on port 29090"  
✅ Browser: http://localhost:8081 shows Temporal UI  
✅ Workflows appear in Temporal UI when events fire  
✅ Activity logs saved to bp_activity_logs table  

---

## 📋 Deployment Checklist

- [ ] Temporal running (Terminal 1)
- [ ] Temporal UI accessible (port 8081)
- [ ] Worker running and connected (Terminal 3)
- [ ] Trigger engine running and listening (Terminal 4)
- [ ] Test workflow created and executed
- [ ] Activity logs visible in database
- [ ] Temporal UI showing completed workflows
- [ ] All logs indicating healthy operation

---

## 🎯 Next Steps

1. ✅ Start Temporal server locally
2. ✅ Start Temporal UI in Docker
3. ✅ Start worker process
4. ✅ Start trigger engine
5. ✅ Create test trigger and BP
6. ✅ Fire test event
7. ✅ Monitor in Temporal UI
8. ✅ Verify logs in database

**You're ready to start!** 🚀

Run this in Terminal 1:
```bash
cd /Users/eganpj/GitHub/semlayer
./start_temporal_locally.sh
```

Then follow the quick start section above!
