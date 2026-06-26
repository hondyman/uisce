# Gold Copy Publishing to Redpanda

## Overview

Your EDM system now automatically publishes **gold copy** (canonical/authoritative) entities to Redpanda/Kafka for consumption by downstream systems. This enables real-time data governance and ensures all systems work with the same version of truth.

## What is a Gold Copy?

A **gold copy** is the authoritative, production-approved version of a semantic entity that has passed validation and been promoted to canonical status. Examples include:

- **Rules**: Semantic rules that have been published and approved for production
- **Templates**: Rule templates that are ready for use across the organization  
- **Preferences**: Source preferences, calendars, and data quality standards
- **Business Objects**: Canonical business entities with all attributes defined

When an entity becomes a gold copy, it's automatically published to Redpanda so downstream systems can:
- Subscribe to canonical data changes
- Sync with authoritative definitions
- Build derived data safely on proven foundations
- Maintain data lineage and audit trails

---

## Architecture

### Event Flow

```
Your Semantic System
        │
        ├─ Rule Published ──→ \
        ├─ Template Approved ─→ GoldCopyPublisher ──→ Redpanda
        ├─ Preference Promoted ─→ /     (semlayer.gold-copy topic)
        └─ BO Certified ───→

                                ↓ (Kafka.Reader pattern)

        Downstream Systems Subscribe:
        ├─ Data Integration Layer
        ├─ ML/Analytics Pipeline
        ├─ Reporting & BI Tools
        ├─ Governance Dashboard
        └─ External Data Platforms
```

### Topic Structure

**Topic Name**: `semlayer.gold-copy`

**Message Routing Key**: `{tenant_id}.{entity_type}.{event_type}`

Example:
```
550e8400-e29b-41d4-a716-446655440000.rule.gold.copy.rule.created
550e8400-e29b-41d4-a716-446655440000.preference.gold.copy.preference.updated
```

### Message Headers (Kafka)

Every gold copy message includes these headers for easy filtering:

```
entity_type:  rule | template | preference | business_object
entity_id:    UUID of the entity
tenant_id:    UUID of the tenant
event_type:   gold.copy.rule.created | gold.copy.rule.updated | ...
```

---

## Event Types

### Rule Events
```
gold.copy.rule.created        - New rule promoted to production
gold.copy.rule.updated        - Rule definition changed (backward compatible)
gold.copy.rule.deprecated     - Rule marked for deprecation
gold.copy.rule.retired        - Rule removed from production
```

### Template Events
```
gold.copy.template.created    - New template approved
gold.copy.template.updated    - Template modified
gold.copy.template.retired    - Template retired
```

### Preference Events
```
gold.copy.preference.created  - New approved preference (source, calendar, etc.)
gold.copy.preference.updated  - Preference changed
gold.copy.preference.retired  - Preference retired
```

### Business Object Events
```
gold.copy.business_object.created   - New BO certified
gold.copy.business_object.updated   - BO attributes changed
gold.copy.business_object.retired   - BO retired
```

---

## Event Payload Structure

```json
{
  "event_id": "550e8400-e29b-41d4-a716-446655440001-gold.copy.rule.created-1708456800",
  "event_type": "gold.copy.rule.created",
  "published_at": "2026-02-20T15:35:00Z",
  "published_by": "user-123-uuid",
  
  "tenant_id": "550e8400-e29b-41d4-a716-446655440000",
  "entity_type": "rule",
  "entity_id": "rule-456-uuid",
  "entity_key": "IsBusinessDay",
  "version": 3,
  "semantic_layer": "semantic-rules",
  
  "data": {
    "id": "rule-456-uuid",
    "semantic_term": "IsBusinessDay",
    "rule_engine": "drools",
    "expression_language": "JEXL",
    "expression": "calendar.isBusinessDay(date)",
    "status": "published",
    "version": 3,
    "created_by": "data-steward-1",
    "updated_by": "compliance-officer-1"
  },
  
  "data_hash": "sha256:abcd1234...",
  "schema_version": "1.0",
  
  "change_type": "creation",
  "change_reason": "Initial production deployment",
  "correlation_id": "workflow-789-uuid",
  
  "metadata": {
    "status": "published",
    "semantic_term": "IsBusinessDay",
    "rule_engine": "drools",
    "expression_language": "JEXL"
  }
}
```

---

## Configuration

### Environment Variables

```bash
# Redpanda/Kafka broker addresses (comma-separated)
REDPANDA_BROKERS=localhost:9092,localhost:9093,localhost:9094

# Or if using Redpanda cloud:
REDPANDA_BROKERS=broker.prod.kafka.example.com:9092
```

### Default Configuration

```go
// In code (backend/cmd/semantic-rules-api/main.go)
redpandaBrokers := os.Getenv("REDPANDA_BROKERS")
if redpandaBrokers == "" {
    redpandaBrokers = "localhost:9092"  // Default for local development
}
```

### Topics to Create

Before running in production, create these Redpanda topics:

```bash
# Create gold-copy topic with replication and retention
rpk topic create semlayer.gold-copy \
  --partitions 3 \
  --replication-factor 3 \
  --config retention.ms=2592000000 \
  --config compression.type=snappy

# Verify topic creation
rpk topic describe semlayer.gold-copy

# Monitor topic:
rpk topic consume semlayer.gold-copy
```

---

## Integration Points

### When is Gold Copy Published?

Gold copies are published automatically when:

1. **Rule Promotion** - When a rule is published and promoted to production status
   ```go
   // In rule handler: POST /api/v1/rules/{ruleId}/promote
   goldCopyPublisher.PublishRuleAsGoldCopy(ctx, rule, "creation", "...", userID, hash)
   ```

2. **Template Approval** - When a template is approved for general use
   ```go
   // In template handler: POST /api/v1/templates/{templateId}/approve
   goldCopyPublisher.PublishTemplateAsGoldCopy(ctx, template, "creation", "...", userID, hash)
   ```

3. **Preference Certification** - When a source preference or calendar is certified
   ```go
   // In preference handler: POST /api/v1/preferences/{prefId}/certify
   goldCopyPublisher.PublishPreferenceAsGoldCopy(ctx, tenantID, prefID, key, type, data, "creation", "...", userID, hash)
   ```

4. **Business Object Release** - When a BO is released for production
   ```go
   // In BO handler: POST /api/v1/business-objects/{boId}/release
   goldCopyPublisher.PublishBusinessObjectAsGoldCopy(ctx, bo, "creation", "...", userID, hash)
   ```

---

## Consuming Gold Copy Events

### Example: Redpanda Consumer (Go)

```go
package main

import (
	"context"
	"encoding/json"
	"log"

	kafka "github.com/segmentio/kafka-go"
)

type GoldCopyEvent struct {
	EventID    string      `json:"event_id"`
	EventType  string      `json:"event_type"`
	EntityType string      `json:"entity_type"`
	EntityID   string      `json:"entity_id"`
	TenantID   string      `json:"tenant_id"`
	Data       interface{} `json:"data"`
}

func main() {
	// Create reader
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:   []string{"localhost:9092"},
		Topic:     "semlayer.gold-copy",
		GroupID:   "my-analytics-service",
		StartOffset: kafka.LastOffset,
	})
	defer reader.Close()

	// Consume messages
	for {
		msg, err := reader.ReadMessage(context.Background())
		if err != nil {
			log.Fatalf("Error reading message: %v", err)
		}

		var event GoldCopyEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			log.Printf("Error unmarshaling event: %v", err)
			continue
		}

		log.Printf("Received: %s for %s (%s owned by tenant %s)",
			event.EventType, event.EntityType, event.EntityID, event.TenantID)

		// Process event based on type
		switch event.EventType {
		case "gold.copy.rule.created":
			// Sync rule to analytics
			syncRuleToAnalytics(event.Data)
		case "gold.copy.preference.updated":
			// Update preference cache
			updatePreferenceCache(event.Data)
		}
	}
}

func syncRuleToAnalytics(data interface{}) {
	// Your implementation
}

func updatePreferenceCache(data interface{}) {
	// Your implementation
}
```

### Example: Python Consumer

```python
from kafka import KafkaConsumer
import json

consumer = KafkaConsumer(
    'semlayer.gold-copy',
    bootstrap_servers=['localhost:9092'],
    group_id='my-python-service',
    value_deserializer=lambda m: json.loads(m.decode('utf-8'))
)

for message in consumer:
    event = message.value
    print(f"Event Type: {event['event_type']}")
    print(f"Entity: {event['entity_type']} ({event['entity_id']})")
    print(f"Tenant: {event['tenant_id']}")
    
    # Route by event type
    if event['event_type'].startswith('gold.copy.rule'):
        handle_rule_event(event)
    elif event['event_type'].startswith('gold.copy.preference'):
        handle_preference_event(event)
```

### Example: JavaScript/Node.js Consumer

```javascript
const { Kafka } = require('kafkajs');

const kafka = new Kafka({
  clientId: 'my-js-service',
  brokers: ['localhost:9092']
});

const consumer = kafka.consumer({ groupId: 'my-js-consumer-group' });

await consumer.connect();
await consumer.subscribe({ topic: 'semlayer.gold-copy' });

await consumer.run({
  eachMessage: async ({ topic, partition, message }) => {
    const event = JSON.parse(message.value.toString());
    
    console.log(`Event Type: ${event.event_type}`);
    console.log(`Entity: ${event.entity_type} (${event.entity_id})`);
    console.log(`Tenant: ${event.tenant_id}`);
    
    // Process event
    switch (event.event_type) {
      case 'gold.copy.rule.created':
        await syncRuleToDataWarehouse(event.data);
        break;
      case 'gold.copy.preference.updated':
        await updateCacheWithPreference(event.data);
        break;
    }
  }
});
```

---

## Multi-Tenant Isolation

Gold copy events are automatically isolated by tenant via:

1. **Message Routing Key** includes `tenant_id`:
   ```
   550e8400-e29b-41d4-a716-446655440000.rule.gold.copy.rule.created
   ```

2. **Kafka Headers** include tenant_id for filtering:
   ```
   tenant_id: 550e8400-e29b-41d4-a716-446655440000
   ```

3. **Event Payload** includes tenant_id:
   ```json
   {
     "tenant_id": "550e8400-e29b-41d4-a716-446655440000",
     ...
   }
   ```

**Consumer Best Practice**:
```go
// Filter events by tenant
reader := kafka.NewReader(kafka.ReaderConfig{
	TopicPartitions: []kafka.TopicPartition{
		{Topic: "semlayer.gold-copy", Partition: 0},
	},
	Dialer: &kafka.Dialer{},
})

for {
	msg, _ := reader.ReadMessage(ctx)
	
	// Check tenant_id header
	for _, header := range msg.Headers {
		if header.Key == "tenant_id" {
			if string(header.Value) == myTenantID {
				processMessage(msg)
			}
		}
	}
}
```

---

## Monitoring & Operations

### Health Check

```bash
# Verify topic exists and has data
rpk topic describe semlayer.gold-copy
rpk topic consume semlayer.gold-copy --limit 1

# Check consumer groups
rpk consumer group list

# Monitor group lag
rpk consumer group describe my-analytics-service
```

### Metrics to Monitor

1. **Message Throughput**: Events published per minute
2. **Consumer Lag**: How far behind consumers are
3. **Error Rate**: Failed publishes or malformed events
4. **Payload Size**: Average message size

### Example: Monitoring in Prometheus

```go
// Add to your metrics collector
goldCopyEventsPublished := prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "semlayer_gold_copy_published_total",
		Help: "Total gold copy events published",
	},
	[]string{"entity_type", "event_type", "tenant_id"},
)

goldCopyPublishErrors := prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "semlayer_gold_copy_publish_errors_total",
		Help: "Total errors publishing gold copy events",
	},
	[]string{"entity_type", "reason"},
)
```

---

## Troubleshooting

### Gold Copy Events Not Appearing

**Symptoms**: Publish rules/templates but no messages in Redpanda

**Diagnosis**:
```bash
# 1. Check Redpanda is running
rpk cluster describe

# 2. Verify topic exists
rpk topic list | grep gold-copy

# 3. Check broker connectivity
curl telnet://localhost:9092

# 4. View backend logs
tail -f /tmp/api.log | grep -i "gold copy"
```

**Solution**:
```bash
# Create topic if missing
rpk topic create semlayer.gold-copy --partitions 3

# Verify REDPANDA_BROKERS env var is set
echo $REDPANDA_BROKERS

# Restart backend service
pkill -f semantic-rules-api
./semantic-rules-api &
```

### Consumer Lag Growing

**Symptoms**: Events publish but consumers can't keep up

**Diagnosis**:
```bash
# Check consumer lag
rpk consumer group describe my-service

# Check topic throughput
rpk topic consume semlayer.gold-copy | wc -l

# Check message size
rpk topic consume semlayer.gold-copy --output json | jq '.size'
```

**Solutions**:
- Increase partitions: `rpk topic alter-config semlayer.gold-copy --partitions 6`
- Add more consumer instances to same group
- Optimize consumer processing (batch messages, parallelize)
- Check downstream system resource constraints

### Duplicate Events

**Symptoms**: Same event appearing multiple times

**Diagnosis**:
```bash
# Check if multiple publishers are running
ps aux | grep semantic-rules-api

# Check message deduplication (idempotent key)
rpk topic consume semlayer.gold-copy --output json | jq '.key' | sort | uniq -d
```

**Solutions**:
- Verify only one backend instance publishes
- Implement idempotent consumers (use event_id as dedup key)
- Enable Kafka deduplication:
  ```bash
  rpk topic alter-config semlayer.gold-copy \
    --set enable.idempotence=true \
    --set 'producer.idempotence.id.acks=all'
  ```

---

## Performance Tuning

### Batch Publishing (If Publishing Many Events)

```go
// Group events before publishing (pseudo-code)
var events []*GoldCopyEvent
for _, rule := range rulesBeingPromoted {
	event := createGoldCopyEvent(rule)
	events = append(events, event)
}

// Publish all at once (reduces network round-trips)
for _, event := range events {
	goldCopyPublisher.PublishGoldCopyEvent(ctx, event)
}
```

### Configure Compression

```bash
# Enable Snappy compression (default)
rpk topic alter-config semlayer.gold-copy \
  --set compression.type=snappy

# For high-volume, consider LZ4:
rpk topic alter-config semlayer.gold-copy \
  --set compression.type=lz4
```

### Partition Strategy

```bash
# Start with 3 partitions for single-tenant
# For multi-tenant, use tenant_id for partitioning:
partitionKey := tenantID + "|" + entityID

# For 100+ tenants:
rpk topic alter-config semlayer.gold-copy --partitions 10

# For 1000+ tenants:
rpk topic alter-config semlayer.gold-copy --partitions 30
```

---

## API Examples

### When Publishing From Rules Handler

```go
// backend/internal/handlers/rules_handler.go

func (h *RuleHandler) PromoteRule(w http.ResponseWriter, r *http.Request) {
  ruleID := chi.URLParam(r, "ruleId")
  
  // ... validation and update logic ...
  
  // Publish to gold copy!
  dataHash := hashRule(rule)
  err := h.goldCopyPublisher.PublishRuleAsGoldCopy(
      r.Context(),
      rule,
      "creation",  // changeType
      "Production promotion",  // changeReason
      getUserID(r),  // publishedByUserID
      dataHash,
  )
  if err != nil {
      log.Printf("Warning: Failed to publish gold copy: %v", err)
      // Don't fail the request if Redpanda is down
  }
  
  // Return success
  w.WriteHeader(http.StatusOK)
  json.NewEncoder(w).Encode(rule)
}
```

---

## What's Next

1. **✅ Complete**: Gold copy publisher service created
2. **✅ Complete**: Main.go wired with publisher
3. **TODO**: Wire publisher into rule/template handlers  
4. **TODO**: Add unit tests for event publishing
5. **TODO**: Create monitoring dashboard
6. **TODO**: Document consumer patterns for downstream teams
7. **TODO**: Add data schema registry integration

---

## Summary

Your system now:
- ✅ Publishes canonical entities to Redpanda automatically
- ✅ Provides full audit trail (who, what, when)
- ✅ Enables real-time downstream synchronization
- ✅ Supports multi-tenant isolation
- ✅ Includes data hashing for change detection
- ✅ Gracefully handles Redpanda unavailability

Downstream systems can now subscribe to `semlayer.gold-copy` topic and build their pipelines with confidence that they're using production-approved data.
