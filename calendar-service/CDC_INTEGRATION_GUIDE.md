# 🔗 CDC Integration Implementation Guide

## Current Status

✅ **Cache layer**: Fully implemented (L1+L2 with Hasura fallback)  
✅ **Metrics**: Prometheus vectors tracking resolution by source  
✅ **Invalidation hooks**: Methods exist in `CDCProcessor` ready for integration  
❌ **CDC consumer loop**: Needs to be hooked into actual Redpanda consumer  

---

## Step-by-Step CDC Integration

### Step 1: Enable Redpanda Consumer in CDCProcessor.Run()

**File**: `internal/redpanda/consumer.go`

```go
import "github.com/twmb/franz-go/pkg/kgo"

func (p *CDCProcessor) Run(ctx context.Context) error {
	// 1. Create Kafka client
	client, err := kgo.NewClient(
		kgo.SeedBrokers(p.brokers...),
		kgo.ConsumeTopics(p.topics...),
		kgo.ConsumerGroup("calendar-cdc-group"),
		kgo.FetchMaxWait(500*time.Millisecond),
	)
	if err != nil {
		return fmt.Errorf("create kafka client: %w", err)
	}
	defer client.Close()

	p.logger.Info("CDC processor starting", "brokers", p.brokers, "topics", p.topics)

	// 2. Main consume loop
	for {
		select {
		case <-ctx.Done():
			p.logger.Info("CDC processor shutting down")
			return ctx.Err()
		default:
		}

		fetches := client.PollFetches(ctx)
		if fetches.IsClientClosed() {
			return nil
		}

		// 3. Handle errors
		fetches.EachError(func(topic string, partition int32, err error) {
			p.logger.WithError(err).Error("CDC fetch error",
				"topic", topic, "partition", partition,
			)
		})

		// 4. Process each record
		iter := fetches.RecordIter()
		for !iter.Done() {
			record := iter.Next()
			if err := p.processRecord(ctx, record); err != nil {
				p.logger.WithError(err).Warn("Failed to process CDC record",
					"topic", record.Topic,
					"partition", record.Partition,
					"offset", record.Offset,
				)
			}
		}
	}
}

func (p *CDCProcessor) processRecord(ctx context.Context, record *kgo.Record) error {
	// Parse Debezium CDC event
	var event CDCEvent
	if err := json.Unmarshal(record.Value, &event); err != nil {
		return fmt.Errorf("parse CDC event: %w", err)
	}

	p.logger.Debug("Processing CDC event",
		"table", event.Table,
		"operation", event.Op,
	)

	// === PROFILE_CALENDARS changes (mapping invalidation) ===
	if event.Table == "profile_calendars" && event.Op != "r" { // r=snapshot
		tenantID, calendarID := p.extractFieldsFromEvent(event, "tenant_id", "calendar_id")
		if tenantID != "" && calendarID != "" {
			go p.InvalidateProfileNameCacheForChange(ctx, tenantID, "", calendarID, event.Op)
		}
	}

	// === CALENDARS/PROFILES/BLACKOUTS changes (resolved cache invalidation) ===
	if (event.Table == "calendars" || event.Table == "schedule_profiles" || event.Table == "blackouts") && event.Op != "r" {
		tenantID := p.extractFieldFromEvent(event, "tenant_id")
		if tenantID != "" {
			go p.processCalendarChange(ctx, &CalendarChangeEvent{
				Entity:   event.Table,
				TenantID: tenantID,
				Region:   "us-east-1", // Extract from event if available
			})
		}
	}

	return nil
}

func (p *CDCProcessor) extractFieldFromEvent(event CDCEvent, field string) string {
	data := event.After
	if data == nil {
		data = event.Before
	}
	if data == nil {
		return ""
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return ""
	}

	if val, ok := m[field].(string); ok {
		return val
	}
	return ""
}

func (p *CDCProcessor) extractFieldsFromEvent(event CDCEvent, fields ...string) (string, string) {
	data := event.After
	if data == nil {
		data = event.Before
	}
	if data == nil {
		return "", ""
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return "", ""
	}

	vals := make([]string, len(fields))
	for i, field := range fields {
		if val, ok := m[field].(string); ok {
			vals[i] = val
		}
	}

	if len(vals) >= 2 {
		return vals[0], vals[1]
	}
	return "", ""
}
```

### Step 2: Wire Up in Main Server

**File**: `cmd/server/main.go`

```go
// After initializing cacheClient and availabilityChecker

cdcProcessor, err := redpanda.NewCDCProcessor(
	strings.Split(os.Getenv("REDPANDA_BROKERS"), ","),
	[]string{"cdc_calendar.public.profile_calendars", "cdc_calendar.public.calendars"},
	temporalClient,
	cacheClient,
	hasuraClient,
	availabilityChecker,
	logger,
)
if err != nil {
	logger.WithError(err).Fatal("Failed to create CDC processor")
}

// Start CDC processor in background
go func() {
	if err := cdcProcessor.Run(ctx); err != nil && err != context.Canceled {
		logger.WithError(err).Error("CDC processor failed")
	}
}()

// Ensure cleanup
defer cdcProcessor.Close()
```

### Step 3: Add CDC Metrics

**File**: `internal/metrics/metrics.go`

```go
var (
	// CDC Invalidation Metrics
	CDCInvalidationTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "calendar_cdc_invalidation_total",
			Help: "Total cache invalidations triggered by CDC",
		},
		[]string{"table", "operation"},
	)

	CDCInvalidationLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "calendar_cdc_invalidation_duration_seconds",
			Help:    "Duration of CDC-triggered cache invalidation",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"table", "operation"},
	)
)
```

### Step 4: Update InvalidateProfileNameCacheForChange to Record Metrics

**File**: `internal/redpanda/consumer.go`

```go
func (p *CDCProcessor) InvalidateProfileNameCacheForChange(ctx context.Context, tenantID, profileID, calendarID string, operation string) error {
	startTime := time.Now()
	
	logger := p.logger.WithFields(logrus.Fields{
		"tenant_id":   tenantID,
		"calendar_id": calendarID,
		"operation":   operation,
	})

	if p.availabilityChecker != nil {
		p.availabilityChecker.InvalidateProfileNameCache(tenantID, calendarID)
		logger.Debug("Invalidated profile mapping cache")
		
		// Record metrics
		metrics.CDCInvalidationTotal.WithLabelValues("profile_calendars", operation).Inc()
		metrics.CDCInvalidationLatency.WithLabelValues("profile_calendars", operation).
			Observe(time.Since(startTime).Seconds())
	}

	return nil
}
```

---

## Testing the CDC Integration

### Test 1: Monitor CDC Events

```bash
# 1. Start watching metrics
watch -n 1 'curl -s http://localhost:8081/metrics | grep calendar_cdc_invalidation'

# 2. In another terminal, make a profile change
curl -X POST http://localhost:8080/v1/graphql \
  -H "X-Hasura-Admin-Secret: secret" \
  -H "Content-Type: application/json" \
  -d '{
    "query": "mutation { 
      update_profile_calendars(
        where: {calendar_id: {_eq: \"test-cal\"}}
        _set: {active: false}
      ) { affected_rows }
    }"
  }'

# 3. Expected metrics to increment:
# calendar_cdc_invalidation_total{table="profile_calendars",operation="UPDATE"} 1
```

### Test 2: Verify Cache Invalidation After CDC

```bash
# 1. Prime the cache (first request)
curl -X POST http://localhost:8081/api/v1/availability \
  -H "X-Tenant-ID: 550e8400-e29b-41d4-a716-446655440000" \
  -d '{"calendar_id":"test-cal",...}'

# Expected: ~50ms (Hasura query), metric: source="hasura"

# 2. Second request (cached)
curl -X POST http://localhost:8081/api/v1/availability \
  -H "X-Tenant-ID: 550e8400-e29b-41d4-a716-446655440000" \
  -d '{"calendar_id":"test-cal",...}'

# Expected: <1ms (L1 cache), metric: source="cache_l1"

# 3. Delete profile_calendars mapping via Hasura
curl -X POST http://localhost:8080/v1/graphql \
  -H "X-Hasura-Admin-Secret: secret" \
  -d '{
    "query": "mutation { 
      delete_profile_calendars(
        where: {calendar_id: {_eq: \"test-cal\"}}
      ) { affected_rows }
    }"
  }'

# 4. Wait 2s for CDC processing
sleep 2

# 5. Next request (cache miss after invalidation)
curl -X POST http://localhost:8081/api/v1/availability \
  -H "X-Tenant-ID: 550e8400-e29b-41d4-a716-446655440000" \
  -d '{"calendar_id":"test-cal",...}'

# Expected: ~50ms (Hasura query again), metric: source="hasura"
```

---

## Environment Variables Required

```bash
# Redpanda
REDPANDA_BROKERS=kafka-broker1:9092,kafka-broker2:9092

# Cache
CACHE_ENABLED=true
REDIS_URL=redis://localhost:6379

# Hasura
HASURA_ENDPOINT=http://hasura:8080/v1/graphql
HASURA_ADMIN_SECRET=your-admin-secret

# Temporal (if using workflows)
TEMPORAL_ENDPOINT=temporal:7233

# Logging
LOG_LEVEL=info
```

---

## Monitoring Dashboard Queries

```promql
# CDC event processing rate
rate(calendar_cdc_invalidation_total[5m])

# CDC invalidation latency p95
histogram_quantile(0.95, rate(calendar_cdc_invalidation_duration_seconds_bucket[5m]))

# Profile resolution metrics after CDC
calendar_profile_resolution_total{source="cache_l1"} / (calendar_profile_resolution_total{source=~".*"})
```

---

## Troubleshooting CDC Integration

### CDC Events Not Being Consumed

1. **Check Redpanda connectivity**
```bash
# Verify brokers are reachable
kafka-broker-api-versions.sh --bootstrap-server kafka-broker:9092
```

2. **Check consumer group lag**
```bash
# Using kafka CLI
kafka-consumer-groups.sh --bootstrap-server kafka-broker:9092 --group calendar-cdc-group --describe
```

3. **Enable debug logging**
```bash
# In main.go
logrus.SetLevel(logrus.DebugLevel)
```

### Invalidation Not Clearing Cache

1. **Check Redis connection**
```bash
redis-cli PING
redis-cli KEYS "profile_name:*" # Should see mappings before invalidation
```

2. **Monitor invalidation calls**
```bash
# Check logs for "Invalidated profile mapping cache" entries
docker logs calendar-service-dev | grep "Invalidated profile mapping"
```

3. **Check L1 cache**
```bash
# Add debug logging in InvalidateProfileNameCache
logger.Info("L1 cache size before invalidation", "size", len(c.localCache.data))
```

---

## Performance Expectations

After CDC integration:

- **Cache invalidation latency**: <5ms (async L1 clear, 2s timeout for L2)
- **Next query after invalidation**: ~50ms (Hasura query)
- **Subsequent cached queries**: <1ms (L1 hit)
- **Cache warm-up time**: ~5 minutes (5-min L1 TTL)
- **Cross-instance invalidation**:  ~100-200ms via Pub/Sub

---

## Next Steps

1. ✅ Copy CDC integration code above
2. ✅ Wire up in `cmd/server/main.go`
3. ✅ Test with single profile_calendars change
4. ✅ Monitor metrics + logs
5. ✅ Add Grafana dashboard
6. ✅ Load test with concurrent updates
7. ✅ Deploy to production
