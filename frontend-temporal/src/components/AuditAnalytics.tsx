import React, { useState, useEffect } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useABAC } from '../hooks/useABAC'

interface AuditEntry {
  id: string
  timestamp: string
  actorId: string
  action: string
  workflowId?: string
  runId?: string
  status: 'success' | 'failed'
  errorMessage?: string
  input?: any
  reason?: string
}

interface AnalyticsSummary {
  totalActions: number
  successRate: number
  topActions: { action: string; count: number }[]
  recentActivity: AuditEntry[]
  errorRate: number
}

const fetchAuditSummary = async (): Promise<AnalyticsSummary> => {
  // This would ideally be a dedicated audit summary endpoint
  // For now, we'll aggregate from available data
  const res = await fetch('/api/temporal/audit/summary')
  return res.ok ? res.json() : {
    totalActions: 0,
    successRate: 0,
    topActions: [],
    recentActivity: [],
    errorRate: 0
  }
}

const fetchAuditEntries = async (filters: any): Promise<AuditEntry[]> => {
  const params = new URLSearchParams()
  if (filters.startDate) params.append('startDate', filters.startDate)
  if (filters.endDate) params.append('endDate', filters.endDate)
  if (filters.action) params.append('action', filters.action)
  if (filters.actorId) params.append('actorId', filters.actorId)

  const res = await fetch(`/api/temporal/audit?${params}`)
  return res.ok ? res.json() : []
}

export const AuditAnalytics: React.FC = () => {
  const { evaluate } = useABAC()
  const [selectedEntry, setSelectedEntry] = useState<AuditEntry | null>(null)
  const [filters, setFilters] = useState({
    startDate: '',
    endDate: '',
    action: '',
    actorId: ''
  })

  const { data: summary, isLoading: summaryLoading } = useQuery(['audit-summary'], fetchAuditSummary)
  const { data: auditEntries = [], isLoading: entriesLoading, refetch: refetchEntries } = useQuery(
    ['audit-entries', filters],
    () => fetchAuditEntries(filters)
  )

  // Check permissions on component mount
  useEffect(() => {
    const checkPermissions = async () => {
      const canView = await evaluate('read', 'audit')
      if (!canView) {
        alert('Access denied: Insufficient permissions to view audit logs')
      }
    }
    checkPermissions()
  }, [evaluate])

  const handleFilterChange = (key: string, value: string) => {
    setFilters(prev => ({ ...prev, [key]: value }))
  }

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'success': return 'text-green-600 bg-green-50'
      case 'failed': return 'text-red-600 bg-red-50'
      default: return 'text-gray-600 bg-gray-50'
    }
  }

  const getActionColor = (action: string) => {
    const colors: { [key: string]: string } = {
      'create': 'bg-blue-100 text-blue-800',
      'update': 'bg-yellow-100 text-yellow-800',
      'delete': 'bg-red-100 text-red-800',
      'read': 'bg-green-100 text-green-800',
      'signal': 'bg-purple-100 text-purple-800',
      'terminate': 'bg-orange-100 text-orange-800'
    }
    return colors[action] || 'bg-gray-100 text-gray-800'
  }

  if (summaryLoading) {
    return <div className="p-4">Loading audit analytics...</div>
  }

  return (
    <div className="p-4 max-w-7xl mx-auto">
      <div className="mb-6">
        <h2 className="text-2xl font-semibold mb-2">Audit Analytics Dashboard</h2>
        <p className="text-gray-600">Comprehensive audit logs and analytics for Temporal workflow operations</p>
      </div>

      {/* Summary Cards */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
        <div className="bg-white p-4 rounded-lg shadow">
          <div className="text-2xl font-bold text-blue-600">{summary?.totalActions || 0}</div>
          <div className="text-sm text-gray-600">Total Actions</div>
        </div>
        <div className="bg-white p-4 rounded-lg shadow">
          <div className="text-2xl font-bold text-green-600">{summary?.successRate || 0}%</div>
          <div className="text-sm text-gray-600">Success Rate</div>
        </div>
        <div className="bg-white p-4 rounded-lg shadow">
          <div className="text-2xl font-bold text-red-600">{summary?.errorRate || 0}%</div>
          <div className="text-sm text-gray-600">Error Rate</div>
        </div>
        <div className="bg-white p-4 rounded-lg shadow">
          <div className="text-2xl font-bold text-purple-600">{summary?.recentActivity?.length || 0}</div>
          <div className="text-sm text-gray-600">Recent Activities</div>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Top Actions Chart */}
        <div className="bg-white rounded-lg shadow">
          <div className="p-4 border-b">
            <h3 className="font-medium">Top Actions</h3>
          </div>
          <div className="p-4">
            {summary?.topActions?.length ? (
              <div className="space-y-3">
                {summary.topActions.map((item, index) => (
                  <div key={item.action} className="flex items-center justify-between">
                    <div className="flex items-center gap-2">
                      <span className="text-sm font-medium text-gray-600">#{index + 1}</span>
                      <span className={`px-2 py-1 rounded-full text-xs font-medium ${getActionColor(item.action)}`}>
                        {item.action}
                      </span>
                    </div>
                    <span className="text-sm font-medium">{item.count}</span>
                  </div>
                ))}
              </div>
            ) : (
              <div className="text-center text-gray-500 py-8">No action data available</div>
            )}
          </div>
        </div>

        {/* Filters */}
        <div className="bg-white rounded-lg shadow">
          <div className="p-4 border-b">
            <h3 className="font-medium">Filters</h3>
          </div>
          <div className="p-4 space-y-3">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Start Date</label>
              <input
                type="date"
                value={filters.startDate}
                onChange={(e) => handleFilterChange('startDate', e.target.value)}
                className="w-full px-3 py-2 border rounded"
                aria-label="Start date filter"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">End Date</label>
              <input
                type="date"
                value={filters.endDate}
                onChange={(e) => handleFilterChange('endDate', e.target.value)}
                className="w-full px-3 py-2 border rounded"
                aria-label="End date filter"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Action</label>
              <select
                value={filters.action}
                onChange={(e) => handleFilterChange('action', e.target.value)}
                className="w-full px-3 py-2 border rounded"
                aria-label="Action filter"
              >
                <option value="">All Actions</option>
                <option value="create">Create</option>
                <option value="update">Update</option>
                <option value="delete">Delete</option>
                <option value="read">Read</option>
                <option value="signal">Signal</option>
                <option value="terminate">Terminate</option>
              </select>
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Actor ID</label>
              <input
                type="text"
                value={filters.actorId}
                onChange={(e) => handleFilterChange('actorId', e.target.value)}
                className="w-full px-3 py-2 border rounded"
                placeholder="Filter by actor"
                aria-label="Actor ID filter"
              />
            </div>
            <button
              onClick={() => refetchEntries()}
              className="w-full px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700"
            >
              Apply Filters
            </button>
          </div>
        </div>

        {/* Recent Activity */}
        <div className="bg-white rounded-lg shadow">
          <div className="p-4 border-b">
            <h3 className="font-medium">Recent Activity</h3>
          </div>
          <div className="max-h-64 overflow-y-auto">
            {summary?.recentActivity?.length ? (
              <div className="divide-y divide-gray-200">
                {summary.recentActivity.map((entry) => (
                  <div
                    key={entry.id}
                    className="p-3 hover:bg-gray-50 cursor-pointer"
                    onClick={() => setSelectedEntry(entry)}
                  >
                    <div className="flex items-center justify-between mb-1">
                      <span className={`px-2 py-1 rounded-full text-xs font-medium ${getActionColor(entry.action)}`}>
                        {entry.action}
                      </span>
                      <span className={`px-2 py-1 rounded-full text-xs ${getStatusColor(entry.status)}`}>
                        {entry.status}
                      </span>
                    </div>
                    <div className="text-xs text-gray-600">
                      {entry.actorId} • {new Date(entry.timestamp).toLocaleString()}
                    </div>
                  </div>
                ))}
              </div>
            ) : (
              <div className="text-center text-gray-500 py-8">No recent activity</div>
            )}
          </div>
        </div>
      </div>

      {/* Detailed Audit Log */}
      <div className="mt-6 bg-white rounded-lg shadow">
        <div className="p-4 border-b">
          <h3 className="font-medium">Detailed Audit Log ({auditEntries.length} entries)</h3>
        </div>
        <div className="max-h-96 overflow-y-auto">
          <table className="w-full">
            <thead className="bg-gray-50 sticky top-0">
              <tr>
                <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase">Timestamp</th>
                <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase">Actor</th>
                <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase">Action</th>
                <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase">Workflow</th>
                <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase">Status</th>
                <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase">Details</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-200">
              {entriesLoading ? (
                <tr>
                  <td colSpan={6} className="px-4 py-8 text-center text-gray-500">
                    Loading audit entries...
                  </td>
                </tr>
              ) : auditEntries.length ? (
                auditEntries.map((entry) => (
                  <tr
                    key={entry.id}
                    className="hover:bg-gray-50 cursor-pointer"
                    onClick={() => setSelectedEntry(entry)}
                  >
                    <td className="px-4 py-2 text-sm text-gray-600">
                      {new Date(entry.timestamp).toLocaleString()}
                    </td>
                    <td className="px-4 py-2 text-sm font-mono">
                      {entry.actorId.slice(-8)}
                    </td>
                    <td className="px-4 py-2 text-sm">
                      <span className={`px-2 py-1 rounded-full text-xs font-medium ${getActionColor(entry.action)}`}>
                        {entry.action}
                      </span>
                    </td>
                    <td className="px-4 py-2 text-sm font-mono">
                      {entry.workflowId?.slice(-8) || 'N/A'}
                    </td>
                    <td className="px-4 py-2 text-sm">
                      <span className={`px-2 py-1 rounded-full text-xs ${getStatusColor(entry.status)}`}>
                        {entry.status}
                      </span>
                    </td>
                    <td className="px-4 py-2 text-sm text-gray-600">
                      {entry.errorMessage ? 'Error' : entry.reason || 'Success'}
                    </td>
                  </tr>
                ))
              ) : (
                <tr>
                  <td colSpan={6} className="px-4 py-8 text-center text-gray-500">
                    No audit entries found
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </div>

      {/* Entry Details Modal */}
      {selectedEntry && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50">
          <div className="bg-white rounded-lg max-w-2xl w-full max-h-[80vh] overflow-y-auto">
            <div className="p-4 border-b flex justify-between items-center">
              <h3 className="font-medium">Audit Entry Details</h3>
              <button
                onClick={() => setSelectedEntry(null)}
                className="text-gray-400 hover:text-gray-600"
              >
                ✕
              </button>
            </div>
            <div className="p-4 space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700">Entry ID</label>
                  <p className="font-mono text-sm">{selectedEntry.id}</p>
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700">Timestamp</label>
                  <p className="text-sm">{new Date(selectedEntry.timestamp).toLocaleString()}</p>
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700">Actor ID</label>
                  <p className="font-mono text-sm">{selectedEntry.actorId}</p>
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700">Action</label>
                  <p className="text-sm">
                    <span className={`px-2 py-1 rounded-full text-xs font-medium ${getActionColor(selectedEntry.action)}`}>
                      {selectedEntry.action}
                    </span>
                  </p>
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700">Workflow ID</label>
                  <p className="font-mono text-sm">{selectedEntry.workflowId || 'N/A'}</p>
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700">Status</label>
                  <p className="text-sm">
                    <span className={`px-2 py-1 rounded-full text-xs ${getStatusColor(selectedEntry.status)}`}>
                      {selectedEntry.status}
                    </span>
                  </p>
                </div>
              </div>

              {selectedEntry.reason && (
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">Reason</label>
                  <p className="text-sm bg-gray-50 p-2 rounded">{selectedEntry.reason}</p>
                </div>
              )}

              {selectedEntry.errorMessage && (
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">Error Message</label>
                  <p className="text-sm bg-red-50 p-2 rounded text-red-700">{selectedEntry.errorMessage}</p>
                </div>
              )}

              {selectedEntry.input && (
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">Input Data</label>
                  <pre className="text-xs bg-gray-50 p-2 rounded overflow-auto max-h-32">
                    {JSON.stringify(selectedEntry.input, null, 2)}
                  </pre>
                </div>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
