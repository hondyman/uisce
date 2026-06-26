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

export default function LiveEventsWidget({ onSelect }: { onSelect?: (ev: any) => void }) {
  const [events, setEvents] = useState<any[]>([])
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
    const t = setInterval(fetchEvents, 5000)
    return () => clearInterval(t)
  }, [fetchEvents])

  return (
    <div className="p-2">
      <h4 className="text-sm font-medium">Live Triggers</h4>
      <div className="max-h-56 overflow-auto bg-white p-1 mt-2 border rounded">
        {events.length === 0 ? <div className="text-sm text-gray-500">No events</div> : events.map((ev: any, i: number) => (
          <div key={i} className="border-b border-gray-200 p-2 text-sm cursor-pointer hover:bg-gray-50" onClick={() => onSelect && onSelect(ev)}>{ev.type || JSON.stringify(ev)}</div>
        ))}
      </div>
    </div>
  )
}
