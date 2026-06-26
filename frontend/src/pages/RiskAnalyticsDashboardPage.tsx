import React, { useState, useEffect } from 'react';
import {
  AlertTriangle,
  TrendingDown,
  Activity,
  BarChart3,
  LineChart,
  PieChart as PieChartIcon,
  RefreshCw,
  AlertCircle,
  CheckCircle,
  Eye,
  EyeOff,
  Download,
  Filter,
} from 'lucide-react';
import { useTenant } from '../contexts/TenantContext';
import { getSelectedRegion } from '../lib/region';
import { devLog } from '../utils/devLogger';
import { getCardClasses, getTextClasses, getButtonClasses, getAlertClasses } from '../utils/darkModeHelpers';

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
  valueAtRisk: number; // VaR 95%
  conditionalVaR: number; // CVaR 95%
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

interface RiskFactor {
  factorName: string;
  exposure: number;
  sensitivity: number;
  contribution: number;
}

interface BacktestComparison {
  portfolioId: string;
  rec1: string;
  rec2: string;
  winner: string;
  performanceDiff: number;
  riskDiff: number;
  sharpeDiff: number;
  createdAt: string;
}

/**
 * Risk Analytics Dashboard Page
 * Comprehensive risk analysis, metrics visualization, and stress testing
 */
export const RiskAnalyticsDashboardPage: React.FC = () => {
  const { tenant, datasource } = useTenant();

  const [portfolios, setPortfolios] = useState<RiskMetrics[]>([]);
  const [selectedPortfolio, setSelectedPortfolio] = useState<RiskMetrics | null>(null);
  const [riskFactors, setRiskFactors] = useState<RiskFactor[]>([]);
  const [comparisons, setComparisons] = useState<BacktestComparison[]>([]);
  const [loading, setLoading] = useState(false);
  const [refreshing, setRefreshing] = useState(false);

  const [showAdvanced, setShowAdvanced] = useState(false);
  const [riskTolerance, setRiskTolerance] = useState('medium');

  const [toast, setToast] = useState<{ type: 'success' | 'error'; message: string } | null>(null);

  // Initialize
  useEffect(() => {
    if (tenant && datasource) {
      loadRiskMetrics();
      devLog('Risk Analytics Dashboard initialized', { tenantId: tenant.id, datasourceId: datasource.id });
    }
  }, [tenant, datasource]);

  const showToast = (type: 'success' | 'error', message: string) => {
    setToast({ type, message });
    setTimeout(() => setToast(null), 3000);
  };

  const loadRiskMetrics = async () => {
    setLoading(true);
    try {
      const response = await fetch('/api/portfolio-risk-metrics', {
        headers: {
          'X-User-ID': tenant?.id || '',
          'X-Tenant-ID': tenant?.id || '',
          'X-Tenant-Datasource-ID': datasource?.id || '',
          'X-Tenant-Region': getSelectedRegion(),
        },
      });

      if (!response.ok) throw new Error('Failed to fetch risk metrics');
      const data = await response.json();
      setPortfolios(data || []);
      if (data && data.length > 0) {
        setSelectedPortfolio(data[0]);
        loadRiskFactors(data[0].portfolioId);
      }
      devLog('Risk metrics loaded', { count: data?.length || 0 });
    } catch (error) {
      console.error('Failed to load risk metrics:', error);
      showToast('error', 'Failed to load risk metrics');
    } finally {
      setLoading(false);
    }
  };

  const loadRiskFactors = async (portfolioId: string) => {
    try {
      const response = await fetch(`/api/risk-factors?portfolio_id=${portfolioId}`, {
        headers: {
          'X-User-ID': tenant?.id || '',
          'X-Tenant-ID': tenant?.id || '',
          'X-Tenant-Datasource-ID': datasource?.id || '',
        },
      });

      if (!response.ok) throw new Error('Failed to fetch risk factors');
      const data = await response.json();
      setRiskFactors(data || []);
      devLog('Risk factors loaded', { portfolioId, count: data?.length || 0 });
    } catch (error) {
      console.error('Failed to load risk factors:', error);
    }
  };

  const getRiskLevel = (volatility: number, drawdown: number): { level: string; color: string } => {
    if (volatility > 20 || drawdown < -30) {
      return { level: 'High Risk', color: 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-200' };
    }
    if (volatility > 12 || drawdown < -20) {
      return { level: 'Medium Risk', color: 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-200' };
    }
    return { level: 'Low Risk', color: 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-200' };
  };

  const getMetricStatus = (value: number, thresholds: { good: number; warning: number }, isHigherBetter: boolean) => {
    if (isHigherBetter) {
      if (value >= thresholds.good) return { status: 'good', color: 'text-green-600 dark:text-green-400' };
      if (value >= thresholds.warning) return { status: 'warning', color: 'text-yellow-600 dark:text-yellow-400' };
      return { status: 'bad', color: 'text-red-600 dark:text-red-400' };
    } else {
      if (value <= thresholds.good) return { status: 'good', color: 'text-green-600 dark:text-green-400' };
      if (value <= thresholds.warning) return { status: 'warning', color: 'text-yellow-600 dark:text-yellow-400' };
      return { status: 'bad', color: 'text-red-600 dark:text-red-400' };
    }
  };

  if (!tenant || !datasource) {
    return (
      <div className={`${getCardClasses()} p-8 bg-gradient-to-br from-blue-50 to-blue-50/50 dark:from-blue-950/20 dark:to-blue-950/10`}>
        <AlertCircle className="w-6 h-6 text-yellow-600 dark:text-yellow-400 mb-2" />
        <p className={getTextClasses('primary')}>Please select a tenant and datasource to view risk analytics.</p>
      </div>
    );
  }

  return (
    <div className={`${getCardClasses()} space-y-6 p-6`}>
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className={`text-3xl font-bold ${getTextClasses('primary')}`}>Risk Analytics Dashboard</h1>
          <p className={`${getTextClasses('secondary')} mt-1`}>Portfolio risk metrics, analysis, and stress testing</p>
        </div>
        <div className="flex items-center gap-3">
          <button
            onClick={async () => {
              setRefreshing(true);
              await loadRiskMetrics();
              setRefreshing(false);
            }}
            disabled={refreshing}
            className={`${getButtonClasses('secondary')} flex items-center gap-2 disabled:opacity-50`}
            title="Refresh risk metrics"
            aria-label="Refresh risk metrics"
          >
            <RefreshCw className={`w-5 h-5 ${refreshing ? 'animate-spin' : ''}`} />
            Refresh
          </button>
          <button
            className={`${getButtonClasses('secondary')} flex items-center gap-2`}
            title="Export risk report"
            aria-label="Export risk report"
          >
            <Download className="w-5 h-5" />
            Export
          </button>
        </div>
      </div>

      {/* Toast */}
      {toast && (
        <div className={`p-4 rounded-lg flex items-center gap-3 ${toast.type === 'success' ? getAlertClasses('success') : getAlertClasses('error')}`}>
          {toast.type === 'success' ? (
            <CheckCircle className="w-5 h-5 text-green-600 dark:text-green-400" />
          ) : (
            <AlertCircle className="w-5 h-5 text-red-600 dark:text-red-400" />
          )}
          <span className={toast.type === 'success' ? 'text-green-800 dark:text-green-200' : 'text-red-800 dark:text-red-200'}>
            {toast.message}
          </span>
        </div>
      )}

      {loading ? (
        <div className="text-center py-12">
          <div className="animate-spin inline-block w-8 h-8 border-4 border-slate-300 dark:border-slate-600 border-t-blue-600 rounded-full"></div>
          <p className={`${getTextClasses('secondary')} mt-4`}>Loading risk metrics...</p>
        </div>
      ) : portfolios.length === 0 ? (
        <div className="text-center py-12 border-2 border-dashed border-slate-300 dark:border-slate-600 rounded-lg">
          <AlertTriangle className="w-12 h-12 text-slate-400 dark:text-slate-500 mx-auto mb-4" />
          <p className={getTextClasses('secondary')}>No portfolios with risk metrics available</p>
        </div>
      ) : (
        <>
          {/* Portfolio Selector */}
          <div className="flex items-center gap-4">
            <select
              value={selectedPortfolio?.portfolioId || ''}
              onChange={(e) => {
                const selected = portfolios.find((p) => p.portfolioId === e.target.value);
                if (selected) {
                  setSelectedPortfolio(selected);
                  loadRiskFactors(selected.portfolioId);
                }
              }}
              className={`${getCardClasses()} px-4 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500`}
              title="Select portfolio"
              aria-label="Select portfolio"
            >
              {portfolios.map((p) => (
                <option key={p.portfolioId} value={p.portfolioId}>
                  {p.portfolioName}
                </option>
              ))}
            </select>

            <select
              value={riskTolerance}
              onChange={(e) => setRiskTolerance(e.target.value)}
              className="px-4 py-2 border border-slate-300 dark:border-border-dark rounded-lg bg-white dark:bg-surface-dark text-slate-900 dark:text-text-light focus:outline-none focus:ring-2 focus:ring-blue-500"
              title="Select risk tolerance"
              aria-label="Select risk tolerance"
            >
              <option value="low">Low Risk Tolerance</option>
              <option value="medium">Medium Risk Tolerance</option>
              <option value="high">High Risk Tolerance</option>
            </select>

            <button
              onClick={() => setShowAdvanced(!showAdvanced)}
              className={`${getButtonClasses('secondary')} flex items-center gap-2`}
              title="Toggle advanced metrics"
              aria-label="Toggle advanced metrics"
            >
              <BarChart3 className="w-4 h-4" />
              Advanced
            </button>
          </div>

          {selectedPortfolio && (
            <>
              {/* Key Metrics Grid */}
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
                {/* Expected Return */}
                <div className={`${getCardClasses()} p-4`}>
                  <div className="flex items-center justify-between mb-2">
                    <p className={`text-sm font-medium ${getTextClasses('secondary')}`}>Expected Return</p>
                    <TrendingDown className="w-4 h-4 text-slate-400 dark:text-slate-500" />
                  </div>
                  <p className={`text-2xl font-bold ${getTextClasses('primary')}`}>
                    {selectedPortfolio.expectedReturn.toFixed(2)}%
                  </p>
                  {selectedPortfolio.trend && (
                    <p
                      className={`text-xs mt-2 ${
                        selectedPortfolio.trend.returnChange >= 0
                          ? 'text-green-600 dark:text-green-400'
                          : 'text-red-600 dark:text-red-400'
                      }`}
                    >
                      {selectedPortfolio.trend.returnChange >= 0 ? '+' : ''}
                      {selectedPortfolio.trend.returnChange.toFixed(2)}% from last period
                    </p>
                  )}
                </div>

                {/* Volatility */}
                <div className={`${getCardClasses()} p-4`}>
                  <div className="flex items-center justify-between mb-2">
                    <p className={`text-sm font-medium ${getTextClasses('secondary')}`}>Volatility (Annual)</p>
                    <Activity className="w-4 h-4 text-slate-400 dark:text-slate-500" />
                  </div>
                  <p className={`text-2xl font-bold ${getTextClasses('primary')}`}>
                    {selectedPortfolio.volatility.toFixed(2)}%
                  </p>
                  {selectedPortfolio.trend && (
                    <p
                      className={`text-xs mt-2 ${
                        selectedPortfolio.trend.volatilityChange <= 0
                          ? 'text-green-600 dark:text-green-400'
                          : 'text-red-600 dark:text-red-400'
                      }`}
                    >
                      {selectedPortfolio.trend.volatilityChange >= 0 ? '+' : ''}
                      {selectedPortfolio.trend.volatilityChange.toFixed(2)}% from last period
                    </p>
                  )}
                </div>

                {/* Sharpe Ratio */}
                <div className={`${getCardClasses()} p-4`}>
                  <p className={`text-sm font-medium ${getTextClasses('secondary')} mb-2`}>Sharpe Ratio</p>
                  <p className={`text-2xl font-bold ${getTextClasses('primary')}`}>
                    {selectedPortfolio.sharpeRatio.toFixed(2)}
                  </p>
                  <p className={`text-xs ${getTextClasses('muted')} mt-2`}>Risk-adjusted returns</p>
                </div>

                {/* Max Drawdown */}
                <div className={`${getCardClasses()} p-4`}>
                  <div className="flex items-center justify-between mb-2">
                    <p className={`text-sm font-medium ${getTextClasses('secondary')}`}>Max Drawdown</p>
                    <AlertTriangle className="w-4 h-4 text-red-400" />
                  </div>
                  <p className="text-2xl font-bold text-red-600 dark:text-red-400">
                    {selectedPortfolio.maxDrawdown.toFixed(2)}%
                  </p>
                  <p className="text-xs text-gray-500 dark:text-gray-400 mt-2">Peak-to-trough decline</p>
                </div>
              </div>

              {/* Risk Assessment */}
              <div className={`${getCardClasses()} p-6`}>
                <h2 className={`text-xl font-bold ${getTextClasses('primary')} mb-4`}>Risk Assessment</h2>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                  {/* Overall Risk Level */}
                  <div>
                    <p className={`text-sm font-medium ${getTextClasses('secondary')} mb-3`}>Overall Risk Level</p>
                    <div
                      className={`p-4 rounded-lg ${getRiskLevel(selectedPortfolio.volatility, selectedPortfolio.maxDrawdown).color}`}
                    >
                      <p className="font-semibold">
                        {getRiskLevel(selectedPortfolio.volatility, selectedPortfolio.maxDrawdown).level}
                      </p>
                      <p className="text-xs mt-1">Based on volatility and drawdown metrics</p>
                    </div>
                  </div>

                  {/* Beta and Alpha */}
                  <div className="space-y-3">
                    <div>
                      <p className={`text-sm font-medium ${getTextClasses('secondary')}`}>Beta</p>
                      <p className={`text-lg font-semibold ${getTextClasses('primary')}`}>
                        {selectedPortfolio.beta.toFixed(2)}
                        <span className={`text-xs ${getTextClasses('muted')} ml-2`}>
                          {selectedPortfolio.beta > 1 ? '(more volatile than market)' : '(less volatile than market)'}
                        </span>
                      </p>
                    </div>
                    <div>
                      <p className={`text-sm font-medium ${getTextClasses('secondary')}`}>Alpha</p>
                      <p className={`text-lg font-semibold ${getTextClasses('primary')}`}>
                        {selectedPortfolio.alpha.toFixed(2)}%
                        <span className={`text-xs ${getTextClasses('muted')} ml-2`}>
                          {selectedPortfolio.alpha > 0 ? '(outperforming)' : '(underperforming)'}
                        </span>
                      </p>
                    </div>
                  </div>
                </div>
              </div>

              {/* Value at Risk */}
              <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                <div className={`${getCardClasses()} p-6`}>
                  <h3 className={`font-semibold ${getTextClasses('primary')} mb-4`}>Value at Risk (VaR)</h3>
                  <div className="space-y-4">
                    <div>
                      <p className={`text-sm ${getTextClasses('secondary')} mb-2`}>VaR (95% confidence)</p>
                      <div className="bg-blue-50 dark:bg-blue-900/20 p-3 rounded-lg border border-blue-200 dark:border-blue-800">
                        <p className="text-lg font-semibold text-blue-600 dark:text-blue-400">
                          {selectedPortfolio.valueAtRisk.toFixed(2)}%
                        </p>
                        <p className={`text-xs ${getTextClasses('muted')} mt-1`}>
                          Maximum expected loss with 95% probability
                        </p>
                      </div>
                    </div>
                    <div>
                      <p className={`text-sm ${getTextClasses('secondary')} mb-2`}>CVaR (95% confidence)</p>
                      <div className="bg-red-50 dark:bg-red-900/20 p-3 rounded-lg border border-red-200 dark:border-red-800">
                        <p className="text-lg font-semibold text-red-600 dark:text-red-400">
                          {selectedPortfolio.conditionalVaR.toFixed(2)}%
                        </p>
                        <p className={`text-xs ${getTextClasses('muted')} mt-1`}>
                          Expected loss in worst 5% of scenarios
                        </p>
                      </div>
                    </div>
                  </div>
                </div>

                {/* Concentration Risk */}
                <div className={`${getCardClasses()} p-6`}>
                  <h3 className={`font-semibold ${getTextClasses('primary')} mb-4`}>Concentration Risk</h3>
                  <div className="space-y-4">
                    <div>
                      <p className={`text-sm ${getTextClasses('secondary')} mb-2`}>Top 1 Holding</p>
                      <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2">
                        <div
                          className="bg-blue-600 dark:bg-blue-400 h-2 rounded-full transition-all"
                          style={{ width: `${Math.min(selectedPortfolio.concentration.top1 * 2, 100)}%` }}
                        ></div>
                      </div>
                      <p className={`text-sm font-medium ${getTextClasses('primary')} mt-1`}>
                        {selectedPortfolio.concentration.top1.toFixed(1)}%
                      </p>
                    </div>
                    <div>
                      <p className={`text-sm ${getTextClasses('secondary')} mb-2`}>Top 5 Holdings</p>
                      <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2">
                        <div
                          className="bg-amber-600 dark:bg-amber-400 h-2 rounded-full transition-all"
                          style={{ width: `${Math.min(selectedPortfolio.concentration.top5, 100)}%` }}
                        ></div>
                      </div>
                      <p className={`text-sm font-medium ${getTextClasses('primary')} mt-1`}>
                        {selectedPortfolio.concentration.top5.toFixed(1)}%
                      </p>
                    </div>
                    <div>
                      <p className={`text-sm ${getTextClasses('secondary')} mb-2`}>Top 10 Holdings</p>
                      <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2">
                        <div
                          className="bg-green-600 dark:bg-green-400 h-2 rounded-full transition-all"
                          style={{ width: `${Math.min(selectedPortfolio.concentration.top10, 100)}%` }}
                        ></div>
                      </div>
                      <p className={`text-sm font-medium ${getTextClasses('primary')} mt-1`}>
                        {selectedPortfolio.concentration.top10.toFixed(1)}%
                      </p>
                    </div>
                  </div>
                </div>
              </div>

              {/* Risk Factors */}
              {riskFactors.length > 0 && (
                <div className={`${getCardClasses()} p-6`}>
                  <h3 className={`font-semibold ${getTextClasses('primary')} mb-4`}>Risk Factor Analysis</h3>
                  <div className="space-y-3">
                    {riskFactors.map((factor, idx) => (
                      <div key={idx} className="flex items-center justify-between p-3 bg-gray-50 dark:bg-gray-800/50 rounded-lg">
                        <div>
                          <p className={`font-medium ${getTextClasses('primary')}`}>{factor.factorName}</p>
                          <p className={`text-xs ${getTextClasses('muted')} mt-1`}>
                            Exposure: {factor.exposure.toFixed(2)} | Sensitivity: {factor.sensitivity.toFixed(3)}
                          </p>
                        </div>
                        <div className="text-right">
                          <p className={`text-sm font-semibold ${getTextClasses('primary')}`}>
                            {(factor.contribution * 100).toFixed(2)}%
                          </p>
                          <p className={`text-xs ${getTextClasses('muted')}`}>contribution</p>
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              )}

              {/* Advanced Metrics */}
              {showAdvanced && (
                <div className={`${getCardClasses()} p-6 bg-gray-50 dark:bg-gray-800/50`}>
                  <h3 className={`font-semibold ${getTextClasses('primary')} mb-4`}>Advanced Risk Metrics</h3>
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                    <div>
                      <p className={`text-sm ${getTextClasses('secondary')}`}>Sortino Ratio</p>
                      <p className={`text-xl font-semibold ${getTextClasses('primary')}`}>
                        {selectedPortfolio.sortinoRatio.toFixed(2)}
                      </p>
                      <p className={`text-xs ${getTextClasses('muted')} mt-1`}>Downside risk-adjusted returns</p>
                    </div>
                    <div>
                      <p className={`text-sm ${getTextClasses('secondary')}`}>Diversification Ratio</p>
                      <p className={`text-xl font-semibold ${getTextClasses('primary')}`}>
                        {selectedPortfolio.diversificationRatio.toFixed(2)}
                      </p>
                      <p className={`text-xs ${getTextClasses('muted')} mt-1`}>Diversification benefit</p>
                    </div>
                  </div>
                </div>
              )}

              {/* Recommendations */}
              <div className={`${getCardClasses()} p-6 bg-blue-50 dark:bg-blue-900/10`}>
                <h3 className={`font-semibold ${getTextClasses('primary')} mb-3 flex items-center gap-2`}>
                  <AlertCircle className="w-5 h-5 text-blue-600 dark:text-blue-400" />
                  Risk Recommendations
                </h3>
                <ul className={`space-y-2 text-sm ${getTextClasses('secondary')}`}>
                  {selectedPortfolio.volatility > 20 && (
                    <li>• Consider increasing bond allocation to reduce portfolio volatility</li>
                  )}
                  {selectedPortfolio.concentration.top1 > 30 && (
                    <li>• Diversify top holding - concentration risk is elevated</li>
                  )}
                  {selectedPortfolio.sharpeRatio < 1 && (
                    <li>• Risk-adjusted returns are below target - review allocation strategy</li>
                  )}
                  {selectedPortfolio.maxDrawdown < -30 && (
                    <li>• Recent drawdown was significant - consider rebalancing</li>
                  )}
                  {selectedPortfolio.beta > 1.2 && (
                    <li>• Portfolio is more volatile than market - consider defensive positions</li>
                  )}
                  {selectedPortfolio.alpha < 0 && (
                    <li>• Underperforming market - review selection methodology</li>
                  )}
                </ul>
              </div>
            </>
          )}
        </>
      )}
    </div>
  );
};

export default RiskAnalyticsDashboardPage;
