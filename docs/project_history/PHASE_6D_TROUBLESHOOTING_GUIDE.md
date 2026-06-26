# Phase 6d Troubleshooting & Tuning Guide

## 🔍 Diagnosing Resilience Issues

### Issue: Circuit Breaker Stuck in OPEN State

**Symptoms:**
- All requests returning "service unavailable"
- Metrics show circuit state = 2 (OPEN)
- No requests hitting the actual service

**Root Causes:**
1. FailureThreshold too low (opens too easily)
2. Timeout too long (waits forever to try recovery)
3. Service genuinely unavailable

**Diagnosis:**
```go
// Check circuit state and metrics
state := orchestrator.GetCircuitState("service-name")
metrics := orchestrator.GetMetrics("service-name")

log.Printf("Circuit State: %s", state)
log.Printf("Failures: %d, FailureRate: %.2f%%", 
  metrics.TotalFailures, metrics.FailureRate*100)
log.Printf("TimeInOpen: %v", time.Since(metrics.OpenedAt))
```

**Solutions:**
1. **Increase FailureThreshold**
   ```go
   config.FailureThreshold = 10  // Was 5, now allow more failures
   ```
   - Rationale: Only open after 10 consecutive failures instead of 5
   - Use when: Service has expected high error rates (e.g., validation)

2. **Decrease Timeout**
   ```go
   config.Timeout = 15 * time.Second  // Was 60s, retry sooner
   ```
   - Rationale: Try recovery more frequently
   - Use when: Service recovers quickly

3. **Verify Service Health**
   ```bash
   curl -v http://service:8080/health
   # Check: HTTP 200 with {status: "healthy"}
   ```
   - Rationale: If service is actually down, circuit is working correctly
   - Action: Fix the service, then circuit will auto-recover

4. **Check Logs for Actual Errors**
   ```bash
   kubectl logs service-pod | grep -i error | tail -20
   ```
   - Rationale: Identify if errors are transient or persistent
   - Action: Fix root cause or increase threshold temporarily

---

### Issue: High Retry Attempt Rates

**Symptoms:**
- `resilience_retry_attempts_total` rapidly increasing
- `resilience_retry_success_percent` low (< 30%)
- Slow requests

**Root Causes:**
1. Service returning transient errors (network glitches)
2. FailureThreshold on circuit breaker too high (retrying failed requests)
3. Retry backoff too aggressive (not giving service time to recover)

**Diagnosis:**
```go
metrics := orchestrator.GetMetrics()
log.Printf("Total Attempts: %d", metrics.RetryAttempts)
log.Printf("Success Rate: %.2f%%", metrics.RetrySuccessRate*100)
log.Printf("Avg Attempts: %.2f", metrics.AvgAttemptsPerRequest)
```

**Solutions:**
1. **Increase InitialBackoff**
   ```go
   config.InitialBackoff = 200 * time.Millisecond  // Was 100ms
   ```
   - Rationale: Give service more time before retrying
   - Use when: Service needs time to recover (e.g., connection pool warming)

2. **Decrease BackoffMultiplier**
   ```go
   config.BackoffMultiplier = 1.5  // Was 2.0
   ```
   - Rationale: More lenient backoff curve
   - Use when: Service often needs only one retry

3. **Reduce MaxAttempts**
   ```go
   config.MaxAttempts = 2  // Was 3
   ```
   - Rationale: Fail faster, reduce total latency
   - Use when: Retries are unsuccessful (hitting circuit breaker)

4. **Investigate Error Type**
   ```bash
   # Check if errors are retryable
   grep -i "retryable.*false" service.log | head -5
   ```
   - Rationale: Identify if errors should be retried at all
   - Action: Implement RetryableError interface to skip non-transient errors

---

### Issue: Rate Limiter Rejecting Too Many Requests

**Symptoms:**
- `resilience_rate_limit_denied_requests_total` high
- HTTP 429 responses increasing
- Dashboard shows "Active Rate Limits" > 0

**Root Causes:**
1. RequestsPerSec too low
2. Burst capacity too small
3. Traffic spike not expected

**Diagnosis:**
```go
metrics := orchestrator.GetMetrics()
log.Printf("Requested: %d, Allowed: %d, Denied: %d",
  metrics.TotalRequests, metrics.AllowedRequests, metrics.DeniedRequests)
log.Printf("Denial Rate: %.2f%%", 
  float64(metrics.DeniedRequests)/float64(metrics.TotalRequests)*100)
```

**Solutions:**
1. **Increase RequestsPerSec**
   ```go
   config.RequestsPerSec = 200  // Was 100
   ```
   - Rationale: Allow more sustained throughput
   - Use when: Legitimate traffic increased

2. **Increase BurstSize**
   ```go
   config.BurstSize = 1000  // Was 500
   ```
   - Rationale: Allow larger traffic spikes
   - Use when: Bursts are legitimate (batch operations)

3. **Analyze Traffic Pattern**
   ```bash
   # Check if spike is real or anomaly
   kubectl top nodes
   kubectl top pods
   # Check metrics: rate_limit_deny_rate
   ```
   - Rationale: Distinguish legitimate spike from attack
   - Action: Scale horizontally if legitimate, investigate if anomaly

4. **Gradual Increase**
   ```go
   // Don't jump from 100 to 1000 immediately
   // Increase by 50% intervals
   
   // Current: 100 req/sec, 200 burst
   // Step 1: 150 req/sec, 300 burst
   // Step 2: 200 req/sec, 400 burst
   ```
   - Rationale: Avoid system shock
   - Use when: Ramping up for known traffic increase

---

### Issue: Bulkhead Queue Full (Rejections)

**Symptoms:**
- `resilience_bulkhead_queue_full` rejections appearing
- `resilience_bulkhead_current_concurrent` at max
- HTTP 503 responses: "Service overloaded"

**Root Causes:**
1. MaxConcurrent too low for workload
2. Operations slow down (timeouts increase)
3. Downstream service slow

**Diagnosis:**
```go
metrics := orchestrator.GetMetrics()
log.Printf("Max Concurrent: %d, Current: %d, Queued: %d",
  metrics.MaxConcurrent, metrics.CurrentConcurrent, metrics.QueuedCount)
log.Printf("Queue Full Rejections: %d", metrics.QueueFullRejections)
```

**Solutions:**
1. **Increase MaxConcurrent**
   ```go
   config.MaxConcurrent = 100  // Was 50
   ```
   - Rationale: Allow more parallel operations
   - Use when: Operations complete quickly

2. **Increase QueueSize**
   ```go
   config.QueueSize = 500  // Was 200
   ```
   - Rationale: Buffer spiky traffic
   - Use when: Bursts are expected

3. **Reduce Operation Duration**
   - Root cause: Operations taking too long
   - Check: Timeout settings, database slow queries, network latency
   - Solutions:
     * Add database indexes
     * Reduce payload size
     * Implement caching
     * Optimize query

4. **Scale Horizontally**
   ```bash
   kubectl scale deployment validation-service --replicas=3
   ```
   - Rationale: Each instance gets its own bulkhead pool
   - Use when: Single instance can't handle volume

---

### Issue: Timeout Errors Increasing

**Symptoms:**
- `resilience_timeout_rate` increasing
- HTTP 408 responses: "Request Timeout"
- Slow requests in logs

**Root Causes:**
1. DefaultTimeout too aggressive
2. Downstream service slow
3. Network latency high

**Diagnosis:**
```go
metrics := orchestrator.GetMetrics()
log.Printf("Timeouts: %d, Timeout Rate: %.2f%%",
  metrics.TimeoutCount, metrics.TimeoutRate*100)

// Check actual operation latencies
log.Printf("Avg Operation Time: %v", metrics.AvgOperationTime)
log.Printf("P95 Operation Time: %v", metrics.P95OperationTime)
log.Printf("P99 Operation Time: %v", metrics.P99OperationTime)
```

**Solutions:**
1. **Increase DefaultTimeout**
   ```go
   config.DefaultTimeout = 20 * time.Second  // Was 10s
   ```
   - Rationale: Give operations more time
   - Use when: Slow operations are legitimate (search, complex rules)

2. **Check Downstream Service**
   ```bash
   # Measure latency to downstream service
   curl -w "Time: %{time_total}s" http://downstream:8080/api/...
   ```
   - Rationale: If downstream slow, timeout won't help
   - Action: Fix downstream service

3. **Add Caching Layer**
   ```go
   // Cache results to reduce service calls
   cachedResult := cache.Get(key)
   if cachedResult != nil {
    return cachedResult  // No timeout risk
   }
   ```
   - Rationale: Bypass slow service for repeated queries
   - Use when: High cache hit rate expected

4. **Set Service-Specific Timeouts**
   ```go
   // Different timeouts per service type
   config := map[string]TimeoutConfig{
     "validation": {DefaultTimeout: 5 * time.Second},      // Fast
     "search": {DefaultTimeout: 20 * time.Second},         // Slow
     "notification": {DefaultTimeout: 30 * time.Second},   // Can wait
   }
   ```
   - Rationale: Not all services have same speed requirements
   - Use when: Mixed service portfolio

---

### Issue: High Memory Usage

**Symptoms:**
- Container memory increasing over time
- OOMKilled pods
- Memory leaks suspected

**Root Causes:**
1. Bulkhead queue growing unbounded
2. Metrics not cleaned up
3. Goroutine leaks

**Diagnosis:**
```go
// Check queue sizes
metrics := orchestrator.GetMetrics()
for service, m := range metrics {
  log.Printf("%s - Queue: %d, Goroutines: %d",
    service, m.QueuedCount, m.GoroutineCount)
}

// Runtime metrics
log.Printf("Goroutines: %d", runtime.NumGoroutine())
log.Printf("Memory Alloc: %v MB", m.Alloc/1024/1024)
```

**Solutions:**
1. **Set QueueSize Limit**
   ```go
   config.QueueSize = 500  // Reject if queue reaches 500
   ```
   - Rationale: Prevent unbounded memory growth
   - Use when: Can't scale further

2. **Reduce Bulkhead Size**
   ```go
   config.MaxConcurrent = 25  // Was 50
   ```
   - Rationale: Fewer goroutines = less memory
   - Use when: Latency acceptable, memory critical

3. **Clear Old Metrics**
   ```go
   // Periodically clean old metrics
   ticker := time.NewTicker(5 * time.Minute)
   for range ticker.C {
    orchestrator.PruneOldMetrics()
   }
   ```
   - Rationale: Don't keep historical data forever
   - Use when: Metrics growing unbounded

---

## ⚙️ Tuning for Your Workload

### For CPU-Bound Services (Rule Engine, Validation)

```go
config := resilience.OrchestrationConfig{
  CircuitBreaker: resilience.CircuitBreakerConfig{
    FailureThreshold: 5,      // More lenient
    SuccessThreshold: 2,
    Timeout: 60 * time.Second, // Longer to recover
  },
  Retry: resilience.RetryPolicy{
    MaxAttempts: 3,
    InitialBackoff: 100 * time.Millisecond,
    BackoffMultiplier: 2.0,    // Exponential backoff helps
  },
  RateLimit: resilience.RateLimitConfig{
    RequestsPerSec: 50,   // Limit CPU load
    BurstSize: 100,
  },
  Bulkhead: resilience.BulkheadConfig{
    MaxConcurrent: 25,    // CPU cores / 2-4
    QueueSize: 100,
  },
}
```

**Rationale:**
- Low rate limit to avoid CPU saturation
- Smaller bulkhead (match CPU cores)
- Exponential backoff gives CPU time to recover
- Longer circuit timeout for CPU-intensive recovery

---

### For I/O-Bound Services (Database, API Calls)

```go
config := resilience.OrchestrationConfig{
  CircuitBreaker: resilience.CircuitBreakerConfig{
    FailureThreshold: 10,     // More lenient
    SuccessThreshold: 3,
    Timeout: 30 * time.Second,
  },
  Retry: resilience.RetryPolicy{
    MaxAttempts: 5,           // More retries help I/O
    InitialBackoff: 50 * time.Millisecond,
    BackoffMultiplier: 1.5,   // Gentler backoff
  },
  RateLimit: resilience.RateLimitConfig{
    RequestsPerSec: 200,  // I/O can handle high concurrency
    BurstSize: 500,
  },
  Bulkhead: resilience.BulkheadConfig{
    MaxConcurrent: 100,   // Can handle high concurrency
    QueueSize: 500,
  },
}
```

**Rationale:**
- High rate limit (I/O doesn't consume CPU)
- More retries (I/O failures often transient)
- Larger bulkhead (many I/O ops can overlap)
- Gentler backoff (I/O recovers fast)

---

### For Async Services (Notifications, Events)

```go
config := resilience.OrchestrationConfig{
  CircuitBreaker: resilience.CircuitBreakerConfig{
    FailureThreshold: 20,     // Very lenient
    SuccessThreshold: 5,
    Timeout: 120 * time.Second, // Long timeout acceptable
  },
  Retry: resilience.RetryPolicy{
    MaxAttempts: 10,          // Retry many times
    InitialBackoff: 500 * time.Millisecond,
    BackoffMultiplier: 1.2,   // Slow backoff
  },
  RateLimit: resilience.RateLimitConfig{
    RequestsPerSec: 1000, // High throughput
    BurstSize: 5000,
  },
  Bulkhead: resilience.BulkheadConfig{
    MaxConcurrent: 500,   // Async can handle huge concurrency
    QueueSize: 5000,
  },
}
```

**Rationale:**
- Very lenient failure handling (async can wait)
- Many retries (no user waiting)
- High concurrency (fire-and-forget)
- Large queue (batch operations)

---

## 📈 Performance Tuning Checklist

### Baseline Measurements

Before tuning, measure baseline:

```bash
# 1. Measure latency distribution
curl -w "Time: %{time_total}s\n" http://service:8080/api/endpoint | tail -100

# 2. Check error rate
kubectl logs service-pod | grep -i error | wc -l

# 3. Check concurrent connections
netstat -an | grep -c ESTABLISHED

# 4. Check response time percentiles
# Use: Grafana → Performance Analysis → Response Time Percentiles
```

### Iterative Tuning Process

```
1. Baseline: Measure current performance
   ├─ Latency p50, p95, p99
   ├─ Error rate
   ├─ Throughput (req/sec)
   └─ Resource usage (CPU, memory)

2. Hypothesis: Identify bottleneck
   ├─ Too many timeouts? → Increase timeout
   ├─ Circuit breaker open? → Increase threshold
   ├─ Rate limited? → Increase rate
   └─ Queue full? → Increase bulkhead

3. Adjust: Change one variable
   └─ Only one change per test round

4. Measure: Run test again
   └─ Compare to baseline

5. Evaluate: Improvement?
   ├─ Yes: Keep change, iterate
   └─ No: Revert, try different adjustment

6. Stabilize: Run production-like load for 5 min
   └─ Watch for degradation or errors

7. Document: Record final config
   └─ For future reference and rollback
```

### Example Tuning Session

```
ITERATION 1: CIRCUIT BREAKER
  Baseline: Error rate 15%, p95 latency 2s
  Theory: Too many transient errors, circuit opening too fast
  Change: FailureThreshold 5 → 8
  Result: Error rate 8%, p95 latency 1.5s ✓ KEEP
  
ITERATION 2: BULKHEAD
  Baseline: Bulkhead rejection 2%, queue size avg 10
  Theory: Bulkhead too small for concurrent load
  Change: MaxConcurrent 50 → 75
  Result: Rejection 0.1%, p95 latency 1.2s ✓ KEEP
  
ITERATION 3: RATE LIMIT
  Baseline: Rate limit denial 1%, throughput 1000 req/sec
  Theory: Rate limit slightly conservative
  Change: RequestsPerSec 100 → 120
  Result: Denial 0%, throughput 1200 req/sec ✓ KEEP
  
ITERATION 4: TIMEOUT
  Baseline: Timeout rate 0.5%, p99 latency 5s
  Theory: P99 operations need more time
  Change: DefaultTimeout 10s → 12s
  Result: Timeout rate 0.1%, p99 latency 11s ✗ REVERT
           (Worse p99, not helpful)
  
FINAL CONFIG:
  - FailureThreshold: 8
  - MaxConcurrent: 75
  - RequestsPerSec: 120
  - DefaultTimeout: 10s
```

---

## 🚨 Emergency Responses

### Circuit Breaker Won't Close

**Quick Fix:**
```go
// Manual reset (use judiciously)
orchestrator.ResetCircuitBreaker("service-name")
```

**Verification:**
```bash
# Check state after reset
curl http://localhost:8080/debug/circuit-state
# Should show: "service-name": "closed"
```

---

### All Requests Getting Rate Limited

**Quick Fix:**
```go
// Temporarily increase limit
orchestrator.SetRateLimit("service-name", 1000, 5000)
```

**Verification:**
```bash
# Check denial rate drops
curl http://localhost:8080/metrics | grep rate_limit_denied
```

---

### Bulkhead Queue Maxed Out

**Quick Fix:**
```go
// Temporarily increase queue
orchestrator.SetBulkheadSize("service-name", 100, 1000)
```

**Verification:**
```bash
# Check queue clears
curl http://localhost:8080/metrics | grep bulkhead_queue_size
```

---

## 📊 Expected Baseline Metrics

**Healthy System:**
```
Circuit Breaker:
  - State: CLOSED (most of the time)
  - Failure Rate: < 5%
  - State Transitions: < 1 per hour

Retry:
  - Retry Rate: < 1% of requests
  - Success Rate: > 80% (of retries)

Timeout:
  - Timeout Rate: < 0.5% of requests
  - P99 Latency: < 10s

Rate Limit:
  - Denial Rate: < 0.1% of requests
  - Burst Usage: < 50% capacity

Bulkhead:
  - Rejection Rate: < 0.1%
  - Queue Full: 0 times per hour
  - Utilization: 30-70% of capacity
```

**If Any Are Wrong:**
1. **Higher Circuit Breaker Failure Rate**: Service unstable
2. **Higher Timeout Rate**: Latency issue or slow database
3. **Higher Rate Limit Denial**: Legitimate traffic increase
4. **Higher Bulkhead Rejection**: Capacity too small

---

**Troubleshooting Ready: Now you can confidently diagnose and fix any resilience pattern issue!**
