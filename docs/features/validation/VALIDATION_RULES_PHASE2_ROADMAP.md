# VALIDATION RULES - PHASE 2 ROADMAP (RabbitMQ Integration)

**Current Phase**: ✅ Phase 1 Complete (REST API deployed)
**Next Phase**: ⏳ Phase 2 - RabbitMQ Integration (Starting Oct 26)
**Timeline**: 2-3 weeks for full integration

---

## 📊 CURRENT STATE (POST-DEPLOYMENT)

### ✅ What's Live (Phase 1)
- REST API with 8 endpoints
- PostgreSQL database (2 tables, 7 indexes)
- React UI with form builder
- Multi-tenant support
- Audit trail tracking
- Input validation & error handling

### ❌ What's NOT Yet Integrated (Phase 2)
- RabbitMQ event consumer
- Event-driven rule execution
- Automatic validation triggers
- Real-time async processing

### 🎯 Why Phase 2 Matters
**Current State**: Manual/API-driven validation
- Users call REST API to execute rules
- Synchronous only (wait for response)
- No automatic triggering
- Perfect for MVP ✅

**Phase 2 Goal**: Event-driven validation
- Rules execute automatically on data changes
- Async background processing
- Real-time feedback via events
- Production-grade automation ✅

---

## 🔄 RABBITMQ INTEGRATION ARCHITECTURE

### Current Event Stream (Exists)
```
Data Changes
  ↓
SemanticPublisher (already built)
  ├─ semantic.changes exchange
  ├─ semantic.drift exchange
  ├─ semantic.audit exchange
  └─ semantic.notifications exchange
  ↓
RabbitMQ (port 5673)
  ├─ Topic exchanges
  ├─ Routing keys
  └─ Existing consumers (semantic layer)
```

### Phase 2 Addition (To Build)
```
Data Changes
  ↓
SemanticPublisher (existing)
  ├─ semantic.changes exchange
  └─ publish event (e.g., "customer.created", "order.updated")
  ↓
RabbitMQ (port 5673)
  ├─ NEW: validation-rules exchange
  ├─ NEW: validation-results queue
  └─ Routing: *.*.* → validation consumer
  ↓
ValidationRuleConsumer (NEW - to build)
  ├─ Subscribe to validation-rules exchange
  ├─ Filter events by target_entity
  ├─ Query database for matching rules
  ├─ Execute validation engine
  ├─ Publish results via ValidationResultPublisher
  └─ Update audit trail
  ↓
Downstream Consumers (Future)
  ├─ Alert service (webhooks, email, Slack)
  ├─ Analytics service (metrics, dashboards)
  ├─ Remediation service (auto-fixes)
  └─ Compliance service (audit logs)
```

---

## 🛠️ PHASE 2 BUILD PLAN

### Week 1: Foundation (Oct 26 - Nov 1)

#### Day 1-2: Data Provider Implementation
**File**: `backend/internal/validation/data_provider.go`

```go
type DataProvider interface {
    // Get specific field value from entity
    GetFieldValue(ctx context.Context, tenantID, entity, id, field string) (interface{}, error)
    
    // Check field uniqueness
    CheckUniqueness(ctx context.Context, tenantID, entity, field, value string) (bool, error)
    
    // Check referential integrity
    CheckReferentialIntegrity(ctx context.Context, tenantID, sourceEntity, sourceID, 
        sourceField, targetEntity, targetField string) (bool, error)
    
    // Generic query for complex validations
    QueryData(ctx context.Context, query string, params ...interface{}) ([]map[string]interface{}, error)
}

type PostgresDataProvider struct {
    db *sql.DB
}
```

**Why needed**: 
- Uniqueness validation requires DB queries
- Referential integrity needs FK lookups
- Complex rules need data access
- Currently: Engine uses mock data only

**Effort**: 1-2 days
**Tests**: Unit tests with mock database

#### Day 3-5: RabbitMQ Consumer
**File**: `backend/internal/events/validation_consumer.go`

```go
type ValidationRuleConsumer struct {
    channel *amqp.Channel
    engine *validation.ValidationEngine
    dataProvider validation.DataProvider
    publisher *ValidationResultPublisher
    db *sql.DB
}

func (c *ValidationRuleConsumer) Start(ctx context.Context) error {
    // 1. Declare validation-rules exchange
    // 2. Create queue
    // 3. Bind to exchange with routing key
    // 4. Consume messages
    // 5. Execute rules
    // 6. Publish results
}

func (c *ValidationRuleConsumer) handleMessage(msg *amqp.Delivery) error {
    // 1. Parse SemanticChangeEvent
    // 2. Extract target_entity and entity_id
    // 3. Query matching validation rules from DB
    // 4. Execute each rule with actual data
    // 5. Publish results
    // 6. Update audit trail
}
```

**Event Example**:
```json
{
  "event_type": "entity_created",
  "entity_type": "Customer",
  "entity_id": "uuid-1234",
  "tenant_id": "uuid-tenant",
  "datasource_id": "uuid-datasource",
  "timestamp": "2025-10-26T10:00:00Z",
  "data": {
    "id": "uuid-1234",
    "email": "user@example.com",
    "name": "John Doe"
  }
}
```

**Effort**: 2-3 days
**Tests**: Integration tests with RabbitMQ

### Week 2: Result Publishing & Wiring (Nov 2 - Nov 8)

#### Day 1-2: Validation Result Publisher
**File**: `backend/internal/events/validation_publisher.go`

```go
type ValidationResultPublisher struct {
    channel *amqp.Channel
}

type ValidationResultEvent struct {
    RuleID        string                   `json:"rule_id"`
    RuleName      string                   `json:"rule_name"`
    RuleType      string                   `json:"rule_type"`
    TenantID      string                   `json:"tenant_id"`
    EntityID      string                   `json:"entity_id"`
    EntityType    string                   `json:"entity_type"`
    Status        string                   `json:"status"` // pass, fail
    Message       string                   `json:"message"`
    Severity      string                   `json:"severity"`
    ExecutedAt    time.Time                `json:"executed_at"`
    ExecutionTime int64                    `json:"execution_time_ms"`
}

func (p *ValidationResultPublisher) PublishResult(ctx context.Context, result ValidationResultEvent) error {
    // Publish to validation.results exchange
    // Route to appropriate consumers
}
```

**Effort**: 1-2 days

#### Day 3-4: Integration into Startup
**File**: `backend/cmd/server/main.go`

```go
// In startup sequence (around line 370):
validationConsumer, err := events.NewValidationRuleConsumer(
    amqpChannel,
    validationEngine,
    dataProvider,
    resultPublisher,
    db,
)
if err != nil {
    logging.GetLogger().Sugar().Warnf("Failed to initialize validation consumer: %v", err)
}

// Start consumer in background
go validationConsumer.Start(context.Background())

logging.GetLogger().Sugar().Info("Validation rule consumer started")
```

**Effort**: 1 day

#### Day 5: End-to-End Testing
- Publish test events to semantic.changes
- Verify validation consumer picks them up
- Check rules execute correctly
- Verify results published to validation.results
- Confirm audit trail updated

**Effort**: 1 day

### Week 3: Polish & Production Readiness (Nov 9 - Nov 15)

#### Day 1-2: Error Handling & Retry Logic
```go
// Handle consumer failures gracefully
- Max retries: 3
- Backoff strategy: exponential (1s, 2s, 4s)
- Dead-letter queue for failures
- Log all errors with full context
```

#### Day 3: Performance Optimization
```go
- Batch rule execution (execute 10+ rules in one go)
- Connection pooling for RabbitMQ
- Caching of rule definitions
- Rate limiting to prevent overload
```

#### Day 4: Monitoring & Observability
```go
- Log validation events (INFO level)
- Track consumer lag
- Monitor execution times
- Alert on repeated failures
- Metrics: rules_executed, rules_passed, rules_failed
```

#### Day 5: Integration Testing & Documentation
```
- Full end-to-end scenario tests
- Load testing (100s of rules, 1000s of events)
- Documentation of new components
- Troubleshooting guide
- Deployment checklist for Phase 2
```

---

## 📋 BUILD CHECKLIST

### Pre-Build (Before Week of Oct 26)
- [ ] Review current RabbitMQ setup in `docker-compose.yml`
- [ ] Verify SemanticPublisher integration
- [ ] List all existing exchanges and queues
- [ ] Identify event format and routing patterns
- [ ] Plan queue names and routing keys

### Build Phase
- [ ] Implement DataProvider interface
- [ ] Build ValidationRuleConsumer
- [ ] Build ValidationResultPublisher
- [ ] Integrate into server startup
- [ ] Add error handling & retries
- [ ] Add monitoring & metrics

### Testing Phase
- [ ] Unit tests for DataProvider
- [ ] Integration tests with RabbitMQ
- [ ] End-to-end validation flows
- [ ] Load testing (performance)
- [ ] Error scenario testing

### Deployment Phase
- [ ] Document new environment variables
- [ ] Update deployment checklist
- [ ] Create runbook for operations
- [ ] Plan rollout strategy
- [ ] Get approval from stakeholders

---

## 💾 NEW FILES TO CREATE

| File | Lines | Purpose |
|------|-------|---------|
| `backend/internal/validation/data_provider.go` | 150-200 | DB query interface |
| `backend/internal/validation/postgres_provider.go` | 200-300 | PostgreSQL implementation |
| `backend/internal/events/validation_consumer.go` | 300-400 | RabbitMQ consumer |
| `backend/internal/events/validation_publisher.go` | 100-150 | Result publisher |
| `backend/internal/handlers/validation_handler.go` | 200-300 | Message handlers |
| `backend/internal/validation/model.go` (update) | - | Add event structs |
| `backend/migrations/add_validation_audit_events.sql` | 50-100 | Link audit to events |

**Total new code**: ~1,500-2,000 lines

---

## 🚀 DEPLOYMENT TIMELINE

| Week | Phase | Deliverable | Status |
|------|-------|-------------|--------|
| Oct 19 | 1 | REST API + UI | ✅ COMPLETE |
| Oct 26 | 2 | Data Provider + Consumer | ⏳ READY TO START |
| Nov 2 | 2 | Result Publisher + Integration | ⏳ WEEK 2 |
| Nov 9 | 2 | Polish + Testing | ⏳ WEEK 3 |
| Nov 16 | 2 | Deploy to Production | ⏳ PRODUCTION |

---

## 🎯 SUCCESS METRICS (Phase 2)

### Functional Requirements
- [ ] Validation rules execute automatically on semantic change events
- [ ] Results published to RabbitMQ within 1 second
- [ ] Audit trail records event-triggered executions
- [ ] Errors handled gracefully with retry logic
- [ ] Failed messages sent to dead-letter queue

### Non-Functional Requirements
- [ ] Consumer latency: < 500ms per event
- [ ] Throughput: 1000+ events/second
- [ ] Error rate: < 1% (99%+ success)
- [ ] Memory usage: < 500MB
- [ ] Zero data loss in case of failures

### Operational Requirements
- [ ] Documented troubleshooting guide
- [ ] Monitoring alerts configured
- [ ] Runbook for on-call support
- [ ] Performance baseline established
- [ ] Rollback procedure documented

---

## 🔗 INTEGRATION POINTS

### Existing Services That Will Connect
1. **SemanticPublisher** (already running)
   - Will publish events that trigger validation
   - Validation consumer subscribes to semantic.changes

2. **Semantic Layer** (core system)
   - Data change events feed validation pipeline
   - Validation results inform semantic drift detection

3. **Audit Service** (existing)
   - Validation execution recorded in audit_events
   - Compliance tracking via audit trail

### Future Services That Will Consume Results
1. **Alert Service** (to build)
   - Webhook notifications on rule violations
   - Email/Slack alerts

2. **Analytics Service** (to build)
   - Validation metrics dashboard
   - Rule effectiveness tracking

3. **Remediation Service** (to build)
   - Auto-fix capabilities
   - Workflow integration

---

## 📚 REFERENCE ARCHITECTURE

### RabbitMQ Exchanges (After Phase 2)
```
semantic.changes
  ↓ (customer.created, order.updated, etc.)
  ├─ Subscribers: data warehouse, analytics, validation
  └─ [Semantic change events]

semantic.drift
  ↓ (existing drift detection)
  └─ [Drift detection events]

semantic.audit
  ↓ (existing audit trail)
  └─ [Audit events]

semantic.notifications
  ↓ (existing notifications)
  └─ [Notification events]

validation.results (NEW in Phase 2)
  ↓ (validation.passed, validation.failed)
  ├─ Subscribers: alerts, analytics, compliance
  └─ [Validation result events]

validation.alerts (NEW in Phase 2)
  ↓ (rule.violation)
  └─ [Alert subscribers]
```

---

## 🔐 SECURITY CONSIDERATIONS

### Tenant Isolation (Phase 2)
- ✅ Consumer filters events by tenant_id
- ✅ Only processes rules for that tenant
- ✅ Results include tenant_id for audit
- ✅ No cross-tenant data access possible

### Message Validation
- ✅ Validate event schema
- ✅ Check tenant authorization
- ✅ Verify entity_id format
- ✅ Sanitize JSON payloads

### Error Handling
- ✅ Don't expose database schema in errors
- ✅ Log errors without sensitive data
- ✅ Use generic error messages
- ✅ Track failed attempts

---

## 📞 QUESTIONS TO ANSWER BEFORE STARTING

1. **Event Format**: What do SemanticChangeEvents look like?
   - Field names? Timezone handling? Nested data?

2. **Queue Strategy**: Should validation use individual queues per tenant or shared queue?
   - Performance implications?
   - Scaling strategy?

3. **Failure Handling**: What happens if validation consumer crashes?
   - Replay events from dead-letter queue?
   - Manual review of failed validations?

4. **Performance**: How many validation rules per tenant?
   - Expected rule count: 10? 100? 1,000?
   - Expected events/second: 10? 100? 1,000?

5. **Integration**: Are there existing RabbitMQ patterns to follow?
   - Other consumers implemented already?
   - Shared utilities for consumer setup?
   - Standard error handling patterns?

---

## 📖 DOCUMENTATION TO CREATE

| Document | Purpose |
|----------|---------|
| `VALIDATION_RULES_PHASE2_ARCHITECTURE.md` | Detailed Phase 2 design |
| `VALIDATION_RULES_PHASE2_TESTING.md` | Test strategy and scenarios |
| `VALIDATION_RULES_PHASE2_DEPLOYMENT.md` | Deployment checklist |
| `RABBITMQ_CONSUMER_RUNBOOK.md` | Operations guide |
| `VALIDATION_RULES_TROUBLESHOOTING.md` | Common issues & fixes |

---

## ✨ SUMMARY

**Current Status**: Phase 1 ✅ COMPLETE
- REST API operational
- Database persisted
- UI ready for users
- 4 test rules created

**Next Phase**: Phase 2 ⏳ READY TO START
- RabbitMQ consumer (event-driven execution)
- Data provider (DB queries for validation)
- Result publisher (broadcast outcomes)
- Integration testing

**Timeline**: 2-3 weeks (Oct 26 - Nov 15)
**Effort**: ~2,000 lines of Go code
**Priority**: HIGH (enables automation)

**Ready to start Phase 2?** Check prerequisites and begin Week 1 on Oct 26.

---

**Planning Phase**: October 19, 2025
**Implementation Target**: Week of October 26, 2025
**Production Target**: November 16, 2025
**Status**: ✅ Phase 1 Live | ⏳ Phase 2 Planned

