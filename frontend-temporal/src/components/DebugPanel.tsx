import React, { useEffect, useState, useCallback } from 'react'

function getTenantHeaders() {
  try {
    const t = window.localStorage.getItem('selected_tenant')
    const d = window.localStorage.getItem('selected_datasource')
    const tenant = t ? JSON.parse(t) : null
    const datasource = d ? JSON.parse(d) : null
    const headers: Record<string, string> = {}
    if (tenant && tenant.id) headers['X-Tenant-ID'] = tenant.id
    if (datasource && datasource.id) headers['X-Tenant-Datasource-ID'] = datasource.id
    return headers
  } catch (err) {
    return {}
  }
}

export default function DebugPanel() {
  const [events, setEvents] = useState<any[]>([])
  const [publishResp, setPublishResp] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  const headersBase = getTenantHeaders()

  const fetchEvents = useCallback(async () => {
    try {
      const res = await fetch(`/api/v1/triggers/events`, { headers: headersBase })
      if (res.ok) {
        const body = await res.json()
        setEvents(body || [])
      } else {
        setEvents([])
      }
    } catch (err) {
      setEvents([])
    }
  }, [headersBase])

  useEffect(() => {
    fetchEvents()
    const t = setInterval(() => {
      fetchEvents()
    }, 5000)
    return () => clearInterval(t)
  }, [fetchEvents])

  const publishTest = async () => {
    setLoading(true)
    setPublishResp(null)
    try {
      const res = await fetch(`/api/_debug/publish-event`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', ...headersBase },
        body: JSON.stringify({ type: 'ui.debug.test', payload: { ts: Date.now() } })
      })
      const text = await res.text()
      setPublishResp(`status=${res.status} body=${text}`)
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
        <button onClick={publishTest} disabled={loading} className="px-3 py-1 rounded bg-indigo-600 text-white disabled:opacity-60">{loading ? 'Publishing...' : 'Publish test event'}</button>
        {publishResp && <div className="mt-2 text-sm"><strong>Response:</strong> {publishResp}</div>}
      </div>
    </div>
  )
}
