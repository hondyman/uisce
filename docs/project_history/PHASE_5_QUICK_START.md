# ⚡ Phase 5: Trigger System - Quick Start Guide

**Status:** Ready to implement  
**Timeline:** 2-3 weeks  
**Scope:** 3 Core Triggers (Save, Field Change, Delete)  
**Start:** After Phase 1-4 deployed to production

---

## 🎯 What Gets Built

### Before Phase 5 (Phase 1-4)
```
User creates rule: "Order Total must be > 0"
Rule sits in database waiting
User must manually call validation endpoint
```

### After Phase 5
```
User creates rule: "Order Total must be > 0"
Trigger configured: "Save" event on "orders" entity
User saves order with total=-50
  ↓ AUTOMATIC
TRIGGER fires → Validation runs → Error blocks save ✅
```

---

## 📊 Phase 5 Breakdown

### Database (1 day)
```sql
-- NEW table: validation_triggers
CREATE TABLE validation_triggers (
    id UUID PRIMARY KEY,
    tenant_id UUID,
    datasource_id UUID,
    trigger_type VARCHAR(50),      -- "save", "field_change", "delete"
    target_entity VARCHAR(100),    -- "orders", "line_items"
    target_field VARCHAR(100),     -- For "field_change" only
    rule_ids UUID[],               -- Array of rule IDs
    is_active BOOLEAN,
    -- ... more fields
);
```

**Effort:** Database schema + migration (1 day)

### Backend (3-4 days)
```go
// NEW: TriggerEngine
type TriggerEngine struct { ... }

// Hook into API endpoints
OnSave()        // Before INSERT/UPDATE
OnFieldChange() // When field modified
OnDelete()      // Before DELETE
```

**File:** `backend/internal/services/trigger_engine.go` (350 lines)  
**Integration:** Update `api.go` endpoints (30-50 line changes each)  
**Effort:** 3-4 days

### Frontend (2-3 days)
```typescript
// NEW: Trigger management UI
<TriggerConfiguration />
  ├─ Show triggers for entity
  ├─ Add new trigger
  ├─ Link rules to trigger
  └─ Enable/disable triggers
```

**File:** `frontend/src/components/TriggerConfiguration.tsx` (400 lines)  
**Effort:** 2-3 days

### Testing (3-4 days)
```
E2E Test 1: Save trigger
  - Create rule "Total > 0"
  - Create "save" trigger on "orders"
  - Try to save order with total=-50
  - Expect: BLOCKED with error ✅

E2E Test 2: Field Change trigger
  - Change "total" field from 100 to -50
  - Expect: BLOCKED during field validation ✅

E2E Test 3: Delete trigger
  - Create rule "Cannot delete active orders"
  - Try to delete order with status="active"
  - Expect: BLOCKED ✅
```

**Effort:** 3-4 days

### Deployment (1 day)
- Deploy to staging
- QA testing
- Deploy to production

**Effort:** 1 day

---

## 🏗️ Week-by-Week Plan

### Week 1: Foundation
- **Day 1:** Database schema + migration
- **Days 2-3:** Backend TriggerEngine implementation
- **Day 4:** API integration (save, field_change, delete)
- **Day 5:** Unit tests for TriggerEngine

### Week 2: Frontend & Testing
- **Days 1-2:** Frontend TriggerConfiguration component
- **Days 3-4:** E2E tests + debugging
- **Day 5:** Staging deployment + QA

### Week 3: Production (Optional)
- **Days 1-2:** QA sign-off + final tweaks
- **Days 3-4:** Production deployment + monitoring
- **Day 5:** Post-deployment metrics

---

## 💻 Code Locations

| Component | File | Lines | Time |
|-----------|------|-------|------|
| **DB Migration** | `backend/db/migrations/2025_10_21_add_triggers.sql` | 50 | 1d |
| **TriggerEngine** | `backend/internal/services/trigger_engine.go` | 350 | 3d |
| **API Updates** | `backend/internal/api/api.go` | ±100 | 1d |
| **Frontend UI** | `frontend/src/components/TriggerConfiguration.tsx` | 400 | 3d |
| **Tests** | `backend/services/trigger_engine_test.go` | 300 | 3d |
| **Tests** | `frontend/src/components/TriggerConfiguration.test.tsx` | 200 | 2d |

**Total New Code:** ~1,300 lines  
**Total Changes:** 100-150 lines existing code

---

## 🚀 Implementation Order

### Step 1: Database (Day 1)
```bash
# 1. Create migration file
# 2. Define validation_triggers table
# 3. Create index on (tenant_id, datasource_id, trigger_type, target_entity)
# 4. Apply migration to local/staging
# 5. Verify with: \d validation_triggers
```

### Step 2: Backend Core (Days 2-3)
```go
// Create trigger_engine.go with:
type TriggerEngine struct {
    db *sqlx.DB
    validationEngine *ValidationRuleEngine
}

// Implement:
FetchTriggers()     // Find matching triggers
ExecuteTriggers()   // Run validation for each
OnSave()            // Hook for save
OnFieldChange()     // Hook for field change
OnDelete()          // Hook for delete
```

### Step 3: API Integration (Day 4)
```go
// Update api.go endpoints:
CreateOrder()    // Add: triggerEngine.OnSave()
UpdateOrder()    // Add: triggerEngine.OnFieldChange() + OnSave()
DeleteOrder()    // Add: triggerEngine.OnDelete()
```

### Step 4: Frontend (Days 5-6)
```typescript
// Create TriggerConfiguration.tsx with:
- List triggers for selected entity
- Add new trigger (dropdown for type)
- Select rules to attach
- Enable/disable toggle
- Delete button
```

### Step 5: Tests (Days 7-8)
```go
// trigger_engine_test.go
Test_OnSave_BlocksInvalid()
Test_OnFieldChange_ValidatesField()
Test_OnDelete_ChecksRules()
Test_FetchTriggers_ReturnsCorrect()
```

### Step 6: Deploy (Day 9)
```bash
# Staging
# QA
# Production
```

---

## 🎯 Decision Points

### Before Starting, Decide:

**1. Do all 3 core triggers at once or staggered?**
- **Option A:** All 3 together (Week 1-2) ← RECOMMENDED
- **Option B:** Start with "save" only, add others later

**2. Separate branch or main?**
- **Option A:** Feature branch `phase-5-triggers` ← RECOMMENDED
- **Option B:** Commit directly to main

**3. Test coverage requirement?**
- **Option A:** 90%+ coverage (thorough)
- **Option B:** 70%+ coverage (faster)

---

## ✅ Definition of Done

Phase 5a complete when:

- [x] Database migration applied (staging + prod)
- [x] TriggerEngine fully implemented (3 hooks)
- [x] API endpoints integrated
- [x] Frontend UI for trigger management
- [x] E2E tests pass (3 core triggers)
- [x] Documentation updated
- [x] Deployed to staging + QA verified
- [x] Performance metrics meet targets
- [x] Ready for production

---

## 📈 Expected Outcomes

### Before Phase 5
- Rules exist but need manual API calls
- No automatic validation on save/delete
- Users must remember to validate

### After Phase 5
- Automatic validation on save/field change/delete
- Rules block invalid operations automatically
- Zero manual validation needed
- Workday-like user experience ✅

---

## 🎓 Workday Trigger Examples

**Example 1: Order Validation**
```
Trigger: Save on orders
Rules:
  1. Total > 0
  2. Status in ['pending', 'approved']
  3. Customer exists

Result: Save order → Triggers fire → If ANY rule fails → BLOCK
```

**Example 2: Field Validation**
```
Trigger: Field change on total
Rules:
  1. Cannot decrease by > 50%
  2. Cannot change after approval

Result: User edits total → Triggers fire → If ANY rule fails → BLOCK
```

**Example 3: Delete Protection**
```
Trigger: Delete on orders
Rules:
  1. Status != 'active'
  2. No related shipments

Result: Delete click → Triggers fire → If ANY rule fails → BLOCK
```

---

## 🎉 Ready to Start?

Say one of:

- **"Start Phase 5 backend"** → Begin with TriggerEngine.go
- **"Start Phase 5 frontend"** → Begin with TriggerConfiguration.tsx
- **"Start Phase 5 database"** → Begin with migration
- **"Phase 5 full spec"** → See detailed spec (PHASE_5_TRIGGER_SYSTEM_SPECIFICATION.md)
- **"Proceed to Phase 1-4 deployment first"** → Back to production deployment

Which would you like? 🚀
