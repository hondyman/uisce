# End-to-End Security Synchronization Deployment Guide

This guide walks you through deploying the complete Debezium-based security synchronization system.

## Prerequisites

- Docker and Docker Compose installed
- PostgreSQL 12+ with superuser access
- RabbitMQ 3.12+
- Go 1.21+

---

## Step 1: Configure PostgreSQL for Logical Replication

### 1.1 Update postgresql.conf

Add or modify these settings in your PostgreSQL configuration file:

```conf
wal_level = logical
max_replication_slots = 10
max_wal_senders = 10
```

**Location:** Usually `/var/lib/postgresql/data/postgresql.conf` or `/etc/postgresql/*/main/postgresql.conf`

### 1.2 Restart PostgreSQL

```bash
# Docker
docker-compose restart postgres

# Or system service
sudo systemctl restart postgresql
```

### 1.3 Verify Configuration

```bash
psql -U postgres -d alpha -c "SHOW wal_level;"
# Should return: logical
```

---

## Step 2: Apply IAM Schema Migration

```bash
cd /Users/eganpj/GitHub/semlayer

# Apply IAM schema
psql -U postgres -d alpha -f backend/internal/migrations/011_iam_schema.sql
```

**Expected output:**
```
CREATE SCHEMA
CREATE TABLE
...
IAM Schema Created
role_count: 2
permission_count: 9
```

---

## Step 3: Create Debezium Publication (as superuser)

```bash
psql -U postgres -d alpha <<EOF
-- Create publication
CREATE PUBLICATION iam_security_publication FOR TABLE 
    iam.roles,
    iam.permissions,
    iam.role_permissions,
    iam.user_roles,
    iam.security_events;

-- Verify
SELECT * FROM pg_publication WHERE pubname = 'iam_security_publication';
EOF
```

---

## Step 4: Start Debezium and RabbitMQ

### 4.1 Start Services

```bash
docker-compose -f docker-compose.debezium.yml up -d debezium rabbitmq
```

### 4.2 Verify Debezium is Running

```bash
docker logs semlayer-debezium

# Should see:
# "Snapshot completed"
# "Streaming started"
```

### 4.3 Verify RabbitMQ

Open RabbitMQ Management UI: http://localhost:15672
- Username: `guest`
- Password: `guest`

Check for:
- Exchange: `security_sync_exchange`
- Queues: `security_sync.postgresql`, `security_sync.hasura`, etc.

---

## Step 5: Build and Start Sync Worker

### 5.1 Build Worker

```bash
cd backend
go build -o security-sync-worker ./cmd/security-sync-worker
```

### 5.2 Start Worker (Development)

```bash
export RABBITMQ_URL="amqp://guest:guest@localhost:5672/"
export POSTGRES_URL="postgres://postgres:postgres@localhost:5432/alpha"
export HASURA_URL="http://localhost:8080"
export HASURA_ADMIN_SECRET="myadminsecretkey"
export SUPERSET_URL="http://localhost:8088"
export SUPERSET_USERNAME="admin"
export SUPERSET_PASSWORD="admin"

./security-sync-worker
```

### 5.3 Start Worker (Docker)

```bash
docker-compose -f docker-compose.debezium.yml up -d security-sync-worker
```

---

## Step 6: Test the System

### 6.1 Create a Test Role

```bash
curl -X POST http://localhost:8080/api/roles \
  -H "Content-Type: application/json" \
  -H "Cookie: session_token=YOUR_SESSION_TOKEN" \
  -d '{
    "role_name": "data_analyst",
    "description": "Data analyst role with read access",
    "is_global_admin": false
  }'
```

### 6.2 Monitor Event Flow

**Check Debezium logs:**
```bash
docker logs -f semlayer-debezium
```

**Check Sync Worker logs:**
```bash
docker logs -f semlayer-security-sync
```

**Check RabbitMQ:**
- Go to http://localhost:15672/#/queues
- Click on `security_sync.postgresql`
- Check "Get messages"

### 6.3 Verify Synchronization

**PostgreSQL:**
```sql
-- Check if PostgreSQL role was created
SELECT rolname FROM pg_roles WHERE rolname LIKE 'tenant_%';
```

**Hasura:**
```bash
curl -X POST http://localhost:8080/v1/metadata \
  -H "x-hasura-admin-secret: myadminsecretkey" \
  -H "Content-Type: application/json" \
  -d '{"type": "export_metadata", "args": {}}'
```

Look for permissions for the new role.

---

## Step 7: Assign Role to User

```bash
curl -X POST http://localhost:8080/api/users/USER_ID/roles \
  -H "Content-Type: application/json" \
  -H "Cookie: session_token=YOUR_SESSION_TOKEN" \
  -d '{
    "role_id": "ROLE_ID_FROM_STEP_6"
  }'
```

**Expected behavior:**
1. User's JWT tokens are invalidated (forced re-login)
2. New login generates JWT with updated roles
3. Hasura permissions apply immediately
4. Superset RLS rules are created

---

## Step 8: Monitor Sync Status

### 8.1 Check Sync Status Table

```sql
SELECT 
    e.event_type,
    e.created_at,
    s.system,
    s.status,
    s.error_message
FROM iam.security_events e
JOIN iam.sync_status s ON e.event_id = s.event_id
WHERE e.processed = false
ORDER BY e.created_at DESC
LIMIT 10;
```

### 8.2 Check Replication Lag

```sql
SELECT 
    slot_name,
    pg_size_pretty(pg_wal_lsn_diff(pg_current_wal_lsn(), restart_lsn)) AS replication_lag
FROM pg_replication_slots
WHERE slot_name = 'iam_security_slot';
```

---

## Troubleshooting

### Issue: Debezium not starting

**Check:**
```bash
docker logs semlayer-debezium
```

**Common causes:**
- `wal_level` not set to `logical`
- Publication not created
- PostgreSQL not accessible from Docker

**Solution:**
```bash
# Restart PostgreSQL after changing wal_level
docker-compose restart postgres

# Recreate publication
psql -U postgres -d alpha -c "DROP PUBLICATION IF EXISTS iam_security_publication;"
psql -U postgres -d alpha -f debezium/postgres_setup.sql
```

### Issue: Sync worker not processing events

**Check RabbitMQ connection:**
```bash
docker logs semlayer-security-sync | grep "RabbitMQ"
```

**Check queue bindings:**
- Go to http://localhost:15672/#/exchanges
- Click `security_sync_exchange`
- Verify bindings to queues

### Issue: Events not appearing in RabbitMQ

**Check Debezium sink configuration:**
```bash
docker exec semlayer-debezium cat /debezium/conf/application.properties
```

**Verify publication:**
```sql
SELECT * FROM pg_publication_tables WHERE pubname = 'iam_security_publication';
```

---

## Production Deployment Checklist

- [ ] PostgreSQL `wal_level = logical` configured
- [ ] Debezium replication slot created
- [ ] RabbitMQ cluster configured with persistence
- [ ] Sync worker deployed with auto-restart
- [ ] Monitoring alerts configured for:
  - Replication lag > 1MB
  - Sync failures > 5%
  - Queue depth > 1000 messages
- [ ] Backup strategy for replication slot
- [ ] Log aggregation configured
- [ ] Security audit log retention policy set

---

## Performance Tuning

### PostgreSQL

```conf
# Increase WAL retention
wal_keep_size = 1GB

# Tune checkpoint settings
checkpoint_timeout = 15min
max_wal_size = 2GB
```

### RabbitMQ

```conf
# Increase message TTL if needed
message_ttl = 86400000  # 24 hours

# Tune prefetch count
consumer_prefetch_count = 10
```

### Sync Worker

```go
// Adjust QoS in consumeQueue function
err = ch.Qos(10, 0, false)  // Process 10 messages at a time
```

---

## Monitoring Queries

### Event Processing Rate

```sql
SELECT 
    event_type,
    COUNT(*) as total,
    COUNT(*) FILTER (WHERE processed = true) as processed,
    COUNT(*) FILTER (WHERE processed = false) as pending
FROM iam.security_events
WHERE created_at > now() - interval '1 hour'
GROUP BY event_type;
```

### Sync Success Rate

```sql
SELECT 
    system,
    status,
    COUNT(*) as count,
    ROUND(100.0 * COUNT(*) / SUM(COUNT(*)) OVER (PARTITION BY system), 2) as percentage
FROM iam.sync_status
WHERE synced_at > now() - interval '1 hour'
GROUP BY system, status
ORDER BY system, status;
```

---

## Next Steps

1. **Implement remaining sync worker handlers** (see TODOs in code)
2. **Add retry logic** for failed syncs
3. **Set up monitoring dashboards** (Grafana + Prometheus)
4. **Configure alerting** for sync failures
5. **Test disaster recovery** procedures
6. **Document runbooks** for common issues

---

## Support

For issues or questions:
1. Check logs: `docker logs semlayer-debezium` and `docker logs semlayer-security-sync`
2. Verify PostgreSQL publication: `SELECT * FROM pg_publication_tables;`
3. Check RabbitMQ queues: http://localhost:15672
4. Review sync status: `SELECT * FROM iam.sync_status WHERE status = 'failed';`
