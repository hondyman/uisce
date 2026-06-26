# Gold Copy Publishing - Implementation Status

## ✅ Completed Components

### 1. Gold Copy Publisher Service
**File**: `backend/internal/services/gold_copy_publisher.go` (360+ lines)

✅ Complete implementation with:
- 12 gold copy event types defined
- Comprehensive GoldCopyEvent struct with audit trail, data hashing, schema versioning
- 5 publishing methods: PublishGoldCopyEvent, PublishRuleAsGoldCopy, PublishTemplateAsGoldCopy, PublishPreferenceAsGoldCopy, PublishBusinessObjectAsGoldCopy
- Kafka writer integration with headers-based filtering
- Multi-tenant isolation via routing keys and message headers
- Graceful shutdown support

### 2. Main.go Integration
**File**: `backend/cmd/semantic-rules-api/main.go`

✅ Complete with:
- GoldCopyPublisher initialization (env var: REDPANDA_BROKERS)
- Graceful shutdown handling with Close() call
- Service starts without errors

### 3. Documentation
✅ Two comprehensive guides created:
- **GOLD_COPY_PUBLISHING.md** - Feature overview, consumption patterns, troubleshooting
- **GOLD_COPY_INTEGRATION_GUIDE.md** - Step-by-step handler integration instructions

---

## ⏳ Remaining Integration Work (15-20 minutes)

### Phase 1: Update RuleHandler (5 minutes)

**File**: `backend/internal/handlers/rules_handler.go` (lines 48-50)
**Location**: RuleHandler struct definition

**Step 1.1**: Add goldCopyPublisher field to struct
```go
type RuleHandler struct {
	db                    *sql.DB                         // PostgreSQL connection pool
	cache                 interface{}                     // Your cache layer
	goldCopyPublisher     *services.GoldCopyPublisher    // ADD THIS LINE
}
```

**Step 1.2**: Update constructor
```go
// Update NewRuleHandlerWithDB (around line 22 in rules_handler_impl.go)
func NewRuleHandlerWithDB(db *sql.DB, goldCopyPublisher *services.GoldCopyPublisher) *RuleHandler {
	return &RuleHandler{
		db:                db,
		goldCopyPublisher: goldCopyPublisher,  // ADD THIS
	}
}
```

---

### Phase 2: Hook PromoteRule to Gold Copy (5 minutes)

**File**: `backend/internal/handlers/rules_handler_impl.go`
**Location**: PromoteRule method (around line 398)

**Step 2.1**: In PromoteRule method, after updating rule status to "production", add:

Find this code (around line 479):
```go
	// Audit log
	h.auditLog(ctx, tenantID, userID, "RULE_PROMOTED", ruleID, map[string]string{
		"fromStage":  rule.Status,
		"toStage":    req.ToStage,
		"newVersion": fmt.Sprintf("%d", newVersion),
	})
```

**Add this AFTER the audit log and BEFORE the response** (around line 487):

```go
	// Publish to gold copy if promoted to production
	if req.ToStage == "production" {
		ruleData := map[string]interface{}{
			"id":            ruleID,
			"name":          rule.Name,
			"business_object": rule.BusinessObject,
			"description":   rule.Description,
			"status":        "production",
			"version":       newVersion,
			"updated_by":    userID,
		}
		
		dataHash := hashData(ruleData)
		err := h.goldCopyPublisher.PublishRuleAsGoldCopy(
			ctx,
			ruleData,
			"creation",  // changeType: first time to production
			"Promoted to production status",
			userID,
			dataHash,
		)
		if err != nil {
			log.Printf("Warning: Failed to publish rule to gold copy: %v", err)
			// Don't fail the request - just log the warning
		}
	}
```

---

### Phase 3: Add Hash Helper Function (2 minutes)

**File**: `backend/internal/handlers/rules_handler_impl.go`
**Location**: Add to bottom of file (after other handler methods)

```go
// hashData computes SHA256 hash of data for change detection
func hashData(data interface{}) string {
	import (
		"crypto/sha256"
		"encoding/hex"
		"encoding/json"
	)
	
	jsonData, _ := json.Marshal(data)
	hash := sha256.Sum256(jsonData)
	return "sha256:" + hex.EncodeToString(hash[:])
}
```

---

### Phase 4: Update main.go Handler Registration (3 minutes)

**File**: `backend/cmd/semantic-rules-api/main.go`
**Location**: Where handlers are created

Find where RuleHandler is instantiated (search for `NewRuleHandlerWithDB`):

**From**:
```go
ruleHandler := handlers.NewRuleHandlerWithDB(db)
```

**To**:
```go
ruleHandler := handlers.NewRuleHandlerWithDB(db, goldCopyPublisher)
```

Same for other handlers if you add gold copy support to them:
- TemplateHandler
- PreferenceHandler  
- BusinessObjectHandler

---

### Phase 5: TestingWorkflow (5 minutes)

**Quick Test - Rule Publishing to Gold Copy**:

```bash
# 1. Create a rule
RULE_ID=$(curl -s -X POST http://localhost:8080/api/v1/rules \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 550e8400-e29b-41d4-a716-446655440000" \
  -H "X-User-ID: test-user-123" \
  -d '{
    "businessObject": "Account",
    "name": "IsActiveAccount",
    "description": "Rule to identify active accounts",
    "defaultAction": "allow"
  }' | jq -r '.id')

echo "Created Rule: $RULE_ID"

# 2. Publish to testing
curl -s -X POST http://localhost:8080/api/v1/rules/$RULE_ID/publish \
  -H "X-Tenant-ID: 550e8400-e29b-41d4-a716-446655440000" \
  -H "X-User-ID: test-user-123"

# 3. Promote to staging
curl -s -X POST http://localhost:8080/api/v1/rules/$RULE_ID/promote \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 550e8400-e29b-41d4-a716-446655440000" \
  -H "X-User-ID: test-user-123" \
  -d '{"toStage": "staging"}'

# 4. Promote to PRODUCTION (this should trigger gold copy!)
curl -s -X POST http://localhost:8080/api/v1/rules/$RULE_ID/promote \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 550e8400-e29b-41d4-a716-446655440000" \
  -H "X-User-ID: test-user-123" \
  -d '{"toStage": "production"}'

# 5. Verify gold copy event in Redpanda
rpk topic consume semlayer.gold-copy --limit 1 | jq '.'
```

**Expected Output in Redpanda**:
```json
{
  "event_id": "...",
  "event_type": "gold.copy.rule.created",
  "entity_type": "rule",
  "entity_id": "...",
  "entity_key": "IsActiveAccount",
  "tenant_id": "550e8400-e29b-41d4-a716-446655440000",
  "published_by": "test-user-123",
  "data": {
    "id": "...",
    "name": "IsActiveAccount",
    "business_object": "Account",
    "status": "production",
    "version": 3
  }
}
```

---

## Optional Enhancements

### Enhancement 1: Update Event on Rule Changes (5 minutes)
If a rule is updated after being in production, publish update event:

```go
// In UpdateRule handler, after update to production rule:
if rule.Status == "production" {
	err := h.goldCopyPublisher.PublishRuleAsGoldCopy(
		ctx,
		updatedRuleData,
		"update",  // changeType: this is an update, not creation
		"Rule definition updated in production",
		userID,
		dataHash,
	)
}
```

### Enhancement 2: Deprecation Events (5 minutes)
When a rule is deprecated (say, replaced by a newer version):

```go
// In DeprecateRule handler (if you have one):
err := h.goldCopyPublisher.PublishRuleAsGoldCopy(
	ctx,
	ruleData,
	"deprecation",  // changeType
	"Rule deprecated - use IsActiveAccount_v2 instead",
	userID,
	dataHash,
)
```

### Enhancement 3: Metrics (10 minutes)
Add Prometheus metrics to track gold copy publishes:

```go
// Add to metrics package
var goldCopyPublishedCount = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "semlayer_gold_copy_published_total",
		Help: "Total gold copy events published",
	},
	[]string{"entity_type", "tenant_id"},
)

// In PromoteRule handler, after successful publish:
goldCopyPublishedCount.WithLabelValues("rule", tenantID).Inc()
```

### Enhancement 4: Integration with Templates (5 minutes)
Same pattern for TemplatesHandler:

Find `templates_handler.go` and its `ApproveTemplate()` method:
```go
// Add after template approval succeeds:
if template.Status == "approved" {
	err := h.goldCopyPublisher.PublishTemplateAsGoldCopy(...)
}
```

---

## Files You'll Need to Modify

| File | Change | Difficulty | Time |
|------|--------|-----------|------|
| `rules_handler.go` | Add goldCopyPublisher field | Easy | 1 min |
| `rules_handler_impl.go` | Hook into PromoteRule | Medium | 2 min |
| `rules_handler_impl.go` | Add hashData() helper | Easy | 1 min |
| `main.go` | Pass publisher to handlers | Easy | 1 min |
| **Total** | **4 files** | **Easy-Medium** | **5 min** |

---

## Success Criteria Checklist

- [ ] RuleHandler struct updated with goldCopyPublisher field
- [ ] Constructor updated to accept gold copy publisher
- [ ] PromoteRule method publishes to gold copy when promoting to "production"
- [ ] hashData() helper function added
- [ ] main.go passes publisher instance to RuleHandler constructor
- [ ] Build succeeds: `go build ./cmd/semantic-rules-api/main.go` ✅ No errors
- [ ] Test workflow executes successfully
- [ ] Redpanda topic shows gold copy event with correct fields
- [ ] Event includes: event_type, entity_id, tenant_id, data, data_hash, published_by

---

## System State Summary

### Backend
- ✅ Gold copy publisher service: Complete
- ✅ Main.go initialization: Complete
- ✅ Graceful shutdown: Complete
- ⏳ RuleHandler integration: Ready to implement
- ⏳ TemplateHandler integration: Ready for implementation
- ⏳ Other handlers: Optional

### Downstream
- ✅ Redpanda topic configured: `semlayer.gold-copy`
- ✅ Multi-tenant isolation: Implemented
- ✅ Event schema: Comprehensive (12 event types)
- ⏳ Consumer examples: See GOLD_COPY_PUBLISHING.md

### Testing
- ✅ Backend compiled and running
- ✅ API endpoints responding (schedule CRUD tested)
- ⏳ Gold copy end-to-end test: Waiting for handler integration

---

## Next Steps (in priority order)

1. **Implement Phase 1-4** above (~5 minutes)
2. **Run test workflow** to verify end-to-end
3. **Check Redpanda topic** for gold copy events
4. **Optional: Add metrics** for monitoring
5. **Optional: Repeat for TemplateHandler** using same pattern
6. **Document**: How downstream teams consume gold copy events

---

## Quick Reference Commands

**Build backend**:
```bash
cd backend && go build -o semantic-rules-api ./cmd/semantic-rules-api/main.go
```

**Run backend**:
```bash
./semantic-rules-api
```

**Check Redpanda**:
```bash
rpk topic consume semlayer.gold-copy --limit 5
```

**Check service running**:
```bash
curl http://localhost:8080/health
```

---

## Questions?

Refer to:
- **GOLD_COPY_PUBLISHING.md** - How the feature works
- **GOLD_COPY_INTEGRATION_GUIDE.md** - Detailed integration patterns
- **Service code**: `backend/internal/services/gold_copy_publisher.go` - Comprehensive comments
