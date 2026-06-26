# Usice MDM - Architecture Overview

## System Diagram

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          Frontend Layer (React)                              │
├─────────────────────────────────────────────────────────────────────────────┤
│  Ops Console (CalendarSourcesPanel.tsx)                                      │
│  - Ingestion control (year, regions)                                         │
│  - Source management (toggle activate/deactivate)                            │
│  - Job monitoring (status, records, conflicts)                               │
│  - Real-time subscriptions (GraphQL)                                         │
└─────────────────────────────────────────────────────────────────────────────┘
                                    ↓ (GraphQL + REST)
┌─────────────────────────────────────────────────────────────────────────────┐
│                        API Gateway Layer (Go)                                │
├─────────────────────────────────────────────────────────────────────────────┤
│  Endpoints:                                                                  │
│  • POST /api/v1/mdm/calendar/ingest       (Trigger ingestion cycle)        │
│  • GET  /api/v1/mdm/sources               (List all sources + status)      │
│  • PATCH /api/v1/mdm/sources/{id}/activate/deactivate                      │
│  • GET  /api/v1/calendar/golden           (Query golden calendar)           │
│  • GET  /api/v1/calendar/is-business-day  (Check single date)              │
│  • GET  /api/v1/mdm/conflicts             (List pending conflicts)          │
│                                                                              │
│  Features:                                                                   │
│  • X-Tenant-ID header isolation                                            │
│  • X-User-Role authorization                                               │
│  • Event publishing to Redpanda                                            │
└─────────────────────────────────────────────────────────────────────────────┘
                ↓ (Pubsub events)                          ↓ (SQL queries)
     ┌──────────────────────────┐              ┌──────────────────────────┐
     │  Redpanda Event Broker   │              │    PostgreSQL (External) │
     │  (Streaming Platform)    │              │   100.84.126.19:5432     │
     └──────────────────────────┘              └──────────────────────────┘
             ↓                                          ↑ ↓ ↓
      Topics:                                  ┌─────────────────────────┐
      • calendar-updates                       │   Semantic Layer        │
      • conflicts                              ├─────────────────────────┤
      • source-events                          │ semantic_terms (7)      │
      • ingestion-jobs                         │ business_objects (N)    │
                                               │ dsls/rules registry     │
     ↓ (Event stream)                         └─────────────────────────┘
     │                                                 ↑ ↓
     │                                         ┌─────────────────────────┐
     └──→ Downstream Systems                   │   Golden Records        │
         (Reporting, Analytics, ML)            ├─────────────────────────┤
                                               │ mdm_calendar_golden     │
                                               │ (authoritative source)  │
                                               └─────────────────────────┘
                                                        ↑ ↓
                                               ┌─────────────────────────┐
                                               │   Ingestion Pipeline    │
                                               ├─────────────────────────┤
└──────────────────────────────────────────────┤ 1. Source Registry      │
│ Semantic Engine (Go) & Rules Engine          │ 2. Fetch (4 active)     │
├──────────────────────────────────────────────┤ 3. Store (staging)      │
│ Orchestrator:                                │ 4. Survivorship         │
│ • getActiveSources()  (dynamic registry)     │ 5. Conflict detect      │
│ • fetchNagerDate()    (free, 100+ countries) │ 6. Publish events       │
│ • fetchOpenHolidays() (free, open data)      └─────────────────────────┘
│ • fetchPythonService()[Workalendar]          
│ • fetchPythonService()[HolidaysPyPI]         Data Sources:
│ • runSurvivorship()   (WASM rules)           ✓ NagerDate (active)
│ • detectConflicts()   (disagreement)         ✓ OpenHolidays (active)
│ • storeLineage()      (audit trail)          ✓ Workalendar (active)
│                                              ✓ HolidaysPyPI (active)
└──────────────────────────────────────────────┗ TradingHours (stub)
│                                               ✗ EODHD (stub)
┌───────────────────────────────────────────────┗ Xignite (stub)
│                                                ✗ Finnhub (stub)
│ Python Microservices:
├───────────────────────────────────────────────┐
│ Workalendar Service (Flask, Port 8000)        │
│ • GET /health                                 │
│ • GET /holidays?region=US&year=2026           │
│ • GET /is-holiday?region=US&date=2026-01-01  │
│ • Supports: US, GB, FR, DE, ES, JP, CN, AU   │
│                                               │
│ Holidays PyPI Service (Flask, Port 8001)      │
│ • Same endpoints as Workalendar               │
│ • Supports: 12+ countries + US states         │
│ • Region-aware: "US-CA", "US-TX", etc.       │
└───────────────────────────────────────────────┘
```

---

## Data Flow Diagram

### Ingestion Cycle

```
User Action (Frontend or API)
        ↓
  POST /api/v1/mdm/calendar/ingest
        ↓
  Publish: ingestion_started event
        ↓
┌─────────────────────────────────────┐
│   Get Active Sources from Registry   │  ← Dynamic! Can toggle without code
├─────────────────────────────────────┤
│ SELECT source_name FROM              │
│   mdm_source_registry                │
│ WHERE is_active = true               │
│ ORDER BY priority_score              │
└─────────────────────────────────────┘
        ↓
  For each region, year:
        ├─→ Fetch NagerDate (priority 4)
        │       ↓ parse JSON → store staging
        ├─→ Fetch OpenHolidays (priority 4)
        │       ↓ parse JSON → store staging
        ├─→ Fetch Workalendar via service (priority 3)
        │       ↓ HTTP → parse JSON → store staging
        └─→ Fetch HolidaysPyPI via service (priority 3)
                ↓ HTTP → parse JSON → store staging
        ↓
┌─────────────────────────────────────┐
│   Run Survivorship Rules (Go)        │
├─────────────────────────────────────┤
│ For each date:                       │
│ 1. Collect candidate values (4 src)  │
│ 2. Sort by priority_score (lower=🥇)│
│ 3. Tiebreak: confidence_score       │
│ 4. Select winner                     │
│ 5. Detect conflicts (disagreement)   │
│ 6. Calculate confidence (0-100)      │
└─────────────────────────────────────┘
        ├─→ INSERT INTO mdm_calendar_golden
        │         tenant_id, region, date,
        │         is_business_day, holiday_name,
        │         winning_source, confidence_score
        │
        └─→ INSERT INTO mdm_calendar_lineage
                tenant_id, date, winning_source,
                all_candidates, decision_reason
        ↓
  For conflicts detected:
        ├─→ Publish: conflict event
        │   (what disagreed, confidence, severity)
        │
        └─→ INSERT INTO mdm_stewardship_queue
                issue_type, description, priority
        ↓
  Publish: ingestion_completed event
        ├─→ job_id, status, records_ingested,
        │   conflicts_detected, duration
        │
        └─→ Send to Redpanda topic: ingestion-jobs
                (partitioned by tenant_id)
```

### Query Flow

```
User Query (Frontend or Reporting System)
        ↓
GET /api/v1/calendar/golden?region=US&start_date=2026-01-01&end_date=2026-12-31
        ↓
Validate X-Tenant-ID header
        ↓
SELECT calendar_date, is_business_day, holiday_name, confidence_score
FROM mdm_calendar_golden
WHERE tenant_id = $1                   ← RLS enforces tenant isolation
  AND region_code = $2
  AND calendar_date BETWEEN $3 AND $4
ORDER BY calendar_date
        ↓
Return JSON array with golden records
        ↓
Downstream System (Reporting, Analytics, ML, Compliance)
```

---

## Database Schema Layers

### Layer 1: Semantic Model
```
semantic_terms
├── CalendarDate
├── IsBusinessDay
├── RegionCode
├── HolidayName
├── SourceOfRecord
├── ConfidenceScore
└── IngestionTimestamp

business_objects
└── HolidaySchedule
    ├── term: CalendarDate
    ├── term: IsBusinessDay
    ├── term: RegionCode
    ├── term: HolidayName
    └── term: ConfidenceScore
```

### Layer 2: Source Registry (Dynamic)
```
mdm_source_registry
├── source_name (8 total)
├── source_type (free, commercial)
├── is_active (toggle on/off)
├── priority_score (determines winner)
├── confidence_base (0-100, source quality)
├── api_endpoint (URL or service)
├── health_status (healthy, degraded, down)
└── last_successful_run
```

### Layer 3: Golden Records (Multi-Tenant)
```
mdm_calendar_golden (RLS enforced)
├── tenant_id (isolation key)
├── region_code
├── calendar_date
├── is_business_day (survivorship winner)
├── holiday_name (survivorship winner)
├── winning_source (lineage)
├── confidence_score (0-100)
└── created_at
```

### Layer 4: Staging & Lineage
```
mdm_calendar_source (RLS enforced)
├── tenant_id
├── source_name
├── raw_data (JSON)
└── ingested_at

mdm_calendar_lineage (RLS enforced)
├── tenant_id
├── date
├── winning_source
├── all_candidates (JSON array)
├── decision_reason (why this source won)
└── created_at
```

### Layer 5: Operations & Conflicts
```
mdm_ingestion_jobs (RLS enforced)
├── id
├── tenant_id
├── job_type (manual, scheduled)
├── status (started, completed, failed)
├── records_ingested
├── conflicts_detected
├── started_at
├── completed_at

mdm_stewardship_queue (RLS enforced)
├── id
├── tenant_id
├── issue_type (SOURCE_DISAGREEMENT, etc.)
├── description
├── status (PENDING, RESOLVED, ESCALATED)
├── priority (CRITICAL, HIGH, MEDIUM, LOW)
└── created_at
```

---

## Rules Engine (Survivorship Algorithm)

```go
ExecuteSurvivorship(candidates []CandidateValue) → SurvivingRecord

Algorithm:
  1. Sort candidates by priority_score ASC (lower = higher priority)
  2. For ties, use confidence_score DESC (higher = better)
  3. SELECT candidates[0] as winner
  4. IF all_unique(candidates[].value) == false
     THEN detect_conflicts() → flag for stewardship
  5. confidence = (agreement_count / total_count) * 100
  6. RETURN SurvivingRecord{
       value: winner.value,
       source: winner.source,
       confidence: confidence,
       conflict_detected: conflict_detected
     }

Example (US holiday check for 2026-07-04):
  Input sources:
  • NagerDate (priority 4, confidence 90): July 4 = holiday (Independence Day)
  • OpenHolidays (priority 4, confidence 85): July 4 = holiday (Independence Day)
  • Workalendar (priority 3, confidence 92): July 4 = holiday (Independence Day)
  • HolidaysPyPI (priority 3, confidence 88): July 4 = holiday (Independence Day)

  Sorted by (priority, confidence):
  1. NagerDate (4, 90) ← WINNER
  2. OpenHolidays (4, 85)
  3. Workalendar (3, 92)
  4. HolidaysPyPI (3, 88)

  All sources agree → confidence = 100% ✓
  Winner: NagerDate, Holiday Name: Independence Day
```

---

## Event Types (Redpanda Publishing)

### 1. Calendar Update Event
```json
{
  "event_type": "calendar_update",
  "event_id": "uuid",
  "tenant_id": "uuid",
  "region_code": "US",
  "calendar_date": "2026-07-04",
  "is_business_day": false,
  "holiday_name": "Independence Day",
  "source_of_record": "NagerDate",
  "confidence_score": 100,
  "lineage": {
    "surviving_value": "July 4 is holiday",
    "all_candidates": ["NagerDate", "OpenHolidays", "Workalendar", "HolidaysPyPI"],
    "decision_reason": "All sources agree"
  },
  "timestamp": "2026-01-15T10:30:00Z"
}
```
Partition: `tenant_id`
Topic: `calendar-updates`

### 2. Conflict Event
```json
{
  "event_type": "conflict_detected",
  "event_id": "uuid",
  "tenant_id": "uuid",
  "region_code": "US",
  "calendar_date": "2026-12-24",
  "conflict_type": "SOURCE_DISAGREEMENT",
  "severity": "HIGH",
  "details": {
    "winner": "NagerDate",
    "disagreeing_sources": ["TradingHours", "OpenHolidays"],
    "winning_confidence": 65
  },
  "timestamp": "2026-01-15T10:30:00Z"
}
```
Topic: `conflicts`

### 3. Source Activation Event
```json
{
  "event_type": "source_activated",
  "event_id": "uuid",
  "source_name": "TradingHours",
  "activated_by": "user@company.com",
  "timestamp": "2026-01-15T10:30:00Z"
}
```
Topic: `source-events`

### 4. Ingestion Job Events
```json
{
  "event_type": "ingestion_started",
  "event_id": "uuid",
  "job_id": "uuid",
  "tenant_id": "uuid",
  "regions": ["US", "GB"],
  "year": 2026,
  "timestamp": "2026-01-15T10:30:00Z"
}
```

```json
{
  "event_type": "ingestion_completed",
  "event_id": "uuid",
  "job_id": "uuid",
  "tenant_id": "uuid",
  "status": "success",
  "records_ingested": 250,
  "conflicts_detected": 3,
  "duration_seconds": 45,
  "timestamp": "2026-01-15T10:30:00Z"
}
```
Topic: `ingestion-jobs`

---

## Component Interaction Matrix

| Component A | Component B | Interaction | Protocol | Data |
|-------------|-------------|-------------|----------|------|
| API Gateway | PostgreSQL | Query/Update | SQL/TCP | mdm_calendar_* tables |
| API Gateway | Redpanda | Publish | Kafka API | Event JSON |
| Semantic Engine | PostgreSQL | Query/Update | SQL/TCP | Source registry, golden records |
| Semantic Engine | Redpanda | Publish | Kafka API | Ingestion events |
| Semantic Engine | Workalendar | Fetch | HTTP REST | Holiday data JSON |
| Semantic Engine | Holidays PyPI | Fetch | HTTP REST | Holiday data JSON |
| Semantic Engine | NagerDate | Fetch | HTTP REST | Holiday data JSON |
| Semantic Engine | OpenHolidays | Fetch | HTTP REST | Holiday data JSON |
| Frontend | API Gateway | Query/Mutate | GraphQL | Calendar data |
| Frontend | Redpanda | Subscribe | GraphQL WS | Event updates |
| RLS Policy | PostgreSQL | Enforce | Row filter | tenant_id = $1 |
| Redpanda | Downstream Sys | Consume | Kafka API | Event stream |

---

## Deployment Architecture

```
┌────────────────────────────────────────────────────────────────┐
│                   Docker Host (Macbook)                        │
├────────────────────────────────────────────────────────────────┤
│  Network: usice-network (172.28.0.0/16)                       │
│                                                                │
│  ┌──────────────────┐                                         │
│  │  Redpanda        │ (Port 9092)      Event Broker           │
│  │  + Schema Reg    │ (Port 8081)      Schema Management      │
│  └──────────────────┘                                         │
│                                                                │
│  ┌──────────────────┐  ┌──────────────────┐                  │
│  │  Workalendar     │  │  Holidays PyPI   │                  │
│  │  Flask Service   │  │  Flask Service   │                  │
│  │  (Port 8000)     │  │  (Port 8001)     │                  │
│  └──────────────────┘  └──────────────────┘                  │
│                                                                │
│  ┌──────────────────────────────────────────┐                │
│  │         Semantic Engine (Go)              │                │
│  │  • Orchestrator                           │                │
│  │  • Rules Engine                           │                │
│  │  • Publisher                              │                │
│  │  (Port 9000, background)                 │                │
│  └──────────────────────────────────────────┘                │
│                                                                │
│  ┌──────────────────────────────────────────┐                │
│  │         API Gateway (Go)                  │                │
│  │  • HTTP REST endpoints                    │                │
│  │  • Tenant isolation                       │                │
│  │  • Event publishing                       │                │
│  │  (Port 8080)                             │                │
│  └──────────────────────────────────────────┘                │
│                                                                │
│  ┌──────────────────────────────────────────┐                │
│  │         Frontend (React)                  │                │
│  │  • Ops Console                            │                │
│  │  • Source Management                      │                │
│  │  • Job Monitoring                         │                │
│  │  (Port 3000)                             │                │
│  └──────────────────────────────────────────┘                │
│                                                                │
│  ┌──────────────────┐  ┌──────────────────┐                  │
│  │  Redpanda Console│  │     Adminer      │                  │
│  │  (Port 8888)     │  │  (Port 8889)     │                  │
│  │  Kafka Admin UI  │  │  DB Admin UI     │                  │
│  └──────────────────┘  └──────────────────┘                  │
│                                                                │
│  Outbound Connections:                                        │
│  ├─→ External PostgreSQL (100.84.126.19:5432)               │
│  └─→ External Services (NagerDate, OpenHolidays APIs)       │
│                                                                │
└────────────────────────────────────────────────────────────────┘
```

---

## Multi-Tenancy Design

### Isolation Levels

```
Application Layer: X-Tenant-ID header validation
                   ↓
   API Layer: X-Tenant-ID in context
                   ↓
 Database Layer: RLS policy WHERE tenant_id = session context
                   ↓
  Event Layer: Redpanda partitioning by tenant_id (order guarantee)
                   ↓
Frontend Layer: Query filters by tenant (no cross-tenant visibility)
```

### RLS Policy Example
```sql
CREATE POLICY tenant_isolation ON mdm_calendar_golden
  USING (tenant_id = current_setting('app.current_tenant'));

-- In application:
SET app.current_tenant = '00000000-0000-0000-0000-000000000001';
SELECT * FROM mdm_calendar_golden;  -- Only sees tenant's data
```

---

## Monitoring & Observability

### Key Metrics

```
Source Health:
  • Last successful run timestamp
  • Error rate (failures / total runs)
  • Average response time
  • Data freshness (hours stale)

Ingestion Performance:
  • Records ingested per minute
  • Survivorship algorithm duration
  • Conflict detection rate
  • Event publishing latency

Data Quality:
  • Confidence score distribution
  • Agreement percentage between sources
  • Conflict resolution time
  • Stewardship queue depth

Database Health:
  • Connection pool utilization
  • Query performance (p95, p99)
  • Replication lag
  • Storage used

Event Stream:
  • Messages/sec published
  • Topic partition lag
  • Consumer groups
  • Message retention
```

### Dashboard Layout (TUI or Grafana)

```
┌──────────────────────────────────────────────────────────┐
│  Usice MDM - Operations Dashboard                        │
├──────────────────────────────────────────────────────────┤
│                                                          │
│  Source Status          Ingestion Jobs        Conflicts │
│  ┌─────────────────┐   ┌──────────────────┐  ┌────────┐│
│  │ Nager: ✓ 99%   │   │ Last: 45s ago    │  │Pending:│
│  │ OpenHol: ✓ 98% │   │ Status: ✓ Success│  │3   🔴 │
│  │ Workcal: ✓ 97% │   │ Conflicts: 0     │  │        │
│  │ Holidays: ↓ 45%│   │ Records: 250     │  │        │
│  │ Trading: ⊘ 0%  │   │ Next: 01:00 UTC  │  │        │
│  └─────────────────┘   └──────────────────┘  └────────┘│
│                                                          │
│  Data Quality               Events/min                  │
│  Confidence: 98.5% 📊      Published: 1,250/min 📈    │
│  Agreement: 99.2% ✓        Latency: 2.3ms avg         │
│  Freshness: < 1hr 🕐       Topic lag: 0 ✓             │
│                                                          │
└──────────────────────────────────────────────────────────┘
```

---

## Runbook: Common Operations

### Activate a Commercial Source

1. **Via Ops Console**
   ```
   Navigate: http://localhost:3000 → MDM Calendar Management
   Find: TradingHours (is_active = false)
   Click: Activate button
   Verify: Shows green checkmark
   ```

2. **Via Database**
   ```sql
   UPDATE mdm_source_registry
   SET is_active = true
   WHERE source_name = 'TradingHours';
   ```

3. **Impact**
   - Next ingestion cycle includes TradingHours
   - Event published: source_activated
   - No downtime or redeployment needed

### Resolve a Conflict

1. **Identify**
   ```sql
   SELECT id, issue_type, description, severity
   FROM mdm_stewardship_queue
   WHERE status = 'PENDING'
   ORDER BY priority DESC;
   ```

2. **Review**
   - Check lineage table for competing candidates
   - Evaluate business context
   - Decide: accept winner or override

3. **Resolve**
   ```sql
   -- Accept current winner
   UPDATE mdm_stewardship_queue
   SET status = 'RESOLVED'
   WHERE id = $1;
   
   -- Or override (trigger new survivorship)
   INSERT INTO mdm_stewardship_queue
   SET status = 'RESOLVED', resolution = 'OVERRIDE_APPLIED';
   ```

### Emergency Disable Source

```bash
# If a source is returning bad data:
curl -X PATCH http://localhost:8080/api/v1/mdm/sources/nager-date-id/deactivate \
  -H "X-User-Role: global_ops"
# Next ingestion will skip that source
```

---

## Production Checklist Before Go-Live

- [ ] Postgres backups configured
- [ ] Redpanda persistence verified
- [ ] API rate limiting implemented
- [ ] Authentication/authorization enabled
- [ ] Monitoring/alerting configured
- [ ] Log aggregation setup
- [ ] Incident response runbook
- [ ] Disaster recovery tested
- [ ] Performance benchmarks met
- [ ] Load testing completed
- [ ] Security audit passed
- [ ] Regulatory compliance reviewed

---

**This architecture supports enterprise-grade master data management with Workday-level sophistication, while maintaining operational simplicity through dynamic source registry, event-driven streaming, and comprehensive audit trails.**

