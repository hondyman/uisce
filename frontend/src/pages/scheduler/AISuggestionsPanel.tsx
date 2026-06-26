import React, { useState, useEffect } from 'react';
import { useActor } from '../../contexts/ActorContext';
import './SchedulerConsole.css';

interface AISuggestion {
  id: string;
  type: 'scheduling' | 'dag_optimization' | 'new_job' | 'risk_alert' | 'consolidation' | 'semantic_drift';
  title: string;
  description: string;
  impact: string;
  riskLevel: 'low' | 'medium' | 'high';
  affectedJobs?: string[];
  affectedTenants?: string[];
  proposedFix: string;
  confidence: number;
  createdAt: string;
}

interface AISuggestionsPanelProps {
  maxItems?: number;
  showAllLink?: boolean;
  onApplyFix?: (suggestionId: string) => void;
  onDismiss?: (suggestionId: string, reason: string) => void;
}

/**
 * AISuggestionsPanel - Inline AI suggestions component
 * Shows tenant-scoped suggestions for Tenant Ops, cross-tenant for Global Ops
 */
const AISuggestionsPanel: React.FC<AISuggestionsPanelProps> = ({
  maxItems = 5,
  showAllLink = true,
  onApplyFix,
  onDismiss,
}) => {
  const { role, tenantId, permissions } = useActor();
  const [suggestions, setSuggestions] = useState<AISuggestion[]>([]);
  const [selectedSuggestion, setSelectedSuggestion] = useState<AISuggestion | null>(null);
  const [dismissReason, setDismissReason] = useState('');
  const [showDismiss, setShowDismiss] = useState(false);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchSuggestions();
  }, [tenantId, role]);

  const fetchSuggestions = async () => {
    setLoading(true);
    try {
      // Simulated data - would fetch from API
      const data: AISuggestion[] = [
        {
          id: 'sug-001',
          type: 'scheduling',
          title: 'Stagger heavy pre-agg jobs',
          description: 'Move EU Pre-Agg from 2:00 AM to 1:45 AM to reduce contention with APAC jobs.',
          impact: '15% latency reduction',
          riskLevel: 'low',
          affectedJobs: ['EU Pre-Agg'],
          affectedTenants: permissions.canViewCrossTenant ? ['T-002', 'T-005'] : undefined,
          proposedFix: 'Update schedule to 1:45 AM local time',
          confidence: 0.92,
          createdAt: '2026-01-17T08:00:00Z',
        },
        {
          id: 'sug-002',
          type: 'dag_optimization',
          title: 'Parallelize data load steps',
          description: 'Split serial data loads into parallel branches for faster execution.',
          impact: '30% faster DAG execution',
          riskLevel: 'medium',
          affectedJobs: ['Risk Batch DAG - Step 2', 'Risk Batch DAG - Step 3'],
          affectedTenants: permissions.canViewCrossTenant ? ['All'] : undefined,
          proposedFix: 'Restructure DAG to parallelize Load Data steps',
          confidence: 0.85,
          createdAt: '2026-01-17T07:30:00Z',
        },
        {
          id: 'sug-003',
          type: 'risk_alert',
          title: 'Pre-Agg approaching SLO threshold',
          description: 'Recent runs averaging 4.2s, SLO target is 5s. Suggest proactive optimization.',
          impact: 'Prevent potential SLO breach',
          riskLevel: 'high',
          affectedJobs: ['Positions Pre-Agg'],
          affectedTenants: permissions.canViewCrossTenant ? ['T-002'] : undefined,
          proposedFix: 'Increase timeout and add query optimization',
          confidence: 0.88,
          createdAt: '2026-01-17T06:00:00Z',
        },
      ];

      // Add global-ops-only suggestions
      if (permissions.canViewCrossTenant) {
        data.push(
          {
            id: 'sug-004',
            type: 'consolidation',
            title: 'Merge similar DAGs across tenants',
            description: 'Pre-Agg pipelines for T-001 and T-003 share 80% of steps.',
            impact: '20% resource reduction, simplified maintenance',
            riskLevel: 'medium',
            affectedJobs: ['T-001 Pre-Agg', 'T-003 Pre-Agg'],
            affectedTenants: ['T-001', 'T-003'],
            proposedFix: 'Create shared global DAG with tenant-specific parameters',
            confidence: 0.78,
            createdAt: '2026-01-17T05:00:00Z',
          },
          {
            id: 'sug-005',
            type: 'new_job',
            title: 'Create data quality scan',
            description: 'Recurring null value errors suggest adding proactive data quality scan.',
            impact: 'Prevent 5+ downstream failures per week',
            riskLevel: 'low',
            affectedJobs: [],
            affectedTenants: ['T-005', 'T-015'],
            proposedFix: 'Generate new Data Quality Scan job from template',
            confidence: 0.91,
            createdAt: '2026-01-17T04:00:00Z',
          }
        );
      }

      setSuggestions(data);
    } finally {
      setLoading(false);
    }
  };

  const getTypeIcon = (type: string): string => {
    switch (type) {
      case 'scheduling': return '📅';
      case 'dag_optimization': return '🔧';
      case 'new_job': return '➕';
      case 'risk_alert': return '⚠️';
      case 'consolidation': return '🔗';
      case 'semantic_drift': return '📉';
      default: return '💡';
    }
  };

  const getRiskColor = (level: string): string => {
    switch (level) {
      case 'high': return '#dc2626';
      case 'medium': return '#ca8a04';
      case 'low': return '#16a34a';
      default: return '#6b7280';
    }
  };

  const handleApply = (suggestion: AISuggestion) => {
    onApplyFix?.(suggestion.id);
    setSelectedSuggestion(null);
  };

  const handleDismiss = (suggestion: AISuggestion) => {
    if (dismissReason.trim()) {
      onDismiss?.(suggestion.id, dismissReason);
      setShowDismiss(false);
      setDismissReason('');
      setSelectedSuggestion(null);
      // Remove from list
      setSuggestions(prev => prev.filter(s => s.id !== suggestion.id));
    }
  };

  const displaySuggestions = suggestions.slice(0, maxItems);

  if (loading) {
    return (
      <div className="ai-panel-loading">
        <div className="loading-spinner" />
        Analyzing scheduling patterns...
      </div>
    );
  }

  return (
    <div className="ai-suggestions-panel">
      {/* Header */}
      <div className="panel-header">
        <h3>
          <span className="ai-icon">🤖</span>
          AI Suggestions
          {permissions.canViewCrossTenant && (
            <span className="header-badge">Cross-Tenant</span>
          )}
        </h3>
        {showAllLink && suggestions.length > maxItems && (
          <a href="/scheduler-intelligence/ai" className="view-all-link">
            View All ({suggestions.length})
          </a>
        )}
      </div>

      {/* Suggestions List */}
      <div className="suggestions-list">
        {displaySuggestions.length === 0 ? (
          <div className="empty-state">
            <span className="empty-icon">✨</span>
            <p>No suggestions at this time</p>
            <span className="empty-subtext">Everything looks optimized!</span>
          </div>
        ) : (
          displaySuggestions.map(suggestion => (
            <div
              key={suggestion.id}
              className={`suggestion-card ${selectedSuggestion?.id === suggestion.id ? 'expanded' : ''}`}
            >
              <div 
                className="suggestion-header"
                onClick={() => setSelectedSuggestion(
                  selectedSuggestion?.id === suggestion.id ? null : suggestion
                )}
              >
                <span className="type-icon">{getTypeIcon(suggestion.type)}</span>
                <div className="suggestion-title">
                  <h4>{suggestion.title}</h4>
                  <span className="suggestion-type">{suggestion.type.replace('_', ' ')}</span>
                </div>
                <div className="suggestion-meta">
                  <span 
                    className="risk-badge"
                    style={{ backgroundColor: getRiskColor(suggestion.riskLevel) }}
                  >
                    {suggestion.riskLevel}
                  </span>
                  <span className="confidence">{Math.round(suggestion.confidence * 100)}%</span>
                </div>
              </div>

              {selectedSuggestion?.id === suggestion.id && (
                <div className="suggestion-details">
                  <p className="description">{suggestion.description}</p>
                  
                  <div className="detail-row">
                    <strong>Impact:</strong>
                    <span className="impact">{suggestion.impact}</span>
                  </div>

                  {suggestion.affectedJobs && suggestion.affectedJobs.length > 0 && (
                    <div className="detail-row">
                      <strong>Affected Jobs:</strong>
                      <span>{suggestion.affectedJobs.join(', ')}</span>
                    </div>
                  )}

                  {permissions.canViewCrossTenant && suggestion.affectedTenants && (
                    <div className="detail-row">
                      <strong>Affected Tenants:</strong>
                      <span>
                        {suggestion.affectedTenants[0] === 'All' 
                          ? 'All tenants' 
                          : suggestion.affectedTenants.join(', ')}
                      </span>
                    </div>
                  )}

                  <div className="detail-row">
                    <strong>Proposed Fix:</strong>
                    <span>{suggestion.proposedFix}</span>
                  </div>

                  {!showDismiss ? (
                    suggestion.type === 'semantic_drift' ? (
                      <div className="suggestion-actions">
                        <button 
                          className="apply-btn"
                          onClick={() => window.open('/intelligence/semantic', '_blank')}
                        >
                          Open Semantic Diff
                        </button>
                        <button 
                          className="dismiss-btn"
                          onClick={() => setShowDismiss(true)}
                        >
                          Dismiss
                        </button>
                      </div>
                    ) : (
                      <div className="suggestion-actions">
                        <button 
                          className="apply-btn"
                          onClick={() => handleApply(suggestion)}
                        >
                          Apply Fix → ChangeSet
                        </button>
                        <button 
                          className="dismiss-btn"
                          onClick={() => setShowDismiss(true)}
                        >
                          Dismiss
                        </button>
                      </div>
                    )
                  ) : (
                    <div className="dismiss-form">
                      <input
                        type="text"
                        value={dismissReason}
                        onChange={e => setDismissReason(e.target.value)}
                        placeholder="Reason for dismissing (optional)"
                      />
                      <div className="dismiss-actions">
                        <button 
                          className="confirm-dismiss"
                          onClick={() => handleDismiss(suggestion)}
                        >
                          Confirm Dismiss
                        </button>
                        <button 
                          className="cancel-dismiss"
                          onClick={() => {
                            setShowDismiss(false);
                            setDismissReason('');
                          }}
                        >
                          Cancel
                        </button>
                      </div>
                    </div>
                  )}
                </div>
              )}
            </div>
          ))
        )}
      </div>
    </div>
  );
};

export default AISuggestionsPanel;
