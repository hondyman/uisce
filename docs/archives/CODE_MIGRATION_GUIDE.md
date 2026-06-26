# Code Update Guide: Consolidate Metrics & DAX Data Sources

**Date:** November 3, 2025  
**Objective:** Update all code and Hasura to query from consolidated `public` schema tables instead of domain-specific schemas

---

## 📋 Overview

After running the consolidation migration, your code needs to change from:
```sql
-- OLD: Query domain-specific schemas
SELECT * FROM banking.metrics_registry
SELECT * FROM capital_markets.dax_functions
```

To:
```sql
-- NEW: Query consolidated public schema
SELECT * FROM public.metrics_registry WHERE schema_domain = 'banking'
SELECT * FROM public.dax_functions WHERE schema_domain = 'capital_markets'
```

---

## 🔄 Code Update Patterns

### Pattern 1: Backend Go - Single Domain Query

**Before:**
```go
// Query metrics from a specific domain schema
var metrics []Metric
err := db.Select(&metrics, `SELECT * FROM banking.metrics_registry ORDER BY node_id`)
```

**After:**
```go
// Query from consolidated table with domain filter
var metrics []Metric
err := db.Select(&metrics, `
    SELECT * FROM public.metrics_registry 
    WHERE schema_domain = $1 
    ORDER BY node_id
`, "banking")
```

---

### Pattern 2: Backend Go - Multi-Domain Query

**Before:**
```go
// Get metrics across multiple domains (tedious)
var allMetrics []Metric
domains := []string{"banking", "retail", "insurance"}
for _, domain := range domains {
    var metrics []Metric
    err := db.Select(&metrics, fmt.Sprintf(`
        SELECT * FROM %s.metrics_registry WHERE category = $1
    `, domain), "performance")
    allMetrics = append(allMetrics, metrics...)
}
```

**After:**
```go
// Much cleaner - single query
var allMetrics []Metric
domains := []string{"banking", "retail", "insurance"}
query, args, err := sqlx.In(`
    SELECT * FROM public.metrics_registry 
    WHERE schema_domain IN (?) 
    AND category = $1
    ORDER BY schema_domain, node_id
`, domains, "performance")
err = db.Select(&allMetrics, query, args...)
```

---

### Pattern 3: Backend Go - DAX Functions

**Before:**
```go
// Query DAX functions from domain schema
var functions []DAXFunction
err := db.Select(&functions, `SELECT * FROM banking.dax_functions`)
```

**After:**
```go
// Query from consolidated table
var functions []DAXFunction
err := db.Select(&functions, `
    SELECT * FROM public.dax_functions 
    WHERE schema_domain = $1 
    ORDER BY name
`, "banking")
```

---

### Pattern 4: Frontend TypeScript - API Calls

**Before:**
```typescript
// Loading from individual domain schemas
async function loadMetrics(domain: string) {
  const response = await fetch(`/api/bundles/${domain}`);
  const bundle = await response.json();
  return bundle.metrics;
}
```

**After:**
```typescript
// Still use same endpoint structure, but backend queries consolidated table
async function loadMetrics(domain: string) {
  const response = await fetch(`/api/bundles/${domain}`);
  const bundle = await response.json();
  // Backend now filters from public.metrics_registry
  return bundle.metrics;
}
```

---

## 🗄️ Hasura Configuration

### If Using Hasura Metadata

**Step 1: Update or Create Hasura Metadata**

Create/update `hasura/metadata/tables/public_metrics_registry.yaml`:

```yaml
table:
  schema: public
  name: metrics_registry

object_relationships:
  - name: by_domain
    using:
      foreign_key_constraint_on: schema_domain

select_permissions:
  - role: user
    permission:
      columns:
        - id
        - node_id
        - schema_domain
        - category
        - description
        - formula_type
        - formula
        - arguments
        - badge
        - function_class
        - functions_used
        - governance_status
        - audience
        - tags
        - created_at
        - updated_at
      filter:
        schema_domain:
          _eq: X-Hasura-Domain

  - role: admin
    permission:
      columns: "*"
      filter: {}

array_relationships:
  - name: by_domain_functions
    using:
      manual_configuration:
        remote_table:
          schema: public
          name: dax_functions
        column_mapping:
          schema_domain: schema_domain
```

Create/update `hasura/metadata/tables/public_dax_functions.yaml`:

```yaml
table:
  schema: public
  name: dax_functions

select_permissions:
  - role: user
    permission:
      columns:
        - id
        - name
        - schema_domain
        - class
        - badge
        - description
        - created_at
      filter:
        schema_domain:
          _eq: X-Hasura-Domain

  - role: admin
    permission:
      columns: "*"
      filter: {}
```

**Step 2: Create GraphQL Queries**

`hasura/metadata/query_collections.yaml`:

```yaml
queries:
  - name: GetMetricsByDomain
    query: |
      query GetMetricsByDomain($domain: String!) {
        public_metrics_registry(
          where: { schema_domain: { _eq: $domain } }
          order_by: { node_id: asc }
        ) {
          id
          node_id
          schema_domain
          category
          description
          formula_type
          formula
          arguments
          badge
          function_class
          functions_used
          governance_status
          audience
          tags
          created_at
          updated_at
        }
      }

  - name: GetDAXFunctionsByDomain
    query: |
      query GetDAXFunctionsByDomain($domain: String!) {
        public_dax_functions(
          where: { schema_domain: { _eq: $domain } }
          order_by: { name: asc }
        ) {
          id
          name
          schema_domain
          class
          badge
          description
          created_at
        }
      }

  - name: GetAllMetrics
    query: |
      query GetAllMetrics($domains: [String!]!) {
        public_metrics_registry(
          where: { schema_domain: { _in: $domains } }
          order_by: { schema_domain: asc, node_id: asc }
        ) {
          id
          node_id
          schema_domain
          category
          description
          formula_type
          formula
          created_at
        }
      }
```

---

## 🔧 Files to Update

Create a comprehensive list by running:

```bash
bash find_schema_references.sh > /tmp/files_to_update.txt
cat /tmp/files_to_update.txt
```

Common files to update:

1. **Backend API Handlers**
   - `backend/internal/api/bundles_routes.go`
   - Any bundle loading/fetching handlers
   - DAX function endpoints

2. **Backend Services**
   - Services that load metrics from database
   - Services that fetch DAX functions
   - Any database access layer

3. **Frontend API Calls**
   - Components that fetch bundles
   - DAX function reference components
   - Metric viewers

4. **Database Migrations**
   - Any new migration scripts
   - Seed scripts that populate metrics
   - Test fixtures

5. **Configuration Files**
   - Hasura metadata (if using Hasura)
   - GraphQL schema definitions
   - API documentation

---

## 📝 Step-by-Step Implementation

### Step 1: Create Backup Script (optional but recommended)

```bash
#!/bin/bash
# backup_code_changes.sh

git stash
git checkout -b feature/consolidate-metrics-sources
git stash pop
```

### Step 2: Update Backend Database Layer

**File:** `backend/internal/services/metrics_service.go` (create if needed)

```go
package services

import (
	"context"
	"github.com/jmoiron/sqlx"
)

type MetricsService struct {
	db *sqlx.DB
}

// GetMetricsByDomain retrieves all metrics for a specific domain
func (s *MetricsService) GetMetricsByDomain(ctx context.Context, domain string) ([]Metric, error) {
	var metrics []Metric
	err := s.db.SelectContext(ctx, &metrics, `
		SELECT 
			id, node_id, schema_domain, category, description, 
			formula_type, formula, arguments, badge, function_class, 
			functions_used, governance_status, audience, tags, 
			created_at, updated_at
		FROM public.metrics_registry
		WHERE schema_domain = $1
		ORDER BY node_id
	`, domain)
	return metrics, err
}

// GetMetricsByDomains retrieves metrics from multiple domains
func (s *MetricsService) GetMetricsByDomains(ctx context.Context, domains []string) ([]Metric, error) {
	var metrics []Metric
	query, args, err := sqlx.In(`
		SELECT 
			id, node_id, schema_domain, category, description, 
			formula_type, formula, arguments, badge, function_class, 
			functions_used, governance_status, audience, tags, 
			created_at, updated_at
		FROM public.metrics_registry
		WHERE schema_domain IN (?)
		ORDER BY schema_domain, node_id
	`, domains)
	if err != nil {
		return nil, err
	}
	query = s.db.Rebind(query)
	err = s.db.SelectContext(ctx, &metrics, query, args...)
	return metrics, err
}

// GetMetricByNodeID retrieves a specific metric by node_id and domain
func (s *MetricsService) GetMetricByNodeID(ctx context.Context, domain, nodeID string) (*Metric, error) {
	var metric Metric
	err := s.db.GetContext(ctx, &metric, `
		SELECT 
			id, node_id, schema_domain, category, description, 
			formula_type, formula, arguments, badge, function_class, 
			functions_used, governance_status, audience, tags, 
			created_at, updated_at
		FROM public.metrics_registry
		WHERE schema_domain = $1 AND node_id = $2
	`, domain, nodeID)
	return &metric, err
}
```

**File:** `backend/internal/services/dax_functions_service.go` (create if needed)

```go
package services

import (
	"context"
)

type DAXFunctionsService struct {
	db *sqlx.DB
}

// GetDAXFunctionsByDomain retrieves all DAX functions for a domain
func (s *DAXFunctionsService) GetDAXFunctionsByDomain(ctx context.Context, domain string) ([]DAXFunction, error) {
	var functions []DAXFunction
	err := s.db.SelectContext(ctx, &functions, `
		SELECT 
			id, name, schema_domain, class, badge, description, created_at
		FROM public.dax_functions
		WHERE schema_domain = $1
		ORDER BY name
	`, domain)
	return functions, err
}

// GetDAXFunctionsByDomains retrieves DAX functions from multiple domains
func (s *DAXFunctionsService) GetDAXFunctionsByDomains(ctx context.Context, domains []string) ([]DAXFunction, error) {
	var functions []DAXFunction
	query, args, err := sqlx.In(`
		SELECT 
			id, name, schema_domain, class, badge, description, created_at
		FROM public.dax_functions
		WHERE schema_domain IN (?)
		ORDER BY schema_domain, name
	`, domains)
	if err != nil {
		return nil, err
	}
	query = s.db.Rebind(query)
	err = s.db.SelectContext(ctx, &functions, query, args...)
	return functions, err
}

// GetDAXFunctionByName retrieves a specific function
func (s *DAXFunctionsService) GetDAXFunctionByName(ctx context.Context, domain, name string) (*DAXFunction, error) {
	var fn DAXFunction
	err := s.db.GetContext(ctx, &fn, `
		SELECT 
			id, name, schema_domain, class, badge, description, created_at
		FROM public.dax_functions
		WHERE schema_domain = $1 AND name = $2
	`, domain, name)
	return &fn, err
}
```

### Step 3: Update API Endpoints

**Pattern for Bundle Handler:**

```go
// Before - querying domain schema
func (h *BundleHandler) GetBundle(w http.ResponseWriter, r *http.Request) {
	domain := chi.URLParam(r, "domain")
	
	var metrics []Metric
	err := h.db.Select(&metrics, fmt.Sprintf(`
		SELECT * FROM %s.metrics_registry
	`, domain))
	// ...
}

// After - querying consolidated table
func (h *BundleHandler) GetBundle(w http.ResponseWriter, r *http.Request) {
	domain := chi.URLParam(r, "domain")
	
	var metrics []Metric
	err := h.db.Select(&metrics, `
		SELECT * FROM public.metrics_registry 
		WHERE schema_domain = $1
	`, domain)
	// ...
}
```

### Step 4: Update Frontend Components

**Pattern for React/TypeScript:**

```typescript
// Before
async function loadBundleMetrics(domain: string) {
	const response = await fetch(`/api/metrics?domain=${domain}`);
	return response.json();
}

// After (API stays same, backend query changes)
// The endpoint doesn't change, just the database query behind it
async function loadBundleMetrics(domain: string) {
	const response = await fetch(`/api/metrics?domain=${domain}`);
	return response.json();
}
```

---

## 🧪 Testing Strategy

### Unit Tests

```go
func TestGetMetricsByDomain(t *testing.T) {
	// Arrange
	db := setupTestDB()
	defer db.Close()
	service := NewMetricsService(db)
	
	// Act
	metrics, err := service.GetMetricsByDomain(context.Background(), "banking")
	
	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, metrics)
	for _, m := range metrics {
		assert.Equal(t, "banking", m.SchemaDomain)
	}
}

func TestGetMetricsByDomains(t *testing.T) {
	// Multi-domain query test
	db := setupTestDB()
	defer db.Close()
	service := NewMetricsService(db)
	
	metrics, err := service.GetMetricsByDomains(
		context.Background(), 
		[]string{"banking", "retail"},
	)
	
	assert.NoError(t, err)
	domainMap := make(map[string]int)
	for _, m := range metrics {
		domainMap[m.SchemaDomain]++
	}
	assert.Greater(t, domainMap["banking"], 0)
	assert.Greater(t, domainMap["retail"], 0)
}
```

### Integration Tests

```bash
# Test the consolidated tables exist and have data
psql -h localhost -U postgres -d alpha << 'EOF'
SELECT COUNT(*) as metrics_count FROM public.metrics_registry;
SELECT COUNT(*) as dax_count FROM public.dax_functions;
SELECT COUNT(DISTINCT schema_domain) as domains FROM public.metrics_registry;
EOF
```

---

## ✅ Verification Checklist

After updating code:

- [ ] All `SELECT * FROM {domain}.metrics_registry` updated
- [ ] All `SELECT * FROM {domain}.dax_functions` updated
- [ ] Queries use `WHERE schema_domain = $1` or `WHERE schema_domain IN (...)`
- [ ] Multi-domain queries use `sqlx.In()` properly
- [ ] Tests pass with new queries
- [ ] Hasura metadata updated (if applicable)
- [ ] GraphQL queries point to `public_metrics_registry` table
- [ ] Frontend tests pass
- [ ] Integration tests pass
- [ ] Code reviewed
- [ ] Ready for deployment

---

## 🚀 Deployment Steps

1. **Code Review**
   ```bash
   git push origin feature/consolidate-metrics-sources
   # Create PR for review
   ```

2. **Run Tests**
   ```bash
   go test ./...
   npm test
   ```

3. **Stage Deployment**
   - Deploy to staging environment
   - Run smoke tests
   - Verify data integrity

4. **Production Deployment**
   - Run migration first (see STEP_BY_STEP_IMPLEMENTATION.md Step 4)
   - Deploy code
   - Monitor logs
   - Verify metrics loading

---

## 🔄 Rollback Plan

If issues occur:

1. **Code Rollback**
   ```bash
   git revert <commit-hash>
   ```

2. **Database Rollback**
   ```bash
   # Restore from backup
   pg_restore -h localhost -U postgres -d alpha alpha_backup_*.dump
   ```

3. **Hasura Rollback**
   - Revert metadata changes
   - Restart Hasura

---

## 📊 Query Performance Notes

The consolidated table with indexes on `schema_domain` should perform as well or better than before:

```sql
-- Performance comparison
EXPLAIN ANALYZE
SELECT * FROM public.metrics_registry 
WHERE schema_domain = 'banking' 
AND category = 'performance';

-- vs old way (simulating old queries)
EXPLAIN ANALYZE
SELECT * FROM banking.metrics_registry 
WHERE category = 'performance';
```

Both should use indexes efficiently.

---

## 📞 Support

Refer to these documents for additional help:

1. **CONSOLIDATION_PLAN.md** - Architecture overview
2. **STEP_BY_STEP_IMPLEMENTATION.md** - Database migration steps
3. **QUICK_REFERENCE.md** - SQL patterns and commands
4. **VISUAL_GUIDE.md** - Data flow diagrams

