# Risk & Compliance Console - React Router Setup Guide

## Integration with Existing App

Your `App.tsx` already has admin dashboards. The Risk & Compliance Console adds a new `/console/*` route tree.

---

## Option 1: Nested Route Approach (Recommended)

Add to your existing router in `App.tsx`:

```typescript
import { consoleRoutes } from './router/consoleRoutes';
import { ConsoleLayout } from './layout';

// In your Routes:
<Routes>
  {/* Existing admin routes */}
  <Route path="/dashboard" element={<SemanticRuleBuilderDashboard />} />
  <Route path="/impact-analysis" element={<ImpactAnalysisDashboard />} />
  {/* ... other routes ... */}

  {/* NEW: Risk & Compliance Console */}
  <Route path="/console/*" element={
    <ConsoleLayout>
      {consoleRoutes}
    </ConsoleLayout>
  } />
</Routes>
```

This keeps:
- `/dashboard` → Admin dashboards (existing)
- `/console/dashboard` → Risk & Compliance Console (new)
- `/console/etl/runs` → ETL runs
- `/console/risk/scenarios/:id/lineage` → Scenario lineage

---

## Option 2: Separate App Component

Create a separate entry point for the console:

```typescript
// AppConsole.tsx
import { BrowserRouter } from 'react-router-dom';
import { QueryClientProvider } from '@tanstack/react-query';
import { ThemeProvider } from '@mui/material/styles';
import { createQueryClient } from './config/queryClient';
import { lightTheme } from './themes';
import { consoleRoutes } from './router/consoleRoutes';
import { ConsoleLayout } from './layout';

export function AppConsole() {
  const queryClient = createQueryClient();
  
  return (
    <QueryClientProvider client={queryClient}>
      <ThemeProvider theme={lightTheme}>
        <BrowserRouter>
          <ConsoleLayout>
            {consoleRoutes}
          </ConsoleLayout>
        </BrowserRouter>
      </ThemeProvider>
    </QueryClientProvider>
  );
}
```

Then in `main.tsx`:

```typescript
import ReactDOM from 'react-dom/client';
import { AppConsole } from './AppConsole';

ReactDOM.createRoot(document.getElementById('root')!).render(
  <AppConsole />
);
```

---

## Key Configuration Files

### 1. Query Client Setup
**File**: `frontend/src/config/queryClient.ts`

```typescript
import { QueryClient } from '@tanstack/react-query';

export const createQueryClient = () => {
  return new QueryClient({
    defaultOptions: {
      queries: {
        staleTime: 1000 * 60 * 5, // 5 minutes
        gcTime: 1000 * 60 * 10,   // 10 minutes
        retry: (failureCount, error) => {
          if (error?.status >= 400 && error?.status < 500) return false;
          return failureCount < 2;
        },
        refetchOnWindowFocus: false,
      },
      mutations: {
        retry: (failureCount, error) => {
          if (error?.status >= 400 && error?.status < 500) return false;
          return failureCount < 1;
        },
      },
    },
  });
};

export const queryKeys = {
  dashboard: {
    compliance: (tenantId, valuationDate) => ['dashboard', 'compliance', tenantId, valuationDate],
    risk: (tenantId, valuationDate) => ['dashboard', 'risk', tenantId, valuationDate],
    sparklines: (tenantId) => ['dashboard', 'sparklines', tenantId],
    etlHealth: (tenantId) => ['dashboard', 'etl-health', tenantId],
    alerts: (tenantId, valuationDate) => ['dashboard', 'alerts', tenantId, valuationDate],
  },
  etl: {
    list: (filters) => ['etl-runs', 'list', filters],
    detail: (id) => ['etl-runs', 'detail', id],
  },
  wasm: {
    list: (moduleName) => ['wasm-versions', moduleName],
  },
  ruleLineage: {
    detail: (ruleId, filters) => ['rule-lineage', ruleId, filters],
  },
  scenarioLineage: {
    detail: (scenarioId, filters) => ['scenario-lineage', scenarioId, filters],
  },
};
```

### 2. Router Setup
**File**: `frontend/src/router/consoleRoutes.tsx`

```typescript
import { Routes, Route } from 'react-router-dom';
import {
  DashboardHome,
  ETLRunsPage,
  WASMVersionsPage,
  RuleLineagePage,
  ScenarioLineagePage,
} from '../pages/console';

export const consoleRoutes = (
  <Routes>
    <Route path="/console/dashboard" element={<DashboardHome />} />
    <Route path="/console/etl/runs" element={<ETLRunsPage />} />
    <Route path="/console/etl/runs/:runId" element={<ETLRunsPage />} />
    <Route path="/console/etl/wasm" element={<WASMVersionsPage />} />
    <Route path="/console/compliance/rules/:ruleId/lineage" element={<RuleLineagePage />} />
    <Route path="/console/risk/scenarios/:scenarioId/lineage" element={<ScenarioLineagePage />} />
    <Route path="/console" element={<DashboardHome />} />
  </Routes>
);
```

### 3. Theme Setup (Optional)
**File**: `frontend/src/themes/index.ts`

```typescript
import { createTheme } from '@mui/material/styles';

export const lightTheme = createTheme({
  palette: {
    mode: 'light',
    primary: { main: '#1976d2' },
    secondary: { main: '#dc004e' },
    background: { default: '#fafafa', paper: '#ffffff' },
  },
});

export const darkTheme = createTheme({
  palette: {
    mode: 'dark',
    primary: { main: '#90caf9' },
    secondary: { main: '#f48fb1' },
    background: { default: '#121212', paper: '#1e1e1e' },
  },
});
```

---

## Integration into Existing App.tsx

The simplest approach: add the console as a nested route.

```typescript
// In your App.tsx

import { BrowserRouter, Routes, Route } from 'react-router-dom';
import { consoleRoutes } from './router/consoleRoutes';
import { ConsoleLayout } from './layout';

// ... your existing imports ...

export function App() {
  const [themeMode, setThemeMode] = useState('light');
  
  return (
    <BrowserRouter>
      <Routes>
        {/* Existing admin routes */}
        <Route path="/dashboard" element={<SemanticRuleBuilderDashboard />} />
        <Route path="/impact-analysis" element={<ImpactAnalysisDashboard />} />
        {/* ... rest of your routes ... */}

        {/* NEW: Risk & Compliance Console */}
        <Route path="/console/*" element={
          <ConsoleLayout onThemeChange={setThemeMode}>
            {consoleRoutes}
          </ConsoleLayout>
        } />
      </Routes>
    </BrowserRouter>
  );
}
```

This gives you:

**Before (Admin Dashboards)**:
```
/dashboard
/impact-analysis
/rule-comparison
/edm-exports
```

**After (Plus Console)**:
```
/dashboard                              (existing)
/impact-analysis                        (existing)
/console/dashboard                      (NEW)
/console/etl/runs                       (NEW)
/console/etl/runs/{id}                  (NEW)
/console/etl/wasm                       (NEW)
/console/compliance/rules/{id}/lineage  (NEW)
/console/risk/scenarios/{id}/lineage    (NEW)
```

---

## Query Client Provider Placement

If you have a global QueryClientProvider at the app root:

```typescript
// main.tsx or index.tsx
import { QueryClientProvider } from '@tanstack/react-query';
import { createQueryClient } from './config/queryClient';
import App from './App';

const queryClient = createQueryClient();

ReactDOM.createRoot(document.getElementById('root')!).render(
  <QueryClientProvider client={queryClient}>
    <CssBaseline />
    <ThemeProvider theme={theme}>
      <App />
    </ThemeProvider>
  </QueryClientProvider>,
);
```

Then you can remove `QueryClientProvider` from `App.tsx` and it will be shared globally.

---

## ENV Variables (if needed)

```bash
# .env
VITE_API_BASE_URL=http://localhost:8080/api
VITE_TENANT_ID=tenant-1
VITE_THEME_MODE=light
```

Then in your API hooks:

```typescript
const baseUrl = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/api';

useQuery({
  queryKey: ['dashboard-compliance', tenantId, valuationDate],
  queryFn: async () => {
    const res = await fetch(`${baseUrl}/dashboard/compliance?tenant_id=${tenantId}&valuation_date=${valuationDate}`);
    if (!res.ok) throw new Error('Failed to fetch');
    return res.json();
  },
});
```

---

## URL Navigation Examples

### From Sidebar
```typescript
// ConsoleSidebar.tsx
const navigate = (href: string) => {
  window.location.href = href;
};

<ListItemButton onClick={() => navigate('/console/dashboard')}>
  Dashboard
</ListItemButton>
```

### From Dashboard to Lineage
```typescript
// In a card click handler
window.location.href = `/console/risk/scenarios/${scenarioId}/lineage`;
```

### Dynamic Links in Tables
```typescript
// ETLRunTable.tsx
<TableCell>
  <Link href={`/console/etl/runs/${row.etl_run_id}`}>
    {row.etl_run_id}
  </Link>
</TableCell>
```

---

## Testing the Integration

### 1. Start your Go backend
```bash
go run ./cmd/server
# Server running on :8080
```

### 2. Check API endpoints exist
```bash
curl http://localhost:8080/api/dashboard/compliance?tenant_id=tenant-1&valuation_date=2024-01-15
```

### 3. Start your frontend dev server
```bash
cd frontend
npm run dev
# Vite running on :5173
```

### 4. Navigate to console
```
http://localhost:5173/console/dashboard
```

### 5. Verify:
- ✅ Sidebar appears on left
- ✅ TopBar appears on top with search + tenant switcher
- ✅ Dashboard loads with KPIs
- ✅ Data fetches from Go backend
- ✅ React Query DevTools shows queries (if installed)

---

## React Query DevTools (for debugging)

```bash
npm install @tanstack/react-query-devtools
```

Then in your App.tsx:

```typescript
import { ReactQueryDevtools } from '@tanstack/react-query-devtools';

export function App() {
  return (
    <>
      {/* Your routes */}
      <ReactQueryDevtools initialIsOpen={false} />
    </>
  );
}
```

Now press `Ctrl+Alt+Q` to open the devtools and debug queries.

---

## Common Issues & Solutions

### Issue: "Cannot find module"
**Solution**: Ensure all files are created in correct locations per the directory structure in RISK_AND_COMPLIANCE_CONSOLE.md

### Issue: "useQueryClient access error"
**Solution**: Wrap your component with `<QueryClientProvider>` (should be at app root)

### Issue: "API endpoints return 404"
**Solution**: Verify Go backend has these routes registered in chi router

### Issue: "Layout sidebar doesn't appear"
**Solution**: Ensure `<ConsoleLayout>` wraps your routes, not nested inside them

---

Done! Your console is ready to deploy. 🚀
