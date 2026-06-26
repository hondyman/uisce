import React, { useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { useIncident } from "../hooks/useIncident";
import { useRCA } from "../hooks/useRCA";
import { useIncidentPattern, useSimilarIncidents } from "../hooks/usePatterns";
import { IncidentHeader } from "../components/IncidentHeader";
import { IncidentActions } from "../components/IncidentActions";
import { IncidentRCA } from "../components/IncidentRCA";
import { IncidentActionsPanel } from "../components/IncidentActionsPanel";
import { IntelligentRCAPanel } from "../components/IntelligentRCAPanel";
import { PatternMatchPanel } from "../components/PatternMatchPanel";
import { SmartRCASuggestions } from "../components/SmartRCASuggestions";
import { ExecuteActionModal } from "../components/ExecuteActionModal";
import { OpsTimeline } from "../components/OpsTimeline";
import { Spinner } from "../components/Feedback";
import { Card } from "../components/Card";

export function IncidentDetailPage() {
  const { incidentId } = useParams();
  const navigate = useNavigate();
  const [selectedSmartAction, setSelectedSmartAction] = useState<string | null>(null);
  const incidentQuery = useIncident(incidentId || null);
  const rcaQuery = useRCA(incidentId || "");
  const patternQuery = useIncidentPattern(incidentId || "");
  const similarQuery = useSimilarIncidents(incidentId || "");

  if (!incidentId) {
    return (
      <div className="page">
        <h1>Incident Not Found</h1>
        <p>No incident ID provided. Please select an incident from the timeline.</p>
      </div>
    );
  }

  if (incidentQuery.isLoading) {
    return (
      <div className="page loading-page">
        <Spinner size="lg" />
        <h2>Loading incident details...</h2>
      </div>
    );
  }

  if (incidentQuery.isError || !incidentQuery.data) {
    return (
      <div className="page error-page">
        <h1>Error Loading Incident</h1>
        <p>Failed to load incident details. Please try again.</p>
        <button onClick={() => navigate("/admin/operations")} className="back-button">
          ← Back to Timeline
        </button>
      </div>
    );
  }

  const { incident, events } = incidentQuery.data;

  return (
    <div className="page">
      <div className="incident-header-nav">
        <button onClick={() => navigate(-1)} className="back-button">
          ← Back
        </button>
        <h1>{incident.title}</h1>
      </div>

      <div className="incident-grid">
        {/* Main sections */}
        <div className="incident-main">
          <IncidentHeader incident={incident} />
          <IncidentActions incident={incident} />
          <IncidentRCA incident={incident} events={events} />
          
          {/* Intelligent RCA - Correlation Scoring Analysis */}
          {rcaQuery.data && (
            <IntelligentRCAPanel 
              rca={rcaQuery.data} 
              isLoading={rcaQuery.isLoading}
            />
          )}
          
          {/* Smart RCA Suggestions - Combines RCA + Pattern Matching */}
          {rcaQuery.data && (
            <SmartRCASuggestions
              rca={rcaQuery.data}
              pattern={patternQuery.data}
              similarities={similarQuery.data?.similarities || []}
              onSuggestedActionClick={setSelectedSmartAction}
              isLoading={rcaQuery.isLoading || patternQuery.isLoading}
            />
          )}
          
          {/* Pattern Matching - Recurring Incident Detection */}
          {patternQuery.data && (
            <PatternMatchPanel
              pattern={patternQuery.data}
              similarities={similarQuery.data?.similarities || []}
              isLoading={patternQuery.isLoading}
            />
          )}
          
          <IncidentActionsPanel incidentId={incident.id} incidentStatus={incident.status} />
          
          {/* Modal for executing smart-suggested actions */}
          {selectedSmartAction && (
            <ExecuteActionModal
              incidentId={incident.id}
              actionType={selectedSmartAction}
              isOpen={!!selectedSmartAction}
              onClose={() => setSelectedSmartAction(null)}
              onSuccess={() => {
                setSelectedSmartAction(null);
                incidentQuery.refetch();
              }}
            />
          )}
        </div>

        {/* Event timeline sidebar */}
        <div className="incident-sidebar">
          <Card title={`Timeline (${events.length} events)`} className="timeline-card">
            <IncidentEventList events={events} />
          </Card>
        </div>
      </div>
    </div>
  );
}

function IncidentEventList({ events }: { events: any[] }) {
  if (events.length === 0) {
    return <p className="no-events">No events in this incident</p>;
  }

  return (
    <div className="event-list">
      {events.map((event) => (
        <div key={event.id} className={`event-item severity-${event.severity}`}>
          <div className="event-marker">{getEventIcon(event.event_type)}</div>
          <div className="event-content">
            <div className="event-title">{event.title}</div>
            <div className="event-meta">
              <span className="event-type">{event.event_type.replace(/_/g, " ")}</span>
              <span className="event-time">
                {new Date(event.occurred_at).toLocaleTimeString()}
              </span>
            </div>
          </div>
        </div>
      ))}
    </div>
  );
}

function getEventIcon(eventType: string): string {
  const icons: Record<string, string> = {
    alert: "🚨",
    fingerprint: "👆",
    tenant_health: "🏢",
    endpoint_health: "🔗",
    latency_anomaly: "📈",
    incident_opened: "📌",
    incident_closed: "✅",
  };
  return icons[eventType] || "📍";
}

const styles = `
.incident-header-nav {
  display: flex;
  align-items: center;
  gap: 1rem;
  margin-bottom: 2rem;
}

.back-button {
  padding: 0.5rem 1rem;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: 4px;
  cursor: pointer;
  font-weight: 600;
  transition: all 150ms ease;
}

.back-button:hover {
  background: var(--color-accent);
  color: white;
  border-color: var(--color-accent);
}

.incident-header-nav h1 {
  margin: 0;
  flex: 1;
}

.incident-grid {
  display: grid;
  grid-template-columns: 1fr 300px;
  gap: 2rem;
}

.incident-main {
  display: flex;
  flex-direction: column;
  gap: 2rem;
}

.incident-sidebar {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.timeline-card {
  position: sticky;
  top: 1rem;
}

.no-events {
  text-align: center;
  padding: 1rem;
  color: var(--color-text-secondary);
  font-size: 0.9rem;
  margin: 0;
}

.event-list {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  max-height: 600px;
  overflow-y: auto;
}

.event-item {
  display: flex;
  gap: 0.75rem;
  padding: 0.75rem;
  border-left: 3px solid;
  border-radius: 4px;
  background: var(--color-surface);
  transition: all 150ms ease;
}

.event-item:hover {
  background: var(--color-bg);
  transform: translateX(4px);
}

.event-item.severity-critical {
  border-left-color: #ef4444;
}

.event-item.severity-error {
  border-left-color: #f87171;
}

.event-item.severity-warning {
  border-left-color: #eab308;
}

.event-item.severity-info {
  border-left-color: #3b82f6;
}

.event-marker {
  font-size: 1.25rem;
  flex-shrink: 0;
}

.event-content {
  flex: 1;
}

.event-title {
  font-size: 0.85rem;
  font-weight: 600;
  color: var(--color-text-primary);
  margin-bottom: 0.25rem;
  word-break: break-word;
}

.event-meta {
  display: flex;
  gap: 0.5rem;
  flex-wrap: wrap;
}

.event-type,
.event-time {
  font-size: 0.75rem;
  color: var(--color-text-secondary);
  padding: 2px 6px;
  background: var(--color-bg);
  border-radius: 2px;
}

.loading-page {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 60vh;
  gap: 1rem;
}

.error-page {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 60vh;
  text-align: center;
}

@media (max-width: 1024px) {
  .incident-grid {
    grid-template-columns: 1fr;
  }

  .timeline-card {
    position: relative;
    top: auto;
  }
}
`;

export default styles;
