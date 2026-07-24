# Conversational Load Testing Playbook

This playbook provides comprehensive testing strategies for the conversational layer of the governance-native semantic platform.

## Overview

The conversational layer handles multi-turn dialogues, schema fetches, governance checks, and natural language to SQL compilation. This playbook covers:

- **Single-turn spikes**: Burst QPS testing of NL→query compilations
- **Multi-turn dialogues**: 5-10 turn sessions with clarifications and rewrites
- **Hot entity campaigns**: Cache efficiency testing with repeated queries
- **Adverse conditions**: Delayed services, cache misses, invalidation storms
- **Endurance testing**: 8-24 hour runs for memory leak detection

## Test Scenarios

### 1. Single-Turn Spike Testing

**Objective**: Validate throughput and latency for simple, clear requests under burst load.

```bash
# Run single-turn spike test
curl -X POST http://localhost:8080/conversational-load-test \
  -H "Content-Type: application/json" \
  -d '{
    "duration": "300",
    "concurrency": 50,
    "request_rate": 200,
    "max_turns_per_conversation": 1,
    "ambiguous_request_ratio": 0.1
  }'
```

**Expected Results**:
- p95 latency ≤ 1.5× baseline single-turn SLO
- Cache hit rate ≥ 85%
- Error rate < 1%

### 2. Multi-Turn Dialogue Testing

**Objective**: Test conversational refinement with clarifications and iterative query building.

```bash
# Run multi-turn dialogue test
curl -X POST http://localhost:8080/conversational-load-test \
  -H "Content-Type: application/json" \
  -d '{
    "duration": "600",
    "concurrency": 20,
    "request_rate": 10,
    "max_turns_per_conversation": 8,
    "think_time_between_turns": 2000,
    "ambiguous_request_ratio": 0.6
  }'
```

**Expected Results**:
- Average turns per conversation: 3-5
- Guardrail intervention rate: 40-60%
- p95 per-turn latency ≤ 2× single-turn SLO

### 3. Hot Entity Campaign Testing

**Objective**: Test cache efficiency when many users query the same metrics/domains.

```bash
# Run hot entity campaign test
curl -X POST http://localhost:8080/conversational-load-test \
  -H "Content-Type: application/json" \
  -d '{
    "duration": "300",
    "concurrency": 100,
    "request_rate": 500,
    "hot_entity_ratio": 0.8,
    "max_turns_per_conversation": 1
  }'
```

**Expected Results**:
- Cache hit rate ≥ 95%
- p99 latency ≤ 50ms
- No cache stampedes or thundering herd effects

### 4. Adverse Conditions Testing

**Objective**: Test system resilience under various failure conditions.

#### Delayed Policy Service
```bash
curl -X POST http://localhost:8080/adverse-conditions-test \
  -H "Content-Type: application/json" \
  -d '{
    "scenario_name": "delayed_policy",
    "duration_seconds": 300
  }'
```

#### Cache Miss Storm
```bash
curl -X POST http://localhost:8080/adverse-conditions-test \
  -H "Content-Type: application/json" \
  -d '{
    "scenario_name": "cache_storm",
    "duration_seconds": 600
  }'
```

#### Downstream Throttling
```bash
curl -X POST http://localhost:8080/adverse-conditions-test \
  -H "Content-Type: application/json" \
  -d '{
    "scenario_name": "throttling",
    "duration_seconds": 300
  }'
```

#### Combined Stress Test
```bash
curl -X POST http://localhost:8080/adverse-conditions-test \
  -H "Content-Type: application/json" \
  -d '{
    "scenario_name": "combined",
    "duration_seconds": 900
  }'
```

## Key Metrics to Monitor

### Performance Metrics
- **p50/p95/p99 per turn and per conversation**
- **Requests per second (RPS)**
- **Average response time**
- **Error rate by category**

### Conversational Metrics
- **Guardrail intervention rate** (% rewrites/blocks)
- **Cache hit rates** (schema, prompts, decisions)
- **Average turns per conversation**
- **Clarification success rate**

### Error Taxonomy
- **Timeout errors**: Network/service delays
- **Policy fetch errors**: Governance service issues
- **Planner errors**: Schema/query planning failures
- **Governance errors**: Access control violations

## Test Data Preparation

### Intent Bank Creation
Create a curated set of realistic intents per domain:

```go
// Finance domain intents
financeIntents := []string{
    "Show me total revenue for Q1",
    "What is the profit margin trend?",
    "Revenue by sales region",
    "Top 10 products by sales",
    "Customer acquisition cost by channel",
}

// Marketing domain intents
marketingIntents := []string{
    "Campaign performance by channel",
    "Customer lifetime value analysis",
    "Lead conversion rates",
    "Marketing spend ROI",
    "Attribution modeling results",
}
```

### Ambiguous Request Generation
Generate 20-30% intentionally ambiguous requests:

```go
ambiguousIntents := []string{
    "Show me sales",           // Could be gross/net, by time period
    "What is the margin?",     // Could be gross/net/profit
    "Give me revenue",         // Could be total/recurring, by segment
    "List by category",        // Could be product/customer category
    "Show profit",             // Could be various profit types
}
```

## Workload Modeling

### Realistic User Behavior
- **Think time**: 1-5 seconds between turns
- **Concurrency ramps**: Gradual increase from baseline to peak
- **Session distribution**: Mix of short (1-2 turns) and long (5-10 turns) conversations

### Rate Limiting
```go
// Model realistic request patterns
workloadModel := &WorkloadModel{
    BaselineRPS:     50,
    PeakRPS:        500,
    RampUpDuration:  5 * time.Minute,
    ThinkTime:       2 * time.Second,
    SessionLength:   4, // Average turns per conversation
}
```

## Circuit Breaker Testing

### Fast-Fail Scenarios
Test rapid failure detection and graceful degradation:

1. **Ambiguity threshold exceeded**: Return "we need clarification" quickly
2. **Service unavailability**: Circuit breaker prevents cascade failures
3. **Resource exhaustion**: Shed non-critical work (verbose traces)

### Recovery Testing
- **Service restoration**: Automatic recovery after failures
- **Gradual ramp-up**: Controlled increase in traffic after recovery
- **State consistency**: Ensure no data corruption during failures

## Endurance Testing

### Long-Run Scenarios
```bash
# 8-hour endurance test
curl -X POST http://localhost:8080/conversational-load-test \
  -H "Content-Type: application/json" \
  -d '{
    "duration": "28800",
    "concurrency": 20,
    "request_rate": 50,
    "endurance_mode": true,
    "progress_interval": 300
  }'
```

### Memory Leak Detection
Monitor for:
- **Growing heap usage** over time
- **Increasing GC pressure**
- **Cache memory growth** without bounds
- **Goroutine leaks** from background workers

### Cache Drift Detection
- **Prompt cache effectiveness** degradation
- **Schema cache staleness**
- **Template compilation** performance decay

## Performance Baselines

### Single-Turn SLOs
- **p50**: ≤ 200ms
- **p95**: ≤ 500ms
- **p99**: ≤ 1000ms
- **Error rate**: < 0.1%

### Multi-Turn SLOs
- **Per-turn p95**: ≤ 750ms
- **Conversation completion**: ≤ 5 seconds
- **Guardrail accuracy**: > 95%

### Cache Performance
- **Schema cache hit rate**: > 90%
- **Prompt cache hit rate**: > 85%
- **Decision cache hit rate**: > 95%

## Automated Test Suite

### CI/CD Integration
```yaml
# .github/workflows/load-test.yml
name: Conversational Load Testing
on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  load-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - name: Run conversational load tests
        run: |
          npm run conversational-load-test
      - name: Run adverse conditions tests
        run: |
          npm run adverse-conditions-test
      - name: Generate performance report
        run: |
          npm run generate-performance-report
```

### Performance Regression Detection
- **Historical comparison**: Compare against baseline performance
- **Trend analysis**: Detect gradual performance degradation
- **Anomaly detection**: Flag unexpected performance changes

## Troubleshooting Guide

### High Latency Issues
1. **Check cache hit rates**: Low rates indicate cache inefficiencies
2. **Monitor GC pressure**: Frequent GC can cause latency spikes
3. **Analyze lock contention**: Use `go tool pprof` for mutex profiling
4. **Check downstream services**: Policy service or database delays

### Memory Issues
1. **Heap profiling**: Identify memory allocation hotspots
2. **Goroutine leaks**: Check for unbounded goroutine creation
3. **Cache memory growth**: Monitor cache size and eviction rates
4. **Object pooling**: Ensure proper reuse of expensive objects

### Error Rate Issues
1. **Categorize errors**: Use error taxonomy for root cause analysis
2. **Check service dependencies**: Policy, schema, and database services
3. **Monitor circuit breakers**: Ensure proper failure isolation
4. **Review guardrail logic**: False positives/negatives in ambiguity detection

## Success Criteria

### Functional Requirements
- ✅ **Zero data leakage** across tenants
- ✅ **Guardrail accuracy** > 95%
- ✅ **Query compilation success** > 99%
- ✅ **Conversation completion** > 98%

### Performance Requirements
- ✅ **p95 per-turn latency** ≤ 1.5× SLO
- ✅ **Cache hit rates** > 85%
- ✅ **Error rates** < 1%
- ✅ **Memory stability** (no leaks in 24h tests)

### Resilience Requirements
- ✅ **Graceful degradation** under adverse conditions
- ✅ **Automatic recovery** from service failures
- ✅ **Backpressure handling** prevents cascade failures
- ✅ **Circuit breaker effectiveness** > 95%

This playbook provides a comprehensive framework for testing the conversational layer under various conditions, ensuring robust performance and reliability in production environments.
