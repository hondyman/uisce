/**
 * Example: Complete Dashboard Integration
 * Shows how to wire everything together in your app
 */

// ============================================================================
// STEP 1: Configure QueryClient (in main.tsx or setup.ts)
// ============================================================================

import { QueryClient, QueryClientProvider } from 'react-query';

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 2,
      retryDelay: (attemptIndex) => Math.min(1000 * 2 ** attemptIndex, 30000),
      staleTime: 60 * 1000, // 1 minute
      cacheTime: 5 * 60 * 1000, // 5 minutes
    },
  },
});

export { queryClient };

// ============================================================================
// STEP 2: Wrap App with Providers (in main.tsx or App.tsx)
// ============================================================================

import React from 'react';
import { BrowserRouter as Router, Routes, Route } from 'react-router-dom';
import { DashboardProvider } from './contexts/DashboardContext';
import { queryClient } from './config/queryClient';

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <DashboardProvider>
        <Router>
          <Routes>
            <Route path="/dashboard" element={<DashboardHome />} />
            {/* Other routes */}
          </Routes>
        </Router>
      </DashboardProvider>
    </QueryClientProvider>
  );
}

export default App;

// ============================================================================
// STEP 3: Import and use Dashboard (in your route)
// ============================================================================

import { DashboardHome } from './pages/dashboard';

// Now available at: /dashboard

// ============================================================================
// STEP 4: Access Dashboard Data in Custom Components
// ============================================================================

import { useComplianceKPIs } from './hooks/useDashboardData';
import { useDashboardContext } from './contexts/DashboardContext';

export function CustomComplianceWidget() {
  const { selectedTenant, valuationDate } = useDashboardContext();
  const compliance = useComplianceKPIs(
    selectedTenant?.id || null,
    valuationDate
  );

  if (compliance.isLoading) return <div>Loading...</div>;
  if (compliance.error) return <div>Error loading compliance data</div>;

  return (
    <div>
      <h2>Compliance Summary</h2>
      <p>Pass Rate: {(compliance.data?.pass_rate || 0) * 100}%</p>
      <p>Hard Breaches: {compliance.data?.hard_breaches}</p>
    </div>
  );
}

// ============================================================================
// STEP 5: Build Tenant Selector (optional custom implementation)
// ============================================================================

import { useDashboardContext } from './contexts/DashboardContext';

export function TenantSwitcher() {
  const { selectedTenant, selectTenant } = useDashboardContext();

  const tenants = [
    { id: 'acme-asset-mgmt', name: 'Acme Asset Management' },
    { id: 'global-wealth', name: 'Global Wealth Partners' },
    { id: 'institutional-inv', name: 'Institutional Investors LLC' },
  ];

  return (
    <select
      value={selectedTenant?.id || ''}
      onChange={(e) => {
        const tenant = tenants.find((t) => t.id === e.target.value);
        if (tenant) selectTenant(tenant);
      }}
    >
      <option value="">Select Tenant...</option>
      {tenants.map((tenant) => (
        <option key={tenant.id} value={tenant.id}>
          {tenant.name}
        </option>
      ))}
    </select>
  );
}

// ============================================================================
// STEP 6: Add Custom Dashboard Page (extending DashboardHome)
// ============================================================================

import { DashboardHome as BaseDashboard } from './pages/dashboard';
import { ConsoleHeader, ConsoleLayout } from './pages/dashboard/LayoutComponents';

export function CustomRiskDashboard() {
  return (
    <ConsoleLayout>
      <ConsoleHeader
        title="Custom Risk Analysis"
        subtitle="Portfolio-level deep dive with factor attribution"
      />
      {/* Add custom components here */}
      <BaseDashboard />
    </ConsoleLayout>
  );
}

// ============================================================================
// STEP 7: Example - Auth Integration
// ============================================================================

import { useEffect, useState } from 'react';
import { useDashboardContext } from './contexts/DashboardContext';

export function ProtectedDashboard() {
  const { selectedTenant } = useDashboardContext();
  const [isAuthorized, setIsAuthorized] = useState(false);

  useEffect(() => {
    // Verify user has access to selected tenant
    if (selectedTenant) {
      fetch(`/api/auth/verify?tenant_id=${selectedTenant.id}`)
        .then((r) => r.ok && setIsAuthorized(true))
        .catch(() => setIsAuthorized(false));
    }
  }, [selectedTenant]);

  if (!selectedTenant) return <div>Please select a tenant</div>;
  if (!isAuthorized) return <div>Access denied to this tenant</div>;

  return <DashboardHome />;
}

// ============================================================================
// STEP 8: Example - Alerts Center Page
// ============================================================================

import { useAlerts } from './hooks/useDashboardData';
import { AlertsPanel } from './pages/dashboard/OperationsComponents';

export function AlertsCenterPage() {
  const { selectedTenant, valuationDate } = useDashboardContext();
  const alerts = useAlerts(selectedTenant?.id || null, valuationDate);

  return (
    <ConsoleLayout>
      <ConsoleHeader
        title="Alert Center"
        subtitle="All operational alerts across your portfolio"
      />
      <AlertsPanel
        data={alerts.data}
        isLoading={alerts.isLoading}
        error={alerts.error}
      />
    </ConsoleLayout>
  );
}

// ============================================================================
// STEP 9: Example - ETL Status Page
// ============================================================================

import { useETLHealth } from './hooks/useDashboardData';
import { ETLHealth } from './pages/dashboard/OperationsComponents';

export function ETLStatusPage() {
  const { selectedTenant } = useDashboardContext();
  const etl = useETLHealth(selectedTenant?.id || null);

  const handleTriggerETL = async () => {
    try {
      const response = await fetch('/api/dashboard/etl/trigger', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          tenant_id: selectedTenant?.id,
        }),
      });
      const data = await response.json();
      console.log('ETL triggered:', data);
      // Refetch status
      etl.refetch();
    } catch (error) {
      console.error('Failed to trigger ETL:', error);
    }
  };

  return (
    <ConsoleLayout>
      <ConsoleHeader
        title="ETL Operations"
        subtitle="Pipeline execution and performance metrics"
      />
      <ETLHealth
        data={etl.data}
        isLoading={etl.isLoading}
        error={etl.error}
        onTriggerRun={handleTriggerETL}
      />
    </ConsoleLayout>
  );
}

// ============================================================================
// STEP 10: Example - Navbar Integration
// ============================================================================

import { TenantSwitcher } from './components/TenantSwitcher';
import { useDashboardContext } from './contexts/DashboardContext';

export function MainNavbar() {
  const { selectedTenant } = useDashboardContext();

  return (
    <nav className="bg-slate-900 text-white p-4 flex items-center justify-between">
      <h1>Risk & Compliance Platform</h1>
      <div className="flex items-center gap-4">
        <TenantSwitcher />
        {selectedTenant && (
          <div className="text-sm text-slate-300">
            {selectedTenant.name}
          </div>
        )}
        <div className="w-10 h-10 rounded-full bg-blue-600 flex items-center justify-center">
          JD
        </div>
      </div>
    </nav>
  );
}

// ============================================================================
// STEP 11: Example - Environment Variables (.env.local)
// ============================================================================

/*
# Dashboard API Configuration
REACT_APP_API_BASE_URL=http://localhost:8080/api

# Query Client Configuration (optional)
REACT_APP_QUERY_STALE_TIME=60000
REACT_APP_QUERY_CACHE_TIME=300000
REACT_APP_QUERY_RETRY_COUNT=2

# Feature Flags (optional)
REACT_APP_ENABLE_ETL_TRIGGER=true
REACT_APP_ENABLE_ALERT_EXPORT=true
REACT_APP_DARK_MODE_DEFAULT=false
*/

// ============================================================================
// STEP 12: Example - App.tsx with All Wiring
// ============================================================================

import React, { useEffect } from 'react';
import { BrowserRouter, Routes, Route } from 'react-router-dom';
import { QueryClientProvider } from 'react-query';
import { DashboardProvider } from './contexts/DashboardContext';
import { queryClient } from './config/queryClient';
import { MainNavbar } from './components/MainNavbar';
import { DashboardHome } from './pages/dashboard';
import { AlertsCenterPage } from './pages/AlertsCenter';
import { ETLStatusPage } from './pages/ETLStatus';

function AppContent() {
  return (
    <>
      <MainNavbar />
      <Routes>
        <Route path="/dashboard" element={<DashboardHome />} />
        <Route path="/alerts" element={<AlertsCenterPage />} />
        <Route path="/etl" element={<ETLStatusPage />} />
      </Routes>
    </>
  );
}

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <DashboardProvider>
        <BrowserRouter>
          <AppContent />
        </BrowserRouter>
      </DashboardProvider>
    </QueryClientProvider>
  );
}

// ============================================================================
// STEP 13: Example - Mock API Response (for dev/testing)
// ============================================================================

// Intercept fetch requests with MSW (Mock Service Worker) or similar

export const dashboardMocks = {
  complianceKPIs: {
    total_rules: 1240,
    pass_rate: 0.92,
    hard_breaches: 12,
    soft_breaches: 34,
    top_failing_rules: [
      { rule_code: 'MAX_ISSUER_5', failures: 7 },
      { rule_code: 'SECTOR_LIMIT_20', failures: 5 },
    ],
  },

  riskKPIs: {
    avg_volatility: 0.112,
    avg_var_95: 0.031,
    avg_var_99: 0.052,
    worst_scenario: {
      scenario_id: '550e8400-uuid',
      name: 'Equity -20%',
      pnl: -1234567.89,
    },
    top_factors: [
      { factor_id: 'VALUE', contribution: 0.07 },
      { factor_id: 'SIZE', contribution: 0.04 },
    ],
  },

  sparklines: {
    pass_rate: Array.from({ length: 7 }, (_, i) => ({
      date: new Date(Date.now() - (6 - i) * 24 * 60 * 60 * 1000)
        .toISOString()
        .split('T')[0],
      value: 0.91 + Math.random() * 0.02,
    })),
    hard_breaches: Array.from({ length: 7 }, (_, i) => ({
      date: new Date(Date.now() - (6 - i) * 24 * 60 * 60 * 1000)
        .toISOString()
        .split('T')[0],
      value: 10 + Math.random() * 10,
    })),
    volatility: Array.from({ length: 7 }, (_, i) => ({
      date: new Date(Date.now() - (6 - i) * 24 * 60 * 60 * 1000)
        .toISOString()
        .split('T')[0],
      value: 0.10 + Math.random() * 0.02,
    })),
    etl_duration: Array.from({ length: 7 }, (_, i) => ({
      date: new Date(Date.now() - (6 - i) * 24 * 60 * 60 * 1000)
        .toISOString()
        .split('T')[0],
      value: 120 + Math.random() * 20,
    })),
  },

  etlHealth: {
    last_run: {
      etl_run_id: '550e8400-uuid',
      status: 'SUCCESS',
      duration_ms: 132000,
      rules_evaluated: 1240,
      scenarios_evaluated: 32,
      wasm_version: 'risk-compliance-v1.3.2',
    },
  },

  alerts: {
    hard_breaches: [
      {
        rule_code: 'MAX_ISSUER_5',
        portfolio_id: '550e8400-portfolio',
        metric: 0.061,
      },
    ],
    scenario_losses: [
      {
        scenario_id: '550e8400-scenario',
        name: 'Equity -20%',
        pnl: -1234567.89,
      },
    ],
    etl_failures: [],
    soft_breaches: [],
    reg_breaches: [],
  },
};

// ============================================================================
// Done! Your dashboard is fully integrated 🎉
// ============================================================================
