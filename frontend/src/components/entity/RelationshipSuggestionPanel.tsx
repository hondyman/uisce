/**
 * RelationshipSuggestionPanel Component
 * 
 * Displays AI-powered relationship suggestions with confidence scores,
 * rationale, and one-click application UI.
 */

import React, { useState } from 'react';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '../ui/card';
import { Button } from '../ui/button';
import { Badge } from '../ui/badge';
import { Alert, AlertDescription } from '../ui/alert';
import { Progress } from '../ui/progress';
import { AlertCircle, Zap, ChevronDown, ChevronUp, Check, X } from 'lucide-react';
import type { RelationshipSuggestion } from '../../services/businessEntitySemanticService';
import './RelationshipSuggestionPanel.css';

interface RelationshipSuggestionPanelProps {
  suggestions: RelationshipSuggestion[];
  isLoading: boolean;
  error: Error | null;
  onApplySuggestion: (suggestion: RelationshipSuggestion) => Promise<void>;
  entityName: string;
}

const RelationshipSuggestionPanel: React.FC<RelationshipSuggestionPanelProps> = ({
  suggestions,
  isLoading,
  error,
  onApplySuggestion,
  entityName,
}) => {
  const [expandedId, setExpandedId] = useState<string | null>(suggestions[0]?.id || null);
  const [applyingId, setApplyingId] = useState<string | null>(null);
  const [appliedIds, setAppliedIds] = useState<Set<string>>(new Set());

  const handleApply = async (suggestion: RelationshipSuggestion) => {
    setApplyingId(suggestion.id);
    try {
      await onApplySuggestion(suggestion);
      setAppliedIds((prev) => new Set(prev).add(suggestion.id));
    } finally {
      setApplyingId(null);
    }
  };

  const getConfidenceColor = (confidence: number): string => {
    if (confidence >= 0.8) return 'text-green-600';
    if (confidence >= 0.6) return 'text-yellow-600';
    return 'text-orange-600';
  };

  const getConfidenceBgColor = (confidence: number): string => {
    if (confidence >= 0.8) return 'bg-green-100';
    if (confidence >= 0.6) return 'bg-yellow-100';
    return 'bg-orange-100';
  };

  if (isLoading) {
    return (
      <Card className="relationship-suggestion-panel">
        <CardHeader>
          <CardTitle className="text-base">Suggested Relationships</CardTitle>
          <CardDescription>AI-powered suggestions to link related objects</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex items-center justify-center gap-2 py-8">
            <div className="spinner" />
            <p className="text-sm text-gray-600">Analyzing relationships...</p>
          </div>
        </CardContent>
      </Card>
    );
  }

  if (error) {
    return (
      <Card className="relationship-suggestion-panel">
        <CardHeader>
          <CardTitle className="text-base">Suggested Relationships</CardTitle>
        </CardHeader>
        <CardContent>
          <Alert variant="destructive">
            <AlertCircle className="h-4 w-4" />
            <AlertDescription>{error.message}</AlertDescription>
          </Alert>
        </CardContent>
      </Card>
    );
  }

  if (suggestions.length === 0) {
    return (
      <Card className="relationship-suggestion-panel">
        <CardHeader>
          <CardTitle className="text-base">Suggested Relationships</CardTitle>
          <CardDescription>No relationship suggestions available</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="empty-suggestions">
            <p className="text-sm text-gray-600">
              No relationships were suggested for {entityName}. This could mean there are no
              related entities based on foreign keys or other signals.
            </p>
          </div>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card className="relationship-suggestion-panel">
      <CardHeader>
        <div className="flex items-center justify-between">
          <div>
            <CardTitle className="text-base">Suggested Relationships</CardTitle>
            <CardDescription>
              {suggestions.length} relationship{suggestions.length !== 1 ? 's' : ''} detected
            </CardDescription>
          </div>
          <div className="flex items-center gap-2">
            <Zap className="h-4 w-4 text-yellow-500" />
            <Badge variant="outline" className="text-xs">
              AI-Powered
            </Badge>
          </div>
        </div>
      </CardHeader>
      <CardContent className="space-y-3">
        {suggestions.map((suggestion, index) => (
          <div key={suggestion.id} className="suggestion-card">
            <div
              className="suggestion-header"
              onClick={() =>
                setExpandedId(expandedId === suggestion.id ? null : suggestion.id)
              }
              role="button"
              tabIndex={0}
            >
              <div className="suggestion-header-left">
                <div className="flex items-center gap-2">
                  <span className="suggestion-number">{index + 1}</span>
                  <div className="flex-1">
                    <p className="suggestion-title">
                      {entityName} → (Related Entity)
                    </p>
                    <p className="suggestion-subtitle">
                      {suggestion.rationale}
                    </p>
                  </div>
                </div>
              </div>

              <div className="suggestion-header-right">
                <div className={`confidence-badge ${getConfidenceBgColor(suggestion.confidence)}`}>
                  <span className={`confidence-text ${getConfidenceColor(suggestion.confidence)}`}>
                    {Math.round(suggestion.confidence * 100)}%
                  </span>
                </div>
                {expandedId === suggestion.id ? (
                  <ChevronUp className="h-4 w-4 text-gray-500" />
                ) : (
                  <ChevronDown className="h-4 w-4 text-gray-500" />
                )}
              </div>
            </div>

            {expandedId === suggestion.id && (
              <div className="suggestion-details">
                {/* Scoring Breakdown */}
                <div className="scoring-breakdown">
                  <h4 className="scoring-title">Confidence Breakdown</h4>
                  <div className="score-items">
                    <div className="score-item">
                      <span className="score-label">Foreign Key Presence</span>
                      <Progress value={suggestion.scoring_breakdown.fk_presence * 100} />
                      <span className="score-value">
                        {Math.round(suggestion.scoring_breakdown.fk_presence * 100)}%
                      </span>
                    </div>

                    <div className="score-item">
                      <span className="score-label">Join Frequency</span>
                      <Progress value={suggestion.scoring_breakdown.join_frequency * 100} />
                      <span className="score-value">
                        {Math.round(suggestion.scoring_breakdown.join_frequency * 100)}%
                      </span>
                    </div>

                    <div className="score-item">
                      <span className="score-label">Name Similarity</span>
                      <Progress value={suggestion.scoring_breakdown.name_similarity * 100} />
                      <span className="score-value">
                        {Math.round(suggestion.scoring_breakdown.name_similarity * 100)}%
                      </span>
                    </div>

                    <div className="score-item">
                      <span className="score-label">Text Similarity</span>
                      <Progress value={suggestion.scoring_breakdown.text_similarity * 100} />
                      <span className="score-value">
                        {Math.round(suggestion.scoring_breakdown.text_similarity * 100)}%
                      </span>
                    </div>

                    <div className="score-item">
                      <span className="score-label">Edge Type Prior</span>
                      <Progress value={suggestion.scoring_breakdown.edge_type_prior * 100} />
                      <span className="score-value">
                        {Math.round(suggestion.scoring_breakdown.edge_type_prior * 100)}%
                      </span>
                    </div>
                  </div>
                </div>

                {/* Actions */}
                <div className="suggestion-actions">
                  {appliedIds.has(suggestion.id) ? (
                    <div className="applied-status">
                      <Check className="h-4 w-4 text-green-600" />
                      <span className="text-sm text-green-600">Applied</span>
                    </div>
                  ) : (
                    <>
                      <Button
                        onClick={() => handleApply(suggestion)}
                        disabled={applyingId === suggestion.id}
                        size="sm"
                        className="apply-button"
                      >
                        {applyingId === suggestion.id ? (
                          <>
                            <div className="spinner-small" />
                            Applying...
                          </>
                        ) : (
                          <>
                            <Check className="h-4 w-4 mr-2" />
                            Accept
                          </>
                        )}
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => setExpandedId(null)}
                        className="dismiss-button"
                      >
                        <X className="h-4 w-4 mr-1" />
                        Dismiss
                      </Button>
                    </>
                  )}
                </div>
              </div>
            )}
          </div>
        ))}
      </CardContent>
    </Card>
  );
};

export default RelationshipSuggestionPanel;
