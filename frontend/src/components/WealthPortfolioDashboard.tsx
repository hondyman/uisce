import React, { useState, useEffect } from 'react';
import { TrendingUp, DollarSign, Users, PieChart, Target, ArrowUpRight, ArrowDownRight } from 'lucide-react';
import { AreaChart, Area, BarChart, Bar, PieChart as RePieChart, Pie, Cell, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Legend } from 'recharts';

interface PortfolioMetrics {
  total_aum: number;
  total_clients: number;
  avg_portfolio_value: number;
  ytd_return: number;
}

const WealthPortfolioDashboard: React.FC = () => {
  const [metrics, setMetrics] = useState<PortfolioMetrics | null>(null);
  const [assetAllocation, setAssetAllocation] = useState<any[]>([]);
  const [performanceData, setPerformanceData] = useState<any[]>([]);
  const [topPortfolios, setTopPortfolios] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchDashboardData();
  }, []);

  const fetchDashboardData = async () => {
    try {
      // Fetch key metrics
      const metricsQuery = {
        measures: [
          'clients.total_aum',
          'clients.client_count',
          'portfolios.avg_portfolio_value',
          'performance.avg_ytd_return',
        ],
        limit: 1,
      };

      // Fetch asset allocation
      const allocationQuery = {
        measures: ['asset_allocation.allocation_value'],
        dimensions: ['asset_allocation.asset_class'],
        limit: 10,
      };

      // Fetch performance over time
      const performanceQuery = {
        measures: ['performance.avg_ytd_return', 'performance.avg_benchmark_return'],
        timeDimensions: [{
          dimension: 'performance.date',
          granularity: 'month',
        }],
        limit: 12,
      };

      // Fetch top portfolios
      const topPortfoliosQuery = {
        measures: ['portfolios.total_value', 'portfolios.total_gain_loss'],
        dimensions: ['portfolios.portfolio_name'],
        order: { 'portfolios.total_value': 'desc' },
        limit: 5,
      };

      const [metricsRes, allocationRes, performanceRes, topPortfoliosRes] = await Promise.all([
        fetch('/api/semantic/query', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json', 'X-Tenant-ID': 'default-tenant' },
          body: JSON.stringify(metricsQuery),
        }),
        fetch('/api/semantic/query', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json', 'X-Tenant-ID': 'default-tenant' },
          body: JSON.stringify(allocationQuery),
        }),
        fetch('/api/semantic/query', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json', 'X-Tenant-ID': 'default-tenant' },
          body: JSON.stringify(performanceQuery),
        }),
        fetch('/api/semantic/query', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json', 'X-Tenant-ID': 'default-tenant' },
          body: JSON.stringify(topPortfoliosQuery),
        }),
      ]);

      const metricsData = await metricsRes.json();
      const allocationData = await allocationRes.json();
      const performanceDataRes = await performanceRes.json();
      const topPortfoliosData = await topPortfoliosRes.json();

      setMetrics(metricsData.data[0] || {});
      setAssetAllocation(allocationData.data || []);
      setPerformanceData(performanceDataRes.data || []);
      setTopPortfolios(topPortfoliosData.data || []);
    } catch (error) {
      console.error('Failed to fetch dashboard data:', error);
    } finally {
      setLoading(false);
    }
  };

  const COLORS = ['#3b82f6', '#10b981', '#f59e0b', '#ef4444', '#8b5cf6', '#ec4899'];

  if (loading) {
    return (
      <div className="flex items-center justify-center h-screen bg-gradient-to-br from-slate-50 via-blue-50 to-indigo-50 dark:from-slate-900 dark:via-slate-800 dark:to-indigo-950">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-500"></div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-50 via-blue-50 to-indigo-50 dark:from-slate-900 dark:via-slate-800 dark:to-indigo-950">
      {/* Header */}
      <div className="bg-white/80 dark:bg-slate-900/80 backdrop-blur-xl border-b border-slate-200 dark:border-slate-700 px-6 py-4">
        <div className="max-w-7xl mx-auto">
          <div className="flex items-center space-x-4">
            <div className="p-3 bg-gradient-to-br from-green-500 to-emerald-600 rounded-xl">
              <TrendingUp className="w-6 h-6 text-white" />
            </div>
            <div>
              <h1 className="text-2xl font-bold bg-gradient-to-r from-green-600 to-emerald-600 bg-clip-text text-transparent">
                Portfolio Analytics
              </h1>
              <p className="text-sm text-slate-600 dark:text-slate-400">
                Wealth management insights and performance
              </p>
            </div>
          </div>
        </div>
      </div>

      <div className="max-w-7xl mx-auto px-6 py-8 space-y-6">
        {/* Key Metrics */}
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
          <MetricCard
            icon={<DollarSign className="w-5 h-5" />}
            label="Total AUM"
            value={`$${((metrics?.total_aum || 0) / 1000000).toFixed(1)}M`}
            change={8.2}
            color="green"
          />
          <MetricCard
            icon={<Users className="w-5 h-5" />}
            label="Total Clients"
            value={(metrics?.total_clients || 0).toLocaleString()}
            change={5.3}
            color="blue"
          />
          <MetricCard
            icon={<PieChart className="w-5 h-5" />}
            label="Avg Portfolio Value"
            value={`$${((metrics?.avg_portfolio_value || 0) / 1000).toFixed(0)}K`}
            change={3.1}
            color="purple"
          />
          <MetricCard
            icon={<Target className="w-5 h-5" />}
            label="YTD Return"
            value={`${((metrics?.ytd_return || 0) * 100).toFixed(2)}%`}
            change={(metrics?.ytd_return || 0) * 100}
            color="indigo"
          />
        </div>

        {/* Charts Row 1 */}
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          {/* Performance Chart */}
          <div className="bg-white/80 dark:bg-slate-900/80 backdrop-blur-xl rounded-2xl border border-slate-200 dark:border-slate-700 p-6">
            <h3 className="text-lg font-semibold text-slate-900 dark:text-white mb-4">
              Performance vs Benchmark
            </h3>
            <ResponsiveContainer width="100%" height={300}>
              <AreaChart data={performanceData}>
                <defs>
                  <linearGradient id="portfolioGradient" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="#10b981" stopOpacity={0.3}/>
                    <stop offset="95%" stopColor="#10b981" stopOpacity={0}/>
                  </linearGradient>
                  <linearGradient id="benchmarkGradient" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="#3b82f6" stopOpacity={0.3}/>
                    <stop offset="95%" stopColor="#3b82f6" stopOpacity={0}/>
                  </linearGradient>
                </defs>
                <CartesianGrid strokeDasharray="3 3" stroke="#e2e8f0" />
                <XAxis dataKey="performance.date" stroke="#64748b" fontSize={12} />
                <YAxis stroke="#64748b" fontSize={12} />
                <Tooltip />
                <Legend />
                <Area 
                  type="monotone" 
                  dataKey="performance.avg_ytd_return" 
                  stroke="#10b981" 
                  fill="url(#portfolioGradient)"
                  name="Portfolio"
                  strokeWidth={2}
                />
                <Area 
                  type="monotone" 
                  dataKey="performance.avg_benchmark_return" 
                  stroke="#3b82f6" 
                  fill="url(#benchmarkGradient)"
                  name="Benchmark"
                  strokeWidth={2}
                />
              </AreaChart>
            </ResponsiveContainer>
          </div>

          {/* Asset Allocation */}
          <div className="bg-white/80 dark:bg-slate-900/80 backdrop-blur-xl rounded-2xl border border-slate-200 dark:border-slate-700 p-6">
            <h3 className="text-lg font-semibold text-slate-900 dark:text-white mb-4">
              Asset Allocation
            </h3>
            <ResponsiveContainer width="100%" height={300}>
              <RePieChart>
                <Pie
                  data={assetAllocation}
                  cx="50%"
                  cy="50%"
                  labelLine={false}
                  label={(entry) => entry['asset_allocation.asset_class']}
                  outerRadius={100}
                  dataKey="asset_allocation.allocation_value"
                >
                  {assetAllocation.map((entry, index) => (
                    <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                  ))}
                </Pie>
                <Tooltip formatter={(value: any) => `$${(value / 1000000).toFixed(2)}M`} />
              </RePieChart>
            </ResponsiveContainer>
          </div>
        </div>

        {/* Top Portfolios */}
        <div className="bg-white/80 dark:bg-slate-900/80 backdrop-blur-xl rounded-2xl border border-slate-200 dark:border-slate-700 p-6">
          <h3 className="text-lg font-semibold text-slate-900 dark:text-white mb-4">
            Top Portfolios by Value
          </h3>
          <div className="space-y-3">
            {topPortfolios.map((portfolio, idx) => (
              <div
                key={idx}
                className="flex items-center justify-between p-4 bg-slate-50 dark:bg-slate-800 rounded-xl hover:shadow-md transition-all"
              >
                <div className="flex items-center space-x-4">
                  <div className="w-10 h-10 bg-gradient-to-br from-blue-500 to-indigo-600 rounded-lg flex items-center justify-center text-white font-bold">
                    {idx + 1}
                  </div>
                  <div>
                    <div className="font-semibold text-slate-900 dark:text-white">
                      {portfolio['portfolios.portfolio_name']}
                    </div>
                    <div className="text-sm text-slate-500 dark:text-slate-500">
                      Gain/Loss: ${(portfolio['portfolios.total_gain_loss'] / 1000).toFixed(1)}K
                    </div>
                  </div>
                </div>
                <div className="text-right">
                  <div className="text-xl font-bold text-slate-900 dark:text-white">
                    ${(portfolio['portfolios.total_value'] / 1000000).toFixed(2)}M
                  </div>
                  <div className={`text-sm font-semibold ${
                    portfolio['portfolios.total_gain_loss'] >= 0 ? 'text-green-600' : 'text-red-600'
                  }`}>
                    {portfolio['portfolios.total_gain_loss'] >= 0 ? '+' : ''}
                    {((portfolio['portfolios.total_gain_loss'] / portfolio['portfolios.total_value']) * 100).toFixed(2)}%
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
};

// Metric Card Component
const MetricCard: React.FC<{
  icon: React.ReactNode;
  label: string;
  value: string;
  change: number;
  color: 'green' | 'blue' | 'purple' | 'indigo';
}> = ({ icon, label, value, change, color }) => {
  const colorClasses = {
    green: 'from-green-500 to-emerald-600',
    blue: 'from-blue-500 to-cyan-600',
    purple: 'from-purple-500 to-fuchsia-600',
    indigo: 'from-indigo-500 to-blue-600',
  };

  const isPositive = change >= 0;

  return (
    <div className="bg-white/80 dark:bg-slate-900/80 backdrop-blur-xl rounded-xl p-6 border border-slate-200 dark:border-slate-700 hover:shadow-lg transition-all">
      <div className={`inline-flex p-3 rounded-lg bg-gradient-to-br ${colorClasses[color]} mb-4`}>
        <div className="text-white">{icon}</div>
      </div>
      <div className="text-3xl font-bold text-slate-900 dark:text-white mb-1">{value}</div>
      <div className="flex items-center justify-between">
        <div className="text-sm font-medium text-slate-600 dark:text-slate-400">{label}</div>
        <div className={`flex items-center space-x-1 text-sm font-semibold ${
          isPositive ? 'text-green-600' : 'text-red-600'
        }`}>
          {isPositive ? <ArrowUpRight className="w-4 h-4" /> : <ArrowDownRight className="w-4 h-4" />}
          <span>{Math.abs(change).toFixed(1)}%</span>
        </div>
      </div>
    </div>
  );
};

export default WealthPortfolioDashboard;
