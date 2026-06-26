# Complete MDM Implementation Roadmap

## Timeline: 2-4 Weeks to Full Production System

---

# PHASE 1: Free Calendar Sources + Core Infrastructure
**Timeline:** 2-4 hours | **Status:** NEXT ← YOU ARE HERE

## Deliverables
- ✅ 4 free data sources ingesting (Nager, OpenHolidays, Workalendar, Holidays PyPI)
- ✅ Docker Compose stack running (9 services)
- ✅ Golden calendar populated with 300+ records
- ✅ Calendar service querying MDM successfully
- ✅ Lineage & audit trail working

## Success Criteria
```bash
# By end of Phase 1, these should all pass:
✅ docker-compose ps → All 9 containers "Up"
✅ curl http://localhost:8000/health → healthy
✅ SELECT COUNT(*) FROM edm.mdm_calendar_source → 300+
✅ SELECT COUNT(*) FROM edm.mdm_calendar_golden → 250+
✅ curl http://localhost:8080/api/v1/calendar/is-business-day?date=2026-12-25 → false
```

## Start Here
→ **Follow:** [PHASE_1_DEPLOYMENT_GUIDE.md](./PHASE_1_DEPLOYMENT_GUIDE.md)

---

# PHASE 2: Real-Time Event Streaming
**Timeline:** 4-6 hours | **Start:** After Phase 1 passes

## Objective
Connect Redpanda message broker so downstream systems get real-time calendar updates.

## Deliverables
- [ ] Calendar events published to Redpanda topics
- [ ] Event schema (Protobuf/Avro) defined
- [ ] Example consumer (trading platform simulator)
- [ ] React subscription component (real-time updates)

## Implementation Tasks

### 2.1 Event Publisher Implementation (2 hours)
```go
// File: internal/publisher/calendar_events.go
- Create CalendarEventPublisher struct
- Implement PublishCalendarUpdate() method
- Add event routing by tenant ID (order guarantee)
- Error handling with retry logic
- Metrics tracking
```

### 2.2 Event Schema Definition (1 hour)
```protobuf
// File: proto/calendar_events.proto
syntax = "proto3";

message CalendarEvent {
  string event_id = 1;
  string event_type = 2;        // CALENDAR_UPDATE, HOLIDAY_ADDED, etc
  string tenant_id = 3;
  string region = 4;
  string calendar_date = 5;     // ISO 8601
  bool is_business_day = 6;
  string holiday_name = 7;
  string source_system = 8;
  int32 confidence_score = 9;
  string operation = 10;        // CREATE, UPDATE, DELETE
  int64 timestamp = 11;
}
```

### 2.3 Event Publishing Integration (1 hour)
```go
// Location: internal/mdm/orchestrator.go
// Modify: runSurvivorship() method
// Add: After golden record upsert, publish event
err := o.publishCalendarEvent(ctx, tenantID, result.Changes)
```

### 2.4 Downstream Consumer Example (1 hour)
```go
// File: services/trading-consumer/main.go
// Shows how trading platform consumes calendar events
// Updates its own calendar cache in real-time
// Example log: "Received CALENDAR_UPDATE for US/2026-12-25"
```

### 2.5 React Real-Time Subscription (1 hour)
```tsx
// File: frontend/src/hooks/useCalendarSubscription.ts
// Apollo Client subscription to calendar events
// Auto-updates displayed calendar as events arrive
useSubscription(CALENDAR_UPDATES_SUBSCRIPTION)
```

## Success Metrics
```bash
✅ Redpanda console shows calendar-updates topic
✅ Events published: docker-compose logs api-gateway | grep "event_published"
✅ Consumer receives events: docker-compose logs trading-consumer | grep "CALENDAR_UPDATE"
✅ React component updates without page reload
```

## Container Changes
```yaml
# Add to docker-compose.mdm.yml:
- trading-consumer:9999  # Example consumer
- Frontend updated with WebSocket for subscriptions
```

---

# PHASE 3: Commercial Sources + Production Hardening
**Timeline:** 4-6 hours | **Start:** After Phase 2 passes

## Objective
Activate commercial data sources and harden system for enterprise reliability.

## Deliverables
- [ ] TradingHours, EODHD, Xignite integrated
- [ ] Automated failover from primary to backup sources
- [ ] Health monitoring & alerting
- [ ] Performance optimization (caching, indexing)
- [ ] Multi-tenant isolation verified
- [ ] Backup & recovery procedures

## Implementation Tasks

### 3.1 Commercial Source Activation (2 hours)
```sql
-- File: schema/commercial_sources_setup.sql
-- Update mdm_source_registry to activate

-- Environment setup
export TRADINGHOURS_API_KEY="your-key"
export EODHD_API_KEY="your-key"
export XIGNITE_API_KEY="your-key"

-- In UI: Toggle sources on in Ops Console
-- Or via API:
PATCH /api/v1/mdm/sources/{source_id}
{
  "is_active": true,
  "api_key_secret_name": "tradinghours/prod"
}
```

### 3.2 Source Failover Logic (1 hour)
```go
// File: internal/mdm/failover.go
type FailoverStrategy struct {
  Primary   SourceConfig
  Secondary SourceConfig
  Tertiary  SourceConfig
}

// Method: SelectBestSource()
// Returns source based on health checks
// Falls back if primary fails
```

### 3.3 Health Monitoring (1 hour)
```go
// File: internal/observability/calendar_monitor.go
- Source health checks every 5 minutes
- Confidence score validation
- Data quality metrics
- Completeness checks (% of calendar covered)
- Conflict rate monitoring
- Alert thresholds
```

### 3.4 Performance Optimization (1 hour)
```sql
-- File: schema/indexes_and_optimization.sql
CREATE INDEX idx_calendar_golden_tenant_date 
  ON edm.mdm_calendar_golden(tenant_id, calendar_date);

CREATE INDEX idx_calendar_source_registry_active 
  ON edm.mdm_source_registry(is_active, priority_score);

-- Partitioning by tenant_id for multi-tenant efficiency
PARTITION BY LIST (tenant_id)
```

### 3.5 Multi-Tenant Validation (1 hour)
```bash
# File: tests/integration/multitenant_test.go
- Create separate test tenants
- Verify data isolation via RLS
- Confirm querying one tenant doesn't leak another's data
- Performance with N tenants
- Cache key separation
```

## Prerequisites
```bash
# Get API keys from providers
- TradingHours: https://www.tradinghours.com/signup
- EODHD: https://eodhd.com/register
- Xignite: https://www.xignite.com/contact

# Store securely
export TRADINGHOURS_API_KEY="..."  # In .env or Vault
```

## Success Metrics
```bash
✅ TradingHours activated: is_active=true in db
✅ High-quality data ingested: SELECT COUNT(*) FROM edm.mdm_calendar_source WHERE source_system='TradingHours'
✅ Failover tested: Primary source down, secondary takes over
✅ Health checks passing: /health endpoint returns all sources status
✅ Multi-tenant verified: SELECT COUNT(DISTINCT tenant_id) → 3+ tenants
✅ Query performance: P95 latency < 100ms
```

## Container Updates
```yaml
# In docker-compose.mdm.yml:
# Add monitoring services if not present:
- prometheus:9090  (metrics collection)
- grafana:3000     (visualization)
- alertmanager     (alert routing)
```

---

# Summary: From Now to Production

## Week 1
- **Day 1:** Complete Phase 1 (free sources)
- **Day 2:** Phase 2 start (event streaming)
- **Day 3:** Phase 2 complete + Phase 3 prep
- **Day 4:** Phase 3 implementation begins

## Week 2
- **Day 5-6:** Phase 3 hardening
- **Day 7:** Integration testing
- **Day 8-9:** Production readiness review
- **Day 10:** Production deployment

## Week 3-4
- Day 11-15: Monitoring & tuning
- Day 16-20: Extended validation
- Day 21-28: Support & optimization

---

# Quick Reference: Key Files by Phase

## Phase 1 Files
```
calendar-service/
├── schema/001_mdm_init.sql          ← Database
├── docker-compose.mdm.yml           ← Infrastructure
├── internal/mdm/orchestrator.go     ← Ingestion engine
├── internal/mdm/handler.go          ← API handlers
├── services/workalendar-adapter/    ← Python service
├── services/holidays-adapter/       ← Python service
└── PHASE_1_DEPLOYMENT_GUIDE.md      ← YOU ARE HERE
```

## Phase 2 Additional Files
```
calendar-service/
├── internal/publisher/redpanda.go           ← Event publisher
├── services/trading-consumer/               ← Example consumer
├── frontend/src/hooks/useCalendarSub.ts     ← React subscriptions
└── proto/calendar_events.proto              ← Event schema
```

## Phase 3 Additional Files
```
calendar-service/
├── schema/commercial_sources_setup.sql      ← API keys
├── internal/mdm/failover.go                 ← Failover logic
├── internal/observability/monitor.go        ← Health checks
├── tests/integration/multitenant_test.go    ← Validation
└── docker-compose.production.yml            ← Prod config
```

---

# Architecture Evolution

## Phase 1: Core
```
Nager/OpenHolidays (API) ─→ ┐
Workalendar (Python)       ├→ Semantic Engine → MDM Golden → Calendar Service
Holidays PyPI (Python)     ┘
```

## Phase 2: +Events
```
[Phase 1] ─→ Redpanda ─→ Trading Consumer
                     ├→ Analytics System
                     └→ React Frontend (subscriptions)
```

## Phase 3: +Enterprise
```
[Phase 2] + TradingHours ─┐
         + EODHD       ├→ Failover Logic ─→ [Phase 2 + monitoring]
         + Xignite    ─┘
                    + Health Checks
                    + Multi-tenant Validation
                    + Performance Optimization
```

---

# Go/No-Go Decision Gates

### Phase 1 Go/No-Go
- [ ] 250+ golden records populated
- [ ] All 4 free sources active
- [ ] Zero errors in logs
- [ ] Calendar service test passes
- **Decision:** Continue to Phase 2? → YES → Proceed

### Phase 2 Go/No-Go
- [ ] Events published to Redpanda
- [ ] Consumer receives events
- [ ] React component updates in real-time
- [ ] Performance acceptable (< 500ms)
- **Decision:** Continue to Phase 3? → YES → Proceed

### Phase 3 Go/No-Go
- [ ] Commercial sources active
- [ ] Data quality from commercial sources validated
- [ ] Failover tested and working
- [ ] Multi-tenant isolation verified
- **Decision:** Deploy to production? → YES → Launch

---

# Support & Escalation

## Phase 1 Issues
→ [PHASE_1_DEPLOYMENT_GUIDE.md](./PHASE_1_DEPLOYMENT_GUIDE.md) - Common Issues section

## Phase 2 Issues
→ Will be documented in PHASE_2_EVENT_STREAMING.md (creates when you start Phase 2)

## Phase 3 Issues
→ Will be documented in PHASE_3_PRODUCTION_HARDENING.md (creates when you start Phase 3)

---

**You are here:** PHASE 1 - START WITH [PHASE_1_DEPLOYMENT_GUIDE.md](./PHASE_1_DEPLOYMENT_GUIDE.md)

Good luck! 🚀
