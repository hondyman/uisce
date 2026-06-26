import React, { useMemo } from "react";
import { RCAResult } from "../hooks/useRCA";
import {
  formatConfidenceScore,
  formatConfidenceColor,
  formatEventChain,
  getRemediationIcon,
  getRemediationLabel,
  getPriorityColor,
} from "../hooks/useRCA";
import "./IntelligentRCAPanel.css";

interface IntelligentRCAPanelProps {
  rca: RCAResult;
  isLoading?: boolean;
}

export const IntelligentRCAPanel: React.FC<IntelligentRCAPanelProps> = ({
  rca,
  isLoading = false,
}) => {
  if (isLoading) {
    return (
      <div className="intelligent-rca-panel loading">
        <div className="spinner"></div>
        <p>Computing intelligent RCA...</p>
      </div>
    );
  }

  if (!rca || !rca.suspected_root_cause) {
    return (
      <div className="intelligent-rca-panel">
        <p className="no-data">
          Insufficient event data for intelligent RCA analysis.
        </p>
      </div>
    );
  }

  const rootCause = rca.suspected_root_cause;
  const chain = rca.causality_chain;
  const confidence = rca.confidence_score;
  const suggestions = rca.suggested_remediations;
  const services = rca.affected_services;

  const eventTypeLabel = useMemo(() => {
    const type = rootCause.event?.event_type || "Unknown";
    return type.replace(/_/g, " ");
  }, [rootCause]);

  return (
    <div className="intelligent-rca-panel">
      {/* Confidence Header */}
      <div className="rca-header">
        <div className="confidence-badge">
          <div
            className="confidence-circle"
            style={{
              borderColor: formatConfidenceColor(confidence),
              backgroundColor: `${formatConfidenceColor(confidence)}22`,
            }}
          >
            <span className="confidence-percent">
              {Math.round(confidence * 100)}%
            </span>
          </div>
          <div className="confidence-text">
            <div className="label">Analysis Confidence</div>
            <div className="value">{formatConfidenceScore(confidence)}</div>
          </div>
        </div>
      </div>

      {/* Root Cause Section */}
      <div className="rca-section">
        <h3 className="section-title">🎯 Suspected Root Cause</h3>
        <div className="root-cause-card">
          <div className="cause-header">
            <div className="cause-type-badge">
              {eventTypeLabel.charAt(0).toUpperCase() + eventTypeLabel.slice(1)}
            </div>
            <div className="cause-metrics">
              <span className="metric">
                Causality Score:{" "}
                <strong>
                  {Math.round(rootCause.causality_score * 100)}%
                </strong>
              </span>
              <span className="metric">
                Impact Score:{" "}
                <strong>
                  {Math.round(rootCause.impact_score * 100)}%
                </strong>
              </span>
            </div>
          </div>

          <div className="cause-details">
            <div className="detail-row">
              <span className="label">Occurred:</span>
              <span className="value">
                {new Date(rootCause.event.occurred_at).toLocaleString()}
              </span>
            </div>
            {rootCause.event.tenant_id && (
              <div className="detail-row">
                <span className="label">Tenant:</span>
                <span className="value">
                  {rootCause.event.tenant_id.substring(0, 8)}...
                </span>
              </div>
            )}
            {rootCause.event.endpoint_path && (
              <div className="detail-row">
                <span className="label">Endpoint:</span>
                <span className="value">{rootCause.event.endpoint_path}</span>
              </div>
            )}
            {rootCause.event.description && (
              <div className="detail-row">
                <span className="label">Description:</span>
                <span className="value">{rootCause.event.description}</span>
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Causality Chain Section */}
      {chain.length > 0 && (
        <div className="rca-section">
          <h3 className="section-title">🔗 Causality Chain</h3>
          <div className="chain-description">
            {formatEventChain(chain)}
          </div>

          <div className="chain-timeline">
            {chain.map((event, idx) => (
              <div key={idx} className="chain-step">
                <div className="step-number">{idx + 1}</div>
                <div className="step-content">
                  <div className="step-type">
                    {event.event?.event_type?.replace(/_/g, " ") ||
                      "Unknown Event"}
                  </div>
                  <div className="step-metrics">
                    <span className="metric">
                      Causality:{" "}
                      {Math.round(event.causality_score * 100)}%
                    </span>
                    <span className="metric">
                      Impact: {Math.round(event.impact_score * 100)}%
                    </span>
                  </div>
                  <div className="step-time">
                    {new Date(
                      event.event?.occurred_at
                    ).toLocaleTimeString()}
                  </div>
                </div>
                {idx < chain.length - 1 && <div className="chain-arrow">↓</div>}
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Affected Services Section */}
      {services.length > 0 && (
        <div className="rca-section">
          <h3 className="section-title">📍 Affected Services</h3>
          <div className="services-grid">
            {services.map((service, idx) => (
              <div key={idx} className="service-tag">
                {service}
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Remediation Suggestions Section */}
      {suggestions.length > 0 && (
        <div className="rca-section">
          <h3 className="section-title">💡 Suggested Remediations</h3>
          <div className="remediation-cards">
            {suggestions.map((suggestion, idx) => (
              <div
                key={idx}
                className="remediation-card"
                style={{
                  borderLeftColor: getPriorityColor(suggestion.priority),
                }}
              >
                <div className="remediation-header">
                  <span className="remediation-icon">
                    {getRemediationIcon(suggestion.action_type)}
                  </span>
                  <span className="remediation-name">
                    {getRemediationLabel(suggestion.action_type)}
                  </span>
                  <span
                    className="priority-badge"
                    style={{
                      backgroundColor: getPriorityColor(
                        suggestion.priority
                      ),
                    }}
                  >
                    {suggestion.priority}
                  </span>
                </div>

                <div className="remediation-content">
                  <p className="remediation-reason">
                    <strong>Why:</strong> {suggestion.reason}
                  </p>

                  <div className="remediation-metrics">
                    <div className="metric-row">
                      <span className="metric-label">Confidence:</span>
                      <div className="metric-bar">
                        <div
                          className="metric-fill"
                          style={{
                            width: `${suggestion.confidence * 100}%`,
                            backgroundColor: getPriorityColor(
                              suggestion.priority
                            ),
                          }}
                        ></div>
                      </div>
                      <span className="metric-value">
                        {Math.round(suggestion.confidence * 100)}%
                      </span>
                    </div>

                    {suggestion.recurrence_count > 0 && (
                      <div className="metric-row">
                        <span className="metric-label">
                          Past Successes:
                        </span>
                        <span className="metric-value">
                          {suggestion.recurrence_count}
                          {suggestion.recurrence_count === 1
                            ? " incident"
                            : " incidents"}
                        </span>
                      </div>
                    )}
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
};
