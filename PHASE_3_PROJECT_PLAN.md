# 🚀 Phase 3: Advanced Scenario Analysis & Stress Testing - Project Plan

**Status**: 🟢 **INITIATED**  
**Date Started**: February 22, 2026  
**Target Completion**: 2 weeks  
**Quality Standard**: Enterprise Grade (⭐⭐⭐⭐⭐) + TypeScript Strict Mode  

---

## Executive Overview

Building an institutional-grade **Scenario Analysis & Stress Testing Platform** with real-time execution, collaborative annotations, and multi-portfolio comparison capabilities.

**Core Focus**: Empower portfolio managers to:
- ✅ Configure custom stress test scenarios with interactive sliders
- ✅ Run simulations across 15+ portfolios in parallel (WASM engine)
- ✅ Compare multiple scenarios side-by-side with live results
- ✅ Collaborate in real-time with annotations and comments
- ✅ Export comprehensive analysis reports

---

## Phase 3 Scope

### 5 Major Components to Build

#### 1️⃣ **ScenarioConfigDialog** 
- Interactive modal for defining stress scenarios
- Real-time slider inputs (Equity, Rates, Volatility, Credit Spreads)
- Portfolio selection (All vs. Currently Compared)
- WASM performance optimization
- **Status**: 0% Complete

#### 2️⃣ **SimulationProgress Screen**
- Live execution dashboard
- Progress bars with portfolio processing status
- Real-time results preview table
- Abort simulation capability
- Configuration summary sidebar
- **Status**: 0% Complete

#### 3️⃣ **MultiScenarioComparison Dashboard**
- Clustered bar charts (multiple scenarios)
- Comparative data grid with variance highlighting
- Filter and sort capabilities
- PnL/Volatility/VaR metric toggles
- **Status**: 0% Complete

#### 4️⃣ **CollaborativeAnnotations Panel**
- Right sidebar for team annotations
- Real-time comments with user avatars
- Marker-based cell references
- @mention support
- Sharable insights dashboard
- **Status**: 0% Complete

#### 5️⃣ **Streaming Results Table**
- 15+ row data grid
- Live update animations
- Performance optimized rendering
- Responsive columns
- **Status**: 0% Complete

### Supporting Infrastructure

#### Types & Interfaces
```typescript
interface StressScenario {
  id: string;
  name: string;
  equityMove: number;        // -100 to +100 %
  rateShift: number;         // -500 to +500 bps
  volatilityChange: number;  // -100 to +200 %
  creditSpreadWidening: number; // -100 to +500 bps
  portfoliosIncluded: PortfolioId[];
  createdAt: Date;
  createdBy: UserId;
}

interface SimulationResult {
  simulationId: string;
  portfolioId: string;
  scenarioId: string;
  pnl: number;           // Simulated PnL in millions
  confidence: number;    // 0-100 %
  status: 'success' | 'processing' | 'failed';
  processingTime: number; // ms
}

interface Annotation {
  id: string;
  userId: string;
  userName: string;
  timestamp: Date;
  text: string;
  cellReference?: string;  // e.g., "Tech Growth - COVID PnL"
  mentions?: UserId[];
}
```

#### API Endpoints (Backend Required)
```
POST   /api/v1/simulations              - Start simulation
GET    /api/v1/simulations/:id          - Get simulation status
GET    /api/v1/simulations/:id/results  - Stream results
DELETE /api/v1/simulations/:id          - Abort simulation
POST   /api/v1/annotations              - Add annotation
GET    /api/v1/annotations              - Get annotations
```

#### Custom Hooks
```
useSimulation()              - Start & manage simulations
useSimulationResults()       - Stream results with WebSocket
useAnnotations()             - Fetch & manage annotations
useScenarioComparison()      - Compare multiple scenarios
useMultiplayer()             - Real-time collaboration state
```

---

## Implementation Roadmap

### Week 1: Foundation & Dialogs
- [x] Create TypeScript interfaces & types
- [ ] Build ScenarioConfigDialog component
- [ ] Create useSimulation hook
- [ ] Set up WebSocket for streaming results

### Week 2: Dashboards & Collaboration
- [ ] Build SimulationProgress screen
- [ ] Build MultiScenarioComparison dashboard
- [ ] Build CollaborativeAnnotations panel
- [ ] Integrate real-time collaboration

### Week 3: Testing & Polish
- [ ] Unit tests (all components)
- [ ] Integration tests (data flow)
- [ ] E2E tests (user workflows)
- [ ] Performance optimization
- [ ] Accessibility audit

### Week 4: Production & Deployment
- [ ] TypeScript strict compilation
- [ ] Production build verification
- [ ] Staging deployment
- [ ] QA sign-off
- [ ] Production deployment

---

## Technical Architecture

### Component Tree

```
PortfolioDetailPage
├── Tabs Navigation (Updated)
│   └── "Scenario Stress Tests" (NEW)
│
├── ScenarioStessTestsTab
│   ├── ScenarioSelector
│   ├── LaunchSimulationButton
│   │
│   ├── (If Simulating)
│   │   └── SimulationProgress
│   │       ├── Sidebar: Configuration Summary
│   │       └── Main: Live Results Preview
│   │
│   ├── (If Complete)
│   │   └── MultiScenarioComparison
│   │       ├── Sidebar: Aggregated Impact
│   │       ├── Main: Charts + Data Grid
│   │       └── Right: CollaborativeAnnotations
│   │
│   └── Dialogs
│       └── ScenarioConfigDialog
└── MUI Components Throughout
```

### State Management

```typescript
// Component-level state (React hooks)
const [scenarios, setScenarios] = useState<StressScenario[]>();
const [activeSimulation, setActiveSimulation] = useState<SimulationRun>();
const [results, setResults] = useState<SimulationResult[]>();
const [annotations, setAnnotations] = useState<Annotation[]>();

// Global context (if needed)
SimulationContext: {
  activeRun: SimulationRun;
  results: SimulationResult[];
  isLive: boolean;
}

CollaborationContext: {
  activeUsers: User[];
  activeAnnotations: Annotation[];
}
```

---

## File Structure (New)

```
frontend/src/
├── pages/portfolio/
│   ├── scenarios/
│   │   ├── ScenarioStressTestsTab.tsx       (Main container)
│   │   ├── ScenarioConfigDialog.tsx         (Modal)
│   │   ├── SimulationProgress.tsx           (Live dashboard)
│   │   ├── MultiScenarioComparison.tsx      (Results dashboard)
│   │   ├── CollaborativeAnnotations.tsx     (Right panel)
│   │   ├── StreamingResultsTable.tsx        (Data grid)
│   │   └── index.ts                         (Exports)
│   │
│   └── PortfolioDetailPage.tsx              (Updated tabs)
│
├── hooks/
│   ├── useSimulation.ts                     (NEW)
│   ├── useSimulationResults.ts              (NEW)
│   ├── useAnnotations.ts                    (NEW)
│   ├── useScenarioComparison.ts             (NEW)
│   ├── useMultiplayer.ts                    (NEW - Real-time collaboration)
│   └── [existing hooks...]
│
├── types/
│   ├── scenarios.ts                         (NEW - Interfaces)
│   ├── simulation.ts                        (NEW - Simulation types)
│   ├── collaboration.ts                     (NEW - Annotation types)
│   └── [existing types...]
│
├── services/
│   ├── simulationService.ts                 (NEW - API calls)
│   ├── scenarioService.ts                   (NEW - Scenario CRUD)
│   ├── annotationService.ts                 (NEW - Annotation API)
│   └── [existing services...]
│
└── components/
    └── [shared components as needed]
```

---

## API Contracts (Backend Specification)

### Simulation Execution

**Start Simulation**
```
POST /api/v1/simulations
Request: {
  scenarioId: string;
  portfolioIds: string[];
}
Response: {
  simulationId: string;
  status: "queued";
  estimatedDuration: number; // seconds
}
```

**Get Live Results (WebSocket)**
```
WS /api/v1/simulations/:id/stream
Message: {
  type: "progress" | "result" | "complete" | "error";
  progress?: number;
  portfoliosProcessed?: number;
  totalPortfolios?: number;
  result?: SimulationResult;
}
```

### Collaboration

**Add Annotation**
```
POST /api/v1/annotations
Request: {
  simulationId: string;
  userId: string;
  text: string;
  cellReference?: string;
  mentions?: string[];
}
Response: {
  annotationId: string;
  timestamp: Date;
}
```

---

## UI/UX Patterns

### Material UI Component Usage

```typescript
// ScenarioConfigDialog
<Dialog open={open} maxWidth="sm">
  <DialogTitle>Configure Stress Test Scenario</DialogTitle>
  <DialogContent>
    <TextField label="Scenario Name" />
    <Slider label="Equity Market Move" min={-100} max={100} />
    <Slider label="Interest Rate Shift" min={-500} max={500} />
    <ToggleButtonGroup>
      <ToggleButton>All Portfolios</ToggleButton>
      <ToggleButton>Currently Compared</ToggleButton>
    </ToggleButtonGroup>
  </DialogContent>
  <DialogActions>
    <Button>Cancel</Button>
    <Button variant="contained">Run Simulation</Button>
  </DialogActions>
</Dialog>

// SimulationProgress
<Box display="flex">
  <Box flex={1}>
    <LinearProgress variant="determinate" value={78} />
    <DataGrid rows={results} columns={columns} />
  </Box>
  <Box width={300} borderLeft={1}>
    {/* Configuration Summary */}
  </Box>
</Box>

// MultiScenarioComparison
<Box>
  <BarChart data={scenarioResults} />
  <DataGrid rows={portfolios} columns={columns} />
  <Box borderLeft={1}>
    {/* Annotations Panel */}
  </Box>
</Box>
```

### Dark Mode Support
- ✅ All components via `useTheme()`
- ✅ Charts: Dynamic colors (light/dark)
- ✅ Tables: Proper contrast
- ✅ Modals: Theme-aware backgrounds

---

## Performance Targets

| Metric | Target | Current | Status |
|--------|--------|---------|--------|
| Dialog open time | < 300ms | TBD | 🔄 |
| Results update (WebSocket) | < 100ms | TBD | 🔄 |
| 15-portfolio grid render | < 500ms | TBD | 🔄 |
| Memory footprint | < 50MB | TBD | 🔄 |
| Bundle size (code-split) | < 80KB | TBD | 🔄 |

---

## Testing Strategy

### Unit Tests
```typescript
// ScenarioConfigDialog.test.tsx
- Slider input ranges validated
- Form submission triggers callback
- Dark mode colors apply
- Dialog keyboard navigation

// SimulationProgress.test.tsx
- Progress bar updates correctly
- Results stream renders live
- Abort button works
- WebSocket errors handled

// MultiScenarioComparison.test.tsx
- Charts render with data
- Grid sorts/filters work
- Metric toggle switches data
- Annotations display correctly

// Hooks
- useSimulation: Start/abort/cleanup
- useSimulationResults: Stream parsing
- useAnnotations: CRUD operations
```

### Integration Tests
```typescript
// Scenario workflow
1. Open dialog → Configure → Run → Get results → Compare
2. Add annotation → See in real-time → Share with team
3. Multiple users collaborating on same simulation
```

### E2E Tests (Playwright)
```typescript
// Complete user journeys
- Portfolio Manager: Configure scenario → Monitor execution → Export report
- Risk Analyst: Compare 3 scenarios → Add findings → Share with compliance
- Compliance Officer: View annotated results → Approve stress test
```

---

## Quality Checklist

### Code Quality
- [ ] 100% TypeScript strict mode
- [ ] 100% type coverage
- [ ] Zero `any` types
- [ ] All props documented
- [ ] Error handling comprehensive
- [ ] Loading states for all async
- [ ] Empty states handled

### Performance
- [ ] Component render < 200ms
- [ ] WebSocket updates < 100ms
- [ ] Memory leaks prevented
- [ ] Bundle size optimized
- [ ] Lazy loading for heavy components

### UX/Accessibility
- [ ] Dark mode support verified
- [ ] Keyboard navigation works
- [ ] WCAG AA colors verified
- [ ] Loading skeletons shown
- [ ] Error messages clear
- [ ] Real-time indicators visible

### Production Readiness
- [ ] No console errors
- [ ] No warnings on build
- [ ] TypeScript strict compilation ✅
- [ ] Tests passing ✅
- [ ] Error reporting configured
- [ ] Performance monitoring set up
- [ ] Deployment checklist complete

---

## Success Criteria

### Scope Completion
- ✅ All 5 components built & tested
- ✅ 100% Material UI (zero Tailwind)
- ✅ Production code quality
- ✅ Full TypeScript type safety
- ✅ Real-time collaboration
- ✅ WebSocket integration
- ✅ Comprehensive documentation

### Quality Metrics
- ✅ Zero TypeScript errors
- ✅ 100% prop typing
- ✅ All error states handled
- ✅ 90%+ test coverage
- ✅ < 500ms load time
- ✅ Dark mode perfect
- ✅ Responsive all breakpoints

### Deployment
- ✅ Staging verified
- ✅ QA sign-off
- ✅ Performance benchmarked
- ✅ Security reviewed
- ✅ Production ready
- ✅ Monitoring configured

---

## Timeline

| Week | Sprint | Focus | Deliverable |
|------|--------|-------|-------------|
| 1 | Foundation | Types, Dialogs, Hooks | ScenarioConfigDialog ✓ |
| 2 | Dashboards | Progress, Comparison, Collab | SimulationProgress ✓ |
| 2 | Integration | Real-time, WebSocket | MultiScenarioComparison ✓ |
| 3 | Testing | Unit, Integration, E2E | 100% Test Coverage ✓ |
| 4 | Polish | Performance, Accessibility | Production Ready ✓ |

---

## Dependencies & Prerequisites

### Already Available ✅
- React 18.2.0
- Material UI 5.18.0
- TypeScript 5.4.5
- Recharts 2.15.4
- usePortfolioData hook

### Need to Confirm ❓
- [ ] Backend simulation API ready?
- [ ] WebSocket infrastructure set up?
- [ ] WASM engine available?
- [ ] Multi-user session management?
- [ ] Real-time collaboration backend?
- [ ] Annotation persistence layer?

### Optional Enhancements 💡
- Real-time collaboration via WebSocket
- Multi-user presence indicators
- Annotation versioning & history
- Simulation caching
- Batch scenario execution
- PDF export with annotations

---

## Risk Mitigation

| Risk | Impact | Mitigation |
|------|--------|-----------|
| WebSocket latency | User experience | Implement polling fallback |
| Memory leaks (streaming) | Performance degradation | Strict cleanup in useEffect |
| WASM compilation errors | Simulation failure | Graceful error handling |
| Real-time sync issues | Data inconsistency | Optimistic updates + reconciliation |
| Large result sets | Slow rendering | Virtual scrolling + pagination |

---

## Documentation Deliverables

- [ ] Component API documentation
- [ ] Hook usage guide
- [ ] WebSocket protocol specification
- [ ] Type definitions reference
- [ ] Integration guide
- [ ] Troubleshooting guide
- [ ] Deployment checklist

---

## Next Steps

1. **Confirm Backend APIs** - Verify simulation & collaboration endpoints
2. **Create Type Definitions** - Build TypeScript interfaces
3. **Build ScenarioConfigDialog** - Start Week 1 work
4. **Parallel: Create Hooks** - useSimulation, useSimulationResults
5. **Integration Phase** - Connect components & APIs

---

**Phase 3 Status**: 🟢 Ready to Begin  
**Quality Target**: ⭐⭐⭐⭐⭐ Enterprise Grade  
**Deployment Date**: TBD (2 weeks from start)  

---

**Ready to Start Week 1: Foundation & Dialogs Phase?**
