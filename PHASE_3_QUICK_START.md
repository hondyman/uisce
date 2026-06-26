# Phase 3 Quick Start Guide - Backend Integration

**Date:** February 20, 2026  
**Status:** ✅ ALL FILES VERIFIED - Ready for Backend Integration  
**Time Estimate:** 5 hours to production

---

## ✅ Pre-Flight Checks Completed

### File Verification
- ✅ **Backend Handler** → `/backend/internal/handlers/rules_handler.go` (589 lines)
- ✅ **Database Schema** → `/backend/migrations/003_semantic_rules_schema.sql` (317 lines)
- ✅ **Frontend Service** → `/frontend/src/services/ruleService.ts` (232 lines)
- ✅ **Frontend Components** → 5 Material-UI components in `/frontend/src/components/`
- ✅ **Frontend Hooks** → 3 hooks (useRuleBuilder, useSemanticTerms, useSimulation)
- ✅ **Documentation** → 6 comprehensive guides (3,500+ lines)

---

## 🎯 IMMEDIATE ACTIONS (Start Here)

### Phase 1: Database Setup (15 minutes)

**Step 1.1: Backup existing database**
```bash
cd /Users/eganpj/GitHub/semlayer
pg_dump -h 100.84.126.19 -U admin -d alpha > backup_$(date +%Y%m%d_%H%M%S).sql
echo "✅ Backup created"
```

**Step 1.2: Run migration**
```bash
psql -h 100.84.126.19 -U admin -d alpha < backend/migrations/003_semantic_rules_schema.sql
```

**Step 1.3: Verify tables created**
```bash
psql -h 100.84.126.19 -U admin -d alpha << EOF
\dt edm.*
SELECT COUNT(*) as semantic_terms FROM edm.semantic_terms;
SELECT COUNT(*) as workflows FROM edm.approval_workflows;
EOF
```

**Expected Output:**
```
List of relations
 Schema | Name                    | Type  | Owner
────────┼────────────────────────┼───────┼──────
 edm    | approval_workflows     | table | admin
 edm    | rule_approvals         | table | admin
 edm    | rule_execution_history | table | admin
 edm    | rule_steps             | table | admin
 edm    | rule_versions          | table | admin
 edm    | rules                  | table | admin
 edm    | semantic_terms         | table | admin

semantic_terms  │ 7
workflows       │ 3
```

---

### Phase 2: Backend Implementation (1-2 hours)

**Step 2.1: Install dependencies**
```bash
cd /Users/eganpj/GitHub/semlayer/backend
go get github.com/lib/pq
go get github.com/google/uuid
go mod tidy
```

**Step 2.2: Navigate to handler file**
```bash
cd /Users/eganpj/GitHub/semlayer/backend/internal/handlers
ls -la rules_handler.go
```

**Step 2.3: Implement database methods**

Open `rules_handler.go` and implement these 9 methods. Example implementation for GORM:

```go
// 1. saveRule - INSERT or UPDATE rule
func (h *RuleHandler) saveRule(rule *Rule) error {
    result := h.db.Save(rule)
    return result.Error
}

// 2. getRule - SELECT single rule by ID and tenant
func (h *RuleHandler) getRule(ruleID, tenantID string) (*Rule, error) {
    rule := &Rule{}
    result := h.db.Where("id = ? AND tenant_id = ?", ruleID, tenantID).First(rule)
    if result.Error == gorm.ErrRecordNotFound {
        return nil, fmt.Errorf("rule not found")
    }
    return rule, result.Error
}

// 3. deleteRule - DELETE rule
func (h *RuleHandler) deleteRule(ruleID string) error {
    return h.db.Delete(&Rule{}, "id = ?", ruleID).Error
}

// 4. listRules - SELECT rules by business object and status
func (h *RuleHandler) listRules(businessObject, status, tenantID string) ([]Rule, error) {
    var rules []Rule
    query := h.db.Where("business_object = ? AND tenant_id = ?", businessObject, tenantID)
    if status != "" {
        query = query.Where("status = ?", status)
    }
    result := query.Order("created_at DESC").Limit(50).Find(&rules)
    return rules, result.Error
}

// 5-9. Similar patterns for getRuleVersions, recordApproval, etc.
```

**Step 2.4: Test handler methods**
```bash
cd /Users/eganpj/GitHub/semlayer/backend
go test -v ./internal/handlers -run TestRuleHandler 2>&1 | tee test_output.log
```

**Step 2.5: Register routes in main.go**
```go
// In your main.go or router setup:
router := mux.NewRouter()
ruleHandler := handlers.NewRuleHandler()
ruleHandler.RegisterRoutes(router)
```

**Step 2.6: Start backend server**
```bash
cd /Users/eganpj/GitHub/semlayer/backend
go run cmd/main.go
# Or: go build && ./semantic-engine

# Verify it's running:
curl -s http://localhost:8080/api/v1/rules -H "X-Tenant-ID: test" | head -20
```

---

### Phase 3: Frontend Configuration (5 minutes)

**Step 3.1: Create environment file**
```bash
cd /Users/eganpj/GitHub/semlayer/frontend
cat > .env.local << EOF
REACT_APP_API_URL=http://localhost:8080/api/v1
EOF
echo "✅ Environment configured"
```

**Step 3.2: Verify service file exists**
```bash
ls -la /Users/eganpj/GitHub/semlayer/frontend/src/services/ruleService.ts
```

Expected: ✅ File exists with 13 API functions

---

### Phase 4: Component Testing (1 hour)

**Step 4.1: Start frontend dev server**
```bash
cd /Users/eganpj/GitHub/semlayer/frontend
npm start

# Should see: "Compiled successfully!"
# Visit: http://localhost:3000
```

**Step 4.2: Verify in browser (http://localhost:3000/rules)**
- [ ] No console errors
- [ ] Material-UI components render
- [ ] Three-column layout visible
- [ ] Semantic catalog displays 7 terms
- [ ] Can expand/collapse term categories

**Step 4.3: Test drag-and-drop**
- [ ] Can drag term from left panel
- [ ] Can drop into center panel
- [ ] Step appears in editor

**Step 4.4: Test API connectivity**
In browser console:
```javascript
// Import and test
import { ruleService } from './services/ruleService';

// Test list rules (should return empty array initially)
ruleService.listRules('calendar')
  .then(rules => console.log('✅ API works:', rules.length, 'rules'))
  .catch(err => console.error('❌ API failed:', err.message));
```

---

### Phase 5: API Integration Testing (2-3 hours)

**Step 5.1: Test Endpoint 1 - Create Rule**
```bash
TENANT_ID=$(uuidgen)
USER_ID=$(uuidgen)

curl -X POST http://localhost:8080/api/v1/rules \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "X-User-ID: $USER_ID" \
  -H "Content-Type: application/json" \
  -d '{
    "businessObject": "calendar",
    "name": "Weekend Override",
    "description": "Use golden record for weekends",
    "steps": [
      {
        "priority": 1,
        "condition": {
          "semanticTerm": "IsBusinessDay",
          "operator": "equals",
          "value": "false"
        },
        "action": {
          "useField": "source_field",
          "confidence": 95
        },
        "description": "Exclude weekends"
      }
    ]
  }' | jq .

# Expected: 201 Created with rule_id in response
# Save rule_id for next tests: export RULE_ID=...
```

**Step 5.2: Test Endpoint 2 - List Rules**
```bash
curl -s "http://localhost:8080/api/v1/rules?businessObject=calendar" \
  -H "X-Tenant-ID: $TENANT_ID" | jq '.[] | {id, name, status}'

# Expected: Shows your new rule with status "draft"
```

**Step 5.3: Test Endpoint 3 - Publish Rule**
```bash
curl -X POST "http://localhost:8080/api/v1/rules/$RULE_ID/publish" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "X-User-ID: $USER_ID" \
  -H "Content-Type: application/json" \
  -d '{
    "version": 1,
    "description": "Ready for testing"
  }' | jq '.status'

# Expected: "testing"
```

**Step 5.4: Test Endpoint 4 - Request Approval**
```bash
curl -X POST "http://localhost:8080/api/v1/rules/$RULE_ID/approve" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "X-User-ID: $USER_ID" \
  -H "Content-Type: application/json" \
  -d '{
    "version": 2,
    "role": "data_steward",
    "action": "approve",
    "comments": "Tested and approved"
  }' | jq .

# Expected: 200 OK
```

**Step 5.5: Test Endpoint 5 - Simulate Rule**
```bash
curl -X POST "http://localhost:8080/api/v1/rules/$RULE_ID/simulate" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "X-User-ID: $USER_ID" \
  -H "Content-Type: application/json" \
  -d '{
    "testData": {
      "dates": ["2026-02-20", "2026-02-21", "2026-02-22"],
      "regions": ["GB"]
    }
  }' | jq '.executionTrace[0]'

# Expected: Shows execution trace with date, region, winning rule, confidence
```

**Step 5.6: Test All 13 Endpoints (Use Postman Collection)**

Create `postman_collection.json`:
```json
{
  "info": {"name": "Rules API - Phase 3"},
  "item": [
    {"method": "POST", "url": "http://localhost:8080/api/v1/rules"},
    {"method": "GET", "url": "http://localhost:8080/api/v1/rules"},
    {"method": "GET", "url": "http://localhost:8080/api/v1/rules/{{ruleId}}"},
    {"method": "PUT", "url": "http://localhost:8080/api/v1/rules/{{ruleId}}"},
    {"method": "DELETE", "url": "http://localhost:8080/api/v1/rules/{{ruleId}}"},
    {"method": "POST", "url": "http://localhost:8080/api/v1/rules/{{ruleId}}/publish"},
    {"method": "POST", "url": "http://localhost:8080/api/v1/rules/{{ruleId}}/promote"},
    {"method": "POST", "url": "http://localhost:8080/api/v1/rules/{{ruleId}}/simulate"},
    {"method": "GET", "url": "http://localhost:8080/api/v1/rules/{{ruleId}}/versions"},
    {"method": "GET", "url": "http://localhost:8080/api/v1/rules/{{ruleId}}/diff"},
    {"method": "POST", "url": "http://localhost:8080/api/v1/rules/{{ruleId}}/rollback"},
    {"method": "POST", "url": "http://localhost:8080/api/v1/rules/{{ruleId}}/approve"},
    {"method": "GET", "url": "http://localhost:8080/api/v1/approvals/pending"}
  ]
}
```

---

## 📋 Success Checklist

### Database ✅
- [ ] Migration runs without errors
- [ ] 6 tables created (verify with `\dt edm.*`)
- [ ] 7 semantic_terms populated
- [ ] 3 approval_workflows populated
- [ ] RLS policy active

### Backend ✅
- [ ] All 9 database methods implemented
- [ ] Backend compiles (`go build`)
- [ ] Server starts on port 8080
- [ ] All 13 endpoints respond (HTTP 200-404)
- [ ] Status transitions work (draft → testing → staging → production)

### Frontend ✅
- [ ] Dev server starts without errors
- [ ] All 5 components render
- [ ] Material-UI styling applied
- [ ] Drag-and-drop functional
- [ ] No console errors

### Integration ✅
- [ ] API calls succeed from browser
- [ ] Workflow (create → publish → approve → promote) works end-to-end
- [ ] Error messages display correctly
- [ ] Simulation returns results

**When all items checked:** 🎉 **READY FOR PRODUCTION**

---

## 🆘 Troubleshooting

### Port Already in Use
```bash
# Find process using port 8080
lsof -i :8080

# Kill it
kill -9 <PID>
```

### Database Connection Failed
```bash
# Test connection
psql -h 100.84.126.19 -U admin -d alpha -c "SELECT version();"

# If fails, check:
# 1. PostgreSQL running: pg_isready
# 2. Credentials correct in .env
# 3. alpha database exists: psql -l | grep alpha
```

### CORS Error in Browser
Add CORS middleware to backend:
```go
import "github.com/gorilla/handlers"

// Before RegisterRoutes:
router.Use(handlers.CORS(
    handlers.AllowedOrigins([]string{"http://localhost:3000"}),
    handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE"}),
    handlers.AllowedHeaders([]string{"*"}),
))
```

### Components Not Rendering
```bash
# Check TypeScript errors
cd frontend
npx tsc --noEmit

# Check imports
grep -r "SemanticRuleBuilder" src/
```

---

## 📚 Reference Documents

1. **[PHASE_3_INTEGRATION_ROADMAP.md](./PHASE_3_INTEGRATION_ROADMAP.md)** - Detailed 5-step integration guide
2. **[PHASE_3_ARCHITECTURE_GUIDE.md](./PHASE_3_ARCHITECTURE_GUIDE.md)** - System design & workflows
3. **[PHASE_3_DEPLOYMENT_CHECKLIST.md](./PHASE_3_DEPLOYMENT_CHECKLIST.md)** - Pre-deployment verification
4. **[PHASE_3_QUICK_REFERENCE.md](./PHASE_3_QUICK_REFERENCE.md)** - Quick lookup guide
5. **[PHASE_3_STATUS_DASHBOARD.md](./PHASE_3_STATUS_DASHBOARD.md)** - Project status overview

---

## ⏱️ Timeline

| Phase | Task | Time | Status |
|-------|------|------|--------|
| 1 | Database Setup | 15 min | ⏳ Ready |
| 2 | Backend Implementation | 1-2 hrs | ⏳ Ready |
| 3 | Frontend Config | 5 min | ⏳ Ready |
| 4 | Component Testing | 1 hr | ⏳ Ready |
| 5 | API Integration Testing | 2-3 hrs | ⏳ Ready |
| **Total** | **Production Ready** | **~5 hours** | ⏳ Starting Now |

---

## 🚀 Next Immediate Step

**NOW:** Run Phase 1 (Database Setup)

```bash
cd /Users/eganpj/GitHub/semlayer
pg_dump -h 100.84.126.19 -U admin -d alpha > backup_$(date +%Y%m%d_%H%M%S).sql
psql -h 100.84.126.19 -U admin -d alpha < backend/migrations/003_semantic_rules_schema.sql
echo "✅ Database migration complete"
```

Then proceed to Phase 2 (Backend Implementation).

---

**Status:** ✅ **READY TO BEGIN**  
**Generated:** 2026-02-20  
**Version:** 1.0.0
