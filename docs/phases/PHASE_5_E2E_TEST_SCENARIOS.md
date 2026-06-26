# Phase 5: E2E Testing Scenarios

## End-to-End Test Suite

This document outlines comprehensive E2E test scenarios for the "Add Relationship" feature.

---

## Test Scenario 1: Complete Relationship Discovery Workflow

**Title:** User discovers and applies a direct foreign key relationship

**Preconditions:**
- System is running with test database
- User is authenticated with valid tenant
- Two entities exist: "Customers" and "Orders"

**Steps:**

1. User navigates to Relationship Discovery Modal
   - Expected: Modal loads with entity selector
   - Verification: Modal visible with "Discover Relationships" button

2. User selects base entity "Customers"
   - Expected: Entity selected, discovery options shown
   - Verification: Base entity selector shows "Customers"

3. User clicks "Discover Relationships"
   - Expected: System discovers direct FK relationship to "Orders"
   - Verification: Direct relationship tab shows "Orders" with confidence 0.95

4. User reviews confidence score
   - Expected: Green confidence badge (≥0.8)
   - Verification: Badge color matches confidence level

5. User clicks "Apply Relationship"
   - Expected: Relationship saved to database
   - Verification: Success message displayed, relationship persisted

**Postconditions:**
- Relationship saved to `entity_relationship` table
- Audit trail created
- Semantic model regeneration triggered

---

## Test Scenario 2: Multi-Hop Path Discovery

**Title:** User discovers multi-hop relationship paths

**Preconditions:**
- Three entities with FK chains: Customers → Orders → LineItems → Products
- MaxHopDepth set to 3

**Steps:**

1. User opens Relationship Discovery Modal
2. User selects base entity "Customers"
3. User enables "Include Multi-Hop Paths"
4. User sets MaxHopDepth to 3
5. User clicks "Discover Relationships"

**Expected Results:**
- Direct relationships shown in "Direct Relationships" tab
- Multi-hop paths (depth 2-3) shown in "Multi-Hop Paths" tab
- Each path displays hop-by-hop breakdown
- Confidence score decreases with each hop

**Verification:**
- Path visualization shows all hops
- Cardinality correctly calculated (1:N × 1:N = 1:M)
- Confidence = confidence1 × confidence2 × confidence3

---

## Test Scenario 3: Self-Service Report Building

**Title:** User builds and executes a multi-entity report

**Preconditions:**
- Relationships between Customers, Orders, Products exist
- Report builder interface accessible
- SQL generation service running

**Steps:**

1. User opens Report Builder
2. User selects base entity "Customers"
3. User multi-selects related entities: Orders, Products
4. User adds metric: SUM(Orders.Amount) with alias "Total Sales"
5. User adds dimension: Customers.Region
6. User adds filter: Orders.Date > 2024-01-01
7. User clicks "Generate SQL"

**Expected Results:**
- SQL query generated correctly with multi-table JOIN
- Query includes WHERE clause for date filter
- GROUP BY on region dimension
- SUM aggregation on amount metric

**Verification:**
- SQL preview modal shows correct query
- SQL is valid PostgreSQL syntax
- Column aliases match user input

**Steps (continued):**

8. User clicks "Execute Report"
9. System executes query and displays results

**Expected Results:**
- Result table shows 3 columns: Region, Total Sales, Count
- Rows grouped by region
- Pagination set to 20 rows/page

**Verification:**
- Results table populated with correct data
- Pagination controls functional
- Row count accurate

---

## Test Scenario 4: Model Regeneration on Relationship Change

**Title:** Semantic model regenerates when relationship is applied

**Preconditions:**
- Relationship discovery complete
- Model regeneration service running

**Steps:**

1. User applies a new relationship
2. Database trigger fires
3. Model regeneration job queued
4. Model regeneration executed

**Expected Results:**
- New model signature generated
- Version incremented
- Entity attribute mappings updated

**Verification:**
- Check `model_version_history` table for new entry
- Model signature SHA256 updated
- Version number incremented from previous

---

## Test Scenario 5: Multi-Tenant Isolation

**Title:** Tenant B cannot see Tenant A's relationships

**Preconditions:**
- Two tenants created: A and B
- Tenant A has relationship applied
- Tenant B accessing same entity

**Steps:**

1. Tenant A discovers relationship for "Customers" → "Orders"
2. Tenant A applies relationship
3. Tenant B calls same endpoint with same entity ID
4. Tenant B requests are scoped to Tenant B's datasource

**Expected Results:**
- Tenant A sees their discovered relationship
- Tenant B does not see Tenant A's relationship
- Query results isolated by X-Tenant-ID header

**Verification:**
- Query scoped by WHERE tenant_id = ?
- Headers validated on every request
- No cross-tenant data leakage

---

## Test Scenario 6: Error Handling - Missing Tenant Context

**Title:** Request without tenant context returns 400

**Preconditions:**
- API endpoint accessible
- Test client ready

**Steps:**

1. Client sends POST /api/relationships/discover
2. No X-Tenant-ID header included
3. Request processed

**Expected Results:**
- Response status: 400 Bad Request
- Error message: "X-Tenant-ID and X-Tenant-Datasource-ID headers required"
- No processing attempted

**Verification:**
- Status code = 400
- Error logged for audit
- No database modifications

---

## Test Scenario 7: Error Handling - Invalid Confidence Score

**Title:** Relationship with confidence < 0.5 is rejected

**Preconditions:**
- Discovery complete
- System configured to reject low-confidence relationships

**Steps:**

1. System discovers relationship with confidence 0.3
2. User sees red confidence badge
3. User attempts to apply relationship

**Expected Results:**
- Warning dialog shown
- User confirms or cancels
- If confirmed, relationship applied with warning

**Verification:**
- Low confidence relationships highlighted
- User explicitly confirms risky actions
- Audit trail shows user awareness

---

## Test Scenario 8: Performance - Large Dataset

**Title:** System handles relationship discovery with 1000+ entities

**Preconditions:**
- Database populated with 1000+ entities
- Performance benchmarks set

**Steps:**

1. User initiates discovery on large dataset
2. System processes discovery with max hops 3
3. Results displayed with pagination

**Expected Results:**
- Discovery completes within 5 seconds
- Results paginated efficiently
- UI responsive during processing

**Verification:**
- API response time < 5s
- Memory usage < 512MB
- Database queries optimized with indexes

---

## Test Scenario 9: Edge Case - Circular Relationship

**Title:** System handles circular entity relationships

**Preconditions:**
- Entities with circular FK: A → B → A

**Steps:**

1. User initiates discovery from entity A
2. System detects circular path at depth 2
3. Results returned with cycle detection

**Expected Results:**
- Circular path detected and marked
- Path exploration stops at cycle
- No infinite loops

**Verification:**
- Circular path not repeated infinitely
- Performance unaffected
- Audit log shows cycle detection

---

## Test Scenario 10: Data Validation - Invalid Input

**Title:** API rejects invalid request payloads

**Test Cases:**

### 10a: Missing Required Fields
- Request: `/api/relationships/discover` without entity_attribute_id
- Expected: 400 Bad Request with message "entity_attribute_id required"

### 10b: Invalid JSON
- Request: `/api/relationships/apply` with malformed JSON
- Expected: 400 Bad Request with message "invalid JSON"

### 10c: Invalid Confidence Score
- Request: Confidence value > 1.0 or < 0.0
- Expected: 400 Bad Request with validation error

### 10d: Invalid Link Type
- Request: Link type not in enum [DIRECT_FK, SEMANTIC, MULTI_HOP]
- Expected: 400 Bad Request with validation error

**Verification:**
- All invalid inputs rejected with 400
- Error messages are helpful
- No database corruption

---

## Integration Test Matrix

| Component | Integration | Test Type | Status |
|-----------|------------|-----------|--------|
| Frontend Modal | Backend API | E2E | ✅ |
| Frontend Hooks | API Endpoints | Unit | ✅ |
| Backend Handlers | Database | Integration | ✅ |
| Database Triggers | Model Regeneration | Integration | ✅ |
| Multi-Tenant | All Layers | System | ✅ |
| Error Handling | All Layers | System | ✅ |
| Performance | Large Dataset | Load | ✅ |

---

## Test Execution Checklist

- [ ] All unit tests passing (backend)
- [ ] All unit tests passing (frontend)
- [ ] All integration tests passing
- [ ] All E2E scenarios verified
- [ ] Performance benchmarks met
- [ ] Multi-tenant isolation verified
- [ ] Error handling validated
- [ ] Edge cases covered
- [ ] Security audit complete
- [ ] Documentation reviewed

---

## Performance Benchmarks

| Metric | Target | Threshold |
|--------|--------|-----------|
| Relationship Discovery | < 2s | 5s |
| Multi-Hop Discovery (5 hops) | < 5s | 10s |
| Report Generation | < 3s | 10s |
| Report Execution (100 rows) | < 2s | 10s |
| Model Regeneration | < 5s | 15s |
| API Response Time (p95) | < 500ms | 2s |
| Database Query Time (p95) | < 100ms | 500ms |

---

## Security Checklist

- [ ] SQL injection prevention verified
- [ ] Multi-tenant isolation verified
- [ ] Input validation complete
- [ ] Error messages don't leak sensitive data
- [ ] Authentication required on all endpoints
- [ ] Authorization checks in place
- [ ] Audit trail complete
- [ ] Rate limiting configured

---

## Sign-Off

- [ ] QA Lead: _______________
- [ ] Dev Lead: _______________
- [ ] Product Owner: _______________
- [ ] Date: _______________

---

**All E2E tests should pass before Phase 6 (Deployment)**
