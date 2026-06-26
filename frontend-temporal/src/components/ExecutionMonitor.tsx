import React, { useState, useEffect } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useABAC } from '../hooks/useABAC'

const fetchExecutions = async () => {
  const res = await fetch('/api/temporal/executions')
  return res.ok ? res.json() : []
}

export default function ExecutionMonitor() {
  const { evaluate } = useABAC()
  const [selectedExecution, setSelectedExecution] = useState<any | null>(null)
  const [autoRefresh, setAutoRefresh] = useState(true)
  const { data: executions = [], refetch } = useQuery(['executions'], fetchExecutions, {
    refetchInterval: autoRefresh ? 5000 : false, // Refresh every 5 seconds if enabled
  })

  // Check permissions on component mount
  useEffect(() => {
    const checkPermissions = async () => {
      const canView = await evaluate('read', 'executions')
      if (!canView) {
        alert('Access denied: Insufficient permissions to view executions')
      }
    }
    checkPermissions()
  }, [evaluate])

  const handleTerminate = async (executionId: string) => {
    const canTerminate = await evaluate('delete', `execution-${executionId}`)
    if (!canTerminate) {
      alert('Access denied: Insufficient permissions to terminate executions')
      return
    }

    if (!confirm('Are you sure you want to terminate this execution?')) return

    try {
      const response = await fetch(`/api/temporal/executions/${executionId}/terminate`, {
        method: 'POST'
      })
      if (response.ok) {
        alert('Execution terminated successfully')
        refetch()
      } else {
        alert('Failed to terminate execution')
      }
    } catch (error) {
      console.error('Error terminating execution:', error)
      alert('Error terminating execution')
    }
  }

  const getStatusColor = (status: string) => {
    switch (status?.toLowerCase()) {
      case 'running': return 'text-blue-600'
      case 'completed': return 'text-green-600'
      case 'failed': return 'text-red-600'
      case 'terminated': return 'text-orange-600'
      default: return 'text-gray-600'
    }
  }

  return (
    <div className="p-4">
      <div className="flex justify-between items-center mb-4">
        <h2 className="text-xl font-semibold">Workflow Executions</h2>
        <div className="flex items-center gap-2">
          <label className="flex items-center gap-2">
            <input
              type="checkbox"
              checked={autoRefresh}
              onChange={(e) => setAutoRefresh(e.target.checked)}
              className="form-checkbox"
            />
            <span className="text-sm">Auto-refresh</span>
          </label>
          <button
            onClick={() => refetch()}
            className="px-3 py-1 bg-blue-600 text-white rounded text-sm hover:bg-blue-700"
          >
            Refresh
          </button>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        <div className="bg-white rounded-lg shadow">
          <div className="p-4 border-b">
            <h3 className="font-medium">Executions ({executions.length})</h3>
          </div>
          <div className="max-h-96 overflow-y-auto">
            <table className="w-full">
              <thead className="bg-gray-50 sticky top-0">
                <tr>
                  <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase">ID</th>
                  <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase">Status</th>
                  <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase">Start Time</th>
                  <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-200">
                {executions.map((e: any) => (
                  <tr
                    key={e.id}
                    className={`hover:bg-gray-50 cursor-pointer ${selectedExecution?.id === e.id ? 'bg-blue-50' : ''}`}
                    onClick={() => setSelectedExecution(e)}
                  >
                    <td className="px-4 py-2 text-sm font-mono">{e.id?.slice(-8)}</td>
                    <td className={`px-4 py-2 text-sm font-medium ${getStatusColor(e.status)}`}>
                      {e.status}
                    </td>
                    <td className="px-4 py-2 text-sm text-gray-600">
                      {e.startTime ? new Date(e.startTime).toLocaleString() : 'N/A'}
                    </td>
                    <td className="px-4 py-2 text-sm">
                      {e.status === 'Running' && (
                        <button
                          onClick={(ev) => { ev.stopPropagation(); handleTerminate(e.id) }}
                          className="px-2 py-1 bg-red-600 text-white rounded text-xs hover:bg-red-700"
                        >
                          Terminate
                        </button>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
            {executions.length === 0 && (
              <div className="p-8 text-center text-gray-500">
                No executions found
              </div>
            )}
          </div>
        </div>

        {selectedExecution && (
          <div className="bg-white rounded-lg shadow">
            <div className="p-4 border-b">
              <h3 className="font-medium">Execution Details</h3>
            </div>
            <div className="p-4 space-y-3">
              <div>
                <label className="text-sm font-medium text-gray-500">Execution ID</label>
                <p className="font-mono text-sm">{selectedExecution.id}</p>
              </div>
              <div>
                <label className="text-sm font-medium text-gray-500">Workflow ID</label>
                <p className="font-mono text-sm">{selectedExecution.workflowId}</p>
              </div>
              <div>
                <label className="text-sm font-medium text-gray-500">Status</label>
                <p className={`text-sm font-medium ${getStatusColor(selectedExecution.status)}`}>
                  {selectedExecution.status}
                </p>
              </div>
              <div>
                <label className="text-sm font-medium text-gray-500">Start Time</label>
                <p className="text-sm">
                  {selectedExecution.startTime ? new Date(selectedExecution.startTime).toLocaleString() : 'N/A'}
                </p>
              </div>
              <div>
                <label className="text-sm font-medium text-gray-500">End Time</label>
                <p className="text-sm">
                  {selectedExecution.endTime ? new Date(selectedExecution.endTime).toLocaleString() : 'N/A'}
                </p>
              </div>
              {selectedExecution.error && (
                <div>
                  <label className="text-sm font-medium text-gray-500">Error</label>
                  <pre className="text-xs bg-red-50 p-2 rounded mt-1 overflow-auto max-h-32">
                    {selectedExecution.error}
                  </pre>
                </div>
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
