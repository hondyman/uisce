# Testing Guide: Business Term Mapping Enhancements

## ✅ Status: Backend Deployed Successfully

The backend has been rebuilt with all enhancements and is running on `http://localhost:8080`.

## Prerequisites

1. **Tenant Scope Selected** (Required by agents.md)
   - Open Fabric Builder UI
   - Use tenant picker to select:
     - Tenant
     - Product
     - Datasource
   - Verify selection: `localStorage.getItem('selected_tenant')`

2. **Database Access**
   ```bash
   psql postgres://postgres:postgres@host.docker.internal:5432/alpha?sslmode=disable
   ```

## Test Scenarios

### Test 1: Update Business Term (PUT Endpoint)

**Objective**: Verify business term updates work correctly

**Steps**:
1. Get an existing business term ID:
   ```sql
   SELECT id, term_name, description 
   FROM catalog_node 
   WHERE node_type = 'business_term' 
   LIMIT 5;
   ```

2. Test the PUT endpoint:
   ```bash
   curl -X PUT "http://localhost:8080/api/business-terms/{TERM_ID}" \
     -H "Content-Type: application/json" \
     -H "X-Tenant-ID: {TENANT_ID}" \
     -H "X-Tenant-Datasource-ID: {DATASOURCE_ID}" \
     -d '{
       "term_name": "Customer Full Name",
       "description": "The complete name of a customer",
       "category": "Customer Data"
     }'
   ```

**Expected Result**:
- Status: `200 OK`
- Response contains updated term with normalized name (title case, no underscores)
- Database reflects changes

**Validation**:
```sql
SELECT id, term_name, description, qualified_path
FROM catalog_node
WHERE id = '{TERM_ID}';
```

---

### Test 2: Business Term Name Normalization

**Objective**: Verify title case conversion works

**Test Cases**:

| Input                  | Expected Output          |
|------------------------|--------------------------|
| `customer_first_name`  | `Customer First Name`    |
| `accountID`            | `Account ID`             |
| `first-name`           | `First Name`             |
| `CUSTOMER_EMAIL`       | `Customer Email`         |
| `customerFirstNAME`    | `Customer First NAME`    |

**Steps**:
```bash
# Test each case
curl -X PUT "http://localhost:8080/api/business-terms/{TERM_ID}" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: {TENANT_ID}" \
  -H "X-Tenant-Datasource-ID: {DATASOURCE_ID}" \
  -d '{
    "term_name": "customer_first_name"
  }'
```

**Expected Result**: Response shows `"term_name": "Customer First Name"`

---

### Test 3: Confidence Breakdown in Suggestions

**Objective**: Verify confidence breakdown is returned

**Steps**:
1. Get a semantic term ID:
   ```sql
   SELECT id, term_name 
   FROM catalog_node 
   WHERE node_type = 'semantic_term' 
   LIMIT 5;
   ```

2. Request suggestions:
   ```bash
   curl -X GET "http://localhost:8080/api/semantic-terms/{TERM_ID}/suggest-business-terms" \
     -H "X-Tenant-ID: {TENANT_ID}" \
     -H "X-Tenant-Datasource-ID: {DATASOURCE_ID}"
   ```

**Expected Response Structure**:
```json
{
  "suggestions": [
    {
      "business_term_id": "uuid",
      "term_name": "Customer First Name",
      "confidence": 0.87,
      "reason": "Strong name similarity, Good profile alignment",
      "description": "The first name of the customer",
      "categories": ["Customer", "Personal Information"],
      "confidence_breakdown": [
        {
          "label": "Name similarity",
          "score": 0.95,
          "weight": 0.5,
          "details": "Expanded 2 variations, exact match found"
        },
        {
          "label": "Profile alignment",
          "score": 0.75,
          "weight": 0.35,
          "details": "65% value overlap, 80% pattern overlap"
        },
        {
          "label": "Data type alignment",
          "score": 1.0,
          "weight": 0.15,
          "details": "Compatible: VARCHAR"
        }
      ]
    }
  ]
}
```

**Validation**:
- Each suggestion has `confidence_breakdown` array
- Breakdown has 3 components (name, profile, data type)
- Weights sum to 1.0 (0.5 + 0.35 + 0.15)
- Each component has label, score, weight, and details

---

### Test 4: Enhanced Match Endpoint with Breakdown

**Objective**: Verify enhanced matching returns breakdown

**Steps**:
```bash
curl -X POST "http://localhost:8080/api/semantic-mappings/enhanced-match" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: {TENANT_ID}" \
  -H "X-Tenant-Datasource-ID: {DATASOURCE_ID}" \
  -d '{
    "table_name": "customers",
    "column_name": "cust_first_nm",
    "data_type": "VARCHAR(50)",
    "sample_values": ["John", "Jane", "Michael"],
    "existing_business_terms": [
      {
        "node_id": "uuid1",
        "term_name": "Customer First Name"
      },
      {
        "node_id": "uuid2",
        "term_name": "First Name"
      }
    ]
  }'
```

**Expected Response**:
```json
{
  "results": {
    "Customer First Name": {
      "confidence": 0.92,
      "reason": "Strong name similarity with abbreviation expansion",
      "confidence_breakdown": [...]
    },
    "First Name": {
      "confidence": 0.78,
      "reason": "Good name similarity",
      "confidence_breakdown": [...]
    }
  }
}
```

---

### Test 5: Frontend UI Testing

**Objective**: Verify the BusinessTermMapper component works end-to-end

**Steps**:

1. **Navigate to Semantic Mapper**:
   - Open Fabric Builder
   - Select tenant/product/datasource
   - Go to Semantic Mapper page

2. **Test Expandable Rows**:
   - Click a semantic term row
   - Verify it expands to show suggestions
   - Verify collapse works

3. **Test Confidence Visualization**:
   - Look at confidence badges
   - Verify color coding:
     - Red: < 0.3
     - Orange: 0.3 - 0.5
     - Yellow: 0.5 - 0.7
     - Light Green: 0.7 - 0.85
     - Dark Green: > 0.85

4. **Test Accept Suggestion**:
   - Click "Accept" on a suggestion
   - Verify success message
   - Verify edge is created in database:
     ```sql
     SELECT * FROM catalog_edge 
     WHERE source_id = '{SEMANTIC_TERM_ID}' 
     AND target_id = '{BUSINESS_TERM_ID}';
     ```

5. **Test Reject Suggestion**:
   - Click "Reject" on a suggestion
   - Verify it disappears from list
   - (Future: Verify feedback is stored)

6. **Test Custom Term Creation**:
   - Expand a row without suggestions
   - Click "Create New Term"
   - Enter term name (e.g., "test_name")
   - Verify normalized name appears: "Test Name"
   - Submit
   - Verify new term is created:
     ```sql
     SELECT * FROM catalog_node 
     WHERE term_name = 'Test Name' 
     AND node_type = 'business_term';
     ```

7. **Test Edit Existing Term**:
   - Click "Edit" on a mapped term
   - Update description
   - Save
   - Verify update via PUT endpoint was successful

8. **Test Autocomplete Search**:
   - Start typing in "Link Existing" field
   - Verify autocomplete suggestions appear
   - Select a term
   - Verify edge is created

---

### Test 6: Error Handling

**Objective**: Verify proper error responses

**Test Cases**:

1. **Invalid Term ID**:
   ```bash
   curl -X PUT "http://localhost:8080/api/business-terms/invalid-uuid" \
     -H "X-Tenant-ID: {TENANT_ID}"
   ```
   - Expected: `400 Bad Request` - Invalid UUID

2. **Non-existent Term**:
   ```bash
   curl -X PUT "http://localhost:8080/api/business-terms/00000000-0000-0000-0000-000000000000" \
     -H "X-Tenant-ID: {TENANT_ID}"
   ```
   - Expected: `404 Not Found` - Business term not found

3. **Missing Tenant Scope**:
   ```bash
   curl -X PUT "http://localhost:8080/api/business-terms/{TERM_ID}"
   ```
   - Expected: `400 Bad Request` - Missing tenant parameters

4. **Duplicate Term Name** (if unique constraint exists):
   ```bash
   curl -X PUT "http://localhost:8080/api/business-terms/{TERM_ID}" \
     -d '{"term_name": "Existing Term Name"}'
   ```
   - Expected: `409 Conflict` - Term name already exists

---

### Test 7: Performance Testing

**Objective**: Verify confidence calculations are performant

**Steps**:
1. Test with large suggestion set (100+ business terms)
2. Measure response time for enhanced-match endpoint
3. Expected: < 2 seconds for 100 terms

**Benchmark Command**:
```bash
time curl -X POST "http://localhost:8080/api/semantic-mappings/enhanced-match" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: {TENANT_ID}" \
  -d @large_match_request.json
```

---

### Test 8: Database Validation

**Objective**: Verify database integrity after operations

**Queries**:

1. **Check Business Terms**:
   ```sql
   SELECT 
     id, 
     term_name, 
     description, 
     qualified_path,
     node_type
   FROM catalog_node 
   WHERE node_type = 'business_term'
   ORDER BY created_at DESC 
   LIMIT 10;
   ```

2. **Check Semantic-Business Mappings**:
   ```sql
   SELECT 
     e.id,
     sn.term_name as semantic_term,
     bn.term_name as business_term,
     e.edge_type,
     e.created_at
   FROM catalog_edge e
   JOIN catalog_node sn ON e.source_id = sn.id
   JOIN catalog_node bn ON e.target_id = bn.id
   WHERE e.edge_type = 'is_a'
   ORDER BY e.created_at DESC
   LIMIT 10;
   ```

3. **Check for Orphaned Edges**:
   ```sql
   SELECT e.* 
   FROM catalog_edge e
   LEFT JOIN catalog_node sn ON e.source_id = sn.id
   LEFT JOIN catalog_node tn ON e.target_id = tn.id
   WHERE sn.id IS NULL OR tn.id IS NULL;
   ```
   - Expected: 0 rows

4. **Verify Name Normalization**:
   ```sql
   SELECT term_name 
   FROM catalog_node 
   WHERE node_type = 'business_term'
     AND (term_name LIKE '%_%' 
          OR term_name != INITCAP(term_name));
   ```
   - Expected: 0 rows (all normalized)

---

## Integration Test Checklist

- [ ] Backend builds successfully
- [ ] Backend starts without errors
- [ ] PUT /business-terms/{id} endpoint works
- [ ] Business term names are normalized (title case)
- [ ] Confidence breakdown is returned in suggestions
- [ ] Enhanced match endpoint includes breakdown
- [ ] Frontend expandable rows work
- [ ] Confidence colors display correctly
- [ ] Accept suggestion creates edge
- [ ] Reject suggestion removes from list
- [ ] Custom term creation normalizes name
- [ ] Edit existing term updates database
- [ ] Autocomplete search works
- [ ] Error responses are correct
- [ ] Database integrity maintained
- [ ] Performance is acceptable (< 2s for 100 terms)

---

## Common Issues & Solutions

### Issue: "Missing tenant scope"
**Solution**: Ensure tenant picker is used in UI, or add headers:
```bash
-H "X-Tenant-ID: {TENANT_ID}"
-H "X-Tenant-Datasource-ID: {DATASOURCE_ID}"
```

### Issue: "Business term not found"
**Solution**: Verify term exists and belongs to the specified tenant:
```sql
SELECT * FROM catalog_node 
WHERE id = '{TERM_ID}' 
  AND tenant_id = '{TENANT_ID}';
```

### Issue: Confidence always 0
**Solution**: Check if abbreviation service is initialized:
```bash
docker compose logs backend | grep "abbreviation"
```

### Issue: Name not normalized
**Solution**: Verify `normalizeBusinessTermName` is called in `UpdateBusinessTerm`:
```go
businessTermName := normalizeBusinessTermName(req.TermName)
```

### Issue: Breakdown not returned
**Solution**: Check function signature returns 3 values:
```go
confidence, reason, breakdown := s.EnhancedCalculateSemanticConfidence(...)
```

---

## Next Steps

1. **Run all tests** from this guide
2. **Document results** (pass/fail for each test)
3. **Fix any issues** found during testing
4. **Implement remaining features**:
   - Rejection tracking (database table + API)
   - Description/category generation
   - Heat map visualization in UI
   - Confidence breakdown modal
5. **User Acceptance Testing** with real tenant data

---

## Monitoring & Debugging

**Check Backend Logs**:
```bash
docker compose logs backend -f
```

**Check Database Activity**:
```sql
SELECT * FROM pg_stat_activity 
WHERE application_name = 'semlayer-backend';
```

**Profile Performance**:
```bash
curl http://localhost:8080/debug/pprof/profile?seconds=30 > profile.out
go tool pprof profile.out
```

**Database Query Performance**:
```sql
SELECT query, mean_exec_time, calls 
FROM pg_stat_statements 
WHERE query LIKE '%business_term%'
ORDER BY mean_exec_time DESC;
```

---

## Success Criteria

✅ All 8 test scenarios pass  
✅ No database integrity issues  
✅ Response times < 2s  
✅ Error handling is correct  
✅ UI is responsive and intuitive  
✅ Business term names follow naming convention  
✅ Confidence breakdown provides actionable insights  

