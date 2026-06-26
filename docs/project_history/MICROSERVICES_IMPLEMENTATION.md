# Microservices Implementation Guide

## Quick Start: Getting Commands Running

### 1. Initialize Command Bus in Main

```go
// In your backend/cmd/server/main.go

// Create command publisher (for API Gateway layer)
commandPublisher, err := services.NewCommandPublisher(rabbitmqURL)
if err != nil {
    log.Fatalf("Failed to create command publisher: %v", err)
}
defer commandPublisher.Close()

// Create command consumer (for BO Microservice)
commandConsumer, err := services.NewCommandConsumer(rabbitmqURL, "bo-service")
if err != nil {
    log.Fatalf("Failed to create command consumer: %v", err)
}
defer commandConsumer.Close()

// Create BO command handler
boCommandHandler := services.NewBOCommandHandler(boService, eventPublisher)

// Register handlers for each command type
commandConsumer.RegisterHandler(services.CommandCreateBO, boCommandHandler.HandleCreateBO)
commandConsumer.RegisterHandler(services.CommandUpdateBO, boCommandHandler.HandleUpdateBO)
commandConsumer.RegisterHandler(services.CommandDeleteBO, boCommandHandler.HandleDeleteBO)
commandConsumer.RegisterHandler(services.CommandCloneBO, boCommandHandler.HandleCloneBO)

// Start consuming commands
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

go func() {
    if err := commandConsumer.Subscribe(ctx, "command.bo.*"); err != nil {
        log.Printf("Command consumer error: %v", err)
    }
}()

// Update handler to use command bus
boHandler := handlers.NewBusinessObjectHandler(boService, eventPublisher, commandPublisher)
```

### 2. Environment Configuration

Add to `config.yaml`:

```yaml
rabbitmq:
  url: "amqp://guest:guest@localhost:5672/"
  # Optional: disable for testing
  enabled: true
```

Or use environment variable:

```bash
export RABBITMQ_URL="amqp://guest:guest@localhost:5672/"
```

### 3. RabbitMQ Docker Setup

Already configured in `docker-compose.yml`. Ensure these services are running:

```bash
docker-compose up -d rabbitmq postgres
```

### 4. Test Command Execution

#### Option A: Via HTTP API

```bash
# Create BO via command bus
curl -X POST http://localhost:8080/api/business-objects \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: tenant-123" \
  -H "X-User-ID: user-456" \
  -d '{
    "key": "customer",
    "displayName": "Customer",
    "displayNamePl": "Customers",
    "category": "people"
  }'

# Expected response (after command processed):
# 201 Created
# { "id": "...", "key": "customer", ... }
```

#### Option B: Direct Command Publishing

```go
// In tests or CLI tools
publisher, _ := services.NewCommandPublisher(rabbitmqURL)

correlationID, err := publisher.PublishCommand(
    ctx,
    services.CommandCreateBO,
    "tenant-123",
    "user-456",
    map[string]interface{}{
        "key": "product",
        "displayName": "Product",
    },
)

log.Printf("Command published with correlation ID: %s", correlationID)
```

### 5. Monitoring Commands

#### Check RabbitMQ Management UI

```
http://localhost:15672
Username: guest
Password: guest
```

Look for:
- **Exchanges**: `semlayer.commands`, `semlayer.events`, `semlayer.replies`
- **Queues**: `bo-service-commands`, `bo-service-events`
- **Message rates**: Should see publish/consume activity

#### Check Logs

```bash
# Watch command processing
docker-compose logs -f backend

# Look for:
# ✅ Command published: command.bo.create
# ⚙️  Executing command: command.bo.create
# ✅ Command completed: command.bo.create
```

## Architecture in Current Code

### API Gateway Layer (HTTP Handler)

**File**: `backend/internal/handlers/businessobject_handler.go`

```go
type BusinessObjectHandler struct {
    boService      *services.BusinessObjectService  // Fallback for direct calls
    eventPublisher *services.EventPublisher         // Publish events
    commandBus     *services.CommandPublisher       // Command bus
    enabled        bool                             // Is command bus active?
}

// When command bus is enabled:
// 1. Handler publishes command → semlayer.commands
// 2. Handler creates temp reply queue
// 3. Handler waits for response (10s timeout)
// 4. Handler returns response to client

// When command bus is disabled:
// 1. Handler calls service directly
// 2. Handler publishes event if enabled
// 3. Monolith mode (backward compatible)
```

### Microservice Layer (Command Consumer)

**File**: `backend/internal/services/bo_command_handler.go`

```go
type BOCommandHandler struct {
    boService      *services.BusinessObjectService  // Business logic
    eventPublisher *services.EventPublisher         // Publish events
}

// Implements handlers for each command:
// - HandleCreateBO()    → calls service.CreateBusinessObject()
// - HandleUpdateBO()    → calls service.UpdateBusinessObject()
// - HandleDeleteBO()    → calls service.DeleteBusinessObject()
// - HandleCloneBO()     → calls service.CloneBusinessObject()

// Each handler:
// 1. Extracts data from command
// 2. Calls service to execute logic
// 3. Publishes event on success
// 4. Returns response with status
```

### Message Bus Layer

**File**: `backend/internal/services/command_bus.go`

```go
// CommandPublisher - used by API Gateway
type CommandPublisher struct {
    channel         *amqp.Channel
    commandExchange string         // "semlayer.commands" (topic, transient)
    replyExchange   string         // "semlayer.replies" (direct, transient)
}

// CommandConsumer - used by BO Microservice
type CommandConsumer struct {
    channel         *amqp.Channel
    queue           string                              // "bo-service-commands"
    handlers        map[CommandType]CommandHandler     // Registered handlers
}

// Both implement graceful shutdown
// Both handle connection failures gracefully
```

## Request Flow Example

### Sequence: Create Business Object

```
1. CLIENT sends HTTP POST
   POST /api/business-objects
   Header: X-Tenant-ID: tenant-123
   Body: { key: "customer", displayName: "Customer" }

2. API GATEWAY (BusinessObjectHandler)
   ├─ Validates headers
   ├─ Deserializes request
   ├─ Publishes Command to semlayer.commands
   │  {
   │    id: "cmd-abc123",
   │    type: "command.bo.create",
   │    correlation_id: "corr-xyz789",
   │    tenant_id: "tenant-123",
   │    user_id: "user-456",
   │    data: { key, displayName, ... },
   │    timestamp: now()
   │  }
   ├─ Creates temp reply queue (auto-delete)
   ├─ Binds to semlayer.replies with routing key = corr-xyz789
   └─ Waits for response (10s timeout)

3. COMMAND CONSUMER (in BO Service)
   ├─ Receives command from queue
   ├─ Looks up handler for command.bo.create
   ├─ Calls BOCommandHandler.HandleCreateBO()

4. COMMAND HANDLER (Business Logic Layer)
   ├─ Extracts data from command
   ├─ Calls boService.CreateBusinessObject()
   │  (Database insert, validation, etc.)
   ├─ Gets back: BO { id, key, displayName, ... }
   ├─ Publishes Event to semlayer.events
   │  {
   │    id: "evt-def456",
   │    type: "event.bo.created",
   │    correlation_id: "corr-xyz789",  ← Links back to command
   │    entity_type: "business_object",
   │    entity_id: "bo-123",
   │    data: { full BO object },
   │    ...
   │  }
   └─ Creates response: CommandResponse { status: "success", data: BO }

5. RESPONSE PUBLISHER (BO Service)
   ├─ Marshals CommandResponse to JSON
   ├─ Publishes to semlayer.replies
   │  {
   │    correlation_id: "corr-xyz789",
   │    status: "success",
   │    data: { BO object },
   │    ...
   │  }

6. API GATEWAY (waiting in handler)
   ├─ Receives message on reply queue
   ├─ Unmarshals CommandResponse
   ├─ Status is "success", extracts data
   └─ Returns to client

7. CLIENT receives HTTP 201 Created
   {
     "id": "bo-123",
     "key": "customer",
     "displayName": "Customer",
     ...
   }
```

## Extending to Other Commands

To add a new command type (e.g., `CommandUpdateField`):

### 1. Add Command Type

```go
// In event_publisher.go
const (
    CommandUpdateBO CommandType = "command.bo.update"
    CommandUpdateField CommandType = "command.bo.update-field"  // NEW
)
```

### 2. Add Command Constant to Handler

```go
// In businessobject_handler.go
correlationID, err := h.commandBus.PublishCommand(
    r.Context(),
    services.CommandUpdateField,  // NEW
    tenantID,
    userID,
    updateData,
)
```

### 3. Implement Command Handler

```go
// In bo_command_handler.go
func (bch *BOCommandHandler) HandleUpdateField(ctx context.Context, command *Command) (*CommandResponse, error) {
    // Extract field name and new value from command
    // Call service to update field
    // Publish event
    // Return response
}
```

### 4. Register Handler

```go
// In main.go
commandConsumer.RegisterHandler(services.CommandUpdateField, boCommandHandler.HandleUpdateField)
```

### 5. Start Consuming

```go
commandConsumer.Subscribe(ctx, "command.bo.*")  // Already matches new command type
```

## Debugging Checklist

- [ ] RabbitMQ running: `docker-compose ps rabbitmq`
- [ ] Command bus enabled in config
- [ ] CommandPublisher created without errors
- [ ] CommandConsumer created without errors
- [ ] Handlers registered: Look for "✅ Handler registered" logs
- [ ] Consumer subscribed: Look for "📥 Command consumer listening" logs
- [ ] Command published: Look for "📤 Command published" logs
- [ ] Command executed: Look for "⚙️  Executing command" logs
- [ ] Response published: Check reply queue in RabbitMQ UI
- [ ] Handler received response: Look for response processing logs

## Gradual Rollout Strategy

### Stage 1: Deploy with Command Bus (Disabled by Default)
```bash
RABBITMQ_ENABLED=false  # Falls back to direct service calls
```

### Stage 2: Enable for One Command Type
```bash
RABBITMQ_ENABLED=true
# Only use for CommandCreateBO, others use direct calls
```

### Stage 3: Enable for All BO Commands
```bash
RABBITMQ_ENABLED=true
# All CRUD operations use command bus
```

### Stage 4: Extract to Microservice
Move BO service to separate container with its own command consumer

## Performance Tuning

### Message Queue Sizing
```go
// Increase channel prefetch for higher throughput
channel.Qos(
    prefetchCount,  // Per consumer
    prefetchSize,   // Bytes
    global,         // Apply to whole connection
)
```

### Timeout Tuning
```go
// In businessobject_handler.go
response, err := h.waitForCommandResponse(ctx, correlationID, 30*time.Second)  // Increase timeout
```

### Handler Concurrency
```go
// Multiple handler instances consume from same queue
go commandConsumer.Subscribe(ctx, "command.bo.*")  // 1st instance
go commandConsumer.Subscribe(ctx, "command.bo.*")  // 2nd instance (load balance)
```

## Next: Instance Commands

Same pattern for Business Object Instances:

```go
// In bo_command_handler.go (next phase)
func (bch *BOCommandHandler) HandleCreateInstance(ctx context.Context, command *Command) (*CommandResponse, error) {
    // Similar to HandleCreateBO but for instances
}

// Register in main.go
commandConsumer.RegisterHandler(services.CommandCreateInstance, boCommandHandler.HandleCreateInstance)
```
