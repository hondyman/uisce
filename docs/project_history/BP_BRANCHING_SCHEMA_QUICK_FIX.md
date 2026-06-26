# Quick Fix Reference: BP Branching Schema

## Summary of Errors & Fixes

### Error 1: Foreign Key Constraint Mismatch
```
SQL Error [42830]: ERROR: there is no unique constraint matching given keys 
for referenced table "bp_branch_executions"
```

**What was wrong**: 
- `bp_branch_events` table references `bp_branch_executions(workflow_instance_id)` 
- But `workflow_instance_id` in `bp_branch_executions` had no unique constraint

**What was fixed**:
```sql
-- Added this constraint to bp_branch_executions table:
CONSTRAINT uq_workflow_instance UNIQUE (workflow_instance_id)
```

**File**: `backend/pkg/bp/branching_schema.sql` (line 89)

---

### Error 2: Missing Database Role
```
SQL Error [42704]: ERROR: role "app_user" does not exist
```

**What was wrong**: 
- Schema tried to grant permissions to `app_user` role
- Role was never created in the database

**What was fixed**:
```sql
-- Added this at the beginning of the schema (Section 0):
DO $$ BEGIN
  IF NOT EXISTS (SELECT FROM pg_user WHERE usename = 'app_user') THEN
    CREATE USER app_user WITH PASSWORD 'app_user_password';
  END IF;
END $$;
```

**File**: `backend/pkg/bp/branching_schema.sql` (lines 6-15)

---

## How to Apply the Fix

### Option 1: Re-run the complete schema
```bash
# If tables don't exist yet, just re-run:
psql -U postgres -d alpha -f backend/pkg/bp/branching_schema.sql
```

### Option 2: Apply only the fixes (if tables exist)
```sql
-- In psql:
\c alpha

-- Fix 1: Create the role
DO $$ BEGIN
  IF NOT EXISTS (SELECT FROM pg_user WHERE usename = 'app_user') THEN
    CREATE USER app_user WITH PASSWORD 'app_user_password';
  END IF;
END $$;

-- Fix 2: Add unique constraint (if table exists)
ALTER TABLE bp_branch_executions 
ADD CONSTRAINT uq_workflow_instance UNIQUE (workflow_instance_id);

-- Fix 3: Re-grant permissions
GRANT SELECT, INSERT, UPDATE, DELETE ON bp_branch_executions TO app_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON bp_branch_metrics TO app_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON bp_join_convergences TO app_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON bp_ml_models TO app_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON bp_ab_tests TO app_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON bp_branch_events TO app_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON bp_branch_anomalies TO app_user;
GRANT SELECT ON bp_branch_summary_metrics TO app_user;
```

---

## Verification Steps

### Check 1: Verify role exists
```bash
psql -U postgres -d alpha -c "SELECT * FROM pg_user WHERE usename = 'app_user'"
```
✅ Should return 1 row

### Check 2: Verify unique constraint exists
```bash
psql -U postgres -d alpha -c "SELECT constraint_name FROM information_schema.table_constraints WHERE table_name='bp_branch_executions' AND constraint_type='UNIQUE'"
```
✅ Should return `uq_workflow_instance`

### Check 3: Verify foreign key works
```bash
psql -U postgres -d alpha -c "SELECT constraint_name FROM information_schema.referential_constraints WHERE referenced_table_name='bp_branch_executions'"
```
✅ Should return `fk_branch_events_workflow` (and possibly others)

---

## What's Fixed

| Issue | Before | After | Status |
|-------|--------|-------|--------|
| Missing role | Error on grant | Role created automatically | ✅ FIXED |
| No unique constraint | Foreign key error | `UNIQUE (workflow_instance_id)` added | ✅ FIXED |
| Foreign key mismatch | Foreign key error on `bp_branch_events` create | Foreign key now valid | ✅ FIXED |

---

## Next Steps

The schema is now ready to use:

1. ✅ Apply the schema (fixed version)
2. ✅ Verify all tables created
3. ✅ Verify all permissions granted
4. → Next: Rebuild and register Go handlers
5. → Then: Test branching evaluation endpoints

---

**Date**: October 21, 2025  
**Status**: ✅ SCHEMA FIXED AND READY  
**Reference**: See `BP_BRANCHING_SCHEMA_FIX.md` for detailed analysis
