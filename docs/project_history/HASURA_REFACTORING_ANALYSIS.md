# Hasura GraphQL Refactoring Analysis

## Executive Summary

Analyzed 100+ SQL query instances across the codebase. Identified high-value services that would benefit from Hasura GraphQL conversion to reduce SQL hardcoding, improve type safety, and enable real-time capabilities.

## ✅ Already Refactored

1. **notifications-service** - 5 operations converted
2. **portfolio-management/backtest** - 2 operations converted  
3. **RDL Service** - 6 CRUD operations converted ✅

## 🎯 Top Priority Candidates (High ROI)

### 1. Portfolio Hierarchy Service ⭐⭐⭐⭐⭐
**Location:** `portfolio-management/backend/internal/hierarchy/service_sqlx.go`

**Why High Priority:**
- **13 methods** with embedded SQL
- Multi-tenant architecture ready for Hasura
- Heavy read operations that would benefit from GraphQL caching
- Complex recursive tree queries that Hasura handles well
- Active development area

**SQL Operations:**
- `ValidateHierarchy()` - SELECT rules with tenant/type filtering
- `GetHierarchyRules()` - SELECT all rules for tenant
- `GetHierarchySummary()` - SELECT from view with aggregations
- `GetEntityHierarchy()` - Complex recursive CTE for tree structure
- `GetHierarchyStats()` - Count queries with aggregations
- `CreateHierarchyRule()` - INSERT with ON CONFLICT
- `UpdateHierarchyRule()` - UPDATE rule
- `DeleteHierarchyRule()` - DELETE rule
- `BulkCreateOperations()` - Multiple INSERTs in transaction
- `LogHierarchyAudit()` - INSERT audit log
- `GetHierarchyAuditLog()` - SELECT audit with LIMIT
- `ImportHierarchyRules()` - Bulk INSERT with transaction
- `ValidateEntityConsistency()` - Count validation

**Database Tables:**
- `entity_hierarchy_rules`
- `entity_relationships`
- `entity_hierarchy_audit_log`
- `v_hierarchy_summary` (view)

**Benefits of Hasura Conversion:**
- GraphQL subscriptions for real-time hierarchy updates
- Relationship traversal without complex joins
- Built-in pagination and filtering
- Type-safe schema for complex nested structures
- Reduce ~300+ lines of SQL code

**Estimated Effort:** 8-10 hours
**Business Impact:** High - Core portfolio management functionality

---

### 2. AI Trade Reconciliation Service ⭐⭐⭐⭐
**Location:** `services/ai-trade-reconciliation/backend/`

**Why High Priority:**
- Multiple components with SQL (handlers, rules, reports, activities)
- Read-heavy operations perfect for GraphQL
- Would benefit from subscriptions for real-time updates
- Complex report generation

**Components:**

#### Handlers (`internal/api/handlers.go`)
- `GetReconciliationResults()` - SELECT with pagination
- `GetLatestResult()` - SELECT latest result
- `GetDiscrepancies()` - SELECT discrepancies
- `ListTasks()` - SELECT tasks
- `UpdateTask()` - UPDATE task status
- `ListRules()` - SELECT rules
- `CreateRule()` - INSERT rule

#### Rules Engine (`internal/rules/rules.go`)
- `GetActiveRules()` - SELECT enabled rules
- `CreateOrUpdateRule()` - INSERT/UPDATE with ON CONFLICT

#### Report Engine (`internal/reports/engine.go`)
- `LoadSemanticViews()` - SELECT semantic content
- `SaveTemplate()` - INSERT report template
- `UpdateSections()` - UPDATE template sections
- `UpdateFilters()` - UPDATE template filters
- `UpdateRules()` - UPDATE template rules
- `TrackGeneration()` - INSERT generation record

#### Activities (`temporal/activities/activities.go`)
- `SaveResult()` - INSERT reconciliation result
- `CreateTask()` - INSERT task
- `LogAudit()` - INSERT audit log

**Database Tables:**
- `reconciliation_results`
- `reconciliation_discrepancies`
- `reconciliation_tasks`
- `reconciliation_rules`
- `reconciliation_audit_logs`
- `report_templates`
- `report_generations`
- `semantic_views`

**Benefits:**
- Real-time reconciliation status updates
- Eliminate ~500+ lines of SQL
- Type-safe report generation
- GraphQL relationships between trades/confirms/discrepancies

**Estimated Effort:** 16-20 hours (multiple services)
**Business Impact:** High - Critical financial reconciliation

---

### 3. Business Process Service ⭐⭐⭐⭐
**Location:** `backend/pkg/bp/service.go`

**Why Priority:**
- Core workflow engine with heavy SQL usage
- Complex transactions that Hasura mutations can simplify
- Many INSERT/UPDATE operations
- Integration with multiple systems

**SQL Operations:**
- `CreateBusinessProcess()` - INSERT process + steps (transaction)
- `UpdateBusinessProcess()` - UPDATE + DELETE + INSERT steps
- `GetBusinessProcessCount()` - Count query
- `ExecuteBusinessProcess()` - INSERT execution record
- `LogAuditTrail()` - INSERT audit
- `SaveFormData()` - INSERT/UPDATE form data

**Additional Files with SQL:**
- `branch_evaluator.go` - Branch execution, join convergences
- `branch_complete_evaluator.go` - Analytics, blockchain audit, explainability
- `branch_advanced_evaluators.go` - Semantic intents, scoring, adaptive triggers
- `trigger_engine.go` - Trigger events

**Database Tables:**
- `business_processes`
- `bp_steps`
- `bp_executions`
- `bp_audit_trail`
- `business_process_form_data`
- `bp_branch_executions`
- `bp_join_convergences`
- `bp_branch_analytics_extended`
- `bp_blockchain_audit`
- `bp_explainability_records`
- (20+ tables total)

**Benefits:**
- GraphQL subscriptions for workflow status
- Simplified complex transactions
- Better audit trail queries
- ~800+ lines of SQL eliminated

**Estimated Effort:** 20-24 hours (complex, multiple files)
**Business Impact:** Very High - Core platform capability

---

### 4. Semantic Engine Mapping Service ⭐⭐⭐
**Location:** `services/semantic-engine/internal/services/semantic_mapping_service.go`

**Why Priority:**
- 2,857 lines - largest service file
- Heavy catalog node/edge operations
- Graph operations perfect for GraphQL
- Many INSERT/SELECT operations

**SQL Operations:**
- Node creation and lookup (multiple methods)
- Edge creation for relationships
- Graph traversal queries
- Semantic term matching
- Bulk operations

**Database Tables:**
- `catalog_node`
- `catalog_edge`
- `node_types`
- `edge_types`

**Benefits:**
- Graph relationships native to GraphQL
- Real-time semantic mapping updates
- Eliminate ~400+ lines of SQL
- Better type safety for graph operations

**Estimated Effort:** 12-16 hours
**Business Impact:** Medium-High - Core semantic layer

---

## 🔶 Medium Priority Candidates

### 5. Backtest Service Extensions ⭐⭐⭐
**Location:** `portfolio-management/backend/internal/backtest/service.go`

**Status:** Partially refactored (GetPortfolio, CreatePortfolio done)

**Remaining Operations:**
- `CreateRecommendation()` - INSERT recommendation
- `GetHistoricalPrice()` - SELECT price data
- `SaveBacktestResult()` - INSERT backtest result
- `CompareBacktests()` - INSERT comparison
- `CalculateRiskMetrics()` - INSERT risk metrics

**Estimated Effort:** 4-6 hours

---

### 6. Notification Service Extensions ⭐⭐
**Location:** `portfolio-management/backend/internal/notifications/service.go`

**Remaining Operations:**
- `GetUserEmail()` - SELECT email
- `GetUserPhone()` - SELECT phone
- `RecordDelivery()` - INSERT delivery
- `GetRetryInfo()` - SELECT retry count

**Estimated Effort:** 2-3 hours

---

### 7. Business Object Service ⭐⭐
**Location:** `backend/pkg/meta/service.go`

**Operations:**
- `CreateBusinessObject()` - INSERT core_bo
- `DeprecateBusinessObject()` - UPDATE status

**Estimated Effort:** 2-3 hours

---

### 8. Fabric Builder Services ⭐⭐
**Location:** `services/fabric-builder/api/business_process_handlers.go`

**Operations:**
- `CreateBusinessProcess()` - INSERT process + steps
- `DeleteBusinessProcess()` - DELETE process
- `CreateInstance()` - INSERT instance

**Estimated Effort:** 3-4 hours

---

## 🔻 Lower Priority (Specialized/Script Usage)

### 9. Compliance/Workflow ABAC
- Single INSERT operation
- Low frequency usage

### 10. UMA Rebalance Service
- Single INSERT operation
- Specialized use case

### 11. AI Routing Feedback Loop
- INSERT for routing decisions and outcomes
- Low volume

### 12. Profiler/Scripts
- One-off scripts and utilities
- Not production critical

---

## 📊 Refactoring Impact Summary

| Service | SQL Operations | Estimated Hours | Business Impact | Priority |
|---------|---------------|-----------------|-----------------|----------|
| Portfolio Hierarchy | 13 | 8-10 | High | ⭐⭐⭐⭐⭐ |
| AI Trade Reconciliation | 15+ | 16-20 | High | ⭐⭐⭐⭐ |
| Business Process | 30+ | 20-24 | Very High | ⭐⭐⭐⭐ |
| Semantic Engine | 20+ | 12-16 | Medium-High | ⭐⭐⭐ |
| Backtest Extensions | 5 | 4-6 | Medium | ⭐⭐⭐ |
| Notification Extensions | 4 | 2-3 | Low | ⭐⭐ |
| Business Object | 2 | 2-3 | Low | ⭐⭐ |
| Fabric Builder | 3 | 3-4 | Medium | ⭐⭐ |

**Total Estimated Effort:** 70-90 hours
**Total SQL Statements to Convert:** 90+

---

## 🎯 Recommended Execution Order

### Phase 1: High-Value Quick Wins (2-3 weeks)
1. ✅ **RDL Service** - COMPLETE
2. **Portfolio Hierarchy Service** - Core functionality, high usage
3. **Backtest Service Extensions** - Complete existing work

### Phase 2: Critical Business Services (3-4 weeks)
4. **AI Trade Reconciliation** - Financial critical path
5. **Business Object Service** - Foundation for other services

### Phase 3: Complex Workflow Systems (4-5 weeks)
6. **Business Process Service** - Complex but high value
7. **Semantic Engine** - Large refactor, enable graph queries

### Phase 4: Enhancement & Polish (1-2 weeks)
8. **Notification Extensions** - Complete notifications work
9. **Fabric Builder** - Polish admin features
10. **Remaining small services** - Cleanup

---

## 🛠️ Implementation Pattern

Based on successful RDL Service refactoring:

```go
// 1. Add HasuraClient interface to service struct
type Service struct {
    db     *sqlx.DB
    hasura HasuraClient  // Add this
}

// 2. Create Hasura-enabled constructor
func NewServiceWithHasura(db *sqlx.DB, hasura HasuraClient) *Service {
    return &Service{db: db, hasura: hasura}
}

// 3. Modify each method: Hasura-first, SQL fallback
func (s *Service) GetData(ctx context.Context, id uuid.UUID) (*Data, error) {
    if s.hasura != nil {
        return s.getDataWithHasura(ctx, id)
    }
    // Original SQL code remains
}

// 4. Implement Hasura GraphQL method
func (s *Service) getDataWithHasura(ctx context.Context, id uuid.UUID) (*Data, error) {
    query := `query GetData($id: uuid!) { ... }`
    result, err := s.hasura.Query(query, map[string]interface{}{"id": id.String()})
    // Parse and return
}

// 5. Add comprehensive tests with mock client
```

---

## 📈 Expected Benefits

### Performance
- **Reduced latency** - Hasura's query planner optimization
- **Caching** - Built-in GraphQL response caching
- **N+1 elimination** - GraphQL relationships prevent over-fetching

### Development
- **Type safety** - GraphQL schema validation
- **Less code** - Eliminate ~2000+ lines of SQL
- **Easier testing** - Mock GraphQL responses simpler than SQL

### Features
- **Real-time** - GraphQL subscriptions for live updates
- **Flexible queries** - Frontend can request exact fields needed
- **Better errors** - GraphQL error standardization

### Maintenance
- **Single source of truth** - Hasura schema as documentation
- **Easier refactoring** - Change schema, not scattered SQL
- **Better debugging** - GraphQL query logs and tracing

---

## ⚠️ Considerations

1. **Database Migrations** - Some tables may need creation/tracking in Hasura
2. **Permissions** - Set up row-level security in Hasura
3. **Testing** - Maintain SQL fallback tests alongside Hasura tests
4. **Gradual rollout** - Feature flags for Hasura vs SQL
5. **Monitoring** - Add Hasura query performance tracking

---

## 🚀 Next Steps

**Immediate Action:** Start with Portfolio Hierarchy Service
- High business value
- Clear bounded context
- 13 methods but similar patterns
- Will establish pattern for other services
