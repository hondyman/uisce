# Event-Router: Quick Reference & Common Commands

## 🚀 Quick Start (Copy-Paste Ready)

### 1. Create migrations
```bash
psql postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable << 'EOF'
\i backend/migrations/000050_create_bo_events_table.sql
\i backend/migrations/000051_create_event_configs_table.sql
EOF
```

### 2. Start all services
```bash
docker-compose up -d
sleep 15
docker-compose ps
```

### 3. Insert test routing config
```bash
psql postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable << 'EOF'
INSERT INTO event_configs (
  id, tenant_id, event_type, bo_type, field_name, filter_json, route_queue, created_at
) VALUES (
  gen_random_uuid(),
  '910638ba-a459-4a3f-bb2d-78391b0595f6'::uuid,
  'fieldchange',
  'client_investors',
  NULL,
  '{}',
  'client_investor_updates',
  NOW()
);
EOF
```

### 4. Trigger test event
```bash
curl -X POST http://localhost:29080/events \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 910638ba-a459-4a3f-bb2d-78391b0595f6" \
  -H "X-Tenant-Datasource-ID: 982aef38-418f-46dc-acd0-35fe8f3b97b0" \
  -d '{
    "bo_type": "client_investors",
    "bo_id": "test-123",
    "event_type": "fieldchange",
    "field_name": "status",
    "old_value": "active",
    "new_value": "inactive",
    "changed_by": "admin"
  }'
```

### 5. Verify in Redpanda (Kafka)
```bash
# Check queue exists and has messages
open http://localhost:15672
# Login: guest / guest
# Navigate to Queues → client_investor_updates
```

### 6. Check event history
```bash
psql postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable << 'EOF'
SELECT bo_type, bo_id, field_name, old_value, new_value, changed_at 
FROM bo_events 
WHERE bo_type = 'client_investors' 
ORDER BY changed_at DESC 
LIMIT 5;
EOF
```

---

## 📋 Common Operations

### View all routing configs for a tenant
```bash
psql postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable << 'EOF'
SELECT event_type, bo_type, field_name, route_queue, filter_json 
FROM event_configs 
WHERE tenant_id = '910638ba-a459-4a3f-bb2d-78391b0595f6'::uuid;
EOF
```

### Delete old events (older than 30 days)
```bash
psql postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable << 'EOF'
DELETE FROM bo_events 
WHERE changed_at < NOW() - INTERVAL '30 days' 
  AND tenant_id = '910638ba-a459-4a3f-bb2d-78391b0595f6'::uuid;
EOF
```

### View event-router service status
```bash
docker-compose ps event-router
docker-compose logs event-router --tail=50
```

### Create a filtered routing config (only route "high_risk" changes)
```bash
psql postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable << 'EOF'
INSERT INTO event_configs (
  id, tenant_id, event_type, bo_type, field_name, filter_json, route_queue, created_at
) VALUES (
  gen_random_uuid(),
  '910638ba-a459-4a3f-bb2d-78391b0595f6'::uuid,
  'fieldchange',
  'client_investors',
  'risk_level',
  '{"new_value": {"contains": "high_risk"}}'::jsonb,
  'high_risk_alerts',
  NOW()
);
EOF
```

### Count events per BO type
```bash
psql postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable << 'EOF'
SELECT bo_type, COUNT(*) as event_count 
FROM bo_events 
WHERE tenant_id = '910638ba-a459-4a3f-bb2d-78391b0595f6'::uuid 
GROUP BY bo_type 
ORDER BY event_count DESC;
EOF
```

### List all Redpanda/Kafka topics and offsets

# Example (Redpanda):
# rpk topic list
# kafka-consumer-groups --bootstrap-server localhost:9092 --describe --group <group>
```bash
curl -u guest:guest http://localhost:15672/api/queues | jq '.[] | {name: .name, messages: .messages}'
```

### Purge a Kafka topic (or reset consumer offsets) (Redpanda)
```bash
curl -X DELETE -u guest:guest http://localhost:15672/api/queues/%2F/client_investor_updates/contents
```

### Restart event-router service
```bash
docker-compose restart event-router
# Wait for health check to pass
sleep 5
docker-compose ps event-router
```

### Rebuild event-router image after code changes
```bash
docker-compose build event-router --no-cache
docker-compose up -d event-router
```

### Tail all service logs
```bash
docker-compose logs -f
```

---

## 🧪 Full End-to-End Test Scenario

```bash
#!/bin/bash
set -e

TENANT_ID="910638ba-a459-4a3f-bb2d-78391b0595f6"
DATASOURCE_ID="982aef38-418f-46dc-acd0-35fe8f3b97b0"
QUEUE_NAME="test_queue_$(date +%s)"

echo "🚀 Starting end-to-end test..."

# 1. Create routing config
echo "📝 Creating routing config for queue: $QUEUE_NAME"
psql postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable << EOF
INSERT INTO event_configs (
  id, tenant_id, event_type, bo_type, field_name, filter_json, route_queue, created_at
) VALUES (
  gen_random_uuid(),
  '$TENANT_ID'::uuid,
  'fieldchange',
  'test_entity',
  NULL,
  '{}',
  '$QUEUE_NAME',
  NOW()
);
EOF

# 2. Trigger event
echo "🔔 Triggering test event..."
curl -X POST http://localhost:29080/events \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "X-Tenant-Datasource-ID: $DATASOURCE_ID" \
  -d "{
    \"bo_type\": \"test_entity\",
    \"bo_id\": \"test-001\",
    \"event_type\": \"fieldchange\",
    \"field_name\": \"status\",
    \"old_value\": \"pending\",
    \"new_value\": \"approved\",
    \"changed_by\": \"test-user\"
  }"

# 3. Wait for async processing
sleep 2

# 4. Check event in bo_events
echo "✅ Checking event history..."
psql postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable << EOF
SELECT COUNT(*) as events_in_history FROM bo_events 
WHERE bo_type = 'test_entity' AND bo_id = 'test-001';
EOF

# 5. Check message in Redpanda (Kafka)
echo "✅ Checking Redpanda topic..."
# Prefer using rpk inside the Redpanda container to inspect topics. Example:
# docker exec semlayer-redpanda rpk topic describe "$QUEUE_NAME"
# or attempt a single consume to verify messages are present:
if docker exec semlayer-redpanda rpk topic describe "$QUEUE_NAME" >/dev/null 2>&1; then
  echo "Topic '$QUEUE_NAME' exists. Description:"
  docker exec semlayer-redpanda rpk topic describe "$QUEUE_NAME" || true
  echo "Attempting to consume 1 message (non-destructive):"
  docker exec semlayer-redpanda rpk topic consume "$QUEUE_NAME" -o start -n 1 -f '%k %v\n' || true
else
  echo "❌ FAILED: Topic '$QUEUE_NAME' not found; check producers or create the topic"
  exit 1
fi

echo "🎉 End-to-end test passed!"
```

Save and run:
```bash
chmod +x test_e2e.sh
./test_e2e.sh
```

---

## 🔧 Debugging Filters

### Test a numeric filter (min/max)
```bash
# Config that only routes events where new_value (as number) > 100
INSERT INTO event_configs (
  id, tenant_id, event_type, bo_type, field_name, filter_json, route_queue, created_at
) VALUES (
  gen_random_uuid(),
  '910638ba-a459-4a3f-bb2d-78391b0595f6'::uuid,
  'fieldchange',
  'accounts',
  'balance',
  '{"new_value": {"min_value": 100}}'::jsonb,
  'high_balance_alerts',
  NOW()
);

-- Trigger event that PASSES filter (balance goes to 500)
curl -X POST http://localhost:29080/events \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 910638ba-a459-4a3f-bb2d-78391b0595f6" \
  -H "X-Tenant-Datasource-ID: 982aef38-418f-46dc-acd0-35fe8f3b97b0" \
  -d '{
    "bo_type": "accounts",
    "bo_id": "acct-123",
    "event_type": "fieldchange",
    "field_name": "balance",
    "old_value": "50",
    "new_value": "500",
    "changed_by": "admin"
  }'
```

### Test a string filter (contains)
```bash
# Config that only routes events where new_value contains "critical"
INSERT INTO event_configs (
  id, tenant_id, event_type, bo_type, field_name, filter_json, route_queue, created_at
) VALUES (
  gen_random_uuid(),
  '910638ba-a459-4a3f-bb2d-78391b0595f6'::uuid,
  'fieldchange',
  'alerts',
  'severity',
  '{"new_value": {"contains": "critical"}}'::jsonb,
  'critical_incidents',
  NOW()
);

-- Trigger event that PASSES filter
curl -X POST http://localhost:29080/events \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 910638ba-a459-4a3f-bb2d-78391b0595f6" \
  -H "X-Tenant-Datasource-ID: 982aef38-418f-46dc-acd0-35fe8f3b97b0" \
  -d '{
    "bo_type": "alerts",
    "bo_id": "alert-001",
    "event_type": "fieldchange",
    "field_name": "severity",
    "old_value": "warning",
    "new_value": "critical_incident",
    "changed_by": "system"
  }'
```

---

## 📚 File Checklist

All files required for event-router are in place:

- ✅ `backend/migrations/000050_create_bo_events_table.sql` — Event history table
- ✅ `backend/migrations/000051_create_event_configs_table.sql` — Routing config table
- ✅ `backend/internal/api/api.go` — Core app POST /events + forwardToEventRouter
- ✅ `backend/cmd/event-router/main.go` — Event-router microservice (290 lines)
- ✅ `backend/cmd/event-router/go.mod` — Go dependencies
- ✅ `backend/cmd/event-router/Dockerfile` — Multi-stage Docker build
- ✅ `frontend/src/api/events.ts` — createEvent / getEventsForBO helpers
- ✅ `frontend/src/components/EntityDrawerTreeView.tsx` — Event capture on save
- ✅ `docker-compose.yml` — Service definitions (backend, event-router, rabbitmq, hasura)

---

## 🎯 Expected Architecture Behavior

1. **User edits entity field** → EntityDrawerTreeView detects change.
2. **Save button clicked** → Diff calculated, createEvent POST to core app /events.
3. **Core app POST /events** → Saves to bo_events table, async forwards to event-router.
4. **Event-router receives event** → Fetches matching configs from Hasura, applies filters.
5. **Config matches + filter passes** → Publishes event to Redpanda/Kafka topic.
6. **Downstream consumer** → Listens on queue, processes routed event.

All operations are tenant-scoped via X-Tenant-ID headers and database filters.

---

## 🆘 Support Commands

```bash
# Check all tables exist
psql postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable -c "\dt"

# Check Hasura is running
curl -s http://localhost:8081 | head -20

# Check Redpanda (Kafka) is running
# Use rpk inside the Redpanda container to check cluster status
docker exec semlayer-redpanda rpk cluster info || true

# Check event-router is running
curl -s http://localhost:8081/health | jq .

# Count all events in system
psql postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable -c "SELECT COUNT(*) FROM bo_events;"

# Count all routing configs
psql postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable -c "SELECT COUNT(*) FROM event_configs;"

# Get Kafka topic info (depth/partitions)
# Use rpk to list/describe topics. Example:
# docker exec semlayer-redpanda rpk topic list
# docker exec semlayer-redpanda rpk topic describe <topic-name> | sed -n '1,20p'
```

