import React, { useState, useEffect } from 'react';
import { useTenant } from '../../../context/TenantContext';

interface PreAggregation {
  id: string;
  name: string;
  cube_name: string;
  measures: string[];
  dimensions: string[];
  time_dimension?: string;
  granularity?: string;
  partition_granularity?: string;
  refresh_key: string;
  status: 'ready' | 'building' | 'error' | 'stale';
  last_refresh: string;
  next_refresh: string;
  rows_count: number;
  size_mb: number;
  hit_rate: number;
}

export function CubePreAggPage() {
  const { tenant, datasource } = useTenant();
  const [preAggs, setPreAggs] = useState<PreAggregation[]>([]);
  const [loading, setLoading] = useState(true);
  const [statusFilter, setStatusFilter] = useState<'all' | 'ready' | 'building' | 'error' | 'stale'>('all');

  useEffect(() => {
    if (!tenant?.id || !datasource?.id) return;
    loadPreAggs();
  }, [tenant?.id, datasource?.id]);

  const loadPreAggs = async () => {
    setLoading(true);
    try {
      // Sample data - would call real API
      const samplePreAggs: PreAggregation[] = [
        {
          id: '1',
          name: 'orders_by_day',
          cube_name: 'Orders',
          measures: ['count', 'total_amount'],
          dimensions: ['status'],
          time_dimension: 'created_at',
          granularity: 'day',
          partition_granularity: 'month',
          refresh_key: 'every 1 hour',
          status: 'ready',
          last_refresh: new Date(Date.now() - 3600000).toISOString(),
          next_refresh: new Date(Date.now() + 3600000).toISOString(),
          rows_count: 150000,
          size_mb: 45,
          hit_rate: 87.5,
        },
        {
          id: '2',
          name: 'users_weekly_rollup',
          cube_name: 'Users',
          measures: ['count', 'active_count'],
          dimensions: ['country', 'plan'],
          time_dimension: 'created_at',
          granularity: 'week',
          refresh_key: 'every 6 hours',
          status: 'building',
          last_refresh: new Date(Date.now() - 21600000).toISOString(),
          next_refresh: new Date(Date.now() + 600000).toISOString(),
          rows_count: 0,
          size_mb: 0,
          hit_rate: 0,
        },
        {
          id: '3',
          name: 'products_summary',
          cube_name: 'Products',
          measures: ['count', 'avg_price'],
          dimensions: ['category', 'brand'],
          refresh_key: 'every 24 hours',
          status: 'ready',
          last_refresh: new Date(Date.now() - 86400000).toISOString(),
          next_refresh: new Date(Date.now() + 3600000).toISOString(),
          rows_count: 25000,
          size_mb: 8,
          hit_rate: 92.3,
        },
      ];
      setPreAggs(samplePreAggs);
    } catch (err) {
      console.error('Failed to load pre-aggregations:', err);
    } finally {
      setLoading(false);
    }
  };

  const filteredPreAggs = preAggs.filter((p) => statusFilter === 'all' || p.status === statusFilter);

  if (!tenant || !datasource) {
    return (
      <div className="p-8">
        <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-6 text-center">
          <h2 className="text-lg font-semibold text-yellow-800">Select a Tenant</h2>
          <p className="text-yellow-700 mt-2">
            Please select a tenant and datasource to manage pre-aggregations.
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
          <h1 className="text-2xl font-bold text-gray-900">Pre-Aggregations</h1>
          <p className="text-gray-500 mt-1">
            Manage materialized views for query acceleration
          </p>
        </div>
        <div className="flex items-center gap-3">
          <button className="px-4 py-2 border border-gray-300 text-gray-700 rounded-lg hover:bg-gray-50 transition-colors flex items-center gap-2">
            <RefreshIcon className="w-5 h-5" />
            Refresh All
          </button>
          <button className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors flex items-center gap-2">
            <PlusIcon className="w-5 h-5" />
            New Pre-Agg
          </button>
        </div>
      </div>

      {/* Status Overview */}
      <div className="grid grid-cols-4 gap-6 mb-8">
        <StatusCard
          label="Ready"
          count={preAggs.filter((p) => p.status === 'ready').length}
          icon={CheckIcon}
          color="green"
          onClick={() => setStatusFilter('ready')}
          active={statusFilter === 'ready'}
        />
        <StatusCard
          label="Building"
          count={preAggs.filter((p) => p.status === 'building').length}
          icon={BuildingIcon}
          color="blue"
          onClick={() => setStatusFilter('building')}
          active={statusFilter === 'building'}
        />
        <StatusCard
          label="Stale"
          count={preAggs.filter((p) => p.status === 'stale').length}
          icon={ClockIcon}
          color="yellow"
          onClick={() => setStatusFilter('stale')}
          active={statusFilter === 'stale'}
        />
        <StatusCard
          label="Error"
          count={preAggs.filter((p) => p.status === 'error').length}
          icon={ErrorIcon}
          color="red"
          onClick={() => setStatusFilter('error')}
          active={statusFilter === 'error'}
        />
      </div>

      {/* Filter Toggle */}
      <div className="flex items-center gap-4 mb-6">
        <button
          onClick={() => setStatusFilter('all')}
          className={`px-3 py-1.5 text-sm rounded-lg transition-colors ${
            statusFilter === 'all'
              ? 'bg-indigo-100 text-indigo-700'
              : 'text-gray-600 hover:text-gray-900'
          }`}
        >
          All ({preAggs.length})
        </button>
      </div>

      {/* Pre-Agg Cards */}
      {loading ? (
        <LoadingSkeleton />
      ) : filteredPreAggs.length === 0 ? (
        <EmptyState />
      ) : (
        <div className="space-y-4">
          {filteredPreAggs.map((preAgg) => (
            <PreAggCard key={preAgg.id} preAgg={preAgg} />
          ))}
        </div>
      )}
    </div>
  );
}

interface StatusCardProps {
  label: string;
  count: number;
  icon: React.FC<{ className?: string }>;
  color: 'green' | 'blue' | 'yellow' | 'red';
  onClick: () => void;
  active: boolean;
}

function StatusCard({ label, count, icon: Icon, color, onClick, active }: StatusCardProps) {
  const colorClasses = {
    green: 'bg-green-50 text-green-600 border-green-200',
    blue: 'bg-blue-50 text-blue-600 border-blue-200',
    yellow: 'bg-yellow-50 text-yellow-600 border-yellow-200',
    red: 'bg-red-50 text-red-600 border-red-200',
  };

  return (
    <button
      onClick={onClick}
      className={`bg-white rounded-xl border p-6 text-left transition-all ${
        active ? `ring-2 ring-${color}-400` : 'hover:border-gray-300'
      }`}
    >
      <div className={`p-2 rounded-lg inline-flex ${colorClasses[color]}`}>
        <Icon className="w-5 h-5" />
      </div>
      <p className="text-2xl font-bold text-gray-900 mt-4">{count}</p>
      <p className="text-sm text-gray-500">{label}</p>
    </button>
  );
}

function PreAggCard({ preAgg }: { preAgg: PreAggregation }) {
  const statusConfig = {
    ready: { label: 'Ready', className: 'bg-green-100 text-green-700' },
    building: { label: 'Building', className: 'bg-blue-100 text-blue-700' },
    stale: { label: 'Stale', className: 'bg-yellow-100 text-yellow-700' },
    error: { label: 'Error', className: 'bg-red-100 text-red-700' },
  };

  const status = statusConfig[preAgg.status];

  return (
    <div className="bg-white rounded-xl border border-gray-200 p-6">
      <div className="flex items-start justify-between mb-4">
        <div>
          <div className="flex items-center gap-3">
            <h3 className="font-semibold text-gray-900">{preAgg.name}</h3>
            <span className={`px-2 py-0.5 text-xs rounded ${status.className}`}>
              {status.label}
            </span>
          </div>
          <p className="text-sm text-gray-500 mt-1">Cube: {preAgg.cube_name}</p>
        </div>
        <div className="flex items-center gap-2">
          <button className="px-3 py-1.5 text-sm border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors">
            Build Now
          </button>
          <button className="px-3 py-1.5 text-sm text-indigo-600 hover:text-indigo-700">
            Edit
          </button>
        </div>
      </div>

      <div className="grid grid-cols-4 gap-6">
        {/* Configuration */}
        <div className="col-span-2">
          <h4 className="text-xs font-medium text-gray-500 uppercase mb-2">Configuration</h4>
          <div className="space-y-2 text-sm">
            <div className="flex items-center gap-2">
              <span className="text-gray-500">Measures:</span>
              <span className="font-mono text-gray-900">{preAgg.measures.join(', ')}</span>
            </div>
            <div className="flex items-center gap-2">
              <span className="text-gray-500">Dimensions:</span>
              <span className="font-mono text-gray-900">{preAgg.dimensions.join(', ')}</span>
            </div>
            {preAgg.time_dimension && (
              <div className="flex items-center gap-2">
                <span className="text-gray-500">Time:</span>
                <span className="font-mono text-gray-900">
                  {preAgg.time_dimension} ({preAgg.granularity})
                </span>
              </div>
            )}
            <div className="flex items-center gap-2">
              <span className="text-gray-500">Refresh:</span>
              <span className="text-gray-900">{preAgg.refresh_key}</span>
            </div>
          </div>
        </div>

        {/* Stats */}
        <div>
          <h4 className="text-xs font-medium text-gray-500 uppercase mb-2">Storage</h4>
          <div className="space-y-2 text-sm">
            <div>
              <span className="text-gray-500">Rows: </span>
              <span className="font-medium text-gray-900">{preAgg.rows_count.toLocaleString()}</span>
            </div>
            <div>
              <span className="text-gray-500">Size: </span>
              <span className="font-medium text-gray-900">{preAgg.size_mb} MB</span>
            </div>
            <div>
              <span className="text-gray-500">Hit Rate: </span>
              <span className="font-medium text-green-600">{preAgg.hit_rate}%</span>
            </div>
          </div>
        </div>

        {/* Timing */}
        <div>
          <h4 className="text-xs font-medium text-gray-500 uppercase mb-2">Refresh Status</h4>
          <div className="space-y-2 text-sm">
            <div>
              <span className="text-gray-500">Last: </span>
              <span className="text-gray-900">
                {new Date(preAgg.last_refresh).toLocaleString()}
              </span>
            </div>
            <div>
              <span className="text-gray-500">Next: </span>
              <span className="text-gray-900">
                {new Date(preAgg.next_refresh).toLocaleString()}
              </span>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

function EmptyState() {
  return (
    <div className="text-center py-12 bg-white rounded-xl border border-gray-200">
      <LayersIcon className="w-12 h-12 text-gray-300 mx-auto mb-4" />
      <h3 className="text-lg font-medium text-gray-900">No pre-aggregations</h3>
      <p className="text-gray-500 mt-1">Create your first pre-aggregation to accelerate queries</p>
      <button className="mt-4 px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors">
        Create Pre-Aggregation
      </button>
    </div>
  );
}

function LoadingSkeleton() {
  return (
    <div className="space-y-4">
      {[...Array(3)].map((_, i) => (
        <div key={i} className="bg-white rounded-xl border p-6 animate-pulse">
          <div className="h-5 w-48 bg-gray-200 rounded mb-4" />
          <div className="grid grid-cols-4 gap-6">
            <div className="col-span-2 space-y-2">
              <div className="h-4 w-full bg-gray-100 rounded" />
              <div className="h-4 w-3/4 bg-gray-100 rounded" />
            </div>
            <div className="space-y-2">
              <div className="h-4 w-20 bg-gray-100 rounded" />
              <div className="h-4 w-16 bg-gray-100 rounded" />
            </div>
            <div className="space-y-2">
              <div className="h-4 w-24 bg-gray-100 rounded" />
              <div className="h-4 w-24 bg-gray-100 rounded" />
            </div>
          </div>
        </div>
      ))}
    </div>
  );
}

// Icons
function PlusIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
    </svg>
  );
}

function RefreshIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
    </svg>
  );
}

function CheckIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
    </svg>
  );
}

function BuildingIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
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

function ErrorIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
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
