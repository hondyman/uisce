import { useEffect, useState, useCallback } from 'react'
import temporalService from '../../services/temporalService'

export default function LiveEventsWidget({ onSelect }: { onSelect?: (ev: any) => void }) {
  const [events, setEvents] = useState<any[]>([])

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
