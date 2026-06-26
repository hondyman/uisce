import React, { useState } from "react";
import {
  useIncidentActions,
  OPS_ACTIONS,
  formatActionResult,
} from "../hooks/useOpsActions";
import { ExecuteActionModal } from "./ExecuteActionModal";
import { Spinner } from "./Feedback";
import "./IncidentActionsPanel.css";

export interface IncidentActionsPanelProps {
  incidentId: string;
  incidentStatus: "open" | "closed";
}

export function IncidentActionsPanel({
  incidentId,
  incidentStatus,
}: IncidentActionsPanelProps) {
  const [selectedAction, setSelectedAction] = useState<string | null>(null);
  const [isModalOpen, setIsModalOpen] = useState(false);

  const { data: actions = [], isLoading } = useIncidentActions(incidentId);

  const handleActionClick = (actionId: string) => {
    setSelectedAction(actionId);
    setIsModalOpen(true);
  };

  const isIncidentOpen = incidentStatus === "open";

  return (
    <div className="incident-actions-panel">
      {/* Available Actions Section */}
      {isIncidentOpen && (
        <div className="actions-section">
          <h3>⚡ Available Actions</h3>

          <div className="actions-grid">
            {OPS_ACTIONS.map((action) => (
              <button
                key={action.id}
                className={`action-btn action-${action.severity}`}
                onClick={() => handleActionClick(action.id)}
                title={action.description}
              >
                <span className="action-icon">{action.icon}</span>
                <span className="action-name">{action.name}</span>
              </button>
            ))}
          </div>
        </div>
      )}

      {/* Action History Section */}
      <div className="action-history-section">
        <h3>📋 Action History</h3>

        {isLoading ? (
          <Spinner size="sm" />
        ) : actions.length === 0 ? (
          <p className="empty-state">No actions executed yet</p>
        ) : (
          <div className="action-timeline">
            {actions.map((action) => (
              <div key={action.id} className={`action-item action-${action.status}`}>
                <div className="action-item-header">
                  <span className="action-type">{action.action_type}</span>
                  <span className={`action-status status-${action.status}`}>
                    {action.status.toUpperCase()}
                  </span>
                </div>

                <div className="action-item-body">
                  <p className="action-summary">
                    {formatActionResult(action)}
                  </p>

                  {action.result && (
                    <div className="action-details">
                      {Object.entries(action.result).map(([key, value]) => (
                        <div key={key} className="detail-pair">
                          <span className="detail-key">{key}:</span>
                          <span className="detail-value">
                            {typeof value === "object"
                              ? JSON.stringify(value)
                              : String(value)}
                          </span>
                        </div>
                      ))}
                    </div>
                  )}

                  {action.error_msg && (
                    <div className="action-error">
                      <strong>Error:</strong> {action.error_msg}
                    </div>
                  )}
                </div>

                <div className="action-item-footer">
                  <span className="action-time">
                    {new Date(action.executed_at).toLocaleString()}
                  </span>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Modal for executing action */}
      {selectedAction && (
        <ExecuteActionModal
          incidentId={incidentId}
          actionType={selectedAction}
          isOpen={isModalOpen}
          onClose={() => {
            setIsModalOpen(false);
            setSelectedAction(null);
          }}
          onSuccess={() => {
            // Action history will auto-refresh via query invalidation
          }}
        />
      )}
    </div>
  );
}
