# Quick Start Guide for Localhost PostgreSQL

Since PostgreSQL runs on localhost (not in Docker), follow these steps:

## 1. Configure PostgreSQL for Logical Replication

Edit your local PostgreSQL configuration file:

**macOS (Homebrew):**
```bash
# Find config file
psql -U postgres -c "SHOW config_file;"

# Edit the file (usually /opt/homebrew/var/postgresql@14/postgresql.conf)
nano $(psql -U postgres -t -c "SHOW config_file;")
```

**Add these lines:**
```conf
wal_level = logical
max_replication_slots = 10
max_wal_senders = 10
```

**Restart PostgreSQL:**
```bash
brew services restart postgresql@14
# or
pg_ctl restart -D /opt/homebrew/var/postgresql@14
```

## 2. Apply IAM Schema

```bash
cd /Users/eganpj/GitHub/semlayer
psql -U postgres -d alpha -f backend/internal/migrations/011_iam_schema.sql
```

## 3. Create Publication

```bash
psql -U postgres -d alpha <<EOF
CREATE PUBLICATION iam_security_publication FOR TABLE 
    iam.roles,
    iam.permissions,
    iam.role_permissions,
    iam.user_roles,
    iam.security_events;
EOF
```

## 4. Configure PostgreSQL to Accept Connections from Docker

Edit `pg_hba.conf`:

```bash
# Find pg_hba.conf
psql -U postgres -c "SHOW hba_file;"

# Edit it
nano $(psql -U postgres -t -c "SHOW hba_file;")
```

**Add this line:**
```conf
# Allow Docker containers to connect
host    all             postgres        172.16.0.0/12           trust
host    replication     postgres        172.16.0.0/12           trust
```

**Reload configuration:**
```bash
psql -U postgres -c "SELECT pg_reload_conf();"
```

## 5. Start Debezium and RabbitMQ

```bash
docker-compose -f docker-compose.debezium.yml up -d
```

## 6. Verify Debezium Connection

```bash
# Check Debezium logs
docker logs semlayer-debezium

# Should see:
# "Connected to PostgreSQL"
# "Snapshot started"
# "Snapshot completed"
```

## 7. Test the System

### Create a test role:
```bash
# First, get a session token by logging in
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "admin@uisce.com", "password": "admin123"}' \
  > /tmp/login.json

# Extract token
TOKEN=$(cat /tmp/login.json | jq -r '.access_token')

# Create role
curl -X POST http://localhost:8080/api/roles \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "role_name": "data_analyst",
    "description": "Data analyst with read access",
    "is_global_admin": false
  }'
```

### Monitor the event flow:

**1. Check if event was created in PostgreSQL:**
```bash
psql -U postgres -d alpha -c "SELECT * FROM iam.security_events ORDER BY created_at DESC LIMIT 1;"
```

**2. Check Debezium logs:**
```bash
docker logs -f semlayer-debezium | grep "iam.roles"
```

**3. Check RabbitMQ:**
```bash
# Open browser to http://localhost:15672
# Login: guest/guest
# Go to Queues tab
# Click on security_sync.postgresql
# Check "Get messages"
```

**4. Check sync worker logs:**
```bash
docker logs -f semlayer-security-sync
```

## Troubleshooting

### Issue: Debezium can't connect to PostgreSQL

**Check PostgreSQL is listening on all interfaces:**
```bash
psql -U postgres -c "SHOW listen_addresses;"
# Should be '*' or '0.0.0.0'
```

**If not, edit postgresql.conf:**
```conf
listen_addresses = '*'
```

**Restart PostgreSQL:**
```bash
brew services restart postgresql@14
```

### Issue: Permission denied for replication

**Grant replication permission:**
```bash
psql -U postgres -d alpha <<EOF
ALTER USER postgres WITH REPLICATION;
EOF
```

### Issue: Replication slot not created

**Check if publication exists:**
```bash
psql -U postgres -d alpha -c "SELECT * FROM pg_publication;"
```

**Manually create replication slot:**
```bash
psql -U postgres -d alpha -c "SELECT pg_create_logical_replication_slot('iam_security_slot', 'pgoutput');"
```

## Quick Verification Commands

```bash
# Check PostgreSQL wal_level
psql -U postgres -c "SHOW wal_level;"

# Check publication
psql -U postgres -d alpha -c "SELECT * FROM pg_publication_tables WHERE pubname = 'iam_security_publication';"

# Check replication slots
psql -U postgres -d alpha -c "SELECT * FROM pg_replication_slots;"

# Check Debezium status
docker logs semlayer-debezium | tail -20

# Check RabbitMQ queues
curl -u guest:guest http://localhost:15672/api/queues | jq '.[] | {name: .name, messages: .messages}'
```

## Success Indicators

✅ PostgreSQL `wal_level = logical`
✅ Publication `iam_security_publication` exists with 5 tables
✅ Debezium logs show "Streaming started"
✅ RabbitMQ has 4 queues with bindings
✅ Sync worker logs show "Worker started, waiting for messages"
✅ Creating a role triggers events in all systems
