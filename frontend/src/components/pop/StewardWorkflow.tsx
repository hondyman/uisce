import React, { useState, useEffect } from 'react';
import { devError } from '../../utils/devLogger';
import axios from 'axios';
import { useNotification } from '../../hooks/useNotification';

interface StewardAsset {
  id: string;
  name: string;
  type: 'dynamic_measure' | 'dynamic_parameter' | 'dashboard' | 'query';
  source?: string;
  sql?: string;
  parameters?: any[];
  meta?: Record<string, any>;
  status: 'draft' | 'pending_review' | 'approved' | 'rejected' | 'deprecated';
  created_by: string;
  created_at: string;
  reviewed_by?: string;
  reviewed_at?: string;
  review_notes?: string;
  golden_path: boolean;
}

interface StewardReview {
  id: string;
  asset_id: string;
  reviewer_user_id: string;
  review_type: 'initial' | 'periodic' | 'anomaly' | 'deprecation';
  overall_rating: 'excellent' | 'good' | 'needs_attention' | 'critical';
  review_notes: string;
  status: 'in_progress' | 'completed' | 'overdue';
  due_date: string;
  completed_at?: string;
  action_items?: string[];
}

interface StewardWorkflowProps {
  asset: StewardAsset;
  currentUser: string;
  onAssetUpdate?: (asset: StewardAsset) => void;
  onReviewSubmit?: (review: StewardReview) => void;
  className?: string;
}

export const StewardWorkflow: React.FC<StewardWorkflowProps> = ({
  asset,
  currentUser,
  onAssetUpdate,
  onReviewSubmit,
  className = ''
}) => {
  const [reviews, setReviews] = useState<StewardReview[]>([]);
  const [currentReview, setCurrentReview] = useState<Partial<StewardReview>>({
    asset_id: asset.id,
    reviewer_user_id: currentUser,
    review_type: 'initial',
    overall_rating: 'good',
    review_notes: '',
    status: 'in_progress',
    due_date: new Date(Date.now() + 7 * 24 * 60 * 60 * 1000).toISOString().split('T')[0], // 7 days from now
    action_items: []
  });
  const [loading, setLoading] = useState(false);
  const [showReviewForm, setShowReviewForm] = useState(false);
  const notification = useNotification();

  // Load existing reviews
  const loadReviews = React.useCallback(async () => {
    try {
      const response = await axios.get(`/api/steward/reviews/${asset.id}`);
      setReviews(response.data.reviews || []);
    } catch (error) {
  devError('Failed to load reviews:', error);
    }
  }, [asset.id]);

  useEffect(() => {
    loadReviews();
  }, [loadReviews]);

  const handleStatusChange = async (newStatus: StewardAsset['status']) => {
    setLoading(true);
    try {
      const response = await axios.patch(`/api/steward/assets/${asset.id}`, {
        status: newStatus,
        updated_by: currentUser
      });

      const updatedAsset = response.data.asset;
      onAssetUpdate?.(updatedAsset);
    } catch (error) {
  devError('Failed to update asset status:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleGoldenPathToggle = async () => {
    setLoading(true);
    try {
      const response = await axios.patch(`/api/steward/assets/${asset.id}`, {
        golden_path: !asset.golden_path,
        updated_by: currentUser
      });

      const updatedAsset = response.data.asset;
      onAssetUpdate?.(updatedAsset);
    } catch (error) {
  devError('Failed to toggle golden path:', error);
    } finally {
      setLoading(false);
    }
  };

  const submitReview = async () => {
    if (!currentReview.review_notes?.trim()) {
      notification.error('Please provide review notes');
      return;
    }

    setLoading(true);
    try {
      const reviewData = {
        ...currentReview,
        completed_at: new Date().toISOString(),
        status: 'completed' as const
      };

      const response = await axios.post('/api/steward/reviews', reviewData);
      const newReview = response.data.review;

      setReviews(prev => [newReview, ...prev]);
      setCurrentReview({
        asset_id: asset.id,
        reviewer_user_id: currentUser,
        review_type: 'periodic',
        overall_rating: 'good',
        review_notes: '',
        status: 'in_progress',
        due_date: new Date(Date.now() + 7 * 24 * 60 * 60 * 1000).toISOString().split('T')[0],
        action_items: []
      });
      setShowReviewForm(false);

      onReviewSubmit?.(newReview);
    } catch (error) {
  devError('Failed to submit review:', error);
    } finally {
      setLoading(false);
    }
  };

  const getStatusColor = (status: StewardAsset['status']) => {
    switch (status) {
      case 'approved': return 'bg-green-100 text-green-800';
      case 'pending_review': return 'bg-yellow-100 text-yellow-800';
      case 'rejected': return 'bg-red-100 text-red-800';
      case 'deprecated': return 'bg-gray-100 text-gray-800';
      default: return 'bg-blue-100 text-blue-800';
    }
  };

  const getRatingColor = (rating: StewardReview['overall_rating']) => {
    switch (rating) {
      case 'excellent': return 'text-green-600';
      case 'good': return 'text-blue-600';
      case 'needs_attention': return 'text-yellow-600';
      case 'critical': return 'text-red-600';
      default: return 'text-gray-600';
    }
  };

  return (
    <div className={`steward-workflow bg-white border rounded-lg p-6 ${className}`}>
      {/* Asset Header */}
      <div className="flex justify-between items-start mb-6">
        <div>
          <h3 className="text-lg font-semibold text-gray-900">{asset.name}</h3>
          <p className="text-sm text-gray-600">Type: {asset.type}</p>
          {asset.source && (
            <p className="text-sm text-gray-600">Source: {asset.source}</p>
          )}
        </div>
        <div className="flex items-center space-x-2">
          <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${getStatusColor(asset.status)}`}>
            {asset.status.replace('_', ' ').toUpperCase()}
          </span>
          {asset.golden_path && (
            <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-purple-100 text-purple-800">
              GOLDEN PATH
            </span>
          )}
        </div>
      </div>

      {/* Asset Details */}
      <div className="bg-gray-50 rounded-lg p-4 mb-6">
        <h4 className="text-sm font-medium text-gray-700 mb-2">Asset Details</h4>
        <div className="grid grid-cols-2 gap-4 text-sm">
          <div>
            <span className="font-medium">Created by:</span> {asset.created_by}
          </div>
          <div>
            <span className="font-medium">Created:</span> {new Date(asset.created_at).toLocaleDateString()}
          </div>
          {asset.reviewed_by && (
            <>
              <div>
                <span className="font-medium">Reviewed by:</span> {asset.reviewed_by}
              </div>
              <div>
                <span className="font-medium">Reviewed:</span> {new Date(asset.reviewed_at!).toLocaleDateString()}
              </div>
            </>
          )}
        </div>
        {asset.sql && (
          <div className="mt-3">
            <span className="font-medium text-sm">SQL:</span>
            <pre className="text-xs bg-gray-100 p-2 rounded mt-1 overflow-x-auto">{asset.sql}</pre>
          </div>
        )}
      </div>

      {/* Steward Actions */}
      <div className="border-t pt-6">
        <h4 className="text-md font-medium text-gray-900 mb-4">Steward Actions</h4>
        <div className="flex flex-wrap gap-2 mb-4">
          <button
            onClick={() => handleStatusChange('approved')}
            disabled={loading}
            className="px-3 py-1 bg-green-600 text-white text-sm rounded hover:bg-green-700 disabled:opacity-50"
          >
            Approve
          </button>
          <button
            onClick={() => handleStatusChange('pending_review')}
            disabled={loading}
            className="px-3 py-1 bg-yellow-600 text-white text-sm rounded hover:bg-yellow-700 disabled:opacity-50"
          >
            Request Review
          </button>
          <button
            onClick={() => handleStatusChange('rejected')}
            disabled={loading}
            className="px-3 py-1 bg-red-600 text-white text-sm rounded hover:bg-red-700 disabled:opacity-50"
          >
            Reject
          </button>
          <button
            onClick={() => handleGoldenPathToggle()}
            disabled={loading}
            className={`px-3 py-1 text-sm rounded disabled:opacity-50 ${
              asset.golden_path
                ? 'bg-purple-600 text-white hover:bg-purple-700'
                : 'bg-gray-600 text-white hover:bg-gray-700'
            }`}
          >
            {asset.golden_path ? 'Remove Golden Path' : 'Mark Golden Path'}
          </button>
          <button
            onClick={() => setShowReviewForm(true)}
            className="px-3 py-1 bg-blue-600 text-white text-sm rounded hover:bg-blue-700"
          >
            Add Review
          </button>
        </div>
      </div>

      {/* Review Form */}
      {showReviewForm && (
        <div className="border-t pt-6">
          <h4 className="text-md font-medium text-gray-900 mb-4">Add Review</h4>
          <div className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Review Type
              </label>
              <select
                value={currentReview.review_type}
                onChange={(e) => setCurrentReview(prev => ({ ...prev, review_type: e.target.value as any }))}
                className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                aria-label="Review Type"
              >
                <option value="initial">Initial Review</option>
                <option value="periodic">Periodic Review</option>
                <option value="anomaly">Anomaly Investigation</option>
                <option value="deprecation">Deprecation Review</option>
              </select>
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Overall Rating
              </label>
              <select
                value={currentReview.overall_rating}
                onChange={(e) => setCurrentReview(prev => ({ ...prev, overall_rating: e.target.value as any }))}
                className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                aria-label="Overall Rating"
              >
                <option value="excellent">Excellent</option>
                <option value="good">Good</option>
                <option value="needs_attention">Needs Attention</option>
                <option value="critical">Critical</option>
              </select>
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Review Notes
              </label>
              <textarea
                value={currentReview.review_notes}
                onChange={(e) => setCurrentReview(prev => ({ ...prev, review_notes: e.target.value }))}
                rows={4}
                className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                placeholder="Provide detailed review notes..."
              />
            </div>

            <div className="flex justify-end space-x-2">
              <button
                onClick={() => setShowReviewForm(false)}
                className="px-4 py-2 bg-gray-600 text-white rounded hover:bg-gray-700"
              >
                Cancel
              </button>
              <button
                onClick={submitReview}
                disabled={loading}
                className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 disabled:opacity-50"
              >
                {loading ? 'Submitting...' : 'Submit Review'}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Review History */}
      {reviews.length > 0 && (
        <div className="border-t pt-6">
          <h4 className="text-md font-medium text-gray-900 mb-4">Review History</h4>
          <div className="space-y-3">
            {reviews.map((review) => (
              <div key={review.id} className="bg-gray-50 rounded-lg p-4">
                <div className="flex justify-between items-start mb-2">
                  <div>
                    <span className="font-medium text-gray-900">{review.reviewer_user_id}</span>
                    <span className="text-sm text-gray-600 ml-2">
                      {new Date(review.completed_at || review.due_date).toLocaleDateString()}
                    </span>
                  </div>
                  <span className={`text-sm font-medium ${getRatingColor(review.overall_rating)}`}>
                    {review.overall_rating.replace('_', ' ').toUpperCase()}
                  </span>
                </div>
                <p className="text-sm text-gray-700">{review.review_notes}</p>
                {review.action_items && review.action_items.length > 0 && (
                  <div className="mt-2">
                    <span className="text-xs font-medium text-gray-600">Action Items:</span>
                    <ul className="text-xs text-gray-600 mt-1 list-disc list-inside">
                      {review.action_items.map((item, index) => (
                        <li key={index}>{item}</li>
                      ))}
                    </ul>
                  </div>
                )}
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
};

export default StewardWorkflow;
