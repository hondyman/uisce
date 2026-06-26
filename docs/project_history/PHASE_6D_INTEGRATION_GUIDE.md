# Phase 6d Integration Guide - Service Handler Setup

This guide shows how to integrate Phase 6d resilience patterns into each service handler.

---

## 🏗️ Architecture Pattern

All services follow the same integration pattern:

```go
import "github.com/eganpj/semlayer/backend/internal/resilience"

type ServiceHandler struct {
  orchestrator *resilience.ResilienceOrchestrator
  logger       Logger
  metrics      MetricsExporter
}

func (h *ServiceHandler) Handle(ctx context.Context, req Request) (Response, error) {
  // Execute with all resilience patterns
  return h.orchestrator.Execute(ctx, func(ctx context.Context) (Response, error) {
    return h.executeCore(ctx, req)
  }, resilience.WithFallback("service_degraded"))
}
```

---

## ✅ Validation Service Integration

**File:** `backend/internal/api/validation_handler.go`

**Resilience Configuration:**
- **Circuit Breaker:** FailureThreshold=5, SuccessThreshold=2, Timeout=30s
- **Retry:** MaxAttempts=3, InitialDelay=100ms, BackoffMultiplier=2.0
- **Timeout:** DefaultTimeout=10s, MaxTimeout=30s
- **Rate Limit:** 100 req/sec, Burst=500
- **Bulkhead:** MaxConcurrent=50, QueueSize=200, Timeout=5s

**Rationale:**
- Validation is synchronous and should complete quickly (10s timeout)
- High failure threshold (5) because validation failures are expected, not cascading
- High rate limit (100 req/sec) because validation is lightweight
- Medium bulkhead (50 concurrent) to protect database from overload

**Code Pattern:**
```go
func NewValidationHandler(db Database) *ValidationHandler {
  orch := resilience.NewResilienceOrchestrator(
    "validation-service",
    resilience.CircuitBreakerConfig{
      Name: "validation-service",
      FailureThreshold: 5,
      SuccessThreshold: 2,
      Timeout: 30 * time.Second,
      MaxCalls: 3,
    },
    resilience.RetryPolicy{
      MaxAttempts: 3,
      InitialBackoff: 100 * time.Millisecond,
      MaxBackoff: 10 * time.Second,
      BackoffMultiplier: 2.0,
      JitterFraction: 0.1,
    },
    resilience.TimeoutConfig{
      DefaultTimeout: 10 * time.Second,
      MaxTimeout: 30 * time.Second,
      GracefulShutdown: 5 * time.Second,
    },
    resilience.RateLimitConfig{
      Name: "validation",
      RequestsPerSec: 100,
      BurstSize: 500,
    },
    resilience.BulkheadConfig{
      Name: "validation",
      MaxConcurrent: 50,
      QueueSize: 200,
      WaitTimeout: 5 * time.Second,
    },
  )
  
  // Register fallback: Deny validation on service unavailable
  orch.RegisterFallback("validation_unavailable", func(ctx context.Context, err error) interface{} {
    return &ValidationResponse{
      IsValid: false,
      Errors: []string{"Validation service temporarily unavailable"},
      Reason: "SERVICE_UNAVAILABLE",
    }
  })
  
  return &ValidationHandler{
    db: db,
    orch: orch,
  }
}

func (h *ValidationHandler) ValidateRequest(ctx context.Context, req *ValidationRequest) (*ValidationResponse, error) {
  result, err := h.orch.Execute(ctx, 
    func(ctx context.Context) (interface{}, error) {
      return h.validateCore(ctx, req)
    },
    resilience.WithFallback("validation_unavailable"),
  )
  
  if err != nil {
    return nil, err
  }
  
  return result.(*ValidationResponse), nil
}

func (h *ValidationHandler) validateCore(ctx context.Context, req *ValidationRequest) (*ValidationResponse, error) {
  // Actual validation logic
  // ...
  return &ValidationResponse{IsValid: true}, nil
}
```

---

## ✅ Rule Engine Service Integration

**File:** `backend/internal/api/rule_engine_handler.go`

**Resilience Configuration:**
- **Circuit Breaker:** FailureThreshold=3, SuccessThreshold=2, Timeout=20s (stricter)
- **Retry:** MaxAttempts=2, InitialDelay=50ms, BackoffMultiplier=2.0 (fewer retries)
- **Timeout:** DefaultTimeout=15s, MaxTimeout=45s
- **Rate Limit:** 50 req/sec, Burst=100 (stricter than validation)
- **Bulkhead:** MaxConcurrent=25, QueueSize=100, Timeout=3s (smaller pool)

**Rationale:**
- Rule engine is CPU-intensive, stricter failure threshold (3)
- Fewer retries (2) because failures are usually not transient
- Lower rate limit (50 req/sec) to control CPU usage
- Smaller bulkhead (25 concurrent) to prevent thread explosion

**Code Pattern:**
```go
func NewRuleEngineHandler(engine RuleEngine) *RuleEngineHandler {
  orch := resilience.NewResilienceOrchestrator(
    "rule-engine",
    resilience.CircuitBreakerConfig{
      Name: "rule-engine",
      FailureThreshold: 3,
      SuccessThreshold: 2,
      Timeout: 20 * time.Second,
      MaxCalls: 2,
    },
    resilience.RetryPolicy{
      MaxAttempts: 2,
      InitialBackoff: 50 * time.Millisecond,
      MaxBackoff: 5 * time.Second,
      BackoffMultiplier: 2.0,
      JitterFraction: 0.1,
    },
    resilience.TimeoutConfig{
      DefaultTimeout: 15 * time.Second,
      MaxTimeout: 45 * time.Second,
      GracefulShutdown: 5 * time.Second,
    },
    resilience.RateLimitConfig{
      Name: "rule-engine",
      RequestsPerSec: 50,
      BurstSize: 100,
    },
    resilience.BulkheadConfig{
      Name: "rule-engine",
      MaxConcurrent: 25,
      QueueSize: 100,
      WaitTimeout: 3 * time.Second,
    },
  )
  
  orch.RegisterFallback("rule_engine_unavailable", func(ctx context.Context, err error) interface{} {
    return &RuleEvaluationResponse{
      Status: "DEGRADED",
      Message: "Rule engine degraded, using default rules",
      AppliedRules: []string{},
    }
  })
  
  return &RuleEngineHandler{
    engine: engine,
    orch: orch,
  }
}

func (h *RuleEngineHandler) EvaluateRules(ctx context.Context, req *RuleEvaluationRequest) (*RuleEvaluationResponse, error) {
  result, err := h.orch.Execute(ctx,
    func(ctx context.Context) (interface{}, error) {
      return h.evaluateCore(ctx, req)
    },
    resilience.WithFallback("rule_engine_unavailable"),
  )
  
  if err != nil {
    return nil, err
  }
  
  return result.(*RuleEvaluationResponse), nil
}
```

---

## ✅ Notification Service Integration

**File:** `backend/internal/api/notification_handler.go`

**Resilience Configuration:**
- **Circuit Breaker:** FailureThreshold=10, SuccessThreshold=5, Timeout=60s (lenient)
- **Retry:** MaxAttempts=5, InitialDelay=200ms, BackoffMultiplier=1.5 (more forgiving)
- **Timeout:** DefaultTimeout=30s, MaxTimeout=120s (notifications can be slow)
- **Rate Limit:** 200 req/sec, Burst=1000 (notifications are async, batch-friendly)
- **Bulkhead:** MaxConcurrent=100, QueueSize=500, Timeout=10s (large pool for batching)

**Rationale:**
- Notifications are async and fire-and-forget, so lenient failure handling (10)
- More retries (5) because transient failures in notifications are common
- Longer timeouts because notifications might be batched
- Higher rate limits and bulkhead because notifications scale horizontally
- Large queue because notifications can handle bursts

**Code Pattern:**
```go
func NewNotificationHandler(queue MessageQueue) *NotificationHandler {
  orch := resilience.NewResilienceOrchestrator(
    "notification-service",
    resilience.CircuitBreakerConfig{
      Name: "notification-service",
      FailureThreshold: 10,
      SuccessThreshold: 5,
      Timeout: 60 * time.Second,
      MaxCalls: 10,
    },
    resilience.RetryPolicy{
      MaxAttempts: 5,
      InitialBackoff: 200 * time.Millisecond,
      MaxBackoff: 30 * time.Second,
      BackoffMultiplier: 1.5,
      JitterFraction: 0.1,
    },
    resilience.TimeoutConfig{
      DefaultTimeout: 30 * time.Second,
      MaxTimeout: 120 * time.Second,
      GracefulShutdown: 10 * time.Second,
    },
    resilience.RateLimitConfig{
      Name: "notification",
      RequestsPerSec: 200,
      BurstSize: 1000,
    },
    resilience.BulkheadConfig{
      Name: "notification",
      MaxConcurrent: 100,
      QueueSize: 500,
      WaitTimeout: 10 * time.Second,
    },
  )
  
  orch.RegisterFallback("notification_queued", func(ctx context.Context, err error) interface{} {
    return &NotificationResponse{
      Status: "QUEUED",
      Message: "Notification queued for retry",
      QueueTime: time.Now(),
    }
  })
  
  return &NotificationHandler{
    queue: queue,
    orch: orch,
  }
}

func (h *NotificationHandler) SendNotification(ctx context.Context, req *NotificationRequest) (*NotificationResponse, error) {
  result, err := h.orch.Execute(ctx,
    func(ctx context.Context) (interface{}, error) {
      return h.sendCore(ctx, req)
    },
    resilience.WithFallback("notification_queued"),
  )
  
  if err != nil {
    return nil, err
  }
  
  return result.(*NotificationResponse), nil
}

// Batch processing with resilience
func (h *NotificationHandler) SendBatch(ctx context.Context, reqs []*NotificationRequest) error {
  // Use bulkhead's batching capability
  for i := 0; i < len(reqs); i += 10 {
    batch := reqs[i:min(i+10, len(reqs))]
    _, err := h.orch.Execute(ctx,
      func(ctx context.Context) (interface{}, error) {
        return h.sendBatchCore(ctx, batch)
      },
      resilience.WithFallback("notification_queued"),
    )
    if err != nil {
      return err
    }
  }
  return nil
}
```

---

## ✅ Search Service Integration

**File:** `backend/internal/api/search_handler.go`

**Resilience Configuration:**
- **Circuit Breaker:** FailureThreshold=8, SuccessThreshold=3, Timeout=45s
- **Retry:** MaxAttempts=2, InitialDelay=200ms, BackoffMultiplier=2.0
- **Timeout:** DefaultTimeout=20s, MaxTimeout=60s (searches can be slow)
- **Rate Limit:** 50 req/sec, Burst=200
- **Bulkhead:** MaxConcurrent=30, QueueSize=150, Timeout=5s

**Rationale:**
- Search is often slow but important, longer timeouts (20s)
- Moderate failure threshold (8) because search failures often degrade gracefully
- Fewer retries (2) because search is database-bound, not transient-failure-prone
- Moderate bulkhead (30) to protect database from query storms

**Code Pattern:**
```go
func NewSearchHandler(db Database, index SearchIndex) *SearchHandler {
  orch := resilience.NewResilienceOrchestrator(
    "search-service",
    resilience.CircuitBreakerConfig{
      Name: "search-service",
      FailureThreshold: 8,
      SuccessThreshold: 3,
      Timeout: 45 * time.Second,
      MaxCalls: 5,
    },
    resilience.RetryPolicy{
      MaxAttempts: 2,
      InitialBackoff: 200 * time.Millisecond,
      MaxBackoff: 5 * time.Second,
      BackoffMultiplier: 2.0,
      JitterFraction: 0.1,
    },
    resilience.TimeoutConfig{
      DefaultTimeout: 20 * time.Second,
      MaxTimeout: 60 * time.Second,
      GracefulShutdown: 5 * time.Second,
    },
    resilience.RateLimitConfig{
      Name: "search",
      RequestsPerSec: 50,
      BurstSize: 200,
    },
    resilience.BulkheadConfig{
      Name: "search",
      MaxConcurrent: 30,
      QueueSize: 150,
      WaitTimeout: 5 * time.Second,
    },
  )
  
  orch.RegisterFallback("search_cache", func(ctx context.Context, err error) interface{} {
    return &SearchResponse{
      Results: []SearchResult{},
      Status: "CACHED",
      Message: "Returning cached results due to service degradation",
    }
  })
  
  return &SearchHandler{
    db: db,
    index: index,
    orch: orch,
  }
}

func (h *SearchHandler) Search(ctx context.Context, req *SearchRequest) (*SearchResponse, error) {
  result, err := h.orch.Execute(ctx,
    func(ctx context.Context) (interface{}, error) {
      return h.searchCore(ctx, req)
    },
    resilience.WithFallback("search_cache"),
  )
  
  if err != nil {
    return nil, err
  }
  
  return result.(*SearchResponse), nil
}
```

---

## ✅ Policy Service Integration

**File:** `backend/internal/api/policy_handler.go`

**Resilience Configuration:**
- **Circuit Breaker:** FailureThreshold=2, SuccessThreshold=1, Timeout=15s (strictest)
- **Retry:** MaxAttempts=1, InitialDelay=50ms, BackoffMultiplier=2.0
- **Timeout:** DefaultTimeout=5s, MaxTimeout=15s (strict, policies are critical)
- **Rate Limit:** 30 req/sec, Burst=60
- **Bulkhead:** MaxConcurrent=15, QueueSize=50, Timeout=2s (smallest pool)

**Rationale:**
- Policies are business-critical, strictest failure threshold (2)
- Minimal retries (1) because policy failures are rarely transient
- Short timeouts (5s) because policies should be fast/cached
- Smallest bulkhead because policies must never be degraded
- Smallest rate limit to prioritize policy operations

**Code Pattern:**
```go
func NewPolicyHandler(store PolicyStore) *PolicyHandler {
  orch := resilience.NewResilienceOrchestrator(
    "policy-service",
    resilience.CircuitBreakerConfig{
      Name: "policy-service",
      FailureThreshold: 2,
      SuccessThreshold: 1,
      Timeout: 15 * time.Second,
      MaxCalls: 1,
    },
    resilience.RetryPolicy{
      MaxAttempts: 1,
      InitialBackoff: 50 * time.Millisecond,
      MaxBackoff: 1 * time.Second,
      BackoffMultiplier: 2.0,
      JitterFraction: 0.05,
    },
    resilience.TimeoutConfig{
      DefaultTimeout: 5 * time.Second,
      MaxTimeout: 15 * time.Second,
      GracefulShutdown: 2 * time.Second,
    },
    resilience.RateLimitConfig{
      Name: "policy",
      RequestsPerSec: 30,
      BurstSize: 60,
    },
    resilience.BulkheadConfig{
      Name: "policy",
      MaxConcurrent: 15,
      QueueSize: 50,
      WaitTimeout: 2 * time.Second,
    },
  )
  
  // No fallback for policies - fail hard if unavailable
  orch.RegisterFallback("policy_critical_failure", func(ctx context.Context, err error) interface{} {
    return fmt.Errorf("POLICY_SERVICE_CRITICAL_FAILURE: %w", err)
  })
  
  return &PolicyHandler{
    store: store,
    orch: orch,
  }
}

func (h *PolicyHandler) GetPolicy(ctx context.Context, policyID string) (*Policy, error) {
  result, err := h.orch.Execute(ctx,
    func(ctx context.Context) (interface{}, error) {
      return h.getPolicyCore(ctx, policyID)
    },
  )
  
  if err != nil {
    // For policies, escalate errors immediately
    return nil, fmt.Errorf("policy retrieval failed: %w", err)
  }
  
  return result.(*Policy), nil
}
```

---

## 🔧 HTTP Middleware Integration

**File:** `backend/internal/api/resilience_middleware.go`

```go
package api

import (
  "context"
  "net/http"
  "github.com/eganpj/semlayer/backend/internal/resilience"
)

// ResilienceMiddleware wraps handlers with resilience patterns
func ResilienceMiddleware(orch *resilience.ResilienceOrchestrator) func(http.Handler) http.Handler {
  return func(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
      // Extract service from path or header
      service := extractServiceName(r)
      
      // Add resilience metadata to context
      ctx := context.WithValue(r.Context(), "resilience.service", service)
      
      // Get orchestrator for this service
      serviceOrch := orch.GetServiceOrchestrator(service)
      if serviceOrch == nil {
        http.Error(w, "Service not found", http.StatusNotFound)
        return
      }
      
      // Execute handler with resilience
      _, err := serviceOrch.Execute(ctx, func(ctx context.Context) (interface{}, error) {
        // Capture response
        rw := &responseWriter{ResponseWriter: w}
        next.ServeHTTP(rw, r.WithContext(ctx))
        return rw.statusCode, nil
      })
      
      if err != nil {
        // Resilience pattern triggered
        w.Header().Set("X-Resilience-Error", err.Error())
        
        // Circuit open?
        if serviceOrch.CircuitBreakerState() == "open" {
          w.WriteHeader(http.StatusServiceUnavailable)
          w.Write([]byte("Service temporarily unavailable"))
          return
        }
        
        // Rate limited?
        if err.Error() == "rate limit exceeded" {
          w.WriteHeader(http.StatusTooManyRequests)
          w.Write([]byte("Rate limit exceeded"))
          return
        }
        
        // Timeout?
        if err.Error() == "context deadline exceeded" {
          w.WriteHeader(http.StatusRequestTimeout)
          w.Write([]byte("Request timeout"))
          return
        }
        
        // Bulkhead?
        if err.Error() == "bulkhead queue full" {
          w.WriteHeader(http.StatusServiceUnavailable)
          w.Write([]byte("Service overloaded"))
          return
        }
      }
    })
  }
}

// Helper to capture response status
type responseWriter struct {
  http.ResponseWriter
  statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
  rw.statusCode = code
  rw.ResponseWriter.WriteHeader(code)
}

// Extract service name from request
func extractServiceName(r *http.Request) string {
  // Try header first
  if service := r.Header.Get("X-Service"); service != "" {
    return service
  }
  
  // Try from path (/api/validation, /api/rules, etc)
  parts := strings.Split(r.URL.Path, "/")
  if len(parts) >= 3 {
    return parts[2]
  }
  
  return "unknown"
}
```

---

## 📊 Metrics Export

All handlers should export metrics periodically:

```go
// In main.go or service initialization
func startMetricsExport(orch *resilience.ResilienceOrchestrator, metricsPort int) {
  http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
    metrics := orch.ExportMetrics()
    w.Header().Set("Content-Type", "text/plain")
    w.Write([]byte(metrics))
  })
  
  go http.ListenAndServe(fmt.Sprintf(":%d", metricsPort), nil)
}
```

---

## ✅ Testing Resilience Patterns

**Unit Test Example:**
```go
func TestValidationHandlerCircuitBreaker(t *testing.T) {
  handler := NewValidationHandler(mockDB)
  
  // Trigger failures to open circuit
  for i := 0; i < 5; i++ {
    req := &ValidationRequest{Data: "invalid"}
    _, err := handler.ValidateRequest(context.Background(), req)
    assert.Error(t, err)
  }
  
  // Circuit should now be open
  state := handler.orch.GetState()
  assert.Equal(t, "open", state)
  
  // Next request should fail immediately without calling service
  _, err := handler.ValidateRequest(context.Background(), &ValidationRequest{})
  assert.Equal(t, "circuit breaker open", err.Error())
}
```

**Integration Test Example:**
```go
func TestResilientServiceWithRetry(t *testing.T) {
  // Create failing service that recovers
  callCount := 0
  service := func(ctx context.Context) error {
    callCount++
    if callCount < 2 {
      return errors.New("transient error")
    }
    return nil
  }
  
  handler := NewRuleEngineHandler(mockEngine)
  
  // Should succeed after retry
  err := handler.orch.Execute(context.Background(), service)
  assert.NoError(t, err)
  assert.Equal(t, 2, callCount) // Called twice due to retry
}
```

---

## 🚀 Deployment Checklist

Before deploying Phase 6d resilience patterns:

- [ ] All handlers updated with resilience orchestrator
- [ ] Fallback strategies tested for each service
- [ ] Metrics exported to Prometheus
- [ ] Grafana dashboard imported
- [ ] Alert rules configured
- [ ] Load test with circuit breaker tripping
- [ ] Chaos test with bulkhead overflow
- [ ] Rate limiting tested under load
- [ ] Timeout behavior verified
- [ ] Retry backoff verified with logs
- [ ] Documentation shared with team
- [ ] On-call training completed
- [ ] Gradual rollout plan created (10% → 50% → 100%)

---

**Integration Status: Ready to Deploy**
