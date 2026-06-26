import React, { useState, useEffect } from 'react';
import { useTenant } from '../../../context/TenantContext';

interface DashboardMetrics {
  totalQueries: number;
  cacheHitRate: number;
  p95Latency: number;
  activePreAggs: number;
  totalModels: number;
  activeOrganizations: number;
  activeTenants: number;
  scheduledReports: number;
}

interface RecentQuery {
  id: string;
  cubeName: string;
  query: string;
  duration: number;
  cacheHit: boolean;
  timestamp: string;
}

export function CubeAdminDashboard() {
  const { tenant, datasource } = useTenant();
  const [metrics, setMetrics] = useState<DashboardMetrics | null>(null);
  const [recentQueries, setRecentQueries] = useState<RecentQuery[]>([]);
  const [loading, setLoading] = useState(true);
  const [period, setPeriod] = useState<'1h' | '24h' | '7d' | '30d'>('24h');

  useEffect(() => {
    if (!tenant?.id || !datasource?.id) return;
    loadDashboard();
  }, [tenant?.id, datasource?.id, period]);

  const loadDashboard = async () => {
    setLoading(true);
    try {
      // Calculate date range
      const now = new Date();
      const startDate = new Date();
      switch (period) {
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

      const [analyticsRes, modelsRes] = await Promise.all([
        fetch(
          `/api/cube-admin/analytics/queries?tenant_id=${tenant!.id}&tenant_instance_id=${datasource!.id}&start_date=${startDate.toISOString()}&end_date=${now.toISOString()}&limit=10`
        ),
        fetch(
          `/api/cube-admin/semantic-models?tenant_id=${tenant!.id}&tenant_instance_id=${datasource!.id}`
        ),
      ]);

      if (analyticsRes.ok && modelsRes.ok) {
        const analytics = await analyticsRes.json();
        const models = await modelsRes.json();

        // Compute metrics from analytics
        const queries = analytics || [];
        const totalQueries = queries.length;
        const cacheHits = queries.filter((q: any) => q.cache_hit).length;
        const cacheHitRate = totalQueries > 0 ? (cacheHits / totalQueries) * 100 : 0;
        const latencies = queries.map((q: any) => q.duration_ms).sort((a: number, b: number) => a - b);
        const p95Index = Math.floor(latencies.length * 0.95);
        const p95Latency = latencies[p95Index] || 0;

        setMetrics({
          totalQueries,
          cacheHitRate,
          p95Latency,
          activePreAggs: Math.floor(Math.random() * 50) + 10, // TODO: Real pre-agg count
          totalModels: models?.length || 0,
          activeOrganizations: 3, // TODO: Real org count
          activeTenants: 12, // TODO: Real tenant count
          scheduledReports: 8, // TODO: Real report count
        });

        setRecentQueries(
          queries.slice(0, 5).map((q: any) => ({
            id: q.id,
            cubeName: q.cube_name || 'Unknown',
            query: q.query_hash?.slice(0, 12) + '...',
            duration: q.duration_ms,
            cacheHit: q.cache_hit,
            timestamp: q.executed_at,
          }))
        );
      }
    } catch (err) {
      console.error('Failed to load dashboard:', err);
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
            Please select a tenant and datasource from the header to view Cube Admin metrics.
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
          <h1 className="text-2xl font-bold text-gray-900">Cube Admin Dashboard</h1>
          <p className="text-gray-500 mt-1">
            Monitoring semantic layer for {tenant.display_name}
          </p>
        </div>
        <div className="flex items-center gap-2">
          {(['1h', '24h', '7d', '30d'] as const).map((p) => (
            <button
              key={p}
              onClick={() => setPeriod(p)}
              className={`px-3 py-1.5 text-sm rounded-lg transition-colors ${
                period === p
                  ? 'bg-indigo-600 text-white'
                  : 'bg-white text-gray-600 border hover:bg-gray-50'
              }`}
            >
              {p}
            </button>
          ))}
        </div>
      </div>

      {loading ? (
        <LoadingSkeleton />
      ) : (
        <>
          {/* Metrics Grid */}
          <div className="grid grid-cols-4 gap-6 mb-8">
            <MetricCard
              label="Total Queries"
              value={metrics?.totalQueries.toLocaleString() || '0'}
              icon={QueryIcon}
              trend={12.5}
              color="indigo"
            />
            <MetricCard
              label="Cache Hit Rate"
              value={`${metrics?.cacheHitRate.toFixed(1)}%`}
              icon={CacheIcon}
              trend={3.2}
              color="green"
            />
            <MetricCard
              label="P95 Latency"
              value={`${metrics?.p95Latency}ms`}
              icon={LatencyIcon}
              trend={-8.1}
              color="blue"
            />
            <MetricCard
              label="Active Pre-Aggs"
              value={metrics?.activePreAggs.toString() || '0'}
              icon={LayersIcon}
              color="purple"
            />
          </div>

          {/* Second Row */}
          <div className="grid grid-cols-4 gap-6 mb-8">
            <MetricCard
              label="Semantic Models"
              value={metrics?.totalModels.toString() || '0'}
              icon={CubeIcon}
              color="pink"
            />
            <MetricCard
              label="Organizations"
              value={metrics?.activeOrganizations.toString() || '0'}
              icon={OrgIcon}
              color="orange"
            />
            <MetricCard
              label="Active Tenants"
              value={metrics?.activeTenants.toString() || '0'}
              icon={TenantsIcon}
              color="teal"
            />
            <MetricCard
              label="Scheduled Reports"
              value={metrics?.scheduledReports.toString() || '0'}
              icon={ReportIcon}
              color="yellow"
            />
          </div>

          {/* Recent Queries */}
          <div className="bg-white rounded-xl border border-gray-200 overflow-hidden">
            <div className="px-6 py-4 border-b border-gray-200 flex items-center justify-between">
              <h2 className="text-lg font-semibold text-gray-900">Recent Queries</h2>
              <a
                href="/cube-admin/analytics"
                className="text-sm text-indigo-600 hover:text-indigo-700"
              >
                View all →
              </a>
            </div>
            <div className="divide-y divide-gray-100">
              {recentQueries.length === 0 ? (
                <div className="p-6 text-center text-gray-500">
                  No queries recorded in this period
                </div>
              ) : (
                recentQueries.map((query) => (
                  <div key={query.id} className="px-6 py-4 flex items-center justify-between">
                    <div className="flex items-center gap-4">
                      <div
                        className={`w-2 h-2 rounded-full ${
                          query.cacheHit ? 'bg-green-500' : 'bg-yellow-500'
                        }`}
                      />
                      <div>
                        <p className="font-medium text-gray-900">{query.cubeName}</p>
                        <p className="text-sm text-gray-500 font-mono">{query.query}</p>
                      </div>
                    </div>
                    <div className="text-right">
                      <p className="text-sm font-medium text-gray-900">{query.duration}ms</p>
                      <p className="text-xs text-gray-500">
                        {new Date(query.timestamp).toLocaleTimeString()}
                      </p>
                    </div>
                  </div>
                ))
              )}
            </div>
          </div>

          {/* Quick Actions */}
          <div className="mt-8 grid grid-cols-3 gap-6">
            <QuickAction
              title="Manage Semantic Models"
              description="Browse and edit cube definitions"
              href="/cube-admin/catalog"
              icon={CubeIcon}
            />
            <QuickAction
              title="Configure Pre-Aggregations"
              description="Optimize query performance"
              href="/cube-admin/preaggs"
              icon={LayersIcon}
            />
            <QuickAction
              title="Schedule Reports"
              description="Set up automated exports"
              href="/cube-admin/reports"
              icon={ReportIcon}
            />
          </div>
        </>
      )}
    </div>
  );
}

interface MetricCardProps {
  label: string;
  value: string;
  icon: React.FC<{ className?: string }>;
  trend?: number;
  color: 'indigo' | 'green' | 'blue' | 'purple' | 'pink' | 'orange' | 'teal' | 'yellow';
}

function MetricCard({ label, value, icon: Icon, trend, color }: MetricCardProps) {
  const colorClasses = {
    indigo: 'bg-indigo-50 text-indigo-600',
    green: 'bg-green-50 text-green-600',
    blue: 'bg-blue-50 text-blue-600',
    purple: 'bg-purple-50 text-purple-600',
    pink: 'bg-pink-50 text-pink-600',
    orange: 'bg-orange-50 text-orange-600',
    teal: 'bg-teal-50 text-teal-600',
    yellow: 'bg-yellow-50 text-yellow-600',
  };

  return (
    <div className="bg-white rounded-xl border border-gray-200 p-6">
      <div className="flex items-center justify-between mb-4">
        <div className={`p-2 rounded-lg ${colorClasses[color]}`}>
          <Icon className="w-5 h-5" />
        </div>
        {trend !== undefined && (
          <span
            className={`text-sm font-medium ${
              trend >= 0 ? 'text-green-600' : 'text-red-600'
            }`}
          >
            {trend >= 0 ? '↑' : '↓'} {Math.abs(trend)}%
          </span>
        )}
      </div>
      <p className="text-2xl font-bold text-gray-900">{value}</p>
      <p className="text-sm text-gray-500 mt-1">{label}</p>
    </div>
  );
}

interface QuickActionProps {
  title: string;
  description: string;
  href: string;
  icon: React.FC<{ className?: string }>;
}

function QuickAction({ title, description, href, icon: Icon }: QuickActionProps) {
  return (
    <a
      href={href}
      className="bg-white rounded-xl border border-gray-200 p-6 hover:border-indigo-300 hover:shadow-md transition-all group"
    >
      <Icon className="w-8 h-8 text-indigo-600 mb-4" />
      <h3 className="font-semibold text-gray-900 group-hover:text-indigo-600">{title}</h3>
      <p className="text-sm text-gray-500 mt-1">{description}</p>
    </a>
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

function LatencyIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
    </svg>
  );
}

function LayersIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
    </svg>
  );
}

function CubeIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4" />
    </svg>
  );
}

function OrgIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 21V5a2 2 0 00-2-2H7a2 2 0 00-2 2v16m14 0h2m-2 0h-5m-9 0H3m2 0h5M9 7h1m-1 4h1m4-4h1m-1 4h1m-5 10v-5a1 1 0 011-1h2a1 1 0 011 1v5m-4 0h4" />
    </svg>
  );
}

function TenantsIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z" />
    </svg>
  );
}

function ReportIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 17v-2m3 2v-4m3 4v-6m2 10H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
    </svg>
  );
}
