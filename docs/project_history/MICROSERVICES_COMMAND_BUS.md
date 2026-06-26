# Microservices Architecture: RabbitMQ Command Bus Pattern

## Overview

This document describes the transition from a monolithic REST API architecture to a **microservices architecture using RabbitMQ as a command bus**. All CRUD operations for Business Objects now flow through message queues instead of direct HTTP endpoints.

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────┐
│                          API GATEWAY LAYER (HTTP)                        │
│  • BusinessObjectHandler - thin wrapper around command bus               │
│  • Deserializes requests → publishes commands → waits for responses      │
└──────────────────────────┬──────────────────────────────────────────────┘
                           │
                    HTTP Request
                           │
        ┌──────────────────▼──────────────────┐
        │    COMMAND BUS (RabbitMQ)           │
        │  semlayer.commands (topic exchange) │
        └──────┬───────────────────────┬──────┘
               │                       │
    Command:  │                       │
    Create    │   CommandPublisher    │  CommandConsumer
    Update    │   (API Gateway)       │  (Microservice)
    Delete    │                       │
    Clone     │                       │
               │                       │
        ┌──────▼──────────────────────▼──────┐
        │   BO MICROSERVICE LAYER             │
        │ • BOCommandHandler                  │
        │ • BusinessObjectService (logic)     │
        │ • Executes command handlers         │
        │ • Publishes domain events           │
        └────┬─────────────────────────┬──────┘
             │                         │
       Event │                         │ Response
       Pub   │                         │ on reply
             │                         │ queue
        ┌────▼──────────────────────┐  │
        │ EVENT BUS (RabbitMQ)      │  │
        │semlayer.events (durable)  │  │
        └────────────────────────────┘  │
                                        │
        ┌───────────────────────────────▼──────┐
        │    REPLY QUEUE (RabbitMQ)            │
        │  semlayer.replies (direct exchange)  │
        └──────────┬──────────────────────────┘
                   │
                   │ CommandResponse
                   │
        ┌──────────▼────────────┐
        │  API GATEWAY          │
        │  (Waiting for reply)  │
        │  Returns to client    │
        └───────────────────────┘
```

## Key Concepts

### 1. Command Bus Pattern

**Commands** are requests to perform an action. All BO CRUD operations are implemented as commands:

- `command.bo.create` - Create a new Business Object
- `command.bo.update` - Update an existing BO
- `command.bo.delete` - Delete a BO
- `command.bo.clone` - Clone a BO
- `command.instance.create` - Create BO instance
- `command.instance.update` - Update BO instance
- `command.instance.delete` - Delete BO instance

### 2. Request/Reply Pattern

```
API Gateway               RabbitMQ                    BO Service
    │
    ├─(1) PublishCommand────────► semlayer.commands
    │                                    │
    │                            ┌───────▼─────────┐
    │                            │ Command Handler │
    │                            │ Executes logic  │
    │                            └───────┬─────────┘
    │
    │◄─(2) PublishResponse──────────────┤
    │    (on reply queue)                │
    │                                    │
    ├─(3) WaitForCommandResponse         │
    │                                    │
    ├─(4) Return to Client               │
    │                                    │
```

**Key Feature**: Correlation IDs link commands to responses across the system:
- Command has: `correlationID = uuid`
- Response has: `correlationID = same uuid`
- This enables request tracing and timeout handling

### 3. Event Sourcing for Audit Trail

**Events** are immutable facts about what happened. After every successful command, an event is published:

```
Command Execution Flow:
1. API receives CreateBO request
2. Publishes Command to semlayer.commands
3. BO service processes command
4. BO service publishes BOCreated event
5. Event is persisted in semlayer.events (durable)
6. Other services subscribe to events (audit, analytics, etc.)
7. Response is sent back via reply queue
```

## File Structure

### New Files Created

```
backend/internal/services/
├── command_bus.go               # NEW: Command pub/sub infrastructure
│   ├── CommandPublisher         # Publishes commands
│   ├── CommandConsumer          # Subscribes to commands
│   ├── CommandHandler           # Callback type for handlers
│   ├── CommandResponse          # Response structure
│   └── Command types & enums
│
├── bo_command_handler.go        # NEW: BO-specific command handlers
│   ├── BOCommandHandler         # Main handler service
│   ├── HandleCreateBO           # Command handler: CREATE
│   ├── HandleUpdateBO           # Command handler: UPDATE
│   ├── HandleDeleteBO           # Command handler: DELETE
│   └── HandleCloneBO            # Command handler: CLONE
│
└── event_publisher.go           # REFACTORED: Enhanced with command types
    ├── Command struct           # NEW: Command definition
    ├── CommandType enum         # NEW: Command types
    ├── Event struct             # ENHANCED: Added correlationID
    ├── EventPublisher           # IMPROVED: Better exchange handling
    └── EventConsumer            # IMPROVED: Event store pattern
```

### Modified Files

```
backend/internal/handlers/
└── businessobject_handler.go    # REFACTORED: API Gateway -> Command Bus
    ├── CommandPublisher field   # NEW: Command bus integration
    ├── waitForCommandResponse   # NEW: Reply queue listener
    ├── CreateBusinessObject     # REFACTORED: Publishes command
    ├── UpdateBusinessObject     # REFACTORED: Publishes command
    ├── DeleteBusinessObject     # REFACTORED: Publishes command
    ├── CloneBusinessObject      # REFACTORED: Publishes command
    ├── ListBusinessObjects      # UNCHANGED: Direct read (no command)
    └── GetBusinessObject        # UNCHANGED: Direct read (no command)
```

## Data Flow Examples

### Create Business Object via Command Bus

```go
// 1. HTTP REQUEST (API Gateway)
POST /api/business-objects
{
  "key": "customer",
  "displayName": "Customer",
  ...
}

// 2. HANDLER PUBLISHES COMMAND
handler.commandBus.PublishCommand(
  ctx,
  CommandCreateBO,
  tenantID,
  userID,
  requestData,
)  // Returns correlationID

// 3. COMMAND MESSAGE (to semlayer.commands exchange)
{
  "id": "cmd-123",
  "type": "command.bo.create",
  "tenant_id": "tenant-456",
  "user_id": "user-789",
  "correlation_id": "corr-abc",
  "data": { /* request */ },
  "timestamp": "2024-01-15T10:30:00Z"
}

// 4. COMMAND CONSUMER (BO Microservice)
consumer.RegisterHandler(CommandCreateBO, handler.HandleCreateBO)
consumer.Subscribe(ctx, "command.bo.*")  // Starts listening

// 5. HANDLER EXECUTION
response := handler.HandleCreateBO(ctx, command)
// - Calls boService.CreateBusinessObject()
// - Publishes BOCreated event
// - Returns CommandResponse

// 6. RESPONSE MESSAGE (to semlayer.replies exchange)
{
  "id": "resp-123",
  "correlation_id": "corr-abc",
  "status": "success",
  "message": "BO created: customer",
  "data": { /* created BO */ },
  "timestamp": "2024-01-15T10:30:01Z"
}

// 7. API GATEWAY RECEIVES RESPONSE
// - Waits on temp queue bound to semlayer.replies with routing key = correlationID
// - Receives response within 10 second timeout
// - Returns to HTTP client

// 8. HTTP RESPONSE
200 Created
{
  "id": "bo-123",
  "key": "customer",
  "displayName": "Customer",
  ...
}
```

### Event Published After Command

```go
// After successful command execution, event is published:
{
  "id": "evt-456",
  "type": "event.bo.created",
  "entity_type": "business_object",
  "entity_id": "bo-123",
  "entity_key": "customer",
  "tenant_id": "tenant-456",
  "user_id": "user-789",
  "correlation_id": "corr-abc",  // Links to command
  "data": { /* full BO */ },
  "timestamp": "2024-01-15T10:30:01Z",
  "metadata": {}
}

// Persisted in semlayer.events (durable, DLQ, TTL)
// Subscribers can:
// - Build audit logs
// - Update search indices
// - Trigger workflows
// - Replicate to other systems
```

## Microservices Benefits

### 1. Loose Coupling
- API Gateway doesn't import BO service code
- Services communicate via messages only
- Easy to replace implementations

### 2. Scalability
- Multiple BO service instances can consume commands
- Load balancing via RabbitMQ queues
- Independent scaling per service

### 3. Reliability
- Commands are transient (fire-and-forget semantics)
- Events are persistent (event sourcing)
- Dead letter queues for failed messages

### 4. Audit Trail
- Every command is logged with correlation ID
- Every event is persisted with audit metadata
- Full traceability across services

### 5. Future-Ready
- Easy to extract services to separate containers
- Each service can have its own database
- CQRS pattern can be implemented later

## Fallback Mechanism

If RabbitMQ is not available or disabled:

1. Command bus is marked as disabled
2. API Gateway falls back to direct service calls
3. No command/response semantics, just sync calls
4. Events are still published (if EventPublisher is enabled)
5. System works but without microservices benefits

```go
// Automatic fallback in handler:
if !h.enabled {
    // Direct service call (monolith mode)
    bo, err := h.boService.CreateBusinessObject(...)
    return
}

// Command bus mode
correlationID, err := h.commandBus.PublishCommand(...)
response, err := h.waitForCommandResponse(...)
```

## RabbitMQ Configuration

### Exchanges

| Exchange | Type | Durable | Purpose |
|----------|------|---------|---------|
| `semlayer.commands` | topic | false | Command routing (transient) |
| `semlayer.events` | topic | true | Event distribution (persistent) |
| `semlayer.replies` | direct | false | Command response routing |
| `semlayer.dlx` | topic | true | Dead letter queue for errors |

### Queue Configuration

**Command Queues** (Transient):
- Named: `{service-name}-commands`
- Not durable
- Auto-delete after consumer disconnects
- No TTL (commands discarded if not processed immediately)

**Event Queues** (Persistent):
- Named: `{service-name}-events`
- Durable (survive broker restart)
- TTL: 24 hours
- DLQ: `semlayer.dlx`

### Routing Keys

Commands: `command.bo.create`, `command.bo.update`, `command.bo.delete`, `command.bo.clone`

Events: `business_object.event.bo.created`, `business_object.event.bo.updated`, etc.

## Testing Strategies

### Unit Tests
Test command handlers directly:
```go
handler := NewBOCommandHandler(mockService, mockPublisher)
response, err := handler.HandleCreateBO(ctx, command)
assert.Equal(t, CommandStatusSuccess, response.Status)
```

### Integration Tests
Test with real RabbitMQ:
```go
// Publish command
publisher.PublishCommand(ctx, CommandCreateBO, tenantID, userID, req)

// Consume and handle
consumer.Subscribe(ctx, "command.bo.*")
// Wait for response in reply queue

// Verify event was published
// Verify response is correct
```

### Contract Tests
Test command/response contracts:
- Command serialization/deserialization
- Response correlation ID matching
- Timeout handling
- Error response structure

## Migration Path

### Phase 1: Commands for BO CRUD ✅ COMPLETE
- Implement command publisher/consumer
- Refactor handlers to use commands
- Implement BO command handlers
- Maintain HTTP API compatibility

### Phase 2: Commands for Instances (NEXT)
- Add instance command handlers
- Implement correlation ID tracking
- Add response timeout handling

### Phase 3: Extract to Microservices
- Move BO handlers to separate process
- Deploy as independent service
- Add service discovery
- Implement circuit breakers

### Phase 4: Advanced Patterns
- CQRS (separate read/write models)
- Event replay for testing
- Saga pattern for long-running workflows
- Event sourcing audit logs

## Troubleshooting

### Commands not being processed
1. Check RabbitMQ is running: `docker-compose ps`
2. Verify command consumer is subscribed: Check logs for "listening for"
3. Check command handlers are registered: `consumer.RegisterHandler()`
4. Verify correlation ID matches in response

### Timeouts waiting for response
1. Increase timeout from default 10 seconds
2. Check BO service is running and consuming commands
3. Check that response is being published to reply queue
4. Verify reply queue binding is correct

### Events not persisted
1. Check `semlayer.events` exchange is durable
2. Verify event queue has TTL and DLX configured
3. Check DeliveryMode is `amqp.Persistent` in EventPublisher
4. Look for errors in event consumer logs

## Performance Considerations

**Latency**: Command processing adds ~50-100ms compared to direct calls (includes message overhead)

**Throughput**: Message bus can handle thousands of commands/sec, limited by:
- RabbitMQ broker capacity
- Number of handler goroutines
- Database transaction time

**Optimization Tips**:
- Batch commands when possible
- Use multiple handler instances
- Optimize database queries in handlers
- Monitor correlation ID tracking overhead

## Next Steps

1. ✅ Command bus infrastructure (DONE)
2. ✅ BO command handlers (DONE)
3. ⏳ Instance command handlers (TO DO)
4. ⏳ Integration tests for command/response
5. ⏳ Extract BO service to separate container
6. ⏳ Add service discovery
7. ⏳ Implement circuit breaker pattern
