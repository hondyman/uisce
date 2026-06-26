import React, { useState, useCallback } from 'react';
import styles from './PortfolioAnalysisDashboard.module.css';
import { gql, useQuery } from '@apollo/client';
import {
  ChevronRight,
  ChevronLeft,
  TrendingUp,
  AlertTriangle,
  PieChart,
  BarChart3,
  Activity,
} from 'lucide-react';

const PORTFOLIO_DRILL_DOWN_QUERY = gql`
  query PortfolioDrillDown(
    $portfolioId: uuid!
    $dimension: String!
    $level: Int
    $asOfDate: date
    $tenantId: uuid
    $datasourceId: uuid
  ) {
    analyze_portfolio_drill_down(
      portfolio_id: $portfolioId
      dimension: $dimension
      level: $level
      as_of_date: $asOfDate
      tenant_id: $tenantId
      tenant_instance_id: $datasourceId
    ) {
      dimension_value
      dimension_level
      position_count
      market_value
      cost_basis
      unrealized_gain_loss
      gain_loss_pct
      weight_pct
      has_children
    }
  }
`;

const PORTFOLIO_PERFORMANCE_QUERY = gql`
  query PortfolioPerformance(
    $portfolioId: uuid!
    $startDate: date!
    $endDate: date
    $tenantId: uuid
    $datasourceId: uuid
  ) {
    calculate_portfolio_performance(
      portfolio_id: $portfolioId
      start_date: $startDate
      end_date: $endDate
      tenant_id: $tenantId
      tenant_instance_id: $datasourceId
    ) {
      period_name
      start_value
      end_value
      net_cash_flows
      total_return_pct
      time_weighted_return_pct
      days_held
    }
  }
`;

const CONCENTRATION_RISK_QUERY = gql`
  query ConcentrationRisk(
    $portfolioId: uuid!
    $dimension: String!
    $thresholdPct: Float
    $tenantId: uuid
    $datasourceId: uuid
  ) {
    analyze_concentration_risk(
      portfolio_id: $portfolioId
      dimension: $dimension
      threshold_pct: $thresholdPct
      tenant_id: $tenantId
      tenant_instance_id: $datasourceId
    ) {
      dimension_value
      concentration_pct
      market_value
      risk_level
      exceeds_threshold
    }
  }
`;

interface BreadcrumbItem {
  level: number;
  dimension: string;
  value: string;
}

interface PortfolioAnalysisDashboardProps {
  portfolioId: string;
  tenantId?: string;
  datasourceId?: string;
}

export const PortfolioAnalysisDashboard: React.FC<PortfolioAnalysisDashboardProps> = ({
  portfolioId,
  tenantId,
  datasourceId,
}) => {
  const [activeView, setActiveView] = useState<'analysis' | 'performance' | 'risk'>('analysis');
  const [breadcrumbs, setBreadcrumbs] = useState<BreadcrumbItem[]>([
    { level: 0, dimension: 'portfolio', value: portfolioId },
  ]);
  const [currentDimension, setCurrentDimension] = useState<string>('asset_class');
  const [selectedPeriod, setSelectedPeriod] = useState<'1M' | '3M' | '6M' | '1Y'>('1M');

  // Drill-down query
  const {
    data: drillData,
    loading: drillLoading,
    error: drillError,
  } = useQuery(PORTFOLIO_DRILL_DOWN_QUERY, {
    variables: {
      portfolioId,
      dimension: currentDimension,
      level: breadcrumbs.length,
      asOfDate: new Date().toISOString().split('T')[0],
      tenantId,
      datasourceId,
    },
    skip: activeView !== 'analysis',
  });

  // Performance query
  const {
    data: perfData,
    loading: perfLoading,
  } = useQuery(PORTFOLIO_PERFORMANCE_QUERY, {
    variables: {
      portfolioId,
      startDate: getPeriodStart(selectedPeriod),
      endDate: new Date().toISOString().split('T')[0],
      tenantId,
      datasourceId,
    },
    skip: activeView !== 'performance',
  });

  // Risk query
  const {
    data: riskData,
    loading: riskLoading,
  } = useQuery(CONCENTRATION_RISK_QUERY, {
    variables: {
      portfolioId,
      dimension: 'security',
      thresholdPct: 10,
      tenantId,
      datasourceId,
    },
    skip: activeView !== 'risk',
  });

  const handleDrillDown = useCallback((dimensionValue: string) => {
    const nextDim = getNextDimension(currentDimension);
    setBreadcrumbs((prev) => [
      ...prev,
      {
        level: prev.length,
        dimension: nextDim,
        value: dimensionValue,
      },
    ]);
    setCurrentDimension(nextDim);
  }, [currentDimension]);

  const handleDrillUp = useCallback((targetLevel: number) => {
    const newBreadcrumbs = breadcrumbs.slice(0, targetLevel + 1);
    setBreadcrumbs(newBreadcrumbs);

    // Set dimension based on breadcrumb level
    const dimensionMap: Record<number, string> = {
      0: 'asset_class',
      1: 'sector',
      2: 'industry',
      3: 'security',
    };
    const nextLevel = Math.min(targetLevel + 1, 3);
    setCurrentDimension(dimensionMap[nextLevel] || 'security');
  }, [breadcrumbs]);

  const getNextDimension = (current: string): string => {
    const hierarchy: Record<string, string> = {
      'asset_class': 'sector',
      'sector': 'industry',
      'industry': 'security',
      // treat geography as a branch that resolves to security in the drill-down
      'geography': 'security',
      'security': 'security',
    };
    return hierarchy[current] || 'security';
  };

  const formatCurrency = (value: number): string => {
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: 'USD',
      minimumFractionDigits: 0,
      maximumFractionDigits: 0,
    }).format(value);
  };

  const formatPercent = (value: number): string => {
    return `${value >= 0 ? '+' : ''}${value.toFixed(2)}%`;
  };

  const getRiskColor = (riskLevel: string): string => {
    const colors: Record<string, string> = {
      HIGH: 'text-red-600 bg-red-50',
      MEDIUM: 'text-yellow-600 bg-yellow-50',
      LOW: 'text-green-600 bg-green-50',
    };
    return colors[riskLevel] || 'text-gray-600 bg-gray-50';
  };

  return (
    <div className="space-y-6">
      {/* Navigation Tabs */}
      <div className="flex gap-4 border-b border-gray-200">
        {(['analysis', 'performance', 'risk'] as const).map((view) => (
          <button
            key={view}
            onClick={() => setActiveView(view)}
            className={`px-4 py-2 font-medium transition-colors ${
              activeView === view
                ? 'text-blue-600 border-b-2 border-blue-600'
                : 'text-gray-600 hover:text-gray-900'
            }`}
          >
            {view === 'analysis' && <BarChart3 className="inline mr-2 w-4 h-4" />}
            {view === 'performance' && <TrendingUp className="inline mr-2 w-4 h-4" />}
            {view === 'risk' && <AlertTriangle className="inline mr-2 w-4 h-4" />}
            {view.charAt(0).toUpperCase() + view.slice(1)}
          </button>
        ))}
      </div>

      {/* ANALYSIS VIEW */}
      {activeView === 'analysis' && (
        <div className="space-y-4">
          {/* Breadcrumb Navigation */}
          <div className="flex items-center gap-2 text-sm">
            {breadcrumbs.map((crumb, index) => (
              <React.Fragment key={index}>
                <button
                  onClick={() => handleDrillUp(index)}
                  className={`px-3 py-1 rounded transition-colors ${
                    index === breadcrumbs.length - 1
                      ? 'bg-blue-100 text-blue-700 font-medium'
                      : 'hover:bg-gray-100'
                  }`}
                >
                  {index === 0 ? 'Portfolio' : crumb.value}
                </button>
                {index < breadcrumbs.length - 1 && <ChevronRight className="w-4 h-4" />}
              </React.Fragment>
            ))}
          </div>

          {/* Dimension Selector */}
          <div className="flex gap-4">
            <div className="flex-1">
              <label htmlFor="view-by-select" className="block text-sm font-medium text-gray-700 mb-2">
                View By:
              </label>
              <select
                id="view-by-select"
                title="View By"
                aria-label="View By"
                value={currentDimension}
                onChange={(e: React.ChangeEvent<HTMLSelectElement>) => {
                  setCurrentDimension(e.target.value);
                  setBreadcrumbs([{ level: 0, dimension: 'portfolio', value: portfolioId }]);
                }}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
              >
                <option value="asset_class">Asset Class</option>
                <option value="sector">Sector</option>
                <option value="geography">Geography</option>
                <option value="security">Security</option>
              </select>
            </div>
          </div>

          {/* Drill-Down Table */}
          {drillLoading ? (
            <div className="text-center py-12">Loading...</div>
          ) : drillError ? (
            <div className="text-center py-12 text-red-600">Error loading data</div>
          ) : (
            <div className="border border-gray-200 rounded-lg overflow-hidden">
              <table className="w-full">
                <thead>
                  <tr className="bg-gray-50 border-b border-gray-200">
                    <th className="px-6 py-3 text-left text-sm font-semibold">
                      {currentDimension.replace('_', ' ')}
                    </th>
                    <th className="px-6 py-3 text-right text-sm font-semibold">Positions</th>
                    <th className="px-6 py-3 text-right text-sm font-semibold">Market Value</th>
                    <th className="px-6 py-3 text-right text-sm font-semibold">Cost Basis</th>
                    <th className="px-6 py-3 text-right text-sm font-semibold">Gain/Loss</th>
                    <th className="px-6 py-3 text-right text-sm font-semibold">Return %</th>
                    <th className="px-6 py-3 text-right text-sm font-semibold">Weight %</th>
                    <th className="px-6 py-3"></th>
                  </tr>
                </thead>
                <tbody>
                  {drillData?.analyze_portfolio_drill_down?.map((row: any, idx: number) => (
                    <tr key={idx} className="border-b border-gray-100 hover:bg-gray-50">
                      <td className="px-6 py-4 text-sm font-medium">{row.dimension_value}</td>
                      <td className="px-6 py-4 text-right text-sm">{row.position_count}</td>
                      <td className="px-6 py-4 text-right text-sm font-medium">
                        {formatCurrency(row.market_value)}
                      </td>
                      <td className="px-6 py-4 text-right text-sm">
                        {formatCurrency(row.cost_basis)}
                      </td>
                      <td
                        className={`px-6 py-4 text-right text-sm font-medium ${
                          row.unrealized_gain_loss >= 0 ? 'text-green-600' : 'text-red-600'
                        }`}
                      >
                        {formatCurrency(row.unrealized_gain_loss)}
                      </td>
                      <td
                        className={`px-6 py-4 text-right text-sm ${
                          row.gain_loss_pct >= 0 ? 'text-green-600' : 'text-red-600'
                        }`}
                      >
                        {formatPercent(row.gain_loss_pct)}
                      </td>
                      <td className="px-6 py-4 text-right text-sm">
                        <div className="flex items-center justify-end gap-2">
                          <div className="w-16 h-2 bg-gray-200 rounded-full overflow-hidden">
                            <div
                              className={`h-full bg-blue-500 ${styles.progressInner}`}
                              // set the CSS variable for the module-based rule
                              style={{ ['--progress-width' as any]: `${Math.min(row.weight_pct, 100)}%` } as React.CSSProperties}
                            />
                          </div>
                          <span className="text-xs font-medium w-12 text-right">
                            {row.weight_pct.toFixed(1)}%
                          </span>
                        </div>
                      </td>
                      <td className="px-6 py-4 text-right">
                        {row.has_children && (
                          <button
                            onClick={() => handleDrillDown(row.dimension_value)}
                            className="p-1 hover:bg-gray-200 rounded transition-colors"
                            aria-label={`Drill down into ${row.dimension_value}`}
                            title={`Drill down into ${row.dimension_value}`}
                          >
                            <ChevronRight className="w-4 h-4" />
                          </button>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      )}

      {/* PERFORMANCE VIEW */}
      {activeView === 'performance' && (
        <div className="space-y-4">
          {/* Period Selector */}
          <div className="flex gap-2">
            {(['1M', '3M', '6M', '1Y'] as const).map((period) => (
              <button
                key={period}
                onClick={() => setSelectedPeriod(period)}
                className={`px-4 py-2 rounded-lg font-medium transition-colors ${
                  selectedPeriod === period
                    ? 'bg-blue-600 text-white'
                    : 'bg-gray-100 hover:bg-gray-200'
                }`}
              >
                {period}
              </button>
            ))}
          </div>

          {/* Performance Metrics */}
          {perfLoading ? (
            <div className="text-center py-12">Loading...</div>
          ) : (
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              {perfData?.calculate_portfolio_performance?.map((perf: any, idx: number) => (
                <div key={idx} className="border border-gray-200 rounded-lg p-6 space-y-4">
                  <h3 className="font-semibold text-gray-900">{perf.period_name}</h3>

                  <div className="space-y-3">
                    <div className="flex justify-between">
                      <span className="text-gray-600">Starting Value:</span>
                      <span className="font-medium">{formatCurrency(perf.start_value)}</span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-gray-600">Ending Value:</span>
                      <span className="font-medium">{formatCurrency(perf.end_value)}</span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-gray-600">Net Cash Flows:</span>
                      <span className="font-medium">{formatCurrency(perf.net_cash_flows)}</span>
                    </div>
                    <div className="border-t border-gray-200 pt-3">
                      <div className="flex justify-between">
                        <span className="text-gray-900 font-semibold">Time-Weighted Return:</span>
                        <span
                          className={`font-bold text-lg ${
                            perf.time_weighted_return_pct >= 0
                              ? 'text-green-600'
                              : 'text-red-600'
                          }`}
                        >
                          {formatPercent(perf.time_weighted_return_pct)}
                        </span>
                      </div>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {/* RISK VIEW */}
      {activeView === 'risk' && (
        <div className="space-y-4">
          {riskLoading ? (
            <div className="text-center py-12">Loading...</div>
          ) : (
            <>
              {/* Summary Cards */}
              <div className="grid grid-cols-3 gap-4">
                <div className="border border-gray-200 rounded-lg p-4">
                  <p className="text-gray-600 text-sm">Total Positions</p>
                  <p className="text-2xl font-bold">
                    {riskData?.analyze_concentration_risk?.length || 0}
                  </p>
                </div>
                <div className="border border-red-200 bg-red-50 rounded-lg p-4">
                  <p className="text-gray-600 text-sm">High Risk (&gt;40%)</p>
                  <p className="text-2xl font-bold text-red-600">
                    {riskData?.analyze_concentration_risk?.filter(
                      (r: any) => r.risk_level === 'HIGH'
                    ).length || 0}
                  </p>
                </div>
                <div className="border border-yellow-200 bg-yellow-50 rounded-lg p-4">
                  <p className="text-gray-600 text-sm">Medium Risk (25-40%)</p>
                  <p className="text-2xl font-bold text-yellow-600">
                    {riskData?.analyze_concentration_risk?.filter(
                      (r: any) => r.risk_level === 'MEDIUM'
                    ).length || 0}
                  </p>
                </div>
              </div>

              {/* Concentration Table */}
              <div className="border border-gray-200 rounded-lg overflow-hidden">
                <table className="w-full">
                  <thead>
                    <tr className="bg-gray-50 border-b border-gray-200">
                      <th className="px-6 py-3 text-left text-sm font-semibold">Holding</th>
                      <th className="px-6 py-3 text-right text-sm font-semibold">
                        Concentration %
                      </th>
                      <th className="px-6 py-3 text-right text-sm font-semibold">
                        Market Value
                      </th>
                      <th className="px-6 py-3 text-left text-sm font-semibold">Risk Level</th>
                    </tr>
                  </thead>
                  <tbody>
                    {riskData?.analyze_concentration_risk?.map((risk: any, idx: number) => (
                      <tr
                        key={idx}
                        className={`border-b border-gray-100 ${
                          risk.exceeds_threshold ? 'bg-red-50' : ''
                        }`}
                      >
                        <td className="px-6 py-4 text-sm font-medium">{risk.dimension_value}</td>
                        <td className="px-6 py-4 text-right text-sm font-medium">
                          {risk.concentration_pct.toFixed(2)}%
                        </td>
                        <td className="px-6 py-4 text-right text-sm">
                          {formatCurrency(risk.market_value)}
                        </td>
                        <td className="px-6 py-4">
                          <span
                            className={`px-3 py-1 rounded-full text-sm font-medium ${getRiskColor(
                              risk.risk_level
                            )}`}
                          >
                            {risk.risk_level}
                          </span>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </>
          )}
        </div>
      )}
    </div>
  );
};

// Helper function
function getPeriodStart(period: string): string {
  const now = new Date();
  const dayMap: Record<string, number> = {
    '1D': 1,
    '1W': 7,
    '1M': 30,
    '3M': 90,
    '6M': 180,
    '1Y': 365,
  };
  const days = dayMap[period] || 30;
  const start = new Date(now.getTime() - days * 24 * 60 * 60 * 1000);
  return start.toISOString().split('T')[0];
}
