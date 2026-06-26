# Gold Copy Integration Implementation Guide

Quick reference for integrating gold copy publishing into your handlers.

## Files to Modify

### 1. Rule Handler Integration

**File**: `backend/internal/handlers/rules_handler.go`

Find the method that publishes/promotes a rule to production. Add this code:

```go
// In your rule promotion/publish handler
func (h *RuleHandler) PublishRule(w http.ResponseWriter, r *http.Request) {
  ruleID := chi.URLParam(r, "ruleId")
  tenantID := r.Header.Get("X-Tenant-ID")
  
  // ... your existing validation logic ...
  
  // Get the rule from database
  rule := h.db.GetRule(ruleID, tenantID)
  
  // TODO: When rule promotion happens, add this:
  if rule.Status == "published" {
    dataHash := calculateHash(rule)
    err := h.goldCopyPublisher.PublishRuleAsGoldCopy(
      r.Context(),
      rule,
      "creation",  // changeType: creation, update, deprecation, retirement
      "Production promotion approved",  // changeReason: audit trail
      getUserIDFromToken(r),  // publishedByUserID
      dataHash,  // SHA256 hash for change detection
    )
    if err != nil {
      log.Printf("Warning: Failed to publish gold copy: %v", err)
      // Don't fail the request - Redpanda might be temporarily unavailable
    }
  }
  
  // Return success
  w.Header().Set("Content-Type", "application/json")
  json.NewEncoder(w).Encode(rule)
}
```

**When to call this**:
- ✅ When rule status changes to "published" 
- ✅ When rule is updated after being published (changeType: "update")
- ✅ When rule marked for deprecation (changeType: "deprecation") 
- ✅ When rule is retired (changeType: "retirement")

---

### 2. Template Handler Integration

**File**: `backend/internal/handlers/templates_handler.go`

```go
// In your template approval/publish handler
func (h *TemplateHandler) ApproveTemplate(w http.ResponseWriter, r *http.Request) {
  templateID := chi.URLParam(r, "templateId")
  tenantID := r.Header.Get("X-Tenant-ID")
  
  // ... your existing approval logic ...
  
  template := h.db.GetTemplate(templateID, tenantID)
  
  // TODO: Add this when template is approved:
  if template.Status == "approved" {
    dataHash := calculateHash(template)
    err := h.goldCopyPublisher.PublishTemplateAsGoldCopy(
      r.Context(),
      template,
      "creation",  // changeType
      "Template approved for production use",  // changeReason
      getUserIDFromToken(r),  // publishedByUserID
      dataHash,
    )
    if err != nil {
      log.Printf("Warning: Failed to publish gold copy template: %v", err)
    }
  }
  
  w.Header().Set("Content-Type", "application/json")
  json.NewEncoder(w).Encode(template)
}
```

---

### 3. Preference Handler Integration

**File**: `backend/internal/handlers/preferences_handler.go`

```go
// In your preference certification/promotion handler
func (h *PreferenceHandler) CertifyPreference(w http.ResponseWriter, r *http.Request) {
  prefID := chi.URLParam(r, "prefId")
  tenantID := r.Header.Get("X-Tenant-ID")
  
  // ... your existing certification logic ...
  
  preference := h.db.GetPreference(prefID, tenantID)
  
  // TODO: Add this when preference is certified:
  if preference.Status == "certified" {
    dataHash := calculateHash(preference)
    err := h.goldCopyPublisher.PublishPreferenceAsGoldCopy(
      r.Context(),
      tenantID,
      prefID,
      preference.Key,  // e.g., "SourceThomson", "CalendarUS", "DataQualityPolicy"
      preference.Type,  // e.g., "source", "calendar", "policy"
      preference,  // Full preference data
      "creation",  // changeType
      "Preference certified for production",  // changeReason
      getUserIDFromToken(r),  // publishedByUserID
      dataHash,
    )
    if err != nil {
      log.Printf("Warning: Failed to publish gold copy preference: %v", err)
    }
  }
  
  w.Header().Set("Content-Type", "application/json")
  json.NewEncoder(w).Encode(preference)
}
```

---

### 4. Business Object Handler Integration

**File**: `backend/internal/handlers/business_objects_handler.go`

```go
// In your BO release/certification handler
func (h *BOHandler) ReleaseBusinessObject(w http.ResponseWriter, r *http.Request) {
  boID := chi.URLParam(r, "boId")
  tenantID := r.Header.Get("X-Tenant-ID")
  
  // ... your existing release logic ...
  
  bo := h.db.GetBusinessObject(boID, tenantID)
  
  // TODO: Add this when BO is released:
  if bo.Status == "released" {
    dataHash := calculateHash(bo)
    err := h.goldCopyPublisher.PublishBusinessObjectAsGoldCopy(
      r.Context(),
      bo,
      "creation",  // changeType
      "Business Object released for production",  // changeReason
      getUserIDFromToken(r),  // publishedByUserID
      dataHash,
    )
    if err != nil {
      log.Printf("Warning: Failed to publish gold copy BO: %v", err)
    }
  }
  
  w.Header().Set("Content-Type", "application/json")
  json.NewEncoder(w).Encode(bo)
}
```

---

## Helper Functions to Add

### Add to your handler package

```go
// backend/internal/handlers/helpers.go

import (
  "crypto/sha256"
  "encoding/hex"
  "fmt"
)

// calculateHash computes SHA256 hash of a structure for change detection
func calculateHash(v interface{}) string {
  data, _ := json.Marshal(v)
  hash := sha256.Sum256(data)
  return "sha256:" + hex.EncodeToString(hash[:])
}

// getUserIDFromToken extracts user ID from JWT or auth header
func getUserIDFromToken(r *http.Request) string {
  // Your implementation - extract from Authorization header or context
  // E.g., from JWT claims or OAuth2 user info
  return r.Header.Get("X-User-ID")  // Or extract from JWT
}
```

---

## Constructor Injection Pattern

Make sure your handlers receive the gold copy publisher:

```go
// In your handler initialization
type RuleHandler struct {
  db                    *database.DB
  goldCopyPublisher     *services.GoldCopyPublisher  // ADD THIS
  logger                *log.Logger
}

// Modify constructor
func NewRuleHandler(
  db *database.DB, 
  goldCopyPublisher *services.GoldCopyPublisher,  // ADD THIS PARAM
  logger *log.Logger,
) *RuleHandler {
  return &RuleHandler{
    db:                db,
    goldCopyPublisher: goldCopyPublisher,  // ADD THIS
    logger:            logger,
  }
}

// Same pattern for Template, Preference, BO handlers
```

---

## Update main.go to Pass Publisher to Handlers

**File**: `backend/cmd/semantic-rules-api/main.go`

Already done! But verify it looks like this:

```go
// Initialize gold copy publisher
goldCopyPublisher, err := services.NewGoldCopyPublisher(redpandaBrokers)
if err != nil {
  log.Fatalf("Failed to initialize gold copy publisher: %v", err)
}

// Create handlers with publisher
ruleHandler := handlers.NewRuleHandler(db, goldCopyPublisher, logger)
templateHandler := handlers.NewTemplateHandler(db, goldCopyPublisher, logger)
prefHandler := handlers.NewPreferenceHandler(db, goldCopyPublisher, logger)
boHandler := handlers.NewBOHandler(db, goldCopyPublisher, logger)

// ... register routes ...

// Add graceful shutdown (also already done)
go func() {
  <-sigChan
  if goldCopyPublisher != nil {
    _ = goldCopyPublisher.Close()
  }
}()
```

---

## Testing the Integration

### Test Scenario 1: Publish a Rule

```bash
# 1. Create a rule
curl -X POST http://localhost:8080/api/v1/rules \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 550e8400-e29b-41d4-a716-446655440000" \
  -H "X-User-ID: user-123" \
  -d '{
    "semantic_term": "IsBusinessDay",
    "expression": "calendar.isBusinessDay(date)",
    "rule_engine": "drools"
  }'

# 2. Publish the rule
curl -X POST http://localhost:8080/api/v1/rules/{ruleId}/publish \
  -H "X-Tenant-ID: 550e8400-e29b-41d4-a716-446655440000" \
  -H "X-User-ID: user-123"

# 3. Verify event in Redpanda
rpk topic consume semlayer.gold-copy --limit 1
```

Expected output in Redpanda:
```json
{
  "event_id": "...",
  "event_type": "gold.copy.rule.created",
  "entity_type": "rule",
  "entity_id": "...",
  "entity_key": "IsBusinessDay",
  "tenant_id": "550e8400-e29b-41d4-a716-446655440000",
  "published_by": "user-123",
  "data": { ... full rule data ... }
}
```

### Test Scenario 2: Update a Published Rule

```bash
# 1. Update the rule definition
curl -X PUT http://localhost:8080/api/v1/rules/{ruleId} \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 550e8400-e29b-41d4-a716-446655440000" \
  -H "X-User-ID: user-123" \
  -d '{
    "expression": "calendar.isBusinessDay(date) && !isHoliday(date)"
  }'

# 2. Publish update (if auto-publish is enabled)
# or manually:
curl -X POST http://localhost:8080/api/v1/rules/{ruleId}/publish \
  -H "X-Tenant-ID: 550e8400-e29b-41d4-a716-446655440000" \
  -H "X-User-ID: user-123"

# 3. Verify updated event (changeType: "update")
rpk topic consume semlayer.gold-copy --limit 1
```

---

## Handling Errors Gracefully

**Key principle**: Don't fail the user's request if Redpanda is down!

```go
// Pattern to follow in all handlers:
err := h.goldCopyPublisher.PublishRuleAsGoldCopy(...)
if err != nil {
  // Log the error for operations/debugging
  log.Printf("ERROR: Failed to publish gold copy for rule %s: %v", ruleID, err)
  
  // Send alert to operations (e.g., via Prometheus pushgateway)
  goldCopyPublishErrors.WithLabelValues("rule", err.Error()).Inc()
  
  // BUT: Don't return error to client
  // The rule was still successfully promoted in your database
  // Redpanda is just for downstream systems
}

// Still return success to client
w.WriteHeader(http.StatusOK)
json.NewEncoder(w).Encode(rule)
```

---

## Metrics to Add

```go
// backend/internal/metrics/metrics.go

import "github.com/prometheus/client_golang/prometheus"

var (
  GoldCopyPublished = prometheus.NewCounterVec(
    prometheus.CounterOpts{
      Name: "semlayer_gold_copy_published_total",
      Help: "Total gold copy events successfully published",
    },
    []string{"entity_type", "event_type", "tenant_id"},
  )

  GoldCopyPublishErrors = prometheus.NewCounterVec(
    prometheus.CounterOpts{
      Name: "semlayer_gold_copy_publish_errors_total",
      Help: "Total errors publishing gold copy events",
    },
    []string{"entity_type", "reason"},
  )

  GoldCopyPublishDuration = prometheus.NewHistogramVec(
    prometheus.HistogramOpts{
      Name:    "semlayer_gold_copy_publish_duration_seconds",
      Help:    "Time to publish gold copy events",
      Buckets: []float64{.001, .01, .1, 1},
    },
    []string{"entity_type"},
  )
)

// In init():
prometheus.MustRegister(
  GoldCopyPublished,
  GoldCopyPublishErrors,
  GoldCopyPublishDuration,
)
```

Then use in handlers:
```go
start := time.Now()
err := h.goldCopyPublisher.PublishRuleAsGoldCopy(...)
duration := time.Since(start).Seconds()

if err != nil {
  GoldCopyPublishErrors.WithLabelValues("rule", "connection_timeout").Inc()
} else {
  GoldCopyPublished.WithLabelValues("rule", "gold.copy.rule.created", tenantID).Inc()
  GoldCopyPublishDuration.WithLabelValues("rule").Observe(duration)
}
```

---

## Implementation Checklist

- [ ] **Add constructor param** - Add `goldCopyPublisher *services.GoldCopyPublisher` to all handler constructors
- [ ] **Update main.go** - Pass publisher instance when creating handlers (✅ Already done)
- [ ] **Update RuleHandler** - Add publish call in promote/publish method
- [ ] **Update TemplateHandler** - Add publish call in approve method
- [ ] **Update PreferenceHandler** - Add publish call in certify method
- [ ] **Update BOHandler** - Add publish call in release method
- [ ] **Add helper functions** - Implement `calculateHash()` and `getUserIDFromToken()`
- [ ] **Add error handling** - Wrap all publish calls with error logging (don't fail user request)
- [ ] **Test scenarios** - Run create + publish flow, verify Redpanda events
- [ ] **Add metrics** - Instrument publishing with Prometheus counters
- [ ] **Update logs** - Add structured logging for publish events

---

## Current Status

✅ **Service Created**: `backend/internal/services/gold_copy_publisher.go` (360 lines)
✅ **Main.go Wired**: Publisher initialized and gracefully shut down
⏳ **Handler Integration**: TODO - Add publisher calls to 4 handlers
⏳ **Testing**: TODO - End-to-end test of gold copy publish
⏳ **Monitoring**: TODO - Prometheus metrics

**Estimated time to complete**: 30-45 minutes for all 4 handlers + testing
