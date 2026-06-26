# Event-Router Microservice: Deployment & Test Guide

## Overview
The event-router microservice enables configurable, tenant-scoped event routing. When a field changes in an entity (detected and captured by EntityDrawerTreeView), the core app posts an event to the event-router, which:
1. Fetches matching routing configs from Hasura (event_type, bo_type, filters).
2. Applies filter rules (min_value, max_value, contains).
3. Publishes routed events to RabbitMQ queues for downstream systems.

## Prerequisites
- Docker & Docker Compose installed.
- PostgreSQL running locally (or via docker-compose).
- Tenant ID: `910638ba-a459-4a3f-bb2d-78391b0595f6`
- Tenant Datasource ID: `982aef38-418f-46dc-acd0-35fe8f3b97b0`

## Step 1: Run Migrations

### Option A: Using psql directly
```bash
# Connect to local PostgreSQL
psql postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable

# Run migrations in order
\i backend/migrations/000050_create_bo_events_table.sql
\i backend/migrations/000051_create_event_configs_table.sql

# Verify tables exist
\dt bo_events
\dt event_configs

# Exit
\q
```

### Option B: Using Docker Compose (if you have a migration runner)
```bash
# Copy migrations into the backend container and run them via the server startup
# (The backend Dockerfile should include a migration step; check backend/Dockerfile for details)
```

## Step 2: Start All Services

### Start Docker Compose stack
```bash
# From the workspace root
docker-compose up -d

# Wait 10-15 seconds for services to stabilize
sleep 15

# Verify services are running
docker-compose ps
```

Expected output:
```
NAME                          STATUS              PORTS
semlayer-graphql-engine-1     Up (healthy)        0.0.0.0:8081->8080/tcp
semlayer-backend-1            Up                  0.0.0.0:29080->8080/tcp
semlayer-event-router         Up (healthy)        0.0.0.0:8081->8081/tcp
semlayer-rabbitmq             Up                  0.0.0.0:5672->5672/tcp, 0.0.0.0:15672->15672/tcp
```

### Verify services are responding
```bash
# Check Hasura GraphQL endpoint
curl http://localhost:8081

# Check backend health (if it has a /health endpoint)
curl http://localhost:29080/health

# Check event-router health
curl http://localhost:8081/health

# Check RabbitMQ management UI
open http://localhost:15672  # Username: guest, Password: guest
```

## Step 3: Configure Hasura (Track event_configs table)

### Track event_configs table in Hasura
```bash
# Use Hasura console
open http://localhost:8081

# Or use GraphQL mutation to track table programmatically
HASURA_ADMIN_SECRET="your-admin-secret"  # from .env

curl -X POST http://localhost:8081/v1/graphql \
  -H "Content-Type: application/json" \
  -H "X-Hasura-Admin-Secret: $HASURA_ADMIN_SECRET" \
  -d '{
    "type": "track_table",
    "args": {
      "schema": "public",
      "name": "event_configs"
    }
  }'
```

### Optional: Set RLS for event_configs
```bash
# Use Hasura console to set Row-Level Security for tenant isolation
# Or via SQL (if needed):
# ALTER TABLE event_configs ENABLE ROW LEVEL SECURITY;
# CREATE POLICY rls_event_configs ON event_configs 
#   USING (tenant_id = current_setting('app.tenant_id')::uuid);
```

## Step 4: Insert Test Event Routing Config

### Create a routing rule via direct SQL
```bash
psql postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable

-- Insert a test config: route "fieldchange" events for "client_investors" BO to RabbitMQ queue "client_investor_updates"
INSERT INTO event_configs (
  id, tenant_id, event_type, bo_type, field_name, filter_json, route_queue, created_at
) VALUES (
  gen_random_uuid(),
  '910638ba-a459-4a3f-bb2d-78391b0595f6'::uuid,
  'fieldchange',
  'client_investors',
  NULL,  -- NULL means all fields
  '{}',  -- Empty filter means all events pass
  'client_investor_updates',
  NOW()
);

-- Or with a specific filter (e.g., only route if old_value or new_value contains "high_risk")
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

-- Verify config was inserted
SELECT * FROM event_configs WHERE tenant_id = '910638ba-a459-4a3f-bb2d-78391b0595f6'::uuid;

\q
```

### Or via Hasura GraphQL mutation
```bash
HASURA_ADMIN_SECRET="your-admin-secret"

curl -X POST http://localhost:8081/v1/graphql \
  -H "Content-Type: application/json" \
  -H "X-Hasura-Admin-Secret: $HASURA_ADMIN_SECRET" \
  -H "X-Tenant-ID: 910638ba-a459-4a3f-bb2d-78391b0595f6" \
  -d '{
    "query": "mutation InsertEventConfig($id: uuid!, $tenant_id: uuid!, $event_type: String!, $bo_type: String!, $route_queue: String!) { insert_event_configs_one(object: {id: $id, tenant_id: $tenant_id, event_type: $event_type, bo_type: $bo_type, route_queue: $route_queue}) { id } }",
    "variables": {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "tenant_id": "910638ba-a459-4a3f-bb2d-78391b0595f6",
      "event_type": "fieldchange",
      "bo_type": "client_investors",
      "route_queue": "client_investor_updates"
    }
  }'
```

## Step 5: Trigger a Test Event from the UI

1. Open the Fabric Builder UI: `http://localhost:3000` (or your frontend port).
2. Navigate to **Entity Config → EntityConfigPageV2** (card-based UI).
3. Select a tenant and datasource (use the tenant picker).
4. Click **Edit** on an existing entity (or **Add** a new one).
5. In the drawer, edit a field and click **Save**.
   - This triggers `EntityDrawerTreeView.handleSave()`.
   - It detects the field change, calls `createEvent()`, which POSTs to `/events`.
   - The core app saves the event to `bo_events` and async-forwards it to event-router.
6. The event-router receives it, matches the routing config, and publishes to the RabbitMQ queue.

### Or trigger programmatically with curl
```bash
TENANT_ID="910638ba-a459-4a3f-bb2d-78391b0595f6"
DATASOURCE_ID="982aef38-418f-46dc-acd0-35fe8f3b97b0"

curl -X POST http://localhost:29080/events \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "X-Tenant-Datasource-ID: $DATASOURCE_ID" \
  -d '{
    "bo_type": "client_investors",
    "bo_id": "12345",
    "event_type": "fieldchange",
    "field_name": "risk_level",
    "old_value": "low_risk",
    "new_value": "high_risk",
    "changed_by": "admin@example.com"
  }'
```

Expected response:
```json
{
  "success": true
}
```

## Step 6: Verify Event Routing in RabbitMQ

### Check RabbitMQ Management UI
1. Open `http://localhost:15672`
2. Login with `guest` / `guest`
3. Navigate to **Queues** tab
4. Look for the queue name you specified in the routing config (e.g., `client_investor_updates`)
5. You should see a message count > 0

### Or consume messages via amqp CLI
```bash
# Install amqp-cli if not already installed
# brew install amqp-cli  # or apt-get, depending on OS

# Consume messages from the queue
amqp-cli --url amqp://guest:guest@localhost:5672 queue.declare client_investor_updates
amqp-cli --url amqp://guest:guest@localhost:5672 queue.consume client_investor_updates
```

### Or write a quick Go consumer
```bash
cat > /tmp/consume_test.go << 'EOF'
package main

import (
	"fmt"
	"log"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatal(err)
	}
	defer ch.Close()

	q, err := ch.QueueDeclare("client_investor_updates", false, false, false, false, nil)
	if err != nil {
		log.Fatal(err)
	}

	msgs, err := ch.Consume(q.Name, "", true, false, false, false, nil)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Waiting for messages on queue:", q.Name)
	for msg := range msgs {
		fmt.Printf("Received message:\n%s\n", string(msg.Body))
	}
}
EOF

# Run the consumer
cd /tmp
go run consume_test.go
```

## Step 7: Verify Event History in bo_events

### Query events from the database
```bash
psql postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable

-- Check events for a specific BO
SELECT id, tenant_id, bo_type, bo_id, field_name, old_value, new_value, changed_by, changed_at
FROM bo_events
WHERE bo_type = 'client_investors' AND bo_id = '12345'
ORDER BY changed_at DESC
LIMIT 10;

-- Or check all events for the tenant
SELECT id, tenant_id, bo_type, bo_id, field_name, changed_at
FROM bo_events
WHERE tenant_id = '910638ba-a459-4a3f-bb2d-78391b0595f6'
ORDER BY changed_at DESC
LIMIT 20;

\q
```

## Step 8: Check Event-Router Logs

```bash
# View event-router service logs
docker-compose logs event-router -f

# Or check a specific container
docker logs semlayer-event-router -f
```

Expected log lines:
```
INFO    event-router fetching event configs from Hasura
INFO    event-router config cache refreshed: 2 configs loaded
INFO    event-router received event: bo_type=client_investors, event_type=fieldchange
INFO    event-router matched 1 config(s) for key: client_investors_fieldchange
INFO    event-router published to queue: client_investor_updates (routed_count=1)
```

## Troubleshooting

### Event-router fails to start
- Check HASURA_URL is reachable: `curl http://graphql-engine:8080/v1/graphql`
- Check RABBITMQ_URL is valid: `curl -u guest:guest http://rabbitmq:5672` (should fail, but connection is tested)
- View logs: `docker-compose logs event-router`

### Events not routed to RabbitMQ
- Verify event_configs table has rows matching the event: `SELECT * FROM event_configs WHERE bo_type='client_investors';`
- Check event-router logs for "matched 0 config(s)" messages.
- Verify filter rules apply: if a filter exists and doesn't match, no routing occurs.

### RabbitMQ queue is empty
- Verify events are being posted to core app: check `bo_events` table.
- Verify event-router is running: `docker-compose ps event-router`
- Check event-router received the event: `docker-compose logs event-router | grep "received event"`

### Hasura can't track event_configs
- Ensure Hasura is running: `docker-compose ps graphql-engine`
- Check admin secret matches: `grep HASURA_ADMIN_SECRET .env`
- Verify event_configs table exists: `psql ... -c "\dt event_configs"`

## Architecture Summary

```
Frontend (React)
    ↓
    EntityDrawerTreeView (detect field changes)
    ↓
    createEvent() → POST /events (core app)
    ↓
Backend (Core App)
    ↓
    POST /events handler:
      1. Insert into bo_events (local history)
      2. Async forward to event-router
    ↓
Event-Router Microservice (Go + Gin)
    ↓
    1. Fetch configs from Hasura (GraphQL)
    2. Cache configs in-memory (5-min refresh)
    3. Match event to routing rules
    4. Apply filters (min_value, max_value, contains)
    5. Publish to RabbitMQ queues
    ↓
RabbitMQ Broker
    ↓
    Downstream systems (consumers listening on queues)
```

## Production Checklist

- [ ] Set `HASURA_ADMIN_SECRET` to a strong random value (not "your-admin-secret").
- [ ] Use RabbitMQ credentials (not default guest/guest).
- [ ] Enable TLS for Hasura + RabbitMQ in production.
- [ ] Configure RLS policies in Hasura for tenant isolation.
- [ ] Add monitoring/alerting for event-router health.
- [ ] Add log aggregation (ELK, Datadog, etc.).
- [ ] Test failover: stop event-router, verify core app handles gracefully (fire-and-forget means no data loss).
- [ ] Test RabbitMQ persistence and consumer acknowledgments.
- [ ] Review event_configs regularly for active routing rules.

## Next Steps

1. **Run the deployment commands above** to start all services and verify end-to-end routing.
2. **Extend filter logic** if needed (e.g., regex support, date ranges, custom predicates).
3. **Integrate downstream consumers** that subscribe to RabbitMQ queues and process events.
4. **Add metrics/observability** (Prometheus, OpenTelemetry) to track routed event counts, latencies, queue depths.
5. **Set up RLS and RBAC** in Hasura to enforce multi-tenant security at the GraphQL layer.
