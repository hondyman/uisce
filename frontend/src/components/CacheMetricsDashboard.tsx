import React, { useState, useEffect } from 'react';
import { Activity, TrendingUp, Zap, Database, Clock, BarChart3 } from 'lucide-react';
import { LineChart, Line, AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts';

interface CacheMetrics {
  business_objects: {
    hits: number;
    misses: number;
    hit_rate: number;
    item_count: number;
    memory_bytes: number;
    load_time_ms: number;
  };
  semantic_views: {
    semantic_view_count: number;
    cache_ttl_hours: number;
  };
}

const CacheMetricsDashboard: React.FC = () => {
  const [metrics, setMetrics] = useState<CacheMetrics | null>(null);
  const [historicalData, setHistoricalData] = useState<any[]>([]);

  useEffect(() => {
    fetchMetrics();
    const interval = setInterval(fetchMetrics, 5000); // Refresh every 5 seconds
    return () => clearInterval(interval);
  }, []);

  const fetchMetrics = async () => {
    try {
      const response = await fetch('/api/metadata/unified/metrics');
      const data = await response.json();
      setMetrics(data);
      
      // Add to historical data for charts
      setHistoricalData(prev => [
        ...prev.slice(-20), // Keep last 20 data points
        {
          time: new Date().toLocaleTimeString(),
          hitRate: data.business_objects.hit_rate * 100,
          hits: data.business_objects.hits,
          memory: data.business_objects.memory_bytes / 1024,
        },
      ]);
    } catch (error) {
      console.error('Failed to fetch metrics:', error);
    }
  };

  if (!metrics) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-500"></div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h2 className="text-2xl font-bold text-slate-900 dark:text-white mb-2">
          Cache Performance Dashboard
        </h2>
        <p className="text-slate-600 dark:text-slate-400">
          Real-time monitoring of metadata cache performance
        </p>
      </div>

      {/* Key Metrics */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        <MetricCard
          icon={<Zap className="w-5 h-5" />}
          label="Hit Rate"
          value={`${(metrics.business_objects.hit_rate * 100).toFixed(1)}%`}
          trend={metrics.business_objects.hit_rate > 0.95 ? 'up' : 'down'}
          color="green"
        />
        <MetricCard
          icon={<Activity className="w-5 h-5" />}
          label="Total Hits"
          value={metrics.business_objects.hits.toLocaleString()}
          subtitle={`${metrics.business_objects.misses} misses`}
          color="blue"
        />
        <MetricCard
          icon={<Database className="w-5 h-5" />}
          label="Memory Usage"
          value={`${(metrics.business_objects.memory_bytes / 1024).toFixed(0)} KB`}
          subtitle={`${metrics.business_objects.item_count} items`}
          color="purple"
        />
        <MetricCard
          icon={<Clock className="w-5 h-5" />}
          label="Load Time"
          value={`${metrics.business_objects.load_time_ms}ms`}
          subtitle="Initial warmup"
          color="indigo"
        />
      </div>

      {/* Charts */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Hit Rate Chart */}
        <div className="bg-white dark:bg-slate-800 rounded-2xl border border-slate-200 dark:border-slate-700 p-6">
          <div className="flex items-center justify-between mb-6">
            <h3 className="text-lg font-semibold text-slate-900 dark:text-white">
              Hit Rate Over Time
            </h3>
            <div className="flex items-center space-x-2 text-sm">
              <div className="w-3 h-3 bg-green-500 rounded-full"></div>
              <span className="text-slate-600 dark:text-slate-400">Live</span>
            </div>
          </div>
          <ResponsiveContainer width="100%" height={200}>
            <AreaChart data={historicalData}>
              <defs>
                <linearGradient id="hitRateGradient" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%" stopColor="#10b981" stopOpacity={0.3}/>
                  <stop offset="95%" stopColor="#10b981" stopOpacity={0}/>
                </linearGradient>
              </defs>
              <CartesianGrid strokeDasharray="3 3" stroke="#e2e8f0" />
              <XAxis dataKey="time" stroke="#64748b" fontSize={12} />
              <YAxis stroke="#64748b" fontSize={12} domain={[0, 100]} />
              <Tooltip />
              <Area 
                type="monotone" 
                dataKey="hitRate" 
                stroke="#10b981" 
                fill="url(#hitRateGradient)"
                strokeWidth={2}
              />
            </AreaChart>
          </ResponsiveContainer>
        </div>

        {/* Memory Usage Chart */}
        <div className="bg-white dark:bg-slate-800 rounded-2xl border border-slate-200 dark:border-slate-700 p-6">
          <div className="flex items-center justify-between mb-6">
            <h3 className="text-lg font-semibold text-slate-900 dark:text-white">
              Memory Usage
            </h3>
            <div className="flex items-center space-x-2 text-sm">
              <div className="w-3 h-3 bg-purple-500 rounded-full"></div>
              <span className="text-slate-600 dark:text-slate-400">KB</span>
            </div>
          </div>
          <ResponsiveContainer width="100%" height={200}>
            <LineChart data={historicalData}>
              <CartesianGrid strokeDasharray="3 3" stroke="#e2e8f0" />
              <XAxis dataKey="time" stroke="#64748b" fontSize={12} />
              <YAxis stroke="#64748b" fontSize={12} />
              <Tooltip />
              <Line 
                type="monotone" 
                dataKey="memory" 
                stroke="#a855f7" 
                strokeWidth={2}
                dot={false}
              />
            </LineChart>
          </ResponsiveContainer>
        </div>
      </div>

      {/* Cache Details */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Business Objects Cache */}
        <div className="bg-gradient-to-br from-blue-50 to-indigo-50 dark:from-blue-950/20 dark:to-indigo-950/20 rounded-2xl border border-blue-200 dark:border-blue-800 p-6">
          <div className="flex items-center space-x-3 mb-4">
            <div className="p-3 bg-gradient-to-br from-blue-500 to-indigo-600 rounded-xl">
              <Database className="w-6 h-6 text-white" />
            </div>
            <div>
              <h3 className="text-lg font-semibold text-slate-900 dark:text-white">
                Business Objects Cache
              </h3>
              <p className="text-sm text-slate-600 dark:text-slate-400">In-Memory</p>
            </div>
          </div>
          <div className="space-y-3">
            <CacheDetail label="Items Cached" value={metrics.business_objects.item_count.toString()} />
            <CacheDetail label="Cache Hits" value={metrics.business_objects.hits.toLocaleString()} />
            <CacheDetail label="Cache Misses" value={metrics.business_objects.misses.toString()} />
            <CacheDetail label="Hit Rate" value={`${(metrics.business_objects.hit_rate * 100).toFixed(2)}%`} />
            <CacheDetail label="Memory" value={`${(metrics.business_objects.memory_bytes / 1024).toFixed(2)} KB`} />
          </div>
        </div>

        {/* Semantic Views Cache */}
        <div className="bg-gradient-to-br from-purple-50 to-pink-50 dark:from-purple-950/20 dark:to-pink-950/20 rounded-2xl border border-purple-200 dark:border-purple-800 p-6">
          <div className="flex items-center space-x-3 mb-4">
            <div className="p-3 bg-gradient-to-br from-purple-500 to-pink-600 rounded-xl">
              <BarChart3 className="w-6 h-6 text-white" />
            </div>
            <div>
              <h3 className="text-lg font-semibold text-slate-900 dark:text-white">
                Semantic Views Cache
              </h3>
              <p className="text-sm text-slate-600 dark:text-slate-400">Redis</p>
            </div>
          </div>
          <div className="space-y-3">
            <CacheDetail label="Views Cached" value={metrics.semantic_views.semantic_view_count.toString()} />
            <CacheDetail label="TTL" value={`${metrics.semantic_views.cache_ttl_hours} hours`} />
            <CacheDetail label="Cache Type" value="Distributed" />
            <CacheDetail label="Backend" value="Redis" />
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
  color: 'green' | 'blue' | 'purple' | 'indigo';
}> = ({ icon, label, value, subtitle, trend, color }) => {
  const colorClasses = {
    green: 'from-green-500 to-emerald-600',
    blue: 'from-blue-500 to-cyan-600',
    purple: 'from-purple-500 to-fuchsia-600',
    indigo: 'from-indigo-500 to-blue-600',
  };

  return (
    <div className="bg-white dark:bg-slate-800 rounded-xl p-6 border border-slate-200 dark:border-slate-700 hover:shadow-lg transition-all">
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

// Cache Detail Component
const CacheDetail: React.FC<{ label: string; value: string }> = ({ label, value }) => (
  <div className="flex items-center justify-between py-2 border-b border-slate-200 dark:border-slate-700 last:border-0">
    <span className="text-sm text-slate-600 dark:text-slate-400">{label}</span>
    <span className="text-sm font-semibold text-slate-900 dark:text-white">{value}</span>
  </div>
);

export default CacheMetricsDashboard;
