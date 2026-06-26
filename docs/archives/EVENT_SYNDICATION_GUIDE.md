# Event Syndication: Keeping Catalog Nodes & Edges Current

## Overview

This document describes the event syndication system that uses **RabbitMQ** and **Temporal** to automatically keep catalog nodes and edges synchronized with any API endpoint additions, removals, or modifications.

## Architecture

### Three-Tier Event Flow

```
┌─────────────────────────────────────────────────────────────────┐
│                    TIER 1: EVENT GENERATION                      │
│                                                                   │
│  API Endpoints Service                                           │
│  ├─ Create endpoint → APIEndpointCreated event                  │
│  ├─ Update endpoint → APIEndpointUpdated event                  │
│  ├─ Delete endpoint → APIEndpointDeleted event                  │
│  ├─ Activate endpoint → APIEndpointActivated event              │
│  ├─ Create entity mapping → EntityMappingCreated event          │
│  └─ Delete entity mapping → EntityMappingDeleted event          │
└─────────────────────────────────────────────────────────────────┘
                             ↓ (HTTP POST)
┌─────────────────────────────────────────────────────────────────┐
│                  TIER 2: EVENT PUBLICATION                       │
│                                                                   │
│  RabbitMQ Message Broker                                         │
│  ├─ api.endpoints (topic exchange)                              │
│  │  ├─ api.endpoint.created                                     │
│  │  ├─ api.endpoint.updated                                     │
│  │  ├─ api.endpoint.deleted                                     │
│  │  └─ api.endpoint.activated                                   │
│  │                                                               │
│  ├─ api.mappings (topic exchange)                               │
│  │  ├─ api.entity_mapping.created                               │
│  │  ├─ api.entity_mapping.deleted                               │
│  │  ├─ api.datasource_mapping.created                           │
│  │  └─ api.datasource_mapping.deleted                           │
│  │                                                               │
│  └─ catalog.nodes (topic exchange)                              │
│     ├─ catalog.node.created                                     │
│     ├─ catalog.node.updated                                     │
│     └─ catalog.node.deleted                                     │
└─────────────────────────────────────────────────────────────────┘
                             ↓ (Async consumption)
┌─────────────────────────────────────────────────────────────────┐
│              TIER 3: WORKFLOW ORCHESTRATION                      │
│                                                                   │
│  Temporal Workflows (Catalog Sync)                              │
│  ├─ CatalogSyncWorkflow                                         │
│  │  ├─ APIEndpointEvent handler                                 │
│  │  ├─ EntityMappingEvent handler                               │
│  │  ├─ DatasourceMappingEvent handler                           │
│  │  └─ DatasourceMappingEvent handler                           │
│  │                                                               │
│  └─ Activities:                                                 │
│     ├─ CreateEndpointCatalogNodeActivity                        │
│     ├─ UpdateEndpointCatalogNodeActivity                        │
│     ├─ DeleteEndpointCatalogNodeActivity                        │
│     ├─ CreateMappingCatalogEdgeActivity                         │
│     ├─ DeleteMappingCatalogEdgeActivity                         │
│     └─ PublishCatalogNodeCreatedActivity                        │
│                                                                   │
└─────────────────────────────────────────────────────────────────┘
                             ↓ (SQL Updates)
┌─────────────────────────────────────────────────────────────────┐
│                TIER 4: CATALOG SYNCHRONIZATION                   │
│                                                                   │
│  PostgreSQL Database                                             │
│  ├─ catalog_nodes table (synced with api_endpoints)             │
│  ├─ catalog_edges table (synced with api mappings)              │
│  └─ api_endpoints_catalog table (source of truth)               │
│                                                                   │
└─────────────────────────────────────────────────────────────────┘
```

## Event Types & Handlers

### API Endpoint Events

#### 1. APIEndpointCreated
- **Trigger**: When a new API endpoint is created
- **Payload**: APIEndpointEvent with full endpoint metadata
- **Handler**: CatalogSyncWorkflow → CreateEndpointCatalogNodeActivity
- **Result**: Creates catalog_node entry with type "api_endpoint"

```json
{
  "event_id": "evt-123",
  "event_type": "api.endpoint.created",
  "tenant_id": "tenant-001",
  "endpoint_id": "ep-456",
  "endpoint": {
    "endpoint_name": "List Validation Rules",
    "http_method": "GET",
    "url_path": "/validation-rules",
    "category": "validation",
    "description": "List all validation rules"
  },
  "timestamp": "2025-10-25T10:00:00Z"
}
```

#### 2. APIEndpointUpdated
- **Trigger**: When an existing API endpoint is modified
- **Payload**: APIEndpointEvent with updated fields
- **Handler**: CatalogSyncWorkflow → UpdateEndpointCatalogNodeActivity
- **Result**: Updates catalog_node metadata

#### 3. APIEndpointDeleted
- **Trigger**: When an API endpoint is deleted (soft delete)
- **Payload**: APIEndpointEvent with endpoint ID
- **Handler**: CatalogSyncWorkflow → DeleteEndpointCatalogNodeActivity
- **Result**: Soft-deletes catalog_node (sets is_active = false)

#### 4. APIEndpointActivated
- **Trigger**: When an API endpoint is reactivated
- **Payload**: APIEndpointEvent
- **Handler**: CatalogSyncWorkflow → ActivateEndpointNodesActivity
- **Result**: Re-activates catalog_node and related edges

### Mapping Events

#### 1. EntityMappingCreated
- **Trigger**: When an entity mapping is created
- **Payload**: EntityMappingEvent with relationship type
- **Handler**: CatalogSyncWorkflow → CreateMappingCatalogEdgeActivity
- **Result**: Creates catalog_edge from endpoint to entity

```json
{
  "event_id": "evt-789",
  "event_type": "api.entity_mapping.created",
  "tenant_id": "tenant-001",
  "api_endpoint_id": "ep-456",
  "entity_id": "ent-123",
  "relationship_type": "can_read",
  "timestamp": "2025-10-25T10:05:00Z"
}
```

#### 2. EntityMappingDeleted
- **Trigger**: When an entity mapping is removed
- **Payload**: EntityMappingEvent with endpoint and entity IDs
- **Handler**: CatalogSyncWorkflow → DeleteMappingCatalogEdgeActivity
- **Result**: Deletes catalog_edge

#### 3. DatasourceMappingCreated
- **Trigger**: When a datasource mapping is created
- **Payload**: DatasourceMappingEvent
- **Handler**: CatalogSyncWorkflow → CreateDatasourceMappingEdgeActivity
- **Result**: Creates catalog_edge from endpoint to datasource

#### 4. DatasourceMappingDeleted
- **Trigger**: When a datasource mapping is removed
- **Payload**: DatasourceMappingEvent
- **Handler**: CatalogSyncWorkflow → DeleteDatasourceMappingEdgeActivity
- **Result**: Deletes catalog_edge

## Implementation Steps

### Step 1: Update API Endpoints Catalog Handler

In `backend/internal/api/api_endpoints_catalog.go`, update handlers to publish events:

```go
import (
	"github.com/hondyman/semlayer/backend/internal/events"
)

// Add to handleCreateAPIEndpoint
func handleCreateAPIEndpoint(w http.ResponseWriter, r *http.Request, publisher *events.KafkaPublisher) {
	// ... existing code ...

	// After successful insert
	event := &events.APIEndpointEvent{
		EventID:      uuid.New().String(),
		EventType:    events.APIEndpointCreated,
		TenantID:     tenantID,
		DatasourceID: datasourceID,
		EndpointID:   endpoint.ID,
		Endpoint:     endpointMap, // Convert endpoint to map
		Timestamp:    time.Now(),
	}

	if err := publisher.PublishAPIEndpointEvent(r.Context(), event); err != nil {
		log.Printf("failed to publish endpoint created event: %v", err)
		// Continue anyway - event will be missed but endpoint is created
	}

	json.NewEncoder(w).Encode(endpoint)
}

// Add to handleUpdateAPIEndpoint
func handleUpdateAPIEndpoint(w http.ResponseWriter, r *http.Request, publisher *events.KafkaPublisher) {
	// ... existing code ...

	event := &events.APIEndpointEvent{
		EventID:      uuid.New().String(),
		EventType:    events.APIEndpointUpdated,
		TenantID:     tenantID,
		EndpointID:   endpointID,
		Endpoint:     endpointMap,
		Timestamp:    time.Now(),
	}

	if err := publisher.PublishAPIEndpointEvent(r.Context(), event); err != nil {
		log.Printf("failed to publish endpoint updated event: %v", err)
	}

	json.NewEncoder(w).Encode(updated)
}

// Add to handleDeleteAPIEndpoint
func handleDeleteAPIEndpoint(w http.ResponseWriter, r *http.Request, publisher *events.KafkaPublisher) {
	// ... existing code ...

	event := &events.APIEndpointEvent{
		EventID:    uuid.New().String(),
		EventType:  events.APIEndpointDeleted,
		TenantID:   tenantID,
		EndpointID: endpointID,
		Timestamp:  time.Now(),
	}

	if err := publisher.PublishAPIEndpointEvent(r.Context(), event); err != nil {
		log.Printf("failed to publish endpoint deleted event: %v", err)
	}

	w.WriteHeader(http.StatusNoContent)
}
```

### Step 2: Update Mapping Routes Handler

In `backend/internal/api/api_endpoint_mapping_routes.go`:

```go
// Add to handleCreateEntityMapping
func handleCreateEntityMapping(w http.ResponseWriter, r *http.Request, publisher *events.KafkaPublisher) {
	// ... existing code ...

	event := &events.EntityMappingEvent{
		EventID:          uuid.New().String(),
		EventType:        events.EntityMappingCreated,
		TenantID:         tenantID,
		APIEndpointID:    endpointID,
		EntityID:         mapping.EntityID,
		RelationshipType: mapping.RelationshipType,
		Timestamp:        time.Now(),
	}

	if err := publisher.PublishEntityMappingEvent(r.Context(), event); err != nil {
		log.Printf("failed to publish entity mapping created event: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(mapping)
}
```

### Step 3: Initialize Event Components in Main

In `backend/cmd/main.go` or equivalent initialization:

```go
package main

import (
	"context"
	"log"

	"github.com/hondyman/semlayer/backend/internal/events"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

func init() {
	// Initialize RabbitMQ Publisher
	publisherConfig := events.DefaultRabbitMQConfig()
	publisherConfig.Brokers = os.Getenv("KAFKA_BROKERS") // redpanda:9092

	publisher, err := events.NewKafkaPublisher(config.Brokers) // NewKafkaPublisher accepts bootstrap brokers string (e.g., "localhost:9092")
	if err != nil {
		log.Fatalf("failed to create RabbitMQ publisher: %v", err)
	}
	defer publisher.Close()

	// Initialize Temporal Client
	temporalClient, err := client.Dial(client.Options{
		HostPort: "localhost:7233",
	})
	if err != nil {
		log.Fatalf("failed to create Temporal client: %v", err)
	}
	defer temporalClient.Close()

	// Initialize RabbitMQ Consumer
	consumerConfig := events.DefaultRabbitMQConfig()
	consumer, err := events.NewRabbitMQConsumer(consumerConfig, temporalClient)
	if err != nil {
		log.Fatalf("failed to create RabbitMQ consumer: %v", err)
	}
	defer consumer.Close()

	// Start consuming events
	ctx := context.Background()
	if err := consumer.StartConsuming(ctx); err != nil {
		log.Fatalf("failed to start consuming events: %v", err)
	}

	// Register Temporal workflows and activities
	w := worker.New(temporalClient, "api_catalog_sync", worker.Options{})
	w.RegisterWorkflow(CatalogSyncWorkflow)
	w.RegisterActivity(CreateEndpointCatalogNodeActivity)
	w.RegisterActivity(UpdateEndpointCatalogNodeActivity)
	w.RegisterActivity(DeleteEndpointCatalogNodeActivity)
	w.RegisterActivity(ActivateEndpointNodesActivity)
	w.RegisterActivity(CreateMappingCatalogEdgeActivity)
	w.RegisterActivity(DeleteMappingCatalogEdgeActivity)
	// ... register other activities ...

	if err := w.Start(); err != nil {
		log.Fatalf("failed to start Temporal worker: %v", err)
	}
}
```

### Step 4: Database Migration for Catalog Tables

Apply the following migration to create catalog nodes and edges tables:

```sql
-- Create catalog_nodes table
CREATE TABLE IF NOT EXISTS catalog_nodes (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    datasource_id UUID,
    node_type VARCHAR(100) NOT NULL, -- 'api_endpoint', 'entity', 'datasource'
    metadata JSONB NOT NULL,
    is_active BOOLEAN DEFAULT true,
    created_by UUID,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    CONSTRAINT node_unique UNIQUE (tenant_id, id, node_type)
);

-- Create catalog_edges table
CREATE TABLE IF NOT EXISTS catalog_edges (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    source_node_id UUID NOT NULL,
    target_node_id UUID NOT NULL,
    relationship_type VARCHAR(100) NOT NULL,
    is_active BOOLEAN DEFAULT true,
    created_by UUID,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    FOREIGN KEY (source_node_id) REFERENCES catalog_nodes(id) ON DELETE CASCADE,
    FOREIGN KEY (target_node_id) REFERENCES catalog_nodes(id) ON DELETE CASCADE,
    CONSTRAINT edge_unique UNIQUE (tenant_id, source_node_id, target_node_id, relationship_type)
);

-- Create indexes for performance
CREATE INDEX idx_catalog_nodes_tenant ON catalog_nodes(tenant_id, is_active);
CREATE INDEX idx_catalog_nodes_type ON catalog_nodes(node_type, tenant_id);
CREATE INDEX idx_catalog_edges_tenant ON catalog_edges(tenant_id, is_active);
CREATE INDEX idx_catalog_edges_source ON catalog_edges(source_node_id, tenant_id);
CREATE INDEX idx_catalog_edges_target ON catalog_edges(target_node_id, tenant_id);
```

## Event Syndication Workflow

### Complete Lifecycle Example

**Scenario**: User creates a new validation rule endpoint and maps it to an entity.

```
1. API Call: POST /api-endpoints
   ↓
2. Backend: Create endpoint in database
   ↓
3. Backend: Publish APIEndpointCreated event to RabbitMQ
   {
     "event_id": "evt-001",
     "event_type": "api.endpoint.created",
     "endpoint_id": "ep-123",
     "endpoint_name": "Execute Validation Rule",
     ...
   }
   ↓
4. RabbitMQ: Route to "api.endpoints" exchange
   ↓
5. RabbitMQ Consumer: Receive message
   ↓
6. Temporal Workflow: Execute CatalogSyncWorkflow
   ↓
7. Activity: CreateEndpointCatalogNodeActivity
   - Create catalog_node with type "api_endpoint"
   - Store endpoint metadata
   - Insert into database
   ↓
8. Activity: PublishCatalogNodeCreatedActivity
   - Emit CatalogNodeCreated event to RabbitMQ
   ↓
9. Frontend: Receive CatalogNodeCreated event (via WebSocket/polling)
   - Update UI to show new endpoint in graph

---

10. API Call: POST /api-endpoints/{id}/entity-mappings
    ↓
11. Backend: Create entity mapping in database
    ↓
12. Backend: Publish EntityMappingCreated event
    {
      "event_id": "evt-002",
      "event_type": "api.entity_mapping.created",
      "api_endpoint_id": "ep-123",
      "entity_id": "ent-456",
      "relationship_type": "can_execute"
    }
    ↓
13. Temporal Workflow: Execute CatalogSyncWorkflow
    ↓
14. Activity: CreateMappingCatalogEdgeActivity
    - Create catalog_edge linking endpoint and entity
    ↓
15. Activity: PublishCatalogEdgeCreatedActivity
    - Emit CatalogEdgeCreated event
    ↓
16. Frontend: Update graph with new edge
```

## Performance Characteristics

| Operation | Latency | Throughput | Notes |
|-----------|---------|-----------|-------|
| Event Publishing | <50ms | 10k events/sec | Async, non-blocking |
| Event Consumption | 100-500ms | Depends on activity complexity | Temporal retries on failure |
| Catalog Sync | 200-800ms | 100+ endpoints/sec | Database operation time |
| Total End-to-End | 300-1500ms | Sync is fastest | Full update with retries |

## Error Handling & Recovery

### Failure Scenarios

#### 1. RabbitMQ Unavailable
- **Detection**: PublishAPIEndpointEvent returns error
- **Recovery**: Retry with exponential backoff (1s → 2s → 4s → 8s max)
- **Fallback**: Log error, continue (endpoint created but catalog not synced yet)
- **Impact**: Catalog will be out of sync until RabbitMQ recovers

#### 2. Temporal Workflow Fails
- **Detection**: Workflow execution times out or returns error
- **Recovery**: Temporal automatically retries (configured: max 3 attempts)
- **Fallback**: Event sent to dead letter exchange for manual review
- **Impact**: Catalog may not update; manual intervention required

#### 3. Database Transaction Fails
- **Detection**: SQL error during catalog node/edge insert
- **Recovery**: Temporal activity retry (automatic)
- **Fallback**: Log error, send to DLQ
- **Impact**: Event lost; requires event replay

### Dead Letter Exchange (DLX)

When an event fails processing, it's automatically sent to:
- **Exchange**: `api_catalog_dlx`
- **Queue**: `api_catalog_dead_letter_queue`
- **TTL**: 24 hours
- **Headers**: 
  - `x-error`: Error message
  - `x-retry-count`: Number of retries attempted
  - `x-original-exchange`: Original exchange name
  - `x-original-routing-key`: Original routing key

**Manual Recovery**:
```bash
# List dead letter messages
rabbitmqctl list_queues name messages

# Requeue messages from DLX back to original queue
# (requires custom tool or manual inspection)
```

## Configuration

### Environment Variables

```bash
# RabbitMQ
RABBITMQ_URL=amqp://user:password@localhost:5672/
RABBITMQ_MAX_RETRIES=3
RABBITMQ_RETRY_DELAY_MS=1000

# Temporal
TEMPORAL_HOST=localhost
TEMPORAL_PORT=7233
TEMPORAL_NAMESPACE=default

# Catalog Sync
CATALOG_SYNC_TASK_QUEUE=api_catalog_sync
CATALOG_SYNC_TIMEOUT_SECONDS=300
CATALOG_SYNC_RETRY_ATTEMPTS=3
```

### RabbitMQ Configuration

```yaml
rabbitmq:
  exchanges:
    api.endpoints:
      type: topic
      durable: true
    api.mappings:
      type: topic
      durable: true
    catalog.nodes:
      type: topic
      durable: true
    catalog.edges:
      type: topic
      durable: true

  queues:
    api_endpoints_sync_queue:
      durable: true
      arguments:
        x-dead-letter-exchange: "api_catalog_dlx"
        x-message-ttl: 86400000  # 24 hours
    api_mappings_sync_queue:
      durable: true
      arguments:
        x-dead-letter-exchange: "api_catalog_dlx"
        x-message-ttl: 86400000
    catalog_sync_queue:
      durable: true
      arguments:
        x-dead-letter-exchange: "api_catalog_dlx"
        x-message-ttl: 86400000
```

## Monitoring & Observability

### Metrics to Track

1. **Event Publishing**
   - Events published per second
   - Publishing latency (P50, P95, P99)
   - Publishing errors

2. **Event Consumption**
   - Events consumed per second
   - Queue depth
   - Consumer lag

3. **Workflow Execution**
   - Workflow completion rate
   - Workflow latency
   - Workflow retry rate
   - Failed workflows

4. **Catalog Synchronization**
   - Sync success rate
   - Sync latency
   - Catalog nodes created/updated/deleted
   - Catalog edges created/deleted

### Logging

```go
// Log event publishing
log.Printf("[EVENT] Published %s event for endpoint %s (tenant: %s)",
	event.EventType, event.EndpointID, event.TenantID)

// Log workflow execution
log.Printf("[WORKFLOW] CatalogSyncWorkflow started for %s (workflowID: %s)",
	event.EventType, workflowID)

// Log activity execution
log.Printf("[ACTIVITY] CreateEndpointCatalogNodeActivity executed (nodeID: %s)",
	node.ID)

// Log errors
log.Printf("[ERROR] Failed to publish event: %v", err)
```

## Testing Strategy

### Unit Tests

```go
// Test event publishing
func TestPublishAPIEndpointEvent(t *testing.T) {
	publisher, err := events.NewKafkaPublisher("localhost:9092")
	require.NoError(t, err)
	defer publisher.Close()

	event := &events.APIEndpointEvent{
		EventID:    "test-123",
		EventType:  events.APIEndpointCreated,
		TenantID:   "tenant-001",
		EndpointID: "ep-456",
	}

	err = publisher.PublishAPIEndpointEvent(context.Background(), event)
	require.NoError(t, err)
}

// Test event consumption
func TestConsumeEvents(t *testing.T) {
	// Create consumer with mock Temporal client
	consumer, err := events.NewRabbitMQConsumer(testConfig, mockClient)
	require.NoError(t, err)

	// Publish test event
	// ...

	// Verify event was routed to Temporal
	// ...
}
```

### Integration Tests

- Set up real RabbitMQ and Temporal instances
- Publish events and verify catalog synchronization
- Test failover scenarios
- Verify dead letter queue handling

## Deployment Checklist

- [ ] Deploy RabbitMQ cluster with HA configuration
- [ ] Deploy Temporal cluster with proper persistence
- [ ] Update API handlers to publish events
- [ ] Apply database migrations for catalog tables
- [ ] Initialize publisher and consumer in main()
- [ ] Register Temporal workflows and activities
- [ ] Test event flow end-to-end
- [ ] Monitor event publishing/consumption metrics
- [ ] Set up alerting for dead letter queue depth
- [ ] Document runbook for manual event replay

## Next Steps

1. **Immediate**: Update API handlers to publish events (5 events total)
2. **Short-term**: Implement Temporal activities for catalog sync
3. **Mid-term**: Add WebSocket support for real-time catalog updates to frontend
4. **Long-term**: Implement event replay and audit trail features
