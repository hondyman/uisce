# Portfolio Detail System - Production Integration Guide

## Overview

The **Portfolio Detail System** provides tenant-aware, real-time deep-dive analysis of individual portfolios. Built on React 18 + React Query + TypeScript, it sits cleanly on top of the WASM-driven execution fabric and integrates with the existing console design system.

## Quick Start (3 Steps)

### Step 1: Environment Configuration

Ensure your `.env` file includes:

```env
REACT_APP_API_BASE_URL=http://localhost:8080/api
REACT_APP_ENABLE_PORTFOLIO_MODE=true
```

### Step 2: Router Configuration

Add the portfolio route to your app router:

```typescript
import { PortfolioDetailPage } from '@/pages/portfolio';

const appRoutes = [
  // ... existing routes
  {
    path: '/portfolios/:portfolioId',
    element: <PortfolioDetailPage />,
  },
];
```

### Step 3: Link from Dashboard

From your dashboard Portfolios list, link to:

```typescript
<Link to={`/portfolios/${portfolio.id}`}>
  {portfolio.name}
</Link>
```

---

## API Contract Specification

The Portfolio Detail system requires **5 backend endpoints**. All endpoints return JSON and support the following tenant/temporal parameters:

- Query Param: `tenant_id` (string, required)
- Query Param: `valuation_date` (ISO-8601 date, optional, defaults to today)

All portfolio data is **tenant-scoped** via server-side RLS.

### Endpoint 1: Portfolio Overview

```
GET /api/portfolios/{portfolio_id}/overview
  ?tenant_id=...&valuation_date=YYYY-MM-DD

Response 200 OK:
{
  "portfolio_id": "PF-001",
  "name": "Global Equities Fund",
  "aum_usd": 125000000,
  "currency": "USD",
  "strategy": "Long-Only Equities",
  "benchmark": "MSCI World",
  "inception_date": "2020-01-15",
  "valuation_date": "2024-01-15",
  "ytd_return": 0.1250,
  "benchmark_return": 0.0890,
  "tracking_error": 0.0360
}

Response 404: Portfolio not found
Response 403: Tenant unauthorized
```

**Type Definition:**

```typescript
interface PortfolioOverview {
  portfolio_id: string;
  name: string;
  aum_usd: number;           // in USD
  currency: string;           // e.g., "USD"
  strategy: string;            // e.g., "Long-Only Equities"
  benchmark: string;           // e.g., "MSCI World"
  inception_date: string;      // ISO-8601
  valuation_date: string;      // ISO-8601
  ytd_return: number;          // decimal, e.g., 0.1250 = 12.50%
  benchmark_return: number;    // decimal
  tracking_error: number;      // decimal
}
```

---

### Endpoint 2: Holdings Summary

```
GET /api/portfolios/{portfolio_id}/holdings
  ?tenant_id=...&valuation_date=YYYY-MM-DD

Response 200 OK:
{
  "portfolio_id": "PF-001",
  "total_holdings": 45,
  "top_holdings": [
    {
      "security_id": "AAPL",
      "name": "Apple Inc.",
      "sector": "Technology",
      "weight": 0.0850,           // 8.5%
      "price": 185.32,
      "shares": 512000,
      "market_value_usd": 94997440,
      "change_pct_1d": 0.0125,    // 1.25%
      "change_pct_ytd": 0.3850    // 38.50%
    },
    // ... 9 more holdings
  ],
  "sector_weights": [
    { "sector": "Technology", "weight": 0.3200 },
    { "sector": "Healthcare", "weight": 0.2100 },
    { "sector": "Financials", "weight": 0.1800 },
    // ... more sectors
  ],
  "country_weights": [
    { "country": "United States", "weight": 0.6500 },
    { "country": "Switzerland", "weight": 0.1200 },
    // ... more countries
  ]
}

Response 404: Portfolio not found
Response 403: Tenant unauthorized
```

**Type Definition:**

```typescript
interface Holding {
  security_id: string;
  name: string;
  sector: string;
  weight: number;                // decimal, 0.085 = 8.5%
  price: number;
  shares: number;
  market_value_usd: number;
  change_pct_1d: number;         // decimal
  change_pct_ytd: number;        // decimal
}

interface HoldingsSummary {
  portfolio_id: string;
  total_holdings: number;
  top_holdings: Holding[];       // Top 10
  sector_weights: Array<{ sector: string; weight: number }>;
  country_weights: Array<{ country: string; weight: number }>;
}
```

---

### Endpoint 3: Risk Snapshot

```
GET /api/portfolios/{portfolio_id}/risk
  ?tenant_id=...&valuation_date=YYYY-MM-DD

Response 200 OK:
{
  "portfolio_id": "PF-001",
  "volatility_pct": 0.1850,      // 18.50% annualized
  "var_95": -4250000,             // USD, 95% confidence
  "var_99": -5800000,             // USD, 99% confidence
  "worst_scenario_pnl": -8500000, // USD
  "beta": 1.1200,
  "alpha": 0.0450,
  "sharpe_ratio": 0.8500,
  "sortino_ratio": 1.2300,
  "factor_exposures": [
    {
      "factor_id": "VALUE",
      "exposure": 0.2500       // 25% overweight
    },
    {
      "factor_id": "SIZE",
      "exposure": -0.1200      // 12% underweight
    },
    // ... more factors
  ]
}

Response 404: Portfolio not found
Response 403: Tenant unauthorized
```

**Type Definition:**

```typescript
interface FactorExposure {
  factor_id: string;             // e.g., "VALUE", "MOMENTUM", "GROWTH"
  exposure: number;              // decimal, positive = overweight
}

interface RiskSnapshot {
  portfolio_id: string;
  volatility_pct: number;        // decimal
  var_95: number;                // USD
  var_99: number;                // USD
  worst_scenario_pnl: number;    // USD
  beta: number;
  alpha: number;
  sharpe_ratio: number;
  sortino_ratio: number;
  factor_exposures: FactorExposure[];
}
```

---

### Endpoint 4: Compliance Snapshot

```
GET /api/portfolios/{portfolio_id}/compliance
  ?tenant_id=...&valuation_date=YYYY-MM-DD

Response 200 OK:
{
  "portfolio_id": "PF-001",
  "total_rules_evaluated": 127,
  "passing_rules": 123,
  "pass_rate": 0.9685,           // 96.85%
  "hard_breaches": [
    {
      "rule_code": "SECTOR_CONCENTRATION",
      "metric_value": 0.3500,    // 35% in Technology
      "threshold_value": 0.3000, // max 30%
      "description": "Technology sector exceeds 30% max allocation"
    }
  ],
  "soft_breaches": [
    {
      "rule_code": "LARGE_CAP_BIAS",
      "metric_value": 0.7200,
      "threshold_value": 0.6500,
      "description": "Large-cap bias slightly elevated"
    }
  ]
}

Response 404: Portfolio not found
Response 403: Tenant unauthorized
```

**Type Definition:**

```typescript
interface RuleBreachDetail {
  rule_code: string;
  metric_value: number;
  threshold_value: number;
  description: string;
}

interface ComplianceSnapshot {
  portfolio_id: string;
  total_rules_evaluated: number;
  passing_rules: number;
  pass_rate: number;             // decimal, e.g., 0.9685 = 96.85%
  hard_breaches: RuleBreachDetail[];
  soft_breaches: RuleBreachDetail[];
}
```

---

### Endpoint 5: Scenario Analysis

```
GET /api/portfolios/{portfolio_id}/scenarios
  ?tenant_id=...&valuation_date=YYYY-MM-DD

Response 200 OK:
{
  "portfolio_id": "PF-001",
  "scenario_date": "2024-01-15",
  "results": [
    {
      "scenario_id": "SC001",
      "name": "Interest Rate Shock (+100bps)",
      "description": "Parallel yield curve shift up 100bp",
      "pnl": -2150000,            // USD, loss
      "pnl_pct": -0.0172          // -1.72%
    },
    {
      "scenario_id": "SC002",
      "name": "Tech Sell-Off (-15%)",
      "description": "Technology sector declines 15%",
      "pnl": -3400000,
      "pnl_pct": -0.0272
    },
    {
      "scenario_id": "SC003",
      "name": "Market Rally (+10%)",
      "description": "Broad market rally, +10%",
      "pnl": 12500000,            // USD, gain
      "pnl_pct": 0.1000
    },
    // ... more scenarios
  ]
}

Response 404: Portfolio not found
Response 403: Tenant unauthorized
```

**Type Definition:**

```typescript
interface ScenarioResult {
  scenario_id: string;
  name: string;
  description: string;
  pnl: number;                   // USD
  pnl_pct: number;               // decimal
}

interface ScenarioResults {
  portfolio_id: string;
  scenario_date: string;         // ISO-8601
  results: ScenarioResult[];
}
```

---

## Frontend Architecture

### File Structure

```
src/pages/portfolio/
├── portfolioApi.ts              # HTTP client layer
├── usePortfolioData.ts          # React Query hooks (in src/hooks/)
├── PortfolioCards.tsx           # Card components
├── PortfolioCharts.tsx          # Table + chart components
├── PortfolioDetailPage.tsx      # Main orchestrator page
└── index.ts                     # Barrel exports
```

### Data Flow

```
PortfolioDetailPage (orchestrator)
  ├─ usePortfolioData()
  │   ├─ usePortfolioOverview() → portfolioApi.fetchPortfolioOverview()
  │   ├─ usePortfolioHoldings() → portfolioApi.fetchPortfolioHoldings()
  │   ├─ usePortfolioRisk() → portfolioApi.fetchPortfolioRisk()
  │   ├─ usePortfolioCompliance() → portfolioApi.fetchPortfolioCompliance()
  │   └─ usePortfolioScenarios() → portfolioApi.fetchPortfolioScenarios()
  │
  ├─ PortfolioCards (data display)
  │   ├─ PortfolioOverviewCard
  │   ├─ RiskSnapshotCard
  │   └─ ComplianceSnapshotCard
  │
  └─ PortfolioCharts (tables + visualizations)
      ├─ HoldingsTable
      ├─ SectorWeights
      └─ ScenarioChart
```

---

## React Query Configuration

All portfolio hooks use standard React Query configuration:

```typescript
const queryConfig = {
  staleTime: 60 * 1000,        // 60 seconds before data marked stale
  cacheTime: 5 * 60 * 1000,    // 5 minutes cache retention
  retry: 2,                     // Retry failed requests twice
  retryDelay: 1000,             // 1 second between retries
};
```

### Query Key Structure

All portfolio queries use consistent namespacing:

```typescript
// Individual queries
['portfolio', 'overview', portfolioId, valuationDate]
['portfolio', 'holdings', portfolioId, valuationDate]
['portfolio', 'risk', portfolioId, valuationDate]
['portfolio', 'compliance', portfolioId, valuationDate]
['portfolio', 'scenarios', portfolioId, valuationDate]

// Combined query
['portfolio', 'all', portfolioId, valuationDate]
```

---

## Hook Usage Reference

### `usePortfolioData(portfolioId, valuationDate)`

Combined hook that fetches all 5 data sources:

```typescript
import { usePortfolioData } from '@/pages/portfolio';

const { 
  overview,      // { data, isLoading, error }
  holdings,      // { data, isLoading, error }
  risk,          // { data, isLoading, error }
  compliance,    // { data, isLoading, error }
  scenarios,     // { data, isLoading, error }
  isLoading,     // true if ANY query is loading
  isError,       // true if ANY query has error
} = usePortfolioData('PF-001', '2024-01-15');

if (isLoading) return <LoadingSkeleton />;
if (isError) return <ErrorAlert />;

return (
  <>
    <PortfolioOverviewCard data={overview.data} />
    <HoldingsTable data={holdings.data} />
  </>
);
```

### Individual Hooks

```typescript
import { 
  usePortfolioOverview,
  usePortfolioHoldings,
  usePortfolioRisk,
  usePortfolioCompliance,
  usePortfolioScenarios,
} from '@/pages/portfolio';

// Each returns { data, isLoading, error, isRefetching }
const overview = usePortfolioOverview(portfolioId, valuationDate);
const holdings = usePortfolioHoldings(portfolioId, valuationDate);
const risk = usePortfolioRisk(portfolioId, valuationDate);
const compliance = usePortfolioCompliance(portfolioId, valuationDate);
const scenarios = usePortfolioScenarios(portfolioId, valuationDate);
```

---

## Styling & Dark Mode

### Tailwind Classes

All components use Tailwind CSS with built-in dark mode support:

```typescript
// Light mode (default)
className="bg-white text-slate-900 border-slate-200"

// Dark mode (automatic)
className="dark:bg-slate-900 dark:text-white dark:border-slate-800"

// Combined safe default
className="bg-white dark:bg-slate-900 text-slate-900 dark:text-white"
```

### Color Scheme

- **Primary**: #137fec (blue-600)
- **Success**: #10b981 (green-500)
- **Error**: #ef4444 (red-500)
- **Warning**: #f59e0b (amber-500)
- **Neutral**: slate-50 to slate-900

### Responsive Grid

```typescript
<ConsoleGrid 
  columns={3}        // 3 columns on desktop
  gap="lg"           // Large gap between items
>
  {/* Auto-adapts to smaller screens */}
</ConsoleGrid>
```

---

## Component Reference

### PortfolioDetailPage

Main orchestrator page combining all portfolio views.

**Props**: None (reads `portfolioId` from route params)

**Features**:
- Tab navigation (Overview, Holdings, Risk & Factors, Compliance, Scenarios)
- Multi-tenant support via DashboardContext
- Breadcrumb navigation
- Export PDF button
- Compliance breach alerts
- Loading/error states

**Usage**:

```typescript
<Route path="/portfolios/:portfolioId" element={<PortfolioDetailPage />} />
```

---

### PortfolioOverviewCard

Displays key portfolio metrics: AUM, currency, strategy, benchmark.

**Props**:

```typescript
interface Props {
  data?: PortfolioOverview;
  isLoading?: boolean;
  error?: Error | null;
}
```

**Usage**:

```typescript
<PortfolioOverviewCard 
  data={portfolio.overview.data}
  isLoading={portfolio.overview.isLoading}
  error={portfolio.overview.error}
/>
```

---

### RiskSnapshotCard

Displays risk metrics: volatility, VaR 95/99, worst scenario.

**Props**:

```typescript
interface Props {
  data?: RiskSnapshot;
  isLoading?: boolean;
  error?: Error | null;
}
```

**Usage**:

```typescript
<RiskSnapshotCard 
  data={portfolio.risk.data}
  isLoading={portfolio.risk.isLoading}
  error={portfolio.risk.error}
/>
```

---

### ComplianceSnapshotCard

Displays compliance status: pass rate, breach counts.

**Props**:

```typescript
interface Props {
  data?: ComplianceSnapshot;
  isLoading?: boolean;
  error?: Error | null;
}
```

**Usage**:

```typescript
<ComplianceSnapshotCard 
  data={portfolio.compliance.data}
  isLoading={portfolio.compliance.isLoading}
  error={portfolio.compliance.error}
/>
```

---

### HoldingsTable

Table of top 10 holdings with sector, weight, performance.

**Props**:

```typescript
interface Props {
  data?: HoldingsSummary;
  isLoading?: boolean;
  error?: Error | null;
}
```

**Usage**:

```typescript
<HoldingsTable 
  data={portfolio.holdings.data}
  isLoading={portfolio.holdings.isLoading}
  error={portfolio.holdings.error}
/>
```

---

### SectorWeights

Horizontal progress bar visualization of sector/country allocation.

**Props**:

```typescript
interface Props {
  data?: Array<{ sector: string; weight: number }>;
  isLoading?: boolean;
}
```

**Usage**:

```typescript
<SectorWeights 
  data={portfolio.holdings.data?.sector_weights}
  isLoading={portfolio.holdings.isLoading}
/>
```

---

### ScenarioChart

Bar chart of scenario PnL distribution (red for losses, green for gains).

**Props**:

```typescript
interface Props {
  data?: ScenarioResult[];
  isLoading?: boolean;
  error?: Error | null;
}
```

**Usage**:

```typescript
<ScenarioChart 
  data={portfolio.scenarios.data?.results}
  isLoading={portfolio.scenarios.isLoading}
  error={portfolio.scenarios.error}
/>
```

---

## Integration Examples

### Example 1: Loading Data and Rendering

```typescript
import { PortfolioDetailPage } from '@/pages/portfolio';
import { usePortfolioData } from '@/pages/portfolio';

export function PortfolioPage({ portfolioId }: { portfolioId: string }) {
  const portfolio = usePortfolioData(portfolioId, '2024-01-15');

  if (portfolio.isLoading) {
    return <LoadingSkeleton />;
  }

  if (portfolio.isError) {
    return <ErrorAlert message={portfolio.error?.message} />;
  }

  return (
    <PortfolioDetailPage />
  );
}
```

### Example 2: Manual Refetching

```typescript
import { useQueryClient } from '@tanstack/react-query';
import { usePortfolioData } from '@/pages/portfolio';

export function PortfolioActions() {
  const queryClient = useQueryClient();
  const { portfolioId } = useParams();

  const handleRefresh = () => {
    queryClient.invalidateQueries({
      queryKey: ['portfolio', 'all', portfolioId],
    });
  };

  return (
    <button onClick={handleRefresh}>
      🔄 Refresh Data
    </button>
  );
}
```

### Example 3: Linking from Dashboard

```typescript
import { Link } from 'react-router-dom';

export function PortfolioList({ portfolios }) {
  return (
    <ul>
      {portfolios.map((pf) => (
        <li key={pf.id}>
          <Link to={`/portfolios/${pf.id}`}>
            {pf.name} (${pf.aum_usd / 1000000}M)
          </Link>
        </li>
      ))}
    </ul>
  );
}
```

---

## Error Handling

All components gracefully degrade with appropriate user feedback:

### API Layer Errors

```typescript
// portfolioApi.ts - Throw descriptive errors
if (response.status === 403) {
  throw new Error('Unauthorized: This portfolio may not exist or you lack access');
}
if (response.status === 404) {
  throw new Error('Portfolio not found');
}
if (!response.ok) {
  throw new Error(`API Error: ${response.status} ${response.statusText}`);
}
```

### Component-Level Error Boundaries

```typescript
{portfolio.error && (
  <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-4">
    <p className="text-red-700 dark:text-red-300 text-sm">
      {portfolio.error.message}
    </p>
  </div>
)}
```

---

## Performance Optimizations

### 1. Query Caching

React Query automatically caches all portfolio data for 5 minutes, dramatically reducing API calls.

### 2. Stale-While-Revalidate

Data marked stale after 60 seconds but continues to display while fresh data fetches in background.

### 3. Request Deduplication

Multiple simultaneous requests for same data automatically deduplicated on query key.

### 4. Lazy Loading

Tab content only renders when active, reducing initial load time.

### 5. Update Suppression

`isRefetching` separate from `isLoading` prevents UI flicker during background updates.

---

## Testing Checklist

- [ ] Portfolio loads with valid portfolioId
- [ ] Tab navigation switches between 5 views
- [ ] All 5 API endpoints called on initial load
- [ ] Data displays with correct formatting (currency, percentages)
- [ ] Dark mode toggles all component colors
- [ ] Compliance breaches display with severity indicators
- [ ] Scenarios chart shows correct negative/positive coloring
- [ ] Responsive grid adapts to mobile screen sizes
- [ ] Error states show user-friendly messages
- [ ] Loading skeletons appear while fetching
- [ ] Multi-tenant isolation respected (tenant_id param included)

---

## Production Readiness Checklist

- [ ] All 5 API endpoints implemented in Go backend
- [ ] Server-side RLS enforces tenant isolation
- [ ] Error responses include appropriate HTTP status codes (403, 404, 500)
- [ ] API response times under 500ms per endpoint
- [ ] All TypeScript types validated against backend responses
- [ ] Dark mode CSS classes verified in production build
- [ ] Responsive breakpoints tested on mobile/tablet/desktop
- [ ] React Query cache configuration matches deployment latency profile
- [ ] Environment variables configured for staging and production
- [ ] Error logging includes stack traces for backend debugging
- [ ] Documentation updated with go backend implementation details
- [ ] Load testing validates concurrent portfolio lookups

---

## Debugging

### Enable React Query DevTools

```typescript
import { ReactQueryDevtools } from '@tanstack/react-query-devtools';

export function App() {
  return (
    <>
      <YourApp />
      <ReactQueryDevtools initialIsOpen={true} />
    </>
  );
}
```

### Check Query State

```typescript
const queryClient = useQueryClient();
const queryState = queryClient.getQueryState(['portfolio', 'overview', 'PF-001']);
console.log('Query state:', queryState);
// { data, dataUpdatedAt, error, errorUpdatedAt, isInvalidated, status, fetchStatus }
```

### Monitor API Calls

Open browser DevTools Network tab and filter by `/api/portfolios/` to see:
- Request URL + query params
- Response status + size
- Timing (download time, processing time)
- Cache hit/miss indicators

---

## Support & Questions

For issues with:
- **Component rendering**: Check TypeScript types match API responses exactly
- **API failures**: Verify tenant_id included in query params; check backend RLS
- **Styling bugs**: Clear Tailwind cache with `npm run build:css`
- **Performance**: Profile with React DevTools Profiler; check React Query DevTools cache

