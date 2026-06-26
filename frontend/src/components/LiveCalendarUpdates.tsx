import React, { useState } from 'react';
import {
  useCalendarSubscription,
  CalendarEvent,
  IngestionEvent,
  ConflictEvent,
} from '../hooks/useCalendarSubscription';

/**
 * LiveCalendarUpdates Component
 *
 * Displays real-time calendar updates from Redpanda event stream
 * Shows:
 * - Latest calendar update with holiday info
 * - Connection status and update rate
 * - Recent calendar event history (last 100)
 * - Ingestion lifecycle events
 * - Data conflicts requiring attention
 *
 * Features:
 * - Color-coded events (business days vs holidays)
 * - Confidence score visualization
 * - Source system attribution
 * - Auto-refresh of statistics
 */
export const LiveCalendarUpdates: React.FC = () => {
  const {
    calendarEvents,
    ingestionEvents,
    conflictEvents,
    isConnected,
    error,
    lastCalendarUpdate,
    lastIngestionUpdate,
    stats,
    clearEvents,
  } = useCalendarSubscription();

  const [expandedTab, setExpandedTab] = useState<'calendar' | 'ingestion' | 'conflicts'>('calendar');

  if (error) {
    return (
      <div className="p-4 bg-red-50 border border-red-200 rounded-lg">
        <div className="flex items-center gap-2 mb-2">
          <span className="text-2xl">⚠️</span>
          <h2 className="text-lg font-bold text-red-900">Connection Error</h2>
        </div>
        <p className="text-sm text-red-700 mb-4">{error.message}</p>
        <p className="text-sm text-gray-600">
          Check that Redpanda is running and GraphQL subscriptions are enabled.
        </p>
      </div>
    );
  }

  return (
    <div className="w-full bg-white rounded-lg border border-gray-200 shadow-sm overflow-hidden">
      {/* Header */}
      <div className="bg-gradient-to-r from-blue-50 to-indigo-50 border-b border-gray-200 p-4">
        <div className="flex items-center justify-between mb-3">
          <div className="flex items-center gap-3">
            <div className="text-2xl">📡</div>
            <div>
              <h2 className="text-lg font-bold text-gray-900">Real-Time Calendar Updates</h2>
              <p className="text-sm text-gray-500">Live event stream from Redpanda</p>
            </div>
          </div>
          <div className="text-right">
            <div className="flex items-center gap-2 justify-end mb-1">
              <div
                className={`w-3 h-3 rounded-full animate-pulse ${
                  isConnected ? 'bg-green-500' : 'bg-gray-300'
                }`}
              />
              <span className="text-sm font-medium text-gray-700">
                {isConnected ? '✅ Connected' : '❌ Disconnected'}
              </span>
            </div>
            <div className="text-xs text-gray-500">
              {stats.updateRate.toFixed(2)} events/sec
            </div>
          </div>
        </div>

        {/* Statistics Row */}
        <div className="grid grid-cols-3 gap-3">
          <div className="bg-white rounded p-2 border border-gray-100">
            <p className="text-xs text-gray-500">Calendar Updates</p>
            <p className="text-lg font-bold text-blue-600">{stats.totalCalendarEvents}</p>
          </div>
          <div className="bg-white rounded p-2 border border-gray-100">
            <p className="text-xs text-gray-500">Ingestions</p>
            <p className="text-lg font-bold text-purple-600">{stats.totalIngestionEvents}</p>
          </div>
          <div className="bg-white rounded p-2 border border-gray-100">
            <p className="text-xs text-gray-500">Conflicts</p>
            <p className="text-lg font-bold text-red-600">{stats.totalConflicts}</p>
          </div>
        </div>
      </div>

      {/* Latest Update Alert */}
      {lastCalendarUpdate && (
        <div
          className={`border-b border-gray-200 p-4 ${
            lastCalendarUpdate.isBusinessDay
              ? 'bg-blue-50 border-l-4 border-blue-500'
              : 'bg-amber-50 border-l-4 border-amber-500'
          }`}
        >
          <p className="text-xs font-medium text-gray-600 mb-2">Latest Update</p>
          <div className="flex items-center justify-between">
            <div>
              <p className="text-lg font-bold text-gray-900">{lastCalendarUpdate.calendarDate}</p>
              <div className="flex items-center gap-2 mt-1">
                <span className="text-sm font-medium text-gray-700">{lastCalendarUpdate.region}</span>
                {lastCalendarUpdate.exchange && (
                  <>
                    <span className="text-gray-300">•</span>
                    <span className="text-sm text-gray-600">{lastCalendarUpdate.exchange}</span>
                  </>
                )}
              </div>
            </div>
            <div className="text-right">
              <p className="text-2xl mb-1">
                {lastCalendarUpdate.isBusinessDay ? '💼' : '🎉'}
              </p>
              <p className="text-xs font-medium text-gray-600">
                {lastCalendarUpdate.isBusinessDay ? 'Business Day' : 'Holiday'}
              </p>
              {lastCalendarUpdate.holidayName && (
                <p className="text-sm font-bold text-gray-900 mt-1">{lastCalendarUpdate.holidayName}</p>
              )}
            </div>
          </div>
          <div className="flex items-center gap-4 mt-3 pt-3 border-t border-gray-200 border-opacity-50">
            <div className="flex items-center gap-1">
              <span className="text-xs text-gray-500">Confidence</span>
              <div className="flex items-center gap-1">
                <div className="w-24 h-2 bg-gray-200 rounded-full overflow-hidden">
                  <div
                    className="h-full bg-green-500 rounded-full"
                    style={{ width: `${lastCalendarUpdate.confidenceScore}%` }}
                  />
                </div>
                <span className="text-xs font-bold text-gray-900">
                  {lastCalendarUpdate.confidenceScore}%
                </span>
              </div>
            </div>
            <span className="text-xs text-gray-500">•</span>
            <div className="flex items-center gap-1">
              <span className="text-xs text-gray-500">From</span>
              <span className="text-xs font-medium bg-gray-100 px-2 py-0.5 rounded">
                {lastCalendarUpdate.sourceSystem}
              </span>
            </div>
          </div>
        </div>
      )}

      {/* Tabs */}
      <div className="border-b border-gray-200 flex">
        <button
          onClick={() => setExpandedTab('calendar')}
          className={`flex-1 px-4 py-3 text-sm font-medium border-b-2 transition-colors ${
            expandedTab === 'calendar'
              ? 'border-blue-500 text-blue-600 bg-blue-50'
              : 'border-transparent text-gray-600 hover:text-gray-900'
          }`}
        >
          📅 Calendar Events ({stats.totalCalendarEvents})
        </button>
        <button
          onClick={() => setExpandedTab('ingestion')}
          className={`flex-1 px-4 py-3 text-sm font-medium border-b-2 transition-colors ${
            expandedTab === 'ingestion'
              ? 'border-purple-500 text-purple-600 bg-purple-50'
              : 'border-transparent text-gray-600 hover:text-gray-900'
          }`}
        >
          ⚙️ Ingestions ({stats.totalIngestionEvents})
        </button>
        <button
          onClick={() => setExpandedTab('conflicts')}
          className={`flex-1 px-4 py-3 text-sm font-medium border-b-2 transition-colors ${
            expandedTab === 'conflicts'
              ? 'border-red-500 text-red-600 bg-red-50'
              : 'border-transparent text-gray-600 hover:text-gray-900'
          }`}
        >
          ⚠️ Conflicts ({stats.totalConflicts})
        </button>
      </div>

      {/* Content */}
      <div className="max-h-96 overflow-y-auto">
        {expandedTab === 'calendar' && (
          <CalendarEventsTab events={calendarEvents} />
        )}
        {expandedTab === 'ingestion' && (
          <IngestionEventsTab events={ingestionEvents} />
        )}
        {expandedTab === 'conflicts' && (
          <ConflictsTab events={conflictEvents} />
        )}
      </div>

      {/* Footer */}
      <div className="border-t border-gray-200 bg-gray-50 p-3 flex items-center justify-between">
        <p className="text-xs text-gray-500">
          {stats.totalCalendarEvents + stats.totalIngestionEvents + stats.totalConflicts} total events
        </p>
        <button
          onClick={clearEvents}
          className="text-xs font-medium text-gray-600 hover:text-gray-900 px-3 py-1 rounded hover:bg-gray-200 transition-colors"
        >
          Clear
        </button>
      </div>
    </div>
  );
};

/**
 * Calendar Events Tab Content
 */
const CalendarEventsTab: React.FC<{ events: CalendarEvent[] }> = ({ events }) => {
  if (events.length === 0) {
    return (
      <div className="p-8 text-center text-gray-500">
        <p className="text-sm">No calendar updates received yet</p>
        <p className="text-xs text-gray-400 mt-1">Waiting for calendar changes...</p>
      </div>
    );
  }

  return (
    <div className="divide-y divide-gray-200">
      {events.map((event) => (
        <div key={event.eventId} className="p-3 hover:bg-blue-50 transition-colors">
          <div className="flex items-start justify-between mb-2">
            <div>
              <p className="font-mono text-sm font-bold text-gray-900">{event.calendarDate}</p>
              <p className="text-xs text-gray-500 mt-0.5">
                {event.region}
                {event.exchange && ` • ${event.exchange}`}
              </p>
            </div>
            <span className="text-lg">{event.isBusinessDay ? '💼' : '🎉'}</span>
          </div>
          {event.holidayName && (
            <p className="text-xs font-bold text-gray-700 mb-2">{event.holidayName}</p>
          )}
          <div className="flex items-center gap-2 flex-wrap">
            <span className="text-xs bg-gray-100 px-2 py-0.5 rounded">
              {event.sourceSystem}
            </span>
            <span className="text-xs bg-blue-100 text-blue-700 px-2 py-0.5 rounded">
              {event.confidenceScore}% confidence
            </span>
            {event.ruleApplied && (
              <span className="text-xs bg-green-100 text-green-700 px-2 py-0.5 rounded">
                {event.ruleApplied}
              </span>
            )}
          </div>
        </div>
      ))}
    </div>
  );
};

/**
 * Ingestion Events Tab Content
 */
const IngestionEventsTab: React.FC<{ events: IngestionEvent[] }> = ({ events }) => {
  if (events.length === 0) {
    return (
      <div className="p-8 text-center text-gray-500">
        <p className="text-sm">No ingestion events yet</p>
      </div>
    );
  }

  return (
    <div className="divide-y divide-gray-200">
      {events.map((event) => (
        <div key={event.ingestionId} className="p-3 hover:bg-purple-50 transition-colors">
          <div className="flex items-center justify-between mb-2">
            <p className="font-bold text-sm text-gray-900">{event.eventType}</p>
            <span
              className={`text-xs font-bold px-2 py-0.5 rounded ${
                event.status === 'SUCCESS'
                  ? 'bg-green-100 text-green-700'
                  : event.status === 'FAILURE'
                    ? 'bg-red-100 text-red-700'
                    : 'bg-yellow-100 text-yellow-700'
              }`}
            >
              {event.status}
            </span>
          </div>
          {event.eventType === 'COMPLETED' && (
            <div className="text-xs text-gray-600 space-y-1 mt-2">
              <p>
                <span className="text-gray-500">Records:</span> {event.recordsIngested} (
                <span className="text-green-600">+{event.recordsCreated}</span>
                <span className="text-gray-500">/</span>
                <span className="text-blue-600">~{event.recordsUpdated}</span>)
              </p>
              <p>
                <span className="text-gray-500">Conflicts:</span> {event.conflictsDetected} (
                <span className="text-green-600">✓{event.conflictsResolved}</span>)
              </p>
              <p>
                <span className="text-gray-500">Sources:</span> {event.sourcesSucceeded}/
                {event.sourcesQueried}
              </p>
              <p>
                <span className="text-gray-500">Duration:</span> {event.durationMs}ms
              </p>
            </div>
          )}
        </div>
      ))}
    </div>
  );
};

/**
 * Conflicts Tab Content
 */
const ConflictsTab: React.FC<{ events: ConflictEvent[] }> = ({ events }) => {
  if (events.length === 0) {
    return (
      <div className="p-8 text-center text-gray-500">
        <p className="text-sm">No conflicts detected</p>
        <p className="text-xs text-gray-400 mt-1">Great! Data is consistent across sources</p>
      </div>
    );
  }

  return (
    <div className="divide-y divide-gray-200">
      {events.map((event) => (
        <div
          key={event.conflictId}
          className={`p-3 transition-colors ${
            event.severity === 4
              ? 'bg-red-50 border-l-2 border-red-500'
              : event.severity === 3
                ? 'bg-orange-50 border-l-2 border-orange-500'
                : 'bg-yellow-50 border-l-2 border-yellow-500'
          }`}
        >
          <div className="flex items-start justify-between mb-2">
            <div>
              <p className="font-mono text-sm font-bold text-gray-900">{event.calendarDate}</p>
              <p className="text-xs text-gray-500 mt-0.5">
                {event.region} • {event.fieldName}
              </p>
            </div>
            <span className="text-lg">{event.resolved ? '✅' : '⚠️'}</span>
          </div>
          <p className="text-xs text-gray-700 mb-2">{event.reason}</p>
          <div className="flex items-center gap-1 flex-wrap">
            {event.sourceSystems.map((system, idx) => (
              <span key={idx} className="text-xs bg-gray-100 px-2 py-0.5 rounded">
                {system}
              </span>
            ))}
          </div>
        </div>
      ))}
    </div>
  );
};

export default LiveCalendarUpdates;
