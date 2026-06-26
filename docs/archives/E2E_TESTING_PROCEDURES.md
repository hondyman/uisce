# E2E Testing Procedures - Workflow Timeout Triggers

**Duration:** 25 minutes  
**Date:** October 21, 2024  
**Status:** ✅ Ready to Execute

---

## Overview

This guide provides comprehensive end-to-end testing procedures for the Workflow Timeout Triggers feature, covering:
- API endpoint validation
- Database state verification
- Frontend UI integration testing
- Multi-tenant isolation verification
- Error handling scenarios
- Real-world workflow simulation

---

## Prerequisites

### Environment Setup

```bash
# 1. Verify backend is running
lsof -i :8080
# Should show: COMMAND PID ... LISTEN (Go server)

# 2. Verify database is running
psql postgres://postgres:postgres@host.docker.internal:5432/alpha?sslmode=disable -c "SELECT 1;"
# Should return: 1

# 3. Verify frontend is running
lsof -i :3000
# Should show: COMMAND PID ... LISTEN (React dev server)

# 4. Set environment variables
export TENANT_ID="00000000-0000-0000-0000-000000000001"
export DATASOURCE_ID="00000000-0000-0000-0000-000000000001"
export API_BASE="http://localhost:8080"
export ADMIN_TOKEN="your-test-token"
```

### Database State

```sql
-- Verify database connection
\c alpha

-- Check migration table
SELECT version, description, installed_on 
FROM schema_migrations 
WHERE description LIKE '%timeout%'
ORDER BY installed_on DESC;

-- Should return 1 row: 
-- 2025_10_20_workflow_timeout_triggers | ...

-- Check sample data
SELECT workflow_name, step_name, due_hours, is_active, COUNT(*) as count
FROM workflow_timeout_triggers
WHERE tenant_id = '00000000-0000-0000-0000-000000000001'
GROUP BY workflow_name, step_name, due_hours, is_active;

-- Should return 3 rows (HireEmployee, OrderApproval, InvoiceProcessing)
```

---

## Test Suite 1: API Endpoint Validation (10 min)

### Test 1.1: GET /api/workflow-timeout-triggers (List)

**Objective:** Verify list endpoint returns all triggers for tenant

```bash
# Execute request
curl -X GET \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "X-Tenant-Datasource-ID: $DATASOURCE_ID" \
  "$API_BASE/api/workflow-timeout-triggers" \
  | jq '.'

# Expected Response:
# HTTP 200 OK
# Body: Array of 3 triggers with fields:
#   - id (UUID)
#   - tenant_id (matches X-Tenant-ID)
#   - workflow_name (HireEmployee, OrderApproval, InvoiceProcessing)
#   - step_name (ManagerApproval, CreditApproval, PaymentApproval)
#   - due_hours (48, 24, 72)
#   - trigger_percentages ([80, 100])
#   - actions (array of TimeoutAction objects)
#   - is_active (true)
#   - created_at (ISO timestamp)
#   - updated_at (ISO timestamp)

# Validation Checklist:
✓ Status code is 200
✓ Response is valid JSON
✓ Array contains exactly 3 items
✓ Each item has all required fields
✓ trigger_percentages is [80, 100] by default
✓ All tenant_id values match X-Tenant-ID header
✓ is_active is true for sample data
```

**SQL Verification:**
```sql
-- Verify database state matches API response
SELECT id, workflow_name, step_name, due_hours, is_active, COUNT(*) as count
FROM workflow_timeout_triggers
WHERE tenant_id = '00000000-0000-0000-0000-000000000001'
  AND is_active = true
GROUP BY id, workflow_name, step_name, due_hours, is_active
ORDER BY workflow_name;

-- Should return:
-- HireEmployee    | ManagerApproval   | 48 | t
-- OrderApproval   | CreditApproval    | 24 | t
-- InvoiceProcessing | PaymentApproval | 72 | t
```

---

### Test 1.2: POST /api/workflow-timeout-triggers (Create)

**Objective:** Verify new timeout trigger creation

```bash
# 1. Create trigger
curl -X POST \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "X-Tenant-Datasource-ID: $DATASOURCE_ID" \
  -H "Content-Type: application/json" \
  -d '{
    "workflow_name": "ApprovalProcess",
    "step_name": "VPApproval",
    "due_hours": 36,
    "trigger_percentages": [75, 90, 100],
    "actions": [
      {"percent": 75, "type": "notify", "target": "assignee", "message": "60% through deadline"},
      {"percent": 90, "type": "notify", "target": "manager", "message": "Almost due!"},
      {"percent": 100, "type": "escalate", "target": "vp", "message": "Approval overdue"}
    ]
  }' \
  "$API_BASE/api/workflow-timeout-triggers" \
  | jq '.id' > /tmp/new_trigger_id.txt

# Save the returned ID
NEW_TRIGGER_ID=$(cat /tmp/new_trigger_id.txt | tr -d '"')
echo "Created trigger: $NEW_TRIGGER_ID"

# Expected Response:
# HTTP 201 Created
# Body: TimeoutTrigger object with:
#   - id (UUID, newly generated)
#   - tenant_id (matches X-Tenant-ID)
#   - workflow_name: "ApprovalProcess"
#   - step_name: "VPApproval"
#   - due_hours: 36
#   - trigger_percentages: [75, 90, 100]
#   - actions: array with 3 items
#   - is_active: true (defaults to true)
#   - created_at: current timestamp
#   - updated_at: current timestamp

# Validation Checklist:
✓ Status code is 201
✓ Response is valid JSON
✓ id field is populated (UUID format)
✓ tenant_id matches X-Tenant-ID
✓ All input fields present in response
✓ is_active defaults to true
✓ created_at and updated_at are present
✓ trigger_percentages matches input
✓ actions array has 3 items
```

**SQL Verification:**
```sql
-- Verify new trigger in database
SELECT id, workflow_name, step_name, due_hours, trigger_percentages, is_active
FROM workflow_timeout_triggers
WHERE id = 'NEW_TRIGGER_ID' AND tenant_id = '00000000-0000-0000-0000-000000000001';

-- Should return 1 row with matching values
-- Note: trigger_percentages is stored as JSON
SELECT actions_json FROM workflow_timeout_triggers WHERE id = 'NEW_TRIGGER_ID';
-- Should show 3 action objects

-- Verify trigger count increased
SELECT COUNT(*) FROM workflow_timeout_triggers 
WHERE tenant_id = '00000000-0000-0000-0000-000000000001' AND is_active = true;
-- Should return 4 (3 original + 1 new)
```

---

### Test 1.3: GET /api/workflow-timeout-triggers/{triggerId} (Read)

**Objective:** Verify single trigger retrieval

```bash
# Use the ID from Test 1.2
TRIGGER_ID=$NEW_TRIGGER_ID

# Execute request
curl -X GET \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "X-Tenant-Datasource-ID: $DATASOURCE_ID" \
  "$API_BASE/api/workflow-timeout-triggers/$TRIGGER_ID" \
  | jq '.'

# Expected Response:
# HTTP 200 OK
# Body: Single TimeoutTrigger object matching created trigger
#   (Same as response from Test 1.2)

# Validation Checklist:
✓ Status code is 200
✓ Response is valid JSON
✓ id matches request parameter
✓ tenant_id matches X-Tenant-ID
✓ All fields present
✓ Data matches database

# Test 1.3b: Not Found Error
curl -X GET \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "X-Tenant-Datasource-ID: $DATASOURCE_ID" \
  "$API_BASE/api/workflow-timeout-triggers/00000000-0000-0000-0000-000000000999" \
  | jq '.'

# Expected Response:
# HTTP 404 Not Found
# Body: {"error": "Trigger not found"}

# Test 1.3c: Tenant Isolation (Try with different tenant)
curl -X GET \
  -H "X-Tenant-ID: 11111111-1111-1111-1111-111111111111" \
  -H "X-Tenant-Datasource-ID: $DATASOURCE_ID" \
  "$API_BASE/api/workflow-timeout-triggers/$TRIGGER_ID" \
  | jq '.'

# Expected Response:
# HTTP 404 Not Found
# Body: {"error": "Trigger not found"}
# (Should NOT return data, even though trigger exists in DB)
```

---

### Test 1.4: PUT /api/workflow-timeout-triggers/{triggerId} (Update)

**Objective:** Verify trigger update

```bash
# Execute update
curl -X PUT \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "X-Tenant-Datasource-ID: $DATASOURCE_ID" \
  -H "Content-Type: application/json" \
  -d '{
    "workflow_name": "ApprovalProcess",
    "step_name": "VPApproval",
    "due_hours": 48,
    "trigger_percentages": [70, 85, 100],
    "actions": [
      {"percent": 70, "type": "notify", "target": "assignee", "message": "50% through deadline"},
      {"percent": 85, "type": "notify", "target": "manager", "message": "Nearly due!"},
      {"percent": 100, "type": "escalate", "target": "vp", "message": "Approval critically overdue"}
    ]
  }' \
  "$API_BASE/api/workflow-timeout-triggers/$TRIGGER_ID" \
  | jq '.'

# Expected Response:
# HTTP 200 OK
# Body: Updated TimeoutTrigger with:
#   - id: unchanged
#   - tenant_id: unchanged
#   - due_hours: 48 (changed from 36)
#   - trigger_percentages: [70, 85, 100] (changed)
#   - actions: updated array
#   - updated_at: new timestamp (later than created_at)

# Validation Checklist:
✓ Status code is 200
✓ Response is valid JSON
✓ id unchanged
✓ tenant_id unchanged
✓ due_hours changed to 48
✓ trigger_percentages changed to [70, 85, 100]
✓ actions array updated
✓ updated_at is newer than before

# Verify in database
SELECT id, due_hours, updated_at
FROM workflow_timeout_triggers
WHERE id = '$TRIGGER_ID' AND tenant_id = '00000000-0000-0000-0000-000000000001';
```

**SQL Verification:**
```sql
-- Verify update
SELECT updated_at FROM workflow_timeout_triggers 
WHERE id = 'TRIGGER_ID' AND tenant_id = '00000000-0000-0000-0000-000000000001';
-- Should show recent timestamp

-- Verify actions were updated
SELECT actions_json FROM workflow_timeout_triggers 
WHERE id = 'TRIGGER_ID';
-- Should show updated action messages
```

---

### Test 1.5: DELETE /api/workflow-timeout-triggers/{triggerId} (Delete)

**Objective:** Verify soft-delete functionality

```bash
# Execute delete
curl -X DELETE \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "X-Tenant-Datasource-ID: $DATASOURCE_ID" \
  "$API_BASE/api/workflow-timeout-triggers/$TRIGGER_ID" \
  | jq '.'

# Expected Response:
# HTTP 200 OK
# Body: {"message": "Trigger deleted successfully"}

# Validation Checklist:
✓ Status code is 200
✓ Response contains success message

# 1.5b: Verify trigger is soft-deleted (still in DB but inactive)
curl -X GET \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "X-Tenant-Datasource-ID: $DATASOURCE_ID" \
  "$API_BASE/api/workflow-timeout-triggers/$TRIGGER_ID" \
  | jq '.'

# Expected Response:
# HTTP 404 Not Found
# Body: {"error": "Trigger not found"}
# (Should not return, because queries filter by is_active=true implicitly)

# BUT verify in database that record still exists:
SELECT id, is_active FROM workflow_timeout_triggers 
WHERE id = '$TRIGGER_ID' AND tenant_id = '00000000-0000-0000-0000-000000000001';
# Should return: is_active = false

# 1.5c: Verify trigger count decreased in list
curl -X GET \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "X-Tenant-Datasource-ID: $DATASOURCE_ID" \
  "$API_BASE/api/workflow-timeout-triggers" \
  | jq 'length'

# Should return 3 (1 new trigger was created in Test 1.2, then deleted)
# Back to original 3 sample triggers
```

**SQL Verification:**
```sql
-- Verify soft delete
SELECT id, is_active FROM workflow_timeout_triggers 
WHERE id = 'TRIGGER_ID' AND tenant_id = '00000000-0000-0000-0000-000000000001';
-- Should return: is_active = false

-- Verify total count (soft-deleted not included in normal queries)
SELECT COUNT(*) FROM workflow_timeout_triggers
WHERE tenant_id = '00000000-0000-0000-0000-000000000001' AND is_active = true;
-- Should return 3

-- Verify total count including soft-deleted
SELECT COUNT(*) FROM workflow_timeout_triggers
WHERE tenant_id = '00000000-0000-0000-0000-000000000001';
-- Should return 4
```

---

### Test 1.6: POST /api/workflow-timeout-triggers/{triggerId}/test (Test)

**Objective:** Verify trigger test/simulation functionality

```bash
# Get a valid trigger ID
TRIGGER_ID_FOR_TEST=$(curl -s -H "X-Tenant-ID: $TENANT_ID" \
  "$API_BASE/api/workflow-timeout-triggers" | jq -r '.[0].id')

echo "Testing with trigger: $TRIGGER_ID_FOR_TEST"

# Execute test
curl -X POST \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "X-Tenant-Datasource-ID: $DATASOURCE_ID" \
  "$API_BASE/api/workflow-timeout-triggers/$TRIGGER_ID_FOR_TEST/test" \
  | jq '.'

# Expected Response:
# HTTP 200 OK
# Body: {
#   "message": "Test executed successfully",
#   "actions": 2,  // number of actions configured
#   "details": [
#     {"percent": 80, "type": "notify", "target": "assignee", "message": "Due soon"},
#     {"percent": 100, "type": "escalate", "target": "hr_director", "message": "Escalated"}
#   ]
# }

# Validation Checklist:
✓ Status code is 200
✓ Response contains success message
✓ actions count matches trigger configuration
✓ details array populated
✓ Each action has type, target, message

# Verify audit log entry was created
SELECT workflow_name, action, details, created_at
FROM workflow_audit_log
WHERE action = 'timeout_trigger_test'
ORDER BY created_at DESC LIMIT 1;

# Should show recent entry with trigger test details
```

---

## Test Suite 2: Error Handling (5 min)

### Test 2.1: Missing Required Header

```bash
# Missing X-Tenant-ID header
curl -X GET \
  "$API_BASE/api/workflow-timeout-triggers" \
  | jq '.'

# Expected: HTTP 400 Bad Request
# Body: {"error": "X-Tenant-ID header is required"}

✓ Validation: Header validation working correctly
```

### Test 2.2: Invalid JSON Body

```bash
# POST with invalid JSON
curl -X POST \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "Content-Type: application/json" \
  -d '{invalid json}' \
  "$API_BASE/api/workflow-timeout-triggers" \
  | jq '.'

# Expected: HTTP 400 Bad Request
# Body: {"error": "Invalid request body"}

✓ Validation: JSON parsing error handling working
```

### Test 2.3: Missing Required Fields

```bash
# POST without required workflow_name
curl -X POST \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "Content-Type: application/json" \
  -d '{"step_name": "Test", "due_hours": 24, "actions": []}' \
  "$API_BASE/api/workflow-timeout-triggers" \
  | jq '.'

# Expected: HTTP 400 Bad Request or 500 Internal Server Error
# (Validation depends on database constraints)

✓ Validation: Required field validation working
```

### Test 2.4: Cross-Tenant Access Prevention

```bash
# Create trigger with Tenant A
TRIGGER_ID=$(curl -s -X POST \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -H "Content-Type: application/json" \
  -d '{"workflow_name":"Test","step_name":"Step","due_hours":24,"actions":[]}' \
  "$API_BASE/api/workflow-timeout-triggers" | jq -r '.id')

# Try to access with Tenant B
curl -X GET \
  -H "X-Tenant-ID: 99999999-9999-9999-9999-999999999999" \
  "$API_BASE/api/workflow-timeout-triggers/$TRIGGER_ID" \
  | jq '.'

# Expected: HTTP 404 Not Found
# Body: {"error": "Trigger not found"}

✓ Validation: Multi-tenant isolation verified
```

---

## Test Suite 3: Frontend Integration (5 min)

### Test 3.1: UI Load and Display

**Objective:** Verify frontend loads and displays triggers

```bash
# 1. Open browser to frontend
open "http://localhost:3000"

# 2. Navigate to Workflow Timeouts page
# URL should be: http://localhost:3000/workflow-timeouts

# 3. Check for:
✓ Page title: "Workflow Timeout Triggers"
✓ Tenant selector visible (if needed)
✓ Create Trigger button visible
✓ Table loaded with 3 sample triggers:
  - HireEmployee | ManagerApproval | 48h
  - OrderApproval | CreditApproval | 24h
  - InvoiceProcessing | PaymentApproval | 72h

# 4. Check browser console for errors
# Open DevTools: Cmd+Option+I
# Console tab should show NO errors
# Should see API calls logged
```

### Test 3.2: Create via UI

```bash
# 1. Click "Create Timeout Trigger" button
# 2. Fill form:
#    - Workflow: "LeaveApproval"
#    - Step: "ManagerApproval"
#    - Due Hours: 72
#    - Actions:
#      * 80% - Notify Manager
#      * 100% - Escalate to HR

# 3. Click "Create Trigger"
# 4. Verify:
✓ Loading spinner appears
✓ Success message shown: "Timeout trigger created"
✓ New trigger appears in table
✓ API call visible in Network tab (POST /api/workflow-timeout-triggers)

# 5. Check database:
SELECT COUNT(*) FROM workflow_timeout_triggers 
WHERE tenant_id = 'YOUR_TENANT' AND is_active = true;
# Should be 4 (3 original + 1 new)
```

### Test 3.3: Update via UI

```bash
# 1. Click "Edit" on first trigger
# 2. Form should populate with trigger data
# 3. Change "Due Hours" from 48 to 96
# 4. Click "Update Trigger"
# 5. Verify:
✓ Loading spinner appears
✓ Success message shown: "Timeout trigger updated"
✓ Table updates with new value
✓ API call visible in Network tab (PUT /api/workflow-timeout-triggers/{id})

# 6. Click edit again to verify change persisted
✓ Due Hours now shows 96
```

### Test 3.4: Delete via UI

```bash
# 1. Click "Delete" on a trigger
# 2. Confirmation modal appears
# 3. Click "Delete" in modal
# 4. Verify:
✓ Loading spinner appears
✓ Success message shown: "Timeout trigger deleted"
✓ Trigger removed from table
✓ API call visible in Network tab (DELETE /api/workflow-timeout-triggers/{id})

# 5. Verify count decreased
✓ Table now shows 3 triggers (or 4 if you added one)
```

### Test 3.5: Test via UI

```bash
# 1. Click "Test" on a trigger
# 2. Confirmation modal shows trigger details
# 3. Click "Execute Test"
# 4. Verify:
✓ Loading spinner appears
✓ Success message shown with action count
✓ Example: "Test executed successfully. Actions: 2"
✓ API call visible in Network tab (POST /api/workflow-timeout-triggers/{id}/test)

# 5. Check database audit log
SELECT * FROM workflow_audit_log 
WHERE action = 'timeout_trigger_test'
ORDER BY created_at DESC LIMIT 1;
# Should show recent test entry
```

---

## Test Suite 4: Performance & Load (Optional - 5 min)

### Test 4.1: Response Time Benchmark

```bash
# List endpoint with 1 trigger
time curl -s -H "X-Tenant-ID: $TENANT_ID" \
  "$API_BASE/api/workflow-timeout-triggers" | jq 'length'

# Expected: <100ms response time
# Typical: 20-50ms with local database

# Create endpoint
time curl -s -X POST \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "Content-Type: application/json" \
  -d '{"workflow_name":"Perf","step_name":"Test","due_hours":24,"actions":[]}' \
  "$API_BASE/api/workflow-timeout-triggers" > /dev/null

# Expected: <200ms response time
# Typical: 50-100ms with local database
```

### Test 4.2: Database Query Performance

```sql
-- Analyze list query performance
EXPLAIN ANALYZE
SELECT id, tenant_id, workflow_name, step_name, due_hours,
       trigger_percentages, actions_json, is_active, created_at, updated_at
FROM workflow_timeout_triggers
WHERE tenant_id = '00000000-0000-0000-0000-000000000001'
ORDER BY workflow_name, step_name;

-- Check index usage
SELECT schemaname, tablename, indexname, idx_scan, idx_tup_read, idx_tup_fetch
FROM pg_stat_user_indexes
WHERE tablename = 'workflow_timeout_triggers';

-- Expected: Index on (tenant_id, is_active) should be used
```

---

## Test Summary Checklist

```
API Endpoint Tests:
✓ [1.1] GET /list - Returns all triggers
✓ [1.2] POST /create - Creates new trigger
✓ [1.3] GET /{id} - Returns single trigger
✓ [1.4] PUT /{id} - Updates trigger
✓ [1.5] DELETE /{id} - Soft-deletes trigger
✓ [1.6] POST /{id}/test - Tests trigger

Error Handling Tests:
✓ [2.1] Missing X-Tenant-ID header rejected
✓ [2.2] Invalid JSON rejected
✓ [2.3] Missing required fields rejected
✓ [2.4] Cross-tenant access prevented

Frontend Integration Tests:
✓ [3.1] UI loads and displays triggers
✓ [3.2] Create via UI works
✓ [3.3] Update via UI works
✓ [3.4] Delete via UI works
✓ [3.5] Test via UI works

Performance Tests (Optional):
✓ [4.1] Response times acceptable
✓ [4.2] Database queries use indexes

Database Verification:
✓ Data persists correctly
✓ Soft-delete working (is_active flag)
✓ Audit log entries created
✓ Tenant isolation enforced
```

---

## Troubleshooting

### Issue: "X-Tenant-ID header is required"

**Problem:** API returns 400 error with this message

**Solution:**
```bash
# Verify header is being sent
curl -v -X GET \
  -H "X-Tenant-ID: $TENANT_ID" \
  "$API_BASE/api/workflow-timeout-triggers" 2>&1 | grep -i "x-tenant-id"

# Should see: X-Tenant-ID: 00000000-0000-0000-0000-000000000001

# If not appearing, check environment variable
echo $TENANT_ID
# If empty, set it: export TENANT_ID="00000000-0000-0000-0000-000000000001"
```

### Issue: "Trigger not found" when trigger exists

**Problem:** GET returns 404 but trigger is in database

**Solution:**
1. Verify tenant ID matches:
   ```bash
   # Check what tenant ID is in database
   psql -c "SELECT DISTINCT tenant_id FROM workflow_timeout_triggers LIMIT 1;"
   
   # Compare with header
   echo $TENANT_ID
   ```

2. Verify trigger is active:
   ```bash
   # Check is_active flag
   psql -c "SELECT id, is_active FROM workflow_timeout_triggers WHERE id = 'TRIGGER_ID';"
   # Should return: is_active = true
   ```

### Issue: Frontend shows "No triggers loaded"

**Problem:** Frontend displays empty list

**Solution:**
1. Check browser console for errors (Cmd+Option+I)
2. Verify tenant is selected in localStorage:
   ```javascript
   localStorage.getItem('selected_tenant')
   // Should return: {"id": "...", "display_name": "..."}
   ```
3. Check Network tab for failed API requests
4. Verify backend is running: `curl $API_BASE/health`

### Issue: "Database error" from API

**Problem:** API returns 500 with "Database error"

**Solution:**
```bash
# Check database connection
psql postgres://postgres:postgres@host.docker.internal:5432/alpha?sslmode=disable \
  -c "SELECT 1 FROM workflow_timeout_triggers LIMIT 1;"

# Check backend logs
tail -f /var/log/semlayer/backend.log | grep -i error

# Verify schema exists
psql -c "\dt workflow_timeout_triggers"
# Should show table
```

---

## Expected Results

After running all tests, you should have:

✅ **API Endpoints:** All 6 endpoints working correctly  
✅ **Error Handling:** Proper HTTP status codes and error messages  
✅ **Frontend:** UI fully functional with real API integration  
✅ **Database:** Data persisting correctly with soft-deletes  
✅ **Tenant Isolation:** Cross-tenant access prevented  
✅ **Audit Trail:** Test executions logged  

**System is ready for production deployment.**

---

*End of E2E Testing Procedures*
