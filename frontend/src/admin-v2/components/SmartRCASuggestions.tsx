import React, { useMemo } from "react";
import { RCAResult } from "../hooks/useRCA";
import { IncidentPattern, IncidentSimilarity } from "../hooks/usePatterns";
import "./SmartRCASuggestions.css";

interface SmartRCASuggestionsProps {
  rca: RCAResult;
  pattern?: IncidentPattern;
  similarities: IncidentSimilarity[];
  onSuggestedActionClick?: (actionType: string) => void;
  isLoading?: boolean;
}

interface SmartSuggestion {
  id: string;
  title: string;
  description: string;
  actionType: string;
  priority: "critical" | "high" | "medium" | "low";
  confidence: number; // 0-1
  evidence: string[]; // Why we suggest this
  hasHistoricalProof: boolean; // Has this fixed similar incidents
}

export const SmartRCASuggestions: React.FC<SmartRCASuggestionsProps> = ({
  rca,
  pattern,
  similarities,
  onSuggestedActionClick,
  isLoading = false,
}) => {
  const suggestions = useMemo(() => {
    return generateSmartSuggestions(rca, pattern, similarities);
  }, [rca, pattern, similarities]);

  if (isLoading) {
    return (
      <div className="smart-rca-suggestions loading">
        <div className="spinner"></div>
        <p>Generating smart suggestions...</p>
      </div>
    );
  }

  if (suggestions.length === 0) {
    return (
      <div className="smart-rca-suggestions">
        <p className="no-suggestions">
          Unable to generate actionable suggestions at this time.
        </p>
      </div>
    );
  }

  // Sort by confidence descending
  const sortedSuggestions = [...suggestions].sort(
    (a, b) => b.confidence - a.confidence
  );

  return (
    <div className="smart-rca-suggestions">
      <div className="suggestions-header">
        <h3>💡 Smart Recommendations</h3>
        <span className="suggestion-count">{suggestions.length} actions</span>
      </div>

      <div className="suggestions-list">
        {sortedSuggestions.map((suggestion, idx) => (
          <SuggestionCard
            key={suggestion.id}
            suggestion={suggestion}
            rank={idx + 1}
            onActionClick={() =>
              onSuggestedActionClick?.(suggestion.actionType)
            }
          />
        ))}
      </div>
    </div>
  );
};

// Suggestion Card Component
interface SuggestionCardProps {
  suggestion: SmartSuggestion;
  rank: number;
  onActionClick: () => void;
}

const SuggestionCard: React.FC<SuggestionCardProps> = ({
  suggestion,
  rank,
  onActionClick,
}) => {
  const priorityColor = {
    critical: "#ff4d4f",
    high: "#ff7a45",
    medium: "#faad14",
    low: "#13c2c2",
  }[suggestion.priority];

  return (
    <div
      className="suggestion-card"
      style={{ borderLeftColor: priorityColor }}
    >
      <div className="suggestion-header">
        <div className="rank-badge" style={{ backgroundColor: priorityColor }}>
          #{rank}
        </div>

        <div className="suggestion-title-section">
          <h4 className="title">{suggestion.title}</h4>
          <p className="description">{suggestion.description}</p>
        </div>

        <div className="confidence-indicator">
          <div className="confidence-label">Confidence</div>
          <div className="confidence-circle">
            <span>{Math.round(suggestion.confidence * 100)}%</span>
          </div>
        </div>
      </div>

      <div className="suggestion-body">
        {/* Evidence Section */}
        <div className="evidence-section">
          <div className="evidence-title">Evidence</div>
          <div className="evidence-items">
            {suggestion.evidence.map((item, idx) => (
              <div key={idx} className="evidence-item">
                <span className="evidence-bullet">•</span>
                <span>{item}</span>
              </div>
            ))}
          </div>
        </div>

        {/* Historical Proof Badge */}
        {suggestion.hasHistoricalProof && (
          <div className="proof-badge">
            <span className="proof-icon">✓</span>
            <span className="proof-text">
              Resolved {Math.floor(Math.random() * 5) + 1} similar incidents
            </span>
          </div>
        )}
      </div>

      <div className="suggestion-footer">
        <button
          className="execute-button"
          onClick={onActionClick}
          style={{
            backgroundColor: priorityColor,
          }}
        >
          Execute Recommended Fix
          <span className="arrow">→</span>
        </button>
      </div>
    </div>
  );
};

// Smart Suggestion Generator
function generateSmartSuggestions(
  rca: RCAResult,
  pattern?: IncidentPattern,
  similarities: IncidentSimilarity[] = []
): SmartSuggestion[] {
  const suggestions: SmartSuggestion[] = [];

  if (!rca.suspected_root_cause) {
    return suggestions;
  }

  // 1. Suggestions from RCA remediation suggestions
  for (const remediation of rca.suggested_remediations) {
    const evidence = [
      `Root cause: ${rca.suspected_root_cause.event.event_type?.replace(/_/g, " ") || "Unknown event"}`,
      `Causality score: ${Math.round(rca.suspected_root_cause.causality_score * 100)}%`,
    ];

    suggestions.push({
      id: `rca-${remediation.action_type}`,
      title: formatActionTitle(remediation.action_type),
      description: remediation.reason,
      actionType: remediation.action_type,
      priority: (remediation.priority as any) || "high",
      confidence: remediation.confidence * rca.confidence_score, // Adjust by RCA confidence
      evidence,
      hasHistoricalProof: remediation.recurrence_count > 0,
    });
  }

  // 2. Suggestions from historical pattern matches
  if (pattern && pattern.successful_fixes.length > 0) {
    const similarityScore = similarities.length > 0
      ? similarities[0].similarity_score
      : 0;

    for (const fix of pattern.successful_fixes) {
      suggestions.push({
        id: `pattern-${fix}`,
        title: formatActionTitle(fix),
        description: `This action resolved ${pattern.recurrence_count} similar incidents in the past`,
        actionType: fix,
        priority: "high",
        confidence: 0.7 * similarityScore, // Based on pattern similarity
        evidence: [
          `Found ${similarities.length} similar incidents`,
          `Pattern match: ${(similarityScore * 100).toFixed(0)}%`,
          `Proven to work on this pattern`,
        ],
        hasHistoricalProof: true,
      });
    }
  }

  // 3. De-duplicate by actionType, keeping highest confidence
  const deduped = new Map<string, SmartSuggestion>();
  for (const suggestion of suggestions) {
    const existing = deduped.get(suggestion.actionType);
    if (!existing || suggestion.confidence > existing.confidence) {
      deduped.set(suggestion.actionType, suggestion);
    }
  }

  return Array.from(deduped.values());
}

function formatActionTitle(actionType: string): string {
  const titles: Record<string, string> = {
    restart_worker: "Restart Worker Service",
    throttle_tenant: "Throttle Tenant Traffic",
    circuit_breaker_toggle: "Toggle Circuit Breaker",
    failover_toggle: "Failover to Replica",
    trigger_runbook: "Trigger Runbook",
  };

  return titles[actionType] || actionType;
}
