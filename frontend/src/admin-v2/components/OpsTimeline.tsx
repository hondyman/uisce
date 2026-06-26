import React, { useState } from "react";
import { useOpsTimeline, OpsEvent } from "../hooks/useOpsTimeline";
import { Spinner } from "./Feedback";
import "./OpsTimeline.css";

export interface OpsTimelineProps {
  since?: string;
  limit?: number;
  onEventClick?: (event: OpsEvent) => void;
}

/**
 * Real-time operations timeline component
 * Displays recent events and incidents in chronological order
 */
export function OpsTimeline({
  since = "1h",
  limit = 200,
  onEventClick,
}: OpsTimelineProps) {
  const { data, isLoading, error } = useOpsTimeline(since, limit);
  const [selectedSeverity, setSelectedSeverity] = useState<string | null>(null);

  if (isLoading) return <Spinner size="md" />;
  if (error) return <div className="error">Failed to load timeline</div>;

  const events = data?.events ?? [];

  // Filter by severity if selected
  const filteredEvents = selectedSeverity
    ? events.filter((e) => e.severity === selectedSeverity)
    : events;

  const severityCounts = {
    info: events.filter((e) => e.severity === "info").length,
    warning: events.filter((e) => e.severity === "warning").length,
    error: events.filter((e) => e.severity === "error").length,
    critical: events.filter((e) => e.severity === "critical").length,
  };

  const formatTime = (isoString: string) => {
    const d = new Date(isoString);
    return d.toLocaleTimeString() + " " + d.toLocaleDateString();
  };

  const eventIcon = (eventType: string) => {
    const icons: Record<string, string> = {
      alert: "🚨",
      fingerprint: "👆",
      tenant_health: "🏢",
      endpoint_health: "🔗",
      latency_anomaly: "📈",
      incident_opened: "📌",
      incident_closed: "✅",
    };
    return icons[eventType] || "•";
  };

  return (
    <div className="ops-timeline">
      {/* Filter controls */}
      <div className="timeline-controls">
        <button
          className={`filter-btn ${!selectedSeverity ? "active" : ""}`}
          onClick={() => setSelectedSeverity(null)}
        >
          All ({events.length})
        </button>
        <button
          className={`filter-btn severity-critical ${
            selectedSeverity === "critical" ? "active" : ""
          }`}
          onClick={() => setSelectedSeverity("critical")}
        >
          Critical ({severityCounts.critical})
        </button>
        <button
          className={`filter-btn severity-error ${
            selectedSeverity === "error" ? "active" : ""
          }`}
          onClick={() => setSelectedSeverity("error")}
        >
          Error ({severityCounts.error})
        </button>
        <button
          className={`filter-btn severity-warning ${
            selectedSeverity === "warning" ? "active" : ""
          }`}
          onClick={() => setSelectedSeverity("warning")}
        >
          Warning ({severityCounts.warning})
        </button>
        <button
          className={`filter-btn severity-info ${
            selectedSeverity === "info" ? "active" : ""
          }`}
          onClick={() => setSelectedSeverity("info")}
        >
          Info ({severityCounts.info})
        </button>
      </div>

      {/* Timeline */}
      <div className="timeline-container">
        {filteredEvents.length === 0 ? (
          <div className="empty-state">
            <p>No events in this time range</p>
          </div>
        ) : (
          filteredEvents.map((event) => (
            <div
              key={event.id}
              className={`timeline-item severity-${event.severity}`}
              onClick={() => onEventClick?.(event)}
              role="button"
              tabIndex={0}
            >
              <div className="timeline-marker">
                <span className="timeline-icon">
                  {eventIcon(event.event_type)}
                </span>
              </div>

              <div className="timeline-content">
                <div className="timeline-header">
                  <h3 className="timeline-title">{event.title}</h3>
                  <span className={`severity-badge severity-${event.severity}`}>
                    {event.severity.toUpperCase()}
                  </span>
                </div>

                <div className="timeline-meta">
                  <span className="event-type">{event.event_type}</span>
                  <span className="event-scope">scope: {event.scope}</span>
                  {event.tenant_id && (
                    <span className="event-tenant">tenant: {event.tenant_id.substring(0, 8)}</span>
                  )}
                  {event.endpoint_path && (
                    <span className="event-endpoint">{event.endpoint_path}</span>
                  )}
                </div>

                <div className="timeline-time">
                  {formatTime(event.occurred_at)}
                </div>

                {event.incident_id && (
                  <div className="timeline-incident">
                    Incident: {event.incident_id.substring(0, 8)}
                  </div>
                )}
              </div>
            </div>
          ))
        )}
      </div>
    </div>
  );
}
