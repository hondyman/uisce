import React, { useState, useMemo } from 'react';
import { getMultiUpgradeOverview, activateVersion, startCanary, rollbackVersion } from '../api';
import { useUpgradeWebSocket } from '../hooks/useWebSocket';

interface UpgradeStatusResponse {
  core_version: string;
  status: "pending" | "ready" | "canary" | "active" | "rolled_back";
  warnings: string[];
  blockers: string[];
}

interface UpgradeOverviewResponse {
  schema_version: string;
  changelog?: Array<{
    version: string;
    date: string;
    description: string;
  }>;
  report: any; // DiffReport
  aliases: any; // AliasMap
  status: UpgradeStatusResponse;
  ui_hints?: {
    needs_diff_review: boolean;
    needs_extension_fix: boolean;
    needs_query_run: boolean;
  };
}

interface MultiUpgradeOverviewResponse {
  versions: UpgradeOverviewResponse[];
}

type Props = {
  onSelectVersion: (coreVersion: string) => void;
  onActivate?: (coreVersion: string) => void;
  onCanary?: (coreVersion: string) => void;
  onRollback?: (coreVersion: string) => void;
};

// Using shared types from ../../types/upgrade

export const VersionsTable: React.FC<Props> = ({
  onSelectVersion,
  onActivate,
  onCanary,
  onRollback
}) => {
  const [sortKey, setSortKey] = useState<'core_version' | 'status' | 'schema_version'>('core_version');
  const [sortDir, setSortDir] = useState<'asc' | 'desc'>('desc');
  const [statusFilter, setStatusFilter] = useState<string[]>([]);
  const [search, setSearch] = useState('');
  const [data, setData] = useState<MultiUpgradeOverviewResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // WebSocket for real-time updates
  const { isConnected: wsConnected, getStatusForVersion } = useUpgradeWebSocket();

  // Load data on component mount
  React.useEffect(() => {
    loadData();
  }, []);

  const loadData = async (
    coreVersions?: string[],
    statuses?: string[],
    sort?: string
  ) => {
    setLoading(true);
    setError(null);
    try {
      const result = await getMultiUpgradeOverview(coreVersions, statuses, sort);
      setData(result);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load versions');
    } finally {
      setLoading(false);
    }
  };

  const toggleSort = (key: typeof sortKey) => {
    if (sortKey === key) {
      setSortDir(sortDir === 'asc' ? 'desc' : 'asc');
    } else {
      setSortKey(key);
      setSortDir('asc');
    }
  };

  const toggleStatus = (status: string) => {
    setStatusFilter((prev) =>
      prev.includes(status) ? prev.filter((s) => s !== status) : [...prev, status]
    );
  };

  const filteredSorted = useMemo(() => {
    if (!data?.versions) return [];

    let versions = [...data.versions];

    // Merge real-time status updates from WebSocket
    versions = versions.map(v => {
  const realTimeStatus = getStatusForVersion(v.status.core_version);
      if (realTimeStatus) {
        return {
          ...v,
          status: {
            ...v.status,
    status: realTimeStatus.status as "pending" | "ready" | "canary" | "active" | "rolled_back",
            warnings: realTimeStatus.warnings || v.status.warnings,
            blockers: realTimeStatus.blockers || v.status.blockers,
          }
        };
      }
      return v;
    });

    // Filter by status
    if (statusFilter.length > 0) {
      versions = versions.filter((v) => statusFilter.includes(v.status.status));
    }

    // Search by core version
    if (search.trim()) {
      versions = versions.filter((v) =>
        v.status.core_version.toLowerCase().includes(search.toLowerCase())
      );
    }

    // Sort
    versions.sort((a, b) => {
      let aVal: string | number = '';
      let bVal: string | number = '';
      if (sortKey === 'core_version') {
        aVal = a.status.core_version;
        bVal = b.status.core_version;
      } else if (sortKey === 'status') {
        aVal = a.status.status;
        bVal = b.status.status;
      } else if (sortKey === 'schema_version') {
        aVal = a.schema_version;
        bVal = b.schema_version;
      }
      if (aVal < bVal) return sortDir === 'asc' ? -1 : 1;
      if (aVal > bVal) return sortDir === 'asc' ? 1 : -1;
      return 0;
    });

    return versions;
  }, [data?.versions, sortKey, sortDir, statusFilter, search, getStatusForVersion]);

  const handleRefresh = () => {
    loadData();
  };

  const handleFilteredRefresh = () => {
    const statuses = statusFilter.length > 0 ? statusFilter : undefined;
    const sort = sortKey === 'core_version' && sortDir === 'desc' ? undefined : `${sortKey}_${sortDir}`;
    loadData(undefined, statuses, sort);
  };

  // Action handlers
  const handleActivate = async (coreVersion: string) => {
    try {
      await activateVersion(coreVersion);
      // Refresh data to show updated status
      await loadData();
      // Call parent handler if provided
      onActivate?.(coreVersion);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to activate version');
    }
  };

  const handleCanary = async (coreVersion: string) => {
    // For demo purposes, use some default tenants
    const tenants = ['tenantA', 'tenantB'];
    try {
      await startCanary(coreVersion, tenants);
      // Refresh data to show updated status
      await loadData();
      // Call parent handler if provided
      onCanary?.(coreVersion);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to start canary deployment');
    }
  };

  const handleRollback = async (coreVersion: string) => {
    try {
      await rollbackVersion(coreVersion);
      // Refresh data to show updated status
      await loadData();
      // Call parent handler if provided
      onRollback?.(coreVersion);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to rollback version');
    }
  };

  const statusColor = (status: string) => {
    switch (status) {
      case 'active': return 'bg-green-100 text-green-800 border-green-200';
      case 'canary': return 'bg-yellow-100 text-yellow-800 border-yellow-200';
      case 'ready': return 'bg-blue-100 text-blue-800 border-blue-200';
      case 'pending': return 'bg-gray-100 text-gray-800 border-gray-200';
      case 'rolled_back': return 'bg-red-100 text-red-800 border-red-200';
      default: return 'bg-gray-100 text-gray-800 border-gray-200';
    }
  };

  const getRowHighlightClass = (status: string) => {
    switch (status) {
      case 'active': return 'bg-green-50 hover:bg-green-100 border-l-4 border-l-green-400';
      case 'canary': return 'bg-yellow-50 hover:bg-yellow-100 border-l-4 border-l-yellow-400';
      case 'ready': return 'bg-blue-50 hover:bg-blue-100 border-l-4 border-l-blue-400';
      case 'pending': return 'bg-gray-50 hover:bg-gray-100 border-l-4 border-l-gray-400';
      case 'rolled_back': return 'bg-red-50 hover:bg-red-100 border-l-4 border-l-red-400';
      default: return 'hover:bg-gray-50 border-l-4 border-l-gray-300';
    }
  };

  if (loading && !data) {
    return (
      <div className="flex items-center justify-center p-8">
        <div className="text-gray-500">Loading versions...</div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="p-4 bg-red-50 border border-red-200 rounded">
        <div className="text-red-800 font-medium">Error loading versions</div>
        <div className="text-red-600 text-sm mt-1">{error}</div>
        <button
          onClick={handleRefresh}
          className="mt-2 px-3 py-1 bg-red-600 text-white rounded text-sm hover:bg-red-700"
        >
          Retry
        </button>
      </div>
    );
  }

  return (
    <div className="versions-table space-y-4">
      {/* Connection Status */}
      <div className="flex items-center gap-2 mb-4">
        <div className={`w-2 h-2 rounded-full ${wsConnected ? 'bg-green-500' : 'bg-red-500'}`}></div>
        <span className="text-sm text-gray-600">
          {wsConnected ? 'Real-time updates active' : 'Real-time updates offline'}
        </span>
      </div>

      {/* Controls */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <input
            type="text"
            placeholder="Search core version..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="border rounded px-3 py-2 text-sm"
          />
          <div className="flex items-center gap-2">
            {['pending', 'ready', 'canary', 'active', 'rolled_back'].map((status) => (
              <label key={status} className="flex items-center gap-1 text-sm">
                <input
                  type="checkbox"
                  checked={statusFilter.includes(status)}
                  onChange={() => toggleStatus(status)}
                  className="rounded"
                />
                <span className="capitalize">{status.replace('_', ' ')}</span>
              </label>
            ))}
          </div>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={handleFilteredRefresh}
            className="px-3 py-2 bg-blue-600 text-white rounded text-sm hover:bg-blue-700"
          >
            Apply Filters
          </button>
          <button
            onClick={handleRefresh}
            className="px-3 py-2 bg-gray-600 text-white rounded text-sm hover:bg-gray-700"
          >
            Refresh All
          </button>
        </div>
      </div>

      {/* Table */}
      <div className="border rounded overflow-hidden">
        <table className="min-w-full">
          <thead className="bg-gray-50">
            <tr>
              <th className="px-4 py-3 text-left">
                <button
                  className="font-medium text-gray-900 hover:text-gray-700 cursor-pointer"
                  onClick={() => toggleSort('core_version')}
                >
                  Core Version {sortKey === 'core_version' && (sortDir === 'asc' ? '↑' : '↓')}
                </button>
              </th>
              <th className="px-4 py-3 text-left">
                <button
                  className="font-medium text-gray-900 hover:text-gray-700 cursor-pointer"
                  onClick={() => toggleSort('status')}
                >
                  Status {sortKey === 'status' && (sortDir === 'asc' ? '↑' : '↓')}
                </button>
              </th>
              <th className="px-4 py-3 text-left">
                <button
                  className="font-medium text-gray-900 hover:text-gray-700 cursor-pointer"
                  onClick={() => toggleSort('schema_version')}
                >
                  Schema Version {sortKey === 'schema_version' && (sortDir === 'asc' ? '↑' : '↓')}
                </button>
              </th>
              <th className="px-4 py-3 text-left font-medium text-gray-900">Warnings</th>
              <th className="px-4 py-3 text-left font-medium text-gray-900">Blockers</th>
              <th className="px-4 py-3 text-left font-medium text-gray-900">Actions</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-200">
            {filteredSorted.map((v) => (
              <tr key={v.status.core_version} className={`${getRowHighlightClass(v.status.status)} transition-colors duration-150`}>
                <td className="px-4 py-3 text-sm font-mono">{v.status.core_version}</td>
                <td className="px-4 py-3">
                  <span className={`inline-flex px-3 py-1 text-xs font-semibold rounded-full border ${statusColor(v.status.status)}`}>
                    {v.status.status.replace('_', ' ')}
                  </span>
                </td>
                <td className="px-4 py-3 text-sm font-mono">{v.schema_version}</td>
                <td className="px-4 py-3 text-sm">
                  {v.status.warnings.length > 0 ? (
                    <span className="text-yellow-600 font-medium">{v.status.warnings.length}</span>
                  ) : (
                    <span className="text-gray-400">0</span>
                  )}
                </td>
                <td className="px-4 py-3 text-sm">
                  {v.status.blockers.length > 0 ? (
                    <span className="text-red-600 font-medium">{v.status.blockers.length}</span>
                  ) : (
                    <span className="text-gray-400">0</span>
                  )}
                </td>
                <td className="px-4 py-3">
                  <div className="flex items-center gap-2 flex-wrap">
                    {onActivate && (
                      <button
                        className={`px-2 py-1 text-xs font-medium rounded transition-colors duration-150 ${
                          v.status.status === 'active'
                            ? 'bg-gray-300 text-gray-500 cursor-not-allowed'
                            : 'bg-green-600 text-white hover:bg-green-700 active:bg-green-800'
                        }`}
                        onClick={() => handleActivate(v.status.core_version)}
                        disabled={v.status.status === 'active'}
                        title={v.status.status === 'active' ? 'Already active' : 'Activate this version'}
                      >
                        Activate
                      </button>
                    )}
                    {onCanary && (
                      <button
                        className={`px-2 py-1 text-xs font-medium rounded transition-colors duration-150 ${
                          v.status.status === 'canary'
                            ? 'bg-gray-300 text-gray-500 cursor-not-allowed'
                            : 'bg-yellow-600 text-white hover:bg-yellow-700 active:bg-yellow-800'
                        }`}
                        onClick={() => handleCanary(v.status.core_version)}
                        disabled={v.status.status === 'canary'}
                        title={v.status.status === 'canary' ? 'Already in canary' : 'Start canary deployment'}
                      >
                        Canary
                      </button>
                    )}
                    {onRollback && (
                      <button
                        className={`px-2 py-1 text-xs font-medium rounded transition-colors duration-150 ${
                          v.status.status === 'rolled_back'
                            ? 'bg-gray-300 text-gray-500 cursor-not-allowed'
                            : 'bg-red-600 text-white hover:bg-red-700 active:bg-red-800'
                        }`}
                        onClick={() => handleRollback(v.status.core_version)}
                        disabled={v.status.status === 'rolled_back'}
                        title={v.status.status === 'rolled_back' ? 'Already rolled back' : 'Rollback to previous version'}
                      >
                        Rollback
                      </button>
                    )}
                    <button
                      className="px-2 py-1 text-xs font-medium bg-blue-600 text-white rounded hover:bg-blue-700 active:bg-blue-800 transition-colors duration-150"
                      onClick={() => onSelectVersion(v.status.core_version)}
                      title="View detailed information"
                    >
                      View
                    </button>
                  </div>
                </td>
              </tr>
            ))}
            {filteredSorted.length === 0 && (
              <tr>
                <td colSpan={6} className="px-4 py-8 text-center text-gray-500">
                  {data?.versions.length === 0 ? 'No versions available' : 'No versions match your filters'}
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>

      {/* Summary */}
      {data && (
        <div className="text-sm text-gray-600">
          Showing {filteredSorted.length} of {data.versions.length} versions
        </div>
      )}
    </div>
  );
};
