# Calendar Service: Deployment & Setup Guide

## 📋 Prerequisites

- Remote PostgreSQL 13+
- Hasura GraphQL Engine (deployed)
- Golang 1.21+ (for local development)
- Docker & Docker Compose (for local services)
- Node 18+ (for frontend)
- Git

## 🗄️ Phase 1: Database Setup (Remote PostgreSQL)

### Step 1: Connect to Remote PostgreSQL

```bash
# Using psql
psql -h postgres.example.com -U admin -d semlayer

# Or via tunnel (if in private network)
ssh -L 5432:postgres.example.com:5432 bastion.example.com
psql -h localhost -U admin -d semlayer
```

### Step 2: Create Schema

```bash
# Option A: Copy-paste entire docs/schema.sql
psql -h postgres.example.com -U admin -d semlayer < docs/schema.sql

# Option B: Create schema step-by-step
psql -h postgres.example.com -U admin -d semlayer -c "
CREATE EXTENSION IF NOT EXISTS uuid-ossp;
CREATE TABLE calendars (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    ...
);
"
```

### Step 3: Verify Schema

```bash
# Check tables created
psql -h postgres.example.com -U admin -d semlayer -c "\dt"

# Expected output:
# public | audit_log
# public | blackouts
# public | calendars
# public | external_calendar_connections
# public | profile_calendars
# public | schedule_profiles
```

### Step 4: Enable Row-Level Security (RLS)

```bash
# Set search_path for all connections
ALTER DATABASE semlayer SET search_path = public;

# Verify RLS enabled
psql -h postgres.example.com -U admin -d semlayer -c "
  SELECT schemaname, tablename, rowsecurity 
  FROM pg_tables 
  WHERE schemaname = 'public' AND tablename LIKE '%calendar%';
"
```

---

## 🌐 Phase 2: Hasura Integration (Remote)

### Step 1: Connect PostgreSQL to Hasura

1. Open Hasura Console: `https://hasura.example.com/console`
2. Go to **Data** → **Connect Database**
3. Choose **PostgreSQL** and enter connection string:
   ```
   postgresql://admin:password@postgres.example.com:5432/semlayer
   ```
4. Click **Connect Database**

### Step 2: Track Tables

1. In Hasura Console, go to **Data** → **[DB Name]** → **Tables**
2. Click "Track" for each table:
   - `calendars`
   - `schedule_profiles`
   - `profile_calendars`
   - `blackouts`
   - `external_calendar_connections`
   - `audit_log`

### Step 3: Configure Row-Level Security (RLS)

For **calendars** table:

1. Click `calendars` → **Permissions**
2. Add "User" role (or your custom role):
   - **Select**: ✅ (check permission)
   - **Custom check**: `tenant_id` equals `X-Hasura-Tenant-Id` header
   - Click **Save Permissions**

3. Repeat for other tables: `schedule_profiles`, `blackouts`, `audit_log`

### Step 4: Test GraphQL Query

In Hasura Console, open **GraphiQL**:

```graphql
query {
  calendars(limit: 10) {
    id
    name
    region
    holidays
  }
}
```

Add header: `X-Hasura-Tenant-Id: 550e8400-e29b-41d4-a716-446655440000`

Expected: Returns calendars for that tenant.

---

## 🐳 Phase 3: Local Development Setup

### Step 1: Clone & Navigate

```bash
cd /Users/eganpj/GitHub/semlayer/calendar-service
```

### Step 2: Update `.env.local`

Edit `.env.local` with your actual remote endpoints:

```bash
# Remote Services (on cloud/VPS)
HASURA_ENDPOINT=https://hasura.example.com/v1/graphql
HASURA_ADMIN_SECRET=your-admin-secret-here
POSTGRES_HOST=postgres.example.com
POSTGRES_USER=admin
POSTGRES_PASSWORD=your-secure-password
POSTGRES_DB=semlayer

# Local Services (in Docker Compose)
REDPANDA_BROKERS=redpanda:9092
REDIS_URL=redis://redis:6379
TEMPORAL_HOST=temporal
TEMPORAL_PORT=7233

# Service Configuration
ENVIRONMENT=local
SERVER_PORT=8081
LOG_LEVEL=debug
CACHE_TTL_MINUTES=60
ENABLE_TEMPORAL=false  # Set to true when ready for workflows
```

### Step 3: Start Local Services

```bash
# Start all local services (Redpanda, Debezium, Redis, Calendar Service)
make dev

# Expected output:
# Starting Calendar Service on port 8081
# Connected to Hasura at https://hasura.example.com/v1/graphql
# Connected to Redis at redis://redis:6379
# CDC processor connected to redpanda:9092
```

### Step 4: Test Health Check

```bash
curl http://localhost:8081/health

# Expected response:
# {"status":"healthy","timestamp":"2026-03-01T10:00:00Z"}
```

### Step 5: Test Calendar Endpoint

```bash
# Get all calendars for a tenant
curl -X GET http://localhost:8081/api/v1/calendars \
  -H "X-Hasura-Tenant-Id: 550e8400-e29b-41d4-a716-446655440000"

# Create a calendar
curl -X POST http://localhost:8081/api/v1/calendars \
  -H "X-Hasura-Tenant-Id: 550e8400-e29b-41d4-a716-446655440000" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Asia Holidays",
    "region": "APAC",
    "holidays": [{"date": "2026-02-10", "name": "Lunar New Year", "severity": "HIGH"}]
  }'
```

---

## 🔌 Phase 4: Debezium CDC Setup

### Goal
Capture changes from remote PostgreSQL → pipe to Redpanda → trigger Golang consumer

### Prerequisites
- Remote PostgreSQL with logical replication enabled
- Debezium container in compose (started via `make dev`)

### Step 1: Enable Logical Replication on Remote PostgreSQL

Connect as admin:

```sql
-- Set variables in postgresql.conf (SSH to DB server)
wal_level = logical
max_wal_senders = 10
max_replication_slots = 10
```

Restart PostgreSQL:

```bash
# On remote server
sudo systemctl restart postgresql

# Or if using AWS RDS: Modify parameter group, set wal_level = logical
```

### Step 2: Create Replication Slot & User

```sql
-- Connect to remote PostgreSQL as admin
SELECT * FROM pg_create_logical_replication_slot('semlayer_slot', 'pgoutput');

-- Create replication user
CREATE USER replication_user WITH REPLICATION ENCRYPTED PASSWORD 'replication_password';
GRANT CONNECT ON DATABASE semlayer TO replication_user;
GRANT USAGE ON SCHEMA public TO replication_user;
GRANT SELECT ON ALL TABLES IN SCHEMA public TO replication_user;
```

### Step 3: Register Debezium Connector

```bash
# Check Debezium is running
curl http://localhost:8083/connectors

# Create PostgreSQL connector
curl -X POST http://localhost:8083/connectors \
  -H "Content-Type: application/json" \
  -d '{
    "name": "semlayer-postgres-cdc",
    "config": {
      "connector.class": "io.debezium.connector.postgresql.PostgresConnector",
      "database.hostname": "postgres.example.com",
      "database.port": "5432",
      "database.user": "replication_user",
      "database.password": "replication_password",
      "database.dbname": "semlayer",
      "database.server.name": "semlayer_postgres",
      "table.include.list": "public.calendars,public.schedule_profiles,public.blackouts",
      "plugin.name": "pgoutput",
      "publication.name": "semlayer_publication",
      "slot.name": "semlayer_slot"
    }
  }'
```

### Step 4: Verify Debezium Running

```bash
# Check connector status
curl http://localhost:8083/connectors/semlayer-postgres-cdc/status

# Expected: "state": "RUNNING"

# Check Redpanda topics created
rpk topic list

# Expected topics (auto-created by Debezium):
# semlayer_postgres.public.calendars
# semlayer_postgres.public.schedule_profiles
# semlayer_postgres.public.blackouts
```

### Step 5: Monitor CDC

```bash
# Start log watcher
make logs

# Or in separate terminal, watch Redpanda consumer
docker-compose logs -f redpanda
```

---

## 🚀 Phase 5: Production Deployment

### Option A: Deploy to Kubernetes (Recommended)

#### 1. Build Docker Image

```bash
cd calendar-service
docker build -t myregistry.azurecr.io/calendar-service:v1.0.0 .
docker push myregistry.azurecr.io/calendar-service:v1.0.0
```

#### 2. Create Kubernetes Manifests

Create `k8s/calendar-service.yaml`:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: calendar-service
  namespace: production
spec:
  replicas: 3  # High availability
  selector:
    matchLabels:
      app: calendar-service
  template:
    metadata:
      labels:
        app: calendar-service
    spec:
      containers:
      - name: calendar-service
        image: myregistry.azurecr.io/calendar-service:v1.0.0
        ports:
        - containerPort: 8081
        env:
        - name: HASURA_ENDPOINT
          valueFrom:
            configMapKeyRef:
              name: calendar-config
              key: hasura-endpoint
        - name: HASURA_ADMIN_SECRET
          valueFrom:
            secretKeyRef:
              name: calendar-secrets
              key: hasura-admin-secret
        # ... more env vars
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8081
          initialDelaySeconds: 10
          periodSeconds: 10

---
apiVersion: v1
kind: Service
metadata:
  name: calendar-service
  namespace: production
spec:
  type: ClusterIP
  ports:
  - port: 80
    targetPort: 8081
  selector:
    app: calendar-service
```

#### 3. Deploy to Cluster

```bash
kubectl apply -f k8s/calendar-service.yaml

# Monitor rollout
kubectl rollout status deployment/calendar-service -n production

# Check pods
kubectl get pods -n production -l app=calendar-service
```

### Option B: Deploy to AWS ECS

See `docs/ecs-deployment.md`

### Option C: Deploy to Azure Container Apps

See `docs/azure-deployment.md`

---

## ✅ Verification Checklist

- [ ] Remote PostgreSQL schema created
- [ ] Hasura connected to PostgreSQL
- [ ] Hasura tables tracked
- [ ] Hasura RLS permissions configured
- [ ] Local services started (`make dev`)
- [ ] Health check passing (curl /health)
- [ ] Calendar endpoints working (curl /api/v1/calendars)
- [ ] Debezium connector registered
- [ ] CDC topics in Redpanda
- [ ] Golang consumer processing events
- [ ] Production image built
- [ ] Production deployment tested

---

## 🐛 Troubleshooting

### Issue: Cannot connect to remote PostgreSQL

```bash
# Check connectivity
telnet postgres.example.com 5432

# Check credentials
psql -h postgres.example.com -U admin -d semlayer

# Check firewall rules (AWS Security Groups, GCP Firewall, etc.)
```

### Issue: Hasura GraphQL queries return null

```bash
# Check RLS policy
SELECT * FROM pg_policies WHERE tablename = 'calendars';

# Manually set header in curl
curl -X POST https://hasura.example.com/v1/graphql \
  -H "X-Hasura-Tenant-Id: 550e8400-e29b-41d4-a716-446655440000" \
  -H "Content-Type: application/json" \
  -d '{"query":"query { calendars { id name } }"}'
```

### Issue: Debezium connector not running

```bash
# Check logs
docker-compose logs debezium

# Check connector config
curl http://localhost:8083/connectors/semlayer-postgres-cdc

# Delete and recreate
curl -X DELETE http://localhost:8083/connectors/semlayer-postgres-cdc
# Then re-register
```

### Issue: Services can't reach each other in Docker Compose

```bash
# Check networks
docker network ls

# Inspect network
docker network inspect semlayer_semlayer

# Verify DNS
docker-compose exec calendar-service nslookup redis

# Test connectivity from calendar-service
docker-compose exec calendar-service curl http://redis:6379/ping
```

---

## 📞 Support

- **Local Development Issues**: Run `make logs` and check error messages
- **Remote PostgreSQL Issues**: Check PostgreSQL logs on remote server
- **Hasura Issues**: Check Hasura Console for query errors
- **Debezium Issues**: Check Debezium REST API responses
- **Production Issues**: Check pod logs `kubectl logs -f <pod-name>`

---

## 🔐 Security Checklist

- [ ] All secrets in environment variables (not in code)
- [ ] HTTPS enabled for Hasura endpoint
- [ ] PostgreSQL logical replication user has minimal permissions
- [ ] Redpanda in private network (not exposed to internet)
- [ ] Kubernetes RBAC configured
- [ ] Network policies restrict traffic
- [ ] Audit logging enabled
- [ ] Secrets rotated regularly

---

Next: [QUICKSTART.md](./QUICKSTART.md) for local development workflow
