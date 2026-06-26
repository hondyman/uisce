# Portfolio Management Frontend Implementation Guide

## Overview

This guide covers the three frontend components for the Portfolio Management System that complete the full-stack implementation. All components integrate with the backend API and follow the existing Semlayer design patterns.

## Components Created

### 1. **Portfolio Dashboard Page** (`PortfolioDashboardPage.tsx`)
**Location**: `frontend/src/pages/PortfolioDashboardPage.tsx`

**Purpose**: Main portfolio management interface for viewing and managing multiple portfolios and their holdings.

**Key Features**:
- Portfolio list with quick stats (total value, day change, holdings count)
- Portfolio creation with modal form
- Holdings detail view with filtering and sorting
- Performance metrics (day change, monthly return, yearly return)
- Portfolio selection and switching
- Holdings allocation tracking
- Top holdings visualization
- Export functionality
- Real-time refresh capability

**UI Components**:
- Portfolio cards with selection highlighting
- Holdings table with comprehensive metrics
- Portfolio summary sidepanel
- Top holdings allocation breakdown
- Create portfolio modal

**API Integration**:
- `GET /api/portfolios` - Fetch user portfolios
- `POST /api/portfolios` - Create new portfolio
- `DELETE /api/portfolios/{id}` - Delete portfolio
- `GET /api/holdings` - Fetch portfolio holdings

**State Management**:
- `portfolios`: Array of portfolio objects
- `selectedPortfolio`: Currently selected portfolio
- `holdings`: Holdings for selected portfolio
- `formData`: Create portfolio form state

### 2. **Recommendation Review Page** (`RecommendationReviewPage.tsx`)
**Location**: `frontend/src/pages/RecommendationReviewPage.tsx`

**Purpose**: Interface for reviewing, approving, and implementing investment recommendations.

**Key Features**:
- Recommendation list with status indicators
- Recommendation type and priority badges
- Detailed recommendation view with full analysis
- Target allocation comparison (current vs. target)
- Recommended actions visualization (BUY/SELL/HOLD)
- Status workflow (draft → proposed → accepted → implemented)
- Notes and approval commentary
- Filtering by status and type
- Recommendation creation with guided form

**UI Components**:
- Recommendation cards with status badges
- Detailed recommendation panel
- Action items with color-coded types
- Status update section with action buttons
- Create recommendation modal with multi-field form

**API Integration**:
- `GET /api/recommendations` - Fetch recommendations
- `POST /api/recommendations` - Create recommendation
- `GET /api/recommendation-status` - Fetch recommendation details
- `PATCH /api/recommendation-status` - Update recommendation status
- `DELETE /api/recommendations/{id}` - Delete recommendation

**State Management**:
- `recommendations`: Array of recommendations
- `selectedRec`: Currently selected recommendation
- `formData`: Create recommendation form state
- `statusNotes`: Notes for status updates
- `filterStatus` / `filterType`: Filter controls

### 3. **Risk Analytics Dashboard Page** (`RiskAnalyticsDashboardPage.tsx`)
**Location**: `frontend/src/pages/RiskAnalyticsDashboardPage.tsx`

**Purpose**: Comprehensive portfolio risk analysis and metrics visualization with stress testing capability.

**Key Features**:
- Risk metric dashboard with key indicators
- Overall risk assessment (Low/Medium/High)
- Beta and alpha analysis
- Value at Risk (VaR) and Conditional VaR calculation display
- Concentration risk visualization with progress bars
- Risk factor analysis with exposure/sensitivity breakdown
- Advanced metrics (Sortino ratio, diversification ratio)
- Risk recommendations engine
- Portfolio selection and switching
- Risk tolerance selector
- Export risk reports
- Trend tracking for metrics

**UI Components**:
- Metric cards with trend indicators
- Risk level badge with contextual coloring
- VaR/CVaR comparison boxes
- Concentration risk progress bars
- Risk factor breakdown table
- Risk recommendation list
- Advanced metrics panel (toggleable)

**API Integration**:
- `GET /api/portfolio-risk-metrics` - Fetch risk metrics
- `GET /api/risk-factors` - Fetch risk factor analysis

**State Management**:
- `portfolios`: Risk metrics for all portfolios
- `selectedPortfolio`: Currently selected portfolio's risk metrics
- `riskFactors`: Factor analysis for selected portfolio
- `riskTolerance`: User's risk tolerance setting
- `showAdvanced`: Advanced metrics visibility toggle

## Integration with Existing Architecture

### Tenant Context
All components use `useTenant()` hook to access:
- `tenant`: Tenant object with ID
- `datasource`: Datasource object with ID

### Headers
Every API call includes required headers:
```typescript
{
  'X-User-ID': tenant?.id,
  'X-Tenant-ID': tenant?.id,
  'X-Tenant-Datasource-ID': datasource?.id,
  'Content-Type': 'application/json' // for POST/PATCH
}
```

### Logging
All components use `devLog()` utility for development debugging:
```typescript
devLog('Component Name initialized', { tenantId, datasourceId });
```

### Toast Notifications
Consistent toast notifications for success/error feedback:
```typescript
const showToast = (type: 'success' | 'error', message: string) => {
  setToast({ type, message });
  setTimeout(() => setToast(null), 3000);
};
```

## Styling and Theming

### Design System
- **Colors**: Tailwind dark mode compatible
- **Icons**: Lucide React icons (consistent with existing pages)
- **Responsive**: Mobile-first responsive grid layouts
- **Accessibility**: ARIA labels, semantic HTML, keyboard navigation

### Dark Mode Support
All components include full dark mode support via:
- `dark:` Tailwind prefix for all color variants
- Automatic color scheme detection

### Components Reusability
Common patterns for reuse:
- Status badge rendering functions
- Modal forms
- Data filtering and sorting
- Metric cards with trend indicators

## File Integration Checklist

- [ ] Create `/frontend/src/pages/PortfolioDashboardPage.tsx`
- [ ] Create `/frontend/src/pages/RecommendationReviewPage.tsx`
- [ ] Create `/frontend/src/pages/RiskAnalyticsDashboardPage.tsx`
- [ ] Import components in routing file (MainNavigation.tsx or equivalent)
- [ ] Add route entries to router configuration
- [ ] Add navigation menu items (if applicable)
- [ ] Verify TenantContext availability
- [ ] Confirm devLogger import works
- [ ] Test with mock API responses
- [ ] Validate accessibility compliance

## Backend API Requirements

Ensure these endpoints are implemented and running:

### Portfolio Endpoints
- `GET /api/portfolios` - Return array of Portfolio objects
- `POST /api/portfolios` - Create portfolio, return created object
- `GET /api/holdings?portfolio_id={id}` - Return array of Holding objects
- `DELETE /api/portfolios/{id}` - Delete portfolio

### Recommendation Endpoints
- `GET /api/recommendations` - Return array of Recommendation objects
- `POST /api/recommendations` - Create recommendation
- `GET /api/recommendation-status?id={id}` - Get recommendation details
- `PATCH /api/recommendation-status?id={id}` - Update status
- `DELETE /api/recommendations/{id}` - Delete recommendation

### Risk Analytics Endpoints
- `GET /api/portfolio-risk-metrics` - Return RiskMetrics array
- `GET /api/risk-factors?portfolio_id={id}` - Return RiskFactor array
- `POST /api/backtest/run` - Execute backtest (referenced but not required)
- `GET /api/backtest/results` - Fetch backtest results (optional)

## Data Type Definitions

### Key Interfaces Used

```typescript
// Portfolio
interface Portfolio {
  id: string;
  name: string;
  currency: string;
  totalValue: number;
  holdingsCount: number;
  allocation: Array<{ symbol: string; percentage: number; value: number }>;
  metrics: {
    dayChange: number;
    dayChangePercent: number;
    monthReturn: number;
    yearReturn: number;
  };
  lastUpdated: string;
}

// Holding
interface Holding {
  id: string;
  symbol: string;
  name: string;
  quantity: number;
  averageCost: number;
  currentPrice: number;
  currentValue: number;
  gainLoss: number;
  gainLossPercent: number;
  allocation: number;
  assetClass: string;
  sector: string;
  beta?: number;
  volatility?: number;
}

// Recommendation
interface Recommendation {
  id: string;
  portfolioId: string;
  portfolioName: string;
  createdBy: string;
  title: string;
  description: string;
  type: 'rebalance' | 'tactical' | 'strategic';
  status: 'draft' | 'proposed' | 'accepted' | 'rejected' | 'implemented';
  targetAllocations: Array<{
    symbol: string;
    targetPercentage: number;
    currentPercentage: number;
  }>;
  recommendedActions: Array<{
    type: 'BUY' | 'SELL' | 'HOLD';
    symbol: string;
    amount: number;
    rationale: string;
  }>;
  rationale: string;
  expectedReturn: number;
  timeHorizon: string;
  riskScore: number;
  priority: 'low' | 'medium' | 'high';
  createdAt: string;
  metadata?: Record<string, any>;
}

// RiskMetrics
interface RiskMetrics {
  portfolioId: string;
  portfolioName: string;
  expectedReturn: number;
  volatility: number;
  sharpeRatio: number;
  sortinoRatio: number;
  beta: number;
  alpha: number;
  maxDrawdown: number;
  valueAtRisk: number;
  conditionalVaR: number;
  diversificationRatio: number;
  concentration: {
    top1: number;
    top5: number;
    top10: number;
  };
  correlationMatrix: Record<string, number>;
  asOfDate: string;
  trend?: {
    returnChange: number;
    volatilityChange: number;
    sharpeChange: number;
  };
}
```

## Navigation Integration

Add to your routing configuration:

```typescript
import PortfolioDashboardPage from './pages/PortfolioDashboardPage';
import RecommendationReviewPage from './pages/RecommendationReviewPage';
import RiskAnalyticsDashboardPage from './pages/RiskAnalyticsDashboardPage';

const routes = [
  {
    path: '/portfolio/dashboard',
    component: PortfolioDashboardPage,
    label: 'Portfolio Dashboard',
    icon: 'PieChart'
  },
  {
    path: '/portfolio/recommendations',
    component: RecommendationReviewPage,
    label: 'Recommendations',
    icon: 'TrendingUp'
  },
  {
    path: '/portfolio/risk-analytics',
    component: RiskAnalyticsDashboardPage,
    label: 'Risk Analytics',
    icon: 'AlertTriangle'
  }
];
```

## Testing Checklist

### PortfolioDashboardPage
- [ ] Portfolios load from API
- [ ] Can create new portfolio
- [ ] Can select portfolio from list
- [ ] Holdings display correctly
- [ ] Filtering by asset class works
- [ ] Sorting by value/allocation/gain works
- [ ] Delete portfolio confirms before deletion
- [ ] Metrics calculations display correctly
- [ ] Dark mode renders properly
- [ ] Responsive on mobile/tablet

### RecommendationReviewPage
- [ ] Recommendations load from API
- [ ] Can create new recommendation
- [ ] Can select recommendation
- [ ] Status workflow buttons work
- [ ] Filtering by status/type works
- [ ] Notes input accepts text
- [ ] Delete confirms before deletion
- [ ] Target allocations display
- [ ] Actions show correct type colors
- [ ] Priority colors display correctly

### RiskAnalyticsDashboardPage
- [ ] Risk metrics load from API
- [ ] Portfolio selector works
- [ ] Risk level assessment is accurate
- [ ] All metric calculations display
- [ ] Concentration bars render correctly
- [ ] Risk factors display with sensitivity
- [ ] Advanced metrics toggle works
- [ ] Risk recommendations populate
- [ ] Trend indicators show correctly
- [ ] VaR/CVaR explanations are clear

## Performance Optimization

- Use React.memo for metrics cards
- Implement pagination for large holding lists
- Debounce filter/sort changes
- Cache portfolio data when possible
- Lazy load risk factor details

## Future Enhancements

1. **Chart Integration**: Add ECharts or Chart.js for visualization
   - Portfolio allocation pie charts
   - Risk metric trend charts
   - Volatility heatmaps

2. **Export Features**: PDF/CSV export of:
   - Portfolio holdings
   - Risk reports
   - Recommendation comparisons

3. **Real-time Updates**: WebSocket integration for:
   - Price updates
   - Risk metric recalculations
   - Status change notifications

4. **Advanced Filtering**: Add date range pickers, multi-select filters

5. **Comparison Views**: Side-by-side portfolio comparison

## Support and Troubleshooting

### API Connection Issues
- Verify backend is running on correct port
- Check X-Tenant-ID and X-Tenant-Datasource-ID headers
- Ensure tenant/datasource context is populated

### Data Not Loading
- Check browser console for error messages
- Verify API responses match expected interfaces
- Check TenantContext initialization

### Styling Issues
- Verify Tailwind CSS is configured
- Check dark mode class is applied to root element
- Ensure Lucide icons are installed

## Version Information
- Frontend Framework: React 18+
- UI Library: Tailwind CSS 3+
- Icons: Lucide React 0.263+
- Build Tool: Vite or Create React App

---

**Status**: ✅ Ready to Deploy
**Last Updated**: October 30, 2025
**Compatibility**: Semlayer v1.0.0+
