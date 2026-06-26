# Temporal with External PostgreSQL - Configuration Guide

This guide provides a complete setup for running Temporal with an external PostgreSQL database, including Docker configuration, database initialization, and validation procedures.

## 1. Prepare Host PostgreSQL

### Configure Access
Edit `postgresql.conf` to enable external connections:
```ini
listen_addresses = '*'
```

Edit `pg_hba.conf` to allow Docker subnet connections:
```
host all all 172.17.0.0/16 md5
host all all 127.0.0.1/32 trust
```

Reload PostgreSQL:
```bash
pg_ctl reload
```

### Enable SSL (Optional, Recommended for Production)
1. Generate or place certificates in PostgreSQL data directory
2. Set in `postgresql.conf`:
```ini
ssl = on
ssl_cert_file = 'server.crt'
ssl_key_file = 'server.key'
```
3. Reload PostgreSQL
4. Use `sslmode=require` in connection strings when connecting from containers

## 2. Create Role and Databases (Idempotent)

Run this script to create the Temporal role and required databases:

```bash
DBURL="postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"

psql "$DBURL" <<'SQL'
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'temporal') THEN
    CREATE ROLE temporal LOGIN PASSWORD 'temporal';
  END IF;
END
$$;

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = 'temporal') THEN
    CREATE DATABASE temporal OWNER temporal;
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = 'temporal_visibility') THEN
    CREATE DATABASE temporal_visibility OWNER temporal;
  END IF;
END
$$;
SQL

# Grant privileges
for db in temporal temporal_visibility; do
  psql "postgres://postgres:postgres@localhost:5432/$db?sslmode=disable" <<'SQL'
ALTER DATABASE CURRENT_DATABASE() OWNER TO temporal;
GRANT ALL PRIVILEGES ON DATABASE CURRENT_DATABASE() TO temporal;
ALTER SCHEMA public OWNER TO temporal;
GRANT ALL ON SCHEMA public TO temporal;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO temporal;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO temporal;
SQL
done
```

## 3. Docker Setup Options

### Option A: Auto-Setup (Quick Start)
The `temporalio/auto-setup` image automatically initializes databases and starts the server:

```bash
docker rm -f temporal-host 2>/dev/null || true

docker run -d --name temporal-host \
  --add-host=host.docker.internal:host-gateway \
  -p 7233:7233 \
  -p 8088:8088 \
  -e DB=postgresql \
  -e DB_PORT=5432 \
  -e POSTGRES_SEEDS=host.docker.internal \
  -e POSTGRES_USER=temporal \
  -e POSTGRES_PWD=temporal \
  -e DBNAME=temporal \
  -e VISIBILITY_DBNAME=temporal_visibility \
  -e POSTGRES_CONNECT_ATTRIBUTES=sslmode=disable \
  temporalio/auto-setup:latest

sleep 5
docker logs -f temporal-host
```

### Option B: Manual Schema Application
For more control, use the admin-tools image:

```bash
# Core schema
docker run --rm --add-host=host.docker.internal:host-gateway \
  -e POSTGRES_USER=temporal -e POSTGRES_PWD=temporal \
  temporalio/admin-tools:latest \
  temporal-sql-tool --pl postgres12 --ep host.docker.internal -p 5432 \
  -u temporal --pw temporal --db temporal \
  setup-schema -v 0.0

# Apply versioned schemas
docker run --rm --add-host=host.docker.internal:host-gateway \
  -e POSTGRES_USER=temporal -e POSTGRES_PWD=temporal \
  temporalio/admin-tools:latest \
  temporal-sql-tool --pl postgres12 --ep host.docker.internal -p 5432 \
  -u temporal --pw temporal --db temporal \
  update-schema -d /etc/temporal/schema/postgresql.v12/temporal/versioned

# Visibility schema
docker run --rm --add-host=host.docker.internal:host-gateway \
  -e POSTGRES_USER=temporal -e POSTGRES_PWD=temporal \
  temporalio/admin-tools:latest \
  temporal-sql-tool --pl postgres12 --ep host.docker.internal -p 5432 \
  -u temporal --pw temporal --db temporal_visibility \
  setup-schema -v 0.0

docker run --rm --add-host=host.docker.internal:host-gateway \
  -e POSTGRES_USER=temporal -e POSTGRES_PWD=temporal \
  temporalio/admin-tools:latest \
  temporal-sql-tool --pl postgres12 --ep host.docker.internal -p 5432 \
  -u temporal --pw temporal --db temporal_visibility \
  update-schema -d /etc/temporal/schema/postgresql.v12/visibility/versioned
```

## 4. Docker Compose Configuration

Save the following as `docker-compose.temporal.yml`:

```yaml
version: "3.8"

services:
  temporal:
    image: temporalio/auto-setup:latest
    container_name: temporal-host
    extra_hosts:
      - "host.docker.internal:host-gateway"
    ports:
      - "7233:7233"  # gRPC
      - "8088:8088"  # HTTP/debug
    environment:
      - DB=postgresql
      - POSTGRES_SEEDS=host.docker.internal
      - POSTGRES_USER=temporal
      - POSTGRES_PWD=temporal
      - DB_PORT=5432
      - DBNAME=temporal
      - VISIBILITY_DBNAME=temporal_visibility
      - POSTGRES_CONNECT_ATTRIBUTES=sslmode=disable
      - LOG_LEVEL=info
    healthcheck:
      test: ["CMD", "tctl", "--address", "localhost:7233", "namespace", "list"]
      interval: 10s
      timeout: 5s
      retries: 3
      start_period: 20s
    restart: unless-stopped
    networks:
      - temporal-network

  temporal-ui:
    image: temporalio/ui:latest
    container_name: temporal-ui
    ports:
      - "8080:8080"
    environment:
      - TEMPORAL_ADDRESS=temporal:7233
    depends_on:
      - temporal
    restart: unless-stopped
    networks:
      - temporal-network

networks:
  temporal-network:
    driver: bridge
```

Run with:
```bash
docker-compose -f docker-compose.temporal.yml up -d
```

## 5. Validate Setup

### Check Logs
```bash
docker logs --tail 200 temporal-host
```

Look for `"Started Temporal server"` or `"SERVING"` message.

### Check Namespace
```bash
TEMPORAL_ADDRESS=localhost:7233 temporal namespace describe --namespace default
```

### Access UI
Open browser to `http://localhost:8080`

### Test with CLI
```bash
export TEMPORAL_ADDRESS=localhost:7233

# List namespaces
temporal namespace list

# Create a test workflow
temporal workflow execute --namespace default \
  --task-queue test-queue \
  --type TestWorkflow
```

## 6. Common Issues and Solutions

### Connection Failures
**Issue**: Container cannot connect to host Postgres

**Solutions**:
- Verify subnet in `pg_hba.conf`
- Test connectivity: `docker run --rm busybox ping host.docker.internal`
- Confirm Postgres is listening: `psql -h localhost -U postgres -c "SELECT 1"`

### Permission Denied Errors
**Issue**: Schema creation or queries fail with permission errors

**Solutions**:
- Verify temporal user ownership: `psql -h localhost -c "\l" -U postgres`
- Rerun database setup from Step 2
- Check role exists: `psql -h localhost -c "\du" -U postgres`

### Schema Version Errors
**Issue**: Schema version mismatch or missing tables

**Solutions**:
- Clear existing schemas and restart
- Manually run schema update: `temporal-sql-tool update-schema --force`
- Check schema version: `psql -h localhost -d temporal -c "SELECT version FROM schema_version;"`

### Port Already in Use
**Issue**: Port 7233 or 8088 already in use

**Solutions**:
- Kill existing process: `lsof -i :7233 | kill -9`
- Or use different ports in docker-compose

### SSL/TLS Issues (If Enabled)
**Issue**: Connection refused with SSL

**Solutions**:
- Verify `listen_addresses = '*'` in postgresql.conf
- Check SSL certificate permissions: `chmod 600 server.key`
- Use correct sslmode in connection string
- Test SSL locally: `psql -h localhost "postgres://postgres@localhost:5432/postgres?sslmode=require"`

## 7. Database Maintenance

### Backup Databases
```bash
# Backup both databases
pg_dump postgres://temporal:temporal@localhost:5432/temporal \
  | gzip > temporal_backup.sql.gz

pg_dump postgres://temporal:temporal@localhost:5432/temporal_visibility \
  | gzip > temporal_visibility_backup.sql.gz
```

### Restore Databases
```bash
gunzip -c temporal_backup.sql.gz | \
  psql postgres://temporal:temporal@localhost:5432/temporal

gunzip -c temporal_visibility_backup.sql.gz | \
  psql postgres://temporal:temporal@localhost:5432/temporal_visibility
```

### Monitor Database Size
```bash
psql postgres://temporal:temporal@localhost:5432/temporal -c \
  "SELECT datname, pg_size_pretty(pg_database_size(datname)) \
   FROM pg_database WHERE datname IN ('temporal', 'temporal_visibility');"
```

## 8. Integration with Semlayer

Once Temporal is running, integrate with your Semlayer backend:

```yaml
# config.yaml
temporal:
  address: "localhost:7233"
  namespace: "default"
  taskQueue: "semlayer-tasks"
  database:
    host: "host.docker.internal"
    port: 5432
    user: "temporal"
    password: "temporal"
    dbname: "temporal"
```

Test connection from backend:
```bash
temporal workflow list --namespace default
```

## Next Steps

1. **Deploy Workflows**: See [Workflow Orchestration Patterns](./workflow_orchestration_patterns.md)
2. **Monitor Health**: Set up Prometheus metrics and Grafana dashboards
3. **Scale**: Configure worker pools and queue priorities
4. **Backup Strategy**: Implement automated daily backups
5. **Security**: Enable SSL, implement role-based access control, configure firewall rules
