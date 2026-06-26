import React, { useState, useEffect } from 'react';
import { useTenant } from '../../../context/TenantContext';

interface QueryAnalytic {
  id: string;
  tenant_id: string;
  query_hash: string;
  cube_name: string;
  measures: string[];
  dimensions: string[];
  duration_ms: number;
  cache_hit: boolean;
  pre_agg_used: boolean;
  rows_returned: number;
  executed_at: string;
}

interface AnalyticsSummary {
  totalQueries: number;
  avgDuration: number;
  p50Duration: number;
  p95Duration: number;
  p99Duration: number;
  cacheHitRate: number;
  preAggRate: number;
  avgRowsReturned: number;
}

export function CubeQueryAnalyticsPage() {
  const { tenant, datasource } = useTenant();
  const [queries, setQueries] = useState<QueryAnalytic[]>([]);
  const [summary, setSummary] = useState<AnalyticsSummary | null>(null);
  const [loading, setLoading] = useState(true);
  const [dateRange, setDateRange] = useState<'1h' | '24h' | '7d' | '30d'>('24h');
  const [cubeFilter, setCubeFilter] = useState<string>('');
  const [availableCubes, setAvailableCubes] = useState<string[]>([]);

  useEffect(() => {
    if (!tenant?.id || !datasource?.id) return;
    loadAnalytics();
  }, [tenant?.id, datasource?.id, dateRange, cubeFilter]);

  const loadAnalytics = async () => {
    setLoading(true);
    try {
      const now = new Date();
      const startDate = new Date();
      switch (dateRange) {
        case '1h':
          startDate.setHours(now.getHours() - 1);
          break;
        case '24h':
          startDate.setDate(now.getDate() - 1);
          break;
        case '7d':
          startDate.setDate(now.getDate() - 7);
          break;
        case '30d':
          startDate.setDate(now.getDate() - 30);
          break;
      }

      let url = `/api/cube-admin/analytics/queries?tenant_id=${tenant!.id}&tenant_instance_id=${datasource!.id}&start_date=${startDate.toISOString()}&end_date=${now.toISOString()}&limit=100`;
      if (cubeFilter) {
        url += `&cube_name=${cubeFilter}`;
      }

      const res = await fetch(url);
      if (res.ok) {
        const data = await res.json();
        const queryList = data || [];
        setQueries(queryList);

        // Extract unique cubes
        const cubes = [...new Set(queryList.map((q: QueryAnalytic) => q.cube_name))].filter(Boolean) as string[];
        setAvailableCubes(cubes);

        // Calculate summary
        if (queryList.length > 0) {
          const durations = queryList.map((q: QueryAnalytic) => q.duration_ms).sort((a: number, b: number) => a - b);
          const cacheHits = queryList.filter((q: QueryAnalytic) => q.cache_hit).length;
          const preAggHits = queryList.filter((q: QueryAnalytic) => q.pre_agg_used).length;
          const totalRows = queryList.reduce((sum: number, q: QueryAnalytic) => sum + (q.rows_returned || 0), 0);

          setSummary({
            totalQueries: queryList.length,
            avgDuration: Math.round(durations.reduce((a: number, b: number) => a + b, 0) / durations.length),
            p50Duration: durations[Math.floor(durations.length * 0.5)] || 0,
            p95Duration: durations[Math.floor(durations.length * 0.95)] || 0,
            p99Duration: durations[Math.floor(durations.length * 0.99)] || 0,
            cacheHitRate: (cacheHits / queryList.length) * 100,
            preAggRate: (preAggHits / queryList.length) * 100,
            avgRowsReturned: Math.round(totalRows / queryList.length),
          });
        } else {
          setSummary(null);
        }
      }
    } catch (err) {
      console.error('Failed to load analytics:', err);
    } finally {
      setLoading(false);
    }
  };

  if (!tenant || !datasource) {
    return (
      <div className="p-8">
        <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-6 text-center">
          <h2 className="text-lg font-semibold text-yellow-800">Select a Tenant</h2>
          <p className="text-yellow-700 mt-2">
            Please select a tenant and datasource to view query analytics.
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="p-8">
      {/* Header */}
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Query Analytics</h1>
          <p className="text-gray-500 mt-1">
            Monitor query performance for {tenant.display_name}
          </p>
        </div>
        <div className="flex items-center gap-4">
          <select
            value={cubeFilter}
            onChange={(e) => setCubeFilter(e.target.value)}
            className="px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500"
          >
            <option value="">All Cubes</option>
            {availableCubes.map((cube) => (
              <option key={cube} value={cube}>
                {cube}
              </option>
            ))}
          </select>
          <div className="flex items-center gap-1 bg-gray-100 rounded-lg p-1">
            {(['1h', '24h', '7d', '30d'] as const).map((range) => (
              <button
                key={range}
                onClick={() => setDateRange(range)}
                className={`px-3 py-1.5 text-sm rounded-md transition-colors ${
                  dateRange === range
                    ? 'bg-white text-gray-900 shadow-sm'
                    : 'text-gray-600 hover:text-gray-900'
                }`}
              >
                {range}
              </button>
            ))}
          </div>
        </div>
      </div>

      {loading ? (
        <LoadingSkeleton />
      ) : (
        <>
          {/* Summary Cards */}
          {summary && (
            <div className="grid grid-cols-4 gap-6 mb-8">
              <SummaryCard
                label="Total Queries"
                value={summary.totalQueries.toLocaleString()}
                icon={QueryIcon}
                color="indigo"
              />
              <SummaryCard
                label="Cache Hit Rate"
                value={`${summary.cacheHitRate.toFixed(1)}%`}
                subValue={`${summary.preAggRate.toFixed(1)}% pre-agg`}
                icon={CacheIcon}
                color="green"
              />
              <SummaryCard
                label="Avg Duration"
                value={`${summary.avgDuration}ms`}
                subValue={`P50: ${summary.p50Duration}ms`}
                icon={ClockIcon}
                color="blue"
              />
              <SummaryCard
                label="P95 / P99 Latency"
                value={`${summary.p95Duration}ms`}
                subValue={`P99: ${summary.p99Duration}ms`}
                icon={LatencyIcon}
                color="orange"
              />
            </div>
          )}

          {/* Performance Distribution Chart Placeholder */}
          <div className="bg-white rounded-xl border border-gray-200 p-6 mb-8">
            <h2 className="text-lg font-semibold text-gray-900 mb-4">Performance Distribution</h2>
            <div className="h-48 flex items-center justify-center bg-gray-50 rounded-lg border border-dashed border-gray-300">
              <div className="text-center">
                <ChartIcon className="w-12 h-12 text-gray-300 mx-auto mb-2" />
                <p className="text-gray-500">Latency distribution chart</p>
                <p className="text-xs text-gray-400">Integrate with Recharts or Chart.js</p>
              </div>
            </div>
          </div>

          {/* Query Table */}
          <div className="bg-white rounded-xl border border-gray-200 overflow-hidden">
            <div className="px-6 py-4 border-b border-gray-200">
              <h2 className="text-lg font-semibold text-gray-900">Query Log</h2>
            </div>
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead className="bg-gray-50">
                  <tr>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      Cube
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      Query Details
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      Duration
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      Cache
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      Rows
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      Time
                    </th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-100">
                  {queries.length === 0 ? (
                    <tr>
                      <td colSpan={6} className="px-6 py-12 text-center text-gray-500">
                        No queries recorded in this period
                      </td>
                    </tr>
                  ) : (
                    queries.map((query) => (
                      <tr key={query.id} className="hover:bg-gray-50">
                        <td className="px-6 py-4 whitespace-nowrap">
                          <span className="font-medium text-gray-900">{query.cube_name || '-'}</span>
                        </td>
                        <td className="px-6 py-4">
                          <div className="text-sm">
                            <div className="text-gray-900">
                              {query.measures?.length || 0} measures, {query.dimensions?.length || 0} dimensions
                            </div>
                            <div className="text-gray-500 font-mono text-xs truncate max-w-xs">
                              {query.query_hash}
                            </div>
                          </div>
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap">
                          <DurationBadge duration={query.duration_ms} />
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap">
                          <div className="flex items-center gap-2">
                            <CacheIndicator hit={query.cache_hit} />
                            {query.pre_agg_used && (
                              <span className="text-xs bg-purple-100 text-purple-700 px-2 py-0.5 rounded">
                                Pre-agg
                              </span>
                            )}
                          </div>
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-600">
                          {query.rows_returned?.toLocaleString() || '-'}
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                          {new Date(query.executed_at).toLocaleTimeString()}
                        </td>
                      </tr>
                    ))
                  )}
                </tbody>
              </table>
            </div>
          </div>
        </>
      )}
    </div>
  );
}

interface SummaryCardProps {
  label: string;
  value: string;
  subValue?: string;
  icon: React.FC<{ className?: string }>;
  color: 'indigo' | 'green' | 'blue' | 'orange';
}

function SummaryCard({ label, value, subValue, icon: Icon, color }: SummaryCardProps) {
  const colorClasses = {
    indigo: 'bg-indigo-50 text-indigo-600',
    green: 'bg-green-50 text-green-600',
    blue: 'bg-blue-50 text-blue-600',
    orange: 'bg-orange-50 text-orange-600',
  };

  return (
    <div className="bg-white rounded-xl border border-gray-200 p-6">
      <div className="flex items-center justify-between mb-4">
        <div className={`p-2 rounded-lg ${colorClasses[color]}`}>
          <Icon className="w-5 h-5" />
        </div>
      </div>
      <p className="text-2xl font-bold text-gray-900">{value}</p>
      <p className="text-sm text-gray-500 mt-1">{label}</p>
      {subValue && <p className="text-xs text-gray-400 mt-0.5">{subValue}</p>}
    </div>
  );
}

function DurationBadge({ duration }: { duration: number }) {
  let colorClass = 'bg-green-100 text-green-700';
  if (duration > 1000) {
    colorClass = 'bg-red-100 text-red-700';
  } else if (duration > 200) {
    colorClass = 'bg-yellow-100 text-yellow-700';
  }

  return (
    <span className={`px-2 py-1 text-sm rounded ${colorClass}`}>
      {duration}ms
    </span>
  );
}

function CacheIndicator({ hit }: { hit: boolean }) {
  return (
    <span
      className={`px-2 py-0.5 text-xs rounded ${
        hit ? 'bg-green-100 text-green-700' : 'bg-gray-100 text-gray-600'
      }`}
    >
      {hit ? 'Hit' : 'Miss'}
    </span>
  );
}

function LoadingSkeleton() {
  return (
    <div className="space-y-6">
      <div className="grid grid-cols-4 gap-6">
        {[...Array(4)].map((_, i) => (
          <div key={i} className="bg-white rounded-xl border p-6 animate-pulse">
            <div className="h-10 w-10 bg-gray-200 rounded-lg mb-4" />
            <div className="h-8 w-24 bg-gray-200 rounded" />
            <div className="h-4 w-16 bg-gray-100 rounded mt-2" />
          </div>
        ))}
      </div>
    </div>
  );
}

// Icons
function QueryIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8.228 9c.549-1.165 2.03-2 3.772-2 2.21 0 4 1.343 4 3 0 1.4-1.278 2.575-3.006 2.907-.542.104-.994.54-.994 1.093m0 3h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
    </svg>
  );
}

function CacheIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4m0 5c0 2.21-3.582 4-8 4s-8-1.79-8-4" />
    </svg>
  );
}

function ClockIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
    </svg>
  );
}

function LatencyIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
    </svg>
  );
}

function ChartIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
    </svg>
  );
}
