/**
 * RuleGovernance.tsx
 * 
 * Enterprise rule governance components:
 * - Version history with diff viewer
 * - Approval workflow with comments
 * - Impact analysis
 * - Conflict detection
 * - Audit trail
 */

import React, { useState } from 'react';
import {
  History,
  GitBranch,
  MessageSquare,
  CheckCircle,
  XCircle,
  AlertTriangle,
  Clock,
  User,
  Eye,
  ChevronDown,
  ChevronRight,
  Send,
  ThumbsUp,
  ThumbsDown,
  AlertOctagon,
  FileText,
  ArrowRight,
  Diff,
  Shield,
  Zap
} from 'lucide-react';

// ============================================================================
// Types
// ============================================================================

export interface RuleVersion {
  version: number;
  status: 'draft' | 'pending' | 'approved' | 'rejected' | 'active' | 'deprecated';
  created_at: string;
  created_by: string;
  change_summary: string;
  changes: {
    field: string;
    old_value: string;
    new_value: string;
  }[];
}

export interface ApprovalComment {
  id: string;
  user_id: string;
  user_name: string;
  user_avatar?: string;
  content: string;
  timestamp: string;
  type: 'comment' | 'approval' | 'rejection' | 'request_changes';
  resolved?: boolean;
}

export interface ApprovalRequest {
  id: string;
  rule_id: string;
  rule_name: string;
  version: number;
  status: 'pending' | 'approved' | 'rejected' | 'changes_requested';
  requested_by: string;
  requested_at: string;
  reviewers: Array<{
    user_id: string;
    user_name: string;
    status: 'pending' | 'approved' | 'rejected';
    responded_at?: string;
  }>;
  comments: ApprovalComment[];
  impact_analysis?: ImpactAnalysis;
}

export interface ImpactAnalysis {
  affected_records: number;
  estimated_violations: number;
  performance_impact: 'low' | 'medium' | 'high';
  breaking_changes: string[];
  dependent_rules: string[];
  conflicts: RuleConflict[];
}

export interface RuleConflict {
  rule_id: string;
  rule_name: string;
  conflict_type: 'overlap' | 'contradiction' | 'dependency';
  description: string;
  severity: 'warning' | 'error';
  resolution_suggestion?: string;
}

export interface AuditEntry {
  id: string;
  timestamp: string;
  user_id: string;
  user_name: string;
  action: 'created' | 'updated' | 'activated' | 'deactivated' | 'deleted' | 'approved' | 'rejected' | 'promoted';
  details: string;
  metadata?: Record<string, unknown>;
}

// ============================================================================
// Version History Component
// ============================================================================

interface VersionHistoryProps {
  versions: RuleVersion[];
  currentVersion: number;
  onViewVersion: (version: number) => void;
  onRestoreVersion: (version: number) => void;
  onCompareVersions: (v1: number, v2: number) => void;
}

export const VersionHistory: React.FC<VersionHistoryProps> = ({
  versions,
  currentVersion,
  onViewVersion,
  onRestoreVersion,
  onCompareVersions
}) => {
  const [expandedVersion, setExpandedVersion] = useState<number | null>(null);
  const [compareMode, setCompareMode] = useState(false);
  const [selectedVersions, setSelectedVersions] = useState<number[]>([]);

  const getStatusColor = (status: RuleVersion['status']) => {
    switch (status) {
      case 'active': return 'bg-green-100 text-green-700';
      case 'approved': return 'bg-blue-100 text-blue-700';
      case 'pending': return 'bg-yellow-100 text-yellow-700';
      case 'rejected': return 'bg-red-100 text-red-700';
      case 'deprecated': return 'bg-gray-100 text-gray-500';
      default: return 'bg-gray-100 text-gray-600';
    }
  };

  const toggleVersionSelect = (version: number) => {
    if (selectedVersions.includes(version)) {
      setSelectedVersions(selectedVersions.filter(v => v !== version));
    } else if (selectedVersions.length < 2) {
      setSelectedVersions([...selectedVersions, version]);
    }
  };

  return (
    <div className="bg-white rounded-lg border">
      <div className="flex items-center justify-between p-4 border-b">
        <div className="flex items-center gap-2">
          <History size={20} className="text-gray-500" />
          <h3 className="font-semibold text-gray-900">Version History</h3>
          <span className="text-sm text-gray-500">({versions.length} versions)</span>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={() => {
              setCompareMode(!compareMode);
              setSelectedVersions([]);
            }}
            className={`flex items-center gap-1 px-3 py-1.5 rounded-lg text-sm ${
              compareMode ? 'bg-purple-100 text-purple-700' : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
            }`}
          >
            <Diff size={14} />
            {compareMode ? 'Cancel Compare' : 'Compare Versions'}
          </button>
          {compareMode && selectedVersions.length === 2 && (
            <button
              onClick={() => onCompareVersions(selectedVersions[0], selectedVersions[1])}
              className="flex items-center gap-1 px-3 py-1.5 bg-purple-600 text-white rounded-lg text-sm hover:bg-purple-700"
            >
              Compare Selected
            </button>
          )}
        </div>
      </div>

      <div className="divide-y max-h-96 overflow-y-auto">
        {versions.map((version) => (
          <div 
            key={version.version}
            className={`p-4 hover:bg-gray-50 ${version.version === currentVersion ? 'bg-blue-50' : ''}`}
          >
            <div className="flex items-center gap-3">
              {compareMode && (
                <input
                  type="checkbox"
                  checked={selectedVersions.includes(version.version)}
                  onChange={() => toggleVersionSelect(version.version)}
                  disabled={!selectedVersions.includes(version.version) && selectedVersions.length >= 2}
                  className="rounded"
                  title={`Select version ${version.version} for comparison`}
                  aria-label={`Select version ${version.version} for comparison`}
                />
              )}
              
              <button
                onClick={() => setExpandedVersion(expandedVersion === version.version ? null : version.version)}
                className="p-1 hover:bg-gray-200 rounded"
              >
                {expandedVersion === version.version ? <ChevronDown size={16} /> : <ChevronRight size={16} />}
              </button>

              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2">
                  <span className="font-semibold text-gray-900">v{version.version}</span>
                  <span className={`px-2 py-0.5 rounded text-xs font-medium ${getStatusColor(version.status)}`}>
                    {version.status}
                  </span>
                  {version.version === currentVersion && (
                    <span className="px-2 py-0.5 bg-blue-600 text-white rounded text-xs">current</span>
                  )}
                </div>
                <p className="text-sm text-gray-600 truncate mt-0.5">{version.change_summary}</p>
                <div className="flex items-center gap-3 text-xs text-gray-500 mt-1">
                  <span className="flex items-center gap-1">
                    <User size={12} /> {version.created_by}
                  </span>
                  <span className="flex items-center gap-1">
                    <Clock size={12} /> {new Date(version.created_at).toLocaleString()}
                  </span>
                </div>
              </div>

              {!compareMode && (
                <div className="flex items-center gap-2">
                  <button
                    onClick={() => onViewVersion(version.version)}
                    className="p-2 hover:bg-gray-200 rounded text-gray-500"
                    title="View this version"
                  >
                    <Eye size={16} />
                  </button>
                  {version.version !== currentVersion && (
                    <button
                      onClick={() => onRestoreVersion(version.version)}
                      className="px-3 py-1.5 text-sm bg-gray-100 hover:bg-gray-200 rounded"
                    >
                      Restore
                    </button>
                  )}
                </div>
              )}
            </div>

            {expandedVersion === version.version && version.changes.length > 0 && (
              <div className="mt-3 ml-10 space-y-2">
                <div className="text-xs font-semibold text-gray-500 uppercase">Changes</div>
                {version.changes.map((change, idx) => (
                  <div key={idx} className="flex items-start gap-2 text-sm bg-gray-50 rounded p-2">
                    <span className="font-medium text-gray-700 min-w-[100px]">{change.field}:</span>
                    <span className="text-red-600 line-through">{change.old_value || '(empty)'}</span>
                    <ArrowRight size={14} className="text-gray-400 flex-shrink-0 mt-0.5" />
                    <span className="text-green-600">{change.new_value || '(empty)'}</span>
                  </div>
                ))}
              </div>
            )}
          </div>
        ))}
      </div>
    </div>
  );
};

// ============================================================================
// Approval Workflow Component
// ============================================================================

interface ApprovalWorkflowProps {
  request: ApprovalRequest;
  currentUserId: string;
  onApprove: () => void;
  onReject: (reason: string) => void;
  onRequestChanges: (feedback: string) => void;
  onAddComment: (comment: string) => void;
}

export const ApprovalWorkflow: React.FC<ApprovalWorkflowProps> = ({
  request,
  currentUserId,
  onApprove,
  onReject,
  onRequestChanges: _onRequestChanges,
  onAddComment
}) => {
  const [comment, setComment] = useState('');
  const [rejectReason, setRejectReason] = useState('');
  const [showRejectModal, setShowRejectModal] = useState(false);

  const currentUserReviewer = request.reviewers.find(r => r.user_id === currentUserId);
  const canReview = currentUserReviewer && currentUserReviewer.status === 'pending';

  const handleSubmitComment = () => {
    if (comment.trim()) {
      onAddComment(comment);
      setComment('');
    }
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'approved': return <CheckCircle size={16} className="text-green-500" />;
      case 'rejected': return <XCircle size={16} className="text-red-500" />;
      case 'pending': return <Clock size={16} className="text-yellow-500" />;
      default: return <Clock size={16} className="text-gray-400" />;
    }
  };

  return (
    <div className="bg-white rounded-lg border">
      {/* Header */}
      <div className="p-4 border-b">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <GitBranch size={20} className="text-purple-500" />
            <h3 className="font-semibold text-gray-900">Approval Request</h3>
            <span className={`px-2 py-0.5 rounded text-xs font-medium ${
              request.status === 'approved' ? 'bg-green-100 text-green-700' :
              request.status === 'rejected' ? 'bg-red-100 text-red-700' :
              request.status === 'changes_requested' ? 'bg-orange-100 text-orange-700' :
              'bg-yellow-100 text-yellow-700'
            }`}>
              {request.status.replace('_', ' ')}
            </span>
          </div>
          <div className="text-sm text-gray-500">
            Requested {new Date(request.requested_at).toLocaleDateString()}
          </div>
        </div>
      </div>

      {/* Reviewers */}
      <div className="p-4 border-b bg-gray-50">
        <div className="text-sm font-semibold text-gray-700 mb-2">Reviewers</div>
        <div className="flex flex-wrap gap-2">
          {request.reviewers.map((reviewer) => (
            <div 
              key={reviewer.user_id}
              className="flex items-center gap-2 px-3 py-1.5 bg-white rounded-lg border"
            >
              {getStatusIcon(reviewer.status)}
              <span className="text-sm font-medium">{reviewer.user_name}</span>
            </div>
          ))}
        </div>
      </div>

      {/* Impact Analysis */}
      {request.impact_analysis && (
        <div className="p-4 border-b">
          <div className="flex items-center gap-2 mb-3">
            <Zap size={16} className="text-orange-500" />
            <span className="text-sm font-semibold text-gray-700">Impact Analysis</span>
          </div>
          <div className="grid grid-cols-3 gap-4 mb-3">
            <div className="bg-gray-50 rounded p-3">
              <div className="text-2xl font-bold text-gray-900">
                {request.impact_analysis.affected_records.toLocaleString()}
              </div>
              <div className="text-xs text-gray-500">Affected Records</div>
            </div>
            <div className="bg-gray-50 rounded p-3">
              <div className="text-2xl font-bold text-orange-600">
                {request.impact_analysis.estimated_violations.toLocaleString()}
              </div>
              <div className="text-xs text-gray-500">Est. Violations</div>
            </div>
            <div className="bg-gray-50 rounded p-3">
              <div className={`text-2xl font-bold ${
                request.impact_analysis.performance_impact === 'high' ? 'text-red-600' :
                request.impact_analysis.performance_impact === 'medium' ? 'text-yellow-600' :
                'text-green-600'
              }`}>
                {request.impact_analysis.performance_impact.toUpperCase()}
              </div>
              <div className="text-xs text-gray-500">Performance Impact</div>
            </div>
          </div>

          {/* Conflicts */}
          {request.impact_analysis.conflicts.length > 0 && (
            <div className="mt-3">
              <div className="flex items-center gap-2 text-sm font-semibold text-red-700 mb-2">
                <AlertOctagon size={14} />
                {request.impact_analysis.conflicts.length} Conflicts Detected
              </div>
              <div className="space-y-2">
                {request.impact_analysis.conflicts.map((conflict, idx) => (
                  <div key={idx} className={`p-2 rounded text-sm ${
                    conflict.severity === 'error' ? 'bg-red-50 border border-red-200' : 'bg-yellow-50 border border-yellow-200'
                  }`}>
                    <div className="font-medium">{conflict.rule_name}</div>
                    <div className="text-gray-600">{conflict.description}</div>
                    {conflict.resolution_suggestion && (
                      <div className="text-blue-600 mt-1">💡 {conflict.resolution_suggestion}</div>
                    )}
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      )}

      {/* Comments */}
      <div className="p-4 border-b max-h-64 overflow-y-auto">
        <div className="flex items-center gap-2 mb-3">
          <MessageSquare size={16} className="text-gray-500" />
          <span className="text-sm font-semibold text-gray-700">Discussion</span>
        </div>
        <div className="space-y-3">
          {request.comments.length === 0 ? (
            <div className="text-center py-4 text-gray-400 text-sm">No comments yet</div>
          ) : (
            request.comments.map((comment) => (
              <div key={comment.id} className={`p-3 rounded-lg ${
                comment.type === 'approval' ? 'bg-green-50 border border-green-200' :
                comment.type === 'rejection' ? 'bg-red-50 border border-red-200' :
                comment.type === 'request_changes' ? 'bg-orange-50 border border-orange-200' :
                'bg-gray-50'
              }`}>
                <div className="flex items-center justify-between mb-1">
                  <div className="flex items-center gap-2">
                    <span className="font-medium text-sm">{comment.user_name}</span>
                    {comment.type !== 'comment' && (
                      <span className={`px-1.5 py-0.5 rounded text-xs ${
                        comment.type === 'approval' ? 'bg-green-200 text-green-800' :
                        comment.type === 'rejection' ? 'bg-red-200 text-red-800' :
                        'bg-orange-200 text-orange-800'
                      }`}>
                        {comment.type.replace('_', ' ')}
                      </span>
                    )}
                  </div>
                  <span className="text-xs text-gray-500">
                    {new Date(comment.timestamp).toLocaleString()}
                  </span>
                </div>
                <p className="text-sm text-gray-700">{comment.content}</p>
              </div>
            ))
          )}
        </div>
      </div>

      {/* Actions */}
      <div className="p-4">
        {/* Add Comment */}
        <div className="flex gap-2 mb-4">
          <input
            type="text"
            value={comment}
            onChange={(e) => setComment(e.target.value)}
            placeholder="Add a comment..."
            className="flex-1 px-3 py-2 border rounded-lg text-sm"
            onKeyDown={(e) => e.key === 'Enter' && handleSubmitComment()}
          />
          <button
            onClick={handleSubmitComment}
            disabled={!comment.trim()}
            className="px-4 py-2 bg-gray-100 text-gray-700 rounded-lg hover:bg-gray-200 disabled:opacity-50"
            title="Send comment"
            aria-label="Send comment"
          >
            <Send size={16} />
          </button>
        </div>

        {/* Review Actions */}
        {canReview && (
          <div className="flex items-center gap-2">
            <button
              onClick={onApprove}
              className="flex-1 flex items-center justify-center gap-2 px-4 py-2 bg-green-600 text-white rounded-lg hover:bg-green-700"
            >
              <ThumbsUp size={16} />
              Approve
            </button>
            <button
              onClick={() => setShowRejectModal(true)}
              className="flex-1 flex items-center justify-center gap-2 px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700"
            >
              <ThumbsDown size={16} />
              Reject
            </button>
            <button
              onClick={() => setShowRejectModal(true)}
              className="flex-1 flex items-center justify-center gap-2 px-4 py-2 border border-orange-500 text-orange-600 rounded-lg hover:bg-orange-50"
            >
              <AlertTriangle size={16} />
              Request Changes
            </button>
          </div>
        )}
      </div>

      {/* Reject Modal */}
      {showRejectModal && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg shadow-xl w-full max-w-md p-6">
            <h4 className="font-semibold text-lg mb-4">Provide Feedback</h4>
            <textarea
              value={rejectReason}
              onChange={(e) => setRejectReason(e.target.value)}
              placeholder="Explain your decision..."
              className="w-full px-3 py-2 border rounded-lg text-sm min-h-[100px]"
            />
            <div className="flex justify-end gap-2 mt-4">
              <button
                onClick={() => setShowRejectModal(false)}
                className="px-4 py-2 border rounded-lg hover:bg-gray-50"
              >
                Cancel
              </button>
              <button
                onClick={() => {
                  onReject(rejectReason);
                  setShowRejectModal(false);
                  setRejectReason('');
                }}
                disabled={!rejectReason.trim()}
                className="px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 disabled:opacity-50"
              >
                Submit
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

// ============================================================================
// Audit Trail Component
// ============================================================================

interface AuditTrailProps {
  entries: AuditEntry[];
}

export const AuditTrail: React.FC<AuditTrailProps> = ({ entries }) => {
  const getActionIcon = (action: AuditEntry['action']) => {
    switch (action) {
      case 'created': return <FileText size={16} className="text-blue-500" />;
      case 'updated': return <History size={16} className="text-orange-500" />;
      case 'activated': return <CheckCircle size={16} className="text-green-500" />;
      case 'deactivated': return <XCircle size={16} className="text-gray-500" />;
      case 'deleted': return <XCircle size={16} className="text-red-500" />;
      case 'approved': return <ThumbsUp size={16} className="text-green-500" />;
      case 'rejected': return <ThumbsDown size={16} className="text-red-500" />;
      case 'promoted': return <GitBranch size={16} className="text-purple-500" />;
      default: return <Clock size={16} className="text-gray-400" />;
    }
  };

  return (
    <div className="bg-white rounded-lg border">
      <div className="flex items-center gap-2 p-4 border-b">
        <Shield size={20} className="text-gray-500" />
        <h3 className="font-semibold text-gray-900">Audit Trail</h3>
      </div>

      <div className="divide-y max-h-96 overflow-y-auto">
        {entries.map((entry) => (
          <div key={entry.id} className="p-4 hover:bg-gray-50">
            <div className="flex items-start gap-3">
              <div className="mt-1">{getActionIcon(entry.action)}</div>
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2">
                  <span className="font-medium text-gray-900">{entry.user_name}</span>
                  <span className={`px-2 py-0.5 rounded text-xs font-medium ${
                    entry.action === 'created' ? 'bg-blue-100 text-blue-700' :
                    entry.action === 'updated' ? 'bg-orange-100 text-orange-700' :
                    entry.action === 'activated' ? 'bg-green-100 text-green-700' :
                    entry.action === 'deleted' ? 'bg-red-100 text-red-700' :
                    'bg-gray-100 text-gray-700'
                  }`}>
                    {entry.action}
                  </span>
                </div>
                <p className="text-sm text-gray-600 mt-1">{entry.details}</p>
                <div className="text-xs text-gray-500 mt-1">
                  {new Date(entry.timestamp).toLocaleString()}
                </div>
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
};

export default { VersionHistory, ApprovalWorkflow, AuditTrail };
