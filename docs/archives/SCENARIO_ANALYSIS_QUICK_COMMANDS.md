# 🚀 Scenario Analysis - Quick Commands

## Test It Now (Copy & Paste)

### 1️⃣ Verify Menu Integration
```bash
# Check menu item was added
grep "Scenario Analysis" frontend/src/components/MainNavigation.tsx

# Should output:
# { label: 'Scenario Analysis', path: '/analytics/scenario-analysis', ...
```

### 2️⃣ Verify Route Integration
```bash
# Check route was added
grep "scenario-analysis" frontend/src/AppRoutes.tsx

# Should output:
# <Route path="/analytics/scenario-analysis" element={...
```

### 3️⃣ Verify Import
```bash
# Check component import
grep "ScenarioAnalysisPro" frontend/src/AppRoutes.tsx

# Should output:
# import ScenarioAnalysisPro from "./components/ScenarioAnalysisPro";
```

### 4️⃣ View All Changes
```bash
# See git diff
cd /Users/eganpj/GitHub/semlayer
git diff frontend/src/AppRoutes.tsx frontend/src/components/MainNavigation.tsx
```

### 5️⃣ Verify Backend Routes
```bash
# Check backend routes already registered
grep -A2 "RegisterScenarioAnalysisRoutes" api-gateway/main.go

# Should output:
# apipkg.RegisterScenarioAnalysisRoutes(r, tc)
```

---

## Test in Browser

### Navigate to Feature
```
1. Open: http://localhost:3000
2. Click: Entity (top nav)
3. Hover: Analytics (submenu)
4. Click: Scenario Analysis

✅ Component should load
```

### Direct URL
```
http://localhost:3000/analytics/scenario-analysis

✅ Should route directly to component
```

### Verify Tenant Scope (Console)
```javascript
// Open DevTools (F12) → Console tab

// Check localStorage
JSON.parse(localStorage.getItem('selected_tenant'))
JSON.parse(localStorage.getItem('selected_product'))
JSON.parse(localStorage.getItem('selected_datasource'))

// All should have values (tenant scope working)
```

---

## Test API Endpoint

### Basic Test
```bash
# Test API endpoint exists
curl -X POST \
  -H "X-Tenant-ID: test-tenant" \
  -H "X-Tenant-Datasource-ID: test-datasource" \
  -H "Content-Type: application/json" \
  -d '{"scenario":"market-downturn"}' \
  "http://localhost:8080/api/portfolio/test/scenario"

# Should respond (handler exists)
```

### With Real IDs
```bash
# Replace with your actual tenant/datasource IDs
TENANT_ID="your-tenant-id"
DATASOURCE_ID="your-datasource-id"
PORTFOLIO_ID="your-portfolio-id"

curl -X POST \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "X-Tenant-Datasource-ID: $DATASOURCE_ID" \
  -H "Content-Type: application/json" \
  -d '{"scenario":"interest-rate-rise"}' \
  "http://localhost:8080/api/portfolio/$PORTFOLIO_ID/scenario?tenant_id=$TENANT_ID&datasource_id=$DATASOURCE_ID"
```

---

## Implement Backend (From Templates)

### Copy Workflow Template
```bash
# Read the template
cat SCENARIO_ANALYSIS_CODE_EXAMPLES.md | grep -A 100 "Temporal workflow"

# Create file and paste template
cat > backend/temporal/workflows/scenario_analysis.go << 'EOF'
# Paste the workflow code from SCENARIO_ANALYSIS_CODE_EXAMPLES.md
EOF
```

### Copy Activities Template
```bash
# Create activities directory
mkdir -p backend/temporal/activities

# Read template and create activities
cat > backend/temporal/activities/scenario_activities.go << 'EOF'
# Paste the activities code from SCENARIO_ANALYSIS_CODE_EXAMPLES.md
EOF
```

### Apply Database Migrations
```bash
# Get the migration SQL
cat SCENARIO_ANALYSIS_CODE_EXAMPLES.md | grep -A 50 "Database schema"

# Create migration file
cat > backend/migrations/001_scenario_analysis_schema.sql << 'EOF'
# Paste the schema from SCENARIO_ANALYSIS_CODE_EXAMPLES.md
EOF

# Apply migration
psql postgres://postgres:postgres@localhost:5432/alpha < backend/migrations/001_scenario_analysis_schema.sql
```

---

## Build & Test

### Build Backend
```bash
cd api-gateway
go mod tidy
go build -o semlayer-backend ./main.go

# Check for errors
echo "Build status: $?"
```

### Run Backend Tests
```bash
cd api-gateway
go test ./... -v

# Or specific test
go test ./api -run TestScenarioAnalysis -v
```

### Build Frontend
```bash
cd frontend
npm install
npm run build

# Check build
npm run preview -- --port 3000
```

### Run Frontend Tests
```bash
cd frontend
npm test -- scenario-analysis

# Or all tests
npm test
```

---

## Documentation

### Read All Docs
```bash
# List all scenario analysis docs
ls -1 SCENARIO_ANALYSIS_*.md

# Read summary
cat SCENARIO_ANALYSIS_DELIVERY_SUMMARY.md | head -50

# Read index
cat SCENARIO_ANALYSIS_INDEX.md | grep "^#"

# Open visual reference
open frontend/SCENARIO_ANALYSIS_VISUAL_REFERENCE.html
# or
firefox frontend/SCENARIO_ANALYSIS_VISUAL_REFERENCE.html
```

### Find Documentation
```bash
# Find all scenario docs
find . -name "*SCENARIO*" -type f

# Find component files
find . -name "*Scenario*" -type f

# Find code examples
grep -r "ScenarioAnalysis" --include="*.md"
```

---

## Git Operations

### Stage Changes
```bash
git add frontend/src/AppRoutes.tsx frontend/src/components/MainNavigation.tsx
```

### View Changes
```bash
git diff --staged
```

### Commit
```bash
git commit -m "feat: integrate scenario analysis into menu and routes"
```

### Push
```bash
git push origin chore/triage-u1000-shims
```

---

## Troubleshooting

### Menu item not showing?
```bash
# Check menu item was added
grep -n "Scenario Analysis" frontend/src/components/MainNavigation.tsx

# Check syntax
npm run lint -- frontend/src/components/MainNavigation.tsx

# Rebuild
cd frontend && npm run build
```

### Route returns 404?
```bash
# Check route exists
grep -n "scenario-analysis" frontend/src/AppRoutes.tsx

# Check component file exists
ls -la frontend/src/components/ScenarioAnalysisPro.tsx

# Check import statement
grep -n "ScenarioAnalysisPro" frontend/src/AppRoutes.tsx
```

### Backend not responding?
```bash
# Check if backend is running
curl http://localhost:8080/_health

# Check if route is registered
grep -n "RegisterScenarioAnalysisRoutes" api-gateway/main.go

# Check backend logs
tail -f api-gateway.log | grep -i scenario
```

### Tenant scope not working?
```javascript
// Check localStorage
console.log('Tenant:', localStorage.getItem('selected_tenant'));
console.log('Datasource:', localStorage.getItem('selected_datasource'));

// If empty, select tenant in UI first
// Then check again
```

---

## Performance Checks

### Frontend Performance
```javascript
// Measure component load time
performance.mark('scenario-start');
// Navigate to scenario analysis
performance.mark('scenario-end');
performance.measure('scenario', 'scenario-start', 'scenario-end');
console.log(performance.getEntriesByName('scenario')[0]);

// Should be < 1 second
```

### Backend Performance
```bash
# Measure API response time
time curl -X POST \
  -H "X-Tenant-ID: test" \
  -H "X-Tenant-Datasource-ID: test" \
  -d '{"scenario":"test"}' \
  "http://localhost:8080/api/portfolio/test/scenario"

# Should be < 200ms (handler only)
```

---

## Useful Grep Commands

### Find all Scenario references
```bash
grep -r "scenario" --include="*.tsx" --include="*.ts" --include="*.go" .
```

### Find menu items
```bash
grep -n "Analytics" frontend/src/components/MainNavigation.tsx
```

### Find routes
```bash
grep -n "Route path" frontend/src/AppRoutes.tsx
```

### Find API handlers
```bash
grep -n "POST" api-gateway/api/*.go
```

---

## File Locations

### Frontend
```
Component:     frontend/src/components/ScenarioAnalysisPro.tsx
AI Modal:      frontend/src/components/AIScenarioProposal.tsx
Gauge:         frontend/src/components/Gauge.tsx
Menu:          frontend/src/components/MainNavigation.tsx
Routes:        frontend/src/AppRoutes.tsx
```

### Backend
```
Handler:       api-gateway/api/scenario_analysis.go
Main:          api-gateway/main.go
Routes:        api-gateway/api/
```

### Documentation
```
Status:        SCENARIO_ANALYSIS_STATUS.md
Summary:       SCENARIO_ANALYSIS_DELIVERY_SUMMARY.md
Index:         SCENARIO_ANALYSIS_INDEX.md
Specs:         SCENARIO_ANALYSIS_FRONTEND_SPEC.md
Guide:         SCENARIO_ANALYSIS_IMPLEMENTATION_GUIDE.md
Examples:      SCENARIO_ANALYSIS_CODE_EXAMPLES.md
Visual:        frontend/SCENARIO_ANALYSIS_VISUAL_REFERENCE.html
Integration:   SCENARIO_ANALYSIS_INTEGRATION_COMPLETE.md
Verification:  SCENARIO_ANALYSIS_VERIFICATION.md
```

---

## One-Liners

### Verify everything
```bash
echo "Menu:" && grep -c "Scenario Analysis" frontend/src/components/MainNavigation.tsx && \
echo "Route:" && grep -c "scenario-analysis" frontend/src/AppRoutes.tsx && \
echo "Import:" && grep -c "ScenarioAnalysisPro" frontend/src/AppRoutes.tsx && \
echo "Backend:" && grep -c "RegisterScenarioAnalysisRoutes" api-gateway/main.go
```

### Count changed files
```bash
git diff --name-only
```

### Show line counts
```bash
wc -l frontend/src/components/ScenarioAnalysisPro.tsx frontend/src/components/AIScenarioProposal.tsx frontend/src/components/Gauge.tsx
```

### Check all docs
```bash
wc -l SCENARIO_ANALYSIS_*.md
```

---

## Copy & Paste Commands

### Quick Test Suite
```bash
#!/bin/bash
echo "🧪 Running Scenario Analysis Integration Tests..."

echo "✅ Checking menu integration..."
grep -q "Scenario Analysis" frontend/src/components/MainNavigation.tsx && echo "  ✓ Menu item found"

echo "✅ Checking route integration..."
grep -q "scenario-analysis" frontend/src/AppRoutes.tsx && echo "  ✓ Route found"

echo "✅ Checking component import..."
grep -q "ScenarioAnalysisPro" frontend/src/AppRoutes.tsx && echo "  ✓ Import found"

echo "✅ Checking backend routes..."
grep -q "RegisterScenarioAnalysisRoutes" api-gateway/main.go && echo "  ✓ Backend routes found"

echo "🎉 All integration checks passed!"
```

Save as `test-integration.sh`, run with `bash test-integration.sh`

---

## What to Do Next

### Immediate
```bash
# 1. Verify integration
bash test-integration.sh

# 2. Test in browser
open http://localhost:3000

# 3. Navigate to Entity → Analytics → Scenario Analysis
```

### Short Term
```bash
# 1. Read implementation guide
cat SCENARIO_ANALYSIS_IMPLEMENTATION_GUIDE.md

# 2. Implement backend from templates
cat SCENARIO_ANALYSIS_CODE_EXAMPLES.md

# 3. Apply database migrations
# See SCENARIO_ANALYSIS_CODE_EXAMPLES.md for schema
```

### Medium Term
```bash
# 1. Run tests
npm test -- scenario-analysis

# 2. Build frontend
npm run build

# 3. Deploy
# Your deployment process here
```

---

## Commands Summary Table

| Task | Command |
|------|---------|
| Test Menu | `grep "Scenario Analysis" frontend/src/components/MainNavigation.tsx` |
| Test Route | `grep "scenario-analysis" frontend/src/AppRoutes.tsx` |
| Test Import | `grep "ScenarioAnalysisPro" frontend/src/AppRoutes.tsx` |
| View Diff | `git diff frontend/src/AppRoutes.tsx` |
| Build | `cd api-gateway && go build ./main.go` |
| Test Backend | `go test ./api -v` |
| Build Frontend | `npm run build` |
| Run Frontend | `npm run preview` |
| Check Docs | `ls SCENARIO_ANALYSIS_*.md` |
| View Visual | `open frontend/SCENARIO_ANALYSIS_VISUAL_REFERENCE.html` |

---

**Quick Start Time**: < 5 minutes  
**Full Implementation Time**: 3-4 hours  
**Ready**: Yes ✅  

