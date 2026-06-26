# 🚀 Phase 2: Event Streaming - Quick Start

**Goal:** Add real-time event streaming with Redpanda in **4-6 hours**

---

## What You'll Build

```
Semantic Engine (Phase 1)
    ↓ (publishes every calendar update)
Redpanda Broker
    ├─→ Trading Platform (example consumer)
    ├─→ Your Custom Systems
    └─→ React Dashboard (live updates)
```

**Result:** Your entire platform knows about calendar changes instantly, no polling needed.

---

## Phase 2 Deliverables

| Component | Files | Purpose |
|-----------|-------|---------|
| **Event Schema** | `proto/calendar/events/v1/calendar_events.proto` | Protobuf schema for events (CalendarEvent, Conflict, IngestionEvent) |
| **Publisher** | `internal/publisher/redpanda.go` | Publishes events from orchestrator to Redpanda |
| **Integration** | Update to `internal/mdm/orchestrator.go` | Hooks publisher into ingestion cycle |
| **Example Consumer** | `services/trading-consumer/main.go` | Template for building your own consumers |
| **Docker Support** | `services/trading-consumer/Dockerfile` | Containerized consumer |
| **React Hook** | `frontend/src/hooks/useCalendarSubscription.ts` | Subscribe to events in React |
| **React Component** | `frontend/src/components/LiveCalendarUpdates.tsx` | Pre-built dashboard widget |
| **Docker Compose** | Update to `docker-compose.mdm.yml` | Add trading-consumer + Redpanda Console |
| **Docs** | `PHASE_2_EVENT_STREAMING.md` | Detailed 8-step guide |
| **Checklist** | `PHASE_2_VALIDATION_CHECKLIST.md` | Validation & troubleshooting |

---

## Quick Start (Copy-Paste)

### 1. Generate Protobuf Schema (5 min)

```bash
# Create directory
mkdir -p proto/calendar/events/v1

# Create proto/calendar/events/v1/calendar_events.proto
# (Copy from PHASE_2_EVENT_STREAMING.md → Step 1.1)

# Compile to Go
brew install protobuf  # If not installed
protoc --go_out=. --go_opt=paths=source_relative proto/calendar/events/v1/*.proto

# Verify
ls pkg/proto/calendar/events/v1/calendar_events.pb.go
```

### 2. Add Publisher to Backend (30 min)

```bash
# Create internal/publisher/redpanda.go
# (Copy full implementation from PHASE_2_EVENT_STREAMING.md → Step 2)

# Create internal/publisher/publisher.go if needed:
cat > internal/publisher/publisher.go << 'EOF'
package publisher

// Export public interface for use in other packages
EOF

# Test compilation
cd backend && go build ./...
```

### 3. Integrate Publisher into Orchestrator (20 min)

Edit `internal/mdm/orchestrator.go`:

```go
// Add import
import "github.com/usice/calendar-service/internal/publisher"

// Add to struct
type IngestionOrchestrator struct {
	// ... existing fields ...
	eventPublisher *publisher.CalendarEventPublisher
}

// Add to constructor
func NewIngestionOrchestrator(
	db *sql.DB,
	eventPublisher *publisher.CalendarEventPublisher,
	logger *logrus.Entry,
) *IngestionOrchestrator {
	return &IngestionOrchestrator{
		// ... existing initialization ...
		eventPublisher: eventPublisher,
	}
}

// In RunIngestionCycle method, add:
func (o *IngestionOrchestrator) RunIngestionCycle(ctx context.Context, ...) error {
	// At start of ingestion
	o.eventPublisher.PublishIngestionStarted(ctx, ingestionID, tenantID, regions, year, forceRefresh, "system")
	
	// ... existing ingestion code ...
	
	// After each calendar update
	o.eventPublisher.PublishCalendarUpdate(ctx, tenantID, region, exchange, date, ...)
	
	// At end of ingestion
	o.eventPublisher.PublishIngestionCompleted(ctx, ingestionID, tenantID, ...)
	
	return nil
}
```

### 4. Build Trading Consumer (15 min)

```bash
# Create services/trading-consumer/main.go
# (Copy from PHASE_2_EVENT_STREAMING.md → Step 4)

# Create services/trading-consumer/Dockerfile
# (Copy from PHASE_2_EVENT_STREAMING.md → Step 4)

# Create go.mod in services/trading-consumer/
cat > services/trading-consumer/go.mod << 'EOF'
module github.com/usice/calendar-service/services/trading-consumer

go 1.21

require (
	github.com/twmb/franz-go v1.14.0
	google.golang.org/protobuf v1.31.0
)

replace github.com/usice/calendar-service => ../..
EOF

# Download dependencies
cd services/trading-consumer && go mod download

# Build image
docker build -t semlayer/trading-consumer:latest services/trading-consumer
```

### 5. Update Docker Compose (10 min)

Add to `docker-compose.mdm.yml`:

```yaml
  trading-consumer:
    build:
      context: ./services/trading-consumer
      dockerfile: Dockerfile
    container_name: trading-consumer
    environment:
      - REDPANDA_BROKERS=redpanda:9092
    depends_on:
      - redpanda
    networks:
      - usice-network

  redpanda-console:
    image: docker.redpanda.com/redpandadata/console:latest
    container_name: redpanda-console
    environment:
      - KAFKA_BROKERS=redpanda:9092
    ports:
      - "8888:8080"
    depends_on:
      - redpanda
    networks:
      - usice-network
```

### 6. Add React Components (20 min)

```bash
# Create frontend/src/hooks/useCalendarSubscription.ts
# (Copy from PHASE_2_EVENT_STREAMING.md → Step 5)

# Create frontend/src/components/LiveCalendarUpdates.tsx
# (Copy from PHASE_2_EVENT_STREAMING.md → Step 5)

# Verify TypeScript compiles
cd frontend && npm run build

# In your dashboard, add:
import { LiveCalendarUpdates } from './components/LiveCalendarUpdates';

export function Dashboard() {
  return (
    <div>
      <LiveCalendarUpdates />
      {/* ... rest of dashboard ... */}
    </div>
  );
}
```

### 7. Deploy (5 min)

```bash
# Restart services with event streaming
docker-compose -f docker-compose.mdm.yml down
docker-compose -f docker-compose.mdm.yml up -d
sleep 30

# Verify all running
docker-compose -f docker-compose.mdm.yml ps
# Should show 11 services, all "Up"
```

### 8. Test (10 min)

```bash
# Trigger ingestion (publishes hundreds of events)
curl -X POST http://localhost:8080/api/v1/mdm/calendar/ingest \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -d '{"regions": ["US"], "year": 2026, "force_refresh": false}'

# Watch trading consumer receive events
docker-compose logs -f trading-consumer | head -30

# View events in Redpanda Console
open http://localhost:8888

# See real-time updates in React dashboard
# (if you added the component)
```

---

## Success Criteria ✅

All of these should be true:

- [ ] Protobuf schema compiles without errors
- [ ] Go code builds: `cd backend && go build ./...`
- [ ] Docker images built: `docker images | grep trading-consumer`
- [ ] All 11 services running: `docker-compose ps | grep Up | wc -l` = 11
- [ ] Events published: `docker-compose logs trading-consumer | grep "EVENT"`
- [ ] Redpanda Console accessible: `curl http://localhost:8888/health`
- [ ] React component builds: `cd frontend && npm run build` (no errors)
- [ ] 250+ events in calendar-updates topic
- [ ] Real-time dashboard shows live updates

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────┐
│ PHASE 2: EVENT STREAMING ARCHITECTURE                   │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  Semantic Engine (Go)                                   │
│    └─→ Internal Publisher (Go)                          │
│         └─→ Redpanda Broker (Kafka-compatible)          │
│              ├─→ calendar-updates topic                 │
│              ├─→ calendar-conflicts topic               │
│              └─→ ingestion-lifecycle topic              │
│                   │                                      │
│                   ├─→ Trading Consumer (Go)             │
│                   ├─→ Custom Consumers (Your Code)      │
│                   └─→ React Dashboard (Browser)         │
│                                                          │
│  Redpanda Console (UI)                                 │
│    └─→ View topics, messages, consumer groups          │
│                                                          │
└─────────────────────────────────────────────────────────┘
```

---

## Key Concepts

### Events Published

**1. CalendarEvent** - Every calendar day update
```protobuf
{
  date: "2026-12-25",
  is_business_day: false,
  holiday_name: "Christmas",
  region: "US",
  confidence_score: 90,  // 0-100
  source_system: "Workalendar"
}
```

**2. ConflictEvent** - When sources disagree
```protobuf
{
  date: "2026-07-04",
  field: "is_business_day",
  conflicting_values: ["true", "false"],  // Disagreement
  sources: ["Workalendar", "TradingHours"]
}
```

**3. IngestionEvent** - Lifecycle tracking
```protobuf
{
  event_type: "COMPLETED",
  records_ingested: 250,
  conflicts_resolved: 3,
  duration_ms: 5000
}
```

### Topics (Kafka-Compatible)

- **calendar-updates** - Publishing rate: ~50-100/sec during ingestion
- **calendar-conflicts** - Publishing rate: ~1-10/sec
- **ingestion-lifecycle** - Publishing rate: 2 per ingestion (START, COMPLETE)

---

## Usage Examples

### Example 1: Trading System Consumer

```go
// Watch for holidays that will affect trading
for event := range eventChannel {
  if !event.IsBusinessDay {
    fmt.Printf("Holiday: %s (%s)\n", event.HolidayName, event.Region)
    // Notify trading desk, reschedule orders, etc
  }
}
```

### Example 2: Analytics Pipeline

```go
// Stream calendar events to data warehouse
consumer := NewConsumer("analytics-pipeline")
consumer.On("calendar-updates", func(event *CalendarEvent) {
  // Send to BigQuery, Snowflake, etc
  warehouse.Insert(event)
})
```

### Example 3: React Dashboard

```tsx
export function Dashboard() {
  const { calendarEvents, lastUpdate, updateRate } = useCalendarSubscription();
  
  return (
    <div>
      <h2>Live Calendar Updates: {updateRate} events/sec</h2>
      <p>Latest: {lastUpdate?.calendarDate} - {lastUpdate?.holidayName}</p>
      <EventHistory events={calendarEvents} />
    </div>
  );
}
```

---

## Files to Copy & Customize

All files are production-ready but may need adjustment for your environment:

| File | Status | Customization |
|------|--------|---------------|
| `proto/calendar/events/v1/calendar_events.proto` | ✅ Ready | Add your event types as needed |
| `internal/publisher/redpanda.go` | ✅ Ready | Adjust batch sizes/timing for load |
| `services/trading-consumer/main.go` | 📋 Template | Replace cache logic with your system |
| `frontend/src/hooks/useCalendarSubscription.ts` | ✅ Ready | Test Apollo subscription config |
| `frontend/src/components/LiveCalendarUpdates.tsx` | 📋 Template | Customize styling/layout |

---

## Troubleshooting Quick Links

| Problem | Solution |
|---------|----------|
| Protobuf won't compile | See PHASE_2_EVENT_STREAMING.md → Troubleshooting #Protobuf |
| Events not publishing | See PHASE_2_EVENT_STREAMING.md → Troubleshooting #No events |
| Consumer not receiving | See PHASE_2_EVENT_STREAMING.md → Troubleshooting #No events |
| React component errors | See PHASE_2_VALIDATION_CHECKLIST.md → Section "React Component" |

---

## Timeline

| Step | Time | Status |
|------|------|--------|
| 1. Protobuf schema | 5 min | Copy → Compile |
| 2. Publisher impl | 30 min | Copy → Build |
| 3. Orchestrator integration | 20 min | Edit → Test |
| 4. Trading consumer | 15 min | Copy → Build |
| 5. Docker Compose | 10 min | Edit |
| 6. React components | 20 min | Copy → Build |
| 7. Deploy | 5 min | restart services |
| 8. Test | 10 min | Verify all 8 criteria |
| **Total** | **2.5 hours** | **Ready for production** |

---

## Next: Phase 3 (After Phase 2 Complete)

Phase 3 adds:
- Commercial data sources (TradingHours, EODHD, Xignite)
- Failover logic
- Health monitoring
- Production hardening

Estimated: 4-6 hours | Timeline: Can start immediately after Phase 2 validates

---

**Ready to start?** → Open [PHASE_2_EVENT_STREAMING.md](PHASE_2_EVENT_STREAMING.md) and follow Step 1

**Need detailed guidance?** → Check [PHASE_2_VALIDATION_CHECKLIST.md](PHASE_2_VALIDATION_CHECKLIST.md)

**Questions?** → Each guide has troubleshooting sections

---

**Phase 2 Status: 🚀 Ready to Deploy**
