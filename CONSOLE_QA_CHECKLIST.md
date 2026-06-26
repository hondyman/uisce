# Risk & Compliance Console - Verification & Testing Checklist

## Pre-Deployment QA Guide

Use this checklist to verify all components work correctly before staging deployment.

---

## ✅ Phase 1: Component Verification

### Design System Components

- [ ] **StatusBadge Component**
  - [ ] Renders PASS with green background
  - [ ] Renders FAIL with red background
  - [ ] Renders WARN with yellow background
  - [ ] Renders INFO with blue background
  - [ ] All 8 statuses present in color map
  - [ ] Uses MUI Chip correctly
  - [ ] Text color is white for contrast

- [ ] **SeverityBadge Component**
  - [ ] Renders HARD with dark red (#C0392B)
  - [ ] Renders SOFT with orange (#F39C12)
  - [ ] Renders INFO with navy (#2980B9)
  - [ ] Uses MUI Chip correctly

- [ ] **TrendChart Component**
  - [ ] Line renders with data points
  - [ ] X-axis rotated -45 degrees
  - [ ] Y-axis renders with proper formatting
  - [ ] ReferenceLine shows threshold when provided
  - [ ] Tooltip appears on hover
  - [ ] Chart responsive to container width
  - [ ] No animation on initial render (isAnimationActive=false)

- [ ] **Sparkline Component**
  - [ ] Renders as 40px height chart
  - [ ] No axes or labels visible
  - [ ] Just the line visible
  - [ ] Responsive to parent container

- [ ] **SparklineCard Component**
  - [ ] Title displays at top
  - [ ] Latest value displays in h6 weight
  - [ ] Trend percentage shows (e.g., "+5%")
  - [ ] Trend arrow points up/down correctly
  - [ ] Green color for positive trend
  - [ ] Red color for negative trend
  - [ ] Sparkline visible below values

---

### Data Grid Components

- [ ] **ETLRunTable Component**
  - [ ] 7 columns visible (date, status, rules, scenarios, WASM, orchestrator, duration)
  - [ ] StatusBadge renders in status column
  - [ ] Duration calculated from start/end times
  - [ ] Row can be clicked without error
  - [ ] Pagination controls visible
  - [ ] Can change page size (10/25/50/100)
  - [ ] Scrollbar appears at 600px height limit
  - [ ] Loading state shows "Loading..." when fetching

- [ ] **ETLRunDetail Component**
  - [ ] Displays etl_run_id in title
  - [ ] Shows all 8 fields (id, tenant, date, start, end, status, rules, scenarios, WASM, orchestrator, error)
  - [ ] Error summary visible when present
  - [ ] StatusBadge shows correct color
  - [ ] Duration calculated correctly
  - [ ] Error summary in <pre> tag with background

- [ ] **WASMVersionTable Component**
  - [ ] 6 columns visible (version, hash, time, uri, active, actions)
  - [ ] build_hash truncated to 8 chars
  - [ ] artifact_uri is clickable link
  - [ ] "✓ Yes" shows for active versions
  - [ ] "No" shows for inactive versions
  - [ ] Activate button appears only for inactive
  - [ ] Button shows "Activating..." during mutation
  - [ ] is_active updates after activation

- [ ] **RuleLineageTable Component**
  - [ ] 6 columns visible (date, portfolio, status, metric, threshold, run_id)
  - [ ] StatusBadge renders correctly
  - [ ] Metric and threshold values formatted to 6 decimals
  - [ ] etl_run_id is clickable link
  - [ ] Pagination works

- [ ] **ScenarioLineageTable Component**
  - [ ] 4 columns visible (date, portfolio, pnl, run_id)
  - [ ] P&L positive values green
  - [ ] P&L negative values red
  - [ ] etl_run_id is clickable link
  - [ ] Pagination works

---

### Layout Components

- [ ] **ConsoleSidebar Component**
  - [ ] 280px width drawer visible on left
  - [ ] 5 sections visible (Dashboard, Compliance, Risk, ETL, Admin)
  - [ ] Each section has correct subsections
  - [ ] Total 18 navigation items
  - [ ] ListItemButton for each item
  - [ ] Uppercase headers for sections
  - [ ] Subitems indented (pl=2)

- [ ] **ConsoleTopBar Component**
  - [ ] White AppBar with black text
  - [ ] GlobalSearch on left
  - [ ] TenantSwitcher on right
  - [ ] Both components visible together

- [ ] **TenantSwitcher Component**
  - [ ] Dropdown shows 3 mock tenants
  - [ ] Current tenant displayed
  - [ ] Can select different tenant
  - [ ] Selection persists in localStorage
  - [ ] onChange updates state

- [ ] **GlobalSearch Component**
  - [ ] Autocomplete input renders
  - [ ] SearchIcon visible
  - [ ] Can type search query
  - [ ] Mock results appear
  - [ ] Groups by category
  - [ ] Click result navigates

- [ ] **ConsoleBreadcrumbs Component**
  - [ ] All items display in order
  - [ ] Last item not a link
  - [ ] Previous items are clickable
  - [ ] NavigateNextIcon separator visible

- [ ] **ConsoleLayout Component**
  - [ ] Sidebar always visible (left)
  - [ ] TopBar always visible (top)
  - [ ] Content area centered and responsive
  - [ ] Container max width 1280px (xl)
  - [ ] Proper spacing (py=3)

---

## ✅ Phase 2: Page Verification

### DashboardHome Page

- [ ] **Initial Load**
  - [ ] All 5 hooks called (compliance, risk, sparklines, etl, alerts)
  - [ ] Loading states show for each
  - [ ] No console errors

- [ ] **Compliance Card**
  - [ ] Displays total_rules count
  - [ ] Displays pass_rate as percentage
  - [ ] Shows hard/soft breach counts
  - [ ] Green background if >90% pass rate
  - [ ] Orange background if <90% pass rate
  - [ ] Status color correct

- [ ] **Risk Card**
  - [ ] Displays avg_volatility to 4 decimals
  - [ ] Displays avg_var_95 to 4 decimals
  - [ ] Displays avg_var_99 to 4 decimals
  - [ ] Worst scenario box visible with light red background
  - [ ] Scenario name displays
  - [ ] Scenario P&L color coded (red if negative)

- [ ] **4 SparklineCards**
  - [ ] Pass Rate card shows latest % + trend + sparkline
  - [ ] Hard Breaches card shows count + trend + sparkline
  - [ ] Volatility card shows value + trend + sparkline
  - [ ] ETL Duration card shows milliseconds + trend + sparkline
  - [ ] All trends calculate correctly (% change last vs previous)

- [ ] **ETL Health Card**
  - [ ] Last run status displays with StatusBadge
  - [ ] Duration shown in seconds
  - [ ] Rules evaluated count displays
  - [ ] Success rate % displays

- [ ] **Alerts Card**
  - [ ] If no alerts: Shows success alert "No active alerts"
  - [ ] If alerts present: Shows each type (hard breach, soft breach, scenario loss, ETL failure)
  - [ ] Hard breaches show rule_code, metric_value, threshold_value, portfolio_id
  - [ ] Soft breaches show rule_code, metric_value, threshold_value, portfolio_id
  - [ ] Scenario losses show scenario name + P&L
  - [ ] ETL failures show error message + timestamp
  - [ ] Alert severity colors correct (error/warning/info)

- [ ] **Responsive Layout**
  - [ ] Mobile (xs): All cards stack vertically
  - [ ] Tablet (md): 2x2 grid for KPIs, 4x1 for sparklines
  - [ ] Desktop (lg): Proper 3-column layout
  - [ ] No horizontal scrolling

- [ ] **Error Handling**
  - [ ] If hook returns error: Shows error message
  - [ ] If hook returns empty data: Shows empty state
  - [ ] If network fails: Shows retry option (via React Query)

---

### ETLRunsPage

- [ ] **List Mode (no :runId param)**
  - [ ] Breadcrumbs show "ETL & Execution > ETL Runs"
  - [ ] ETLRunTable renders
  - [ ] Clicking row calls onRowClick handler
  - [ ] Browser history updates to `/console/etl/runs/:id`

- [ ] **Detail Mode (:runId param)**
  - [ ] Breadcrumbs show "ETL & Execution > ETL Runs > {id}"
  - [ ] ETLRunDetail renders
  - [ ] Shows all run details
  - [ ] Back navigation works

---

### WASMVersionsPage

- [ ] **Page Loads**
  - [ ] Breadcrumbs show "ETL & Execution > WASM Versions"
  - [ ] WASMVersionTable renders
  - [ ] Shows versions for "risk-engine" module (or specified module)
  - [ ] Activate buttons visible for inactive versions

---

### RuleLineagePage

- [ ] **URL Params**
  - [ ] :ruleId extracted from URL (e.g., "MAX_ISSUER_5")
  - [ ] Breadcrumbs show rule hierarchy

- [ ] **Content**
  - [ ] TrendChart displays metric_value vs threshold_value
  - [ ] Chart clearly shows threshold line
  - [ ] RuleLineageTable displays below chart
  - [ ] Table shows evaluation history

---

### ScenarioLineagePage

- [ ] **URL Params**
  - [ ] :scenarioId extracted from URL (e.g., "equity-shock-20")
  - [ ] Breadcrumbs show scenario hierarchy

- [ ] **Content**
  - [ ] TrendChart displays P&L over time
  - [ ] No threshold line (not applicable for P&L)
  - [ ] ScenarioLineageTable displays below chart
  - [ ] Table shows historical P&L

---

## ✅ Phase 3: React Query Integration

### Query Caching

- [ ] **Query Keys**
  - [ ] Dashboard queries use nested structure: `['dashboard', 'compliance', tenantId, valuationDate]`
  - [ ] ETL queries use: `['etl-runs', 'list', filters]`
  - [ ] WASM queries use: `['wasm-versions', moduleName]`
  - [ ] Lineage queries use: `['rule-lineage', ruleId, filters]`

- [ ] **Stale Time**
  - [ ] Dashboard: 5 minutes stale
  - [ ] Sparklines: 1 minute stale
  - [ ] Tables: 2 minutes stale
  - [ ] After 5 minutes, background refetch occurs

- [ ] **Enabled Conditions**
  - [ ] Dashboard queries only run when tenantId and valuationDate truthy
  - [ ] Table queries only run when module_name or ruleId truthy
  - [ ] No unnecessary queries on mount

- [ ] **Query Invalidation (Mutations)**
  - [ ] After WASM activate: `queryKeys.wasm.all` invalidated
  - [ ] After ETL complete: `queryKeys.etl.all` and `queryKeys.dashboard.all` invalidated
  - [ ] Related queries refetch automatically

---

### Error Handling

- [ ] **Network Errors**
  - [ ] 500 errors retry 2 times with exponential backoff
  - [ ] After retries fail, error displayed in component
  - [ ] User sees "Failed to load" message

- [ ] **Validation Errors**
  - [ ] 400 errors do NOT retry
  - [ ] Immediate error message shown
  - [ ] User sees actionable error (missing tenant_id, etc.)

- [ ] **Loading States**
  - [ ] `isLoading` true on initial fetch
  - [ ] `isFetching` true on background refetch
  - [ ] Component shows loading spinner/skeleton while fetching

---

## ✅ Phase 4: Multi-Tenant Context

- [ ] **Tenant Switcher**
  - [ ] Load page: Reads from localStorage
  - [ ] Change tenant: Value updates in localStorage
  - [ ] Refresh page: Selected tenant persists
  - [ ] Key in localStorage: "selectedTenant"

- [ ] **Tenant in API Calls**
  - [ ] All dashboard hooks read tenantId from component state (via localStorage)
  - [ ] All API calls include `tenant_id` query parameter
  - [ ] Switching tenant triggers refetch of all queries

- [ ] **Multi-Tenant Data**
  - [ ] Compliance data for tenant-1 ≠ Compliance data for tenant-2
  - [ ] When tenant changes, dashboard updates with new data
  - [ ] No data leakage between tenants

---

## ✅ Phase 5: API Integration

### Mock Data (Development)

- [ ] **Dashboard Endpoints**
  - [ ] `useComplianceSummary` fetches from `/api/dashboard/compliance`
  - [ ] `useRiskSummary` fetches from `/api/dashboard/risk`
  - [ ] `useSparklines` fetches from `/api/dashboard/sparklines`
  - [ ] `useETLHealth` fetches from `/api/dashboard/etl-health`
  - [ ] `useAlerts` fetches from `/api/dashboard/alerts`

- [ ] **Entity Endpoints**
  - [ ] `useETLRuns` fetches from `/api/etl-runs`
  - [ ] `useETLRun` fetches from `/api/etl-runs/{id}`
  - [ ] `useWASMVersions` fetches from `/api/wasm-versions`
  - [ ] `useActivateWASMVersion` POSTs to `/api/wasm-versions/{id}/activate`
  - [ ] `useRuleLineage` fetches from `/api/rules/{id}/lineage`
  - [ ] `useScenarioLineage` fetches from `/api/scenarios/{id}/lineage`

### Backend Integration (After Go endpoints implemented)

- [ ] **Status Codes**
  - [ ] 200 OK on success
  - [ ] 400 Bad Request on missing parameters
  - [ ] 404 Not Found on resource not found
  - [ ] 500 Internal Server Error on DB failure

- [ ] **Response Format**
  - [ ] All responses are valid JSON
  - [ ] Response structure matches React component expectations
  - [ ] Numbers formatted correctly (decimals, not strings)
  - [ ] Timestamps in ISO 8601 format (YYYY-MM-DDTHH:MM:SSZ)

- [ ] **Performance**
  - [ ] Dashboard loads within 2 seconds
  - [ ] Tables load within 3 seconds
  - [ ] No N+1 queries in backend
  - [ ] Aggregations computed efficiently

---

## ✅ Phase 6: Accessibility & UX

### Accessibility

- [ ] **Semantic HTML**
  - [ ] Headings use proper h1-h6 hierarchy
  - [ ] Tables use <table> with <thead>/<tbody>
  - [ ] Buttons are <button> elements
  - [ ] Links are <a> elements

- [ ] **ARIA Labels**
  - [ ] Chips have aria-label describing status/severity
  - [ ] Buttons have descriptive text
  - [ ] Icons have aria-label or are hidden from screen readers

- [ ] **Keyboard Navigation**
  - [ ] Tab through all interactive elements
  - [ ] Sidebar items focusable and clickable with Enter
  - [ ] Search autocomplete keyboard accessible
  - [ ] DataGrid keyboard navigable

- [ ] **Color Contrast**
  - [ ] All text readable on background (WCAG AA)
  - [ ] Status badges have sufficient contrast
  - [ ] Disabled buttons visually distinct

### Responsive Design

- [ ] **Mobile (xs, <600px)**
  - [ ] Sidebar collapses or drawer
  - [ ] Cards stack vertically
  - [ ] Tables scrollable horizontally if needed
  - [ ] Touch targets minimum 48x48px

- [ ] **Tablet (md, 600-960px)**
  - [ ] Sidebar visible or collapsible
  - [ ] 2-column layouts for cards
  - [ ] Tables fit screen

- [ ] **Desktop (lg, >960px)**
  - [ ] Sidebar always visible
  - [ ] 3-column layouts
  - [ ] Full table width

- [ ] **Dark Mode (if implemented)**
  - [ ] All components render correctly
  - [ ] Colors sufficient contrast
  - [ ] Theme applies to all MUI components

---

## ✅ Phase 7: Performance

### Bundle Size

- [ ] **React Query**: Included in package.json
- [ ] **Recharts**: Included in package.json
- [ ] **MUI**: Included in package.json
- [ ] **Total bundle <500KB** (with gzip)

### Runtime Performance

- [ ] **No Console Warnings**
  - [ ] No React strict mode warnings
  - [ ] No MUI warnings
  - [ ] No React Query warnings
  - [ ] No TypeScript errors

- [ ] **Render Performance**
  - [ ] Dashboard renders in <500ms
  - [ ] Tables render large datasets without lag
  - [ ] Sparklines animate smoothly
  - [ ] No janky scrolling

- [ ] **Memory Leaks**
  - [ ] No memory increase on page navigation
  - [ ] No memory leak on component unmount
  - [ ] Queries garbage collected after 10 minutes
  - [ ] No circular references in hooks

---

## ✅ Phase 8: Type Safety

### TypeScript Strict Mode

- [ ] **No `any` Types**
  - [ ] All function parameters typed
  - [ ] All component props typed
  - [ ] All hook return types typed

- [ ] **No Type Errors**
  - [ ] `tsc --noEmit` passes without errors
  - [ ] VS Code shows no red squiggles
  - [ ] All imports resolve correctly

- [ ] **Interfaces Correct**
  - [ ] ComplianceSummary interface matches API response
  - [ ] RiskSummary interface matches API response
  - [ ] ETLRun interface matches API response
  - [ ] All entities match backend types

---

## 🔍 Manual Testing Scenarios

### Scenario 1: Daily Dashboard Review
1. Navigate to `/console/dashboard`
2. Verify all KPI cards load
3. Check pass rate % matches expected
4. Check breach counts are accurate
5. Verify sparklines show 7-day trend
6. Check alerts list for today

**Expected**: All data loads and displays correctly ✅

### Scenario 2: ETL Run Investigation
1. Navigate to `/console/etl/runs`
2. View list of recent ETL runs
3. Click on a run to see details
4. Verify all fields populate
5. Check error summary if present
6. Navigate back to list

**Expected**: List loads, detail loads, back navigation works ✅

### Scenario 3: WASM Version Management
1. Navigate to `/console/etl/wasm`
2. View list of WASM versions (e.g., "risk-engine")
3. Identify inactive version
4. Click Activate
5. See button show "Activating..."
6. Verify activation completes

**Expected**: Activation succeeds, button state updates ✅

### Scenario 4: Rule Lineage Analysis
1. Navigate to `/console/compliance/rules/MAX_ISSUER_5/lineage`
2. View TrendChart showing metric vs threshold
3. Scroll to RuleLineageTable
4. Verify historical evaluations display
5. Check status colors correct

**Expected**: Chart loads, table loads, data matches ✅

### Scenario 5: Scenario P&L Tracking
1. Navigate to `/console/risk/scenarios/equity-shock-20/lineage`
2. View TrendChart showing P&L trend
3. Scroll to ScenarioLineageTable
4. Verify P&L values color-coded (green/red)
5. Check historical trend

**Expected**: Chart and table load, colors correct ✅

### Scenario 6: Multi-Tenant Switching
1. Open TenantSwitcher dropdown
2. Select "Acme Capital"
3. Dashboard updates with Acme data
4. Refresh page
5. Verify same tenant selected
6. Switch to "BlackRock Test"
7. Dashboard updates with BlackRock data

**Expected**: Switching works, data changes, persists on refresh ✅

### Scenario 7: Global Search
1. Click GlobalSearch input
2. Type "MAX_ISSUER"
3. Results show relevant rules
4. Click result to navigate
5. Verify correct page loads

**Expected**: Search works, navigation works ✅

---

## 🐛 Common Issues & Fixes

### Issue: "Cannot read property 'map' of undefined"
**Cause**: Hook returns loading/error state before data
**Fix**: Check `isLoading` or `error` before rendering
```typescript
{data?.items?.map(...)}  // Safe access
```

### Issue: "React Query key collision"
**Cause**: Two queries with same key but different params
**Fix**: Use proper key structure
```typescript
['dashboard', 'compliance', tenantId, valuationDate]  // Unique per params
```

### Issue: "Infinite refetch loop"
**Cause**: Query invalidation without proper conditions
**Fix**: Only invalidate when necessary
```typescript
queryClient.invalidateQueries({
  queryKey: queryKeys.wasm.all,
  exact: true,  // Only exact match
})
```

### Issue: "Component doesn't update on tenant change"
**Cause**: tenantId not re-queried when changed
**Fix**: Ensure tenantId in query key
```typescript
// Query key must include tenantId to refetch on change
['dashboard', tenantId, valuationDate]
```

### Issue: "Memory leak warning on unmount"
**Cause**: Unfinished fetch on component unmount
**Fix**: React Query handles this automatically - should not occur

---

## ✅ Final Acceptance Criteria

- [ ] All 38 files exist in correct locations
- [ ] No TypeScript errors or warnings
- [ ] All pages load without console errors
- [ ] All components render correctly
- [ ] Dashboard fully functional with mock/real data
- [ ] All links navigate correctly
- [ ] Multi-tenant context works
- [ ] React Query caching works
- [ ] Error handling works
- [ ] Mobile responsive
- [ ] Accessible (keyboard + screen reader)
- [ ] No memory leaks
- [ ] Performance acceptable (<2s page load)
- [ ] All documentation complete

---

## 📋 Sign-Off Checklist

- [ ] Frontend: All components verified ✅
- [ ] Backend: All endpoints implemented ✅
- [ ] Database: All queries working ✅
- [ ] Integration: React + Go communication verified ✅
- [ ] Security: No data leakage between tenants ✅
- [ ] Performance: Acceptable load times ✅
- [ ] Documentation: Complete and accurate ✅
- [ ] Ready for staging: YES / NO

**Status**: _____________  
**Date**: _______________  
**Verified By**: _____________

---

Done! Use this checklist before releasing to staging. 🎉
