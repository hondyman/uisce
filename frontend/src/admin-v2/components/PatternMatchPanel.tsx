import React from "react";
import {
  formatEventSignature,
  formatPattern,
  getSimilarityColor,
  getSimilarityLabel,
  getConfidencePercentage,
  IncidentPattern,
  IncidentSimilarity,
} from "../hooks/usePatterns";
import "./PatternMatchPanel.css";

interface PatternMatchPanelProps {
  pattern?: IncidentPattern;
  similarities: IncidentSimilarity[];
  isLoading?: boolean;
}

export const PatternMatchPanel: React.FC<PatternMatchPanelProps> = ({
  pattern,
  similarities,
  isLoading = false,
}) => {
  if (isLoading) {
    return (
      <div className="pattern-match-panel loading">
        <div className="spinner"></div>
        <p>Analyzing incident patterns...</p>
      </div>
    );
  }

  if (!pattern) {
    return (
      <div className="pattern-match-panel">
        <p className="no-data">Unable to create pattern fingerprint.</p>
      </div>
    );
  }

  return (
    <div className="pattern-match-panel">
      {/* Pattern Fingerprint Section */}
      <div className="pattern-section">
        <h3 className="section-title">🔍 Pattern Fingerprint</h3>

        <div className="pattern-card">
          {/* Event Signature */}
          <div className="signature-section">
            <div className="signature-label">Event Sequence</div>
            <div className="signature-flow">
              {pattern.event_signature.map((event, idx) => (
                <React.Fragment key={idx}>
                  <span className="event-node">{event.replace(/_/g, " ")}</span>
                  {idx < pattern.event_signature.length - 1 && (
                    <span className="arrow">→</span>
                  )}
                </React.Fragment>
              ))}
            </div>
          </div>

          {/* Pattern Metadata Grid */}
          <div className="pattern-grid">
            <div className="pattern-item">
              <div className="item-label">Pattern ID</div>
              <div className="item-value monospace">{pattern.id.substring(0, 8)}...</div>
            </div>

            <div className="pattern-item">
              <div className="item-label">Severity</div>
              <div className="item-value">
                <span className={`severity-badge ${pattern.severity}`}>
                  {pattern.severity.charAt(0).toUpperCase() +
                    pattern.severity.slice(1)}
                </span>
              </div>
            </div>

            <div className="pattern-item">
              <div className="item-label">Timeline</div>
              <div className="item-value">
                {pattern.timeline_minutes < 60
                  ? `${pattern.timeline_minutes} min`
                  : `${Math.round(pattern.timeline_minutes / 60)} hr`}
              </div>
            </div>

            <div className="pattern-item">
              <div className="item-label">Recurrences</div>
              <div className="item-value">
                <span className="recurrence-badge">
                  {pattern.recurrence_count}
                </span>
              </div>
            </div>

            <div className="pattern-item">
              <div className="item-label">Confidence</div>
              <div className="item-value">
                <div className="confidence-bar">
                  <div
                    className="confidence-fill"
                    style={{
                      width: `${pattern.confidence * 100}%`,
                    }}
                  ></div>
                </div>
                <span className="confidence-percent">
                  {getConfidencePercentage(pattern.confidence)}%
                </span>
              </div>
            </div>
          </div>

          {/* Affected Services */}
          {pattern.affected_services.length > 0 && (
            <div className="services-list">
              <div className="services-label">Affected Services</div>
              <div className="services-items">
                {pattern.affected_services.map((service, idx) => (
                  <span key={idx} className="service-item">
                    {service}
                  </span>
                ))}
              </div>
            </div>
          )}

          {/* Successful Fixes */}
          {pattern.successful_fixes.length > 0 && (
            <div className="fixes-list">
              <div className="fixes-label">Proven Fixes</div>
              <div className="fixes-items">
                {pattern.successful_fixes.map((fix, idx) => (
                  <span key={idx} className="fix-item">
                    ✓ {fix}
                  </span>
                ))}
              </div>
            </div>
          )}
        </div>
      </div>

      {/* Similar Incidents Section */}
      {similarities.length > 0 && (
        <div className="pattern-section">
          <h3 className="section-title">
            🔗 Similar Incidents ({similarities.length})
          </h3>

          <div className="similarities-list">
            {similarities.slice(0, 5).map((similarity, idx) => (
              <div
                key={idx}
                className="similarity-card"
                style={{
                  borderLeftColor: getSimilarityColor(
                    similarity.similarity_score
                  ),
                }}
              >
                <div className="similarity-header">
                  <span className="similarity-badge">
                    {getSimilarityLabel(similarity.similarity_score)}
                  </span>
                  <span className="similarity-score">
                    {Math.round(similarity.similarity_score * 100)}%
                  </span>
                </div>

                <div className="similarity-details">
                  <div className="detail-row">
                    <span className="detail-label">Matched Events:</span>
                    <span className="detail-value">
                      {similarity.matched_events}
                    </span>
                  </div>

                  {similarity.pattern_id && (
                    <div className="detail-row">
                      <span className="detail-label">Pattern ID:</span>
                      <span className="detail-value monospace">
                        {similarity.pattern_id.substring(0, 8)}...
                      </span>
                    </div>
                  )}
                </div>

                <div className="similarity-action">
                  <a href="#" className="view-link">
                    View Similar Incident →
                  </a>
                </div>
              </div>
            ))}

            {similarities.length > 5 && (
              <div className="more-similarities">
                <a href="#" className="view-all-link">
                  View all {similarities.length} similar incidents
                </a>
              </div>
            )}
          </div>
        </div>
      )}

      {/* Pattern History Section */}
      {pattern.first_seen && (
            <div className="pattern-section">
        <h3 className="section-title">📊 Pattern History</h3>

        <div className="history-grid">
          <div className="history-item">
            <div className="item-label">First Seen</div>
            <div className="item-value">
              {new Date(pattern.first_seen).toLocaleDateString()}
            </div>
          </div>

          <div className="history-item">
            <div className="item-label">Last Seen</div>
            <div className="item-value">
              {new Date(pattern.last_seen).toLocaleDateString()}
            </div>
          </div>

          <div className="history-item">
            <div className="item-label">Average Duration</div>
            <div className="item-value">
              {pattern.average_duration < 60
                ? `${pattern.average_duration} min`
                : `${Math.round(pattern.average_duration / 60)} hr`}
            </div>
          </div>

          <div className="history-item">
            <div className="item-label">Time Between</div>
            <div className="item-value">
              {pattern.recurrence_count > 1
                ? "Multiple occurrences"
                : "First occurrence"}
            </div>
          </div>
        </div>
      </div>
      )}
    </div>
  );
};
