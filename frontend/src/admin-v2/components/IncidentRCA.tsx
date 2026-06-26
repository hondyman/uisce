import React from "react";
import { OpsEvent, OpsIncident } from "../hooks/useOpsTimeline";
import { computeIncidentRCA } from "../hooks/useIncident";
import { Card } from "./Card";

interface IncidentRCAProps {
  incident: OpsIncident;
  events: OpsEvent[];
}

export function IncidentRCA({ incident, events }: IncidentRCAProps) {
  const rca = computeIncidentRCA(events);

  return (
    <Card title="Root Cause Analysis">
      <div className="rca-container">
        <div className="rca-section">
          <h4>📌 Suspected Root Cause</h4>
          {rca.suspectedRootCause ? (
            <div className="root-cause-event">
              <div className="rca-event-header">
                <span className={`severity-chip severity-${rca.suspectedRootCause.severity}`}>
                  {rca.suspectedRootCause.severity.toUpperCase()}
                </span>
                <span className="event-title">{rca.suspectedRootCause.title}</span>
              </div>
              <div className="rca-event-details">
                <span className="rca-detail">
                  🕐 {new Date(rca.suspectedRootCause.occurred_at).toLocaleString()}
                </span>
                <span className="rca-detail">
                  📊 Type: {rca.suspectedRootCause.event_type.replace(/_/g, " ")}
                </span>
                {rca.suspectedRootCause.tenant_id && (
                  <span className="rca-detail">
                    🏢 Tenant: {rca.suspectedRootCause.tenant_id.slice(0, 8)}...
                  </span>
                )}
                {rca.suspectedRootCause.endpoint_path && (
                  <span className="rca-detail">
                    🔗 Endpoint: {rca.suspectedRootCause.endpoint_path}
                  </span>
                )}
              </div>
            </div>
          ) : (
            <p className="no-data">No critical events detected in this incident.</p>
          )}
        </div>

        <div className="rca-section">
          <h4>💥 Blast Radius</h4>
          <div className="blast-radius">
            <div className="radius-item">
              <span className="radius-label">Affected Tenants</span>
              {rca.affectedTenants.size > 0 ? (
                <div className="affected-list">
                  {Array.from(rca.affectedTenants).map((t) => (
                    <span key={t} className="affected-badge">
                      {t.slice(0, 8)}...
                    </span>
                  ))}
                </div>
              ) : (
                <p className="no-data-small">None</p>
              )}
            </div>

            <div className="radius-item">
              <span className="radius-label">Affected Endpoints</span>
              {rca.affectedEndpoints.size > 0 ? (
                <div className="affected-list">
                  {Array.from(rca.affectedEndpoints)
                    .slice(0, 5)
                    .map((e) => (
                      <span key={e} className="affected-badge">
                        {e}
                      </span>
                    ))}
                  {rca.affectedEndpoints.size > 5 && (
                    <span className="affected-badge more">
                      +{rca.affectedEndpoints.size - 5} more
                    </span>
                  )}
                </div>
              ) : (
                <p className="no-data-small">None</p>
              )}
            </div>
          </div>
        </div>

        <div className="rca-section">
          <h4>📊 Event Distribution</h4>
          <div className="event-distribution">
            {getLargestEventTypes(events).map(({ type, count }) => (
              <div key={type} className="event-type-stat">
                <span className="stat-label">{type.replace(/_/g, " ")}</span>
                <span className="stat-count">{count}</span>
              </div>
            ))}
          </div>
        </div>
      </div>
    </Card>
  );
}

function getLargestEventTypes(events: OpsEvent[], limit: number = 5) {
  const counts: Record<string, number> = {};
  events.forEach((e) => {
    counts[e.event_type] = (counts[e.event_type] || 0) + 1;
  });

  return Object.entries(counts)
    .sort(([, a], [, b]) => b - a)
    .slice(0, limit)
    .map(([type, count]) => ({ type, count }));
}

const styles = `
.rca-container {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.rca-section {
  padding-bottom: 1.5rem;
  border-bottom: 1px solid var(--color-border);
}

.rca-section:last-child {
  border-bottom: none;
  padding-bottom: 0;
}

.rca-section h4 {
  margin: 0 0 1rem 0;
  font-size: 0.95rem;
  font-weight: 600;
  color: var(--color-text-primary);
}

.root-cause-event {
  padding: 1rem;
  background: var(--color-surface);
  border-left: 4px solid #ef4444;
  border-radius: 4px;
}

.rca-event-header {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  margin-bottom: 0.75rem;
}

.severity-chip {
  padding: 3px 8px;
  border-radius: 3px;
  font-size: 0.75rem;
  font-weight: 700;
}

.severity-chip.severity-critical {
  background: #ef4444;
  color: white;
}

.severity-chip.severity-error {
  background: #f87171;
  color: white;
}

.event-title {
  font-weight: 600;
  color: var(--color-text-primary);
  flex: 1;
}

.rca-event-details {
  display: flex;
  flex-direction: column;
  gap: 0.375rem;
}

.rca-detail {
  font-size: 0.85rem;
  color: var(--color-text-secondary);
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.no-data {
  font-size: 0.9rem;
  color: var(--color-text-secondary);
  margin: 0;
  padding: 1rem;
  text-align: center;
  background: var(--color-bg);
  border-radius: 4px;
}

.no-data-small {
  font-size: 0.85rem;
  color: var(--color-text-secondary);
  margin: 0;
}

.blast-radius {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 1rem;
}

.radius-item {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.radius-label {
  font-size: 0.85rem;
  font-weight: 600;
  color: var(--color-text-primary);
}

.affected-list {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}

.affected-badge {
  padding: 4px 8px;
  background: var(--color-bg);
  border-radius: 3px;
  font-size: 0.8rem;
  color: var(--color-text-secondary);
  border: 1px solid var(--color-border);
}

.affected-badge.more {
  background: var(--color-accent);
  color: white;
  border-color: var(--color-accent);
  font-weight: 600;
}

.event-distribution {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
  gap: 1rem;
}

.event-type-stat {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.75rem;
  background: var(--color-surface);
  border-radius: 4px;
}

.stat-label {
  font-size: 0.85rem;
  font-weight: 500;
  color: var(--color-text-secondary);
  text-transform: capitalize;
}

.stat-count {
  font-size: 1.25rem;
  font-weight: 700;
  color: var(--color-accent);
}

@media (max-width: 768px) {
  .blast-radius {
    grid-template-columns: 1fr;
  }

  .rca-event-header {
    flex-wrap: wrap;
  }
}
`;

export default styles;
