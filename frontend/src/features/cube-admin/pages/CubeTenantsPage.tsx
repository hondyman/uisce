import React, { useState, useEffect } from 'react';
import { useTenant } from '../../../context/TenantContext';

interface TenantInfo {
  id: string;
  name: string;
  display_name: string;
  tier: 'enterprise' | 'standard' | 'starter';
  status: 'active' | 'suspended' | 'pending';
  quota_queries_per_day: number;
  quota_rows_per_query: number;
  quota_storage_mb: number;
  usage_queries_today: number;
  usage_storage_mb: number;
  created_at: string;
  organization_name?: string;
}

export function CubeTenantsPage() {
  const { tenant, datasource } = useTenant();
  const [tenants, setTenants] = useState<TenantInfo[]>([]);
  const [loading, setLoading] = useState(true);
  const [selectedTenant, setSelectedTenant] = useState<TenantInfo | null>(null);
  const [tierFilter, setTierFilter] = useState<'all' | 'enterprise' | 'standard' | 'starter'>('all');
  const [search, setSearch] = useState('');

  useEffect(() => {
    if (!tenant?.id || !datasource?.id) return;
    loadTenants();
  }, [tenant?.id, datasource?.id]);

  const loadTenants = async () => {
    setLoading(true);
    try {
      // In a real implementation, this would call the tenant list API
      // For now, we'll create sample data based on the current tenant
      const sampleTenants: TenantInfo[] = [
        {
          id: tenant!.id,
          name: 'current-tenant',
          display_name: tenant!.display_name || 'Current Tenant',
          tier: 'enterprise',
          status: 'active',
          quota_queries_per_day: 100000,
          quota_rows_per_query: 1000000,
          quota_storage_mb: 10240,
          usage_queries_today: 4523,
          usage_storage_mb: 3200,
          created_at: new Date().toISOString(),
        },
      ];
      setTenants(sampleTenants);
    } catch (err) {
      console.error('Failed to load tenants:', err);
    } finally {
      setLoading(false);
    }
  };

  const filteredTenants = tenants.filter((t) => {
    const matchesSearch =
      t.name.toLowerCase().includes(search.toLowerCase()) ||
      t.display_name.toLowerCase().includes(search.toLowerCase());
    const matchesTier = tierFilter === 'all' || t.tier === tierFilter;
    return matchesSearch && matchesTier;
  });

  if (!tenant || !datasource) {
    return (
      <div className="p-8">
        <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-6 text-center">
          <h2 className="text-lg font-semibold text-yellow-800">Select a Tenant</h2>
          <p className="text-yellow-700 mt-2">
            Please select a tenant and datasource to view tenant management.
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
          <h1 className="text-2xl font-bold text-gray-900">Tenant Management</h1>
          <p className="text-gray-500 mt-1">
            View and manage Cube tenants, quotas, and usage
          </p>
        </div>
        <button className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors flex items-center gap-2">
          <PlusIcon className="w-5 h-5" />
          Add Tenant
        </button>
      </div>

      {/* Filters */}
      <div className="flex items-center gap-4 mb-6">
        <div className="flex-1 relative">
          <SearchIcon className="absolute left-3 top-1/2 -translate-y-1/2 w-5 h-5 text-gray-400" />
          <input
            type="text"
            placeholder="Search tenants..."
            aria-label="Search tenants"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="w-full pl-10 pr-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
          />
        </div>
        <div className="flex items-center gap-2 bg-gray-100 rounded-lg p-1">
          {(['all', 'enterprise', 'standard', 'starter'] as const).map((tier) => (
            <button
              key={tier}
              onClick={() => setTierFilter(tier)}
              className={`px-3 py-1.5 text-sm rounded-md transition-colors capitalize ${
                tierFilter === tier
                  ? 'bg-white text-gray-900 shadow-sm'
                  : 'text-gray-600 hover:text-gray-900'
              }`}
            >
              {tier === 'all' ? 'All Tiers' : tier}
            </button>
          ))}
        </div>
      </div>

      {/* Tenant Table */}
      {loading ? (
        <LoadingSkeleton />
      ) : (
        <div className="bg-white rounded-xl border border-gray-200 overflow-hidden">
          <table className="w-full">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Tenant
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Tier
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Status
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Query Usage
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Storage Usage
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Actions
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100">
              {filteredTenants.length === 0 ? (
                <tr>
                  <td colSpan={6} className="px-6 py-12 text-center text-gray-500">
                    No tenants found
                  </td>
                </tr>
              ) : (
                filteredTenants.map((t) => (
                  <tr key={t.id} className="hover:bg-gray-50">
                    <td className="px-6 py-4">
                      <div>
                        <p className="font-medium text-gray-900">{t.display_name}</p>
                        <p className="text-sm text-gray-500">{t.name}</p>
                      </div>
                    </td>
                    <td className="px-6 py-4">
                      <TierBadge tier={t.tier} />
                    </td>
                    <td className="px-6 py-4">
                      <StatusBadge status={t.status} />
                    </td>
                    <td className="px-6 py-4">
                      <UsageBar
                        used={t.usage_queries_today}
                        quota={t.quota_queries_per_day}
                        unit="queries"
                      />
                    </td>
                    <td className="px-6 py-4">
                      <UsageBar
                        used={t.usage_storage_mb}
                        quota={t.quota_storage_mb}
                        unit="MB"
                      />
                    </td>
                    <td className="px-6 py-4">
                      <div className="flex items-center gap-2">
                        <button
                          onClick={() => setSelectedTenant(t)}
                          className="text-indigo-600 hover:text-indigo-700 text-sm"
                        >
                          View
                        </button>
                        <button className="text-gray-500 hover:text-gray-700 text-sm">
                          Edit
                        </button>
                      </div>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      )}

      {/* Tenant Detail Modal */}
      {selectedTenant && (
        <TenantDetailModal
          tenant={selectedTenant}
          onClose={() => setSelectedTenant(null)}
        />
      )}
    </div>
  );
}

function TierBadge({ tier }: { tier: 'enterprise' | 'standard' | 'starter' }) {
  const colors = {
    enterprise: 'bg-purple-100 text-purple-700',
    standard: 'bg-blue-100 text-blue-700',
    starter: 'bg-gray-100 text-gray-700',
  };

  return (
    <span className={`px-2 py-1 text-xs font-medium rounded capitalize ${colors[tier]}`}>
      {tier}
    </span>
  );
}

function StatusBadge({ status }: { status: 'active' | 'suspended' | 'pending' }) {
  const colors = {
    active: 'bg-green-100 text-green-700',
    suspended: 'bg-red-100 text-red-700',
    pending: 'bg-yellow-100 text-yellow-700',
  };

  return (
    <span className={`px-2 py-1 text-xs font-medium rounded capitalize ${colors[status]}`}>
      {status}
    </span>
  );
}

function UsageBar({ used, quota, unit }: { used: number; quota: number; unit: string }) {
  const percentage = Math.min((used / quota) * 100, 100);
  let barColor = 'bg-green-500';
  if (percentage > 90) barColor = 'bg-red-500';
  else if (percentage > 70) barColor = 'bg-yellow-500';

  return (
    <div>
      <div className="flex items-center justify-between text-xs mb-1">
        <span className="text-gray-600">
          {used.toLocaleString()} / {quota.toLocaleString()} {unit}
        </span>
        <span className="text-gray-400">{percentage.toFixed(0)}%</span>
      </div>
      <div className="h-2 bg-gray-100 rounded-full overflow-hidden">
        <div className={`h-full ${barColor} rounded-full`} style={{ width: `${percentage}%` }} />
      </div>
    </div>
  );
}

interface TenantDetailModalProps {
  tenant: TenantInfo;
  onClose: () => void;
}

function TenantDetailModal({ tenant, onClose }: TenantDetailModalProps) {
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/40" onClick={onClose} />
      <div className="relative bg-white rounded-xl shadow-xl w-full max-w-lg p-6">
        <div className="flex items-start justify-between mb-6">
          <div>
            <h2 className="text-xl font-semibold text-gray-900">{tenant.display_name}</h2>
            <p className="text-gray-500">{tenant.name}</p>
          </div>
          <button
            onClick={onClose}
            className="p-2 hover:bg-gray-100 rounded-lg transition-colors"
            aria-label="Close modal"
          >
            <CloseIcon className="w-5 h-5 text-gray-500" />
          </button>
        </div>

        <div className="space-y-6">
          {/* Status and Tier */}
          <div className="flex items-center gap-4">
            <TierBadge tier={tenant.tier} />
            <StatusBadge status={tenant.status} />
            {tenant.organization_name && (
              <span className="text-sm text-gray-500">
                Org: {tenant.organization_name}
              </span>
            )}
          </div>

          {/* Quotas */}
          <div className="bg-gray-50 rounded-lg p-4 space-y-4">
            <h3 className="font-medium text-gray-900">Quotas & Usage</h3>
            <div>
              <label className="text-sm text-gray-500">Queries per Day</label>
              <UsageBar
                used={tenant.usage_queries_today}
                quota={tenant.quota_queries_per_day}
                unit="queries"
              />
            </div>
            <div>
              <label className="text-sm text-gray-500">Storage</label>
              <UsageBar
                used={tenant.usage_storage_mb}
                quota={tenant.quota_storage_mb}
                unit="MB"
              />
            </div>
            <div>
              <label className="text-sm text-gray-500">Rows per Query</label>
              <p className="font-medium text-gray-900">
                {tenant.quota_rows_per_query.toLocaleString()} rows max
              </p>
            </div>
          </div>

          {/* Created */}
          <div>
            <label className="text-sm text-gray-500">Created</label>
            <p className="text-gray-900">
              {new Date(tenant.created_at).toLocaleDateString()}
            </p>
          </div>
        </div>

        <div className="flex gap-3 mt-6 pt-6 border-t border-gray-200">
          <button className="flex-1 px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors">
            Edit Quotas
          </button>
          <button className="px-4 py-2 border border-gray-300 text-gray-700 rounded-lg hover:bg-gray-50 transition-colors">
            View Usage History
          </button>
        </div>
      </div>
    </div>
  );
}

function LoadingSkeleton() {
  return (
    <div className="bg-white rounded-xl border p-6 space-y-4">
      {[...Array(3)].map((_, i) => (
        <div key={i} className="flex items-center gap-6 animate-pulse">
          <div className="h-10 w-40 bg-gray-200 rounded" />
          <div className="h-6 w-20 bg-gray-200 rounded" />
          <div className="h-6 w-20 bg-gray-200 rounded" />
          <div className="flex-1 h-8 bg-gray-100 rounded" />
          <div className="flex-1 h-8 bg-gray-100 rounded" />
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

function SearchIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
    </svg>
  );
}

function CloseIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
    </svg>
  );
}
