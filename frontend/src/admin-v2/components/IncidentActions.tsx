import React, { useState } from "react";
import { OpsIncident } from "../hooks/useOpsTimeline";
import { useCloseIncidentWithAnalysis } from "../hooks/useIncident";
import { Card } from "./Card";

interface IncidentActionsProps {
  incident: OpsIncident;
}

export function IncidentActions({ incident }: IncidentActionsProps) {
  const [summary, setSummary] = useState("");
  const [rootCause, setRootCause] = useState("");
  const closeIncident = useCloseIncidentWithAnalysis(incident.id);

  const handleClose = () => {
    closeIncident.mutate({ summary, rootCause });
  };

  const isOpen = incident.status === "open";
  const hasAnalysis = summary.trim().length > 0 || rootCause.trim().length > 0;

  return (
    <Card title="Actions">
      {!isOpen ? (
        <div className="closed-indicator">
          <p>This incident has been closed.</p>
          {incident.summary && <p className="detail">Summary: {incident.summary}</p>}
          {incident.root_cause && <p className="detail">RCA: {incident.root_cause}</p>}
        </div>
      ) : (
        <div className="close-form">
          <div className="form-group">
            <label htmlFor="summary">Summary (optional)</label>
            <textarea
              id="summary"
              value={summary}
              onChange={(e) => setSummary(e.target.value)}
              placeholder="Brief summary of the issue and resolution"
              rows={3}
            />
          </div>

          <div className="form-group">
            <label htmlFor="rootCause">Root Cause (optional)</label>
            <textarea
              id="rootCause"
              value={rootCause}
              onChange={(e) => setRootCause(e.target.value)}
              placeholder="What caused this incident? Why did it happen?"
              rows={3}
            />
          </div>

          <button
            className="close-button"
            onClick={handleClose}
            disabled={closeIncident.isPending}
          >
            {closeIncident.isPending ? "Closing..." : "Close Incident"}
          </button>

          {closeIncident.isError && (
            <div className="error-message">
              Failed to close incident: {closeIncident.error?.message}
            </div>
          )}
        </div>
      )}
    </Card>
  );
}

const styles = `
.closed-indicator {
  padding: 1rem;
  background: #d1fae5;
  border-left: 4px solid #059669;
  border-radius: 4px;
}

.closed-indicator p {
  margin: 0;
  color: #065f46;
  font-weight: 600;
}

.closed-indicator .detail {
  margin-top: 0.5rem;
  font-weight: normal;
  font-size: 0.9rem;
  color: #047857;
}

.close-form {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.form-group label {
  font-size: 0.9rem;
  font-weight: 600;
  color: var(--color-text-primary);
}

.form-group textarea {
  padding: 0.75rem;
  border: 1px solid var(--color-border);
  border-radius: 4px;
  font-family: inherit;
  font-size: 0.9rem;
  resize: vertical;
  background: var(--color-surface);
  color: var(--color-text-primary);
}

.form-group textarea:focus {
  outline: none;
  border-color: var(--color-accent);
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.close-button {
  padding: 0.75rem 1.5rem;
  background: #ef4444;
  color: white;
  border: none;
  border-radius: 4px;
  font-weight: 600;
  font-size: 0.95rem;
  cursor: pointer;
  transition: all 150ms ease;
}

.close-button:hover:not(:disabled) {
  background: #dc2626;
  transform: translateY(-1px);
}

.close-button:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.error-message {
  padding: 0.75rem;
  background: #fee2e2;
  border: 1px solid #fecaca;
  border-radius: 4px;
  color: #991b1b;
  font-size: 0.9rem;
}
`;

export default styles;
