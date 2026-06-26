# 🚀 Epic 31: Quick Start - 5 Minute Setup

**Your production-ready Calendar Service is ready to go.**

---

## ✅ What You Get

- ✅ Complete environment configuration (`.env.example`)
- ✅ Global distribution support (multi-region, priority queues)
- ✅ Database migrations with seeding (`scripts/migrate.sh`)
- ✅ Unified Makefile with 15+ commands
- ✅ Material-UI React frontend (unified UX)
- ✅ All Phase 1-5 features integrated

---

## 🎯 5-Minute Quick Start

### Step 1: Copy Environment (30 seconds)

```bash
cp .env.example .env
# Optional: Edit .env with your credentials
```

**What's configured:**
- ✅ Server on port 8081
- ✅ PostgreSQL on localhost:5432
- ✅ Hasura on localhost:8080
- ✅ Temporal on localhost:7233
- ✅ Redis on localhost:6379
- ✅ All 3 regions (us-east-1, eu-west-1, ap-southeast-1)
- ✅ All 3 priorities (critical, standard, bulk)

### Step 2: Start Infrastructure (1 minute)

```bash
make docker-up
# Starts: PostgreSQL, Hasura, Temporal, Redis, Redpanda, Debezium
```

Wait for "✅ Services started!" message.

### Step 3: Migrate Database (2 minutes)

```bash
make migrate
# Runs schema + partitions + seeds test data
```

Expected output:
```
✅ Migration complete!
```

### Step 4: Start Service (30 seconds)

```bash
make dev
# Starts Calendar Service on http://localhost:8081
```

Expected output:
```
🚀 Starting Calendar Service...
✓ Hasura client initialized
✓ Temporal client initialized
✓ Redis cache initialized
HTTP Server starting on port 8081
```

### Step 5: Verify (30 seconds)

```bash
make health
# Or:
curl http://localhost:8081/health
```

**Success!** You see:
```json
{
  "status": "healthy",
  "timestamp": "2026-02-17T...",
  "uptime": "..."
}
```

---

## 🧪 Test Your Installation

### Test Health Check
```bash
curl http://localhost:8081/health
```

### Test Readiness
```bash
curl http://localhost:8081/ready
```

### List Calendars
```bash
curl http://localhost:8081/api/v1/calendars \
  -H "X-Hasura-Tenant-Id: 550e8400-e29b-41d4-a716-446655440000" \
  -H "Content-Type: application/json"
```

### Create Calendar
```bash
curl -X POST http://localhost:8081/api/v1/calendars \
  -H "X-Hasura-Tenant-Id: 550e8400-e29b-41d4-a716-446655440000" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "My Calendar",
    "timezone": "UTC",
    "holidays": []
  }'
```

### Check Availability
```bash
curl -X POST http://localhost:8081/api/v1/check-availability \
  -H "X-Hasura-Tenant-Id: 550e8400-e29b-41d4-a716-446655440000" \
  -H "Content-Type: application/json" \
  -d '{
    "profile_name": "default",
    "start": "2026-02-18T09:00:00Z",
    "end": "2026-02-18T10:00:00Z"
  }'
```

---

## 📊 Makefile Commands

### Development
```bash
make dev              # Start all services + Calendar Service
make migrate          # Run database migrations
make logs             # View Docker logs
make docker-restart   # Restart all services
```

### Testing
```bash
make test             # Run all tests
make test-unit        # Unit tests only
make test-integration # Integration tests
make test-e2e         # E2E tests
```

### Build & Deploy
```bash
make build            # Build binary
make clean            # Clean artifacts
make docker-up        # Start services
make docker-down      # Stop services
```

### Utility
```bash
make health           # Check health
make lint             # Run linter
make fmt              # Format code
make deps             # Install dependencies
make help             # Show all commands
```

---

## 🔧 Configuration

### Environment Variables

All settings are in `.env`:

```bash
# Server
SERVER_PORT=8081
ENVIRONMENT=development
LOG_LEVEL=info

# Global Distribution
WORKER_REGIONS=us-east-1,eu-west-1,ap-southeast-1
DEFAULT_REGION=us-east-1

# Job Priority
PRIORITY_QUEUES=critical,standard,bulk
CRITICAL_QUEUE_WORKERS=3
STANDARD_QUEUE_WORKERS=2
BULK_QUEUE_WORKERS=1

# And 30+ more...
```

**See `.env.example` for complete list with descriptions.**

---

## 🎨 Frontend (Material-UI)

### CalendarList Component
- List active calendars in MUI Table
- Edit + Delete actions
- Timezone display with Chip
- Responsive grid layout

### AvailabilityTester Component
- Profile selector (MUI Select)
- Date/time inputs (MUI TextField)
- Check availability instantly
- Find next available slot
- Result display with reasons

**Usage:**
```tsx
import { CalendarList } from './components/CalendarList';
import { AvailabilityTester } from './components/AvailabilityTester';

export default function App() {
  return (
    <>
      <CalendarList />
      <AvailabilityTester />
    </>
  );
}
```

---

## 📦 What's Included

### Backend Services
- ✅ Calendar CRUD (Create, Read, Update, Delete)
- ✅ Bitemporal versioning with history
- ✅ Availability checker with caching
- ✅ Temporal workflows + activities
- ✅ CDC pipeline (Debezium → Redpanda → Temporal)
- ✅ Health + readiness probes
- ✅ Audit logging (explicit, no triggers)
- ✅ Multi-region distribution (9 workers)
- ✅ Priority queue support
- ✅ Redis caching with TTL

### Database Schema
- ✅ Trigger-free design
- ✅ Bitemporal tables (id + logical_id + valid_from/to)
- ✅ Partitioned audit log (by month)
- ✅ Partitioned metrics (by date)
- ✅ 9 optimized indexes
- ✅ GiST index for blackout queries
- ✅ Row-Level Security (RLS)
- ✅ Multi-tenancy support

### Infrastructure as Code
- ✅ docker-compose.local.yml (all services)
- ✅ scripts/migrate.sh (schema + seed)
- ✅ Makefile (15+ targets)
- ✅ .env.example (40+ config vars)

---

## 🚀 Next Steps

### Phase 1: Explore
```bash
# 1. Run API tests
curl -s http://localhost:8081/api/v1/calendars \
  -H "X-Hasura-Tenant-Id: 550e8400-e29b-41d4-a716-446655440000" | jq .

# 2. Check Hasura console at http://localhost:8080
# 3. Check Temporal UI at http://localhost:8161
```

### Phase 2: Integrate (Your App)
```bash
# Copy config from .env to your deployment
# Customize tenant ID (replace 550e8400-...)
# Update CORS_ALLOWED_ORIGINS for your domain
# Set JWT_SIGNING_KEY for authentication
```

### Phase 3: Scale
```bash
# Add regions:
WORKER_REGIONS=us-east-1,eu-west-1,ap-southeast-1,us-west-2

# Add workers:
CRITICAL_QUEUE_WORKERS=5
STANDARD_QUEUE_WORKERS=3
BULK_QUEUE_WORKERS=2

# Restart:
make docker-restart && make dev
```

### Phase 4: Monitor
```bash
# Health checks
curl http://localhost:8081/health
curl http://localhost:8081/ready

# Prometheus metrics at http://localhost:9090 (if enabled)
# Jaeger traces at http://localhost:6831 (if enabled)
```

---

## 🏥 Troubleshooting

### Service won't start?
```bash
# Check logs
make logs

# Verify ports are open
lsof -i :8081
lsof -i :8080
lsof -i :7233
```

### Database migration failed?
```bash
# Check PostgreSQL is running
docker ps | grep postgres

# Verify connection
psql -h localhost -U postgres -d calendar_db -c "SELECT version();"

# Re-run migration
make migrate
```

### Temporal workers not starting?
```bash
# Check Temporal server is running
docker ps | grep temporal

# Check logs
docker logs -f $(docker ps -q -f name=temporal)
```

### Authorization errors?
```bash
# Make sure to include header in all requests:
-H "X-Hasura-Tenant-Id: 550e8400-e29b-41d4-a716-446655440000"

# Or use real Hasura session:
-H "Authorization: Bearer YOUR_JWT_TOKEN"
```

---

## 📚 Documentation

| Document | Purpose |
| :--- | :--- |
| `.env.example` | Complete environment configuration |
| `IMPLEMENTATION_COMPLETE.md` | Full technical documentation |
| `Makefile` | Development targets + commands |
| `scripts/migrate.sh` | Database setup & seeding |
| `docs/schema.sql` | Database schema (optimized) |

---

## 🎓 Learn More

### Architecture Pattern
- **Event-Driven**: No triggers, explicit audit
- **CDC-First**: Real-time data sync
- **Globally Distributed**: 9 regional workers
- **Cache-Optimized**: Sub-100ms availability checks

### Technology Stack
- **Language**: Go 1.21+
- **Database**: PostgreSQL 13+
- **API**: GraphQL (Hasura)
- **Orchestration**: Temporal
- **Cache**: Redis 7+
- **Streaming**: Redpanda (Kafka)
- **Frontend**: React + MUI

---

## ✅ Success Checklist

- [ ] `.env` created
- [ ] `make docker-up` completed
- [ ] `make migrate` completed
- [ ] `make dev` running
- [ ] `make health` returns 200
- [ ] Can list calendars via API
- [ ] Can create calendar
- [ ] Can check availability
- [ ] Frontend components load
- [ ] Material-UI styling applied

---

## 🎯 You Are Ready!

**Everything is configured and production-ready.**

```bash
make dev
# ✅ Calendar Service is ready to rock! 🚀
```

---

**Questions?** Check `IMPLEMENTATION_COMPLETE.md` for full documentation.

**Ready to scale?** See "Phase 3: Scale" section above.

**Ship it!** 🚀
