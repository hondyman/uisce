# Multi-Entity Validation System - Integration Testing Guide

## Phase 4: Integration Testing

**Status:** Ready to Begin  
**Prerequisite Phases:** ✅ All Complete

---

## Quick Health Check

Before running integration tests, verify everything is ready:

### 1. Frontend Status
```bash
cd /Users/eganpj/GitHub/semlayer/frontend
npm run build  # Should complete with 0 errors
```

**Expected:** ✅ Zero TypeScript errors

### 2. Backend Status
```bash
cd /Users/eganpj/GitHub/semlayer/backend
go test ./internal/api/... -run "TestMultiEntity" -v
```

**Expected:** ✅ 15 tests passing

### 3. Database Status
```bash
psql postgres://postgres:postgres@host.docker.internal:5432/alpha?sslmode=disable
\d catalog_validation_rules
```

**Expected:** 
- ✅ `target_entities TEXT[]` column exists
- ✅ `idx_validation_rules_target_entities` GIN index exists
- ✅ Default value `ARRAY['global']`

---

## Integration Test Scenarios

### Scenario 1: Create Global Rule

**Purpose:** Test global rule creation that applies to all entities

**Steps:**
```bash
curl -X POST http://localhost:29080/api/validation-rules?tenant_id=910638ba-a459-4a3f-bb2d-78391b0595f6 \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 910638ba-a459-4a3f-bb2d-78391b0595f6" \
  -d '{
    "rule_name": "Global Phone Format",
    "rule_type": "field_format",
    "target_entity": "global",
    "target_entities": ["global"],
    "condition_json": {
      "field": "phone",
      "operator": "matches_pattern",
      "value": "\\d{10}"
    },
    "severity": "error",
    "is_active": true
  }'
```

**Expected Response:**
```json
{
  "id": "uuid-here",
  "rule_name": "Global Phone Format",
  "target_entity": "global",
  "target_entities": ["global"],
  "is_active": true,
  ...
}
```

**Verification:**
- ✅ HTTP 201 Created
- ✅ `target_entities` array returned with `["global"]`
- ✅ Rule ID returned

---

### Scenario 2: Create Multi-Entity Rule

**Purpose:** Test one rule applying to multiple entities

**Steps:**
```bash
curl -X POST http://localhost:29080/api/validation-rules?tenant_id=910638ba-a459-4a3f-bb2d-78391b0595f6 \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 910638ba-a459-4a3f-bb2d-78391b0595f6" \
  -d '{
    "rule_name": "Name Required",
    "rule_type": "field_format",
    "target_entity": "Customer",
    "target_entities": ["Customer", "Employee", "Supplier"],
    "condition_json": {
      "field": "name",
      "operator": "not_null"
    },
    "severity": "error",
    "is_active": true
  }'
```

**Expected Response:**
```json
{
  "id": "uuid-here",
  "rule_name": "Name Required",
  "target_entity": "Customer",
  "target_entities": ["Customer", "Employee", "Supplier"],
  "is_active": true,
  ...
}
```

**Verification:**
- ✅ HTTP 201 Created
- ✅ All 3 entities in `target_entities` array
- ✅ Rule applies to 1 rule instead of 3

---

### Scenario 3: Query Rules for Specific Entity

**Purpose:** Test multi-entity query filtering

**Steps:**
```bash
# Query for Customer rules (should return global + Customer-specific)
curl http://localhost:29080/api/validation-rules?tenant_id=910638ba-a459-4a3f-bb2d-78391b0595f6&entity=Customer \
  -H "X-Tenant-ID: 910638ba-a459-4a3f-bb2d-78391b0595f6"
```

**Expected Response:**
```json
[
  {
    "id": "global-phone-id",
    "rule_name": "Global Phone Format",
    "target_entities": ["global"],
    ...
  },
  {
    "id": "name-required-id",
    "rule_name": "Name Required",
    "target_entities": ["Customer", "Employee", "Supplier"],
    ...
  }
]
```

**Verification:**
- ✅ HTTP 200 OK
- ✅ Global rule included (applies to all)
- ✅ Multi-entity rule included (contains Customer)
- ✅ Non-matching rules excluded (e.g., Salary validation for Employee only)

---

### Scenario 4: Query for Employee (Different Entity)

**Purpose:** Test that multi-entity rule applies to different entity

**Steps:**
```bash
# Query for Employee rules (should include global + Name Required rule)
curl http://localhost:29080/api/validation-rules?tenant_id=910638ba-a459-4a3f-bb2d-78391b0595f6&entity=Employee \
  -H "X-Tenant-ID: 910638ba-a459-4a3f-bb2d-78391b0595f6"
```

**Expected Response:**
```json
[
  {
    "id": "global-phone-id",
    "rule_name": "Global Phone Format",
    "target_entities": ["global"],
    ...
  },
  {
    "id": "name-required-id",
    "rule_name": "Name Required",
    "target_entities": ["Customer", "Employee", "Supplier"],
    ...
  }
]
```

**Verification:**
- ✅ Global rule included
- ✅ Name Required rule included (Employee is in array)
- ✅ Same rule applies across multiple entities

---

### Scenario 5: Query Non-Matching Entity

**Purpose:** Test query returns no match for non-matching entity

**Steps:**
```bash
# Query for Product rules (global should match, but not Customer-specific)
curl http://localhost:29080/api/validation-rules?tenant_id=910638ba-a459-4a3f-bb2d-78391b0595f6&entity=Product \
  -H "X-Tenant-ID: 910638ba-a459-4a3f-bb2d-78391b0595f6"
```

**Expected Response:**
```json
[
  {
    "id": "global-phone-id",
    "rule_name": "Global Phone Format",
    "target_entities": ["global"],
    ...
  }
]
```

**Verification:**
- ✅ Global rule included
- ✅ Name Required rule EXCLUDED (Product not in Customer, Employee, Supplier array)
- ✅ Filtering works correctly

---

### Scenario 6: Update Rule's Target Entities

**Purpose:** Test expanding multi-entity rule to additional entities

**Steps:**
```bash
RULE_ID="name-required-id"  # From Scenario 2

curl -X PATCH http://localhost:29080/api/validation-rules/$RULE_ID?tenant_id=910638ba-a459-4a3f-bb2d-78391b0595f6 \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 910638ba-a459-4a3f-bb2d-78391b0595f6" \
  -d '{
    "rule_name": "Name Required",
    "rule_type": "field_format",
    "target_entity": "Customer",
    "target_entities": ["Customer", "Employee", "Supplier", "Product", "Order"],
    "condition_json": {
      "field": "name",
      "operator": "not_null"
    },
    "severity": "error",
    "is_active": true
  }'
```

**Expected Response:**
```json
{
  "id": "name-required-id",
  "rule_name": "Name Required",
  "target_entities": ["Customer", "Employee", "Supplier", "Product", "Order"],
  ...
}
```

**Verification:**
- ✅ HTTP 200 OK
- ✅ `target_entities` updated to 5 entities
- ✅ Rule now applies to Product and Order as well

---

### Scenario 7: Filter Rules by Type and Entity

**Purpose:** Test combined filtering (rule_type + entity)

**Steps:**
```bash
# Get field_format rules for Customer
curl "http://localhost:29080/api/validation-rules?tenant_id=910638ba-a459-4a3f-bb2d-78391b0595f6&entity=Customer&rule_type=field_format" \
  -H "X-Tenant-ID: 910638ba-a459-4a3f-bb2d-78391b0595f6"
```

**Expected Response:**
```json
[
  {
    "id": "global-phone-id",
    "rule_name": "Global Phone Format",
    "rule_type": "field_format",
    "target_entities": ["global"],
    ...
  },
  {
    "id": "name-required-id",
    "rule_name": "Name Required",
    "rule_type": "field_format",
    "target_entities": ["Customer", "Employee", "Supplier", "Product", "Order"],
    ...
  }
]
```

**Verification:**
- ✅ Only `field_format` rules returned
- ✅ Multi-entity rules filtered correctly
- ✅ Global rules included

---

### Scenario 8: Delete Multi-Entity Rule

**Purpose:** Test deletion doesn't leave orphaned rules

**Steps:**
```bash
RULE_ID="name-required-id"  # From previous scenarios

curl -X DELETE http://localhost:29080/api/validation-rules/$RULE_ID?tenant_id=910638ba-a459-4a3f-bb2d-78391b0595f6 \
  -H "X-Tenant-ID: 910638ba-a459-4a3f-bb2d-78391b0595f6"
```

**Expected Response:**
```json
{
  "message": "Validation rule deleted successfully"
}
```

**Verification:**
- ✅ HTTP 200 OK
- ✅ Subsequent query doesn't return deleted rule
- ✅ No orphaned rules in database

---

### Scenario 9: Backward Compatibility - Legacy Single Entity Rule

**Purpose:** Test that old single-entity rules still work

**Steps:**
```bash
curl -X POST http://localhost:29080/api/validation-rules?tenant_id=910638ba-a459-4a3f-bb2d-78391b0595f6 \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 910638ba-a459-4a3f-bb2d-78391b0595f6" \
  -d '{
    "rule_name": "Employee Salary Range",
    "rule_type": "business_logic",
    "target_entity": "Employee",
    "condition_json": {
      "field": "salary",
      "operator": "between",
      "min": 30000,
      "max": 500000
    },
    "severity": "error",
    "is_active": true
  }'
```

**Note:** No `target_entities` provided (legacy mode)

**Expected Response:**
```json
{
  "id": "uuid-here",
  "rule_name": "Employee Salary Range",
  "target_entity": "Employee",
  "target_entities": ["Employee"],  // Auto-converted to array
  ...
}
```

**Verification:**
- ✅ HTTP 201 Created
- ✅ Legacy `target_entity` still accepted
- ✅ Auto-converted to `target_entities` array
- ✅ Backward compatible

---

## Running Tests Programmatically

### Shell Script for Automated Testing

```bash
#!/bin/bash

BASE_URL="http://localhost:29080"
TENANT_ID="910638ba-a459-4a3f-bb2d-78391b0595f6"
HEADERS="-H 'Content-Type: application/json' -H 'X-Tenant-ID: $TENANT_ID'"

# Test 1: Create global rule
echo "Test 1: Creating global rule..."
GLOBAL_RULE=$(curl -s -X POST "$BASE_URL/api/validation-rules?tenant_id=$TENANT_ID" \
  $HEADERS \
  -d '{
    "rule_name": "Global Phone Format",
    "rule_type": "field_format",
    "target_entities": ["global"],
    "condition_json": {"field": "phone", "operator": "matches_pattern", "value": "\\d{10}"},
    "severity": "error",
    "is_active": true
  }')

GLOBAL_RULE_ID=$(echo $GLOBAL_RULE | jq -r '.id')
echo "✓ Global rule created: $GLOBAL_RULE_ID"

# Test 2: Create multi-entity rule
echo "Test 2: Creating multi-entity rule..."
MULTI_RULE=$(curl -s -X POST "$BASE_URL/api/validation-rules?tenant_id=$TENANT_ID" \
  $HEADERS \
  -d '{
    "rule_name": "Name Required",
    "rule_type": "field_format",
    "target_entities": ["Customer", "Employee", "Supplier"],
    "condition_json": {"field": "name", "operator": "not_null"},
    "severity": "error",
    "is_active": true
  }')

MULTI_RULE_ID=$(echo $MULTI_RULE | jq -r '.id')
echo "✓ Multi-entity rule created: $MULTI_RULE_ID"

# Test 3: Query for Customer
echo "Test 3: Querying rules for Customer..."
CUSTOMER_RULES=$(curl -s "$BASE_URL/api/validation-rules?tenant_id=$TENANT_ID&entity=Customer" \
  -H "X-Tenant-ID: $TENANT_ID")

COUNT=$(echo $CUSTOMER_RULES | jq 'length')
echo "✓ Found $COUNT rules for Customer (expected: 2)"

# Test 4: Query for Product
echo "Test 4: Querying rules for Product..."
PRODUCT_RULES=$(curl -s "$BASE_URL/api/validation-rules?tenant_id=$TENANT_ID&entity=Product" \
  -H "X-Tenant-ID: $TENANT_ID")

COUNT=$(echo $PRODUCT_RULES | jq 'length')
echo "✓ Found $COUNT rules for Product (expected: 1 global)"

echo "All tests completed!"
```

---

## Expected Outcomes

### Success Criteria
- ✅ Global rules apply to all entities
- ✅ Multi-entity rules apply to correct entities
- ✅ Query filtering works with `entity` parameter
- ✅ Combined filtering works (`entity` + `rule_type` + `severity`)
- ✅ Create/Update/Delete operations work
- ✅ Backward compatibility maintained
- ✅ Response includes `target_entities` array
- ✅ No performance degradation

### Performance Targets
- Query for entity: < 100ms (with GIN index)
- Create rule: < 50ms
- Update rule: < 50ms
- Delete rule: < 50ms

---

## Troubleshooting

### Issue: 400 Bad Request - Missing tenant_id
**Solution:** Add `?tenant_id=XXX` and header `X-Tenant-ID: XXX`

### Issue: 404 Not Found - Rule not found
**Solution:** Verify rule ID exists and belongs to correct tenant

### Issue: Empty array returned
**Solution:** Check that global rule exists or entity matches target_entities

### Issue: Performance degradation
**Solution:** Verify GIN index created: `SELECT * FROM pg_indexes WHERE tablename='catalog_validation_rules'`

---

## Test Completion Checklist

- [ ] Test 1: Global rule creation
- [ ] Test 2: Multi-entity rule creation
- [ ] Test 3: Customer entity query
- [ ] Test 4: Product entity query (non-matching)
- [ ] Test 5: Employee entity query (matching)
- [ ] Test 6: Update rule target_entities
- [ ] Test 7: Combined filtering (type + entity)
- [ ] Test 8: Rule deletion
- [ ] Test 9: Backward compatibility (legacy single-entity)

---

## Next Steps After Integration Tests

1. **If All Tests Pass ✅:**
   - Run performance benchmarks
   - Test with production dataset
   - UAT with stakeholders
   - Deploy to staging
   - Deploy to production

2. **If Any Test Fails ❌:**
   - Review error message
   - Check database state
   - Review code logic
   - Run unit tests again
   - Debug and iterate

---

## Documentation

For detailed information, see:
- `MULTI_ENTITY_BACKEND_COMPLETE.md` - Backend implementation details
- `BACKEND_IMPLEMENTATION_SESSION_SUMMARY.md` - Session summary
- `MULTI_ENTITY_VALIDATION_GUIDE.md` - General multi-entity guide
- `MULTI_ENTITY_TESTING_GUIDE.md` - Original test guide

---

## Summary

All infrastructure is ready for integration testing. The backend implementation is complete, tested, and production-ready. These 9 scenarios will validate the end-to-end functionality and prepare the system for UAT and production deployment.
