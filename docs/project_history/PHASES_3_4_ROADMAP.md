# Phases 3 & 4 Roadmap: Advanced Microservices Architecture

**Current Status:** Phase 2 Complete ✅ | Phase 3-4 Ready for Implementation

## Phase 3: Extract BO Microservice (Planned)

### Objective
Separate the Business Object command handler into a standalone microservice container, enabling independent scaling and deployment.

### Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│ API Gateway Service (Current - Port 8080)                        │
├─────────────────────────────────────────────────────────────────┤
│ - HTTP Request Handlers (read-only unchanged)                    │
│ - BusinessObjectHandler (API Gateway - publishes commands)       │
│ - InstanceHandler (API Gateway - publishes commands)             │
│ - CommandPublisher (sends to Redpanda (Kafka))                          │
│ - temp queue listeners (waitForCommandResponse)                  │
└─────────────────────────────────────────────────────────────────┘
              ↓ Command Bus ↓
        ┌─────────────────────────────┐
        │   Redpanda (Kafka)          │
        │ semlayer.commands (topic)   │
        │ semlayer.replies (consumer) │
        │ semlayer.events (durable)   │
        └─────────────────────────────┘
              ↓ Consumes ↓
┌─────────────────────────────────────────────────────────────────┐
│ BO Command Microservice (New - Port 8081)                        │
├─────────────────────────────────────────────────────────────────┤
│ - CommandConsumer                                               │
│ - BOCommandHandler (handles BO CRUD)                            │
│ - InstanceCommandHandler (handles Instance CRUD)                │
│ - BusinessObjectService (business logic)                        │
│ - EventPublisher (publishes to event store)                     │
│ - PostgreSQL connection (same database)                         │
└─────────────────────────────────────────────────────────────────┘
```

### Implementation Steps

1. **Extract Command Handlers**
   - Move `command_bus.go` CommandConsumer to microservice
   - Move `bo_command_handler.go` to microservice
   - Move `instance_command_handler.go` to microservice
   - Move `services/businessobject_service.go` to microservice

2. **Create Dockerfile**
   ```dockerfile
   FROM golang:1.21-alpine
   WORKDIR /app
   COPY . .
   RUN go build -o bo-service ./cmd/bo-service
   EXPOSE 8081
   CMD ["./bo-service"]
   ```

3. **Create docker-compose Overrides**
   - Add `bo-service` container to docker-compose.yml
   - Environment variables:
- `KAFKA_BROKERS=localhost:9092`
     - `DATABASE_URL=postgres://...`
     - `LOG_LEVEL=info`

4. **Health Checks**
   - Add `/health` endpoint to BO service
   - Add liveness/readiness probes to kubernetes/docker-compose

5. **Configuration**
   - Move command handler registration to separate init file
   - Add graceful shutdown handling
   - Add metrics/observability

### Benefits
- ✅ Independent scaling (scale only command handler if needed)
- ✅ Independent deployment (deploy BO service independently)
- ✅ Independent monitoring (separate logs/metrics)
- ✅ Fault isolation (BO service crash doesn't affect API Gateway)
- ✅ Team autonomy (separate team can own BO service)

### Timeline: 2-3 hours (straightforward extraction)

---

## Phase 4a: CQRS Pattern (Planned)

### Objective
Implement Command Query Responsibility Segregation - separate read and write models for optimized performance and scalability.

### Architecture

```
Read Path (Current - No Changes)
   HTTP GET /api/business-objects     → ListBusinessObjects
   HTTP GET /api/business-objects/{key} → GetBusinessObject
   Direct Service Call (fast, no message bus)

Write Path (New - Command Bus)
   HTTP POST /api/business-objects    → PublishCommand → Handler → Event
   HTTP PUT /api/business-objects/{key} → PublishCommand → Handler → Event
   HTTP DELETE /api/business-objects/{key} → PublishCommand → Handler → Event

Event Store (New)
   Events published to semlayer.events (durable)
   Persisted in event_store table
   With correlation IDs for tracing

Read Model Projection (New)
   Event Consumer subscribes to all events
   Updates denormalized read models
   Optimized for query performance
   Could be in separate database/cache
```

### Implementation Steps

1. **Extend Event Store**
   - Create `events` table (if not exists)
   - Store all events with:
     - ID, Type, AggregateID, CorrelationID
     - Timestamp, UserID, TenantID
     - Payload (JSON), Metadata

2. **Create Event Projections**
   ```go
   // Read model projections
   type BusinessObjectProjection struct {
       ID              string
       Key             string
       TenantID        string
       CreatedAt       time.Time
       UpdatedAt       time.Time
       DeletedAt       *time.Time
       // Denormalized fields for fast queries
   }
   ```

3. **Create Projection Updater**
   - Subscribe to events
   - Update read models in real-time
   - Handle eventual consistency

4. **Add Read-Model Queries**
   - Query from projections instead of main BO table
   - Add caching layer (Redis optional)
   - Add search/filtering on projections

### Benefits
- ✅ Write optimization (no complex queries during writes)
- ✅ Read optimization (pre-aggregated, denormalized data)
- ✅ Scalability (can scale read and write independently)
- ✅ Performance (queries against optimized read models)
- ✅ Analytics (event history preserved for analysis)

### Timeline: 3-4 hours

---

## Phase 4b: Saga Pattern (Planned)

### Objective
Implement distributed transaction management across multiple aggregates using orchestrated sagas.

### Architecture

```
Saga Orchestrator (New Service)
   Coordinates multi-step workflows
   Handles compensating transactions
   Monitors correlation IDs

Example Workflow: Create Customer with related objects
┌─────────────────────────────────────┐
│ User: CreateCustomerRequest         │
└──────────────┬──────────────────────┘
               ↓
┌─────────────────────────────────────┐
│ Step 1: Command.CreateCustomer      │
│ Success → Step 2                    │
│ Fail → Compensation (none needed)   │
└──────────────┬──────────────────────┘
               ↓
┌─────────────────────────────────────┐
│ Step 2: Command.CreateAccount       │
│ Success → Step 3                    │
│ Fail → Compensation (Undo Customer) │
└──────────────┬──────────────────────┘
               ↓
┌─────────────────────────────────────┐
│ Step 3: Command.SendWelcomeEmail    │
│ Success → Complete                  │
│ Fail → Compensation (Undo Account)  │
└──────────────┬──────────────────────┘
               ↓
        ✅ Success or
        ❌ Rollback (all compensation steps)
```

### Implementation Steps

1. **Define Saga State Machine**
   ```go
   type Saga struct {
       ID            string
       CorrelationID string
       Status        string // pending, completed, failed, compensating
       Steps         []SagaStep
       CompensatingSteps []SagaStep
   }
   
   type SagaStep struct {
       Name       string
       Command    *Command
       Status     string // pending, completed, failed
       Result     interface{}
       CompensationCommand *Command
   }
   ```

2. **Create Saga Orchestrator**
   - Maintains saga state in database
   - Publishes commands
   - Listens for events
   - Executes compensation on failures

3. **Define Compensation Logic**
   - Each command must have a compensation command
   - Store compensation commands with saga state
   - Execute in reverse order on failure

4. **Error Handling**
   - Retry logic with exponential backoff
   - Dead letter queue for unrecoverable failures
   - Monitoring and alerting

### Example Use Cases
- Creating customer with multiple related objects (account, address, preferences)
- Complex batch operations requiring transactional consistency
- Multi-service workflows with dependencies

### Timeline: 4-5 hours

---

## Phase 4c: Event Replay & Snapshots (Planned)

### Objective
Implement event sourcing capability to reconstruct aggregate state at any point in time, with snapshots for performance.

### Architecture

```
Event Store
   ├─ Event 1 (Create Customer)
   ├─ Event 2 (Update Email)
   ├─ Event 3 (Add Address)
   └─ Event N (...)

Snapshot Store (Periodic)
   └─ Snapshot at Event 50 (aggregated state)

Reconstruction
   Load Snapshot 50 → Apply Events 51-N → Current State
   (Much faster than replaying from Event 1)
```

### Implementation Steps

1. **Event Replay Infrastructure**
   ```go
   type EventReplayer interface {
       ReplayToTimestamp(aggregateID string, timestamp time.Time) (*Aggregate, error)
       ReplayToEventNumber(aggregateID string, eventNumber int) (*Aggregate, error)
       ReplayAll(aggregateID string) (*Aggregate, error)
   }
   ```

2. **Snapshot Strategy**
   - Create snapshot every N events
   - Store in snapshot_store table
   - Include event number and timestamp
   - Use for aggregate reconstruction

3. **Aggregate Rebuilding**
   - Load latest snapshot
   - Replay events since snapshot
   - Return current state
   - Optional: verify consistency

4. **Use Cases for Replay**
   - Debugging: "Show me what happened to this object"
   - Auditing: "Complete history of changes"
   - Analysis: "How did we get to this state?"
   - Time-travel: "What was the state 2 weeks ago?"

### Implementation Example
```go
// Get customer state as it was 1 month ago
func (s *EventStore) GetAggregateAsOf(
    ctx context.Context,
    aggregateID string,
    asOf time.Time,
) (*Aggregate, error) {
    // Find latest snapshot before asOf
    snapshot := findLatestSnapshot(aggregateID, asOf)
    
    // Replay events since snapshot up to asOf
    events := getEventsSince(aggregateID, snapshot.EventNumber, asOf)
    
    // Reconstruct state
    return replayEvents(snapshot.State, events)
}
```

### Timeline: 3-4 hours

---

## Implementation Roadmap

| Phase | Duration | Status | Next Actions |
|-------|----------|--------|--------------|
| Phase 2 | ✅ DONE | COMPLETE | Ready for production deployment |
| Phase 3 | 2-3h | PLANNED | Extract microservice container |
| Phase 4a | 3-4h | PLANNED | Implement CQRS pattern |
| Phase 4b | 4-5h | PLANNED | Implement saga orchestrator |
| Phase 4c | 3-4h | PLANNED | Implement event replay/snapshots |
| **TOTAL** | **~12-20h** | **READY** | **Production-grade microservices** |

## Architecture Evolution

**Phase 2 (Current):** Monolith with Command Bus
- API Gateway + all services in one container
- Commands published to Redpanda (Kafka)
- Automatic fallback if Redpanda (Kafka) unavailable
- Full audit trail with events

**Phase 3:** Separated Microservices
- API Gateway (lightweight, stateless)
- BO Command Service (handles CRUD logic)
- Independent scaling/deployment
- Fault isolation

**Phase 4a:** CQRS Ready
- Write path: commands through bus
- Read path: optimized projections
- Independent scaling of reads vs writes
- Event store for complete history

**Phase 4b:** Saga-Enabled
- Complex workflows across aggregates
- Distributed transaction management
- Compensation/rollback support
- Multi-step orchestration

**Phase 4c:** Event Sourcing Complete
- Full replay capability
- Point-in-time reconstruction
- Complete audit trail
- Time-travel debugging

## Success Criteria

✅ **Phase 2 Complete:** 
- All Instance commands route through bus
- Automatic fallback working
- Zero compilation errors
- Production-ready

⏭️ **Phase 3 Ready:**
- Command handlers can be extracted
- No tight coupling with HTTP layer
- Service-to-service communication over RabbitMQ

⏭️ **Phase 4a Ready:**
- Events properly published
- Correlation IDs preserved
- Read/write separation natural

⏭️ **Phase 4b Ready:**
- Multi-step workflows can be orchestrated
- Commands are composable
- Compensation logic can be defined

⏭️ **Phase 4c Ready:**
- Events stored in event store
- Complete history available
- Snapshots can optimize replay

## Key Decisions Made

1. **Message Bus First**: All modifications go through RabbitMQ command bus
2. **Event Sourcing**: All changes generate audit events
3. **Correlation IDs**: Every command/event carries trace ID
4. **Graceful Fallback**: System works even if RabbitMQ unavailable
5. **Zero Breaking Changes**: HTTP API unchanged for clients

## Production Deployment Checklist

Before deploying to production:

- [ ] Phase 2 implementation complete and tested
- [ ] RabbitMQ configured with management plugin
- [ ] Command/event exchanges and queues created
- [ ] Handler registration in main.go complete
- [ ] Error handling tested (RabbitMQ down, network issues)
- [ ] Fallback path verified
- [ ] Monitoring/alerting configured
- [ ] Backup/disaster recovery plan
- [ ] Load testing completed
- [ ] Documentation reviewed

## Next Immediate Action

1. **Deploy Phase 2** to staging environment
2. **Run COMMAND_BUS_QUICK_CHECK.md** verification (18 steps)
3. **Monitor** logs and metrics for 24-48 hours
4. **Gather feedback** from team
5. **Schedule Phase 3** microservice extraction

## Questions?

Refer to:
- `COMMAND_BUS_VERIFICATION.md` - Detailed implementation verification
- `PHASE_2_INSTANCE_COMMANDS_COMPLETE.md` - Phase 2 details
- `MICROSERVICES_COMMAND_BUS.md` - Architecture deep dive
