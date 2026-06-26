import React, { useState } from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { usePortfolioData } from './usePortfolioData';
import { PortfolioDetailPage } from './PortfolioDetailPage';
import { 
  PortfolioOverviewCard, 
  RiskSnapshotCard, 
  ComplianceSnapshotCard,
} from './PortfolioCards';
import {
  HoldingsTable,
  SectorWeights,
  ScenarioChart,
} from './PortfolioCharts';
import { DashboardProvider } from '../dashboard/DashboardContext';

/**
 * EXAMPLE 1: Setting up React Query Provider
 * 
 * Wrap your entire app with QueryClientProvider to enable React Query hooks
 */
export function App() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        staleTime: 60 * 1000,        // 60 seconds
        cacheTime: 5 * 60 * 1000,    // 5 minutes
        retry: 2,
        retryDelay: 1000,
      },
    },
  });

  return (
    <QueryClientProvider client={queryClient}>
      <DashboardProvider>
        {/* Your app routes */}
      </DashboardProvider>
    </QueryClientProvider>
  );
}

/**
 * EXAMPLE 2: Simple Portfolio Detail Page
 * 
 * Minimal setup to render the entire portfolio page
 */
export function SimplePortfolioExample() {
  return <PortfolioDetailPage />;
}

/**
 * EXAMPLE 3: Custom Portfolio Component with State Management
 * 
 * More complex setup with custom handlers and notifications
 */
export function CustomPortfolioComponent() {
  const [activeTab, setActiveTab] = useState('overview');
  const [refreshing, setRefreshing] = useState(false);

  const handleRefresh = async () => {
    setRefreshing(true);
    // Simulate refresh delay
    await new Promise(resolve => setTimeout(resolve, 1000));
    setRefreshing(false);
  };

  return (
    <div className="space-y-4">
      <div className="flex justify-between items-center">
        <h1 className="text-2xl font-bold">Portfolio Analysis</h1>
        <button 
          onClick={handleRefresh}
          disabled={refreshing}
          className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 disabled:opacity-50"
        >
          {refreshing ? '⏳ Refreshing...' : '🔄 Refresh'}
        </button>
      </div>

      {/* Tab Navigation */}
      <div className="flex gap-2 border-b">
        {['overview', 'holdings', 'risk', 'compliance', 'scenarios'].map(tab => (
          <button
            key={tab}
            onClick={() => setActiveTab(tab)}
            className={`px-4 py-2 font-medium ${
              activeTab === tab 
                ? 'text-blue-600 border-b-2 border-blue-600'
                : 'text-gray-600 hover:text-gray-900'
            }`}
          >
            {tab.replace('_', ' ').toUpperCase()}
          </button>
        ))}
      </div>

      {/* Tab Content */}
      <div>
        {activeTab === 'overview' && <PortfolioOverviewTab />}
        {activeTab === 'holdings' && <PortfolioHoldingsTab />}
        {activeTab === 'risk' && <PortfolioRiskTab />}
        {activeTab === 'compliance' && <PortfolioComplianceTab />}
        {activeTab === 'scenarios' && <PortfolioScenariosTab />}
      </div>
    </div>
  );
}

/**
 * EXAMPLE 4: Individual Tab Components
 */

function PortfolioOverviewTab() {
  const portfolio = usePortfolioData('PF-001', '2024-01-15');

  if (portfolio.isLoading) return <div className="p-8 text-center">Loading portfolio...</div>;
  if (portfolio.isError) return <div className="p-8 text-red-600">Error loading portfolio</div>;

  return (
    <div className="grid grid-cols-3 gap-4">
      <PortfolioOverviewCard data={portfolio.overview.data} />
      <RiskSnapshotCard data={portfolio.risk.data} />
      <ComplianceSnapshotCard data={portfolio.compliance.data} />
    </div>
  );
}

function PortfolioHoldingsTab() {
  const portfolio = usePortfolioData('PF-001', '2024-01-15');

  if (portfolio.isLoading) return <div className="p-8">Loading holdings...</div>;

  return (
    <div className="space-y-6">
      <div className="grid grid-cols-2 gap-4">
        <div>
          <h3 className="font-bold mb-4">Sector Allocation</h3>
          <SectorWeights data={portfolio.holdings.data?.sector_weights} />
        </div>
        <div>
          <h3 className="font-bold mb-4">Geographic Distribution</h3>
          <SectorWeights data={portfolio.holdings.data?.country_weights?.map(c => ({
            sector: c.country,
            weight: c.weight,
          }))} />
        </div>
      </div>
      <div>
        <h3 className="font-bold mb-4">Top Holdings</h3>
        <HoldingsTable data={portfolio.holdings.data} />
      </div>
    </div>
  );
}

function PortfolioRiskTab() {
  const portfolio = usePortfolioData('PF-001', '2024-01-15');

  if (portfolio.isLoading) return <div className="p-8">Loading risk data...</div>;

  return (
    <div className="space-y-6">
      <RiskSnapshotCard data={portfolio.risk.data} />
      
      {/* Factor Exposures */}
      <div className="bg-white p-6 rounded-lg border">
        <h3 className="font-bold text-lg mb-4">Factor Exposures</h3>
        <div className="space-y-4">
          {portfolio.risk.data?.factor_exposures.map(factor => (
            <div key={factor.factor_id} className="flex items-center gap-4">
              <span className="w-24 font-medium">{factor.factor_id}</span>
              <div className="flex-1 bg-gray-200 rounded-full h-2">
                <div
                  className={`h-full rounded-full ${
                    factor.exposure > 0 ? 'bg-green-500' : 'bg-red-500'
                  }`}
                  style={{ width: `${Math.min(Math.abs(factor.exposure) * 20, 100)}%` }}
                />
              </div>
              <span className="w-16 text-right font-mono">
                {factor.exposure > 0 ? '+' : ''}{factor.exposure.toFixed(2)}
              </span>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

function PortfolioComplianceTab() {
  const portfolio = usePortfolioData('PF-001', '2024-01-15');

  if (portfolio.isLoading) return <div className="p-8">Loading compliance data...</div>;

  return (
    <div className="space-y-6">
      <ComplianceSnapshotCard data={portfolio.compliance.data} />

      {/* Breaches */}
      <div className="space-y-4">
        <h3 className="font-bold text-lg">Active Breaches</h3>
        
        {portfolio.compliance.data?.hard_breaches.map(breach => (
          <div 
            key={breach.rule_code}
            className="p-4 bg-red-50 border-l-4 border-red-500 rounded"
          >
            <h4 className="font-bold text-red-800">🚨 Hard Breach: {breach.rule_code}</h4>
            <p className="text-sm text-red-700">
              Current: {breach.metric_value.toFixed(3)} | Limit: {breach.threshold_value.toFixed(3)}
            </p>
          </div>
        ))}

        {portfolio.compliance.data?.soft_breaches.map(breach => (
          <div 
            key={breach.rule_code}
            className="p-4 bg-amber-50 border-l-4 border-amber-500 rounded"
          >
            <h4 className="font-bold text-amber-800">⚠️ Soft Breach: {breach.rule_code}</h4>
            <p className="text-sm text-amber-700">
              Current: {breach.metric_value.toFixed(3)} | Limit: {breach.threshold_value.toFixed(3)}
            </p>
          </div>
        ))}

        {
          portfolio.compliance.data &&
          portfolio.compliance.data.hard_breaches.length === 0 &&
          portfolio.compliance.data.soft_breaches.length === 0 && (
            <div className="p-4 bg-green-50 border border-green-200 rounded text-green-800">
              ✓ No compliance breaches detected
            </div>
          )
        }
      </div>
    </div>
  );
}

function PortfolioScenariosTab() {
  const portfolio = usePortfolioData('PF-001', '2024-01-15');

  if (portfolio.isLoading) return <div className="p-8">Loading scenarios...</div>;

  return (
    <div className="space-y-6">
      <div className="bg-white p-6 rounded-lg border">
        <h3 className="font-bold text-lg mb-4">Scenario Impact Analysis</h3>
        <ScenarioChart data={portfolio.scenarios.data?.results} />
      </div>

      {/* Detailed Results Table */}
      <div className="overflow-x-auto">
        <table className="w-full">
          <thead>
            <tr className="border-b bg-gray-50">
              <th className="text-left p-3 font-bold">Scenario</th>
              <th className="text-right p-3 font-bold">PnL Impact</th>
              <th className="text-right p-3 font-bold">% Change</th>
            </tr>
          </thead>
          <tbody>
            {portfolio.scenarios.data?.results.map(scenario => (
              <tr key={scenario.scenario_id} className="border-b hover:bg-gray-50">
                <td className="p-3">
                  <div>
                    <p className="font-medium">{scenario.name}</p>
                    <p className="text-sm text-gray-600">{scenario.description}</p>
                  </div>
                </td>
                <td className={`text-right p-3 font-mono font-bold ${
                  scenario.pnl < 0 ? 'text-red-600' : 'text-green-600'
                }`}>
                  {scenario.pnl < 0 ? '-' : '+'}${Math.abs(scenario.pnl / 1000000).toFixed(1)}M
                </td>
                <td className={`text-right p-3 font-mono ${
                  scenario.pnl_pct < 0 ? 'text-red-600' : 'text-green-600'
                }`}>
                  {scenario.pnl_pct > 0 ? '+' : ''}{(scenario.pnl_pct * 100).toFixed(2)}%
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}

/**
 * EXAMPLE 5: Error Handling Pattern
 * 
 * Graceful error handling with user feedback
 */
export function PortfolioWithErrorHandling({ portfolioId }: { portfolioId: string }) {
  const portfolio = usePortfolioData(portfolioId, '2024-01-15');

  if (portfolio.isLoading) {
    return (
      <div className="flex items-center justify-center h-screen">
        <div className="text-center">
          <div className="animate-spin text-4xl mb-4">📊</div>
          <p>Loading portfolio data...</p>
        </div>
      </div>
    );
  }

  if (portfolio.isError) {
    return (
      <div className="max-w-md mx-auto mt-10 p-6 bg-red-50 border-2 border-red-300 rounded-lg">
        <h2 className="text-lg font-bold text-red-800 mb-2">⚠️ Could not load portfolio</h2>
        <p className="text-red-700 mb-4">
          {portfolio.error?.message || 'An unknown error occurred'}
        </p>
        <button 
          onClick={() => window.location.reload()}
          className="w-full px-4 py-2 bg-red-600 text-white rounded hover:bg-red-700"
        >
          Try Again
        </button>
      </div>
    );
  }

  return <PortfolioDetailPage />;
}

/**
 * EXAMPLE 6: Real-time Portfolio Monitoring
 * 
 * Auto-refresh portfolio data at specified intervals
 */
export function RealTimePortfolioMonitor() {
  const portfolio = usePortfolioData('PF-001', '2024-01-15');
  const [lastUpdated, setLastUpdated] = React.useState(new Date());

  React.useEffect(() => {
    // Set up auto-refresh interval
    const interval = setInterval(() => {
      setLastUpdated(new Date());
      // React Query will handle refetching automatically
    }, 60000); // Refresh every minute

    return () => clearInterval(interval);
  }, []);

  return (
    <div>
      <div className="flex justify-between items-center mb-4">
        <h1 className="text-2xl font-bold">Live Portfolio Monitor</h1>
        <span className="text-sm text-gray-600">
          Last updated: {lastUpdated.toLocaleTimeString()}
        </span>
      </div>
      <PortfolioDetailPage />
    </div>
  );
}

/**
 * EXAMPLE 7: Multi-Portfolio Dashboard
 * 
 * Display multiple portfolios side-by-side
 */
export function MultiPortfolioDashboard() {
  const portfolioIds = ['PF-001', 'PF-002', 'PF-003'];
  const valuationDate = '2024-01-15';

  return (
    <div className="space-y-6">
      <h1 className="text-3xl font-bold">Portfolio Overview</h1>
      <div className="grid grid-cols-3 gap-4">
        {portfolioIds.map(id => (
          <PortfolioCard key={id} portfolioId={id} valuationDate={valuationDate} />
        ))}
      </div>
    </div>
  );
}

function PortfolioCard({ portfolioId, valuationDate }: { portfolioId: string; valuationDate: string }) {
  const portfolio = usePortfolioData(portfolioId, valuationDate);

  if (portfolio.isLoading) {
    return <div className="bg-white p-6 rounded-lg border animate-pulse">Loading...</div>;
  }

  if (portfolio.isError) {
    return <div className="bg-red-50 p-6 rounded-lg border border-red-200">Error loading portfolio</div>;
  }

  return (
    <div className="bg-white p-6 rounded-lg border hover:shadow-lg transition-shadow cursor-pointer">
      <h3 className="font-bold text-lg mb-2">{portfolio.overview.data?.name}</h3>
      <div className="space-y-2 text-sm">
        <p><span className="text-gray-600">AUM:</span> ${(portfolio.overview.data?.aum_usd || 0 / 1000000).toFixed(1)}M</p>
        <p><span className="text-gray-600">Strategy:</span> {portfolio.overview.data?.strategy}</p>
        <p><span className={`font-bold ${portfolio.overview.data?.ytd_return? 'text-green-600' : 'text-red-600'}`}>
          YTD: {((portfolio.overview.data?.ytd_return || 0) * 100).toFixed(2)}%
        </span></p>
      </div>
    </div>
  );
}

/**
 * EXAMPLE 8: Export Portfolio Data
 * 
 * Export portfolio data to CSV/PDF
 */
export function ExportPortfolioData() {
  const portfolio = usePortfolioData('PF-001', '2024-01-15');

  const handleExportCSV = () => {
    if (!portfolio.holdings.data) return;

    const headers = ['Security', 'Sector', 'Weight', '1D Change', 'YTD Change'];
    const rows = portfolio.holdings.data.top_holdings.map(h => [
      h.name,
      h.sector,
      (h.weight * 100).toFixed(2) + '%',
      (h.change_pct_1d * 100).toFixed(2) + '%',
      (h.change_pct_ytd * 100).toFixed(2) + '%',
    ]);

    const csv = [
      headers.join(','),
      ...rows.map(r => r.join(',')),
    ].join('\n');

    const blob = new Blob([csv], { type: 'text/csv' });
    const url = window.URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `portfolio-holdings-${new Date().toISOString().split('T')[0]}.csv`;
    a.click();
  };

  return (
    <button
      onClick={handleExportCSV}
      className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700"
    >
      📥 Export Holdings (CSV)
    </button>
  );
}

/**
 * EXAMPLE 9: Custom Loading Skeleton
 * 
 * Create custom loading states that match component layout
 */
export function PortfolioLoadingSkeleton() {
  return (
    <div className="space-y-4">
      {/* Header skeleton */}
      <div className="h-10 bg-gray-200 rounded animate-pulse w-1/3" />

      {/* Cards skeleton */}
      <div className="grid grid-cols-3 gap-4">
        {[1, 2, 3].map(i => (
          <div key={i} className="bg-white p-6 rounded-lg border">
            <div className="h-4 bg-gray-200 rounded animate-pulse mb-3" />
            <div className="h-8 bg-gray-200 rounded animate-pulse mb-3" />
            <div className="h-4 bg-gray-200 rounded animate-pulse w-2/3" />
          </div>
        ))}
      </div>

      {/* Table skeleton */}
      <div className="bg-white p-6 rounded-lg border">
        <div className="h-6 bg-gray-200 rounded animate-pulse mb-4" />
        {[1, 2, 3, 4, 5].map(i => (
          <div key={i} className="h-4 bg-gray-100 rounded mt-3 animate-pulse" />
        ))}
      </div>
    </div>
  );
}

/**
 * EXAMPLE 10: Dark Mode Toggle
 * 
 * Implement dark mode switching in portfolio component
 */
export function PortfolioWithDarkMode() {
  const [isDark, setIsDark] = React.useState(false);

  React.useEffect(() => {
    if (isDark) {
      document.documentElement.classList.add('dark');
    } else {
      document.documentElement.classList.remove('dark');
    }
  }, [isDark]);

  return (
    <div>
      <button
        onClick={() => setIsDark(!isDark)}
        className="px-4 py-2 bg-gray-200 dark:bg-gray-700 rounded mb-4"
      >
        {isDark ? '☀️' : '🌙'} {isDark ? 'Light' : 'Dark'} Mode
      </button>
      <PortfolioDetailPage />
    </div>
  );
}

/**
 * EXAMPLE 11: Responsive Portfolio Layout
 * 
 * Mobile-optimized portfolio display
 */
export function ResponsivePortfolio() {
  return (
    <div className="w-full max-w-6xl mx-auto">
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {/* Mobile: 1 col | Tablet: 2 cols | Desktop: 3 cols */}
      </div>
    </div>
  );
}

/**
 * EXAMPLE 12: Portfolio Analytics Hooks
 * 
 * Custom hook for portfolio analytics
 */
function usePortfolioAnalytics(portfolioId: string, valuationDate: string) {
  const portfolio = usePortfolioData(portfolioId, valuationDate);

  return {
    // Calculated metrics
    concentration: portfolio.holdings.data?.top_holdings ? (
      portfolio.holdings.data.top_holdings.slice(0, 5).reduce((sum, h) => sum + h. weight, 0)
    ) : 0,
    
    topSector: portfolio.holdings.data?.sector_weights?.[0]?.sector,
    topSectorWeight: portfolio.holdings.data?.sector_weights?.[0]?.weight,
    
    complianceScore: portfolio.compliance.data?.pass_rate || 0,
    
    riskLevel: portfolio.risk.data?.volatility_pct || 0,
    
    scenarioWorstCase: portfolio.scenarios.data?.results.reduce((min, s) => 
      s.pnl < min ? s.pnl : min, 0
    ) || 0,
  };
}

export function PortfolioAnalyticsDashboard() {
  const analytics = usePortfolioAnalytics('PF-001', '2024-01-15');

  return (
    <div className="space-y-4">
      <div className="grid grid-cols-2 gap-4">
        <div className="bg-blue-50 p-4 rounded">
          <p className="text-sm text-gray-600">Top 5 Concentration</p>
          <p className="text-2xl font-bold">{(analytics.concentration * 100).toFixed(1)}%</p>
        </div>
        <div className="bg-green-50 p-4 rounded">
          <p className="text-sm text-gray-600">Compliance Score</p>
          <p className="text-2xl font-bold">{(analytics.complianceScore * 100).toFixed(1)}%</p>
        </div>
      </div>
    </div>
  );
}

/**
 * EXAMPLE 13: Portfolio Comparison Tool
 * 
 * Compare two portfolios side-by-side
 */
export function PortfolioComparison() {
  const portfolio1 = usePortfolioData('PF-001', '2024-01-15');
  const portfolio2 = usePortfolioData('PF-002', '2024-01-15');

  if (portfolio1.isLoading || portfolio2.isLoading) {
    return <div>Loading...</div>;
  }

  return (
    <table className="w-full">
      <thead>
        <tr className="border-b">
          <th className="text-left p-3">Metric</th>
          <th className="text-right p-3">{portfolio1.overview.data?.name}</th>
          <th className="text-right p-3">{portfolio2.overview.data?.name}</th>
        </tr>
      </thead>
      <tbody>
        <tr className="border-b">
          <td className="p-3">AUM</td>
          <td className="text-right p-3">${(portfolio1.overview.data?.aum_usd || 0 / 1000000).toFixed(1)}M</td>
          <td className="text-right p-3">${(portfolio2.overview.data?.aum_usd || 0 / 1000000).toFixed(1)}M</td>
        </tr>
        <tr className="border-b">
          <td className="p-3">Volatility</td>
          <td className="text-right p-3">{(portfolio1.risk.data?.volatility_pct || 0 * 100).toFixed(2)}%</td>
          <td className="text-right p-3">{(portfolio2.risk.data?.volatility_pct || 0 * 100).toFixed(2)}%</td>
        </tr>
        <tr>
          <td className="p-3">Compliance</td>
          <td className="text-right p-3">{(portfolio1.compliance.data?.pass_rate || 0 * 100).toFixed(1)}%</td>
          <td className="text-right p-3">{(portfolio2.compliance.data?.pass_rate || 0 * 100).toFixed(1)}%</td>
        </tr>
      </tbody>
    </table>
  );
}
