import React, { useState, useEffect } from 'react'
import { Layout, Card, Button, Spinner, Badge } from '../components/Layout'
import { RealTimeEvent } from '../types'
import { useWebSocket } from '../hooks/useWebSocket'

export function LiveFeed() {
  const tenantId = localStorage.getItem('tenant_id') || 'default'
  const [events, setEvents] = useState<RealTimeEvent[]>([])
  const [filter, setFilter] = useState<'all' | 'incident' | 'action' | 'sla'>('all')
  const [isPaused, setIsPaused] = useState(false)
  const [autoScroll, setAutoScroll] = useState(true)

  const { isConnected, subscribe, unsubscribe } = useWebSocket({
    tenantId,
    regions: ['us-east-1', 'eu-west-1', 'apac-1'],
    onEvent: (event) => {
      if (!isPaused) {
        setEvents(prev => [event, ...prev].slice(0, 500)) // Keep last 500 events
      }
    }
  })

  useEffect(() => {
    subscribe('us-east-1')
    subscribe('eu-west-1')
    subscribe('apac-1')

    return () => {
      unsubscribe('us-east-1')
      unsubscribe('eu-west-1')
      unsubscribe('apac-1')
    }
  }, [subscribe, unsubscribe])

  const filteredEvents = events.filter(e => {
    if (filter === 'all') return true
    return e.type === filter
  })

  const getEventColor = (type: string) => {
    switch (type) {
      case 'incident':
        return 'bg-red-50 border-red-200'
      case 'action':
        return 'bg-blue-50 border-blue-200'
      case 'sla':
        return 'bg-yellow-50 border-yellow-200'
      default:
        return 'bg-slate-50 border-slate-200'
    }
  }

  const getEventIcon = (type: string) => {
    switch (type) {
      case 'incident':
        return '⚠️'
      case 'action':
        return '▶️'
      case 'sla':
        return '📊'
      default:
        return '•'
    }
  }

  return (
    <Layout sidebar={<FeedSidebar />} header={<FeedHeader isConnected={isConnected} />}>
      <div className="space-y-6">
        {/* Controls */}
        <Card className="flex items-center justify-between p-4">
          <div className="flex gap-3">
            <Button
              variant={filter === 'all' ? 'primary' : 'secondary'}
              onClick={() => setFilter('all')}
              size="sm"
            >
              All
            </Button>
            <Button
              variant={filter === 'incident' ? 'primary' : 'secondary'}
              onClick={() => setFilter('incident')}
              size="sm"
            >
              Incidents
            </Button>
            <Button
              variant={filter === 'action' ? 'primary' : 'secondary'}
              onClick={() => setFilter('action')}
              size="sm"
            >
              Actions
            </Button>
            <Button
              variant={filter === 'sla' ? 'primary' : 'secondary'}
              onClick={() => setFilter('sla')}
              size="sm"
            >
              SLA
            </Button>
          </div>

          <div className="flex gap-3">
            <Button
              variant={autoScroll ? 'primary' : 'secondary'}
              onClick={() => setAutoScroll(!autoScroll)}
              size="sm"
            >
              {autoScroll ? '✓' : ''} Auto-scroll
            </Button>
            <Button
              variant={isPaused ? 'danger' : 'secondary'}
              onClick={() => setIsPaused(!isPaused)}
              size="sm"
            >
              {isPaused ? 'Resume' : 'Pause'}
            </Button>
            <Button
              variant="secondary"
              onClick={() => setEvents([])}
              size="sm"
            >
              Clear
            </Button>
          </div>
        </Card>

        {/* Status */}
        <div className="px-4 py-2 flex items-center gap-2 text-sm">
          <div className={`w-2 h-2 rounded-full ${isConnected ? 'bg-green-600' : 'bg-red-600'}`}></div>
          <span className={isConnected ? 'text-green-700' : 'text-red-700'}>
            {isConnected ? 'Connected' : 'Disconnected'}
          </span>
          <span className="text-slate-600">
            • {filteredEvents.length} events
          </span>
        </div>

        {/* Events Stream */}
        <div id="event-stream" className="space-y-2 max-h-[calc(100vh-300px)] overflow-y-auto">
          {filteredEvents.length > 0 ? (
            filteredEvents.map((event, index) => (
              <EventRow key={`${event.id}-${index}`} event={event} color={getEventColor(event.type)} icon={getEventIcon(event.type)} />
            ))
          ) : (
            <div className="text-center py-12">
              {!isConnected ? (
                <p className="text-slate-500">Connecting to event stream...</p>
              ) : (
                <p className="text-slate-500">No events yet. Waiting for live updates...</p>
              )}
            </div>
          )}
        </div>
      </div>
    </Layout>
  )
}

function EventRow({ event, color, icon }: { event: RealTimeEvent; color: string; icon: string }) {
  const timestamp = new Date(event.timestamp).toLocaleTimeString('en-US', {
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit'
  })

  return (
    <div className={`p-4 border rounded-lg transition-all hover:shadow-md ${color}`}>
      <div className="flex items-start justify-between">
        <div className="flex-1">
          <div className="flex items-center gap-2 mb-2">
            <span className="text-xl">{icon}</span>
            <span className="font-mono text-sm font-bold text-slate-700">{event.type.toUpperCase()}</span>
            <Badge status={getEventStatusBadge(event.type)} />
          </div>
          <p className="text-sm text-slate-700 mb-1">{event.message}</p>
          {event.details && (
            <details className="text-xs text-slate-600 cursor-pointer">
              <summary className="select-none hover:underline">Details</summary>
              <pre className="mt-2 p-2 bg-white bg-opacity-50 rounded overflow-x-auto text-xs">
                {JSON.stringify(event.details, null, 2)}
              </pre>
            </details>
          )}
        </div>
        <div className="text-xs text-slate-500 whitespace-nowrap ml-4">
          {timestamp}
        </div>
      </div>
      <div className="mt-2 flex items-center gap-2 text-xs text-slate-600">
        <span>Chain: {event.chain_id}</span>
        <span>•</span>
        <span>Region: {event.region}</span>
        {event.severity && (
          <>
            <span>•</span>
            <span className={`font-medium ${event.severity > 0.7 ? 'text-red-600' : 'text-yellow-600'}`}>
              Severity: {(event.severity * 100).toFixed(0)}%
            </span>
          </>
        )}
      </div>
    </div>
  )
}

function getEventStatusBadge(type: string): 'resolved' | 'active' | 'pending' {
  switch (type) {
    case 'incident':
      return 'active'
    case 'action':
      return 'pending'
    case 'sla':
      return 'resolved'
    default:
      return 'pending'
  }
}

function FeedSidebar() {
  return (
    <nav className="p-6 space-y-4">
      <a href="/" className="block px-4 py-2 rounded-lg hover:bg-brand-light">
        Dashboard
      </a>
      <a href="/chains" className="block px-4 py-2 rounded-lg hover:bg-brand-light">
        Chains
      </a>
      <a href="/feed" className="block px-4 py-2 rounded-lg bg-blue-600 text-white">
        Live Feed
      </a>
      <a href="/reports" className="block px-4 py-2 rounded-lg hover:bg-brand-light">
        Reports
      </a>
    </nav>
  )
}

function FeedHeader({ isConnected }: { isConnected: boolean }) {
  return (
    <div className="px-8 py-4 flex items-center justify-between border-b border-slate-200">
      <h1 className="text-2xl font-bold text-slate-900">Live Event Stream</h1>
      <div className="flex items-center gap-2">
        <div className={`w-3 h-3 rounded-full animate-pulse ${isConnected ? 'bg-emerald-600' : 'bg-slate-400'}`}></div>
        <span className="text-sm font-medium">{isConnected ? 'Live' : 'Offline'}</span>
      </div>
    </div>
  )
}
