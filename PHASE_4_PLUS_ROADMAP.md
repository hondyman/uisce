# Phase 4+ Roadmap - Risk & Compliance Console

**Current Status**: React Frontend Complete ✅ → Next: Go Backend Implementation

---

## Phase 4, Session 2: Go Backend Implementation (Week 1)

### Objectives
Implement all 11 API endpoints + database queries to serve React frontend

### Deliverables

#### Task 1: Dashboard Aggregate Queries (Day 1)
- [ ] SQL query: GetDashboardComplianceSummary
  - SUM rules, pass rate %, hard/soft breach counts
  - Aggregated by severity (HARD/SOFT/INFO)
  - Handler: GET /api/dashboard/compliance
- [ ] SQL query: GetDashboardRiskSummary
  - AVG volatility, VaR percentiles, worst scenario
  - Exposure breakdown by asset class (equity/rates/credit/fx)
  - Handler: GET /api/dashboard/risk
- [ ] SQL query: GetDashboardSparklines
  - 7-day rolling aggregations (daily)
  - Metrics: pass_rate, hard_breaches, soft_breaches, volatility, etl_duration
  - Handler: GET /api/dashboard/sparklines
- [ ] SQL query: GetDashboardETLHealth
  - Last ETL run record (full details)
  - Success rate, avg duration, total runs
  - Handler: GET /api/dashboard/etl-health
- [ ] SQL query: GetDashboardAlerts
  - All active breaches (hard/soft by rule)
  - Scenario losses (top PnL impacts)
  - Recent ETL failures
  - Handler: GET /api/dashboard/alerts

**Reference**: See `GO_BACKEND_IMPLEMENTATION.md` → Dashboard Endpoints section

---

#### Task 2: ETL Run Endpoints (Day 1)
- [ ] SQL query: ListETLRuns with filters (tenant_id, status, date_range)
  - Return 200 records by default
  - Pagination support (limit parameter)
  - Handler: GET /api/etl-runs
- [ ] SQL query: GetETLRun by ID
  - Full record with error_summary
  - Handler: GET /api/etl-runs/{id}

**Reference**: See `GO_BACKEND_IMPLEMENTATION.md` → ETL Endpoints section

---

#### Task 3: WASM Version Endpoints (Day 2)
- [ ] SQL query: ListWASMVersions by module_name
  - Return versions sorted by build_time DESC
  - Include is_active flag
  - Handler: GET /api/wasm-versions
- [ ] SQL mutation: ActivateWASMVersion
  - Set target version is_active = true
  - Set all other versions is_active = false (for same module)
  - Handler: POST /api/wasm-versions/{id}/activate
  - Success response: Updated version record

**Reference**: See `GO_BACKEND_IMPLEMENTATION.md` → WASM Endpoints section

---

#### Task 4: Lineage Endpoints (Day 2)
- [ ] SQL query: GetRuleLineage with filters (date_range, portfolio_id)
  - Historical evaluations: status, metric_value, threshold_value
  - Include etl_run_id for traceability
  - Handler: GET /api/rules/{ruleId}/lineage
- [ ] SQL query: GetScenarioLineage with filters (date_range, portfolio_id)
  - Historical P&L results
  - Include etl_run_id for traceability
  - Handler: GET /api/scenarios/{scenarioId}/lineage

**Reference**: See `GO_BACKEND_IMPLEMENTATION.md` → Lineage Endpoints section

---

#### Task 5: Chi Router Wiring (Day 3)
```go
// pseudocode structure
func setupConsoleRoutes(r chi.Router) {
  dashboardHandler := &DashboardHandler{DB: db}
  etlHandler := &ETLHandler{DB: db}
  wasmHandler := &WASMHandler{DB: db}
  lineageHandler := &LineageHandler{DB: db}

  r.Route("/api", func(r chi.Router) {
    // Dashboard (5 endpoints)
    r.Get("/dashboard/compliance", dashboardHandler.ComplianceSummary)
    r.Get("/dashboard/risk", dashboardHandler.RiskSummary)
    r.Get("/dashboard/sparklines", dashboardHandler.Sparklines)
    r.Get("/dashboard/etl-health", dashboardHandler.ETLHealth)
    r.Get("/dashboard/alerts", dashboardHandler.Alerts)

    // ETL (2 endpoints)
    r.Get("/etl-runs", etlHandler.ListRuns)
    r.Get("/etl-runs/{id}", etlHandler.GetRun)

    // WASM (2 endpoints)
    r.Get("/wasm-versions", wasmHandler.ListVersions)
    r.Post("/wasm-versions/{id}/activate", wasmHandler.ActivateVersion)

    // Lineage (2 endpoints)
    r.Get("/rules/{ruleId}/lineage", lineageHandler.RuleLineage)
    r.Get("/scenarios/{scenarioId}/lineage", lineageHandler.ScenarioLineage)
  })
}
```

---

#### Task 6: Error Handling & Validation (Day 3)
- [ ] Validate required query parameters (tenant_id, valuation_date, etc.)
- [ ] Return 400 Bad Request if missing
- [ ] Return 404 Not Found if resource doesn't exist
- [ ] Return 500 Internal Server Error on DB failure
- [ ] Log all errors for debugging
- [ ] Add request logging (tenant_id, endpoint, duration)

---

#### Task 7: Testing (Day 4)
- [ ] Unit tests for each handler
- [ ] Integration tests (handler + DB)
- [ ] Test with curl
  ```bash
  curl http://localhost:8080/api/dashboard/compliance?tenant_id=tenant-1&valuation_date=2024-01-15
  ```
- [ ] Verify response JSON matches schema
- [ ] Test error cases (missing params, invalid ID)

**Acceptance Criteria**:
- All 11 endpoints tested and working
- All responses valid JSON
- All error codes correct
- All query parameters validated
- Response times <2s

---

### Resources
- Backend handbook: `GO_BACKEND_IMPLEMENTATION.md`
- Component expectations: `RISK_AND_COMPLIANCE_CONSOLE.md`
- API contracts: `GO_BACKEND_IMPLEMENTATION.md` → Response sections

### Estimate: 3-4 days for experienced backend engineer

---

## Phase 4, Session 3: Testing & Integration (Week 2)

### Objectives
Verify React + Go integration, automated testing, performance validation

### Deliverables

#### Task 1: Integration Testing (Day 1)
- [ ] Start Go backend on localhost:8080
- [ ] Start React frontend on localhost:5173
- [ ] Navigate to http://localhost:5173/console/dashboard
- [ ] Verify all KPIs load and match backend data
- [ ] Test with different valuationDate
- [ ] Test with different tenants
- [ ] Verify localStorage persistence
- [ ] Test network error scenarios (e.g., kill backend)

---

#### Task 2: E2E Tests (Playwright) (Day 2)
```typescript
// Example E2E test
test('Dashboard loads and displays KPIs', async ({ page }) => {
  await page.goto('http://localhost:5173/console/dashboard');
  
  // Wait for dashboard to load
  await page.waitForSelector('[data-testid="compliance-card"]');
  
  // Verify compliance summary
  const totalRules = await page.textContent('[data-testid="total-rules"]');
  expect(totalRules).toMatch(/\d+/);
  
  // Verify sparklines
  const sparklines = await page.locator('svg').count();
  expect(sparklines).toBeGreaterThan(0);
});
```

Playwright tests:
- [ ] Dashboard loads in <2s
- [ ] All cards render
- [ ] Links navigate correctly
- [ ] Sidebar collapses on mobile
- [ ] Tenant switcher updates dashboard
- [ ] Search finds items
- [ ] ETL runs list loads
- [ ] WASM versions load
- [ ] Lineage charts render

---

#### Task 3: Performance Testing (Day 2)
- [ ] Profile React components (React DevTools Profiler)
- [ ] Dashboard render time <500ms
- [ ] Large tables (200+ rows) render smoothly
- [ ] Sparklines don't cause jank
- [ ] Profile API calls (Network tab)
- [ ] Dashboard APIs total <2s
- [ ] Tables load in <1s
- [ ] Check bundle size
- [ ] React Query: Check query cache hits

---

#### Task 4: Unit Tests - React Components (Day 3)
```typescript
// Example component test
import { render, screen } from '@testing-library/react';
import { StatusBadge } from './StatusBadge';

test('StatusBadge renders PASS with green background', () => {
  render(<StatusBadge status="PASS" />);
  const chip = screen.getByRole('button');
  expect(chip).toHaveStyle('background-color: #2ECC71');
});
```

Unit tests:
- [ ] StatusBadge (all 8 statuses)
- [ ] SeverityBadge (all 3 severities)
- [ ] TrendChart (with/without threshold)
- [ ] Sparkline (data loading)
- [ ] SparklineCard (trend calculation)
- [ ] ETLRunTable (pagination, sorting)
- [ ] WASMVersionTable (activate button)
- [ ] ConsoleBreadcrumbs (link rendering)
- [ ] DashboardHome (hook integration)

---

#### Task 5: Unit Tests - API Hooks (Day 3)
```typescript
// Example hook test
import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { useComplianceSummary } from './useComplianceSummary';

test('useComplianceSummary fetches data', async () => {
  const queryClient = new QueryClient();
  const wrapper = ({ children }) => (
    <QueryClientProvider client={queryClient}>
      {children}
    </QueryClientProvider>
  );
  
  const { result } = renderHook(
    () => useComplianceSummary('tenant-1', '2024-01-15'),
    { wrapper }
  );
  
  await waitFor(() => {
    expect(result.current.isLoading).toBe(false);
    expect(result.current.data).toBeDefined();
  });
});
```

Hook tests:
- [ ] useComplianceSummary (query key, enabled condition)
- [ ] useRiskSummary (query key, enabled condition)
- [ ] useSparklines (query key)
- [ ] useETLHealth (query key)
- [ ] useAlerts (query key, enabled condition)
- [ ] useETLRuns (pagination, filters)
- [ ] useWASMVersions (module filter)
- [ ] useActivateWASMVersion (mutation, invalidation)

**Acceptance Criteria**:
- All tests passing (100% pass rate)
- Coverage >80%
- E2E tests cover critical paths
- Performance metrics meet targets
- No console warnings/errors

---

### Estimate: 3-4 days for QA engineer + developer

---

## Phase 4, Session 4: Production Deployment (Week 3)

### Objectives
Deploy to staging, validate, get sign-off, prepare for production

### Deliverables

#### Task 1: Staging Environment Setup (Day 1)
- [ ] Deploy React frontend to staging server/CDN
- [ ] Deploy Go backend to staging server
- [ ] Configure staging database
- [ ] Set environment variables (API_BASE_URL, etc.)
- [ ] Configure CORS (frontend domain allowed)
- [ ] Configure logging for staging
- [ ] Set up error monitoring (Sentry, etc.)

---

#### Task 2: Smoke Testing (Day 1)
**Minimal testing to verify deployment**
- [ ] Navigate to staging console URL
- [ ] Dashboard loads without errors
- [ ] All KPIs display
- [ ] API calls succeed (Network tab)
- [ ] No 404 errors
- [ ] No CORS errors
- [ ] No TypeScript errors (if source-mapped)
- [ ] Mobile view works
- [ ] Tenant switching works

---

#### Task 3: UAT (User Acceptance Testing) (Day 2)
**Have users test the real system**
- [ ] Compliance team reviews dashboard
- [ ] Risk team reviews risk metrics
- [ ] Ops team tests ETL runs page
- [ ] Engineers test WASM version management
- [ ] All users test lineage explorers
- [ ] Collect feedback & bug reports
- [ ] Document any issues

---

#### Task 4: Load Testing (Day 2)
**Verify system handles expected traffic**
```bash
# Example with Apache Bench
ab -n 1000 -c 100 http://staging.console.example.com/api/dashboard/compliance
```

- [ ] 500 concurrent users
- [ ] Dashboard average response time <2s
- [ ] No errors under load
- [ ] Database handles concurrent queries
- [ ] No memory leaks after load test

---

#### Task 5: Security Review (Day 3)
- [ ] Verify authentication (if applicable)
- [ ] Verify authorization (tenants can't see each other's data)
- [ ] No sensitive data in logs
- [ ] No API keys exposed in frontend code
- [ ] HTTPS enabled
- [ ] CORS configured correctly
- [ ] SQL injection prevention validated
- [ ] XSS prevention validated

---

#### Task 6: Documentation & Handoff (Day 3)
- [ ] Update deployment guide for operations team
- [ ] Document API changes
- [ ] Document database changes
- [ ] Provide runbooks for common issues
- [ ] Provide on-call rotation guide
- [ ] Brief operations team on new console

---

### Acceptance Criteria for Production
- [ ] All tests passing (unit, integration, E2E)
- [ ] UAT completed with sign-off
- [ ] Load testing passed
- [ ] Security review passed
- [ ] Performance targets met
- [ ] Documentation complete
- [ ] On-call guide provided
- [ ] Rollback plan in place

---

## Phase 5+: Future Enhancements

### High Priority (Next Quarter)

#### 1. Dark Mode Support
- [ ] Add MUI theme switcher to ConsoleTopBar
- [ ] Save preference to localStorage
- [ ] All components use theme colors
- [ ] Test all components in dark mode

#### 2. Export to CSV
- [ ] Add export button to each DataGrid
- [ ] Export selected rows or all rows
- [ ] Include filters/date range in export
- [ ] Test with large datasets

#### 3. Advanced Filtering
- [ ] Add filter panel to DataGrids
- [ ] Multi-select status/severity
- [ ] Date range picker
- [ ] Portfolio multi-select
- [ ] Save filter presets

#### 4. Real-time Updates (WebSocket)
- [ ] WebSocket connection for ETL updates
- [ ] Dashboard KPIs refresh in real-time
- [ ] Alerts push notification
- [ ] Sparklines update live

### Medium Priority (Following Quarter)

#### 5. Dashboard Customization
- [ ] Drag-to-reorder dashboard cards
- [ ] Hide/show cards
- [ ] Save layout per user
- [ ] Preset layouts (compliance view, risk view, ops view)

#### 6. Advanced Analytics
- [ ] Drill-down from dashboard to rule details
- [ ] Risk factor decomposition
- [ ] Scenario analysis (what-if modeling)
- [ ] Backtesting results
- [ ] Rule performance trends

#### 7. Collaboration Features
- [ ] Comments/annotations on lineage charts
- [ ] Share custom dashboards
- [ ] Alerts with team routing
- [ ] Audit trail (who viewed what)

### Low Priority (Future)

#### 8. Mobile App
- [ ] React Native version for iOS/Android
- [ ] Offline support for cached data
- [ ] Push notifications for alerts
- [ ] Touch-optimized UI

#### 9. Machine Learning Features
- [ ] Anomaly detection in sparklines
- [ ] Predictive alerts
- [ ] Suggest filter values
- [ ] Recommend analysis paths

#### 10. Multi-Language Support
- [ ] Internationalization (i18n) setup
- [ ] Translate UI to additional languages
- [ ] RTL language support

---

## Resource Planning

### Phase 4, Session 2 (Go Backend)
- **Lead**: Senior Backend Engineer
- **Support**: Database Engineer
- **Duration**: 4 days
- **Effort**: 32 hours

### Phase 4, Session 3 (Testing)
- **Lead**: QA Engineer
- **Support**: Frontend Developer
- **Duration**: 4 days
- **Effort**: 32 hours

### Phase 4, Session 4 (Deployment)
- **Lead**: DevOps Engineer
- **Support**: All teams
- **Duration**: 3 days
- **Effort**: 24 hours

### Total Phase 4: 11 days, 88 hours

---

## Success Metrics

### Go Backend (Session 2)
- [ ] All 11 endpoints implemented and tested
- [ ] 100% API contract compliance
- [ ] Response times <2s p95
- [ ] 0 errors on integration test

### Testing (Session 3)
- [ ] E2E test coverage >90%
- [ ] Unit test coverage >80%
- [ ] Load test: 500 concurrent users, <2s p95
- [ ] All security review items passed

### Deployment (Session 4)
- [ ] 0 production incidents in first week
- [ ] UAT sign-off from all stakeholders
- [ ] Dashboard average load time <1.5s
- [ ] <1% error rate
- [ ] All on-call runbooks tested

---

## Communication Plan

### Weekly Status Updates
- **Monday**: Sprint planning (what's being worked on)
- **Wednesday**: Mid-week check-in (blockers, progress)
- **Friday**: Sprint review (completed work, learnings)

### Stakeholder Notifications
- **Phase Complete**: Email all stakeholders with summary
- **Blocking Issues**: Immediate alert to tech lead
- **UAT Ready**: Invite users to test
- **Go-Live**: All-hands notification

---

## Risk Mitigation

### Risk: Database performance on aggregations
- **Mitigation**: Pre-compute aggregations nightly, cache in Redis
- **Contingency**: Add database indexes on common filter columns

### Risk: React Query cache invalidation issues
- **Mitigation**: Comprehensive testing of mutation → invalidation patterns
- **Contingency**: Manual query invalidation triggers in UI

### Risk: Multi-tenant data leakage
- **Mitigation**: Security review specifically for tenant isolation
- **Contingency**: Tenant context validation on every endpoint

### Risk: Performance under load
- **Mitigation**: Load testing with realistic concurrent users
- **Contingency**: Add request rate limiting, implement batch queries

---

## Next Immediate Steps (For Product Manager)

1. **Assign backend engineer** to implement 11 endpoints (using `GO_BACKEND_IMPLEMENTATION.md` as spec)
2. **Assign QA engineer** to set up Playwright tests
3. **Assign DevOps engineer** to prepare staging environment
4. **Schedule kickoff meeting** for Session 2 (go backend work starts)
5. **Brief stakeholders** on timeline: 3 weeks to staging, 4+ weeks to production

---

## Links to Reference Docs

- **React Components**: `RISK_AND_COMPLIANCE_CONSOLE.md`
- **Go Endpoints**: `GO_BACKEND_IMPLEMENTATION.md`
- **Router Setup**: `CONSOLE_ROUTER_SETUP.md`
- **QA Testing**: `CONSOLE_QA_CHECKLIST.md`
- **Phase 4 Summary**: `PHASE_4_SESSION_1_SUMMARY.md`

---

**Ready to start Phase 4, Session 2? Begin with backend implementation using `GO_BACKEND_IMPLEMENTATION.md` as the blueprint. 🚀**
