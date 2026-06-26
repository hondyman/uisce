import React, { useEffect, useState } from 'react';
import axios from 'axios';
import { devError } from '../../utils/devLogger';

interface DynamicMeasure {
  node_id: string;
  node_type: string;
  name: string;
  description: string;
  source_enum: string;
  sql: string;
  type: string;
  tags: string[];
  owner: string;
  version: string;
  golden_path: boolean;
  review_status: 'draft' | 'pending_review' | 'approved' | 'rejected' | 'deprecated';
  steward_group?: string;
  created_at: string;
  updated_at: string;
  review_comments?: ReviewComment[];
  anomaly_detection?: {
    enabled: boolean;
    method: string;
    threshold: number;
  };
}

interface ReviewComment {
  user: string;
  comment: string;
  timestamp: string;
  action: 'comment' | 'approve' | 'reject' | 'flag';
}

interface StewardCockpitProps {
  stewardUser?: string;
}

export const StewardCockpit: React.FC<StewardCockpitProps> = ({
  stewardUser = "patrick"
}) => {
  const [measures, setMeasures] = useState<DynamicMeasure[]>([]);
  const [selectedMeasure, setSelectedMeasure] = useState<DynamicMeasure | null>(null);
  const [filterStatus, setFilterStatus] = useState<string>('all');
  const [filterOwner, setFilterOwner] = useState<string>('all');
  const [loading, setLoading] = useState(true);
  const [commentText, setCommentText] = useState('');

  useEffect(() => {
    loadMeasures();
  }, []);

  const loadMeasures = async () => {
    try {
      setLoading(true);
      const response = await axios.get('/api/catalog?type=dynamic_measure');
      setMeasures(response.data);
    } catch (error) {
      devError('Failed to load measures:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleStatusChange = async (measureId: string, newStatus: string) => {
    try {
      await axios.patch(`/api/catalog/${measureId}/status`, {
        status: newStatus,
        user: stewardUser
      });
      await loadMeasures();
    } catch (error) {
      devError('Failed to update status:', error);
    }
  };

  const handleGoldenPathToggle = async (measureId: string, goldenPath: boolean) => {
    try {
      await axios.patch(`/api/catalog/${measureId}/golden-path`, {
        golden_path: goldenPath,
        user: stewardUser
      });
      await loadMeasures();
    } catch (error) {
      devError('Failed to update golden path:', error);
    }
  };

  const addComment = async (measureId: string, comment: string, action: string = 'comment') => {
    if (!comment.trim()) return;

    try {
      await axios.post(`/api/catalog/${measureId}/comment`, {
        user: stewardUser,
        comment: comment.trim(),
        action
      });
      setCommentText('');
      await loadMeasures();
    } catch (error) {
      devError('Failed to add comment:', error);
    }
  };

  const filteredMeasures = measures.filter(measure => {
    if (filterStatus !== 'all' && measure.review_status !== filterStatus) return false;
    if (filterOwner !== 'all' && measure.owner !== filterOwner) return false;
    return true;
  });

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'approved': return 'bg-green-100 text-green-800';
      case 'pending_review': return 'bg-yellow-100 text-yellow-800';
      case 'rejected': return 'bg-red-100 text-red-800';
      case 'deprecated': return 'bg-gray-100 text-gray-800';
      default: return 'bg-blue-100 text-blue-800';
    }
  };

  const getUniqueOwners = () => {
    return Array.from(new Set(measures.map(m => m.owner)));
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-indigo-600"></div>
      </div>
    );
  }

  return (
    <div className="steward-cockpit bg-white rounded-lg shadow-lg">
      <div className="px-6 py-4 border-b border-gray-200">
        <h2 className="text-xl font-semibold text-gray-900">
          🧭 Steward Review Cockpit
        </h2>
        <p className="text-sm text-gray-600 mt-1">
          Review and manage dynamic measures in your semantic layer
        </p>
      </div>

      {/* Filters */}
      <div className="px-6 py-4 bg-gray-50 border-b border-gray-200">
        <div className="flex space-x-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Status
            </label>
            <select
              value={filterStatus}
              onChange={(e) => setFilterStatus(e.target.value)}
              className="block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 text-sm"
              aria-label="Filter measures by review status"
            >
              <option value="all">All Status</option>
              <option value="draft">Draft</option>
              <option value="pending_review">Pending Review</option>
              <option value="approved">Approved</option>
              <option value="rejected">Rejected</option>
              <option value="deprecated">Deprecated</option>
            </select>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Owner
            </label>
            <select
              value={filterOwner}
              onChange={(e) => setFilterOwner(e.target.value)}
              className="block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 text-sm"
              aria-label="Filter measures by owner"
            >
              <option value="all">All Owners</option>
              {getUniqueOwners().map(owner => (
                <option key={owner} value={owner}>{owner}</option>
              ))}
            </select>
          </div>
        </div>
      </div>

      <div className="flex">
        {/* Measures List */}
        <div className="w-1/2 border-r border-gray-200">
          <div className="p-4">
            <h3 className="text-lg font-medium text-gray-900 mb-4">
              Dynamic Measures ({filteredMeasures.length})
            </h3>

            <div className="space-y-2 max-h-96 overflow-y-auto">
              {filteredMeasures.map(measure => (
                <div
                  key={measure.node_id}
                  onClick={() => setSelectedMeasure(measure)}
                  className={`p-3 rounded-md cursor-pointer border transition-colors ${
                    selectedMeasure?.node_id === measure.node_id
                      ? 'border-indigo-500 bg-indigo-50'
                      : 'border-gray-200 hover:border-gray-300'
                  }`}
                >
                  <div className="flex items-center justify-between mb-2">
                    <h4 className="text-sm font-medium text-gray-900 truncate">
                      {measure.name}
                    </h4>
                    <span className={`px-2 py-1 text-xs rounded-full ${getStatusColor(measure.review_status)}`}>
                      {measure.review_status.replace('_', ' ')}
                    </span>
                  </div>

                  <div className="text-xs text-gray-600 space-y-1">
                    <div>Source: {measure.source_enum}</div>
                    <div>Owner: {measure.owner}</div>
                    <div className="flex items-center space-x-2">
                      <span>Golden Path:</span>
                      {measure.golden_path ? (
                        <span className="text-green-600">✓</span>
                      ) : (
                        <span className="text-gray-400">✗</span>
                      )}
                    </div>
                  </div>

                  <div className="flex flex-wrap gap-1 mt-2">
                    {measure.tags.slice(0, 3).map(tag => (
                      <span
                        key={tag}
                        className="px-2 py-1 text-xs bg-gray-100 text-gray-600 rounded"
                      >
                        {tag}
                      </span>
                    ))}
                    {measure.tags.length > 3 && (
                      <span className="px-2 py-1 text-xs bg-gray-100 text-gray-600 rounded">
                        +{measure.tags.length - 3}
                      </span>
                    )}
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>

        {/* Measure Details */}
        <div className="w-1/2">
          {selectedMeasure ? (
            <div className="p-4">
              <div className="mb-6">
                <h3 className="text-lg font-medium text-gray-900 mb-2">
                  {selectedMeasure.name}
                </h3>
                <p className="text-sm text-gray-600 mb-4">
                  {selectedMeasure.description}
                </p>

                {/* Status Controls */}
                <div className="mb-4">
                  <label className="block text-sm font-medium text-gray-700 mb-2">
                    Review Status
                  </label>
                  <select
                    value={selectedMeasure.review_status}
                    onChange={(e) => handleStatusChange(selectedMeasure.node_id, e.target.value)}
                    className="block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 text-sm"
                    aria-label="Update review status for selected measure"
                  >
                    <option value="draft">Draft</option>
                    <option value="pending_review">Pending Review</option>
                    <option value="approved">Approved</option>
                    <option value="rejected">Rejected</option>
                    <option value="deprecated">Deprecated</option>
                  </select>
                </div>

                {/* Golden Path Toggle */}
                <div className="mb-4">
                  <label className="flex items-center">
                    <input
                      type="checkbox"
                      checked={selectedMeasure.golden_path}
                      onChange={(e) => handleGoldenPathToggle(selectedMeasure.node_id, e.target.checked)}
                      className="rounded border-gray-300 text-indigo-600 shadow-sm focus:border-indigo-300 focus:ring focus:ring-indigo-200 focus:ring-opacity-50"
                    />
                    <span className="ml-2 text-sm text-gray-700">Golden Path</span>
                  </label>
                </div>

                {/* Measure Details */}
                <div className="bg-gray-50 rounded-md p-4 mb-4">
                  <h4 className="text-sm font-medium text-gray-900 mb-3">Technical Details</h4>
                  <dl className="space-y-2 text-sm">
                    <div>
                      <dt className="font-medium text-gray-600">Source:</dt>
                      <dd className="text-gray-900">{selectedMeasure.source_enum}</dd>
                    </div>
                    <div>
                      <dt className="font-medium text-gray-600">Type:</dt>
                      <dd className="text-gray-900">{selectedMeasure.type}</dd>
                    </div>
                    <div>
                      <dt className="font-medium text-gray-600">SQL:</dt>
                      <dd className="text-gray-900 font-mono text-xs bg-white p-2 rounded border mt-1">
                        {selectedMeasure.sql}
                      </dd>
                    </div>
                    <div>
                      <dt className="font-medium text-gray-600">Version:</dt>
                      <dd className="text-gray-900">{selectedMeasure.version}</dd>
                    </div>
                    {selectedMeasure.steward_group && (
                      <div>
                        <dt className="font-medium text-gray-600">Steward Group:</dt>
                        <dd className="text-gray-900">{selectedMeasure.steward_group}</dd>
                      </div>
                    )}
                  </dl>
                </div>

                {/* Anomaly Detection */}
                {selectedMeasure.anomaly_detection?.enabled && (
                  <div className="bg-yellow-50 rounded-md p-4 mb-4">
                    <h4 className="text-sm font-medium text-yellow-900 mb-2">
                      🚨 Anomaly Detection Enabled
                    </h4>
                    <div className="text-sm text-yellow-800">
                      <div>Method: {selectedMeasure.anomaly_detection.method}</div>
                      <div>Threshold: {selectedMeasure.anomaly_detection.threshold}</div>
                    </div>
                  </div>
                )}

                {/* Comments Section */}
                <div className="mb-4">
                  <h4 className="text-sm font-medium text-gray-900 mb-3">Comments & Actions</h4>

                  {/* Add Comment */}
                  <div className="mb-3">
                    <textarea
                      value={commentText}
                      onChange={(e) => setCommentText(e.target.value)}
                      placeholder="Add a comment..."
                      className="block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 text-sm"
                      rows={3}
                    />
                    <div className="flex space-x-2 mt-2">
                      <button
                        onClick={() => addComment(selectedMeasure.node_id, commentText)}
                        className="px-3 py-1 text-sm bg-gray-600 text-white rounded hover:bg-gray-700"
                      >
                        Comment
                      </button>
                      <button
                        onClick={() => addComment(selectedMeasure.node_id, commentText, 'approve')}
                        className="px-3 py-1 text-sm bg-green-600 text-white rounded hover:bg-green-700"
                      >
                        Approve
                      </button>
                      <button
                        onClick={() => addComment(selectedMeasure.node_id, commentText, 'reject')}
                        className="px-3 py-1 text-sm bg-red-600 text-white rounded hover:bg-red-700"
                      >
                        Reject
                      </button>
                      <button
                        onClick={() => addComment(selectedMeasure.node_id, commentText, 'flag')}
                        className="px-3 py-1 text-sm bg-orange-600 text-white rounded hover:bg-orange-700"
                      >
                        Flag
                      </button>
                    </div>
                  </div>

                  {/* Existing Comments */}
                  <div className="space-y-3 max-h-48 overflow-y-auto">
                    {selectedMeasure.review_comments?.map((comment, index) => (
                      <div key={index} className="bg-white border rounded-md p-3">
                        <div className="flex items-center justify-between mb-2">
                          <span className="text-sm font-medium text-gray-900">
                            {comment.user}
                          </span>
                          <span className="text-xs text-gray-500">
                            {new Date(comment.timestamp).toLocaleString()}
                          </span>
                        </div>
                        <p className="text-sm text-gray-700">{comment.comment}</p>
                        {comment.action !== 'comment' && (
                          <span className={`text-xs px-2 py-1 rounded mt-2 inline-block ${
                            comment.action === 'approve' ? 'bg-green-100 text-green-800' :
                            comment.action === 'reject' ? 'bg-red-100 text-red-800' :
                            'bg-orange-100 text-orange-800'
                          }`}>
                            {comment.action}
                          </span>
                        )}
                      </div>
                    ))}
                  </div>
                </div>
              </div>
            </div>
          ) : (
            <div className="flex items-center justify-center h-full text-gray-500">
              Select a measure to view details
            </div>
          )}
        </div>
      </div>
    </div>
  );
};
