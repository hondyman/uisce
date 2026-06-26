import React, { useState, useEffect } from 'react';
import { Activity, TrendingUp, Zap, Clock, BarChart3, AlertCircle } from 'lucide-react';
import { LineChart, Line, BarChart, Bar, PieChart, Pie, Cell, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Legend } from 'recharts';

interface QueryHistory {
  id: string;
  cube_name: string;
  execution_time_ms: number;
  result_rows: number;
  cache_hit: boolean;
  created_at: string;
}

interface PerformanceMetrics {
  total_queries: number;
  cached_queries: number;
  cache_hit_rate: number;
  avg_execution_time: number;
  total_execution_time: number;
}

const SemanticAnalyticsDashboard: React.FC = () => {
  const [metrics, setMetrics] = useState<PerformanceMetrics | null>(null);
  const [history, setHistory] = useState<QueryHistory[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchData();
    const interval = setInterval(fetchData, 10000); // Refresh every 10 seconds
    return () => clearInterval(interval);
  }, []);

  const fetchData = async () => {
    try {
      const [metricsRes, historyRes] = await Promise.all([
        fetch('/api/semantic/analytics/performance', {
          headers: { 'X-Tenant-ID': 'default-tenant' },
        }),
        fetch('/api/semantic/analytics/history', {
          headers: { 'X-Tenant-ID': 'default-tenant' },
        }),
      ]);

      const metricsData = await metricsRes.json();
      const historyData = await historyRes.json();

      setMetrics(metricsData);
      setHistory(historyData.history || []);
    } catch (error) {
      console.error('Failed to fetch analytics:', error);
    } finally {
      setLoading(false);
    }
  };

  // Prepare chart data
  const executionTimeData = history.slice(0, 20).reverse().map((h, idx) => ({
    query: `Q${idx + 1}`,
    time: h.execution_time_ms,
    cached: h.cache_hit,
  }));

  const cacheData = [
    { name: 'Cache Hits', value: metrics?.cached_queries || 0, color: '#10b981' },
    { name: 'Cache Misses', value: (metrics?.total_queries || 0) - (metrics?.cached_queries || 0), color: '#ef4444' },
  ];

  const cubeUsageData = history.reduce((acc, h) => {
    const cube = h.cube_name || 'Unknown';
    acc[cube] = (acc[cube] || 0) + 1;
    return acc;
  }, {} as Record<string, number>);

  const cubeUsageChartData = Object.entries(cubeUsageData).map(([name, count]) => ({
    name,
    queries: count,
  }));

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
            <div className="p-3 bg-gradient-to-br from-purple-500 to-pink-600 rounded-xl">
              <BarChart3 className="w-6 h-6 text-white" />
            </div>
            <div>
              <h1 className="text-2xl font-bold bg-gradient-to-r from-purple-600 to-pink-600 bg-clip-text text-transparent">
                Analytics Dashboard
              </h1>
              <p className="text-sm text-slate-600 dark:text-slate-400">
                Query performance and usage analytics
              </p>
            </div>
          </div>
        </div>
      </div>

      <div className="max-w-7xl mx-auto px-6 py-8 space-y-6">
        {/* Key Metrics */}
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
          <MetricCard
            icon={<Activity className="w-5 h-5" />}
            label="Total Queries"
            value={metrics?.total_queries.toLocaleString() || '0'}
            color="blue"
          />
          <MetricCard
            icon={<Zap className="w-5 h-5" />}
            label="Cache Hit Rate"
            value={`${((metrics?.cache_hit_rate || 0) * 100).toFixed(1)}%`}
            trend={metrics && metrics.cache_hit_rate > 0.8 ? 'up' : 'down'}
            color="green"
          />
          <MetricCard
            icon={<Clock className="w-5 h-5" />}
            label="Avg Execution Time"
            value={`${(metrics?.avg_execution_time || 0).toFixed(0)}ms`}
            color="purple"
          />
          <MetricCard
            icon={<TrendingUp className="w-5 h-5" />}
            label="Cached Queries"
            value={metrics?.cached_queries.toLocaleString() || '0'}
            subtitle={`${metrics?.total_queries || 0} total`}
            color="indigo"
          />
        </div>

        {/* Charts Row 1 */}
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          {/* Execution Time Chart */}
          <div className="bg-white/80 dark:bg-slate-900/80 backdrop-blur-xl rounded-2xl border border-slate-200 dark:border-slate-700 p-6">
            <h3 className="text-lg font-semibold text-slate-900 dark:text-white mb-4">
              Query Execution Time
            </h3>
            <ResponsiveContainer width="100%" height={250}>
              <BarChart data={executionTimeData}>
                <CartesianGrid strokeDasharray="3 3" stroke="#e2e8f0" />
                <XAxis dataKey="query" stroke="#64748b" fontSize={12} />
                <YAxis stroke="#64748b" fontSize={12} />
                <Tooltip />
                <Bar dataKey="time" fill="#8b5cf6" radius={[8, 8, 0, 0]} />
              </BarChart>
            </ResponsiveContainer>
          </div>

          {/* Cache Hit Rate */}
          <div className="bg-white/80 dark:bg-slate-900/80 backdrop-blur-xl rounded-2xl border border-slate-200 dark:border-slate-700 p-6">
            <h3 className="text-lg font-semibold text-slate-900 dark:text-white mb-4">
              Cache Performance
            </h3>
            <ResponsiveContainer width="100%" height={250}>
              <PieChart>
                <Pie
                  data={cacheData}
                  cx="50%"
                  cy="50%"
                  innerRadius={60}
                  outerRadius={90}
                  paddingAngle={5}
                  dataKey="value"
                >
                  {cacheData.map((entry, index) => (
                    <Cell key={`cell-${index}`} fill={entry.color} />
                  ))}
                </Pie>
                <Tooltip />
                <Legend />
              </PieChart>
            </ResponsiveContainer>
          </div>
        </div>

        {/* Charts Row 2 */}
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          {/* Cube Usage */}
          <div className="bg-white/80 dark:bg-slate-900/80 backdrop-blur-xl rounded-2xl border border-slate-200 dark:border-slate-700 p-6">
            <h3 className="text-lg font-semibold text-slate-900 dark:text-white mb-4">
              Cube Usage
            </h3>
            <ResponsiveContainer width="100%" height={250}>
              <BarChart data={cubeUsageChartData} layout="vertical">
                <CartesianGrid strokeDasharray="3 3" stroke="#e2e8f0" />
                <XAxis type="number" stroke="#64748b" fontSize={12} />
                <YAxis dataKey="name" type="category" stroke="#64748b" fontSize={12} width={100} />
                <Tooltip />
                <Bar dataKey="queries" fill="#3b82f6" radius={[0, 8, 8, 0]} />
              </BarChart>
            </ResponsiveContainer>
          </div>

          {/* Recent Queries */}
          <div className="bg-white/80 dark:bg-slate-900/80 backdrop-blur-xl rounded-2xl border border-slate-200 dark:border-slate-700 p-6">
            <h3 className="text-lg font-semibold text-slate-900 dark:text-white mb-4">
              Recent Queries
            </h3>
            <div className="space-y-2 max-h-[250px] overflow-y-auto">
              {history.slice(0, 10).map((query) => (
                <div
                  key={query.id}
                  className="flex items-center justify-between p-3 bg-slate-50 dark:bg-slate-800 rounded-lg"
                >
                  <div className="flex-1">
                    <div className="font-medium text-slate-900 dark:text-white text-sm">
                      {query.cube_name || 'Unknown Cube'}
                    </div>
                    <div className="text-xs text-slate-500 dark:text-slate-500">
                      {query.result_rows} rows • {new Date(query.created_at).toLocaleTimeString()}
                    </div>
                  </div>
                  <div className="flex items-center space-x-2">
                    <span className="text-sm font-semibold text-slate-600 dark:text-slate-400">
                      {query.execution_time_ms}ms
                    </span>
                    {query.cache_hit && (
                      <span className="px-2 py-1 bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-400 rounded-full text-xs font-semibold">
                        CACHED
                      </span>
                    )}
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>

        {/* Performance Insights */}
        <div className="bg-gradient-to-br from-blue-50 to-indigo-50 dark:from-blue-950/20 dark:to-indigo-950/20 rounded-2xl border border-blue-200 dark:border-blue-800 p-6">
          <div className="flex items-start space-x-4">
            <div className="p-3 bg-blue-500 rounded-xl">
              <AlertCircle className="w-6 h-6 text-white" />
            </div>
            <div className="flex-1">
              <h3 className="text-lg font-semibold text-slate-900 dark:text-white mb-2">
                Performance Insights
              </h3>
              <div className="space-y-2 text-sm text-slate-600 dark:text-slate-400">
                {metrics && metrics.cache_hit_rate > 0.8 ? (
                  <p>✅ Excellent cache performance! {((metrics.cache_hit_rate) * 100).toFixed(1)}% of queries are served from cache.</p>
                ) : (
                  <p>⚠️ Cache hit rate is below optimal. Consider adding pre-aggregations for frequently queried data.</p>
                )}
                {metrics && metrics.avg_execution_time < 100 ? (
                  <p>✅ Fast query execution with average time of {metrics.avg_execution_time.toFixed(0)}ms.</p>
                ) : (
                  <p>⚠️ Average query time is {metrics?.avg_execution_time.toFixed(0)}ms. Consider optimizing slow queries.</p>
                )}
              </div>
            </div>
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
  subtitle?: string;
  trend?: 'up' | 'down';
  color: 'blue' | 'green' | 'purple' | 'indigo';
}> = ({ icon, label, value, subtitle, trend, color }) => {
  const colorClasses = {
    blue: 'from-blue-500 to-cyan-600',
    green: 'from-green-500 to-emerald-600',
    purple: 'from-purple-500 to-fuchsia-600',
    indigo: 'from-indigo-500 to-blue-600',
  };

  return (
    <div className="bg-white/80 dark:bg-slate-900/80 backdrop-blur-xl rounded-xl p-6 border border-slate-200 dark:border-slate-700 hover:shadow-lg transition-all">
      <div className={`inline-flex p-3 rounded-lg bg-gradient-to-br ${colorClasses[color]} mb-4`}>
        <div className="text-white">{icon}</div>
      </div>
      <div className="flex items-baseline space-x-2 mb-1">
        <div className="text-3xl font-bold text-slate-900 dark:text-white">{value}</div>
        {trend && (
          <TrendingUp className={`w-5 h-5 ${trend === 'up' ? 'text-green-500' : 'text-red-500 rotate-180'}`} />
        )}
      </div>
      <div className="text-sm font-medium text-slate-600 dark:text-slate-400 mb-1">{label}</div>
      {subtitle && <div className="text-xs text-slate-500 dark:text-slate-500">{subtitle}</div>}
    </div>
  );
};

export default SemanticAnalyticsDashboard;
