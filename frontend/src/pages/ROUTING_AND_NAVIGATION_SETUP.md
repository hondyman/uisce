# Risk & Compliance Console - Routing & Navigation Setup

**Status**: ✅ Routes configured and ready for use  
**Date**: February 22, 2026  

---

## Routes Added to AppRoutes.tsx

### Dashboard Route
```typescript
<Route path="/dashboard" element={<ProtectedRoute><DashboardHome /></ProtectedRoute>} />
```

**URL**: `http://localhost:5173/dashboard`  
**Purpose**: System-level Risk & Compliance Dashboard (tenant-aware operational cockpit)  
**Protection**: Requires authentication via `ProtectedRoute`

### Portfolio Detail Route
```typescript
<Route path="/portfolios/:portfolioId" element={<ProtectedRoute><PortfolioDetailPage /></ProtectedRoute>} />
```

**URL Pattern**: `http://localhost:5173/portfolios/PF-001`  
**Purpose**: Portfolio deep-dive analysis (5 integrated tabs)  
**Parameters**:
- `portfolioId` (required) - Portfolio identifier from URL param
- `valuation_date` (optional) - From DashboardContext

### Redirect Alias (Optional)
```typescript
<Route path="/risk-compliance" element={<Navigate to="/dashboard" replace />} />
```

Provides a friendly alias `/risk-compliance` → `/dashboard`

---

## Navigation Implementation

### 1. Add to Main Navigation Menu

In your main app nav/sidebar component:

```typescript
import { Link } from 'react-router-dom';

export function MainNav() {
  return (
    <nav className="space-y-2">
      {/* Existing nav items */}
      
      {/* New Risk & Compliance section */}
      <div className="mt-6 pt-6 border-t border-slate-200 dark:border-slate-800">
        <p className="text-xs font-bold text-slate-600 dark:text-slate-400 uppercase tracking-wider mb-3">
          Risk & Compliance
        </p>
        <Link 
          to="/dashboard"
          className="flex items-center gap-3 px-3 py-2 rounded-lg hover:bg-slate-100 dark:hover:bg-slate-800 text-slate-700 dark:text-slate-300 transition-colors"
        >
          <span className="text-lg">📊</span>
          <span className="font-medium">Dashboard</span>
        </Link>
        <Link 
          to="/analytics/portfolio-master"
          className="flex items-center gap-3 px-3 py-2 rounded-lg hover:bg-slate-100 dark:hover:bg-slate-800 text-slate-700 dark:text-slate-300 transition-colors"
        >
          <span className="text-lg">💼</span>
          <span className="font-medium">Portfolio Master</span>
        </Link>
      </div>
    </nav>
  );
}
```

### 2. Add to Dashboard Home Page

Create a link to view all portfolios or popular portfolios:

```typescript
export function DashboardHome() {
  // ... existing code ...
  
  return (
    <>
      {/* existing dashboard content */}
      
      {/* Quick Links to Portfolios */}
      <div className="mt-8 grid grid-cols-3 gap-4">
        <Link 
          to="/portfolios/PF-001"
          className="p-4 bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-xl hover:shadow-lg transition-shadow"
        >
          <p className="text-sm font-medium text-slate-600 dark:text-slate-400">Featured Portfolio</p>
          <p className="text-lg font-bold text-slate-900 dark:text-white">Global Equities Fund</p>
          <p className="text-xs text-slate-500 dark:text-slate-400 mt-2">View Details →</p>
        </Link>
        
        {/* More portfolio cards... */}
      </div>
    </>
  );
}
```

### 3. Add Quick Access Button

In your app header/top bar:

```typescript
import { useNavigate } from 'react-router-dom';

export function AppTopBar() {
  const navigate = useNavigate();
  
  return (
    <div className="flex items-center justify-between px-6 py-4 border-b border-slate-200 dark:border-slate-800">
      {/* existing header content */}
      
      <div className="flex items-center gap-3">
        <button
          onClick={() => navigate('/dashboard')}
          className="px-4 py-2 bg-blue-600 text-white text-sm font-bold rounded-lg hover:bg-blue-700 transition-colors flex items-center gap-2"
        >
          📊 Dashboard
        </button>
      </div>
    </div>
  );
}
```

---

## Context Integration

### DashboardContext Setup

The portfolio system relies on `DashboardContext` for multi-tenant support and valuation date selection.

**Required Setup** (if not already done):

```typescript
// App.tsx or main entry point
import { DashboardProvider } from '@/pages/dashboard/DashboardContext';
import { QueryClientProvider } from '@tanstack/react-query';

export function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <DashboardProvider>
        <AppRoutes />
      </DashboardProvider>
    </QueryClientProvider>
  );
}
```

**Context Usage in Components**:

```typescript
import { useDashboardContext } from '@/pages/dashboard/DashboardContext';

export function MyComponent() {
  const { tenant, valuationDate } = useDashboardContext();
  
  // Use tenant.id in API calls
  // Use valuationDate for temporal queries
}
```

---

## Navigation Flow Diagram

```
User Authentication
      ↓
App Home/Dashboard
      ├─ Click "Risk & Compliance Dashboard"
      │  ↓
      │  /dashboard
      │  ├─ Select Tenant
      │  ├─ View KPIs, Alerts, Health Status
      │  └─ Click Portfolio Name
      │     ↓
      │     /portfolios/PF-001
      │     ├─ Tab: Overview (cards + charts)
      │     ├─ Tab: Holdings (top 10 positions)
      │     ├─ Tab: Risk & Factors (exposures)
      │     ├─ Tab: Compliance (breaches)
      │     └─ Tab: Scenarios (PnL)
      │
      └─ Direct URL: /portfolios/PF-001?valuation_date=2024-01-15
         └─ (Valuation date from query or context)
```

---

## URL Examples

### Dashboard (System-Level)
- **Base**: `http://localhost:5173/dashboard`
- **With tenant**: `http://localhost:5173/dashboard?tenant_id=550e8400-e29b-41d4-a716-446655440000`
- **Alias**: `http://localhost:5173/risk-compliance` (redirects to `/dashboard`)

### Portfolio Detail (Portfolio-Level)
- **Portfolio 1**: `http://localhost:5173/portfolios/PF-001`
- **Portfolio 2**: `http://localhost:5173/portfolios/PF-002`
- **With date**: `http://localhost:5173/portfolios/PF-001?valuation_date=2024-01-15`
- **With tenant**: `http://localhost:5173/portfolios/PF-001?tenant_id=tenant-uuid`

---

## Browser History & Back Navigation

### Recommended Navigation Pattern

```typescript
import { useNavigate } from 'react-router-dom';

export function PortfolioDetailPage() {
  const navigate = useNavigate();
  
  const handleBackToDashboard = () => {
    navigate('/dashboard');
  };
  
  const handleBackButton = () => {
    navigate(-1); // Browser native back
  };
  
  return (
    <ConsoleBreadcrumbs
      items={[
        { label: 'Dashboard', href: '/dashboard' },
        { label: 'Portfolio', active: true },
      ]}
    />
  );
}
```

---

## Deeplink Support

### Share Portfolio Link

Users can share portfolio URLs directly:

```typescript
// Copy to clipboard
const url = `${window.location.origin}/portfolios/PF-001?valuation_date=2024-01-15`;
navigator.clipboard.writeText(url);
```

### Email Link Example
```
Check out this portfolio analysis:
http://semlayer.company.com/portfolios/PF-001?valuation_date=2024-01-15
```

---

## Testing the Routes

### 1. Dashboard Access

```bash
# Navigate to dashboard
curl http://localhost:5173/dashboard

# Or via browser
open http://localhost:5173/dashboard
```

Expected: Dashboard Home page with tenant selector and KPI cards

### 2. Portfolio Access

```bash
# Navigate to portfolio detail
curl http://localhost:5173/portfolios/PF-001

# Or via browser
open http://localhost:5173/portfolios/PF-001
```

Expected: Portfolio Detail page with Overview tab active

### 3. Route Protection

```bash
# Try accessing without authentication
curl -L http://localhost:5173/dashboard

# Expected: Redirect to /login
```

---

## Troubleshooting Navigation

### Issue: Routes not found (404)

**Cause**: Missing imports in AppRoutes.tsx  
**Solution**: Verify imports at top of file:
```typescript
import { DashboardHome } from './pages/dashboard/DashboardHome';
import { PortfolioDetailPage } from './pages/portfolio/PortfolioDetailPage';
```

### Issue: Portfolio ID in URL but page not loading

**Cause**: API not returning data for portfolio ID  
**Solution**:
1. Check portfolio ID exists in database
2. Verify tenant_id matches authenticated user
3. Check browser console for API errors
4. Verify API base URL configured: `REACT_APP_API_BASE_URL`

### Issue: Tenant selector not appearing

**Cause**: DashboardProvider not wrapping app  
**Solution**: Wrap app in DashboardProvider at top level:
```typescript
<DashboardProvider>
  <YourApp />
</DashboardProvider>
```

### Issue: Dark mode not working in new routes

**Cause**: ConsoleTopNav not setting dark mode class  
**Solution**: Ensure ConsoleTopNav includes logic to set `dark` class:
```typescript
useEffect(() => {
  if (isDarkMode) {
    document.documentElement.classList.add('dark');
  } else {
    document.documentElement.classList.remove('dark');
  }
}, [isDarkMode]);
```

---

## Performance Considerations

### Route Prefetching

To speed up navigation, prefetch routes on hover:

```typescript
import { useNavigate } from 'react-router-dom';

export function PortfolioLink({ portfolioId }) {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  
  const handleMouseEnter = () => {
    // Prefetch portfolio data
    queryClient.prefetchQuery({
      queryKey: ['portfolio', 'all', portfolioId],
      queryFn: () => usePortfolioData(portfolioId, '2024-01-15'),
    });
  };
  
  return (
    <Link 
      to={`/portfolios/${portfolioId}`}
      onMouseEnter={handleMouseEnter}
    >
      {portfolioId}
    </Link>
  );
}
```

### Route-Based Code Splitting

Consider lazy-loading components for better initial load time:

```typescript
import { lazy, Suspense } from 'react';

const DashboardHome = lazy(() => import('./pages/dashboard/DashboardHome'));
const PortfolioDetailPage = lazy(() => import('./pages/portfolio/PortfolioDetailPage'));

export function AppRoutes() {
  return (
    <Routes>
      <Route 
        path="/dashboard" 
        element={
          <Suspense fallback={<LoadingSpinner />}>
            <ProtectedRoute><DashboardHome /></ProtectedRoute>
          </Suspense>
        } 
      />
      <Route 
        path="/portfolios/:portfolioId" 
        element={
          <Suspense fallback={<LoadingSpinner />}>
            <ProtectedRoute><PortfolioDetailPage /></ProtectedRoute>
          </Suspense>
        } 
      />
    </Routes>
  );
}
```

---

## Production Deployment Checklist

- [ ] Routes tested on staging environment
- [ ] Authentication working for both routes (ProtectedRoute)
- [ ] API endpoints responding with valid data
- [ ] Environment variables configured (`REACT_APP_API_BASE_URL`)
- [ ] Dark mode CSS compiled and included
- [ ] Mobile responsive layouts verified
- [ ] Back/forward browser navigation working
- [ ] Deep links shareable and working
- [ ] Error states display appropriate messages
- [ ] Loading states show during data fetch
- [ ] Multi-tenant isolation respected
- [ ] Analytics tracking added for route changes
- [ ] Documentation updated for end users

---

## Summary

✅ **Routes configured**: `/dashboard` and `/portfolios/:portfolioId`  
✅ **Protection**: Both wrapped in ProtectedRoute for authentication  
✅ **Context**: DashboardProvider integration for tenant/date management  
✅ **Navigation**: Ready for menu integration and deep linking  

**Next Steps**:
1. Test routes in development environment
2. Add navigation menu items
3. Deploy to staging
4. QA testing of full navigation flow
5. Production deployment

