# Schema Consolidation Plan: Metrics Registry & DAX Functions

**Date:** November 3, 2025  
**Target Database:** `alpha` (localhost:5432)  
**Status:** Ready for Implementation

---

## Executive Summary

Your `alpha` database contains 17 domain-specific schemas, each with duplicate `metrics_registry` and `dax_functions` tables. This consolidation plan moves all these tables into the `public` schema for a single source of truth while preserving domain context via a `schema_domain` column.

### Benefits
- **Reduced Duplication:** Eliminates 12 copies of metrics_registry & 8 copies of dax_functions
- **Simplified Management:** One consolidated table per type instead of 12-17 copies
- **Backwards Compatibility:** Optional views allow existing code to continue working
- **Better Governance:** Centralized access control and audit trails
- **Easier Maintenance:** Single point of update for metrics and functions

---

## Current State Analysis

### Domain Schemas (17 total)
```
1.  banking
2.  capital_markets
3.  currency_fx
4.  financial_services
5.  fixed_income
6.  foffice
7.  hdb_catalog
8.  healthcare
9.  hld
10. insurance
11. investment_accounting
12. regulatory
13. report_sys
14. retail
15. semantic_layer
16. sml
17. unified_financial_services
18. wealth_management
```

### Duplicated Tables Found

#### `metrics_registry` (12 schemas, 264 total records)
| Schema | Record Count |
|--------|--------------|
| banking | 10 |
| capital_markets | 10 |
| currency_fx | 11 |
| financial_services | 60 |
| fixed_income | 9 |
| healthcare | 10 |
| insurance | 10 |
| investment_accounting | 16 |
| regulatory | 10 |
| retail | 10 |
| unified_financial_services | 82 |
| wealth_management | 26 |
| **TOTAL** | **264** |

#### `dax_functions` (8 schemas)
- banking
- capital_markets
- financial_services
- healthcare
- insurance
- regulatory
- retail
- unified_financial_services

### Schema Structure

**metrics_registry columns:**
- `node_id` (PK, VARCHAR 255)
- `category` (VARCHAR 100)
- `description` (TEXT)
- `formula_type` (VARCHAR 50)
- `formula` (TEXT)
- `arguments` (JSONB)
- `badge` (VARCHAR 10)
- `function_class` (VARCHAR 50)
- `functions_used` (TEXT[])
- `governance_status` (VARCHAR 50, default 'draft')
- `audience` (TEXT[])
- `tags` (TEXT[])
- `created_at` (TIMESTAMP)
- `updated_at` (TIMESTAMP)

**dax_functions columns:**
- `name` (PK, VARCHAR 100)
- `class` (VARCHAR 50)
- `badge` (VARCHAR 10)
- `description` (TEXT)
- `created_at` (TIMESTAMP)

---

## Implementation Plan

### Phase 1: Prepare (No Data Changes)
1. Review the migration script: `migrations/consolidate_metrics_and_dax.sql`
2. Test on a staging/dev environment (recommended)
3. Backup the `alpha` database
4. Document any custom application logic that queries these tables

### Phase 2: Execute Migration
**File:** `migrations/consolidate_metrics_and_dax.sql`

The migration script:
- ✅ Creates consolidated `public.metrics_registry` table with `schema_domain` column
- ✅ Creates consolidated `public.dax_functions` table with `schema_domain` column
- ✅ Inserts all data from each domain schema (with deduplication via `ON CONFLICT DO NOTHING`)
- ✅ Creates performance indexes
- ✅ Includes verification queries
- ✅ Can be re-run safely (idempotent)

**Expected Migration Time:** < 1 second

**To execute:**
```bash
psql -h localhost -U postgres -d alpha -f migrations/consolidate_metrics_and_dax.sql
```

### Phase 3: Verification
After running the migration:

```sql
-- Verify consolidated tables exist
SELECT * FROM public.metrics_registry LIMIT 5;
SELECT * FROM public.dax_functions LIMIT 5;

-- Check record counts by domain
SELECT schema_domain, COUNT(*) FROM public.metrics_registry GROUP BY schema_domain;
SELECT schema_domain, COUNT(*) FROM public.dax_functions GROUP BY schema_domain;

-- Confirm all records migrated
SELECT COUNT(*) FROM public.metrics_registry;  -- Should be 264
SELECT COUNT(*) FROM public.dax_functions;     -- Should match total from all schemas
```

### Phase 4: Update Application Code

Locate all queries that reference domain-specific tables:

```sql
-- OLD (domain-specific)
SELECT * FROM banking.metrics_registry WHERE node_id = 'X';
SELECT * FROM capital_markets.dax_functions WHERE name = 'Y';

-- NEW (consolidated)
SELECT * FROM public.metrics_registry 
WHERE node_id = 'X' AND schema_domain = 'banking';

SELECT * FROM public.dax_functions 
WHERE name = 'Y' AND schema_domain = 'capital_markets';
```

**Files likely needing updates:**
- `backend/internal/api/` - API endpoints
- `backend/internal/services/` - Business logic
- `migrations/` - Any migration scripts
- Application configuration files

### Phase 5: Create Backwards Compatibility Views (Optional)

If you have many services to update at once, create views in each domain schema:

```sql
CREATE VIEW banking.metrics_registry AS
SELECT node_id, category, description, formula_type, formula, arguments,
       badge, function_class, functions_used, governance_status, audience, tags,
       created_at, updated_at
FROM public.metrics_registry
WHERE schema_domain = 'banking';
```

This allows existing code to work unchanged while you migrate services incrementally.

### Phase 6: Cleanup (After All Code Updated)

Once all application code is updated to use the consolidated tables:

1. **Drop views** (if created in Phase 5)
   ```sql
   DROP VIEW IF EXISTS banking.metrics_registry_view CASCADE;
   DROP VIEW IF EXISTS capital_markets.metrics_registry_view CASCADE;
   -- ... repeat for all schemas
   ```

2. **Drop original tables** (verify no code still references them first!)
   ```sql
   DROP TABLE IF EXISTS banking.metrics_registry CASCADE;
   DROP TABLE IF EXISTS banking.dax_functions CASCADE;
   -- ... repeat for all domain schemas
   ```

3. **Drop empty schemas** (if domain schemas no longer serve a purpose)
   ```sql
   DROP SCHEMA IF EXISTS banking CASCADE;
   -- ... repeat as needed
   ```

---

## Consolidated Table Structures

### `public.metrics_registry`

```sql
CREATE TABLE public.metrics_registry (
    id SERIAL PRIMARY KEY,
    node_id VARCHAR(255) NOT NULL,
    schema_domain VARCHAR(100) NOT NULL,  -- NEW: tracks which domain
    category VARCHAR(100) NOT NULL,
    description TEXT,
    formula_type VARCHAR(50) NOT NULL,
    formula TEXT NOT NULL,
    arguments JSONB,
    badge VARCHAR(10),
    function_class VARCHAR(50),
    functions_used TEXT[],
    governance_status VARCHAR(50) DEFAULT 'draft',
    audience TEXT[],
    tags TEXT[],
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(node_id, schema_domain),
    FOREIGN KEY(schema_domain) REFERENCES public.domains(domain_name)  -- optional
);

CREATE INDEX idx_metrics_registry_schema_domain ON public.metrics_registry(schema_domain);
CREATE INDEX idx_metrics_registry_node_id ON public.metrics_registry(node_id);
CREATE INDEX idx_metrics_registry_category ON public.metrics_registry(category);
```

### `public.dax_functions`

```sql
CREATE TABLE public.dax_functions (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    schema_domain VARCHAR(100) NOT NULL,  -- NEW: tracks which domain
    class VARCHAR(50) NOT NULL,
    badge VARCHAR(10),
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(name, schema_domain)
);

CREATE INDEX idx_dax_functions_schema_domain ON public.dax_functions(schema_domain);
CREATE INDEX idx_dax_functions_name ON public.dax_functions(name);
```

---

## Migration Query Examples

### Before (Domain-Specific)
```sql
-- Get all metrics for a specific domain
SELECT * FROM banking.metrics_registry ORDER BY created_at DESC;

-- Get a specific function
SELECT * FROM financial_services.dax_functions WHERE name = 'SUM_BALANCE';

-- Get metrics by category across domains
SELECT * FROM banking.metrics_registry WHERE category = 'performance';
SELECT * FROM retail.metrics_registry WHERE category = 'performance';
-- ... repeat for each domain
```

### After (Consolidated)
```sql
-- Get all metrics for a specific domain
SELECT * FROM public.metrics_registry 
WHERE schema_domain = 'banking' 
ORDER BY created_at DESC;

-- Get a specific function
SELECT * FROM public.dax_functions 
WHERE name = 'SUM_BALANCE' AND schema_domain = 'financial_services';

-- Get metrics by category across domains
SELECT * FROM public.metrics_registry WHERE category = 'performance';

-- Get metrics from multiple domains
SELECT * FROM public.metrics_registry 
WHERE schema_domain IN ('banking', 'retail') 
AND category = 'performance';

-- Get all unique metrics across all domains
SELECT DISTINCT node_id, formula_type 
FROM public.metrics_registry 
ORDER BY node_id;
```

---

## Rollback Plan

If you need to rollback after migration:

```sql
-- Option 1: Keep original tables, but empty consolidated tables
TRUNCATE public.metrics_registry CASCADE;
TRUNCATE public.dax_functions CASCADE;

-- Option 2: Drop consolidated tables entirely
DROP TABLE IF EXISTS public.dax_functions CASCADE;
DROP TABLE IF EXISTS public.metrics_registry CASCADE;

-- Restore from backup if needed
-- (Your backup strategy here)
```

---

## Recommended Next Steps

1. **Test Migration Script**
   ```bash
   psql -h localhost -U postgres -d alpha -f migrations/consolidate_metrics_and_dax.sql
   ```

2. **Verify Data Integrity**
   - Check record counts match original totals
   - Spot-check a few records
   - Verify foreign key relationships (if any)

3. **Search Codebase for References**
   ```bash
   grep -r "banking\.metrics_registry\|capital_markets\.metrics_registry" backend/
   grep -r "\.dax_functions" backend/
   ```

4. **Update Application Code**
   - Create a list of all affected files
   - Update queries to use `WHERE schema_domain = '...'`
   - Deploy and test

5. **Monitor Performance**
   - Check index usage after migration
   - Monitor query performance with new consolidated tables
   - Adjust indexes if needed

6. **Document Changes**
   - Update API documentation
   - Update data model documentation
   - Record schema changes in changelog

---

## Additional Options for Future Architecture

### Option A: Reference Table for Domains
```sql
CREATE TABLE public.domains (
    id SERIAL PRIMARY KEY,
    domain_name VARCHAR(100) UNIQUE NOT NULL,
    display_name VARCHAR(255),
    description TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

INSERT INTO public.domains (domain_name, display_name) VALUES
('banking', 'Banking'),
('capital_markets', 'Capital Markets'),
-- ... etc
```

Then add foreign key:
```sql
ALTER TABLE public.metrics_registry 
ADD CONSTRAINT fk_metrics_schema_domain 
FOREIGN KEY (schema_domain) REFERENCES public.domains(domain_name);
```

### Option B: Audit Trail
```sql
CREATE TABLE public.metrics_registry_audit (
    id SERIAL PRIMARY KEY,
    metric_id INT REFERENCES public.metrics_registry(id),
    schema_domain VARCHAR(100),
    changed_by VARCHAR(255),
    change_type VARCHAR(50),  -- INSERT, UPDATE, DELETE
    old_values JSONB,
    new_values JSONB,
    changed_at TIMESTAMP DEFAULT NOW()
);
```

### Option C: Tagged Metrics for Cross-Domain Discovery
```sql
-- Add to queries to find metrics from multiple domains
SELECT * FROM public.metrics_registry
WHERE 'cross-domain' = ANY(tags)
ORDER BY schema_domain, node_id;
```

---

## Questions & Support

- **Data Loss Concern?** No - migration uses `ON CONFLICT DO NOTHING`, preserving all unique records
- **Performance Impact?** Minimal - consolidated table with proper indexes performs better
- **Query Latency?** Potential slight improvement due to single index scan instead of multiple queries
- **Downtime Required?** No - migration runs in seconds, can be done during normal operation

---

## Files Provided

1. **`migrations/consolidate_metrics_and_dax.sql`** - Complete migration script (ready to run)
2. **This document** - Detailed consolidation plan and reference guide
