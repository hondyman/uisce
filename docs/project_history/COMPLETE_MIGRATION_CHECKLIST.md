# Complete Migration Checklist: Metrics Consolidation

**Date:** November 3, 2025  
**Project:** Consolidate metrics_registry and dax_functions into public schema  
**Print This:** Yes - Use as your execution guide

---

## 📋 Pre-Migration Setup (Day Before)

### Planning
- [ ] Assign team members to each phase
- [ ] Schedule maintenance window (if needed)
- [ ] Alert stakeholders of changes
- [ ] Review all documentation files
- [ ] Understand current data flow
- [ ] Test rollback procedure

### Environment Setup
- [ ] Verify PostgreSQL 13+ installed
- [ ] Confirm database credentials work
- [ ] Verify psql client available
- [ ] Test backup procedure
- [ ] Confirm Node.js/Go environments working
- [ ] Verify test suite can run
- [ ] Check disk space for backup

### Documentation Review
- [ ] Read: QUICK_REFERENCE.md (5 min)
- [ ] Read: CONSOLIDATION_SUMMARY.md (10 min)
- [ ] Read: STEP_BY_STEP_IMPLEMENTATION.md (20 min)
- [ ] Skim: CONSOLIDATION_PLAN.md (reference only)
- [ ] Review: CODE_MIGRATION_GUIDE.md (reference)
- [ ] Review: BACKEND_REFACTORING_GUIDE.md (reference)
- [ ] Review: FRONTEND_CODE_UPDATE_GUIDE.md (reference)

---

## 🚀 Phase 1: Analysis (30 minutes)

### Step 1.1: Run Database Analysis
```bash
[ ] cd /Users/eganpj/GitHub/semlayer
[ ] python3 analyze_consolidation.py
[ ] Note the output - confirm 12 metrics_registry tables exist
[ ] Note the total record count (should be 264)
[ ] Check migration_report.json for details
[ ] Screenshot results for reference
```

### Step 1.2: Find Code References
```bash
[ ] bash find_schema_references.sh > code_refs.txt
[ ] cat code_refs.txt
[ ] Count total files needing updates
[ ] Identify which services/components affected
[ ] Create list of files by priority
[ ] Document findings in your notes
```

### Step 1.3: Plan Code Changes
```bash
[ ] Determine update strategy (all-at-once vs gradual)
[ ] List backend files to update
[ ] List frontend files to update
[ ] List API handlers to update
[ ] Identify test files that need updates
[ ] Plan deployment sequence
```

---

## 💾 Phase 2: Backup & Database Preparation (5 minutes)

### Step 2.1: Create Database Backup
```bash
[ ] mkdir -p backups/
[ ] pg_dump -h localhost -U postgres -d alpha -Fc > backups/alpha_backup_$(date +%Y%m%d_%H%M%S).dump
[ ] ls -lh backups/alpha_backup_*.dump
[ ] Confirm backup file created successfully
[ ] Test backup restore procedure on test database (if possible)
[ ] Document backup filename for later reference
```

### Step 2.2: Verify Database Connection
```bash
[ ] psql -h localhost -U postgres -d alpha -c "SELECT version();"
[ ] psql -h localhost -U postgres -d alpha -c "SELECT COUNT(*) FROM banking.metrics_registry;"
[ ] Note the current record count
[ ] Confirm tables exist in domain schemas
```

---

## 🔧 Phase 3: Execute Migration (2 minutes)

### Step 3.1: Run Migration Script
```bash
[ ] psql -h localhost -U postgres -d alpha -f migrations/consolidate_metrics_and_dax.sql
[ ] Watch for errors in output
[ ] Confirm all CREATE TABLE statements completed
[ ] Confirm all INSERT statements completed (should show count)
[ ] Confirm all indexes created
[ ] Note any errors or warnings
```

### Step 3.2: Verify Migration Success
```bash
[ ] psql -h localhost -U postgres -d alpha << 'EOF'
    SELECT COUNT(*) as metric_count FROM public.metrics_registry;
    EOF
    Expected: 264 (or close to it)
    [ ] Actual count: _________

[ ] psql -h localhost -U postgres -d alpha << 'EOF'
    SELECT schema_domain, COUNT(*) FROM public.metrics_registry GROUP BY schema_domain;
    EOF
    [ ] All 12 domains represented (banking, retail, etc.)
    [ ] Counts add up to 264

[ ] psql -h localhost -U postgres -d alpha << 'EOF'
    SELECT COUNT(*) as function_count FROM public.dax_functions;
    EOF
    [ ] Count is > 0
    [ ] Actual count: _________

[ ] psql -h localhost -U postgres -d alpha << 'EOF'
    SELECT indexname FROM pg_indexes WHERE tablename = 'metrics_registry';
    EOF
    [ ] See at least 3 indexes
    [ ] Indexes: schema_domain, node_id, category
```

---

## 💻 Phase 4: Update Backend Code (1-2 hours)

### Step 4.1: Identify Files to Update
```bash
[ ] Review code_refs.txt from Step 1.2
[ ] Prioritize by frequency of use:
    [ ] API handlers (highest priority)
    [ ] Core services (high priority)
    [ ] Helper/utility functions (medium)
    [ ] Tests (medium)
    [ ] Documentation (low)
```

### Step 4.2: Update API Handlers
```bash
[ ] Locate bundle handler (likely in backend/internal/api)
[ ] Update to use: SELECT * FROM public.metrics_registry WHERE schema_domain = $1
[ ] Update to use: SELECT * FROM public.dax_functions WHERE schema_domain = $1
[ ] Remove hardcoded schema names from queries
[ ] Use parameterized queries ($1, $2, etc.)
[ ] Compile: go build ./backend/...
[ ] [ ] No compilation errors
```

### Step 4.3: Create Backend Services (if not exists)
```bash
[ ] Create metrics_service.go with consolidated queries
[ ] Create dax_functions_service.go with consolidated queries
[ ] Reference: BACKEND_REFACTORING_GUIDE.md for patterns
[ ] Implement methods:
    [ ] GetMetricsByDomain(ctx, domain)
    [ ] GetMetricsByDomains(ctx, domains)
    [ ] GetMetricByNodeID(ctx, domain, nodeID)
    [ ] GetDAXFunctionsByDomain(ctx, domain)
    [ ] GetDAXFunctionsByDomains(ctx, domains)
[ ] Add error handling
[ ] Add logging
```

### Step 4.4: Update Database Queries
For each file in code_refs.txt:
```bash
[ ] Replace: fmt.Sprintf("FROM %s.metrics_registry", domain)
    With: "FROM public.metrics_registry WHERE schema_domain = $1"
    
[ ] Replace: banking.metrics_registry in queries
    With: public.metrics_registry WHERE schema_domain = 'banking'
    
[ ] Replace: Dynamic schema loops
    With: sqlx.In() for multi-domain queries
```

### Step 4.5: Test Backend Changes
```bash
[ ] go test ./backend/...
[ ] [ ] All tests pass
[ ] [ ] No new failures introduced
[ ] Start backend: PORT=8085 go run ./backend/cmd/server
[ ] Test endpoints manually
[ ] [ ] GET /api/metrics?domain=banking returns data
[ ] [ ] GET /api/dax-functions?domain=banking returns data
[ ] Review logs for errors
[ ] Stop backend (Ctrl+C)
```

---

## 🎨 Phase 5: Update Frontend Code (30 minutes)

### Step 5.1: Create Services
```bash
[ ] Create frontend/src/services/metricsService.ts
    [ ] Reference: FRONTEND_CODE_UPDATE_GUIDE.md
    [ ] Implement: getMetricsByDomain()
    [ ] Implement: getMetricsByDomains()
    [ ] Implement: getDAXFunctionsByDomain()
    [ ] Add: error handling
    [ ] Add: TypeScript types
```

### Step 5.2: Create Hooks
```bash
[ ] Create frontend/src/hooks/useMetrics.ts
    [ ] Implement: useMetrics(domain)
    [ ] Implement: useMetricsMultipleDomains(domains)
    [ ] Add: loading states
    [ ] Add: error handling
    [ ] Add: refetch function
```

### Step 5.3: Update Components
```bash
[ ] Update MetricsList component
    [ ] Use new hooks
    [ ] Display schemaDomain column
    [ ] Add filtering by domain
[ ] Update MetricsViewer component
[ ] Update DAXFunctionReference component
[ ] Update BundleExplorer component
[ ] Compile: npm run build
[ ] [ ] No TypeScript errors
```

### Step 5.4: Test Frontend Changes
```bash
[ ] npm test
[ ] [ ] All tests pass
[ ] npm run dev
[ ] [ ] Frontend starts without errors
[ ] Visit http://localhost:3000
[ ] [ ] Metrics page loads
[ ] [ ] Select different domains
[ ] [ ] Data displays correctly
[ ] [ ] No console errors
```

---

## 🧪 Phase 6: Integration Testing (30 minutes)

### Step 6.1: End-to-End Tests
```bash
[ ] Start backend: PORT=8085 DSN='...' go run ./backend/cmd/server
[ ] Start frontend: npm run dev (in different terminal)
[ ] Test: Load metrics for banking domain
    [ ] [ ] Displays metrics
    [ ] [ ] Shows schema_domain column
    [ ] [ ] All records visible
    
[ ] Test: Switch between domains
    [ ] [ ] Data refreshes correctly
    [ ] [ ] No stale data shown
    
[ ] Test: Multi-domain view
    [ ] [ ] Loads metrics from multiple domains
    [ ] [ ] Filtering works
    [ ] [ ] Sorting works
    
[ ] Test: Error scenarios
    [ ] [ ] Stop backend
    [ ] [ ] Frontend shows error message
    [ ] [ ] Retry button works
    [ ] [ ] Restart backend
```

### Step 6.2: Performance Testing
```bash
[ ] Measure query time for single domain
    Expected: < 100ms
    [ ] Actual: ________ms
    
[ ] Measure query time for 5 domains
    Expected: < 200ms
    [ ] Actual: ________ms
    
[ ] Check database query plan
    [ ] psql -h localhost -U postgres -d alpha
        EXPLAIN ANALYZE 
        SELECT * FROM public.metrics_registry 
        WHERE schema_domain = 'banking';
    [ ] Confirm indexes used
```

### Step 6.3: Data Integrity Verification
```bash
[ ] Verify all records migrated
    psql: SELECT COUNT(*) FROM public.metrics_registry;
    Expected: 264
    [ ] Actual: ________
    
[ ] Verify domain breakdown
    psql: SELECT schema_domain, COUNT(*) 
          FROM public.metrics_registry 
          GROUP BY schema_domain;
    [ ] All 12 domains present
    [ ] Counts match original
    
[ ] Verify no data loss
    [ ] Sample check: Query specific metric
    [ ] psql: SELECT * FROM public.metrics_registry 
              WHERE schema_domain = 'banking' 
              AND node_id = 'METRIC_001';
    [ ] Record exists and data intact
    
[ ] Verify DAX functions
    psql: SELECT COUNT(*) FROM public.dax_functions;
    [ ] All functions migrated
```

---

## 📦 Phase 7: Code Review & Merge (30 minutes)

### Step 7.1: Create Pull Request
```bash
[ ] git status
[ ] git add .
[ ] git commit -m "chore: consolidate metrics_registry and dax_functions

- Migrate 264 metrics from 12 domain schemas to public schema
- Migrate DAX functions from 8 domain schemas to public schema
- Add schema_domain column for domain tracking
- Update backend services to query consolidated tables
- Update frontend components to display consolidated data
- Add indexes for performance
- Maintain backward compatibility with views (optional)"

[ ] git push origin feature/consolidate-metrics
```

### Step 7.2: Code Review Checklist
For reviewers:
```bash
[ ] Backend changes:
    [ ] SQL queries use parameterized inputs
    [ ] No hardcoded schema names
    [ ] Proper error handling
    [ ] Tests updated and passing
    
[ ] Frontend changes:
    [ ] TypeScript types correct
    [ ] No unused imports
    [ ] Error states handled
    [ ] Tests updated and passing
    
[ ] Database migration:
    [ ] Idempotent (safe to run multiple times)
    [ ] No data loss
    [ ] Indexes created
    [ ] Performance acceptable
    
[ ] Documentation:
    [ ] Code changes documented
    [ ] API changes documented
    [ ] Migration steps clear
```

### Step 7.3: Approval & Merge
```bash
[ ] Get approval from 1-2 team members
[ ] Resolve any comments
[ ] Squash commits if needed
[ ] Merge to main branch
[ ] Verify CI/CD pipeline passes
[ ] Tag release (optional)
```

---

## 🚀 Phase 8: Deployment to Production (Varies)

### Step 8.1: Pre-Deployment Checklist
```bash
[ ] All code reviewed and approved
[ ] All tests passing in CI/CD
[ ] Database backup created
[ ] Deployment plan documented
[ ] Team notified
[ ] Monitoring set up
[ ] Rollback procedure tested
```

### Step 8.2: Deploy Backend
```bash
[ ] Deploy migration first:
    psql -h prod.db.com -U postgres -d alpha -f migrations/consolidate_metrics_and_dax.sql
    [ ] Verify success
    [ ] Check record counts
    
[ ] Deploy code:
    [ ] Deploy to canary/staging first
    [ ] Verify API endpoints work
    [ ] Check logs for errors
    
[ ] Deploy to production:
    [ ] Use blue-green deployment if available
    [ ] Monitor error logs
    [ ] Check performance metrics
    [ ] [ ] No errors in logs
```

### Step 8.3: Deploy Frontend
```bash
[ ] Build production bundle:
    npm run build
    [ ] No errors
    [ ] Bundle size acceptable
    
[ ] Deploy to staging:
    [ ] Test key user flows
    [ ] Verify metrics load
    [ ] Check performance
    
[ ] Deploy to production:
    [ ] Use CDN cache busting if needed
    [ ] Monitor browser console for errors
    [ ] Monitor analytics
    [ ] [ ] No 404s
    [ ] [ ] No data loading failures
```

### Step 8.4: Post-Deployment Verification
```bash
[ ] Monitor dashboards for 1 hour
    [ ] No spike in error rates
    [ ] Performance normal
    [ ] All metrics loading
    
[ ] Check user reports
    [ ] No complaints in Slack/support tickets
    
[ ] Verify data integrity (production)
    [ ] psql SELECT COUNT(*) FROM public.metrics_registry;
    [ ] Expected: 264
    [ ] [ ] Actual: ________
    
[ ] Monitor for 24 hours
    [ ] Stable performance
    [ ] No unexpected errors
    
[ ] Send success notification
```

---

## 🔧 Optional: Cleanup Phase (After 24-48 hours of stability)

### Step 9.1: Drop Backwards Compatibility Views
```bash
[ ] Confirm no code still using old schema names
[ ] grep -r "banking.metrics_registry" backend/ frontend/
[ ] [ ] No results found
[ ] Drop views:
    psql << 'EOF'
    DROP VIEW IF EXISTS banking.metrics_registry CASCADE;
    DROP VIEW IF EXISTS capital_markets.metrics_registry CASCADE;
    -- ... repeat for all schemas
    EOF
[ ] Verify dropped
```

### Step 9.2: Drop Old Tables
```bash
[ ] FINAL WARNING: This is irreversible unless you restore backup
[ ] Confirm all code uses public schema
[ ] Confirm tests all passing
[ ] Drop old tables:
    psql << 'EOF'
    DROP TABLE IF EXISTS banking.metrics_registry CASCADE;
    DROP TABLE IF EXISTS banking.dax_functions CASCADE;
    -- ... repeat for all domains
    EOF
[ ] Verify dropped
```

### Step 9.3: Optimize Storage
```bash
[ ] VACUUM ANALYZE;
[ ] Check storage reclaimed
[ ] Verify performance still good
[ ] Document space saved
```

---

## ⚠️ Rollback Procedure (If Issues Occur)

### Immediate Rollback
```bash
[ ] Revert frontend code: git revert <commit>
[ ] Redeploy frontend
[ ] Revert backend code: git revert <commit>
[ ] Restart backend

[ ] Restore database backup:
    pg_restore -h localhost -U postgres -d alpha backups/alpha_backup_*.dump
    
[ ] Wait for stability (15-30 min)
[ ] Verify old data structure restored
[ ] Notify team
```

### Rollback Verification
```bash
[ ] psql -h localhost -U postgres -d alpha -c "SELECT COUNT(*) FROM banking.metrics_registry;"
[ ] Old tables should exist again
[ ] [ ] Metrics loading from old schema
[ ] [ ] Frontend showing data
[ ] [ ] No errors in logs
```

---

## 📊 Success Criteria

After all phases complete, confirm:

```
Database:
  ✓ 1 public.metrics_registry with 264 records
  ✓ 1 public.dax_functions with all functions
  ✓ schema_domain column tracking domains
  ✓ Indexes on schema_domain, node_id, name
  ✓ Performance indexes working

Backend:
  ✓ Queries use public schema + WHERE schema_domain
  ✓ No hardcoded schema names
  ✓ All services updated
  ✓ All tests passing
  ✓ API endpoints working

Frontend:
  ✓ Components display consolidated data
  ✓ Multi-domain queries working
  ✓ Filtering/sorting working
  ✓ Error handling working
  ✓ All tests passing

Code Quality:
  ✓ No console errors
  ✓ No TypeScript errors
  ✓ No Go compilation errors
  ✓ All tests passing
  ✓ Code reviewed and approved

Performance:
  ✓ Query latency: < 100ms single domain
  ✓ Query latency: < 200ms multi-domain
  ✓ No regressions vs baseline
  ✓ ~94% storage reduction achieved

Operations:
  ✓ Zero downtime deployment
  ✓ Rollback tested
  ✓ Team trained
  ✓ Documentation updated
```

---

## 📞 Support & References

If you encounter issues:

1. **Backend Issues:** See CODE_MIGRATION_GUIDE.md
2. **Frontend Issues:** See FRONTEND_CODE_UPDATE_GUIDE.md  
3. **Database Issues:** See CONSOLIDATION_PLAN.md
4. **Architecture Questions:** See CONSOLIDATION_PLAN.md
5. **Quick Reference:** See QUICK_REFERENCE.md

---

## 📝 Sign-Off

```
Project: Metrics Consolidation
Executed By: _____________________ Date: _____
Reviewed By: _____________________ Date: _____
Deployed By: _____________________ Date: _____
Approved By: _____________________ Date: _____

Notes:
____________________________________________________________________
____________________________________________________________________
____________________________________________________________________
```

---

**Print this checklist and track your progress!**

Keep this handy throughout migration - check items off as you complete them and note any issues encountered for the team debrief.

