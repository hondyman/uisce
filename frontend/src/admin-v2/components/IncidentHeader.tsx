import React from "react";
import { OpsIncident } from "../hooks/useOpsTimeline";
import { Card } from "./Card";

interface IncidentHeaderProps {
  incident: OpsIncident;
}

export function IncidentHeader({ incident }: IncidentHeaderProps) {
  const startTime = new Date(incident.started_at);
  const endTime = incident.ended_at ? new Date(incident.ended_at) : null;
  const duration = endTime
    ? Math.round((endTime.getTime() - startTime.getTime()) / 1000 / 60)
    : "ongoing";

  return (
    <Card title={`Incident: ${incident.title}`}>
      <div className="incident-header">
        <div className="header-row">
          <div className="header-field">
            <label>Status</label>
            <span className={`status-badge status-${incident.status}`}>
              {incident.status.toUpperCase()}
            </span>
          </div>
          <div className="header-field">
            <label>Severity</label>
            <span className={`severity-badge severity-${incident.severity}`}>
              {incident.severity.toUpperCase()}
            </span>
          </div>
          <div className="header-field">
            <label>Duration</label>
            <span>{duration === "ongoing" ? duration : `${duration}m`}</span>
          </div>
        </div>

        <div className="header-row">
          <div className="header-field">
            <label>Started</label>
            <span>
              {startTime.toLocaleTimeString()} {startTime.toLocaleDateString()}
            </span>
          </div>
          {endTime && (
            <div className="header-field">
              <label>Ended</label>
              <span>
                {endTime.toLocaleTimeString()} {endTime.toLocaleDateString()}
              </span>
            </div>
          )}
        </div>

        {incident.summary && (
          <div className="summary-section">
            <h4>Summary</h4>
            <p>{incident.summary}</p>
          </div>
        )}

        {incident.root_cause && (
          <div className="rca-section">
            <h4>Root Cause</h4>
            <p>{incident.root_cause}</p>
          </div>
        )}
      </div>
    </Card>
  );
}

const styles = `
.incident-header {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.header-row {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 1rem;
}

.header-field {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.header-field label {
  font-size: 0.875rem;
  color: var(--color-text-secondary);
  font-weight: 600;
  text-transform: uppercase;
}

.header-field span {
  font-size: 1rem;
  color: var(--color-text-primary);
}

.status-badge {
  padding: 6px 12px;
  border-radius: 4px;
  font-weight: 600;
  font-size: 0.875rem;
  width: fit-content;
}

.status-badge.status-open {
  background: #fef3c7;
  color: #92400e;
}

.status-badge.status-closed {
  background: #d1fae5;
  color: #065f46;
}

.severity-badge {
  padding: 6px 12px;
  border-radius: 4px;
  font-weight: 700;
  font-size: 0.875rem;
  text-transform: uppercase;
  width: fit-content;
}

.severity-badge.severity-critical {
  background: rgba(239, 68, 68, 0.2);
  color: #991b1b;
}

.severity-badge.severity-error {
  background: rgba(248, 113, 113, 0.2);
  color: #b91c1c;
}

.severity-badge.severity-warning {
  background: rgba(234, 179, 8, 0.2);
  color: #92400e;
}

.severity-badge.severity-info {
  background: rgba(59, 130, 246, 0.2);
  color: #1e40af;
}

.summary-section,
.rca-section {
  border-left: 4px solid var(--color-accent);
  padding-left: 1rem;
}

.summary-section h4,
.rca-section h4 {
  margin: 0 0 0.5rem 0;
  font-size: 0.95rem;
  font-weight: 600;
  color: var(--color-text-primary);
}

.summary-section p,
.rca-section p {
  margin: 0;
  font-size: 0.9rem;
  line-height: 1.5;
  color: var(--color-text-secondary);
}
`;

export default styles;
