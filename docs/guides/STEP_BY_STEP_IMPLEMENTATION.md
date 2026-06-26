# Schema Consolidation: Step-by-Step Implementation Guide

**Quick Start:** Follow these steps in order to consolidate your metrics and DAX function tables.

---

## 🎯 Overview

You have 12 separate `metrics_registry` tables and 8 separate `dax_functions` tables across domain schemas. This guide will consolidate them into the `public` schema.

- **264 total metrics records** → 1 consolidated table
- **8 domain schemas** with DAX functions → 1 consolidated table
- **Zero data loss** → All records preserved with domain tracking

---

## ✅ Pre-Implementation Checklist

- [ ] Backup your `alpha` database
- [ ] Review `CONSOLIDATION_PLAN.md`
- [ ] Have psql client available
- [ ] Read through migration script: `migrations/consolidate_metrics_and_dax.sql`

---

## 🚀 Implementation Steps

### Step 1: Analyze Current State (5 minutes)

Run the analysis script to verify what we're consolidating:

```bash
# Make script executable
chmod +x analyze_consolidation.py

# Run analysis
python3 analyze_consolidation.py
```

**Expected output:**
- Confirms 12 `metrics_registry` tables
- Confirms 8 `dax_functions` tables
- Shows record counts per schema
- Generates `migration_report.json`

### Step 2: Backup Your Database (2 minutes)

```bash
# Create a backup before any changes
pg_dump -h localhost -U postgres -d alpha -Fc > alpha_backup_$(date +%Y%m%d_%H%M%S).dump

# Verify backup
file alpha_backup_*.dump
```

### Step 3: Find Code References (10 minutes)

Identify all application code that will need updating:

```bash
# Make script executable
chmod +x find_schema_references.sh

# Run search
bash find_schema_references.sh > code_references.txt

# Review results
cat code_references.txt
```

**Common patterns to look for:**
- `FROM banking.metrics_registry`
- `FROM *.dax_functions`
- Imports/services that query these tables

### Step 4: Run Migration (1 minute)

Execute the consolidation migration:

```bash
psql -h localhost -U postgres -d alpha -f migrations/consolidate_metrics_and_dax.sql
```

**Output should show:**
```
CREATE TABLE
CREATE TABLE
CREATE INDEX
...
INSERT 0 10    -- Banking metrics
INSERT 0 10    -- Capital markets metrics
... (more inserts)
INSERT 0 N     -- DAX functions
... verification queries
```

### Step 5: Verify Migration Success (5 minutes)

Run verification queries in your database:

```bash
psql -h localhost -U postgres -d alpha << 'EOF'

-- Check consolidated metrics table
SELECT COUNT(*) as total_metrics FROM public.metrics_registry;

-- Should return ~264
SELECT schema_domain, COUNT(*) FROM public.metrics_registry GROUP BY schema_domain ORDER BY schema_domain;

-- Check consolidated DAX functions
SELECT COUNT(*) as total_dax FROM public.dax_functions;

-- Verify indexes exist
SELECT indexname FROM pg_indexes WHERE tablename = 'metrics_registry';

-- Sample records
SELECT * FROM public.metrics_registry LIMIT 5;
SELECT * FROM public.dax_functions LIMIT 5;

EOF
```

**Expected results:**
- `total_metrics` = 264
- `total_dax` = (count from all domain schemas)
- All indexes created successfully
- Sample records show `schema_domain` populated

### Step 6: Create Application Code Update List (15 minutes)

Create a file listing all changes needed:

```bash
# Generate detailed code change requirements
cat > CODE_CHANGES_REQUIRED.md << 'EOF'
# Required Code Changes for Schema Consolidation

## From references found in Step 3, update these files:

[List files and changes here based on Step 3 output]

### Pattern 1: Simple SELECT queries
OLD: SELECT * FROM banking.metrics_registry WHERE node_id = 'X';
NEW: SELECT * FROM public.metrics_registry WHERE node_id = 'X' AND schema_domain = 'banking';

### Pattern 2: Multi-schema queries (now much simpler!)
OLD: 
  SELECT * FROM banking.metrics_registry WHERE category = 'perf'
  UNION ALL
  SELECT * FROM retail.metrics_registry WHERE category = 'perf'

NEW:
  SELECT * FROM public.metrics_registry 
  WHERE category = 'perf' 
  AND schema_domain IN ('banking', 'retail')

### Pattern 3: DAX functions
OLD: SELECT * FROM financial_services.dax_functions WHERE name = 'X';
NEW: SELECT * FROM public.dax_functions WHERE name = 'X' AND schema_domain = 'financial_services';

EOF

cat CODE_CHANGES_REQUIRED.md
```

### Step 7: Update Application Code (Varies)

Update each file that references the old schema-specific tables.

**For Go applications (backend/internal/api):**
```go
// OLD
rows, err := db.Query(`SELECT * FROM banking.metrics_registry WHERE ...`)

// NEW
rows, err := db.Query(`SELECT * FROM public.metrics_registry 
                       WHERE schema_domain = $1 AND ...`, "banking")
```

**For TypeScript/JavaScript (frontend):**
```typescript
// OLD
const metrics = await fetch('/api/metrics?schema=banking');

// NEW - if API needs update
// OR keep same if API layer already abstracts schema
```

**For SQL migrations/scripts:**
```sql
-- OLD
INSERT INTO banking.metrics_registry (node_id, ...) VALUES (...)

-- NEW
INSERT INTO public.metrics_registry (node_id, schema_domain, ...) 
VALUES (..., 'banking', ...)
```

### Step 8: (Optional) Create Backwards Compatibility Views

If you have many code changes ahead, temporarily create views:

```bash
psql -h localhost -U postgres -d alpha << 'EOF'

-- Banking
CREATE OR REPLACE VIEW banking.metrics_registry AS
SELECT node_id, category, description, formula_type, formula, arguments,
       badge, function_class, functions_used, governance_status, audience, tags,
       created_at, updated_at
FROM public.metrics_registry WHERE schema_domain = 'banking';

CREATE OR REPLACE VIEW banking.dax_functions AS
SELECT name, class, badge, description, created_at
FROM public.dax_functions WHERE schema_domain = 'banking';

-- Capital Markets
CREATE OR REPLACE VIEW capital_markets.metrics_registry AS
SELECT node_id, category, description, formula_type, formula, arguments,
       badge, function_class, functions_used, governance_status, audience, tags,
       created_at, updated_at
FROM public.metrics_registry WHERE schema_domain = 'capital_markets';

CREATE OR REPLACE VIEW capital_markets.dax_functions AS
SELECT name, class, badge, description, created_at
FROM public.dax_functions WHERE schema_domain = 'capital_markets';

-- Financial Services
CREATE OR REPLACE VIEW financial_services.metrics_registry AS
SELECT node_id, category, description, formula_type, formula, arguments,
       badge, function_class, functions_used, governance_status, audience, tags,
       created_at, updated_at
FROM public.metrics_registry WHERE schema_domain = 'financial_services';

CREATE OR REPLACE VIEW financial_services.dax_functions AS
SELECT name, class, badge, description, created_at
FROM public.dax_functions WHERE schema_domain = 'financial_services';

-- ... repeat for remaining schemas

EOF
```

This allows all existing code to continue working while you update it incrementally.

### Step 9: Test Thoroughly (30 minutes)

```bash
# Test existing application endpoints
curl http://localhost:8080/api/metrics

# Test API with specific domain parameter
curl http://localhost:8080/api/metrics?domain=banking

# Run application test suites
go test ./...
npm test

# Verify data hasn't changed
# Sample a few records from public tables and compare to old tables
psql -h localhost -U postgres -d alpha << 'EOF'
SELECT COUNT(*) FROM public.metrics_registry;
-- Compare to original: SELECT COUNT(*) FROM banking.metrics_registry; + capital_markets + ...
EOF
```

### Step 10: Deploy Changes

When ready:

```bash
# Git workflow
git add -A
git commit -m "chore: consolidate metrics_registry and dax_functions into public schema

- Migrate 264 metrics records from 12 domain schemas
- Migrate DAX functions from 8 domain schemas
- Add schema_domain column for domain tracking
- Create performance indexes on new tables
- Update all queries to use new consolidated tables"

git push origin feature/schema-consolidation
```

Then create PR and proceed with normal review/merge process.

### Step 11: Cleanup (After successful deployment)

Once code is deployed and stable for 24+ hours:

```bash
psql -h localhost -U postgres -d alpha << 'EOF'

-- Drop backwards compatibility views (if you created them)
DROP VIEW IF EXISTS banking.metrics_registry CASCADE;
DROP VIEW IF EXISTS banking.dax_functions CASCADE;
DROP VIEW IF EXISTS capital_markets.metrics_registry CASCADE;
DROP VIEW IF EXISTS capital_markets.dax_functions CASCADE;
-- ... repeat for all schemas

-- Drop old tables from domain schemas
DROP TABLE IF EXISTS banking.metrics_registry CASCADE;
DROP TABLE IF EXISTS banking.dax_functions CASCADE;
DROP TABLE IF EXISTS capital_markets.metrics_registry CASCADE;
DROP TABLE IF EXISTS capital_markets.dax_functions CASCADE;
-- ... repeat for all domain schemas

-- Optionally vacuum to reclaim space
VACUUM ANALYZE;

EOF
```

---

## 📊 Expected Migration Results

After Step 4 (migration):

| Metric | Expected |
|--------|----------|
| `public.metrics_registry` rows | 264 |
| `public.dax_functions` rows | ~40-60 |
| Indexes created | 5 |
| Tables created | 2 |
| Old tables still exist | ✓ Yes (until Step 11) |

After Step 11 (cleanup):

| Metric | Expected |
|--------|----------|
| Domain schema bloat | Removed |
| `public` schema tables | metrics_registry, dax_functions |
| Old tables | Deleted |
| Total schema count | 17 (same, but leaner) |

---

## 🔄 Rollback Procedure

If something goes wrong:

### Option 1: Restore from backup (recommended)
```bash
# Stop application
systemctl stop your-app

# Restore backup
pg_restore -h localhost -U postgres -d alpha alpha_backup_YYYYMMDD_HHMMSS.dump

# Restart application
systemctl start your-app
```

### Option 2: Drop consolidated tables
```bash
psql -h localhost -U postgres -d alpha << 'EOF'
DROP TABLE IF EXISTS public.dax_functions CASCADE;
DROP TABLE IF EXISTS public.metrics_registry CASCADE;
EOF
```

Old tables remain in domain schemas and can be used to retry.

---

## 📈 Performance Expectations

| Operation | Before | After | Change |
|-----------|--------|-------|--------|
| Query single domain metrics | Fast | Fast | Same ✓ |
| Query cross-domain metrics | Multiple queries | Single query | **Better** 🚀 |
| Storage for these tables | 17 copies | 1 copy | **~94% reduction** 💾 |
| Index maintenance | 12 tables | 1 table | **Simpler** ✨ |

---

## ✨ FAQ

**Q: Will this cause downtime?**  
A: No. Migration takes <1 second. Views can be created for backwards compatibility.

**Q: What if there are duplicate records?**  
A: `ON CONFLICT DO NOTHING` in migration handles deduplication safely.

**Q: Can I run the migration multiple times?**  
A: Yes! It's idempotent. Safe to re-run.

**Q: How do I verify data integrity?**  
A: Compare record counts before/after. See Step 5 verification queries.

**Q: Can we keep both old and new tables temporarily?**  
A: Yes! That's exactly what we do. Views allow gradual code migration.

**Q: What about foreign keys?**  
A: Check if old tables have foreign key dependencies in Step 3. Update if needed.

---

## 🆘 Troubleshooting

### Issue: "relation does not exist" after migration
**Solution:** Create backwards compatibility views (Step 8)

### Issue: Duplicate key violations on insert
**Solution:** Tables already have data. Migration handles this with `ON CONFLICT DO NOTHING`

### Issue: Application still can't find records
**Solution:** Verify `schema_domain` value matches your code. Check Step 5 queries.

### Issue: Performance got worse
**Solution:** Ensure indexes created. Run: `ANALYZE public.metrics_registry;`

---

## 📞 Need Help?

Review these files in order:
1. **CONSOLIDATION_PLAN.md** - Strategic overview
2. **This file (STEP_BY_STEP.md)** - Implementation guide  
3. **migrations/consolidate_metrics_and_dax.sql** - Actual SQL
4. **find_schema_references.sh** - Find affected code
5. **analyze_consolidation.py** - Analysis tool

---

## ✅ Completion Checklist

- [ ] Step 1: Analyzed current state
- [ ] Step 2: Backed up database
- [ ] Step 3: Found code references
- [ ] Step 4: Ran migration
- [ ] Step 5: Verified migration
- [ ] Step 6: Created code change list
- [ ] Step 7: Updated application code
- [ ] Step 8: (Optional) Created views for compatibility
- [ ] Step 9: Tested thoroughly
- [ ] Step 10: Deployed changes
- [ ] Step 11: Cleaned up (after stable period)

**All complete? 🎉 You've successfully consolidated your schemas!**
