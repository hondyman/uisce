# Portfolio Detail System - Complete Documentation

## Table of Contents

1. [Overview](#overview)
2. [Architecture](#architecture)
3. [Components](#components)
4. [API Integration](#api-integration)
5. [Styling & Theming](#styling--theming)
6. [Performance](#performance)
7. [Multi-Tenancy](#multi-tenancy)
8. [Testing](#testing)
9. [Troubleshooting](#troubleshooting)

---

## Overview

The **Portfolio Detail System** provides tenant-aware, interactive analysis of individual investment portfolios. Designed as a production-ready extension of the Risk & Compliance Dashboard, it enables portfolio managers to:

- **Monitor portfolio composition** across securities, sectors, and geographies
- **Analyze risk exposures** via factors, volatility, VaR, and stress scenarios
- **Track compliance status** against all applicable rules and constraints
- **Evaluate alternative scenarios** with precise PnL impact calculations
- **Make informed decisions** with real-time data integrated to the WASM execution layer

### Key Features

✅ **5 Integrated Data Views**: Overview, Holdings, Risk & Factors, Compliance, Scenario Analysis  
✅ **Tenant-Aware**: Multi-tenant isolation via DashboardContext + server-side RLS  
✅ **Real-Time Data**: React Query with optimized refresh strategies  
✅ **Dark Mode**: Full Tailwind CSS dark mode support  
✅ **Responsive Design**: Mobile-to-desktop adaptive layouts  
✅ **Type-Safe**: Full TypeScript coverage across API, hooks, and components  
✅ **Production-Ready**: Error handling, loading states, and graceful degradation  

---

## Architecture

### System Diagram

```
[Portfolio Manager]
        ↓
[Browser: React 18 + React Query]
        ├─ PortfolioDetailPage (orchestrator)
        │   ├─ Tabs: Overview | Holdings | Risk & Factors | Compliance | Scenarios
        │   └─ Data Management: usePortfolioData() hook
        │
        ├─ API Layer: portfolioApi.ts
        │   ├─ GET /portfolios/{id}/overview
        │   ├─ GET /portfolios/{id}/holdings
        │   ├─ GET /portfolios/{id}/risk
        │   ├─ GET /portfolios/{id}/compliance
        │   └─ GET /portfolios/{id}/scenarios
        │
        └─ Rendering Layer: React Components
            ├─ PortfolioCards (metrics display)
            │   ├─ PortfolioOverviewCard
            │   ├─ RiskSnapshotCard
            │   └─ ComplianceSnapshotCard
            │
            └─ PortfolioCharts (analysis & visualization)
                ├─ HoldingsTable (top 10 positions)
                ├─ SectorWeights (allocation breakdown)
                └─ ScenarioChart (PnL distribution)
        
[Backend: Go + WASM Engine + Postgres]
    ├─ Portfolio Service (holdings, valuation)
    ├─ Risk Service (factors, VaR, scenarios)
    ├─ Compliance Service (rule evaluation)
    └─ RLS Layer (tenant isolation)
```

### Data Flow

```
User navigates to /portfolios/PF-001?valuation_date=2024-01-15
        ↓
PortfolioDetailPage renders with useParams() retrieving portfolioId
        ↓
DashboardContext provides valuationDate + tenant_id
        ↓
usePortfolioData hook fires 5 parallel queries:
        ├─ usePortfolioOverview(PF-001, 2024-01-15)
        ├─ usePortfolioHoldings(PF-001, 2024-01-15)
        ├─ usePortfolioRisk(PF-001, 2024-01-15)
        ├─ usePortfolioCompliance(PF-001, 2024-01-15)
        └─ usePortfolioScenarios(PF-001, 2024-01-15)
        ↓
portfolioApi.ts sends HTTP requests w/ tenant_id query param
        ↓
Backend validates tenant access + computes portfolio metrics
        ↓
React Query caches responses (60s stale, 5m cache)
        ↓
Components render with data:
        ├─ Card components display summary metrics
        ├─ Tab content shows detailed analysis
        ├─ Charts visualize distributions
        └─ Tables provide drill-down capability
        ↓
User interacts (tab switch, export, etc.)
```

---

## Components

### PortfolioDetailPage (280 LOC)

**Purpose**: Main orchestrator combining all portfolio views  
**Location**: `src/pages/portfolio/PortfolioDetailPage.tsx`

**Key Features**:
- Tab-based navigation (Overview, Holdings, Risk & Factors, Compliance, Scenarios)
- Multi-tab content rendering with lazy evaluation
- Compliance breach display with severity indicators
- Breadcrumb navigation + back button
- Responsive grid layouts
- Loading + error states

**Props**: None (reads `portfolioId` from `useParams()`)

**State Management**:
```typescript
const { portfolioId } = useParams<{ portfolioId: string }>();
const { valuationDate } = useDashboardContext();
const portfolio = usePortfolioData(portfolioId, valuationDate);
const [activeTab, setActiveTab] = useState<TabType>('overview');
```

**Data Rendering**:
```typescript
// Parse with optional chaining
const overview = portfolio.overview.data;      // PortfolioOverview | undefined
const holdings = portfolio.holdings.data;      // HoldingsSummary | undefined
const isLoading = portfolio.isLoading;         // boolean

// Conditional rendering
{isLoading && <SkeletonLoader />}
{portfolio.isError && <ErrorBorder message={...} />}
{activeTab === 'overview' && <OverviewTabContent />}
```

**Tab Content Structure**:
- **Overview**: 3-card row (Portfolio Overview, Risk, Compliance) + Holdings + Scenarios
- **Holdings**: Sector breakdown + Country breakdown + Holdings table
- **Risk & Factors**: Risk snapshot + Factor exposure bar chart
- **Compliance**: Compliance card + Breach detail list (hard + soft)
- **Scenarios**: Scenario chart + Detailed results table

---

### PortfolioCards (100 LOC)

**Purpose**: Card components displaying portfolio metrics  
**Location**: `src/pages/portfolio/PortfolioCards.tsx`

#### PortfolioOverviewCard

Displays key portfolio metrics and performance summary.

**Props**:
```typescript
interface Props {
  data?: PortfolioOverview;
  isLoading?: boolean;
  error?: Error | null;
}
```

**Renders**:
- Portfolio name (large heading)
- AUM (with B/M formatting for readability)
- Currency code
- Strategy classification
- Benchmark index
- Inception + valuation dates
- YTD return vs benchmark + tracking error

**Example Data**:
```json
{
  "name": "Global Equities Fund",
  "aum_usd": 125000000,
  "currency": "USD",
  "strategy": "Long-Only Equities",
  "benchmark": "MSCI World",
  "ytd_return": 0.1250,
  "benchmark_return": 0.0890
}
```

#### RiskSnapshotCard

Displays risk metrics in a prominent card.

**Props**:
```typescript
interface Props {
  data?: RiskSnapshot;
  isLoading?: boolean;
  error?: Error | null;
}
```

**Renders**:
- Volatility (annualized %)
- Value-at-Risk 95% confidence (USD)
- Value-at-Risk 99% confidence (USD)
- Worst scenario impact (USD, red background)
- Beta, Sharpe ratio, Sortino ratio as secondary metrics

**Color Coding**:
- Green text for positive metrics (higher Sharpe = better)
- Red text for risk metrics (highlighted in red background)

#### ComplianceSnapshotCard

Displays compliance status summary.

**Props**:
```typescript
interface Props {
  data?: ComplianceSnapshot;
  isLoading?: boolean;
  error?: Error | null;
}
```

**Renders**:
- Total rules evaluated
- Passing rules count
- Pass rate % (with color coding: green ≥95%, amber ≥90%, red <90%)
- Badge count for hard breaches
- Badge count for soft breaches

**Example**:
```
Rules Evaluated: 127
Passing: 123 (96.85%)
❌ Hard Breaches: 1
⚠️ Soft Breaches: 2
```

---

### PortfolioCharts (150 LOC)

**Purpose**: Tables and visualizations for portfolio details  
**Location**: `src/pages/portfolio/PortfolioCharts.tsx`

#### HoldingsTable

Top 10 holdings with sector, weight, and performance.

**Props**:
```typescript
interface Props {
  data?: HoldingsSummary;
  isLoading?: boolean;
  error?: Error | null;
}
```

**Columns**:
| Column | Type | Format | Sort |
|--------|------|--------|------|
| Security | String | Ticker (Name) | Alphabetical |
| Sector | String | Tag badge | Category |
| Weight | Decimal | 8.50% | Numeric ↓ |
| Change 1D | Decimal | +1.25% or -0.50% | Color-coded |
| Change YTD | Decimal | +38.50% or -12.30% | Color-coded |

**Features**:
- Sorting by clicking headers
- Color-coded performance (green for gains, red for losses)
- Hover effects for row highlighting
- Scrollable on mobile
- Loading skeleton while fetching

**Rendering Example**:
```typescript
<table className="w-full">
  <thead>
    <tr className="border-b border-slate-200 dark:border-slate-700">
      <th>Security</th><th>Sector</th><th>Weight</th><th>1D</th><th>YTD</th>
    </tr>
  </thead>
  <tbody>
    {holdings?.map(h => (
      <tr key={h.security_id} className="hover:bg-slate-100 dark:hover:bg-slate-800">
        <td>{h.name}</td>
        <td><Badge>{h.sector}</Badge></td>
        <td>{(h.weight * 100).toFixed(2)}%</td>
        <td className={h.change_pct_1d > 0 ? 'text-green-600' : 'text-red-600'}>
          {h.change_pct_1d.toFixed(2)}%
        </td>
        <td className={h.change_pct_ytd > 0 ? 'text-green-600' : 'text-red-600'}>
          {h.change_pct_ytd.toFixed(2)}%
        </td>
      </tr>
    ))}
  </tbody>
</table>
```

#### SectorWeights

Horizontal progress bar visualization of sector allocation.

**Props**:
```typescript
interface Props {
  data?: Array<{ sector: string; weight: number }>;
  isLoading?: boolean;
}
```

**Renders**:
- Sector name on left
- Horizontal progress bar in center (width = weight %)
- Percentage value on right (aligned right)
- Color gradient from blue (first) to lighter shades

**Example Rendering**:
```
Technology    ████████████████░░  32.00%
Healthcare    ███████████░░░░░░░░  21.00%
Financials    █████████░░░░░░░░░░  18.00%
Industrials   ████████░░░░░░░░░░░  15.00%
Consumer      ███░░░░░░░░░░░░░░░░  8.00%
Utilities     ██░░░░░░░░░░░░░░░░░  4.00%
Energy        ░░░░░░░░░░░░░░░░░░░  2.00%
```

**Features**:
- Hover to highlight sector
- Sorted by weight descending
- Mobile-friendly (stacks on small screens)
- Consistent color coding across all instances

#### ScenarioChart

Bar chart showing scenario PnL distribution.

**Props**:
```typescript
interface Props {
  data?: ScenarioResult[];
  isLoading?: boolean;
  error?: Error | null;
}
```

**Renders**:
- Scenario name below bar
- Bar height proportional to absolute PnL value
- Red bars for PnL < 0 (downside scenarios)
- Green bars for PnL > 0 (upside scenarios)
- PnL value label above each bar (formatted as $XM)

**Example Chart**:
```
Scenario Data:
- Interest Rate Shock (+100bps): -$2.15M → Red bar
- Tech Sell-Off (-15%): -$3.40M → Red bar (taller)
- Market Rally (+10%): +$12.50M → Green bar (tallest)
```

**Features**:
- Automatic y-axis scaling
- Normalized bar heights (largest bar = 100%, others proportional)
- Y-axis labeled in millions USD
- Grid background for easy reading
- Hover tooltips with exact PnL amount

---

## API Integration

### `portfolioApi.ts` (120 LOC)

HTTP client layer handling all portfolio API requests.

**File Structure**:
```typescript
// Type Definitions (80 LOC)
interface PortfolioOverview { ... }
interface HoldingsSummary { ... }
interface RiskSnapshot { ... }
interface ComplianceSnapshot { ... }
interface ScenarioResults { ... }

// Fetch Functions (40 LOC)
export async function fetchPortfolioOverview(...) { }
export async function fetchPortfolioHoldings(...) { }
export async function fetchPortfolioRisk(...) { }
export async function fetchPortfolioCompliance(...) { }
export async function fetchPortfolioScenarios(...) { }
```

**API Base**:
```typescript
const API_BASE = process.env.REACT_APP_API_BASE_URL || 'http://localhost:8080/api';
```

**Common Query Parameters**:
```typescript
const queryParams = {
  tenant_id: tenantId,
  valuation_date: valuationDate,  // ISO-8601 or undefined (default: today)
};
```

**Endpoint Implementations**:

```typescript
// Example: fetchPortfolioOverview
export async function fetchPortfolioOverview(
  portfolioId: string,
  tenantId: string,
  valuationDate: string
): Promise<PortfolioOverview> {
  const url = new URL(`${API_BASE}/portfolios/${portfolioId}/overview`);
  url.searchParams.append('tenant_id', tenantId);
  url.searchParams.append('valuation_date', valuationDate);

  const response = await fetch(url.toString());
  
  if (!response.ok) {
    if (response.status === 403) {
      throw new Error('Unauthorized: Portfolio access denied');
    }
    if (response.status === 404) {
      throw new Error('Portfolio not found');
    }
    throw new Error(`API Error: ${response.status} ${response.statusText}`);
  }

  return response.json();
}
```

**Error Handling**:
- 200: Success → return parsed JSON
- 400: Bad request → throw Error with details
- 403: Unauthorized → throw Error 'Portfolio access denied'
- 404: Not found → throw Error 'Portfolio not found'
- 500: Server error → throw Error with status code

---

### React Query Hooks (`usePortfolioData.ts`)

**File Structure**:
```typescript
// Individual hooks (60 LOC)
export function usePortfolioOverview(...) { }
export function usePortfolioHoldings(...) { }
export function usePortfolioRisk(...) { }
export function usePortfolioCompliance(...) { }
export function usePortfolioScenarios(...) { }

// Combined hook (20 LOC)
export function usePortfolioData(...) { }
```

**Hook Pattern**:
```typescript
export function usePortfolioOverview(
  portfolioId: string | null,
  valuationDate: string
) {
  return useQuery({
    queryKey: ['portfolio', 'overview', portfolioId, valuationDate],
    queryFn: () => 
      fetchPortfolioOverview(portfolioId!, getTenantId(), valuationDate),
    enabled: !!portfolioId,  // Skip if portfolioId is null
    staleTime: 60 * 1000,    // 60 seconds
    cacheTime: 5 * 60 * 1000, // 5 minutes
    retry: 2,
    retryDelay: 1000,
  });
}
```

**Combined Hook** (`usePortfolioData`):
```typescript
export function usePortfolioData(
  portfolioId: string | null,
  valuationDate: string
) {
  const overview = usePortfolioOverview(portfolioId, valuationDate);
  const holdings = usePortfolioHoldings(portfolioId, valuationDate);
  const risk = usePortfolioRisk(portfolioId, valuationDate);
  const compliance = usePortfolioCompliance(portfolioId, valuationDate);
  const scenarios = usePortfolioScenarios(portfolioId, valuationDate);

  return {
    overview,
    holdings,
    risk,
    compliance,
    scenarios,
    isLoading: [overview, holdings, risk, compliance, scenarios]
      .some(q => q.isLoading),
    isError: [overview, holdings, risk, compliance, scenarios]
      .some(q => q.isError),
    error: overview.error || 
           holdings.error || 
           risk.error || 
           compliance.error || 
           scenarios.error,
  };
}
```

**Usage Pattern**:
```typescript
// In component
const portfolio = usePortfolioData(portfolioId, valuationDate);

if (portfolio.isLoading) return <LoadingSkeleton />;
if (portfolio.isError) return <ErrorAlert error={portfolio.error} />;

// All data available and typed
const { aum, name } = portfolio.overview.data;
```

---

## Styling & Theming

### Tailwind CSS Configuration

All components use Tailwind CSS classes with dark mode support.

**Dark Mode Strategy**:
- Prefix with `dark:` for dark mode styles
- Use semantic color names (e.g., `slate-900`, `blue-600`)
- Ensure sufficient contrast for accessibility

**Example Component**:
```typescript
<div className="bg-white dark:bg-slate-900 text-slate-900 dark:text-white p-4 rounded-lg border border-slate-200 dark:border-slate-800">
  <h3 className="text-lg font-bold mb-2">Portfolio Overview</h3>
  <p className="text-sm text-slate-600 dark:text-slate-400">Data loaded successfully</p>
</div>
```

### Color Palette

| Color | Light | Dark | Usage |
|-------|-------|------|-------|
| Background | white (bg-white) | slate-900 (dark:bg-slate-900) | Page/card backgrounds |
| Text Primary | slate-900 | white | Headings, body text |
| Text Secondary | slate-600 | slate-400 | Captions, timestamps |
| Border | slate-200 | slate-800 | Card borders, table lines |
| Primary (Blue) | #137fec (blue-600) | #0084ff (blue-400) | Links, active states |
| Success (Green) | #10b981 (green-500) | #34d399 (green-400) | Positive values |
| Error (Red) | #ef4444 (red-500) | #f87171 (red-400) | Negative values, breaches |
| Warning (Amber) | #f59e0b (amber-500) | #fbbf24 (amber-400) | Soft breaches, cautions |

### Spacing System

```
xs: 0.25rem (4px)     → Tiny gaps, small padding
sm: 0.5rem (8px)      → Small components
md: 1rem (16px)       → Standard padding
lg: 1.5rem (24px)     → Section spacing
xl: 2rem (32px)       → Large gaps
2xl: 3rem (48px)      → Hero sections
```

### Typography

**Heading Hierarchy**:
- `text-4xl font-black` → Page titles (e.g., portfolio name)
- `text-2xl font-bold` → Section headers
- `text-lg font-bold` → Subsection headers
- `text-base font-semibold` → Card titles
- `text-sm font-medium` → Table headers
- `text-xs` → Helper text, captions

**Example**:
```typescript
<h1 className="text-4xl font-black text-slate-900 dark:text-white">
  Global Equities Fund
</h1>
<h2 className="text-2xl font-bold text-slate-800 dark:text-slate-100 mt-4">
  Holdings Breakdown
</h2>
```

### Responsive Grid

```typescript
<ConsoleGrid columns={3} gap="lg">
  {/* Auto-adjusts: 3 columns on desktop, 2 on tablet, 1 on mobile */}
</ConsoleGrid>
```

**Breakpoints**:
- Mobile: < 768px (1 column)
- Tablet: 768px - 1024px (2 columns)
- Desktop: > 1024px (3-4 columns)

---

## Performance

### React Query Caching Strategy

```typescript
staleTime: 60 * 1000        // Mark as stale after 60 seconds
cacheTime: 5 * 60 * 1000    // Keep in memory for 5 minutes
retry: 2                     // Retry failed requests 2x
retryDelay: 1000            // Wait 1 second between retries
```

**Behavior Timeline**:
```
T=0s: User navigates to portfolio page
      ↓ Query fires, fetches from API
T=0.2s: Data arrives, cached, rendered
      ↓
T=60s: Data marked "stale" (still displayed)
      ↓ User stays on page
T=120s: User switches tabs (new queries may fire)
      ↓ Background refetch for stale data
T=300s (5m): Cache entries removed from memory
      ↓ Next load will refetch from API
```

### Query Deduplication

If multiple components request same data simultaneously:
```typescript
// Component A: usePortfolioOverview(id, date)
// Component B: usePortfolioOverview(id, date) 
// → Only 1 API call made; both components receive same cached result
```

### Lazy Tab Loading

Only active tab's content renders/queries:
```typescript
{activeTab === 'holdings' && (
  <HoldingsTab />  // Only mounts when Holdings tab active
)}
{activeTab === 'scenarios' && (
  <ScenariosTab />  // Only mounts when Scenarios tab active
)}
```

### Optimistic UI Updates

Loading skeletons show immediately while data fetches:
```typescript
{portfolio.holdings.isLoading && <TableSkeleton />}
{!portfolio.holdings.isLoading && <HoldingsTable data={...} />}
```

---

## Multi-Tenancy

### Tenant Isolation

All API requests include `tenant_id` query parameter:

```typescript
const url = `/api/portfolios/${id}/overview?tenant_id=${tenantId}&valuation_date=...`;
```

**Backend Responsibility** (Go server):
- Verify authenticated user belongs to tenant_id
- Apply Row-Level Security (RLS) to all queries
- Return 403 Forbidden if unauthorized
- Log all access for audit trail

### DashboardContext Integration

```typescript
import { useDashboardContext } from '@/contexts/DashboardContext';

export function PortfolioDetailPage() {
  const { tenant, valuationDate } = useDashboardContext();
  
  // tenant.id automatically passed to queries
  const portfolio = usePortfolioData(portfolioId, valuationDate);
  
  return <div>Portfolio for {tenant.name}</div>;
}
```

### Multi-Tenant Validation

```typescript
if (!tenant?.id) {
  return <NoTenantError message="Please select a tenant first" />;
}

const portfolio = usePortfolioData(portfolioId, valuationDate);
// tenant.id automatically included in API calls
```

---

## Testing

### Unit Tests

**Component Tests**:
```typescript
import { render, screen } from '@testing-library/react';
import { PortfolioOverviewCard } from './PortfolioCards';

test('renders portfolio name from data', () => {
  const data = { name: 'Global Equities Fund', ... };
  render(<PortfolioOverviewCard data={data} isLoading={false} />);
  
  expect(screen.getByText('Global Equities Fund')).toBeInTheDocument();
});

test('renders loading skeleton while fetching', () => {
  render(<PortfolioOverviewCard data={undefined} isLoading={true} />);
  
  expect(screen.getByTestId('skeleton-loader')).toBeInTheDocument();
});

test('renders error message on fetch failure', () => {
  const error = new Error('Portfolio not found');
  render(
    <PortfolioOverviewCard 
      data={undefined} 
      isLoading={false}
      error={error}
    />
  );
  
  expect(screen.getByText('Portfolio not found')).toBeInTheDocument();
});
```

**Hook Tests**:
```typescript
import { renderHook, waitFor } from '@testing-library/react';
import { usePortfolioData } from './usePortfolioData';

test('fetches all 5 portfolio data sources', async () => {
  const { result } = renderHook(() => 
    usePortfolioData('PF-001', '2024-01-15')
  );
  
  expect(result.current.isLoading).toBe(true);
  
  await waitFor(() => {
    expect(result.current.isLoading).toBe(false);
  });
  
  expect(result.current.overview.data).toBeDefined();
  expect(result.current.holdings.data).toBeDefined();
});
```

### Integration Tests

```typescript
test('portfolio page loads and displays (E2E simulation)', async () => {
  render(<PortfolioDetailPage />, { 
    wrapper: AllProvidersWrapper 
  });
  
  // Wait for data to load
  await waitFor(() => {
    expect(screen.getByText('Global Equities Fund')).toBeInTheDocument();
  });
  
  // Verify all tabs render
  expect(screen.getByText('Overview')).toBeInTheDocument();
  expect(screen.getByText('Holdings')).toBeInTheDocument();
  expect(screen.getByText('Risk & Factors')).toBeInTheDocument();
  
  // Switch to Holdings tab
  userEvent.click(screen.getByText('Holdings'));
  expect(screen.getByText('Sector Breakdown')).toBeInTheDocument();
});
```

### Manual Testing Checklist

- [ ] Portfolio loads with mock portfolioId
- [ ] All 5 API endpoints called on initial render
- [ ] Tab navigation switches between views without navigation
- [ ] Compliance breaches display with correct severity colors
- [ ] Scenarios chart shows red (negative) and green (positive) bars
- [ ] Dark mode toggle switches all colors
- [ ] Mobile layout stacks properly on small screens
- [ ] Error state shows user-friendly message on API failure
- [ ] Loading skeletons appear while fetching
- [ ] Multi-tenant isolation: switching tenant in context reloads portfolio
- [ ] Data persists when switching tabs and returning

---

## Troubleshooting

### Portfolio Returns 404

**Symptom**: "Portfolio not found" error when loading `/portfolios/PF-001`  
**Causes**:
- Portfolio ID doesn't exist in database
- Portfolio belongs to different tenant
- Backend RLS rejecting access

**Solution**:
1. Verify portfolio ID is correct
2. Confirm authenticated user belongs to portfolio's tenant
3. Check backend logs: `docker logs backend | grep RLS`

### No Data Displays (Blank Page)

**Symptom**: Page renders but no portfolio data visible  
**Causes**:
- React Query queries disabled (`enabled: false`)
- Tenant context not set
- API base URL misconfigured

**Solution**:
1. Open browser DevTools → Network tab
2. Verify API calls to `/api/portfolios/*/...` endpoints
3. Check `REACT_APP_API_BASE_URL` environment variable
4. Confirm `useDashboardContext()` returns valid tenant

### Tabs Not Switching

**Symptom**: Clicking tab buttons doesn't change active tab  
**Causes**:
- State update not triggered
- Click handler not attached
- Conditional rendering broken

**Solution**:
1. Verify `setActiveTab` state setter called on button click
2. Check `activeTab` passed to tab content conditional
3. Verify tab components render without console errors

### Dark Mode Not Working

**Symptom**: Dark mode toggle doesn't change component colors  
**Causes**:
- Missing `dark:` prefixed classes
- Tailwind not configured for dark mode
- CSS not compiled

**Solution**:
1. Enable dark mode in `tailwind.config.js`:
   ```javascript
   module.exports = {
     darkMode: 'class',  // or 'media'
     theme: { ... }
   }
   ```
2. Rebuild CSS: `npm run build:css`
3. Verify `dark:` classes present in compiled CSS

### Performance Issues (Slow Tab Switching)

**Symptom**: Noticeable lag when clicking between tabs  
**Causes**:
- Multiple component re-renders
- Large data sets not paginated
- React Query cache not configured

**Solution**:
1. Profile with React DevTools Profiler
2. Check React Query DevTools cache sizes
3. Implement pagination for large tables (> 100 rows)
4. Memoize expensive components with `React.memo`

### API Calls Not Including Tenant

**Symptom**: Backend returns 403, even with valid credentials  
**Causes**:
- Tenant not provided in API request
- `useDashboardContext` returns undefined

**Solution**:
1. Verify `DashboardProvider` wraps entire app
2. Check `useDashboardContext()` returns `{ tenant, valuationDate }`
3. Inspect Network tab: verify `?tenant_id=...` in URL
4. Add console.log in portfolioApi.ts fetch functions

### Compliance Breaches Not Showing

**Symptom**: Compliance section blank even when breaches exist  
**Causes**:
- Breach arrays empty in response
- Conditional rendering not evaluating correctly
- Breach data structure mismatch

**Solution**:
1. Check API response: `console.log(portfolio.compliance.data)`
2. Verify `hard_breaches` and `soft_breaches` arrays have items
3. Confirm data types match TypeScript interfaces
4. Check `ComplianceBreach` component renders without errors

---

## Deployment Checklist

Production deployment requires:

- [ ] All 5 backend API endpoints implemented and tested
- [ ] Environment variables configured (`.env.production`)
- [ ] React Query cache settings adjusted for production latency
- [ ] TypeScript strict mode enabled
- [ ] ESLint pass all checks
- [ ] Dark mode CSS validated in production build
- [ ] API error types handled gracefully
- [ ] Loading skeletons and fallbacks in place
- [ ] Multi-tenant isolation tested with multiple tenants
- [ ] Mobile responsive design verified
- [ ] Performance optimizations applied (React.memo, lazy loading)
- [ ] Security audit: no exposed credentials, proper CORS
- [ ] Analytics tracking for page views and errors
- [ ] Monitoring configured for API latency and error rates
- [ ] Documentation updated for backend development team

---

## File Reference

| File | Lines | Purpose |
|------|-------|---------|
| portfolioApi.ts | 120 | HTTP client + TypeScript types |
| usePortfolioData.ts | 80 | React Query hooks |
| PortfolioCards.tsx | 100 | Card components (Overview, Risk, Compliance) |
| PortfolioCharts.tsx | 150 | Table + chart visualizations |
| PortfolioDetailPage.tsx | 280 | Main orchestrator + tab navigation |
| INTEGRATION_GUIDE.md | ~150 | Backend integration instructions |
| README.md | This file | Complete documentation |

**Total**: ~950 lines of production code

---

## Support

For questions or issues:

1. **Check API responses** in browser DevTools Network tab
2. **Inspect console** for TypeScript/runtime errors
3. **Review React Query DevTools** cache and query states
4. **Verify backend** logs for RLS or data issues
5. **Test with mock data** to isolate frontend vs backend problems

