import React, { useMemo, useCallback, useState } from 'react';
import { RegionSelector } from './RegionSelector';
import { PropagationVisualizer } from './PropagationVisualizer';
import { useEventListener, EventType, StreamedEvent } from '../hooks/useRealtimeEvents';

/**
 * Phase 3.4: Real-Time Incident Dashboard Component
 * Integrates WebSocket event streaming with UI components
 */

export interface RealtimeIncidentDashboardProps {
  tenantId: string;
  initialRegions?: string[];
  onIncidentUpdated?: (incident: StreamedEvent) => void;
}

export const RealtimeIncidentDashboard: React.FC<RealtimeIncidentDashboardProps> = ({
  tenantId,
  initialRegions = [],
  onIncidentUpdated,
}) => {
  // State management
  const [selectedRegions, setSelectedRegions] = useState<string[]>(initialRegions);
  const [isPaused, setIsPaused] = useState(false);
  const [filteredEvents, setFilteredEvents] = useState<StreamedEvent[]>([]);
  const [incidentIndex, setIncidentIndex] = useState<Map<string, StreamedEvent>>(new Map());

  // Streaming hooks
  const {
    events,
    connectionState,
    pause,
    resume,
  } = useEventListener({
    tenantId,
    regions: selectedRegions,
    eventTypes: [
      EventType.IncidentDetected,
      EventType.IncidentUpdated,
      EventType.IncidentResolved,
      EventType.RCACompleted,
      EventType.ActionCompleted,
      EventType.PropagationDetected,
      EventType.RegionFailover,
    ],
  });

  // Update incident index on events
  React.useEffect(() => {
    const newIndex = new Map(incidentIndex);

    for (const event of events) {
      if (event.incident_id) {
        newIndex.set(event.incident_id, event);

        // Notify parent component
        if (onIncidentUpdated) {
          onIncidentUpdated(event);
        }
      }
    }

    setIncidentIndex(newIndex);
    setFilteredEvents(Array.from(newIndex.values()));
  }, [events, onIncidentUpdated, incidentIndex]);

  // Handle region selection
  const handleRegionChange = useCallback((regions: string[]) => {
    setSelectedRegions(regions);
  }, []);

  // Handle pause/resume
  const handleTogglePause = useCallback(() => {
    if (isPaused) {
      resume();
    } else {
      pause();
    }
    setIsPaused(!isPaused);
  }, [isPaused, pause, resume]);

  // Compute propagation paths for visualization
  const propagationPaths = useMemo(() => {
    const paths: Array<{
      fromRegion: string;
      toRegions: string[];
      likelihood: number;
      incidentId: string;
    }> = [];

    for (const event of events) {
      if (event.type === EventType.PropagationDetected && event.payload) {
        const toRegions = event.payload.to_regions || [];
        const likelihood = event.payload.likelihood || 0;

        paths.push({
          fromRegion: event.region || 'unknown',
          toRegions,
          likelihood,
          incidentId: event.incident_id || 'unknown',
        });
      }
    }

    return paths;
  }, [events]);

  // Compute failover events
  const failovers = useMemo(() => {
    return events.filter(e => e.type === EventType.RegionFailover).map(e => ({
      fromRegion: e.payload?.from_region || 'unknown',
      toRegion: e.payload?.to_region || 'unknown',
      timestamp: e.timestamp,
    }));
  }, [events]);

  // Compute RCA results for display
  const rcaResults = useMemo(() => {
    return events.filter(e => e.type === EventType.RCACompleted).map(e => ({
      incidentId: e.incident_id,
      timestamp: e.timestamp,
      results: e.payload,
    }));
  }, [events]);

  // Status badge component
  const ConnectionStatus = () => (
    <div className="flex items-center gap-2 px-3 py-2 rounded bg-gray-100">
      <div
        className={`w-3 h-3 rounded-full ${
          connectionState.isConnected
            ? 'bg-green-500'
            : connectionState.isConnecting
            ? 'bg-yellow-500'
            : 'bg-red-500'
        }`}
      />
      <span className="text-sm font-medium">
        {connectionState.isConnected
          ? 'Connected'
          : connectionState.isConnecting
          ? 'Connecting...'
          : 'Disconnected'}
      </span>
      {connectionState.reconnectAttempt > 0 && (
        <span className="text-xs text-gray-500">
          (Attempt {connectionState.reconnectAttempt})
        </span>
      )}
    </div>
  );

  // Stats component
  const EventStats = () => (
    <div className="grid grid-cols-5 gap-2">
      <div className="bg-blue-50 p-3 rounded">
        <div className="text-xs text-gray-600">Total Incidents</div>
        <div className="text-xl font-bold text-blue-600">
          {incidentIndex.size}
        </div>
      </div>
      <div className="bg-purple-50 p-3 rounded">
        <div className="text-xs text-gray-600">RCA Completed</div>
        <div className="text-xl font-bold text-purple-600">
          {rcaResults.length}
        </div>
      </div>
      <div className="bg-green-50 p-3 rounded">
        <div className="text-xs text-gray-600">Actions Executed</div>
        <div className="text-xl font-bold text-green-600">
          {events.filter(e => e.type === EventType.ActionCompleted).length}
        </div>
      </div>
      <div className="bg-orange-50 p-3 rounded">
        <div className="text-xs text-gray-600">Propagations Detected</div>
        <div className="text-xl font-bold text-orange-600">
          {propagationPaths.length}
        </div>
      </div>
      <div className="bg-red-50 p-3 rounded">
        <div className="text-xs text-gray-600">Failovers Triggered</div>
        <div className="text-xl font-bold text-red-600">
          {failovers.length}
        </div>
      </div>
    </div>
  );

  // Incident list component
  const IncidentList = () => (
    <div className="space-y-2 max-h-96 overflow-y-auto">
      {filteredEvents.length === 0 ? (
        <div className="text-center py-8 text-gray-500">
          No incidents to display
        </div>
      ) : (
        filteredEvents.map(incident => (
          <div
            key={incident.id}
            className="border rounded p-3 hover:bg-gray-50 cursor-pointer transition"
          >
            <div className="flex items-start justify-between">
              <div className="flex-1">
                <div className="font-medium text-sm">
                  {incident.incident_id}
                </div>
                <div className="text-xs text-gray-600 mt-1">
                  Severity: {incident.severity || 'N/A'} | Region: {incident.region || 'N/A'}
                </div>
                <div className="text-xs text-gray-500 mt-1">
                  Last updated: {new Date(incident.timestamp).toLocaleTimeString()}
                </div>
              </div>
              <div
                className={`px-2 py-1 rounded text-xs font-medium ${
                  incident.type === EventType.IncidentResolved
                    ? 'bg-green-100 text-green-800'
                    : incident.severity === 'critical'
                    ? 'bg-red-100 text-red-800'
                    : 'bg-yellow-100 text-yellow-800'
                }`}
              >
                {incident.type}
              </div>
            </div>
          </div>
        ))
      )}
    </div>
  );

  return (
    <div className="w-full h-full bg-white p-6 rounded-lg shadow">
      {/* Header */}
      <div className="mb-6">
        <div className="flex items-center justify-between mb-4">
          <h1 className="text-2xl font-bold text-gray-900">
            Real-Time Incident Dashboard
          </h1>
          <div className="flex gap-3">
            <ConnectionStatus />
            <button
              onClick={handleTogglePause}
              className={`px-4 py-2 rounded font-medium transition ${
                isPaused
                  ? 'bg-blue-600 text-white hover:bg-blue-700'
                  : 'bg-gray-200 text-gray-900 hover:bg-gray-300'
              }`}
            >
              {isPaused ? 'Resume' : 'Pause'}
            </button>
          </div>
        </div>

        {connectionState.error && (
          <div className="bg-red-50 border border-red-200 rounded p-3 text-sm text-red-800">
            {connectionState.error}
          </div>
        )}
      </div>

      {/* Stats */}
      <div className="mb-6">
        <h2 className="text-lg font-semibold text-gray-800 mb-3">
          Real-Time Metrics
        </h2>
        <EventStats />
      </div>

      {/* Region Selector */}
      <div className="mb-6">
        <h2 className="text-lg font-semibold text-gray-800 mb-3">
          Filter by Region
        </h2>
        <RegionSelector
          selectedRegions={selectedRegions}
          onRegionChange={handleRegionChange}
        />
      </div>

      {/* Propagation Visualizer */}
      {propagationPaths.length > 0 && (
        <div className="mb-6">
          <h2 className="text-lg font-semibold text-gray-800 mb-3">
            Cross-Region Propagation
          </h2>
          <PropagationVisualizer
            paths={propagationPaths}
            failovers={failovers}
          />
        </div>
      )}

      {/* Incident List */}
      <div>
        <h2 className="text-lg font-semibold text-gray-800 mb-3">
          Incidents ({filteredEvents.length})
        </h2>
        <IncidentList />
      </div>

      {/* Failover Alerts */}
      {failovers.length > 0 && (
        <div className="mt-6 border-t pt-6">
          <h2 className="text-lg font-semibold text-gray-800 mb-3">
            Recent Failovers
          </h2>
          <div className="space-y-2">
            {failovers.map((failover, idx) => (
              <div
                key={idx}
                className="bg-red-50 border border-red-200 rounded p-3 text-sm"
              >
                <div className="font-medium text-red-900">
                  Failover: {failover.fromRegion} → {failover.toRegion}
                </div>
                <div className="text-xs text-red-700 mt-1">
                  {new Date(failover.timestamp).toLocaleTimeString()}
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* RCA Results */}
      {rcaResults.length > 0 && (
        <div className="mt-6 border-t pt-6">
          <h2 className="text-lg font-semibold text-gray-800 mb-3">
            RCA Analysis Results
          </h2>
          <div className="space-y-2 max-h-32 overflow-y-auto">
            {rcaResults.slice(-5).map((result, idx) => (
              <div
                key={idx}
                className="bg-purple-50 border border-purple-200 rounded p-3 text-sm"
              >
                <div className="font-medium text-purple-900">
                  Incident: {result.incidentId}
                </div>
                <div className="text-xs text-purple-700 mt-1">
                  {new Date(result.timestamp).toLocaleTimeString()}
                </div>
                {result.results?.root_causes && (
                  <div className="text-xs text-purple-600 mt-2">
                    Root Causes: {result.results.root_causes.join(', ')}
                  </div>
                )}
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
};

export default RealtimeIncidentDashboard;
