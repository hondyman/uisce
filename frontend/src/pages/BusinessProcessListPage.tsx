import React, { useEffect, useState } from 'react';
import { useNotification } from '../hooks/useNotification';
import { useConfirm } from '../components/ConfirmProvider';
import { Plus, Edit, Trash2, Play, Archive, ChevronRight, Search, Loader, AlertCircle } from 'lucide-react';
// Reference Trash2 to silence no-unused-vars when icon is intentionally retained for future actions
void Trash2;
import { devError } from '../utils/devLogger';
import { getSelectedRegion } from '../lib/region';

// ============================================================================
// Types
// ============================================================================

interface BusinessProcess {
  id: string;
  processName: string;
  entity: string;
  status: 'draft' | 'published' | 'archived';
  isActive: boolean;
  stepsCount: number;
  totalDurationHours?: number;
  createdBy: string;
  createdAt: string;
  updatedAt?: string;
}

interface ListResponse {
  processes: BusinessProcess[];
  total: number;
  count: number;
}

// ============================================================================
// Business Process List Component
// ============================================================================

const BusinessProcessList: React.FC = () => {
  const [processes, setProcesses] = useState<BusinessProcess[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [searchTerm, setSearchTerm] = useState('');
  const [filterStatus, setFilterStatus] = useState<string>('all');
  const [offset, setOffset] = useState(0);
  const [totalCount, setTotalCount] = useState(0);
  const [deleting, setDeleting] = useState<string | null>(null);
  const confirm = useConfirm();
  const notification = useNotification();

  const limit = 20;

  // Fetch processes
  useEffect(() => {
    fetchProcesses();
  }, [offset, filterStatus]);

  const fetchProcesses = async () => {
    setLoading(true);
    setError(null);

    try {
      const params = new URLSearchParams();
      params.append('offset', offset.toString());
      params.append('limit', limit.toString());

      // Get tenant from localStorage (as per tenant scope requirement)
      const tenantData = localStorage.getItem('selected_tenant');
      const datasourceData = localStorage.getItem('selected_datasource');

      if (!tenantData || !datasourceData) {
        setError('Please select a tenant and datasource first');
        setLoading(false);
        return;
      }

      const tenant = JSON.parse(tenantData);
      const datasource = JSON.parse(datasourceData);

      const response = await fetch(`/api/bp?${params}`, {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
          'X-Tenant-ID': tenant.id,
          'X-Tenant-Datasource-ID': datasource.id,
          'X-Tenant-Region': getSelectedRegion(),
        },
      });

      if (!response.ok) {
        throw new Error(`Failed to fetch processes: ${response.statusText}`);
      }

      const data: ListResponse = await response.json();
      setProcesses(data.processes);
      setTotalCount(data.total);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error occurred');
      devError('Error fetching processes:', err);
    } finally {
      setLoading(false);
    }
  };

  const handleDeleteProcess = async (processId: string) => {
    const confirm = useConfirm();
    if (!(await confirm({ title: 'Archive process', description: 'Are you sure you want to archive this business process?' }))) {
      return;
    }

    setDeleting(processId);

    try {
      const tenantData = localStorage.getItem('selected_tenant');
      const datasourceData = localStorage.getItem('selected_datasource');

      if (!tenantData || !datasourceData) {
        notification.error('Please select a tenant and datasource');
        return;
      }

      const tenant = JSON.parse(tenantData);
      const datasource = JSON.parse(datasourceData);

      const response = await fetch(`/api/bp/${processId}`, {
        method: 'DELETE',
        headers: {
          'X-Tenant-ID': tenant.id,
          'X-Tenant-Datasource-ID': datasource.id,
        },
      });

      if (response.ok) {
        setProcesses(processes.filter(p => p.id !== processId));
      } else {
        notification.error('Failed to delete process');
      }
    } catch (err) {
      devError('Error deleting process:', err);
      notification.error('Error deleting process');
    } finally {
      setDeleting(null);
    }
  };

  const filteredProcesses = processes.filter(p => {
    const matchesSearch = p.processName.toLowerCase().includes(searchTerm.toLowerCase()) ||
                         p.entity.toLowerCase().includes(searchTerm.toLowerCase());
    const matchesStatus = filterStatus === 'all' || p.status === filterStatus;
    return matchesSearch && matchesStatus;
  });

  const getStatusBadge = (status: string) => {
    const baseStyle = 'px-3 py-1 rounded-full text-xs font-semibold';
    switch (status) {
      case 'draft':
        return <span className={`${baseStyle} bg-gray-100 text-gray-800`}>Draft</span>;
      case 'published':
        return <span className={`${baseStyle} bg-green-100 text-green-800`}>Published</span>;
      case 'archived':
        return <span className={`${baseStyle} bg-red-100 text-red-800`}>Archived</span>;
      default:
        return <span className={`${baseStyle} bg-gray-100 text-gray-800`}>{status}</span>;
    }
  };

  const getActiveStatus = (isActive: boolean) => {
    return isActive ? (
      <span className="px-3 py-1 rounded-full text-xs font-semibold bg-blue-100 text-blue-800">
        Active
      </span>
    ) : (
      <span className="px-3 py-1 rounded-full text-xs font-semibold bg-gray-100 text-gray-800">
        Inactive
      </span>
    );
  };

  const pageInfo = {
    start: offset + 1,
    end: Math.min(offset + limit, totalCount),
    total: totalCount,
  };

  return (
    <div className="min-h-screen bg-gray-50 p-8">
      <div className="max-w-7xl mx-auto">
        {/* Header */}
        <div className="mb-8">
          <div className="flex items-center justify-between mb-2">
            <h1 className="text-3xl font-bold text-gray-900">Business Processes</h1>
            <a
              href="/processes/builder"
              className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors flex items-center gap-2"
            >
              <Plus size={20} />
              New Process
            </a>
          </div>
          <p className="text-gray-600">
            Create and manage automated business workflows
          </p>
        </div>

        {/* Filters */}
        <div className="bg-white rounded-lg shadow p-4 mb-6">
          <div className="flex gap-4 items-center">
            <div className="flex-1">
              <div className="relative">
                <Search className="absolute left-3 top-3 text-gray-400" size={20} />
                <input
                  type="text"
                  placeholder="Search by name or entity..."
                  value={searchTerm}
                  onChange={(e) => setSearchTerm(e.target.value)}
                  className="w-full pl-10 pr-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                />
              </div>
            </div>
            <select
              title="Filter by status"
              value={filterStatus}
              onChange={(e) => {
                setFilterStatus(e.target.value);
                setOffset(0);
              }}
              className="px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            >
              <option value="all">All Status</option>
              <option value="draft">Draft</option>
              <option value="published">Published</option>
              <option value="archived">Archived</option>
            </select>
          </div>
        </div>

        {/* Error State */}
        {error && (
          <div className="bg-red-50 border-l-4 border-red-500 p-4 mb-6 flex gap-3">
            <AlertCircle className="text-red-500 flex-shrink-0" size={20} />
            <div>
              <h3 className="font-semibold text-red-900">Error</h3>
              <p className="text-red-700">{error}</p>
            </div>
          </div>
        )}

        {/* Loading State */}
        {loading && (
          <div className="flex justify-center items-center py-12">
            <Loader className="animate-spin text-blue-600" size={40} />
            <span className="ml-3 text-gray-600">Loading processes...</span>
          </div>
        )}

        {/* Empty State */}
        {!loading && filteredProcesses.length === 0 && (
          <div className="bg-white rounded-lg shadow p-12 text-center">
            <div className="text-gray-400 mb-4">
              <Plus size={64} className="mx-auto opacity-50" />
            </div>
            <p className="text-gray-600 text-lg mb-2">No business processes found</p>
            <p className="text-gray-500 text-sm mb-4">
              Create a new business process to get started
            </p>
            <a
              href="/processes/builder"
              className="inline-flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
            >
              <Plus size={20} />
              Create Process
            </a>
          </div>
        )}

        {/* Processes Table */}
        {!loading && filteredProcesses.length > 0 && (
          <div className="bg-white rounded-lg shadow overflow-hidden">
            <table className="w-full">
              <thead className="bg-gray-50 border-b border-gray-200">
                <tr>
                  <th className="px-6 py-3 text-left text-sm font-semibold text-gray-900">
                    Process Name
                  </th>
                  <th className="px-6 py-3 text-left text-sm font-semibold text-gray-900">
                    Entity
                  </th>
                  <th className="px-6 py-3 text-left text-sm font-semibold text-gray-900">
                    Steps
                  </th>
                  <th className="px-6 py-3 text-left text-sm font-semibold text-gray-900">
                    Duration
                  </th>
                  <th className="px-6 py-3 text-left text-sm font-semibold text-gray-900">
                    Status
                  </th>
                  <th className="px-6 py-3 text-left text-sm font-semibold text-gray-900">
                    Created By
                  </th>
                  <th className="px-6 py-3 text-left text-sm font-semibold text-gray-900">
                    Actions
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-200">
                {filteredProcesses.map((process) => (
                  <tr key={process.id} className="hover:bg-gray-50 transition-colors">
                    <td className="px-6 py-4">
                      <a
                        href={`/processes/${process.id}`}
                        className="flex items-center gap-2 text-blue-600 hover:text-blue-700 font-medium"
                      >
                        {process.processName}
                        <ChevronRight size={16} />
                      </a>
                    </td>
                    <td className="px-6 py-4 text-sm text-gray-700">
                      {process.entity}
                    </td>
                    <td className="px-6 py-4 text-sm text-gray-700">
                      <span className="px-3 py-1 bg-blue-100 text-blue-800 rounded-full text-sm font-medium">
                        {process.stepsCount}
                      </span>
                    </td>
                    <td className="px-6 py-4 text-sm text-gray-700">
                      {process.totalDurationHours ? (
                        <span>{process.totalDurationHours}h</span>
                      ) : (
                        <span className="text-gray-400">-</span>
                      )}
                    </td>
                    <td className="px-6 py-4 text-sm">
                      <div className="flex gap-2 items-center">
                        {getStatusBadge(process.status)}
                        {getActiveStatus(process.isActive)}
                      </div>
                    </td>
                    <td className="px-6 py-4 text-sm text-gray-700">
                      {process.createdBy}
                    </td>
                    <td className="px-6 py-4 text-sm">
                      <div className="flex gap-2">
                        <a
                          href={`/processes/builder?id=${process.id}`}
                          className="p-2 text-gray-600 hover:bg-gray-100 rounded transition-colors"
                          title="Edit"
                        >
                          <Edit size={18} />
                        </a>
                        <button
                          onClick={() => {}}
                          className="p-2 text-gray-600 hover:bg-gray-100 rounded transition-colors"
                          title="Run"
                        >
                          <Play size={18} />
                        </button>
                        <button
                          onClick={() => handleDeleteProcess(process.id)}
                          disabled={deleting === process.id}
                          className="p-2 text-red-600 hover:bg-red-50 rounded transition-colors disabled:opacity-50"
                          title="Archive"
                        >
                          {deleting === process.id ? (
                            <Loader size={18} className="animate-spin" />
                          ) : (
                            <Archive size={18} />
                          )}
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>

            {/* Pagination */}
            <div className="bg-gray-50 px-6 py-4 border-t border-gray-200 flex items-center justify-between">
              <div className="text-sm text-gray-600">
                Showing <strong>{pageInfo.start}</strong> to <strong>{pageInfo.end}</strong> of{' '}
                <strong>{pageInfo.total}</strong> processes
              </div>
              <div className="flex gap-2">
                <button
                  onClick={() => setOffset(Math.max(0, offset - limit))}
                  disabled={offset === 0}
                  className="px-4 py-2 border border-gray-300 rounded-lg hover:bg-gray-100 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                >
                  Previous
                </button>
                <button
                  onClick={() => setOffset(offset + limit)}
                  disabled={offset + limit >= totalCount}
                  className="px-4 py-2 border border-gray-300 rounded-lg hover:bg-gray-100 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                >
                  Next
                </button>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
};

export default BusinessProcessList;
