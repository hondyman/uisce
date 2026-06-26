# Semantic Sync Deployment Checklist

## ✅ Pre-Deployment (Completed)

- [x] Database migration fixed and executed
- [x] Trigger `metrics_registry_notify_trigger` created successfully
- [x] Notification channel `metrics_registry_changed` configured
- [x] Semantic Sync service code references correct table and channel names
- [x] React console component created with 4 tabs
- [x] Navigation menu integration completed
- [x] docker-compose service definition configured
- [x] AppRoutes integration completed
- [x] Documentation generated

## 🚀 Deployment Steps

### Step 1: Start Services
```bash
cd /Users/eganpj/GitHub/semlayer

# Start all services including semantic-sync
docker-compose up -d

# Verify all services are running
docker-compose ps
```

**Expected Output**:
```
NAME                            STATUS
semlayer-backend-1              Up (healthy)
semlayer-fabric-builder-1       Up (healthy)
semlayer-frontend-1             Up (healthy)
semlayer-temporal-1             Up (healthy)
semlayer-rabbitmq-1             Up (healthy)
semlayer-postgres-1             Up (healthy)
semlayer-semantic-sync-1        Up (healthy)
```

### Step 2: Verify Semantic Sync Service
```bash
# Check logs to confirm service started
docker logs semlayer-semantic-sync-1

# Expected logs should include:
# ✅ Connected to Postgres
# 🎧 Semantic Sync Service started. Listening for metrics_registry changes...
```

### Step 3: Test Event Trigger
```bash
# Open psql connection and listen for notifications
psql postgres://postgres:postgres@localhost:5432/alpha

# In psql:
LISTEN metrics_registry_changed;

# In another terminal, trigger a change:
psql postgres://postgres:postgres@localhost:5432/alpha -c \
  "UPDATE metrics_registry SET category = 'updated' WHERE id = 1 LIMIT 1;"

# You should see a notification like:
# Asynchronous notification "metrics_registry_changed" received from server process with PID 12345.
```

### Step 4: Verify Schema Generation
```bash
# Check if cube-schemas directory was created and populated
ls -la ./cube-schemas/

# Expected files (after first sync):
# - metrics_pop.js
# - metrics_anomalies.js
# - metrics_atomic.js
```

### Step 5: Access Frontend Console
1. Open browser: `http://localhost:3000`
2. Navigate to: **Entity → Entities → Metric Calc**
3. Expected UI:
   - 4 tabs visible: "Registry", "PoP Trends", "Anomalies", "Runs"
   - "New" badge on menu item
   - Registry tab shows metric list with CRUD buttons
   - Mock data populated for demonstration

## 📊 Monitoring

### Service Health
```bash
# Check semantic-sync container health
docker inspect semlayer-semantic-sync-1 --format='{{.State.Health}}'

# Should return: healthy or starting
```

### Database Activity
```bash
# Monitor trigger invocations in postgres logs
docker logs -f semlayer-postgres-1 | grep metrics_registry_changed

# Monitor semantic-sync processing
docker logs -f semlayer-semantic-sync-1
```

### Schema Generation Logs
```bash
# View all schema regeneration activity
docker logs -f semlayer-semantic-sync-1 | grep -E "regenerate|schemas|SUCCESS|ERROR"
```

## 🔧 Troubleshooting

### Issue: Semantic Sync fails to connect
**Solution**:
```bash
# Verify DATABASE_URL is correct in docker-compose
docker exec semlayer-semantic-sync-1 env | grep DATABASE_URL

# Should output: postgres://postgres:postgres@host.docker.internal:5432/alpha?sslmode=disable
```

### Issue: Trigger not firing
**Solution**:
```bash
# Verify trigger is still created
psql postgres://postgres:postgres@localhost:5432/alpha \
  -c "SELECT tgname FROM pg_trigger WHERE tgname = 'metrics_registry_notify_trigger';"

# Should return one row: metrics_registry_notify_trigger

# Verify trigger is enabled
psql postgres://postgres:postgres@localhost:5432/alpha \
  -c "SELECT tgenabled FROM pg_trigger WHERE tgname = 'metrics_registry_notify_trigger';"

# Should return: O (enabled) or 1 (enabled), NOT D (disabled)
```

### Issue: No schemas generated
**Solution**:
1. Check metrics exist: `SELECT COUNT(*) FROM metrics_registry;`
2. Verify listener is active: `SELECT pg_listening_channels();` (from semantic-sync pod)
3. Check service logs: `docker logs semlayer-semantic-sync-1 | tail -50`
4. Manual trigger: `docker exec semlayer-semantic-sync-1 kill -1 1` (to force refresh)

## ✅ Post-Deployment Validation

- [ ] All services running (docker-compose ps)
- [ ] Semantic Sync logs show "Listening for metrics_registry changes"
- [ ] Frontend console loads without errors
- [ ] Mock data visible in all 4 tabs
- [ ] Navigation menu shows "Metric Calc" with "New" badge
- [ ] Test metric create/update triggers schema regeneration
- [ ] Cube schema files exist in `./cube-schemas/`

## 📝 Notes

- **Graceful Degradation**: If Semantic Sync fails, metrics can still be created via API, just schemas won't auto-regenerate until service restarts
- **Event Guarantee**: Postgres LISTEN/NOTIFY provides "at-least-once" semantics, not "exactly-once"
- **Schema Location**: Cube.js schemas written to mounted volume `./cube-schemas/` for persistence
- **Periodic Fallback**: Service also regenerates schemas every 1 hour even without triggers
- **Tenant Scoping**: When tenant support is added, filter queries by tenant_id in regenerateCubeSchemas()

## 🎯 Success Criteria

✅ **System is ready when**:
1. Semantic Sync container is healthy
2. Console loads with mock data
3. Test metric update triggers log entry in semantic-sync showing "✅ [SUCCESS] Cube schemas regenerated"
4. New schema files appear in `./cube-schemas/`

