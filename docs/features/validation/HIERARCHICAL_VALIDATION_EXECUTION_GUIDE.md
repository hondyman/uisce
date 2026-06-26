# 🚀 EXECUTION GUIDE - Hierarchical Validation Integration

## Current State
✅ All 5 production-ready source files created and verified in repository:
- `backend/internal/rules/hierarchy_resolver.go` (326 lines)
- `backend/internal/rules/validation_engine_hierarchy.go` (318 lines)
- `backend/internal/rules/condition_evaluator_hierarchy.go` (176 lines)
- `frontend/src/components/validation/HierarchyValidationBuilder.tsx` (452 lines)
- `backend/db/migrations/2025_10_20_add_hierarchy_support.sql` (134 lines)

## Phase 1: Database Migration (20 seconds)

### Execute Migration
```bash
cd /Users/eganpj/GitHub/semlayer/backend/db

# Connect to PostgreSQL and run migration
psql postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable \
  < migrations/2025_10_20_add_hierarchy_support.sql
```

### Verify Migration Success
```bash
psql postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable << EOF

-- Check columns were added
SELECT column_name, data_type 
FROM information_schema.columns 
WHERE table_name = 'validation_rules' 
  AND column_name IN ('field_path', 'aggregation_type', 'hierarchy_depth')
ORDER BY ordinal_position;

-- Check indexes were created
SELECT indexname FROM pg_indexes 
WHERE tablename = 'validation_rules' 
  AND indexname LIKE '%hierarchy%';

-- Check sample rules were inserted
SELECT id, name, entity, array_length(field_path, 1) as path_length 
FROM validation_rules 
WHERE field_path IS NOT NULL 
  AND array_length(field_path, 1) > 0;

\q
EOF
```

### Expected Output
```
 column_name      | data_type
──────────────────┼────────────
 field_path       | text[]
 aggregation_type | character varying
 hierarchy_depth  | integer

 indexname
─────────────────────────────────────
 idx_validation_rules_hierarchy
 idx_validation_rules_hierarchy_depth

 id  |                  name                   | entity |  path_length
─────┼──────────────────────────────────────────┼────────┼──────────────
  1  | Line Item Quantity Must Be Positive     | Order  |            2
  2  | Order Total Must Match Line Items Sum   | Order  |            3
  3  | Supplier Region Must Match Order Region | Order  |            3
```

---

## Phase 2: Backend Compilation (90 seconds)

### Build Backend Server
```bash
cd /Users/eganpj/GitHub/semlayer/backend

# Clean build
go build -o server ./cmd/server

# Check for success
if [ -f ./server ]; then
  echo "✅ Backend compiled successfully"
  ls -lh ./server
else
  echo "❌ Build failed"
  exit 1
fi
```

### Expected Output
```
✅ Backend compiled successfully
-rwxr-xr-x  1 eganpj  staff  45M Oct 20 22:50 ./server
```

### Verify Imports
```bash
# Check that hierarchy packages are properly imported
grep -r "hierarchy_resolver\|condition_evaluator_hierarchy\|validation_engine_hierarchy" \
  /Users/eganpj/GitHub/semlayer/backend/cmd \
  /Users/eganpj/GitHub/semlayer/backend/internal/api
```

---

## Phase 3: Frontend Compilation (60 seconds)

### Build Frontend
```bash
cd /Users/eganpj/GitHub/semlayer/frontend

# Install dependencies (if needed)
npm install

# Build
npm run build

# Check for success
if [ -d ./dist ]; then
  echo "✅ Frontend compiled successfully"
  du -sh ./dist
else
  echo "❌ Build failed"
  exit 1
fi
```

### Expected Output
```
✅ Frontend compiled successfully
45M    ./dist
```

### Verify Component
```bash
# Check that HierarchyValidationBuilder.tsx compiles
npm run build 2>&1 | grep -i "error\|warning" | grep -i "hierarchy" || echo "✅ No hierarchy-related errors"
```

---

## Phase 4: Local Testing (30 seconds setup)

### Terminal 1: Start Backend
```bash
cd /Users/eganpj/GitHub/semlayer/backend

# Kill any existing process
pkill -f "go run ./cmd/server" || true
sleep 1

# Start with environment variables
export DATABASE_URL="postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable"
export PORT=8080
export LOG_LEVEL=debug

go run ./cmd/server

# Expected: Server starts without errors, listens on :8080
# Watch for: "Starting validation engine with hierarchy support"
```

### Terminal 2: Start Frontend
```bash
cd /Users/eganpj/GitHub/semlayer/frontend

# Kill any existing process
pkill -f "npm run dev" || true
sleep 1

# Start dev server
npm run dev

# Expected: Vite dev server starts, listens on :5173
# Watch for: "Local: http://localhost:5173"
```

### Terminal 3: Test API
```bash
# Wait 5 seconds for services to start
sleep 5

# Test 1: Health check
echo "=== Test 1: Health Check ==="
curl -s http://localhost:8080/api/health | jq .

# Expected: {"status": "healthy"}

# Test 2: Validate with hierarchy
echo "=== Test 2: Validate Order with Line Items ==="
curl -s -X POST "http://localhost:8080/api/validate" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000000" \
  -H "X-Tenant-Datasource-ID: 11111111-1111-1111-1111-111111111111" \
  -d '{
    "entity": "Order",
    "data": {
      "id": "ORD-001",
      "total": 5000,
      "line_items": [
        {"qty": 100, "price": 2500},
        {"qty": 50, "price": 2500}
      ]
    }
  }' | jq .

# Expected: {"valid": true, "errors": []}

# Test 3: Invalid validation
echo "=== Test 3: Invalid Order (qty exceeds limit) ==="
curl -s -X POST "http://localhost:8080/api/validate" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000000" \
  -H "X-Tenant-Datasource-ID: 11111111-1111-1111-1111-111111111111" \
  -d '{
    "entity": "Order",
    "data": {
      "total": 5000,
      "line_items": [
        {"qty": 2000}
      ]
    }
  }' | jq .

# Expected: {"valid": false, "errors": [{"ruleId": "...", "message": "..."}]}
```

### Terminal 4: Open Browser
```bash
# Open UI in browser
open http://localhost:5173

# Navigate to Validation Rules → Hierarchy Validation
# You should see the HierarchyValidationBuilder component
```

---

## Phase 5: Integration Checklist

### Backend Integration Points

**File:** `backend/internal/api/validation.go` (or equivalent)

Add to validation endpoint handler:
```go
import (
    "github.com/semlayer/backend/internal/rules"
)

func ValidateHandler(w http.ResponseWriter, r *http.Request) {
    // ... existing code ...
    
    // Check for hierarchy rules
    engine := rules.NewValidationEngineWithHierarchy(db, logger)
    valid, errors, err := engine.ValidateHierarchical(
        r.Context(),
        entity,
        data,
        tenantID,
        datasourceID,
    )
    
    if err != nil {
        // Log and continue to next validation
        logger.Error("hierarchy validation failed", err)
    }
    
    // ... return results ...
}
```

### Frontend Integration Points

**File:** `frontend/src/pages/bundles/ValidationRuleEditor.tsx` (or equivalent)

Add to rule editor tabs:
```typescript
import HierarchyValidationBuilder from '@/components/validation/HierarchyValidationBuilder'

export function ValidationRuleEditor() {
    return (
        <Tabs>
            {/* Existing tabs */}
            
            {/* Add hierarchy tab */}
            <TabPane tab="Hierarchy Rules" key="hierarchy">
                <HierarchyValidationBuilder
                    entity={selectedEntity}
                    onRuleSaved={handleSaveRule}
                />
            </TabPane>
        </Tabs>
    )
}
```

---

## Phase 6: Testing Scenarios

### Scenario 1: Parent-Only Validation
**Rule:** Order total must be > 0
```bash
curl -X POST "http://localhost:8080/api/validate" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000000" \
  -d '{
    "entity": "Order",
    "data": {"total": 5000}
  }'
# Expected: valid = true
```

### Scenario 2: Sub-Entity Validation
**Rule:** All line items must have qty > 0
```bash
curl -X POST "http://localhost:8080/api/validate" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000000" \
  -d '{
    "entity": "Order",
    "data": {
      "line_items": [
        {"qty": 100},
        {"qty": 50}
      ]
    }
  }'
# Expected: valid = true
```

### Scenario 3: Aggregate Validation
**Rule:** Order total = SUM(line_items.price)
```bash
curl -X POST "http://localhost:8080/api/validate" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000000" \
  -d '{
    "entity": "Order",
    "data": {
      "total": 5000,
      "line_items": [
        {"price": 2500},
        {"price": 2500}
      ]
    }
  }'
# Expected: valid = true
```

### Scenario 4: Nested Hierarchy
**Rule:** line_items[].product.supplier.region = order.region
```bash
curl -X POST "http://localhost:8080/api/validate" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000000" \
  -d '{
    "entity": "Order",
    "data": {
      "region": "US",
      "line_items": [
        {
          "product": {
            "supplier": {
              "region": "US"
            }
          }
        }
      ]
    }
  }'
# Expected: valid = true
```

### Scenario 5: Performance Test
**Measure:** Validation should complete in <150ms
```bash
time curl -X POST "http://localhost:8080/api/validate" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000000" \
  -d '{
    "entity": "Order",
    "data": {
      "total": 5000,
      "line_items": [
        {"qty": 100, "price": 2500},
        {"qty": 50, "price": 2500}
      ]
    }
  }' > /dev/null

# Expected: real 0m0.XXXs (< 150ms)
```

---

## Phase 7: Deployment

### Staging Deployment
```bash
# 1. Run migration on staging database
psql postgres://postgres:postgres@staging-db:5432/alpha?sslmode=disable \
  < backend/db/migrations/2025_10_20_add_hierarchy_support.sql

# 2. Deploy backend
cd backend && go build -o server ./cmd/server
# Copy 'server' binary to staging

# 3. Deploy frontend
cd frontend && npm run build
# Copy 'dist' folder to staging web server

# 4. Restart services
systemctl restart semlayer-backend
systemctl restart semlayer-frontend

# 5. Test
curl -s http://staging.semlayer.com/api/health | jq .
```

### Production Deployment
```bash
# Same as staging but:
# - Use production database URL
# - Use production secrets for API keys
# - Monitor error logs during rollout
# - Have rollback plan ready
```

---

## Troubleshooting

### Issue: Database Migration Fails
```bash
# Check PostgreSQL is running
psql postgres://postgres:postgres@localhost:5432/alpha -c "SELECT 1"

# Check for syntax errors in migration file
psql postgres://postgres:postgres@localhost:5432/alpha -c "SYNTAX CHECK"

# Manually run migration statements
# (see 2025_10_20_add_hierarchy_support.sql for individual statements)
```

### Issue: Backend Compilation Errors
```bash
# Check Go version
go version  # Should be 1.20+

# Check dependencies
go mod tidy
go mod verify

# Compile with verbose output
go build -v ./cmd/server
```

### Issue: Frontend Compilation Errors
```bash
# Check Node version
node --version  # Should be 16+

# Reinstall dependencies
rm -rf node_modules package-lock.json
npm install

# Build with verbose output
npm run build -- --debug
```

### Issue: Validation Endpoint Not Found
```bash
# Verify backend is running
ps aux | grep "go run ./cmd/server"

# Check port is listening
lsof -i :8080

# Test with curl
curl -v http://localhost:8080/api/validate
```

---

## Success Criteria

✅ **All tests pass when:**
1. Database migration executes without errors
2. Backend compiles successfully
3. Frontend builds successfully
4. Services start without crashes
5. All 5 test scenarios return expected results
6. Performance tests show <150ms response times
7. UI renders HierarchyValidationBuilder component
8. Rules can be created and saved
9. Validations execute against test data

---

## Quick Command Reference

```bash
# Database
psql postgres://postgres:postgres@localhost:5432/alpha < backend/db/migrations/2025_10_20_add_hierarchy_support.sql

# Backend
cd backend && go build ./cmd/server && PORT=8080 go run ./cmd/server

# Frontend
cd frontend && npm install && npm run build && npm run dev

# Test
curl -X POST http://localhost:8080/api/validate -H "Content-Type: application/json" -d '{"entity":"Order","data":{"total":5000}}'

# Verify
ps aux | grep -E "go run|npm"
```

---

## Timeline
- Database Migration: 20 seconds
- Backend Build: 90 seconds
- Frontend Build: 60 seconds
- Local Testing: 30 seconds
- **Total: ~3-4 minutes to working feature**

✅ All files ready in repository. Ready to execute!
