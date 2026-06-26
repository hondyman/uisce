# Execute Now: Start the Migration in 5 Minutes

**You have everything you need. Start here.**

---

## ⏱️ 5-Minute Setup

### 1. Open Terminal (Paste exactly as written)
```bash
cd /Users/eganpj/GitHub/semlayer
```

### 2. Run Analysis (2 minutes)
```bash
python3 analyze_consolidation.py
```

**Expected output:**
- Shows 12 metrics_registry tables
- Shows 8 dax_functions tables
- Shows 264 total metrics
- Creates `migration_report.json`

### 3. Find Code References (2 minutes)
```bash
bash find_schema_references.sh
```

**Expected output:**
- Lists all files with schema references
- Shows how many files need changes
- Creates `code_migration_report.md`

### 4. Create Backup (1 minute)
```bash
mkdir -p backups
pg_dump -h localhost -U postgres -d alpha -Fc > backups/alpha_backup_$(date +%Y%m%d_%H%M%S).dump
echo "✅ Backup created"
```

---

## 🎯 Choose Your Path (Pick One)

### Path A: I Want to Understand First (15 minutes)
```bash
1. Open: README_CONSOLIDATION_START_HERE.md
2. Read: QUICK_REFERENCE.md  
3. Review: CONSOLIDATION_SUMMARY.md
4. Then come back here and run Phase 1
```

### Path B: I Want to Execute Now (Follow this whole document)
```bash
Continue reading below 👇
```

### Path C: I'm a DBA and Want SQL Only (5 minutes)
```bash
1. Verify backup:
   ls -lh backups/
   
2. Run migration:
   psql -h localhost -U postgres -d alpha -f migrations/consolidate_metrics_and_dax.sql
   
3. Verify:
   psql -h localhost -U postgres -d alpha -c "SELECT COUNT(*) FROM public.metrics_registry;"
   
Expected: 264
```

---

## 📋 Full Execution (1-2 Hours)

### Phase 1: Verify Everything (Already Done Above)
```bash
✅ python3 analyze_consolidation.py
✅ bash find_schema_references.sh
✅ Database backup created
```

### Phase 2: Run Migration (2 minutes)
```bash
psql -h localhost -U postgres -d alpha -f migrations/consolidate_metrics_and_dax.sql
```

Wait for it to complete, should see:
```
CREATE TABLE
INSERT 0 264
INSERT 0 [some-number]
CREATE INDEX
CREATE INDEX
CREATE INDEX
```

### Phase 3: Verify Migration (3 minutes)

**Check metrics were moved:**
```bash
psql -h localhost -U postgres -d alpha << 'EOF'
SELECT COUNT(*) as total, COUNT(DISTINCT schema_domain) as domains 
FROM public.metrics_registry;
EOF
```

Should show:
```
 total | domains
-------+---------
   264 |      12
```

**Check DAX functions moved:**
```bash
psql -h localhost -U postgres -d alpha << 'EOF'
SELECT COUNT(*) FROM public.dax_functions;
EOF
```

Should show a count > 0.

### Phase 4: Update Backend Code (30-60 minutes)

**Locate your query files:**
```bash
grep -r "banking.metrics_registry" backend/ frontend/ 2>/dev/null | head -20
```

For each file found:

**Find this:**
```go
// Old pattern
query := fmt.Sprintf("SELECT * FROM %s.metrics_registry WHERE ...", domain)
```

**Replace with:**
```go
// New pattern - use query builder or:
query := "SELECT * FROM public.metrics_registry WHERE schema_domain = $1 AND ..."
// Pass domain as parameter: db.QueryContext(ctx, query, domain, ...)
```

**Compile and test:**
```bash
cd backend
go build ./...
go test ./...
```

### Phase 5: Update Frontend Code (30-60 minutes)

**Create new service:**
```bash
cat > frontend/src/services/metricsService.ts << 'EOF'
export class MetricsService {
  async getMetricsByDomain(domain: string) {
    const response = await fetch(`/api/metrics?domain=${domain}`);
    if (!response.ok) throw new Error('Failed to fetch metrics');
    return response.json();
  }
  
  async getMetricsByDomains(domains: string[]) {
    const response = await fetch(`/api/metrics?domains=${domains.join(',')}`);
    if (!response.ok) throw new Error('Failed to fetch metrics');
    return response.json();
  }
  
  async getDAXFunctionsByDomain(domain: string) {
    const response = await fetch(`/api/dax-functions?domain=${domain}`);
    if (!response.ok) throw new Error('Failed to fetch DAX functions');
    return response.json();
  }
}
EOF
```

**Find and update components:**
```bash
grep -r "banking.metrics_registry\|capital_markets.dax_functions" frontend/ | cut -d: -f1 | sort -u
```

For each component, import the new service and use it instead of hardcoded domain schema names.

**Test:**
```bash
cd frontend
npm run build
npm test
```

### Phase 6: Test Everything (15 minutes)

**Start backend:**
```bash
cd backend
PORT=8085 go run ./cmd/server &
```

**In another terminal, test:**
```bash
curl "http://localhost:8085/api/metrics?domain=banking" | jq .
```

Should return JSON array of metrics.

**Start frontend:**
```bash
cd frontend
npm run dev
```

Open http://localhost:3000 and verify metrics load.

---

## ✅ Verification Checklist

Copy and paste each command, verify it works:

```bash
# 1. Database has consolidated tables
psql -h localhost -U postgres -d alpha -c "SELECT COUNT(*) FROM public.metrics_registry;"
# Expected: 264

# 2. All domains represented
psql -h localhost -U postgres -d alpha -c "SELECT schema_domain, COUNT(*) FROM public.metrics_registry GROUP BY schema_domain;"
# Expected: 12 rows

# 3. Backend compiles
cd backend && go build ./... && cd ..
# Expected: No errors

# 4. Frontend compiles  
cd frontend && npm run build && cd ..
# Expected: No errors

# 5. API responds
curl -s "http://localhost:8085/api/metrics?domain=banking" | jq '.[] | .node_id' | head -5
# Expected: 5 metric node_ids

# 6. Frontend loads
curl -s http://localhost:3000 | grep -q "metrics" && echo "✅ Frontend OK" || echo "❌ Frontend Error"
```

---

## 🚨 If Something Breaks

### Immediate Fix (Undo Everything)
```bash
# Restore database
pg_restore -h localhost -U postgres -d alpha backups/alpha_backup_*.dump

# Revert code
git checkout .
git clean -fd

# Restart services
pkill -f "go run"
npm cache clean --force
```

### Contact Support
Reference these files:
- **SQL Issues:** CONSOLIDATION_PLAN.md
- **Backend Issues:** BACKEND_REFACTORING_GUIDE.md  
- **Frontend Issues:** FRONTEND_CODE_UPDATE_GUIDE.md

---

## 📚 Documentation Reference

If you need details at any point:

| Need Help With | Read This | Time |
|---|---|---|
| Architecture overview | CONSOLIDATION_PLAN.md | 10 min |
| Step-by-step guide | STEP_BY_STEP_IMPLEMENTATION.md | 20 min |
| SQL examples | QUICK_REFERENCE.md | 5 min |
| Backend code patterns | BACKEND_REFACTORING_GUIDE.md | 15 min |
| Frontend code patterns | FRONTEND_CODE_UPDATE_GUIDE.md | 15 min |
| Hasura setup (if needed) | HASURA_CONFIGURATION_GUIDE.md | 20 min |
| Everything | INDEX_CONSOLIDATION.md | 5 min |

---

## 🎯 Success = 3 Things Working

After Phase 6, verify:

1. **Database:** `psql -c "SELECT COUNT(*) FROM public.metrics_registry;"` returns 264
2. **Backend:** `curl "http://localhost:8085/api/metrics?domain=banking"` returns JSON
3. **Frontend:** http://localhost:3000 loads and displays metrics

If all 3 work → **You're done!**

---

## ⏰ Time Breakdown

| Phase | Time | What You Do |
|---|---|---|
| 1. Setup & Analyze | 5 min | Run 2 scripts, create backup |
| 2. Run Migration | 2 min | Execute SQL script |
| 3. Verify Migration | 3 min | Run verification queries |
| 4. Update Backend | 45 min | Update Go code, compile, test |
| 5. Update Frontend | 45 min | Update React code, build, test |
| 6. Full Test | 15 min | Start services, run tests |
| **TOTAL** | **2 hours** | 🎉 Migration Complete |

---

## 🚀 Right Now - Next Step

```bash
# 1. Open second terminal
cd /Users/eganpj/GitHub/semlayer

# 2. Run this ONE command
python3 analyze_consolidation.py && bash find_schema_references.sh

# 3. See the output
cat migration_report.json | head -50
cat code_migration_report.md | head -50

# 4. Report back what you see
# Copy-paste first 50 lines of migration_report.json in Slack
```

**That's it. Do step 1-4 right now, takes 2 minutes.**

Then come back and we'll proceed to the database migration.

---

**You've got this. 💪**

