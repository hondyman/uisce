/**
 * RuleFabricPage.tsx
 * 
 * Main page for managing all Rule Fabric rules across categories:
 * - Data Quality, Compliance, MDM, Wash Trade, Values/ESG, Workflow, Custom
 * 
 * Features:
 * - Category filtering and search
 * - Rule status indicators and governance workflow
 * - Bulk operations
 * - Environment promotion tracking
 */

import React, { useState, useEffect, useMemo, useCallback } from 'react';
import {
  Plus,
  Search,
  Filter,
  RefreshCw,
  MoreVertical,
  Database,
  Shield,
  AlertTriangle,
  TrendingUp,
  GitBranch,
  Zap,
  Play,
  Pause,
  Trash2,
  Copy,
  Eye,
  Edit
} from 'lucide-react';
import { useTenant as useTenantContext } from '../../contexts/TenantContext';

// ============================================================================
// Types
// ============================================================================

type RuleCategory = 'data_quality' | 'compliance' | 'mdm' | 'wash_trade' | 'values' | 'workflow' | 'custom';
type RuleStatus = 'draft' | 'awaiting_approval' | 'active' | 'inactive' | 'deprecated';
type RuleSeverity = 'error' | 'warning' | 'info' | 'hard_block' | 'soft_block';

interface RuleSummary {
  id: string;
  tenant_id: string;
  tenant_instance_id: string;
  name: string;
  display_name: string;
  description: string;
  category: RuleCategory;
  target_entity: string;
  severity: RuleSeverity;
  status: RuleStatus;
  version: number;
  environment: 'dev' | 'test' | 'prod';
  tags: string[];
  last_evaluated?: string;
  evaluation_count?: number;
  failure_count?: number;
  created_by?: string;
  updated_at?: string;
}

interface RuleFabricStats {
  total_rules: number;
  by_category: Record<RuleCategory, number>;
  by_status: Record<RuleStatus, number>;
  by_environment: Record<string, number>;
  recent_failures: number;
}

// ============================================================================
// Configuration
// ============================================================================

const CATEGORY_CONFIG: Record<RuleCategory, { icon: React.ReactNode; color: string; bgColor: string; label: string }> = {
  data_quality: { icon: <Database size={16} />, color: 'text-blue-600', bgColor: 'bg-blue-100', label: 'Data Quality' },
  compliance: { icon: <Shield size={16} />, color: 'text-purple-600', bgColor: 'bg-purple-100', label: 'Compliance' },
  mdm: { icon: <RefreshCw size={16} />, color: 'text-green-600', bgColor: 'bg-green-100', label: 'MDM' },
  wash_trade: { icon: <AlertTriangle size={16} />, color: 'text-red-600', bgColor: 'bg-red-100', label: 'Wash Trade' },
  values: { icon: <TrendingUp size={16} />, color: 'text-teal-600', bgColor: 'bg-teal-100', label: 'Values/ESG' },
  workflow: { icon: <GitBranch size={16} />, color: 'text-orange-600', bgColor: 'bg-orange-100', label: 'Workflow' },
  custom: { icon: <Zap size={16} />, color: 'text-gray-600', bgColor: 'bg-gray-100', label: 'Custom' }
};

const STATUS_CONFIG: Record<RuleStatus, { color: string; bgColor: string; label: string }> = {
  draft: { color: 'text-gray-600', bgColor: 'bg-gray-100', label: 'Draft' },
  awaiting_approval: { color: 'text-yellow-600', bgColor: 'bg-yellow-100', label: 'Pending' },
  active: { color: 'text-green-600', bgColor: 'bg-green-100', label: 'Active' },
  inactive: { color: 'text-gray-500', bgColor: 'bg-gray-100', label: 'Inactive' },
  deprecated: { color: 'text-red-500', bgColor: 'bg-red-100', label: 'Deprecated' }
};

const SEVERITY_CONFIG: Record<RuleSeverity, { color: string; label: string }> = {
  error: { color: 'text-red-600', label: 'Error' },
  warning: { color: 'text-yellow-600', label: 'Warning' },
  info: { color: 'text-blue-600', label: 'Info' },
  hard_block: { color: 'text-red-700', label: 'Block' },
  soft_block: { color: 'text-orange-600', label: 'Soft Block' }
};

// ============================================================================
// API Functions
// ============================================================================

const fetchRules = async (tenantId: string, datasourceId: string, filters: Record<string, string>): Promise<RuleSummary[]> => {
  const params = new URLSearchParams({ tenant_id: tenantId, tenant_instance_id: datasourceId, ...filters });
  const response = await fetch(`/api/rule-fabric/rules?${params}`, {
    headers: {
      'X-Tenant-ID': tenantId,
      'X-Tenant-Datasource-ID': datasourceId
    }
  });
  if (!response.ok) throw new Error('Failed to fetch rules');
  const data = await response.json();
  return data.rules || [];
};

const fetchStats = async (tenantId: string, datasourceId: string): Promise<RuleFabricStats> => {
  const params = new URLSearchParams({ tenant_id: tenantId, tenant_instance_id: datasourceId });
  const response = await fetch(`/api/rule-fabric/stats?${params}`, {
    headers: {
      'X-Tenant-ID': tenantId,
      'X-Tenant-Datasource-ID': datasourceId
    }
  });
  if (!response.ok) throw new Error('Failed to fetch stats');
  return response.json();
};

const deleteRule = async (tenantId: string, datasourceId: string, ruleId: string): Promise<void> => {
  const params = new URLSearchParams({ tenant_id: tenantId, tenant_instance_id: datasourceId });
  const response = await fetch(`/api/rule-fabric/rules/${ruleId}?${params}`, {
    method: 'DELETE',
    headers: {
      'X-Tenant-ID': tenantId,
      'X-Tenant-Datasource-ID': datasourceId
    }
  });
  if (!response.ok) throw new Error('Failed to delete rule');
};

const toggleRuleStatus = async (tenantId: string, datasourceId: string, ruleId: string, newStatus: RuleStatus): Promise<void> => {
  const params = new URLSearchParams({ tenant_id: tenantId, tenant_instance_id: datasourceId });
  const response = await fetch(`/api/rule-fabric/rules/${ruleId}/status?${params}`, {
    method: 'PATCH',
    headers: {
      'Content-Type': 'application/json',
      'X-Tenant-ID': tenantId,
      'X-Tenant-Datasource-ID': datasourceId
    },
    body: JSON.stringify({ status: newStatus })
  });
  if (!response.ok) throw new Error('Failed to update rule status');
};

// ============================================================================
// Sub-Components
// ============================================================================

interface StatsCardProps {
  stats: RuleFabricStats | null;
  loading: boolean;
}

const StatsCards: React.FC<StatsCardProps> = ({ stats, loading }) => {
  if (loading) {
    return (
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6">
        {[1, 2, 3, 4].map(i => (
          <div key={i} className="bg-white rounded-lg border p-4 animate-pulse">
            <div className="h-4 bg-gray-200 rounded w-20 mb-2"></div>
            <div className="h-8 bg-gray-200 rounded w-16"></div>
          </div>
        ))}
      </div>
    );
  }

  if (!stats) return null;

  return (
    <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6">
      <div className="bg-white rounded-lg border p-4">
        <div className="text-sm text-gray-500">Total Rules</div>
        <div className="text-2xl font-bold text-gray-900">{stats.total_rules}</div>
      </div>
      <div className="bg-white rounded-lg border p-4">
        <div className="text-sm text-gray-500">Active</div>
        <div className="text-2xl font-bold text-green-600">{stats.by_status.active || 0}</div>
      </div>
      <div className="bg-white rounded-lg border p-4">
        <div className="text-sm text-gray-500">Pending Approval</div>
        <div className="text-2xl font-bold text-yellow-600">{stats.by_status.awaiting_approval || 0}</div>
      </div>
      <div className="bg-white rounded-lg border p-4">
        <div className="text-sm text-gray-500">Recent Failures</div>
        <div className="text-2xl font-bold text-red-600">{stats.recent_failures}</div>
      </div>
    </div>
  );
};

interface RuleRowProps {
  rule: RuleSummary;
  onView: (rule: RuleSummary) => void;
  onEdit: (rule: RuleSummary) => void;
  onDuplicate: (rule: RuleSummary) => void;
  onDelete: (rule: RuleSummary) => void;
  onToggle: (rule: RuleSummary) => void;
}

const RuleRow: React.FC<RuleRowProps> = ({ rule, onView, onEdit, onDuplicate, onDelete, onToggle }) => {
  const [menuOpen, setMenuOpen] = useState(false);
  const categoryConfig = CATEGORY_CONFIG[rule.category];
  const statusConfig = STATUS_CONFIG[rule.status];
  const severityConfig = SEVERITY_CONFIG[rule.severity];

  return (
    <tr className="hover:bg-gray-50 border-b">
      <td className="px-4 py-3">
        <div className="flex items-center gap-3">
          <div className={`p-2 rounded-lg ${categoryConfig.bgColor}`}>
            <span className={categoryConfig.color}>{categoryConfig.icon}</span>
          </div>
          <div>
            <div className="font-medium text-gray-900">{rule.display_name || rule.name}</div>
            <div className="text-xs text-gray-500 truncate max-w-xs">{rule.description}</div>
          </div>
        </div>
      </td>
      <td className="px-4 py-3">
        <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${categoryConfig.bgColor} ${categoryConfig.color}`}>
          {categoryConfig.label}
        </span>
      </td>
      <td className="px-4 py-3 text-sm text-gray-600">{rule.target_entity}</td>
      <td className="px-4 py-3">
        <span className={`text-xs font-medium ${severityConfig.color}`}>{severityConfig.label}</span>
      </td>
      <td className="px-4 py-3">
        <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${statusConfig.bgColor} ${statusConfig.color}`}>
          {statusConfig.label}
        </span>
      </td>
      <td className="px-4 py-3">
        <span className={`px-2 py-0.5 rounded text-xs font-medium ${
          rule.environment === 'prod' ? 'bg-green-100 text-green-700' :
          rule.environment === 'test' ? 'bg-yellow-100 text-yellow-700' :
          'bg-gray-100 text-gray-700'
        }`}>
          {rule.environment.toUpperCase()}
        </span>
      </td>
      <td className="px-4 py-3 text-sm text-gray-500">
        {rule.last_evaluated ? new Date(rule.last_evaluated).toLocaleDateString() : '-'}
      </td>
      <td className="px-4 py-3">
        <div className="relative">
          <button
            onClick={() => setMenuOpen(!menuOpen)}
            className="p-1 hover:bg-gray-100 rounded"
            title="Rule actions menu"
            aria-label="Rule actions menu"
          >
            <MoreVertical size={16} className="text-gray-500" />
          </button>
          {menuOpen && (
            <>
              <div className="fixed inset-0 z-10" onClick={() => setMenuOpen(false)} />
              <div className="absolute right-0 mt-1 w-48 bg-white rounded-lg shadow-lg border z-20">
                <button
                  onClick={() => { onView(rule); setMenuOpen(false); }}
                  className="w-full flex items-center gap-2 px-4 py-2 text-sm text-gray-700 hover:bg-gray-50"
                >
                  <Eye size={14} /> View Details
                </button>
                <button
                  onClick={() => { onEdit(rule); setMenuOpen(false); }}
                  className="w-full flex items-center gap-2 px-4 py-2 text-sm text-gray-700 hover:bg-gray-50"
                >
                  <Edit size={14} /> Edit Rule
                </button>
                <button
                  onClick={() => { onDuplicate(rule); setMenuOpen(false); }}
                  className="w-full flex items-center gap-2 px-4 py-2 text-sm text-gray-700 hover:bg-gray-50"
                >
                  <Copy size={14} /> Duplicate
                </button>
                <button
                  onClick={() => { onToggle(rule); setMenuOpen(false); }}
                  className="w-full flex items-center gap-2 px-4 py-2 text-sm text-gray-700 hover:bg-gray-50"
                >
                  {rule.status === 'active' ? <Pause size={14} /> : <Play size={14} />}
                  {rule.status === 'active' ? 'Deactivate' : 'Activate'}
                </button>
                <div className="border-t" />
                <button
                  onClick={() => { onDelete(rule); setMenuOpen(false); }}
                  className="w-full flex items-center gap-2 px-4 py-2 text-sm text-red-600 hover:bg-red-50"
                >
                  <Trash2 size={14} /> Delete
                </button>
              </div>
            </>
          )}
        </div>
      </td>
    </tr>
  );
};

// ============================================================================
// Main Component
// ============================================================================

const RuleFabricPage: React.FC = () => {
  const { tenant, datasource } = useTenantContext();
  const [rules, setRules] = useState<RuleSummary[]>([]);
  const [stats, setStats] = useState<RuleFabricStats | null>(null);
  const [loading, setLoading] = useState(true);
  const [statsLoading, setStatsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Filters
  const [searchQuery, setSearchQuery] = useState('');
  const [categoryFilter, setCategoryFilter] = useState<RuleCategory | 'all'>('all');
  const [statusFilter, setStatusFilter] = useState<RuleStatus | 'all'>('all');
  const [envFilter, setEnvFilter] = useState<string>('all');

  const tenantId = tenant?.id;
  const datasourceId = datasource?.id;

  // Load data
  const loadData = useCallback(async () => {
    if (!tenantId || !datasourceId) return;

    setLoading(true);
    setError(null);
    try {
      const filters: Record<string, string> = {};
      if (categoryFilter !== 'all') filters.category = categoryFilter;
      if (statusFilter !== 'all') filters.status = statusFilter;
      if (envFilter !== 'all') filters.environment = envFilter;
      if (searchQuery) filters.search = searchQuery;

      const [rulesData, statsData] = await Promise.all([
        fetchRules(tenantId, datasourceId, filters),
        fetchStats(tenantId, datasourceId)
      ]);
      setRules(rulesData);
      setStats(statsData);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load rules');
    } finally {
      setLoading(false);
      setStatsLoading(false);
    }
  }, [tenantId, datasourceId, categoryFilter, statusFilter, envFilter, searchQuery]);

  useEffect(() => {
    loadData();
  }, [loadData]);

  // Filter rules client-side for search (API handles other filters)
  const filteredRules = useMemo(() => {
    if (!searchQuery) return rules;
    const q = searchQuery.toLowerCase();
    return rules.filter(r =>
      r.name.toLowerCase().includes(q) ||
      r.display_name?.toLowerCase().includes(q) ||
      r.description?.toLowerCase().includes(q) ||
      r.target_entity?.toLowerCase().includes(q)
    );
  }, [rules, searchQuery]);

  // Handlers
  const handleView = (rule: RuleSummary) => {
    window.location.href = `/rule-fabric/${rule.id}`;
  };

  const handleEdit = (rule: RuleSummary) => {
    window.location.href = `/rule-fabric/${rule.id}/edit`;
  };

  const handleDuplicate = (rule: RuleSummary) => {
    window.location.href = `/rule-fabric/new?duplicate=${rule.id}`;
  };

  const handleDelete = async (rule: RuleSummary) => {
    if (!tenantId || !datasourceId) return;
    if (!window.confirm(`Delete rule "${rule.display_name || rule.name}"?`)) return;

    try {
      await deleteRule(tenantId, datasourceId, rule.id);
      setRules(prev => prev.filter(r => r.id !== rule.id));
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete rule');
    }
  };

  const handleToggle = async (rule: RuleSummary) => {
    if (!tenantId || !datasourceId) return;
    const newStatus: RuleStatus = rule.status === 'active' ? 'inactive' : 'active';

    try {
      await toggleRuleStatus(tenantId, datasourceId, rule.id, newStatus);
      setRules(prev => prev.map(r => r.id === rule.id ? { ...r, status: newStatus } : r));
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update rule');
    }
  };

  // No tenant selected
  if (!tenantId || !datasourceId) {
    return (
      <div className="p-8">
        <div className="max-w-2xl mx-auto text-center py-16">
          <AlertTriangle size={48} className="mx-auto text-yellow-500 mb-4" />
          <h2 className="text-xl font-bold text-gray-900 mb-2">Select a Tenant</h2>
          <p className="text-gray-600">
            Please select a tenant and datasource from the header to view and manage rules.
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="p-6 max-w-[1600px] mx-auto">
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Rule Fabric</h1>
          <p className="text-gray-600">Unified rule management across all categories</p>
        </div>
        <div className="flex items-center gap-3">
          <button
            onClick={loadData}
            disabled={loading}
            className="flex items-center gap-2 px-4 py-2 border border-gray-300 rounded-lg hover:bg-gray-50 disabled:opacity-50"
          >
            <RefreshCw size={16} className={loading ? 'animate-spin' : ''} />
            Refresh
          </button>
          <a
            href="/rule-fabric/new"
            className="flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
          >
            <Plus size={16} />
            New Rule
          </a>
        </div>
      </div>

      {/* Stats */}
      <StatsCards stats={stats} loading={statsLoading} />

      {/* Category Quick Filters */}
      <div className="flex flex-wrap gap-2 mb-4">
        <button
          onClick={() => setCategoryFilter('all')}
          className={`px-3 py-1.5 rounded-lg text-sm font-medium transition-colors ${
            categoryFilter === 'all'
              ? 'bg-gray-900 text-white'
              : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
          }`}
        >
          All Categories
        </button>
        {(Object.keys(CATEGORY_CONFIG) as RuleCategory[]).map(cat => {
          const config = CATEGORY_CONFIG[cat];
          return (
            <button
              key={cat}
              onClick={() => setCategoryFilter(cat)}
              className={`flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm font-medium transition-colors ${
                categoryFilter === cat
                  ? `${config.bgColor} ${config.color}`
                  : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
              }`}
            >
              {config.icon}
              {config.label}
              {stats?.by_category[cat] ? (
                <span className="ml-1 text-xs opacity-70">({stats.by_category[cat]})</span>
              ) : null}
            </button>
          );
        })}
      </div>

      {/* Search and Filters */}
      <div className="bg-white rounded-lg border p-4 mb-6">
        <div className="flex flex-wrap items-center gap-4">
          <div className="flex-1 min-w-[200px]">
            <div className="relative">
              <Search size={16} className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
              <input
                type="text"
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                placeholder="Search rules..."
                className="w-full pl-9 pr-4 py-2 border border-gray-300 rounded-lg"
              />
            </div>
          </div>

          <div className="flex items-center gap-2">
            <Filter size={16} className="text-gray-500" />
            <select
              value={statusFilter}
              onChange={(e) => setStatusFilter(e.target.value as RuleStatus | 'all')}
              className="px-3 py-2 border border-gray-300 rounded-lg text-sm"
              title="Filter by status"
              aria-label="Filter by status"
            >
              <option value="all">All Status</option>
              {(Object.keys(STATUS_CONFIG) as RuleStatus[]).map(status => (
                <option key={status} value={status}>{STATUS_CONFIG[status].label}</option>
              ))}
            </select>

            <select
              value={envFilter}
              onChange={(e) => setEnvFilter(e.target.value)}
              className="px-3 py-2 border border-gray-300 rounded-lg text-sm"
              title="Filter by environment"
              aria-label="Filter by environment"
            >
              <option value="all">All Environments</option>
              <option value="dev">Dev</option>
              <option value="test">Test</option>
              <option value="prod">Prod</option>
            </select>
          </div>
        </div>
      </div>

      {/* Error */}
      {error && (
        <div className="mb-6 p-4 bg-red-50 border border-red-200 rounded-lg">
          <p className="text-red-700">{error}</p>
        </div>
      )}

      {/* Rules Table */}
      <div className="bg-white rounded-lg border overflow-hidden">
        {loading ? (
          <div className="p-8 text-center">
            <RefreshCw size={24} className="animate-spin mx-auto text-gray-400 mb-2" />
            <p className="text-gray-500">Loading rules...</p>
          </div>
        ) : filteredRules.length === 0 ? (
          <div className="p-8 text-center">
            <Database size={48} className="mx-auto text-gray-300 mb-4" />
            <h3 className="text-lg font-semibold text-gray-900 mb-2">No Rules Found</h3>
            <p className="text-gray-500 mb-4">
              {searchQuery || categoryFilter !== 'all' || statusFilter !== 'all'
                ? 'Try adjusting your filters'
                : 'Get started by creating your first rule'}
            </p>
            <a
              href="/rule-fabric/new"
              className="inline-flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
            >
              <Plus size={16} />
              Create Rule
            </a>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead className="bg-gray-50 border-b">
                <tr>
                  <th className="px-4 py-3 text-left text-xs font-semibold text-gray-600 uppercase">Rule</th>
                  <th className="px-4 py-3 text-left text-xs font-semibold text-gray-600 uppercase">Category</th>
                  <th className="px-4 py-3 text-left text-xs font-semibold text-gray-600 uppercase">Entity</th>
                  <th className="px-4 py-3 text-left text-xs font-semibold text-gray-600 uppercase">Severity</th>
                  <th className="px-4 py-3 text-left text-xs font-semibold text-gray-600 uppercase">Status</th>
                  <th className="px-4 py-3 text-left text-xs font-semibold text-gray-600 uppercase">Env</th>
                  <th className="px-4 py-3 text-left text-xs font-semibold text-gray-600 uppercase">Last Run</th>
                  <th className="px-4 py-3 w-12"></th>
                </tr>
              </thead>
              <tbody>
                {filteredRules.map(rule => (
                  <RuleRow
                    key={rule.id}
                    rule={rule}
                    onView={handleView}
                    onEdit={handleEdit}
                    onDuplicate={handleDuplicate}
                    onDelete={handleDelete}
                    onToggle={handleToggle}
                  />
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Results count */}
      {!loading && filteredRules.length > 0 && (
        <div className="mt-4 text-sm text-gray-500">
          Showing {filteredRules.length} of {rules.length} rules
        </div>
      )}
    </div>
  );
};

export default RuleFabricPage;
