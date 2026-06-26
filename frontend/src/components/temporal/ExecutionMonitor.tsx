import { useEffect, useState } from 'react'
import temporalService from '../../../services/temporalService'

export default function ExecutionMonitor() {
  const [executions, setExecutions] = useState<any[]>([])
  const [loading, setLoading] = useState(false)

  const fetchExecutions = async () => {
    setLoading(true)
    try {
      const json = await temporalService.listExecutions()
      setExecutions(json || [])
    } catch (err) {
      console.error('failed to fetch executions', err)
      setExecutions([])
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchExecutions()
    const t = setInterval(fetchExecutions, 3000)
    return () => clearInterval(t)
  }, [])

  return (
    <div className="p-3">
      <h3 className="text-lg font-medium">Temporal Executions</h3>
      {loading && <div className="text-sm text-gray-500">Loading…</div>}
      <div className="mt-2 space-y-2">
        {executions.length === 0 && !loading && <div className="text-sm text-gray-500">No workflows</div>}
        {executions.map((e: any, i: number) => (
          <div key={String(e.id || i)} className="p-2 border rounded bg-white">
            <div className="text-sm font-mono">{String(e.workflow_id || e.workflowId || e.id)}</div>
            <div className="text-xs text-gray-600">{String(e.status || e.state || 'unknown')}</div>
          </div>
        ))}
      </div>
    </div>
  )
}
