# Phase 2 Implementation Checklist & Validation

**Target:** 4-6 hours | **Start:** After Phase 1 validation passes

---

## Pre-Implementation (30 min)

- [ ] Read entire [PHASE_2_EVENT_STREAMING.md](PHASE_2_EVENT_STREAMING.md) guide
- [ ] Verify Phase 1 containers running: `docker-compose -f docker-compose.mdm.yml ps`
- [ ] Confirm Redpanda broker accessible: `docker-compose -f docker-compose.mdm.yml logs redpanda | grep "API"`
- [ ] Have `protoc` installed: `protoc --version` (install: `brew install protobuf`)
- [ ] Have Go 1.21+: `go version`

---

## Implementation Steps (Track Progress)

### Step 1: Event Schema (30 min)

- [ ] Create proto directory structure
  ```bash
  mkdir -p proto/calendar/events/v1
  ```

- [ ] Copy `proto/calendar/events/v1/calendar_events.proto` from guide
  
- [ ] Generate Go code
  ```bash
  protoc --go_out=. --go_opt=paths=source_relative proto/calendar/events/v1/*.proto
  ```

- [ ] Verify generated files
  ```bash
  ls -la pkg/proto/calendar/events/v1/
  # Should show: calendar_events.pb.go
  ```

### Step 2: Publisher Implementation (2 hours)

- [ ] Create `internal/publisher/redpanda.go` with full implementation
  
- [ ] Update `internal/mdm/orchestrator.go`:
  - [ ] Add event publisher to struct
  - [ ] Initialize in constructor
  - [ ] Publish events on calendar updates
  - [ ] Publish ingestion lifecycle events

- [ ] Verify Go code compiles
  ```bash
  cd backend && go build ./...
  ```

- [ ] Run tests to ensure no regressions
  ```bash
  cd backend && go test ./internal/mdm/...
  ```

### Step 3: Trading Consumer (1 hour)

- [ ] Create `services/trading-consumer/main.go`
  
- [ ] Create `services/trading-consumer/Dockerfile`
  
- [ ] Build Docker image
  ```bash
  docker build -t semlayer/trading-consumer:latest services/trading-consumer
  ```

- [ ] Verify image built
  ```bash
  docker images | grep trading-consumer
  ```

### Step 4: Docker Compose Update (15 min)

- [ ] Update `docker-compose.mdm.yml` to add:
  - [ ] `trading-consumer` service
  - [ ] `redpanda-console` service (UI)
  
- [ ] Verify compose file is valid
  ```bash
  docker-compose -f docker-compose.mdm.yml config > /dev/null && echo "✅ Config valid" || echo "❌ Config invalid"
  ```

### Step 5: React Components (1 hour)

- [ ] Copy `frontend/src/hooks/useCalendarSubscription.ts`
  
- [ ] Copy `frontend/src/components/LiveCalendarUpdates.tsx`
  
- [ ] Verify TypeScript compiles
  ```bash
  cd frontend && npm run build
  ```

- [ ] Integrate into existing dashboard:
  ```tsx
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

### Step 6: Deployment (15 min)

- [ ] Stop existing services
  ```bash
  docker-compose -f docker-compose.mdm.yml down
  ```

- [ ] Start new services with event streaming
  ```bash
  docker-compose -f docker-compose.mdm.yml up -d
  sleep 30
  ```

- [ ] Verify all services healthy
  ```bash
  docker-compose -f docker-compose.mdm.yml ps
  # Should show: 11 services, all "Up"
  ```

---

## Validation (20 min)

### Test 1: Redpanda Topics

```bash
# Verify topics exist
docker-compose -f docker-compose.mdm.yml exec redpanda rpk topic list

# Expected output:
# calendar-updates
# calendar-conflicts
# ingestion-lifecycle
```

✅ Pass / ❌ Fail

### Test 2: Event Publishing

```bash
# Trigger ingestion to publish events
curl -X POST http://localhost:8080/api/v1/mdm/calendar/ingest \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -d '{
    "regions": ["US"],
    "year": 2026,
    "force_refresh": false
  }' | jq .

# Expected: {"job_id": "...", "status": "QUEUED"}
```

✅ Pass / ❌ Fail

### Test 3: Trading Consumer Receiving Events

```bash
# Watch consumer logs for 20 seconds
timeout 20 docker-compose -f docker-compose.mdm.yml logs -f trading-consumer | head -30

# Expected output:
# [CACHE] 2026-01-02 | 💼 Business Day | US | Confidence: 85% | Source: Workalendar
# [CACHE] 2026-12-25 | 🎉 Holiday | US | Confidence: 90% | Source: NagerDate
```

✅ Pass / ❌ Fail

### Test 4: Redpanda Console

```bash
# Check console is accessible
curl http://localhost:8888/health

# Expected: HTTP 200 with JSON response
```

Then open in browser: `http://localhost:8888`
- [ ] Navigate to Topics
- [ ] Click `calendar-updates`
- [ ] See messages arriving in real-time
- [ ] Click a message to view Protobuf content

✅ Pass / ❌ Fail

### Test 5: Event Count Validation

```bash
# After ingestion completes, check event counts
docker-compose exec redpanda rpk topic describe calendar-updates

# Expected: 250+ messages in topic
```

✅ Pass / ❌ Fail

### Test 6: Conflict Detection

```bash
# Manually trigger a conflict for testing
psql -h 100.84.126.19 -U usice_app -d alpha -c "
INSERT INTO edm.mdm_calendar_source 
(source_name, calendar_date, region, is_business_day, holiday_name, ingestion_date, confidence) 
VALUES 
('FakeSource', '2026-07-04', 'US', true, 'Independence Day', NOW(), 50)
ON CONFLICT DO NOTHING;
"

# Wait for ingestion cycle, check for conflict event
docker-compose logs trading-consumer | grep "CONFLICT"
```

✅ Pass / ❌ Fail

### Test 7: React Component Integration

```bash
# Verify component builds without errors
cd frontend && npm run build 2>&1 | grep -i "error"

# Expected: No output (no errors)
```

Then test in browser:
- [ ] Import `LiveCalendarUpdates` component
- [ ] Add to your dashboard
- [ ] Verify it connects (green dot appears)
- [ ] Manually add a calendar event
- [ ] See it appear in real-time in component

✅ Pass / ❌ Fail

---

## Phase 2 Success Criteria ✅

All of the following must be true:

| Criteria | Status | Notes |
|----------|--------|-------|
| Protobuf schema compiles | [ ] ✅ | No compile errors |
| Go publisher builds | [ ] ✅ | `go build ./...` succeeds |
| Docker images built | [ ] ✅ | Trading-consumer image exists |
| All 11 services running | [ ] ✅ | `docker-compose ps` shows 11 "Up" |
| Events published | [ ] ✅ | 250+ events in `calendar-updates` topic |
| Trading consumer receives | [ ] ✅ | Console shows `[CACHE]` lines |
| Redpanda Console accessible | [ ] ✅ | `http://localhost:8888` loads |
| React component builds | [ ] ✅ | No TypeScript errors |
| Real-time updates visible | [ ] ✅ | Component shows live events |
| Conflict events published | [ ] ✅ | `calendar-conflicts` topic has messages |
| Ingestion lifecycle tracked | [ ] ✅ | STARTED and COMPLETED events captured |
| No error logs | [ ] ✅ | No FATAL or ERROR in docker logs |

**Phase 2 Complete When:** All criteria are ✅

---

## Troubleshooting

### Problem: "Failed to connect to Redpanda"

```bash
# Check Redpanda is running
docker-compose logs redpanda | tail -20

# Check network connectivity
docker-compose exec trading-consumer nc -zv redpanda 9092

# If connection refused, restart Redpanda
docker-compose -f docker-compose.mdm.yml restart redpanda
sleep 10
# Retry
```

### Problem: "No events in topic"

```bash
# Verify ingestion was triggered
docker-compose logs semantic-engine | grep -i "ingest"

# Check for publishing errors
docker-compose logs semantic-engine | grep -i "publish"

# Manually test publisher (if available)
go test -v ./internal/publisher/... -run TestPublish
```

### Problem: "Protobuf compilation fails"

```bash
# Verify protoc installed
which protoc

# If not found, install
brew install protobuf
protoc --version

# Try compilation again
protoc --go_out=. --go_opt=paths=source_relative proto/calendar/events/v1/*.proto
```

### Problem: "Trading consumer exits immediately"

```bash
# Check logs for errors
docker-compose logs trading-consumer | tail -50

# Common issue: Missing go.sum file
cd services/trading-consumer
go mod download
go mod tidy
```

### Problem: "React component not rendering"

```bash
# Verify Apollo Client subscriptions enabled
# In provider setup:
wsLink = createClient({
  url: `wss://localhost:4000/graphql`, // WebSocket URL
});

# Check browser console for GraphQL errors
# In DevTools: Network → WS → See subscription messages
```

### Problem: "Topics not created automatically"

```bash
# Manually create topics
docker-compose exec redpanda rpk topic create calendar-updates --partitions 3 --replicas 1
docker-compose exec redpanda rpk topic create calendar-conflicts --partitions 3 --replicas 1
docker-compose exec redpanda rpk topic create ingestion-lifecycle --partitions 1 --replicas 1
```

---

## Performance Optimization (Optional)

After Phase 2 validates, consider:

```yaml
# In docker-compose.mdm.yml, for high-volume scenarios:

redpanda:
  environment:
    REDPANDA_BROKERS_NUM: 3  # Add more brokers
    REDPANDA_AUTO_CREATE_TOPICS: 'true'
    REDPANDA_LOG_LEVEL: 'info'  # Reduce to 'warn' for production

trading-consumer:
  environment:
    # Consumer batching
    - BATCH_SIZE=100
    - BATCH_TIMEOUT_MS=500
    # Connection pooling
    - MAX_CONCURRENT_REQUESTS=10
    # Performance
    - CPU_LIMIT=2
    - MEMORY_LIMIT=1G
```

---

## Next Steps After Phase 2

Once all criteria pass:

1. **Document Integration:**
   - [ ] Create PHASE_2_COMPLETE.md with results
   - [ ] Record any custom modifications made

2. **Team Handoff:**
   - [ ] Share this checklist with team
   - [ ] Demonstrate real-time dashboard to stakeholders
   - [ ] Collect feedback on event types needed

3. **Proceed to Phase 3:**
   ```bash
   cat COMPLETE_MDM_ROADMAP.md | grep -A 50 "PHASE 3"
   ```

4. **Monitor Production:**
   - [ ] Set up alerts for failed ingestions
   - [ ] Monitor event latency (target: <100ms)
   - [ ] Track data conflict rates

---

## Success Celebration 🎉

**Phase 2 Complete!** You now have:

- ✅ Real-time event streaming with Redpanda
- ✅ Protobuf-serialized events for efficient transfer
- ✅ Example consumer (trading platform) showing integration pattern
- ✅ React component for live dashboard updates
- ✅ Full lifecycle tracking (ingestion events)
- ✅ Conflict detection and escalation

**Next:** Phase 3 (Production Hardening - commercial sources, failover, monitoring)
