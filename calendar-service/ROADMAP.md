# Epic 31: Holiday & Calendar Intelligence - Implementation Roadmap

## 🎯 Goal
Build a **production-ready, AI-augmented calendar management system** that automatically handles holidays, blackouts, and job scheduling across regions and time zones.

## 📋 Phases & Milestones

### Phase 0: Foundation & Local Development (✅ COMPLETED)
**Status**: All files created and ready for development
**Deliverables**:
- [x] Directory structure
- [x] Go module setup  
- [x] Config system
- [x] Local Docker Compose stack
- [x] Hasura client integration
- [x] Stub API handlers
- [x] Makefile for development
- [x] Quick start guide

---

### Phase 1: Core Calendar Management (⏳ NEXT - 1 Week)

**Goal**: Implement basic CRUD operations for calendars with bitemporal versioning

#### 1.1 Database Schema (Day 1)
**Tasks**:
- [ ] Create migration: `calendars` table
  ```sql
  CREATE TABLE calendars (
      id UUID PRIMARY KEY,
      tenant_id UUID NOT NULL,
      name VARCHAR(255) NOT NULL,
      region VARCHAR(100),
      holidays JSONB,
      valid_from TIMESTAMPTZ,
      valid_to TIMESTAMPTZ,
      ...
  );
  ```
- [ ] Create migration: `schedule_profiles` table
- [ ] Create migration: `blackouts` table
- [ ] Create audit tables
- [ ] Create Debezium connector config

**Files to Update**:
- `docs/schema.sql` - centralized schema
- `migrations/` - numbered migration files

#### 1.2 Hasura GraphQL Layer (Day 1-2)
**Tasks**:
- [ ] Track all tables in Hasura
- [ ] Configure Row-Level Security (RLS) permissions
  - All queries filtered by `tenant_id`
  - Soft delete (use `valid_to` for filtering)
- [ ] Configure Hasura Actions for complex mutations
- [ ] Test GraphQL queries via Hasura Console

#### 1.3 Golang Service Implementation (Day 2-3)
**Files to Create**:
- `internal/services/calendar_service.go` - business logic
  - `Create()` - insert new calendar
  - `Update()` - bitemporal versioning (close old, insert new)
  - `Get()` - fetch active version
  - `List()` - list all for tenant
  - `Delete()` - soft delete
  
- `internal/api/calendar_handlers.go` - HTTP handlers (replace stubs)
  - Parse requests
  - Validate tenant isolation
  - Call services
  - Return JSON responses

**Implementation Details**:
```go
// Example: Create a calendar
func (s *CalendarService) Create(ctx context.Context, tenantID, name, region string, holidays json.RawMessage) (*Calendar, error) {
    // 1. Validate inputs
    // 2. Insert via Hasura GraphQL
    // 3. Log audit entry
    // 4. Publish event to Redpanda (optional, for real-time sync)
    // 5. Return ID
}

// Example: Update (bitemporal)
func (s *CalendarService) Update(ctx context.Context, id string, updates map[string]interface{}) (*Calendar, error) {
    // 1. Fetch current active version
    // 2. Close it (set valid_to = now)
    // 3. Insert new version with updates
    // 4. Audit entry
    // 5. Return new version
}
```

#### 1.4 React UI (Day 3-5)
**Components to Create**:
- `src/pages/Calendars.tsx` - main page
- `src/components/CalendarList.tsx` - table of calendars
- `src/components/CalendarForm.tsx` - create/edit form
- `src/components/ProcessingModal.tsx` - for bulk operations

**Features**:
- List calendars for tenant
- Create calendar (modal form)
- Edit calendar (inline or modal)
- Delete calendar (soft delete)
- Display holidays as JSON
- Show valid_from / valid_to for versioning

**New Apollo Queries/Mutations**:
```graphql
query ListCalendars($tenantId: uuid!) {
  calendars(where: {tenant_id: {_eq: $tenantId}, valid_to: {_is_null: true}}) {
    id name region valid_from
  }
}

mutation CreateCalendar($object: calendars_insert_input!) {
  insert_calendars_one(object: $object) { id }
}

mutation UpdateCalendar($id: uuid!, $validTo: timestamptz!) {
  update_calendars_by_pk(pk_columns: {id: $id}, _set: {valid_to: $validTo}) { id }
}
```

#### 1.5 Testing (Day 5)
- [ ] Unit tests for `calendar_service.go`
- [ ] Integration tests: create → update → list
- [ ] API tests with curl/Postman
- [ ] UI tests with Cypress

**Test Cases**:
- Create calendar with valid inputs → success
- Create calendar with invalid region → error
- Update calendar → verify bitemporal versioning
- List calendars → only return active versions
- Delete calendar → set valid_to

#### 1.6 Deployment to Local Dev (Day 5)
- [ ] Build Docker image: `make build`
- [ ] Start services: `make dev`
- [ ] Test endpoints: `curl http://localhost:8081/api/v1/calendars`

**Definition of Done**:
- ✅ All CRUD operations working
- ✅ Tests passing
- ✅ React UI functional
- ✅ Bitemporal versioning verified
- ✅ Audit entries created

---

### Phase 2: Availability Checker & Scheduler Integration (1 Week)

**Goal**: Check if a job can run at a given time, respecting holidays/blackouts

#### 2.1 Core Logic (Day 1-2)
**File**: `internal/availability/checker.go`

```go
func (c *Checker) CheckAvailability(ctx context.Context, tenantID, profileName string, start, end time.Time) (bool, []string, error) {
    // 1. Resolve profile (fetch schedule profile + calendars)
    // 2. Merge holidays/blackouts (union/priority)
    // 3. Check if time range conflicts
    // 4. Return available + reasons
}

func (c *Checker) ResolveProfile(ctx context.Context, tenantID, profileName string) (*ResolvedCalendar, error) {
    // 1. Fetch profile from Hasura
    // 2. For each linked calendar, fetch active holidays
    // 3. Merge per conflict_resolution rules
    // 4. Cache result in Redis for 1 hour
}
```

#### 2.2 REST Endpoint (Day 2)
```go
POST /api/v1/check-availability
{
  "tenant_id": "...",
  "profile_name": "default",
  "start_time": "2026-03-01T10:00:00Z",
  "end_time": "2026-03-01T11:00:00Z"
}

Response:
{
  "available": false,
  "reasons": ["Holiday: 2026-03-01"]
}
```

#### 2.3 React UI Component (Day 2-3)
`src/components/AvailabilityTester.tsx` - Simple form + results display

#### 2.4 Integration (Day 3-5)
- [ ] Hook into scheduler (before job execution)
- [ ] If unavailable → reschedule (next available slot)
- [ ] Log reschedule reason
- [ ] Verify via Redpanda → audit trail

**Definition of Done**:
- ✅ Availability checker working
- ✅ Caching working (Redis)
- ✅ REST endpoint tested
- ✅ React component functional

---

### Phase 3: Event-Driven Propagation (Redpanda + Temporal) (1 Week)

**Goal**: When a calendar changes, automatically reschedule affected jobs

#### 3.1 Redpanda Consumer (Day 1-2)
**File**: `internal/redpanda/consumer.go`

```go
func (p *CDCProcessor) Run(ctx context.Context) error {
    // 1. Connect to Redpanda
    // 2. Subscribe to: postgres.public.calendars, ...
    // 3. For each change:
    //    - Extract tenantID
    //    - Signal Temporal workflow: CalendarChangedWorkflow
    // 4. Use exactly-once semantics (transactions)
}
```

#### 3.2 Temporal Workflow (Day 2-3)
**File**: `internal/temporal/workflows/calendar_changed.go`

```go
func CalendarChangedWorkflow(ctx workflow.Context, tenantID string) error {
    // 1. Listen for CalendarChangedSignal
    // 2. Fetch all jobs for tenant
    // 3. For each job:
    //    - Check if next run is still available
    //    - If blocked → reschedule
    // 4. Update job next_run in DB
    // 5. Log reschedule
}
```

#### 3.3 Activities (Day 3-4)
- `FetchAffectedJobsActivity` - list jobs for tenant
- `ResolveCalendarActivity` - blend profiles
- `RescheduleJobActivity` - update DB
- `PublishEventActivity` - notify via Redpanda

#### 3.4 Testing (Day 4-5)
- [ ] Insert calendar change → observe reschedule
- [ ] Verify Redpanda consumer lag
- [ ] Check Temporal workflow execution

**Definition of Done**:
- ✅ Redpanda consumer processing events
- ✅ Temporal workflows executing
- ✅ Jobs rescheduled automatically
- ✅ No data loss (exactly-once semantics)

---

### Phase 4: Multi-Timezone & International Support (1 Week)

**Goal**: Handle multiple time zones, regions, and schedule profiles

#### 4.1 Data Model (Day 1)
- [ ] Add `timezone` to `schedule_profiles`
- [ ] Add `conflict_resolution` rules (union, intersection, priority)
- [ ] Update schema

#### 4.2 Calendar Resolution Logic (Day 1-2)
Update `ResolveProfile` to:
- [ ] Handle union (all holidays block)
- [ ] Handle priority (highest priority calendar wins)
- [ ] Convert times to profile timezone for comparison

#### 4.3 React UI (Day 2-3)
- `ProfileManager.tsx` - CRUD for profiles
- Multi-select calendars
- Conflict resolution picker
- Timezone selector

#### 4.4 Testing (Day 3-5)
- [ ] Multi-profile resolution working
- [ ] Timezone conversions correct
- [ ] Integration tests

**Definition of Done**:
- ✅ Multiple profiles per tenant
- ✅ Time zone handling correct
- ✅ Conflict resolution rules enforced

---

### Phase 5: External Calendars & AI (2 Weeks)

**Goal**: Sync Google/Outlook calendars, auto-generate holidays, predict blackouts

#### 5.1 External Connection Management (Days 1-4)
- [ ] Database table: `external_calendar_connections`
- [ ] Golang service: OAuth flow, token encryption
- [ ] Temporal workflow: sync job (pull events)
- [ ] React UI: connection list, OAuth popups, manual sync

#### 5.2 AI Enhancements (Days 5-8)
- [ ] Service: `internal/ai/client.go`
- [ ] Activities for: generate holidays, predict blackouts, suggest slots
- [ ] React UI: modal for AI generation + approval flow
- [ ] Audit trail for AI-generated changes

#### 5.3 Predictive Rescheduling (Days 9-10)
- [ ] Temporal workflow: analyze job history
- [ ] ML/heuristics: identify patterns
- [ ] Suggest optimal schedules
- [ ] React: recommendations page

**Definition of Done**:
- ✅ External calendars synced
- ✅ AI generating suggestions
- ✅ Predictive logic working

---

### Phase 6: Analytics & Observability (1 Week)

**Goal**: Dashboards, monitoring, and compliance

#### 6.1 Analytics API (Days 1-3)
- [ ] Endpoints for: conflicts, calendar impact, job sensitivity, trends
- [ ] Materialized views in Postgres
- [ ] Nightly aggregation workflow

#### 6.2 React Dashboard (Days 3-5)
- [ ] Conflict heatmap
- [ ] Calendar impact chart
- [ ] Export to CSV/PDF

#### 6.3 Monitoring (Days 5-7)
- [ ] Prometheus metrics
- [ ] Grafana dashboards
- [ ] Alerts (CDC lag, workflow failures, etc.)

**Definition of Done**:
- ✅ Dashboard functional
- ✅ Monitoring in place
- ✅ Alerts configured

---

## 🚀 Quick Start Commands

### Local Development

```bash
# 1. Update config
cd calendar-service
nano .env.local  # Set HASURA_ENDPOINT and credentials

# 2. Start services
make dev

# 3. Check health
curl http://localhost:8081/health

# 4. Start frontend
cd ../frontend
npm run dev

# 5. Run tests
cd ../calendar-service
go test -v ./...
```

### Deploy to Remote

```bash
# 1. Build Docker image
docker build -t calendar-service:latest .

# 2. Push to registry (e.g., Docker Hub)
docker tag calendar-service:latest yourusername/calendar-service:latest
docker push yourusername/calendar-service:latest

# 3. Deploy to production cluster (K8s, ECS, etc.)
kubectl set image deployment/calendar-service calendar-service=yourusername/calendar-service:latest
```

---

## ✅ Success Criteria

By end of Phase 6, you will have:

1. **Fully functional calendar management system** - create, update, delete calendars
2. **Availability checking** - know if a job can run at a given time
3. **Automatic rescheduling** - when calendars change, jobs reschedule
4. **Multi-timezone support** - handle global deployments
5. **External calendar sync** - pull from Google/Outlook
6. **AI suggestions** - generate holidays, predict blackouts
7. **Full observability** - dashboards, metrics, alerts
8. **Production-ready** -tests, security, documentation

---

## 📊 Effort Estimate

| Phase | Duration | Effort | Team Size |
|-------|----------|--------|-----------|
| 0     | 1 day    | 4h     | 1(-2 people |
| 1     | 1 week   | 40h    | 2-3 people |
| 2     | 1 week   | 40h    | 2-3 people |
| 3     | 1 week   | 40h    | 2-3 people |
| 4     | 1 week   | 40h    | 2-3 people |
| 5     | 2 weeks  | 80h    | 3-4 people |
| 6     | 1 week   | 40h    | 1-2 people |
| **Total** | **~8 weeks** | **~280h** | **2-3 people** |

---

## 📚 Documentation to Write

- [ ] API Specification (OpenAPI/Swagger)
- [ ] Architecture Diagrams (Mermaid)
- [ ] Database Schema Docs
- [ ] Deployment Guide
- [ ] Operations Runbook
- [ ] User Guide (admin features)

---

## 🔗 Next Immediate Actions

1. **Update `.env.local`** with your remote credentials
2. **Run `make dev`** to start local services
3. **Test connectivity** to remote Postgres/Hasura
4. **Begin Phase 1** - Calendar CRUD implementation

Good luck! 🚀
