# RabbitMQ Event-Driven Architecture Decision Document

## Executive Summary

This document outlines the event-driven architecture recommendation for the Northwind Business Object system using RabbitMQ for message publishing and microservices orchestration.

## 🏗️ Architecture Decision: RabbitMQ + Microservices

### Recommendation: **Hybrid Approach**

**Primary Pattern: Monolith with Event Bus**
- Keep the main backend as a single deployable service
- Use RabbitMQ as an **event bus** for internal communication
- Enable future microservices decomposition without major refactoring

### Why RabbitMQ?

| Criterion | RabbitMQ | Direct Events | Redis Pub/Sub |
|-----------|----------|---------------|---------------|
| **Persistence** | ✅ Durable queues | ❌ Lost on restart | ❌ Lost on restart |
| **Message Ordering** | ✅ FIFO per queue | ✅ In-memory only | ❌ No guarantees |
| **Delivery Guarantees** | ✅ At-least-once | ❌ Best effort | ❌ Best effort |
| **Dead Letter Handling** | ✅ Native DLQ support | ❌ Manual | ❌ Manual |
| **Microservices Ready** | ✅ True decoupling | ⚠️ Tightly coupled | ⚠️ Hard to scale |
| **Operational Overhead** | ⚠️ Requires running broker | ✓ Embedded | ⚠️ Separate Redis |

**Verdict:** RabbitMQ is the production-grade choice for event-driven systems with audit and compliance needs.

---

## 📊 Event Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                       Frontend (React)                          │
│  (Displays business objects, creates instances)                 │
└────────────────────┬────────────────────────────────────────────┘
                     │ HTTP REST
                     │
┌────────────────────▼────────────────────────────────────────────┐
│                 Monolithic Backend (Go)                         │
│                                                                 │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │  HTTP Handler Layer                                      │  │
│  │  - POST /api/business-objects (Create BO)               │  │
│  │  - POST /api/bo/{boKey}/instances (Create Instance)     │  │
│  │  - PUT /api/bo/{boKey}/instances/{id} (Update Instance) │  │
│  │  - DELETE /api/bo/{boKey}/instances/{id} (Delete)       │  │
│  └────────────────────┬─────────────────────────────────────┘  │
│                       │                                         │
│  ┌────────────────────▼─────────────────────────────────────┐  │
│  │  Business Logic Layer (Services)                         │  │
│  │  - BusinessObjectService                                │  │
│  │  - BulkImportService                                    │  │
│  │  - WorkflowService                                      │  │
│  │  - EventPublisher  ◄────────────────────┐               │  │
│  └────────────────────┬─────────────────────┼───────────────┘  │
│                       │                     │                   │
│                       │ Publish Events      │ Event Methods     │
│                       ▼                     └───────────────────┘
└───────────────────────┼───────────────────────────────────────────┘
                        │ AMQP
                        │
            ┌───────────▼───────────┐
            │   RabbitMQ Broker     │
            │  (Development/Production)
            │                       │
            │  Exchanges:           │
            │  - semlayer.bo (topic)│
            │                       │
            │  Queues:              │
            │  - bo.created         │
            │  - bo.updated         │
            │  - instance.created   │
            │  - instance.updated   │
            │  - workflow.events    │
            └───────────┬───────────┘
                        │
        ┌───────────────┼───────────────┐
        │               │               │
        ▼               ▼               ▼
    ┌────────┐    ┌──────────┐    ┌─────────────┐
    │ Audit  │    │Workflow  │    │ Notifications
    │Service │    │Engine    │    │ Service
    │        │    │          │    │
    └────────┘    └──────────┘    └─────────────┘
```

---

## 🔄 Event Flow Example: Creating a Business Object Instance

```
1. Frontend sends:
   POST /api/bo/customer/instances
   { "coreFields": { "name": "John Acme" } }

2. Handler validates & calls service:
   instance, err := service.CreateInstance(ctx, tenantID, userID, instance)

3. Service creates in database:
   INSERT INTO bo_instances (...)

4. Service publishes event:
   publisher.PublishInstanceCreated(ctx, instance, userID)

5. Event published to RabbitMQ:
   Exchange: semlayer.bo
   Routing Key: instance.created
   Message:
   {
     "id": "uuid-123",
     "type": "instance.created",
     "tenant_id": "tenant-456",
     "entity_type": "instance",
     "data": { "id": "instance-789", ... },
     "user_id": "user-001",
     "timestamp": "2025-10-18T19:30:00Z"
   }

6. Consumers react:
   - Audit Service: Logs to bo_audit_log
   - Workflow Engine: Triggers notifications
   - Analytics: Records metrics
```

---

## 📦 Implementation Details

### 1. EventPublisher Service (COMPLETED)

Location: `backend/internal/services/event_publisher.go`

**Key Methods:**
```go
type EventPublisher struct {
    conn     *amqp.Connection
    channel  *amqp.Channel
    exchange string
    enabled  bool
}

// Publishing methods
PublishBOCreated(ctx, bo, userID)
PublishBOUpdated(ctx, bo, userID)
PublishBODeleted(ctx, tenantID, boKey, userID)
PublishInstanceCreated(ctx, instance, userID)
PublishInstanceUpdated(ctx, instance, userID)
PublishInstanceDeleted(ctx, tenantID, boKey, instanceID, userID)
PublishWorkflowEvent(ctx, eventType, workflowID, tenantID, userID, data)
```

**Features:**
- ✅ Graceful fallback if RabbitMQ unavailable
- ✅ JSON serialization of events
- ✅ Persistent message delivery
- ✅ Topic-based routing

### 2. EventConsumer (For Future Microservices)

**Pattern for consumers:**
```go
consumer, err := NewEventConsumer(rabbitMQURL, "audit-service-queue")
msgs, err := consumer.Subscribe("instance.*", handleInstanceEvent)

for msg := range msgs {
    var event *BOEvent
    json.Unmarshal(msg.Body, &event)
    // Process event
    msg.Ack(false)
}
```

### 3. Deployment Options

#### Option A: Docker Compose (Development)
```yaml
rabbitmq:
  image: rabbitmq:4-management
  ports:
    - "5672:5672"  # AMQP
    - "15672:15672" # Management UI
  environment:
    RABBITMQ_DEFAULT_USER: guest
    RABBITMQ_DEFAULT_PASS: guest
```

#### Option B: Cloud Managed (Production)
- **AWS:** Amazon MQ for RabbitMQ
- **Azure:** Azure Service Bus
- **GCP:** Google Cloud Pub/Sub

---

## 🛣️ Microservices Migration Path

### Current State (Monolith)
```
+-----------------+
| Single Backend  |
|  (All services) |
+-----------------+
```

### Future State (Decomposed)
```
┌──────────────────┐     ┌──────────────────┐     ┌──────────────────┐
│  Core API        │     │  Audit Service   │     │  Workflow Engine │
│  (BO/Instance    │     │  (Event logs)    │     │  (State machine) │
│   CRUD)          │     │                  │     │                  │
└────────┬─────────┘     └────────┬─────────┘     └────────┬─────────┘
         │                        │                        │
         └────────────┬───────────┴────────────┬───────────┘
                      │                        │
                 AMQP │                        │ Topic Exchange
                      ▼                        ▼
              ┌─────────────────────────────────────┐
              │   RabbitMQ Broker (Message Bus)     │
              │  - Decouples services              │
              │  - Enables independent deployment  │
              │  - Provides audit trail            │
              └─────────────────────────────────────┘
```

### Migration Steps
1. ✅ Implement EventPublisher in monolith (DONE)
2. ⏳ Add consumers for audit events in backend
3. ⏳ Externalize workflow engine as microservice
4. ⏳ Create analytics/reporting microservice
5. ⏳ Implement cross-service authentication

---

## 🔐 Security Considerations

### 1. Message Encryption
```go
// In production, enable TLS for RabbitMQ
// rabbitmq://guest:password@rabbitmq-host:5671
// Use RABBITMQ_AMQP_URI=amqps://...
```

### 2. Access Control
```
- Restrict BO operations by role/tenant
- Tag messages with tenant_id for multi-tenancy
- Validate user_id on all operations
```

### 3. Event Sensitivity
```
- Don't include sensitive data in events (PII)
- Use references (IDs) instead of full records
- Log event consumption for audit
```

---

## 📊 Event Types & Routing

### Core BO Events
```
Routing Pattern: business_object.<action>

- business_object.created     → bo.created queue
- business_object.updated     → bo.updated queue
- business_object.deleted     → bo.deleted queue
- business_object.cloned      → bo.cloned queue
```

### Instance Events
```
Routing Pattern: instance.<action>

- instance.created            → instance.created queue
- instance.updated            → instance.updated queue
- instance.deleted            → instance.deleted queue
```

### Workflow Events
```
Routing Pattern: workflow.<status>

- workflow.started            → workflow.events queue
- workflow.progress           → workflow.events queue
- workflow.completed          → workflow.events queue
- workflow.failed             → workflow.events queue
```

---

## ⚙️ Configuration

### Environment Variables
```bash
# .env.development
RABBITMQ_URL=amqp://guest:guest@localhost:5672

# .env.production
RABBITMQ_URL=amqps://user:pass@rabbitmq.example.com:5671
RABBITMQ_VHOST=/tenants
```

### Backend Initialization
```go
// In api.go / main
eventPublisher, err := services.NewEventPublisher(
    os.Getenv("RABBITMQ_URL"),
)
defer eventPublisher.Close()

boHandler := handlers.NewBusinessObjectHandler(
    boService,
    eventPublisher,
)
```

---

## 📈 Monitoring & Observability

### RabbitMQ Management UI
- Access at `http://localhost:15672` (dev)
- Monitor queue depths
- Inspect dead letter queues

### Metrics to Track
```
- Messages published per type
- Event latency (publish → consume)
- Queue depth (backlog)
- Dead letter queue size
- Consumer lag
```

### Example Prometheus Metrics
```go
type EventMetrics struct {
    PublishedTotal     prometheus.Counter   // Total events published
    PublishedErrors    prometheus.Counter   // Publishing errors
    ConsumedTotal      prometheus.Counter   // Total events consumed
    ConsumeLag         prometheus.Histogram // Consumer lag
}
```

---

## 🚀 Quick Start: Local Development

### 1. Start RabbitMQ
```bash
docker run -d \
  --name rabbitmq \
  -p 5672:5672 \
  -p 15672:15672 \
  rabbitmq:4-management

# Access UI: http://localhost:15672
# Login: guest / guest
```

### 2. Build & Run Backend
```bash
cd backend
go run ./cmd/api/main.go
# Connects to RabbitMQ automatically if RABBITMQ_URL is set
```

### 3. Test Event Publishing
```bash
# Create a BO instance
curl -X POST http://localhost:8080/api/bo/customer/instances \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: tenant-1" \
  -H "X-User-ID: user-1" \
  -d '{
    "businessObjectKey": "customer",
    "coreFieldValues": { "name": "Acme Corp" }
  }'

# Observe in RabbitMQ UI:
# - semlayer.bo exchange created
# - instance.created message published
```

---

## 📋 Fallback Strategy

### Graceful Degradation
If RabbitMQ is unavailable:
1. EventPublisher initialized with `enabled=false`
2. Event publishing calls silently skip (no errors)
3. BO/Instance operations continue normally
4. No data loss (all changes persisted to PostgreSQL)
5. Audit logging still works (database-based)

```go
// Example: CreateInstance with event fallback
func (s *Service) CreateInstance(...) {
    // Create in database (always succeeds)
    err := s.createInDB(instance)
    
    // Try to publish event (graceful skip if RabbitMQ down)
    s.eventPublisher.PublishInstanceCreated(ctx, instance, userID)
    // ↑ Returns nil if RabbitMQ disabled, no error thrown
    
    return instance, nil
}
```

---

## 🔄 Next Steps

1. **Install RabbitMQ Dependency**
   ```bash
   go get github.com/rabbitmq/amqp091-go
   ```

2. **Implement Instance Operations in Service**
   ✅ CreateInstance, GetInstance, ListInstances, UpdateInstance, DeleteInstance

3. **Create Event Consumers**
   - Audit log consumer
   - Workflow event consumer
   - Notification consumer

4. **Implement GraphQL Layer** (separate task)
   - Define schema for BO queries
   - Implement resolvers

5. **Bulk Import/Export Service** (separate task)
   - CSV import handler
   - JSON export handler

6. **Workflow Engine** (separate task)
   - State machine for instance lifecycle
   - Trigger system for events

---

## 📚 Resources

- [RabbitMQ Tutorials](https://www.rabbitmq.com/getstarted.html)
- [AMQP 0.9.1 Protocol](https://www.rabbitmq.com/resources/specs/amqp0-9-1.pdf)
- [Go RabbitMQ Client](https://pkg.go.dev/github.com/rabbitmq/amqp091-go)
- [Event Sourcing Pattern](https://martinfowler.com/eaaDev/EventSourcing.html)

---

## ✅ Decision Summary

| Aspect | Decision | Rationale |
|--------|----------|-----------|
| **Message Broker** | RabbitMQ | Durability, ordering, enterprise-ready |
| **Architecture** | Monolith + Event Bus | Gradual path to microservices |
| **Deployment** | Docker (dev), Managed Cloud (prod) | Cost-effective, scalable |
| **Fallback** | Silent disable if unavailable | Production resilience |
| **Consumer Model** | Multi-consumer per queue | Parallel processing |
| **Event Format** | JSON | Language-agnostic, debuggable |

**Status:** ✅ APPROVED for implementation
