import React, { useState, useEffect } from 'react';
import {
  Plus,
  Trash2,
  TrendingUp,
  TrendingDown,
  PieChart as PieChartIcon,
  AlertCircle,
  CheckCircle,
  Download,
  RefreshCw,
  X,
} from 'lucide-react';
import { useTenant } from '../contexts/TenantContext';
import { useConfirm } from '../components/ConfirmProvider';
import { devLog } from '../utils/devLogger';
import { getCardClasses, getTextClasses, getButtonClasses, getAlertClasses } from '../utils/darkModeHelpers';

interface Portfolio {
  id: string;
  name: string;
  currency: string;
  totalValue: number;
  holdingsCount: number;
  allocation: {
    symbol: string;
    percentage: number;
    value: number;
  }[];
  metrics: {
    dayChange: number;
    dayChangePercent: number;
    monthReturn: number;
    yearReturn: number;
  };
  lastUpdated: string;
}

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

/**
 * Portfolio Dashboard Page
 * Main view for portfolio management, holdings overview, and performance tracking
 */
export const PortfolioDashboardPage: React.FC = () => {
  const { tenant, datasource } = useTenant();

  const [portfolios, setPortfolios] = useState<Portfolio[]>([]);
  const [selectedPortfolio, setSelectedPortfolio] = useState<Portfolio | null>(null);
  const [holdings, setHoldings] = useState<Holding[]>([]);
  const [loading, setLoading] = useState(false);
  const [refreshing, setRefreshing] = useState(false);

  const [showCreateForm, setShowCreateForm] = useState(false);
  // reserved/unused for now; underscore-prefixed to indicate intentional unused state
  const [_showEditForm, _setShowEditForm] = useState(false);
  const [_editingPortfolio, _setEditingPortfolio] = useState<Portfolio | null>(null);

  const [formData, setFormData] = useState({
    name: '',
    currency: 'USD',
  });

  const confirm = useConfirm();

  const [toast, setToast] = useState<{ type: 'success' | 'error'; message: string } | null>(null);
  const [filterAssetClass, setFilterAssetClass] = useState<string>('ALL');
  const [sortBy, setSortBy] = useState<'value' | 'allocation' | 'gainLoss'>('value');

  // Initialize
  useEffect(() => {
    if (tenant && datasource) {
      loadPortfolios();
      devLog('Portfolio Dashboard initialized', { tenantId: tenant.id, datasourceId: datasource.id });
    }
  }, [tenant, datasource]);

  // Load holdings when portfolio selected
  useEffect(() => {
    if (selectedPortfolio) {
      loadHoldings(selectedPortfolio.id);
    }
  }, [selectedPortfolio]);

  const showToast = (type: 'success' | 'error', message: string) => {
    setToast({ type, message });
    setTimeout(() => setToast(null), 3000);
  };

  const loadPortfolios = async () => {
    setLoading(true);
    try {
      const response = await fetch('/api/portfolios', {
        headers: {
          'X-User-ID': tenant?.id || '',
          'X-Tenant-ID': tenant?.id || '',
          'X-Tenant-Datasource-ID': datasource?.id || '',
        },
      });

      if (!response.ok) throw new Error('Failed to fetch portfolios');
      const data = await response.json();
      setPortfolios(data || []);
      if (data && data.length > 0) {
        setSelectedPortfolio(data[0]);
      }
      devLog('Portfolios loaded', { count: data?.length || 0 });
    } catch (error) {
      console.error('Failed to load portfolios:', error);
      showToast('error', 'Failed to load portfolios');
    } finally {
      setLoading(false);
    }
  };

  const loadHoldings = async (portfolioId: string) => {
    try {
      const response = await fetch(`/api/holdings?portfolio_id=${portfolioId}`, {
        headers: {
          'X-User-ID': tenant?.id || '',
          'X-Tenant-ID': tenant?.id || '',
          'X-Tenant-Datasource-ID': datasource?.id || '',
        },
      });

      if (!response.ok) throw new Error('Failed to fetch holdings');
      const data = await response.json();
      setHoldings(data || []);
      devLog('Holdings loaded', { portfolioId, count: data?.length || 0 });
    } catch (error) {
      console.error('Failed to load holdings:', error);
      showToast('error', 'Failed to load holdings');
    }
  };

  const handleCreatePortfolio = async () => {
    if (!formData.name.trim()) {
      showToast('error', 'Portfolio name is required');
      return;
    }

    setRefreshing(true);
    try {
      const response = await fetch('/api/portfolios', {
        method: 'POST',
        headers: {
          'X-User-ID': tenant?.id || '',
          'X-Tenant-ID': tenant?.id || '',
          'X-Tenant-Datasource-ID': datasource?.id || '',
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          name: formData.name,
          currency: formData.currency,
          holdings: [],
        }),
      });

      if (!response.ok) throw new Error('Failed to create portfolio');
      showToast('success', 'Portfolio created successfully');
      setShowCreateForm(false);
      setFormData({ name: '', currency: 'USD' });
      await loadPortfolios();
    } catch (error) {
      console.error('Failed to create portfolio:', error);
      showToast('error', 'Failed to create portfolio');
    } finally {
      setRefreshing(false);
    }
  };

  const handleDeletePortfolio = async (portfolioId: string) => {
    const confirm = useConfirm();
    if (!(await confirm({ title: 'Delete portfolio', description: 'Are you sure you want to delete this portfolio?' }))) return;

    setRefreshing(true);
    try {
      const response = await fetch(`/api/portfolios/${portfolioId}`, {
        method: 'DELETE',
        headers: {
          'X-User-ID': tenant?.id || '',
          'X-Tenant-ID': tenant?.id || '',
          'X-Tenant-Datasource-ID': datasource?.id || '',
        },
      });

      if (!response.ok) throw new Error('Failed to delete portfolio');
      showToast('success', 'Portfolio deleted successfully');
      await loadPortfolios();
    } catch (error) {
      console.error('Failed to delete portfolio:', error);
      showToast('error', 'Failed to delete portfolio');
    } finally {
      setRefreshing(false);
    }
  };

  const filteredHoldings = holdings.filter(
    (h) => filterAssetClass === 'ALL' || h.assetClass === filterAssetClass
  );

  const sortedHoldings = [...filteredHoldings].sort((a, b) => {
    switch (sortBy) {
      case 'value':
        return b.currentValue - a.currentValue;
      case 'allocation':
        return b.allocation - a.allocation;
      case 'gainLoss':
        return b.gainLoss - a.gainLoss;
      default:
        return 0;
    }
  });

  const uniqueAssetClasses = ['ALL', ...new Set(holdings.map((h) => h.assetClass))];

  if (!tenant || !datasource) {
    return (
      <div className={`${getCardClasses()} p-8 bg-gradient-to-br from-blue-50 to-blue-50/50 dark:from-blue-950/20 dark:to-blue-950/10`}>
        <AlertCircle className="w-6 h-6 text-yellow-600 dark:text-yellow-400 mb-2" />
        <p className={getTextClasses('primary')}>Please select a tenant and datasource to manage portfolios.</p>
      </div>
    );
  }

  return (
    <div className={`${getCardClasses()} space-y-6 p-6`}>
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold text-slate-900 dark:text-text-light">Portfolio Dashboard</h1>
          <p className={`${getTextClasses('secondary')} mt-1`}>Manage portfolios and track holdings performance</p>
        </div>
        <div className="flex items-center gap-3">
          <button
            onClick={async () => {
              await loadPortfolios();
              if (selectedPortfolio) await loadHoldings(selectedPortfolio.id);
            }}
            className={`${getButtonClasses('secondary')} flex items-center gap-2`}
            title="Refresh portfolio data"
            aria-label="Refresh portfolio data"
          >
            <RefreshCw className="w-5 h-5" />
            Refresh
          </button>
          <button
            onClick={() => setShowCreateForm(true)}
            className={`${getButtonClasses('primary')} flex items-center gap-2`}
          >
            <Plus className="w-5 h-5" />
            New Portfolio
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

      {/* Portfolio Selector */}
      {loading ? (
        <div className="text-center py-12">
          <div className="animate-spin inline-block w-8 h-8 border-4 border-slate-300 dark:border-slate-600 border-t-blue-600 rounded-full"></div>
          <p className={`${getTextClasses('secondary')} mt-4`}>Loading portfolios...</p>
        </div>
      ) : portfolios.length === 0 ? (
        <div className="text-center py-12 border-2 border-dashed border-slate-300 dark:border-slate-600 rounded-lg">
          <PieChartIcon className="w-12 h-12 text-slate-400 dark:text-slate-500 mx-auto mb-4" />
          <p className={`${getTextClasses('secondary')} mb-4`}>No portfolios yet</p>
          <button onClick={() => setShowCreateForm(true)} className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg transition-colors">
            Create First Portfolio
          </button>
        </div>
      ) : (
        <>
          {/* Portfolio Cards */}
          <div className="grid gap-4 grid-cols-1 md:grid-cols-2 lg:grid-cols-3">
            {portfolios.map((portfolio) => (
              <div
                key={portfolio.id}
                onClick={() => setSelectedPortfolio(portfolio)}
                className={`p-4 rounded-lg border-2 cursor-pointer transition-all ${
                  selectedPortfolio?.id === portfolio.id
                    ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20'
                    : 'border-slate-200 dark:border-border-dark hover:border-blue-300 dark:hover:border-blue-600'
                }`}
              >
                <div className="flex items-start justify-between mb-3">
                  <div>
                    <h3 className={`font-semibold ${getTextClasses('primary')}`}>{portfolio.name}</h3>
                    <p className={`text-sm ${getTextClasses('secondary')}`}>{portfolio.currency}</p>
                  </div>
                  <button
                    onClick={(e) => {
                      e.stopPropagation();
                      handleDeletePortfolio(portfolio.id);
                    }}
                    className="p-1 hover:bg-red-100 dark:hover:bg-red-900/20 rounded transition-colors"
                    title="Delete portfolio"
                    aria-label="Delete portfolio"
                  >
                    <Trash2 className="w-4 h-4 text-red-600 dark:text-red-400" />
                  </button>
                </div>
                <div className="space-y-2">
                  <div className={`text-2xl font-bold ${getTextClasses('primary')}`}>
                    ${portfolio.totalValue.toLocaleString('en-US', { maximumFractionDigits: 2 })}
                  </div>
                  <div className="flex items-center justify-between text-sm">
                    <span className={getTextClasses('secondary')}>{portfolio.holdingsCount} holdings</span>
                    <span
                      className={`flex items-center gap-1 ${
                        portfolio.metrics.dayChange >= 0 ? 'text-green-600 dark:text-green-400' : 'text-red-600 dark:text-red-400'
                      }`}
                    >
                      {portfolio.metrics.dayChange >= 0 ? <TrendingUp className="w-4 h-4" /> : <TrendingDown className="w-4 h-4" />}
                      {portfolio.metrics.dayChangePercent.toFixed(2)}%
                    </span>
                  </div>
                </div>
              </div>
            ))}
          </div>

          {/* Selected Portfolio Details */}
          {selectedPortfolio && (
            <div className="grid gap-6 grid-cols-1 lg:grid-cols-3">
              {/* Holdings List */}
              <div className="lg:col-span-2 space-y-4">
                <div className="flex items-center justify-between">
                  <h2 className={`text-xl font-bold ${getTextClasses('primary')}`}>Holdings</h2>
                  <div className="flex items-center gap-3">
                    <select
                      value={filterAssetClass}
                      onChange={(e) => setFilterAssetClass(e.target.value)}
                      className="px-3 py-2 border border-slate-300 dark:border-border-dark rounded-lg bg-white dark:bg-surface-dark text-slate-900 dark:text-text-light focus:outline-none focus:ring-2 focus:ring-blue-500"
                      title="Filter by asset class"
                      aria-label="Filter by asset class"
                    >
                      {uniqueAssetClasses.map((ac) => (
                        <option key={ac} value={ac}>
                          {ac === 'ALL' ? 'All Assets' : ac}
                        </option>
                      ))}
                    </select>
                    <select
                      value={sortBy}
                      onChange={(e) => setSortBy(e.target.value as any)}
                      className="px-3 py-2 border border-slate-300 dark:border-border-dark rounded-lg bg-white dark:bg-surface-dark text-slate-900 dark:text-text-light focus:outline-none focus:ring-2 focus:ring-blue-500"
                      title="Sort holdings"
                      aria-label="Sort holdings"
                    >
                      <option value="value">Sort by Value</option>
                      <option value="allocation">Sort by Allocation</option>
                      <option value="gainLoss">Sort by Gain/Loss</option>
                    </select>
                  </div>
                </div>

                <div className="space-y-3">
                  {sortedHoldings.length === 0 ? (
                    <div className="text-center py-8 border border-dashed border-slate-300 dark:border-slate-600 rounded-lg">
                      <p className={getTextClasses('secondary')}>No holdings in this portfolio</p>
                    </div>
                  ) : (
                    sortedHoldings.map((holding) => (
                      <div
                        key={holding.id}
                        className="p-4 border border-slate-200 dark:border-border-dark rounded-lg hover:shadow-md transition-shadow bg-white dark:bg-surface-dark"
                      >
                        <div className="flex items-start justify-between mb-3">
                          <div>
                            <h3 className={`font-semibold ${getTextClasses('primary')}`}>{holding.symbol}</h3>
                            <p className={`text-sm ${getTextClasses('secondary')}`}>{holding.name}</p>
                          </div>
                          <div className="text-right">
                            <div className={`text-lg font-bold ${getTextClasses('primary')}`}>
                              ${holding.currentValue.toLocaleString('en-US', { maximumFractionDigits: 2 })}
                            </div>
                            <span
                              className={`text-sm font-medium ${
                                holding.gainLoss >= 0 ? 'text-green-600 dark:text-green-400' : 'text-red-600 dark:text-red-400'
                              }`}
                            >
                              {holding.gainLoss >= 0 ? '+' : ''}
                              {holding.gainLoss.toFixed(2)} ({holding.gainLossPercent.toFixed(2)}%)
                            </span>
                          </div>
                        </div>
                        <div className="grid grid-cols-4 gap-4 text-sm">
                          <div>
                            <p className={getTextClasses('secondary')}>Quantity</p>
                            <p className={`font-medium ${getTextClasses('primary')}`}>{holding.quantity}</p>
                          </div>
                          <div>
                            <p className={getTextClasses('secondary')}>Avg Cost</p>
                            <p className={`font-medium ${getTextClasses('primary')}`}>${holding.averageCost.toFixed(2)}</p>
                          </div>
                          <div>
                            <p className={getTextClasses('secondary')}>Current Price</p>
                            <p className={`font-medium ${getTextClasses('primary')}`}>${holding.currentPrice.toFixed(2)}</p>
                          </div>
                          <div>
                            <p className={getTextClasses('secondary')}>Allocation</p>
                            <p className={`font-medium ${getTextClasses('primary')}`}>{holding.allocation.toFixed(2)}%</p>
                          </div>
                        </div>
                        {holding.beta && (
                          <div className="grid grid-cols-3 gap-4 text-sm mt-3 pt-3 border-t border-slate-200 dark:border-border-dark">
                            <div>
                              <p className={getTextClasses('secondary')}>Beta</p>
                              <p className={`font-medium ${getTextClasses('primary')}`}>{holding.beta.toFixed(2)}</p>
                            </div>
                            <div>
                              <p className={getTextClasses('secondary')}>Volatility</p>
                              <p className={`font-medium ${getTextClasses('primary')}`}>{(holding.volatility || 0).toFixed(2)}%</p>
                            </div>
                            <div>
                              <p className={getTextClasses('secondary')}>Sector</p>
                              <p className={`font-medium ${getTextClasses('primary')}`}>{holding.sector}</p>
                            </div>
                          </div>
                        )}
                      </div>
                    ))
                  )}
                </div>
              </div>

              {/* Portfolio Summary */}
              <div className="space-y-4">
                <div className="bg-gradient-to-br from-blue-50 to-blue-50/50 dark:from-blue-900/20 dark:to-blue-900/10 p-4 rounded-lg border border-blue-200 dark:border-blue-800">
                  <h3 className={`font-semibold ${getTextClasses('primary')} mb-3`}>Portfolio Summary</h3>
                  <div className="space-y-2 text-sm">
                    <div className="flex justify-between">
                      <span className={getTextClasses('secondary')}>Total Value</span>
                      <span className={`font-medium ${getTextClasses('primary')}`}>
                        ${selectedPortfolio.totalValue.toLocaleString('en-US', { maximumFractionDigits: 2 })}
                      </span>
                    </div>
                    <div className="flex justify-between">
                      <span className={getTextClasses('secondary')}>Holdings</span>
                      <span className={`font-medium ${getTextClasses('primary')}`}>{selectedPortfolio.holdingsCount}</span>
                    </div>
                    <div className="flex justify-between pt-2 border-t border-blue-200 dark:border-blue-800">
                      <span className={getTextClasses('secondary')}>Day Change</span>
                      <span
                        className={`font-medium ${
                          selectedPortfolio.metrics.dayChange >= 0
                            ? 'text-green-600 dark:text-green-400'
                            : 'text-red-600 dark:text-red-400'
                        }`}
                      >
                        {selectedPortfolio.metrics.dayChange >= 0 ? '+' : ''}
                        {selectedPortfolio.metrics.dayChangePercent.toFixed(2)}%
                      </span>
                    </div>
                    <div className="flex justify-between">
                      <span className={getTextClasses('secondary')}>Month Return</span>
                      <span
                        className={`font-medium ${
                          selectedPortfolio.metrics.monthReturn >= 0
                            ? 'text-green-600 dark:text-green-400'
                            : 'text-red-600 dark:text-red-400'
                        }`}
                      >
                        {selectedPortfolio.metrics.monthReturn >= 0 ? '+' : ''}
                        {selectedPortfolio.metrics.monthReturn.toFixed(2)}%
                      </span>
                    </div>
                    <div className="flex justify-between">
                      <span className={getTextClasses('secondary')}>Year Return</span>
                      <span
                        className={`font-medium ${
                          selectedPortfolio.metrics.yearReturn >= 0
                            ? 'text-green-600 dark:text-green-400'
                            : 'text-red-600 dark:text-red-400'
                        }`}
                      >
                        {selectedPortfolio.metrics.yearReturn >= 0 ? '+' : ''}
                        {selectedPortfolio.metrics.yearReturn.toFixed(2)}%
                      </span>
                    </div>
                  </div>
                </div>

                                <div className="bg-gradient-to-br from-purple-50 to-purple-50/50 dark:from-purple-900/20 dark:to-purple-900/10 p-4 rounded-lg border border-purple-200 dark:border-purple-800">
                  <h3 className={`font-semibold ${getTextClasses('primary')} mb-3`}>Top Holdings</h3>
                  <div className="space-y-2">
                    {sortedHoldings.slice(0, 5).map((holding) => (
                      <div key={holding.id} className="flex items-center justify-between text-sm">
                        <span className={getTextClasses('secondary')}>{holding.symbol}</span>
                        <span className={`font-medium ${getTextClasses('primary')}`}>{holding.allocation.toFixed(1)}%</span>
                      </div>
                    ))}
                  </div>
                </div>

                <button
                  className={`${getButtonClasses('secondary')} w-full flex items-center justify-center gap-2`}
                  title="Export portfolio data"
                >
                  <Download className="w-4 h-4" />
                  Export
                </button>
              </div>
            </div>
          )}
        </>
      )}

      {/* Create Portfolio Modal */}
      {showCreateForm && (
        <div className="fixed inset-0 bg-black/50 dark:bg-black/70 flex items-center justify-center z-50">
          <div className={`${getCardClasses()} shadow-xl max-w-md w-full mx-4`}>
            <div className="flex items-center justify-between p-6 border-b border-slate-200 dark:border-border-dark">
              <h2 className={`text-2xl font-bold ${getTextClasses('primary')}`}>Create Portfolio</h2>
              <button
                onClick={() => setShowCreateForm(false)}
                className="p-2 hover:bg-slate-100 dark:hover:bg-slate-800 rounded-lg transition-colors"
                title="Close form"
                aria-label="Close form"
              >
                <X className="w-6 h-6 text-slate-600 dark:text-slate-400" />
              </button>
            </div>

            <div className="p-6 space-y-4">
              <div>
                <label className={`block text-sm font-medium ${getTextClasses('primary')} mb-1`}>
                  Portfolio Name *
                </label>
                <input
                  type="text"
                  value={formData.name}
                  onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                  className="w-full px-3 py-2 border border-slate-300 dark:border-border-dark rounded-lg bg-white dark:bg-surface-dark text-slate-900 dark:text-text-light focus:outline-none focus:ring-2 focus:ring-blue-500"
                  placeholder="e.g., Conservative Growth Portfolio"
                />
              </div>

              <div>
                <label className={`block text-sm font-medium ${getTextClasses('primary')} mb-1`}>Currency</label>
                <select
                  value={formData.currency}
                  onChange={(e) => setFormData({ ...formData, currency: e.target.value })}
                  className="w-full px-3 py-2 border border-slate-300 dark:border-border-dark rounded-lg bg-white dark:bg-surface-dark text-slate-900 dark:text-text-light focus:outline-none focus:ring-2 focus:ring-blue-500"
                  title="Select currency"
                  aria-label="Select currency"
                >
                  <option value="USD">USD</option>
                  <option value="EUR">EUR</option>
                  <option value="GBP">GBP</option>
                  <option value="JPY">JPY</option>
                </select>
              </div>
            </div>

            <div className="flex items-center justify-end gap-3 p-6 border-t border-slate-200 dark:border-border-dark">
              <button
                onClick={() => setShowCreateForm(false)}
                className={`${getButtonClasses('secondary')}`}
              >
                Cancel
              </button>
              <button
                onClick={handleCreatePortfolio}
                disabled={refreshing}
                className={`${getButtonClasses('primary')} flex items-center gap-2 disabled:opacity-50 disabled:cursor-not-allowed`}
              >
                <Plus className="w-5 h-5" />
                {refreshing ? 'Creating...' : 'Create'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default PortfolioDashboardPage;
