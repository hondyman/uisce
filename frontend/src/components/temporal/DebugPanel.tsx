import { useCallback, useEffect, useState } from 'react'
import temporalService from '../../../services/temporalService'
import { useABAC } from '../../abac'

export default function DebugPanel() {
  const [events, setEvents] = useState<any[]>([])
  const [publishResp, setPublishResp] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)
  const [canPublish, setCanPublish] = useState<boolean | null>(null)

  const { canExecute } = useABAC()

  const fetchEvents = useCallback(async () => {
    try {
      const body = await temporalService.fetchTriggerEvents()
      setEvents(body || [])
    } catch (err) {
      setEvents([])
    }
  }, [])

  useEffect(() => {
    fetchEvents()
    const t = setInterval(() => {
      fetchEvents()
    }, 5000)
    return () => clearInterval(t)
  }, [fetchEvents])

  useEffect(() => {
    let mounted = true
    ;(async () => {
      try {
        const allowed = await canExecute('temporal.admin', 'temporal')
        if (mounted) setCanPublish(allowed)
      } catch (err) {
        if (mounted) setCanPublish(false)
      }
    })()
    return () => { mounted = false }
  }, [canExecute])

  const publishTest = async () => {
    setLoading(true)
    setPublishResp(null)
    try {
      const res = await temporalService.publishTestEvent({ type: 'ui.debug.test', payload: { ts: Date.now() } })
      setPublishResp(`status=${res.status} body=${JSON.stringify(res.body)}`)
    } catch (err) {
      setPublishResp(`error: ${String(err)}`)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="p-3">
      <h2 className="text-lg font-semibold">Debug Panel</h2>
      <p className="text-sm text-gray-600">Live Events (auto-refresh every 5s)</p>
      <div className="max-h-56 overflow-auto bg-white p-2 mt-2 border rounded">
        {events.length === 0 ? <div className="text-sm text-gray-500">No events</div> : events.map((ev: any, i: number) => (
          <div key={i} className="border-b border-gray-200 p-2 text-sm">{JSON.stringify(ev)}</div>
        ))}
      </div>

      <div className="mt-3">
        <button onClick={publishTest} disabled={loading || canPublish === false} className="px-3 py-1 rounded bg-indigo-600 text-white disabled:opacity-60">{loading ? 'Publishing...' : 'Publish test event'}</button>
        {canPublish === false && <div className="mt-2 text-sm text-red-600">You do not have permission to publish test events.</div>}
        {publishResp && <div className="mt-2 text-sm"><strong>Response:</strong> {publishResp}</div>}
      </div>
    </div>
  )
}
