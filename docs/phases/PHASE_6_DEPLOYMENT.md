# Phase 6: UAT & Production Deployment

**Status**: 🟡 **IN PROGRESS**  
**Date**: October 19, 2025  
**Objective**: Prepare multi-entity validation rules system for production deployment

---

## 📋 Pre-Deployment Checklist

### ✅ Development Phase Complete
- [x] Phase 1: Database schema with multi-entity support (target_entities TEXT[] + GIN index)
- [x] Phase 2: Backend API implementation (3 handlers updated, ANY() operator)
- [x] Phase 3: Unit testing (15/15 tests passing)
- [x] Phase 4: Integration testing (9/9 scenarios passing)
- [x] Phase 5: Performance testing (all metrics exceeded targets)

### ⏳ Deployment Phase (Current)

#### 1. Code Review & Staging
- [ ] Peer review of backend changes (validation_rules_routes.go)
- [ ] Peer review of database migration
- [ ] Deploy to staging environment
- [ ] Run full integration test suite in staging
- [ ] Verify GIN index performance in staging

#### 2. User Acceptance Testing (UAT)
- [ ] Create UAT test plan with stakeholders
- [ ] Test global rules (apply to all entities)
- [ ] Test multi-entity rules (1-N entities)
- [ ] Test query filtering by entity
- [ ] Test combined filtering (entity + type)
- [ ] Test rule updates and expansions
- [ ] Test rule deletion and verification
- [ ] Test backward compatibility with existing rules
- [ ] Gather stakeholder feedback
- [ ] Document any issues found

#### 3. Production Deployment
- [ ] Backup production database
- [ ] Run migration: Add target_entities column
- [ ] Create GIN index on target_entities
- [ ] Deploy updated backend code
- [ ] Verify health check endpoint
- [ ] Run smoke tests
- [ ] Monitor error rates and performance
- [ ] Send deployment notification to team

#### 4. Post-Deployment Monitoring
- [ ] Monitor query latency (target: <100ms)
- [ ] Monitor error rates
- [ ] Monitor database connection pool
- [ ] Set up alerts for performance degradation
- [ ] Collect 1 week of production metrics
- [ ] Generate performance report

---

## 🧪 UAT Test Plan

### UAT Test 1: Global Rules
**Scenario**: Create a rule that applies to ALL entities

```bash
curl -X POST "http://production:29080/api/validation-rules?tenant_id=$TENANT_ID" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{
    "rule_name": "Global_PhoneFormat_UAT",
    "rule_type": "field_format",
    "target_entity": "global",
    "target_entities": ["global"],
    "condition_json": {
      "field": "phone",
      "operator": "matches_pattern",
      "value": "\\\\d{10}"
    },
    "severity": "error",
    "is_active": true
  }'
```

**Acceptance Criteria**:
- [ ] Rule created successfully
- [ ] Rule applies to ALL entity queries
- [ ] Query with any entity returns the global rule
- [ ] Performance: <50ms create, <30ms query

---

### UAT Test 2: Multi-Entity Rules
**Scenario**: Create a rule for specific entities (Customer, Employee, Supplier)

```bash
curl -X POST "http://production:29080/api/validation-rules?tenant_id=$TENANT_ID" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{
    "rule_name": "NameRequired_UAT",
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

**Acceptance Criteria**:
- [ ] Rule created with 3 entities
- [ ] Query for Customer returns the rule
- [ ] Query for Employee returns the rule
- [ ] Query for Supplier returns the rule
- [ ] Query for Product does NOT return the rule
- [ ] Performance: <50ms create, <30ms query

---

### UAT Test 3: Query Filtering by Entity
**Scenario**: Verify filtering works with different entities

```bash
# Query Customer
curl "http://production:29080/api/validation-rules?tenant_id=$TENANT_ID&entity=Customer" \
  -H "X-Tenant-ID: $TENANT_ID"

# Query Employee
curl "http://production:29080/api/validation-rules?tenant_id=$TENANT_ID&entity=Employee" \
  -H "X-Tenant-ID: $TENANT_ID"

# Query Product (not in multi-entity rule)
curl "http://production:29080/api/validation-rules?tenant_id=$TENANT_ID&entity=Product" \
  -H "X-Tenant-ID: $TENANT_ID"
```

**Acceptance Criteria**:
- [ ] Customer query returns 2 rules (global + multi-entity)
- [ ] Employee query returns 2 rules (global + multi-entity)
- [ ] Product query returns 1 rule (global only)
- [ ] Multi-entity rule correctly excluded from Product
- [ ] Performance: <30ms per query

---

### UAT Test 4: Combined Filtering
**Scenario**: Filter by entity AND rule type simultaneously

```bash
curl "http://production:29080/api/validation-rules?tenant_id=$TENANT_ID&entity=Customer&rule_type=field_format" \
  -H "X-Tenant-ID: $TENANT_ID"
```

**Acceptance Criteria**:
- [ ] Returns only field_format rules for Customer
- [ ] Filters applied correctly together
- [ ] Performance: <30ms

---

### UAT Test 5: Rule Updates
**Scenario**: Update a multi-entity rule to add more entities

```bash
# Get existing rule ID first
RULE_ID="<from_uat_test_2>"

curl -X PATCH "http://production:29080/api/validation-rules/$RULE_ID?tenant_id=$TENANT_ID" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{
    "rule_name": "NameRequired_UAT",
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

**Acceptance Criteria**:
- [ ] Rule updated to 5 entities
- [ ] All 5 entities now return the rule in queries
- [ ] Previous test entity (Customer) still returns rule
- [ ] New entities (Product, Order) now return rule
- [ ] Performance: <50ms update, <30ms query

---

### UAT Test 6: Backward Compatibility
**Scenario**: Verify legacy single-entity rules still work

```bash
# Create rule without target_entities array (old format)
curl -X POST "http://production:29080/api/validation-rules?tenant_id=$TENANT_ID" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{
    "rule_name": "LegacyRule_UAT",
    "rule_type": "business_logic",
    "target_entity": "Order",
    "condition_json": {
      "field": "total",
      "operator": ">",
      "value": 0
    },
    "severity": "error",
    "is_active": true
  }'
```

**Acceptance Criteria**:
- [ ] Rule created successfully
- [ ] target_entities auto-populated with ["Order"]
- [ ] Query for Order returns the legacy rule
- [ ] Query works with current code
- [ ] Performance: <50ms create, <30ms query

---

## 🚀 Production Deployment Steps

### Step 1: Pre-Deployment Verification
```bash
# Verify all code changes are reviewed and approved
# Verify all tests passing in staging
# Verify database backup is current
# Verify rollback plan is documented
```

### Step 2: Database Migration
```bash
# On production database:

# 1. Verify column doesn't already exist
SELECT column_name FROM information_schema.columns 
WHERE table_name = 'catalog_validation_rules' 
AND column_name = 'target_entities';

# 2. Add column (if not exists)
ALTER TABLE catalog_validation_rules 
ADD COLUMN target_entities TEXT[] DEFAULT ARRAY['global'];

# 3. Create GIN index
CREATE INDEX idx_validation_rules_target_entities 
ON catalog_validation_rules 
USING GIN (target_entities);

# 4. Verify migration
SELECT indexname FROM pg_indexes 
WHERE tablename = 'catalog_validation_rules' 
AND indexname LIKE '%target%';
```

### Step 3: Application Deployment
```bash
# 1. Stop current backend
systemctl stop semlayer-backend

# 2. Deploy new code
# (deployment process specific to your infrastructure)

# 3. Start new backend
systemctl start semlayer-backend

# 4. Verify health
curl http://localhost:29080/health

# 5. Verify API endpoint
curl http://localhost:29080/api/validation-rules \
  -H "X-Tenant-ID: <test_tenant>"
```

### Step 4: Smoke Tests
```bash
# Run quick smoke tests to verify basic functionality

# Test 1: Create rule
curl -X POST "http://production:29080/api/validation-rules?tenant_id=$TENANT_ID" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "Content-Type: application/json" \
  -d '{...}' 
# Expected: 201 Created

# Test 2: Query rules
curl "http://production:29080/api/validation-rules?tenant_id=$TENANT_ID&entity=Customer" \
  -H "X-Tenant-ID: $TENANT_ID"
# Expected: 200 OK with array of rules

# Test 3: Update rule
curl -X PATCH "http://production:29080/api/validation-rules/$RULE_ID?tenant_id=$TENANT_ID" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "Content-Type: application/json" \
  -d '{...}'
# Expected: 200 OK

# Test 4: Delete rule
curl -X DELETE "http://production:29080/api/validation-rules/$RULE_ID?tenant_id=$TENANT_ID" \
  -H "X-Tenant-ID: $TENANT_ID"
# Expected: 204 No Content
```

### Step 5: Performance Verification
```bash
# Monitor key metrics for 5 minutes

# Query latency (should be <100ms)
for i in {1..10}; do
  time curl "http://production:29080/api/validation-rules?tenant_id=$TENANT_ID&entity=Customer" \
    -H "X-Tenant-ID: $TENANT_ID" > /dev/null
done

# Error rates (should be 0)
curl http://production:29080/metrics | grep -i "http.*error"

# Database connection pool (should be healthy)
psql -U admin -d alpha -c "SELECT count(*) FROM pg_stat_activity WHERE datname='alpha';"
```

---

## 📊 Monitoring & Alerts

### Key Metrics to Monitor

1. **Query Latency**
   - Metric: `query_latency_ms`
   - Target: <100ms (p95)
   - Alert: >150ms

2. **Error Rate**
   - Metric: `http_errors_total`
   - Target: <0.1%
   - Alert: >1%

3. **Throughput**
   - Metric: `http_requests_per_sec`
   - Target: >100 req/sec
   - Alert: <50 req/sec

4. **Database Connection Pool**
   - Metric: `db_connections_active`
   - Target: <80% of max
   - Alert: >90%

5. **GIN Index Usage**
   - Metric: `pg_index_scans`
   - Target: High scans on idx_validation_rules_target_entities
   - Alert: If scans are low, index may not be used

### Alert Setup
```yaml
# Example alert configuration (Prometheus/AlertManager format)
alerts:
  - name: ValidationRulesQueryLatency
    expr: query_latency_ms > 150
    duration: 5m
    severity: warning

  - name: ValidationRulesHighErrorRate
    expr: rate(http_errors_total[5m]) > 0.01
    duration: 5m
    severity: critical

  - name: ValidationRulesLowThroughput
    expr: rate(http_requests_total[5m]) < 50
    duration: 5m
    severity: warning
```

---

## 📝 Rollback Plan

If issues occur post-deployment:

### Immediate Rollback (Option 1: Revert to Previous Version)
```bash
# 1. Stop backend
systemctl stop semlayer-backend

# 2. Restore previous binary
cp /backup/semlayer-backend /usr/local/bin/semlayer-backend

# 3. Start backend
systemctl start semlayer-backend

# 4. Verify health
curl http://localhost:29080/health
```

### Database Rollback (If migration caused issues)
```bash
# 1. Drop the new index (keeps data)
DROP INDEX idx_validation_rules_target_entities;

# 2. Keep the column (has default value, backward compatible)
# The column will stay as is - safe to keep
```

### Full Rollback (Nuclear option)
```bash
# 1. Restore database from backup
pg_restore --clean --if-exists -d alpha < /backup/alpha.sql.gz

# 2. Restore previous backend version
cp /backup/semlayer-backend /usr/local/bin/semlayer-backend

# 3. Restart services
systemctl restart semlayer-backend

# 4. Verify
curl http://localhost:29080/health
```

---

## ✅ Sign-Off Checklist

### Development Team
- [ ] All code reviewed and approved
- [ ] All tests passing (unit, integration, performance)
- [ ] Database migration tested in staging
- [ ] Documentation updated
- [ ] Release notes prepared

### QA/Testing Team
- [ ] UAT test plan executed
- [ ] All 6 UAT scenarios passed
- [ ] No regression issues found
- [ ] Performance targets met
- [ ] Edge cases tested

### Operations Team
- [ ] Deployment procedure reviewed
- [ ] Rollback plan tested
- [ ] Monitoring & alerts configured
- [ ] On-call schedule confirmed
- [ ] Communication plan confirmed

### Product/Stakeholders
- [ ] Feature requirements met
- [ ] Backward compatibility verified
- [ ] Performance acceptable
- [ ] Ready for production launch

---

## 📅 Deployment Timeline

| Phase | Duration | Status |
|-------|----------|--------|
| Code Review | 1 day | ⏳ |
| Staging Deployment | 1 day | ⏳ |
| UAT Execution | 2-3 days | ⏳ |
| UAT Sign-Off | 1 day | ⏳ |
| Production Deployment | 1 hour | ⏳ |
| Smoke Testing | 30 min | ⏳ |
| Monitoring (1 week) | 7 days | ⏳ |

**Total**: ~2 weeks from deployment start to full completion

---

## 📞 Escalation Contacts

| Role | Name | Contact |
|------|------|---------|
| Backend Lead | [Name] | [Email/Phone] |
| Database Admin | [Name] | [Email/Phone] |
| QA Lead | [Name] | [Email/Phone] |
| On-Call | [Name] | [Email/Phone] |
| Escalation | [Name] | [Email/Phone] |

---

## 📚 Documentation

### User Documentation
- [ ] Created: How to create multi-entity validation rules
- [ ] Created: How to query rules by entity
- [ ] Created: How to update/delete rules
- [ ] Created: Troubleshooting guide

### Operational Documentation
- [ ] Created: Deployment procedure
- [ ] Created: Rollback procedure
- [ ] Created: Monitoring guide
- [ ] Created: Alert thresholds

### Technical Documentation
- [ ] Created: API documentation (target_entities field)
- [ ] Created: Database schema documentation
- [ ] Created: Query examples
- [ ] Created: Performance notes

---

## 🎯 Success Criteria

**Phase 6 will be complete when**:

✅ All UAT tests pass  
✅ All stakeholders sign off  
✅ Production deployment successful  
✅ Smoke tests pass  
✅ Performance metrics meet targets  
✅ Zero critical issues in first week  
✅ Documentation complete  

---

## 📊 Phase 6 Status

| Task | Status | Owner | Notes |
|------|--------|-------|-------|
| Code Review | ⏳ Not Started | Dev Lead | Awaiting review scheduling |
| Staging Deploy | ⏳ Not Started | Ops | Awaiting code approval |
| UAT Planning | ⏳ Not Started | QA | Awaiting stakeholder input |
| UAT Execution | ⏳ Not Started | QA | Scheduled for [DATE] |
| Prod Deployment | ⏳ Not Started | Ops | Scheduled for [DATE] |
| Post-Deploy Monitoring | ⏳ Not Started | Ops | 1 week monitoring period |

---

**Next Action**: Schedule code review with development team leads

**Status**: 🟡 **IN PROGRESS - Ready for code review stage**
