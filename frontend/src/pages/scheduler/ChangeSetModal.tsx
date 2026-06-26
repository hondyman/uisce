import React, { useState } from 'react';
import { useActor } from '../../contexts/ActorContext';
import './SchedulerConsole.css';

interface ChangeSetDiff {
  field: string;
  oldValue: string;
  newValue: string;
}

interface ImpactAnalysis {
  affectedJobs: string[];
  affectedDAGs: string[];
  affectedTenants: string[];
  downstreamImpact: number;
  sloImpact: 'none' | 'potential' | 'likely';
  complianceImpact: string;
  dataResidencyRisk: boolean;
}

interface AIReviewSummary {
  summary: string;
  riskAssessment: string;
  recommendations: string[];
  confidence: number;
}

interface ChangeSetData {
  id: string;
  type: string;
  title: string;
  description: string;
  createdBy: string;
  createdAt: string;
  status: 'draft' | 'pending_review' | 'approved' | 'rejected' | 'applied';
  targetId?: string;
  targetName?: string;
  riskScore: number;
  diffs: ChangeSetDiff[];
  impactAnalysis: ImpactAnalysis;
  aiReview?: AIReviewSummary;
  requiredApprovers: number;
  currentApprovals: number;
}

interface ChangeSetModalProps {
  isOpen: boolean;
  onClose: () => void;
  changeSet: ChangeSetData | null;
  mode: 'view' | 'create' | 'approve';
  onSubmit?: (changeSet: ChangeSetData) => void;
  onApprove?: (changeSetId: string, comment: string) => void;
  onReject?: (changeSetId: string, reason: string) => void;
}

/**
 * ChangeSetModal - Governance workflow modal for viewing/creating/approving changes
 */
const ChangeSetModal: React.FC<ChangeSetModalProps> = ({
  isOpen,
  onClose,
  changeSet,
  mode,
  onSubmit,
  onApprove,
  onReject,
}) => {
  const { permissions, role } = useActor();
  const [comment, setComment] = useState('');
  const [rejectReason, setRejectReason] = useState('');
  const [showRejectForm, setShowRejectForm] = useState(false);
  const [riskAcknowledged, setRiskAcknowledged] = useState(false);

  if (!isOpen || !changeSet) return null;

  const isHighRisk = changeSet.riskScore >= 0.7;

  const getRiskColor = (score: number): string => {
    if (score >= 0.7) return '#dc2626';
    if (score >= 0.4) return '#ea580c';
    if (score >= 0.2) return '#ca8a04';
    return '#16a34a';
  };

  const getStatusBadge = (status: string): string => {
    switch (status) {
      case 'draft': return 'status-draft';
      case 'pending_review': return 'status-pending';
      case 'approved': return 'status-approved';
      case 'rejected': return 'status-rejected';
      case 'applied': return 'status-applied';
      default: return '';
    }
  };

  const handleApprove = () => {
    if (onApprove) {
      onApprove(changeSet.id, comment);
      onClose();
    }
  };

  const handleReject = () => {
    if (onReject && rejectReason.trim()) {
      onReject(changeSet.id, rejectReason);
      onClose();
    }
  };

  const handleSubmit = () => {
    if (onSubmit) {
      onSubmit(changeSet);
      onClose();
    }
  };

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="changeset-modal" onClick={e => e.stopPropagation()}>
        {/* Header */}
        <div className="modal-header">
          <div className="header-content">
            <span className={`status-badge ${getStatusBadge(changeSet.status)}`}>
              {changeSet.status.replace('_', ' ')}
            </span>
            <h2>{changeSet.title}</h2>
            <p className="changeset-type">{changeSet.type}</p>
          </div>
          <button className="close-btn" onClick={onClose}>×</button>
        </div>

        {/* Content */}
        <div className="modal-content">
          {/* Risk Score */}
          <div className="risk-section">
            <div className="risk-indicator">
              <span 
                className="risk-circle"
                style={{ backgroundColor: getRiskColor(changeSet.riskScore) }}
              >
                {Math.round(changeSet.riskScore * 100)}%
              </span>
              <span className="risk-label">Risk Score</span>
            </div>
            <div className="approval-status">
              <span className="approvals">
                {changeSet.currentApprovals} / {changeSet.requiredApprovers} approvals
              </span>
            </div>
          </div>

          {/* Diff View */}
          <div className="diff-section">
            <h3>Changes</h3>
            <div className="diff-list">
              {changeSet.diffs.map((diff, i) => (
                <div key={i} className="diff-item">
                  <span className="diff-field">{diff.field}</span>
                  <div className="diff-values">
                    <span className="diff-old">- {diff.oldValue}</span>
                    <span className="diff-new">+ {diff.newValue}</span>
                  </div>
                </div>
              ))}
            </div>
          </div>

          {/* Impact Analysis */}
          <div className="impact-section">
            <h3>Impact Analysis</h3>
            <div className="impact-grid">
              {changeSet.impactAnalysis.affectedJobs.length > 0 && (
                <div className="impact-item">
                  <span className="impact-label">Affected Jobs</span>
                  <span className="impact-value">
                    {changeSet.impactAnalysis.affectedJobs.join(', ')}
                  </span>
                </div>
              )}
              {changeSet.impactAnalysis.affectedTenants.length > 0 && (
                <div className="impact-item">
                  <span className="impact-label">Affected Tenants</span>
                  <span className="impact-value">
                    {changeSet.impactAnalysis.affectedTenants.join(', ')}
                  </span>
                </div>
              )}
              <div className="impact-item">
                <span className="impact-label">Downstream Impact</span>
                <span className="impact-value">
                  {changeSet.impactAnalysis.downstreamImpact} jobs
                </span>
              </div>
              <div className="impact-item">
                <span className="impact-label">SLO Impact</span>
                <span className={`impact-value slo-${changeSet.impactAnalysis.sloImpact}`}>
                  {changeSet.impactAnalysis.sloImpact}
                </span>
              </div>
              {changeSet.impactAnalysis.dataResidencyRisk && (
                <div className="impact-item warning">
                  <span className="impact-label">⚠️ Data Residency</span>
                  <span className="impact-value">Risk Detected</span>
                </div>
              )}
            </div>
          </div>

          {/* AI Review */}
          {changeSet.aiReview && (
            <div className="ai-review-section">
              <h3>
                <span className="ai-icon">🤖</span>
                AI Review Summary
              </h3>
              <div className="ai-content">
                <p className="ai-summary">{changeSet.aiReview.summary}</p>
                <div className="ai-risk">
                  <strong>Risk Assessment:</strong> {changeSet.aiReview.riskAssessment}
                </div>
                {changeSet.aiReview.recommendations.length > 0 && (
                  <div className="ai-recommendations">
                    <strong>Recommendations:</strong>
                    <ul>
                      {changeSet.aiReview.recommendations.map((rec, i) => (
                        <li key={i}>{rec}</li>
                      ))}
                    </ul>
                  </div>
                )}
                <div className="ai-confidence">
                  Confidence: {Math.round(changeSet.aiReview.confidence * 100)}%
                </div>
              </div>
            </div>
          )}

          {/* Approval Controls (for Global Ops in approve mode) */}
          {mode === 'approve' && permissions.canApproveChangeSets && (
            <div className="approval-section">
              <h3>Approval Decision</h3>
              
              {!showRejectForm ? (
                <>
                  <div className="comment-input">
                    <label>Comment (optional)</label>
                    <textarea
                      value={comment}
                      onChange={e => setComment(e.target.value)}
                      placeholder="Add a comment for the approval..."
                    />
                  </div>
                  <div className="approval-actions">
                    <button className="approve-btn" onClick={handleApprove}>
                      ✓ Approve
                    </button>
                    <button 
                      className="reject-btn" 
                      onClick={() => setShowRejectForm(true)}
                    >
                      ✗ Reject
                    </button>
                  </div>
                </>
              ) : (
                <>
                  <div className="reject-input">
                    <label>Rejection Reason (required)</label>
                    <textarea
                      value={rejectReason}
                      onChange={e => setRejectReason(e.target.value)}
                      placeholder="Provide a reason for rejection..."
                      required
                    />
                  </div>
                  <div className="approval-actions">
                    <button 
                      className="reject-btn" 
                      onClick={handleReject}
                      disabled={!rejectReason.trim()}
                    >
                      Confirm Rejection
                    </button>
                    <button 
                      className="cancel-btn" 
                      onClick={() => setShowRejectForm(false)}
                    >
                      Cancel
                    </button>
                  </div>
                </>
              )}
            </div>
          )}

          {/* Submit for Review (for create mode) */}
          {mode === 'create' && (
            <div className="submit-section">
              {isHighRisk && (
                <div className="risk-acknowledgment">
                  <label className="checkbox-label">
                    <input 
                      type="checkbox" 
                      checked={riskAcknowledged}
                      onChange={e => setRiskAcknowledged(e.target.checked)}
                    />
                    I acknowledge this is a HIGH RISK change ({Math.round(changeSet.riskScore * 100)}%).
                    It may require dry-run verification or ops approval.
                  </label>
                </div>
              )}
              <button 
                className="submit-btn" 
                onClick={handleSubmit}
                disabled={isHighRisk && !riskAcknowledged}
              >
                Submit for Review
              </button>
              <p className="submit-note">
                This will create a ChangeSet that requires approval before taking effect.
              </p>
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="modal-footer">
          <div className="footer-meta">
            <span>Created by {changeSet.createdBy}</span>
            <span>•</span>
            <span>{new Date(changeSet.createdAt).toLocaleString()}</span>
          </div>
          <button className="close-btn-secondary" onClick={onClose}>
            Close
          </button>
        </div>
      </div>
    </div>
  );
};

export default ChangeSetModal;
