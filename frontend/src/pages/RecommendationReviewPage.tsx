import React, { useState, useEffect } from 'react';
import { useConfirm } from '../components/ConfirmProvider';
import {
  Plus,
  Check,
  X,
  AlertCircle,
  CheckCircle,
  Zap,
  TrendingUp,
  BarChart3,
} from 'lucide-react';
import { useTenant } from '../contexts/TenantContext';
import { getSelectedRegion } from '../lib/region';
import { devLog } from '../utils/devLogger';

interface TargetAllocation {
  symbol: string;
  targetPercentage: number;
  currentPercentage: number;
}

interface Action {
  type: 'BUY' | 'SELL' | 'HOLD';
  symbol: string;
  amount: number;
  rationale: string;
}

interface Recommendation {
  id: string;
  portfolioId: string;
  portfolioName: string;
  createdBy: string;
  title: string;
  description: string;
  type: 'rebalance' | 'tactical' | 'strategic';
  status: 'draft' | 'proposed' | 'accepted' | 'rejected' | 'implemented';
  targetAllocations: TargetAllocation[];
  recommendedActions: Action[];
  rationale: string;
  expectedReturn: number;
  timeHorizon: string;
  riskScore: number;
  priority: 'low' | 'medium' | 'high';
  createdAt: string;
  metadata?: Record<string, unknown>;
}

/**
 * Recommendation Review UI Page
 * Interface for reviewing, approving, and implementing recommendations
 */
export const RecommendationReviewPage: React.FC = () => {
  const { tenant, datasource } = useTenant();
  const confirm = useConfirm();

  const [recommendations, setRecommendations] = useState<Recommendation[]>([]);
  const [selectedRec, setSelectedRec] = useState<Recommendation | null>(null);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);

  const [showForm, setShowForm] = useState(false);
  // editingRec is reserved for future edit flow; mark as intentionally unused to avoid linter noise
  const [_editingRec, _setEditingRec] = useState<Recommendation | null>(null);
  const [statusNotes, setStatusNotes] = useState('');

  const [formData, setFormData] = useState<{
    title: string;
    description: string;
    type: 'rebalance' | 'tactical' | 'strategic';
    rationale: string;
    expectedReturn: number;
    timeHorizon: string;
    priority: 'low' | 'medium' | 'high';
    targetAllocations: TargetAllocation[];
    recommendedActions: Action[];
  }>({
    title: '',
    description: '',
    type: 'rebalance',
    rationale: '',
    expectedReturn: 0,
    timeHorizon: '6 months',
    priority: 'medium',
    targetAllocations: [] as TargetAllocation[],
    recommendedActions: [] as Action[],
  });

  const [toast, setToast] = useState<{ type: 'success' | 'error'; message: string } | null>(null);
  const [filterStatus, setFilterStatus] = useState<string>('ALL');
  const [filterType, setFilterType] = useState<string>('ALL');

  // Initialize
  useEffect(() => {
    if (tenant && datasource) {
      loadRecommendations();
      devLog('Recommendation Review initialized', { tenantId: tenant.id, datasourceId: datasource.id });
    }
  }, [tenant, datasource]);

  const showToast = (type: 'success' | 'error', message: string) => {
    setToast({ type, message });
    setTimeout(() => setToast(null), 3000);
  };

  const loadRecommendations = async () => {
    setLoading(true);
    try {
      const response = await fetch('/api/recommendations', {
        headers: {
          'X-User-ID': tenant?.id || '',
          'X-Tenant-ID': tenant?.id || '',
          'X-Tenant-Datasource-ID': datasource?.id || '',
        },
      });

      if (!response.ok) throw new Error('Failed to fetch recommendations');
      const data = await response.json();
      setRecommendations(data || []);
      if (data && data.length > 0) {
        setSelectedRec(data[0]);
      }
      devLog('Recommendations loaded', { count: data?.length || 0 });
    } catch (error) {
      console.error('Failed to load recommendations:', error);
      showToast('error', 'Failed to load recommendations');
    } finally {
      setLoading(false);
    }
  };

  const handleCreateRecommendation = async () => {
    if (!formData.title.trim()) {
      showToast('error', 'Recommendation title is required');
      return;
    }

    setSaving(true);
    try {
      const response = await fetch('/api/recommendations', {
        method: 'POST',
        headers: {
          'X-User-ID': tenant?.id || '',
          'X-Tenant-ID': tenant?.id || '',
          'X-Tenant-Datasource-ID': datasource?.id || '',
          'X-Tenant-Region': getSelectedRegion(),
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          ...formData,
          createdBy: tenant?.id,
          status: 'draft',
        }),
      });

      if (!response.ok) throw new Error('Failed to create recommendation');
      showToast('success', 'Recommendation created successfully');
      setShowForm(false);
      await loadRecommendations();
    } catch (error) {
      console.error('Failed to create recommendation:', error);
      showToast('error', 'Failed to create recommendation');
    } finally {
      setSaving(false);
    }
  };

  const handleUpdateStatus = async (recId: string, newStatus: string) => {
    setSaving(true);
    try {
      const response = await fetch(`/api/recommendation-status?id=${recId}`, {
        method: 'PATCH',
        headers: {
          'X-User-ID': tenant?.id || '',
          'X-Tenant-ID': tenant?.id || '',
          'X-Tenant-Datasource-ID': datasource?.id || '',
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          status: newStatus,
          notes: statusNotes,
        }),
      });

      if (!response.ok) throw new Error('Failed to update recommendation');
      showToast('success', 'Recommendation updated successfully');
      setStatusNotes('');
      await loadRecommendations();
    } catch (error) {
      console.error('Failed to update recommendation:', error);
      showToast('error', 'Failed to update recommendation');
    } finally {
      setSaving(false);
    }
  };

  const handleDeleteRecommendation = async (recId: string) => {
    if (!(await confirm({ title: 'Delete recommendation', description: 'Are you sure you want to delete this recommendation?' }))) return;

    setSaving(true);
    try {
      const response = await fetch(`/api/recommendations/${recId}`, {
        method: 'DELETE',
        headers: {
          'X-User-ID': tenant?.id || '',
          'X-Tenant-ID': tenant?.id || '',
          'X-Tenant-Datasource-ID': datasource?.id || '',
        },
      });

      if (!response.ok) throw new Error('Failed to delete recommendation');
      showToast('success', 'Recommendation deleted successfully');
      await loadRecommendations();
    } catch (error) {
      console.error('Failed to delete recommendation:', error);
      showToast('error', 'Failed to delete recommendation');
    } finally {
      setSaving(false);
    }
  };

  const getStatusColor = (status: string): string => {
    const colors: Record<string, string> = {
      draft: 'bg-gray-100 text-gray-800 dark:bg-gray-900/30 dark:text-gray-200',
      proposed: 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-200',
      accepted: 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-200',
      rejected: 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-200',
      implemented: 'bg-purple-100 text-purple-800 dark:bg-purple-900/30 dark:text-purple-200',
    };
    return colors[status] || 'bg-gray-100 text-gray-800';
  };

  const getTypeColor = (type: string): string => {
    const colors: Record<string, string> = {
      rebalance: 'bg-amber-100 text-amber-800 dark:bg-amber-900/30 dark:text-amber-200',
      tactical: 'bg-cyan-100 text-cyan-800 dark:bg-cyan-900/30 dark:text-cyan-200',
      strategic: 'bg-violet-100 text-violet-800 dark:bg-violet-900/30 dark:text-violet-200',
    };
    return colors[type] || 'bg-gray-100 text-gray-800';
  };

  const getPriorityColor = (priority: string): string => {
    const colors: Record<string, string> = {
      low: 'text-blue-600 dark:text-blue-400',
      medium: 'text-yellow-600 dark:text-yellow-400',
      high: 'text-red-600 dark:text-red-400',
    };
    return colors[priority] || 'text-gray-600';
  };

  const filteredRecs = recommendations.filter((rec) => {
    const statusMatch = filterStatus === 'ALL' || rec.status === filterStatus;
    const typeMatch = filterType === 'ALL' || rec.type === filterType;
    return statusMatch && typeMatch;
  });

  if (!tenant || !datasource) {
    return (
      <div className="p-8 bg-gradient-to-br from-blue-50 to-blue-50/50 dark:from-blue-950/20 dark:to-blue-950/10 rounded-lg">
        <AlertCircle className="w-6 h-6 text-yellow-600 dark:text-yellow-400 mb-2" />
        <p className="text-gray-700 dark:text-gray-300">Please select a tenant and datasource to manage recommendations.</p>
      </div>
    );
  }

  return (
    <div className="space-y-6 p-6 bg-white dark:bg-gray-900 rounded-lg shadow-sm">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold text-gray-900 dark:text-white">Recommendation Review</h1>
          <p className="text-gray-600 dark:text-gray-400 mt-1">Review and approve investment recommendations</p>
        </div>
        <button
          onClick={() => setShowForm(true)}
          className="flex items-center gap-2 px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg transition-colors"
          title="Create new recommendation"
          aria-label="Create new recommendation"
        >
          <Plus className="w-5 h-5" />
          New Recommendation
        </button>
      </div>

      {/* Toast */}
      {toast && (
        <div
          className={`p-4 rounded-lg flex items-center gap-3 ${
            toast.type === 'success'
              ? 'bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800'
              : 'bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800'
          }`}
        >
          {toast.type === 'success' ? (
            <CheckCircle className="w-5 h-5 text-green-600 dark:text-green-400" />
          ) : (
            <AlertCircle className="w-5 h-5 text-red-600 dark:text-red-400" />
          )}
          <span className={toast.type === 'success' ? 'text-green-800 dark:text-green-200' : 'text-red-800 dark:text-red-200'}>
            {toast.message}
          </span>
        </div>
      )}

      {/* Filters */}
      <div className="flex items-center gap-4">
        <select
          value={filterStatus}
          onChange={(e) => setFilterStatus(e.target.value)}
          className="px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
          title="Filter by status"
          aria-label="Filter by status"
        >
          <option value="ALL">All Status</option>
          <option value="draft">Draft</option>
          <option value="proposed">Proposed</option>
          <option value="accepted">Accepted</option>
          <option value="rejected">Rejected</option>
          <option value="implemented">Implemented</option>
        </select>

        <select
          value={filterType}
          onChange={(e) => setFilterType(e.target.value)}
          className="px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
          title="Filter by type"
          aria-label="Filter by type"
        >
          <option value="ALL">All Types</option>
          <option value="rebalance">Rebalance</option>
          <option value="tactical">Tactical</option>
          <option value="strategic">Strategic</option>
        </select>
      </div>

      {/* Recommendations List */}
      {loading ? (
        <div className="text-center py-12">
          <div className="animate-spin inline-block w-8 h-8 border-4 border-gray-300 dark:border-gray-600 border-t-blue-600 rounded-full"></div>
          <p className="text-gray-600 dark:text-gray-400 mt-4">Loading recommendations...</p>
        </div>
      ) : filteredRecs.length === 0 ? (
        <div className="text-center py-12 border-2 border-dashed border-gray-300 dark:border-gray-600 rounded-lg">
          <BarChart3 className="w-12 h-12 text-gray-400 dark:text-gray-500 mx-auto mb-4" />
          <p className="text-gray-600 dark:text-gray-400 mb-4">No recommendations found</p>
          <button
            onClick={() => setShowForm(true)}
            className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg transition-colors"
          >
            Create First Recommendation
          </button>
        </div>
      ) : (
        <div className="grid gap-4 grid-cols-1 lg:grid-cols-3">
          {/* Recommendations List */}
          <div className="lg:col-span-1 space-y-3">
            {filteredRecs.map((rec) => (
              <div
                key={rec.id}
                onClick={() => setSelectedRec(rec)}
                className={`p-4 rounded-lg border-2 cursor-pointer transition-all ${
                  selectedRec?.id === rec.id
                    ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20'
                    : 'border-gray-200 dark:border-gray-700 hover:border-blue-300 dark:hover:border-blue-600'
                }`}
              >
                <div className="flex items-start justify-between mb-2">
                  <h3 className="font-semibold text-gray-900 dark:text-white line-clamp-2">{rec.title}</h3>
                  <button
                    onClick={(e) => {
                      e.stopPropagation();
                      handleDeleteRecommendation(rec.id);
                    }}
                    className="p-1 hover:bg-red-100 dark:hover:bg-red-900/20 rounded transition-colors"
                    title="Delete recommendation"
                    aria-label="Delete recommendation"
                  >
                    <X className="w-4 h-4 text-red-600 dark:text-red-400" />
                  </button>
                </div>
                <div className="flex items-center gap-2 flex-wrap">
                  <span className={`px-2 py-1 rounded text-xs font-medium ${getTypeColor(rec.type)}`}>{rec.type}</span>
                  <span className={`px-2 py-1 rounded text-xs font-medium ${getStatusColor(rec.status)}`}>{rec.status}</span>
                </div>
                <p className="text-xs text-gray-500 dark:text-gray-400 mt-2">{rec.portfolioName}</p>
              </div>
            ))}
          </div>

          {/* Selected Recommendation Details */}
          {selectedRec && (
            <div className="lg:col-span-2 space-y-4">
              {/* Main Details */}
              <div className="border border-gray-200 dark:border-gray-700 rounded-lg p-6 space-y-4">
                <div className="flex items-start justify-between">
                  <div>
                    <h2 className="text-2xl font-bold text-gray-900 dark:text-white">{selectedRec.title}</h2>
                    <p className="text-gray-600 dark:text-gray-400 mt-1">{selectedRec.portfolioName}</p>
                  </div>
                  <div className="flex items-center gap-2">
                    <span className={`px-3 py-1 rounded-full text-sm font-medium ${getTypeColor(selectedRec.type)}`}>
                      {selectedRec.type.charAt(0).toUpperCase() + selectedRec.type.slice(1)}
                    </span>
                    <span className={`px-3 py-1 rounded-full text-sm font-medium ${getStatusColor(selectedRec.status)}`}>
                      {selectedRec.status.charAt(0).toUpperCase() + selectedRec.status.slice(1)}
                    </span>
                  </div>
                </div>

                {selectedRec.description && (
                  <div>
                    <h4 className="font-semibold text-gray-900 dark:text-white mb-2">Description</h4>
                    <p className="text-gray-600 dark:text-gray-400">{selectedRec.description}</p>
                  </div>
                )}

                {selectedRec.rationale && (
                  <div>
                    <h4 className="font-semibold text-gray-900 dark:text-white mb-2">Rationale</h4>
                    <p className="text-gray-600 dark:text-gray-400">{selectedRec.rationale}</p>
                  </div>
                )}

                {/* Key Metrics */}
                <div className="grid grid-cols-3 gap-4 pt-4 border-t border-gray-200 dark:border-gray-700">
                  <div>
                    <p className="text-sm text-gray-600 dark:text-gray-400">Expected Return</p>
                    <p className="text-lg font-semibold text-gray-900 dark:text-white">
                      {selectedRec.expectedReturn.toFixed(2)}%
                    </p>
                  </div>
                  <div>
                    <p className="text-sm text-gray-600 dark:text-gray-400">Risk Score</p>
                    <p className="text-lg font-semibold text-gray-900 dark:text-white">{selectedRec.riskScore.toFixed(1)}</p>
                  </div>
                  <div>
                    <p className="text-sm text-gray-600 dark:text-gray-400">Priority</p>
                    <p className={`text-lg font-semibold ${getPriorityColor(selectedRec.priority)}`}>
                      {selectedRec.priority.charAt(0).toUpperCase() + selectedRec.priority.slice(1)}
                    </p>
                  </div>
                </div>
              </div>

              {/* Target Allocations */}
              {selectedRec.targetAllocations.length > 0 && (
                <div className="border border-gray-200 dark:border-gray-700 rounded-lg p-6">
                  <h3 className="font-semibold text-gray-900 dark:text-white mb-4">Target Allocations</h3>
                  <div className="space-y-3">
                    {selectedRec.targetAllocations.map((alloc, idx) => (
                      <div key={idx} className="flex items-center justify-between">
                        <span className="text-gray-900 dark:text-white font-medium">{alloc.symbol}</span>
                        <div className="flex items-center gap-4">
                          <div className="flex items-center gap-1">
                            <span className="text-sm text-gray-600 dark:text-gray-400">Current:</span>
                            <span className="text-sm font-medium text-gray-900 dark:text-white">
                              {alloc.currentPercentage.toFixed(1)}%
                            </span>
                          </div>
                          <TrendingUp className="w-4 h-4 text-blue-600 dark:text-blue-400" />
                          <div className="flex items-center gap-1">
                            <span className="text-sm text-gray-600 dark:text-gray-400">Target:</span>
                            <span className="text-sm font-medium text-green-600 dark:text-green-400">
                              {alloc.targetPercentage.toFixed(1)}%
                            </span>
                          </div>
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              )}

              {/* Recommended Actions */}
              {selectedRec.recommendedActions.length > 0 && (
                <div className="border border-gray-200 dark:border-gray-700 rounded-lg p-6">
                  <h3 className="font-semibold text-gray-900 dark:text-white mb-4">Recommended Actions</h3>
                  <div className="space-y-3">
                    {selectedRec.recommendedActions.map((action, idx) => (
                      <div
                        key={idx}
                        className={`p-3 rounded-lg border ${
                          action.type === 'BUY'
                            ? 'border-green-200 dark:border-green-800 bg-green-50 dark:bg-green-900/20'
                            : action.type === 'SELL'
                            ? 'border-red-200 dark:border-red-800 bg-red-50 dark:bg-red-900/20'
                            : 'border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-900/10'
                        }`}
                      >
                        <div className="flex items-center justify-between mb-1">
                          <span className="font-medium text-gray-900 dark:text-white">{action.symbol}</span>
                          <span
                            className={`px-2 py-1 rounded text-xs font-medium ${
                              action.type === 'BUY'
                                ? 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-200'
                                : action.type === 'SELL'
                                ? 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-200'
                                : 'bg-gray-100 text-gray-800 dark:bg-gray-900/30 dark:text-gray-200'
                            }`}
                          >
                            {action.type}
                          </span>
                        </div>
                        <p className="text-sm text-gray-600 dark:text-gray-400">${action.amount.toLocaleString()}</p>
                        {action.rationale && (
                          <p className="text-xs text-gray-600 dark:text-gray-400 mt-1">{action.rationale}</p>
                        )}
                      </div>
                    ))}
                  </div>
                </div>
              )}

              {/* Status Update Section */}
              <div className="border border-gray-200 dark:border-gray-700 rounded-lg p-6 space-y-4">
                <h3 className="font-semibold text-gray-900 dark:text-white">Update Status</h3>
                <textarea
                  value={statusNotes}
                  onChange={(e) => setStatusNotes(e.target.value)}
                  placeholder="Add notes about this recommendation..."
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
                  rows={3}
                />
                <div className="flex items-center gap-3 flex-wrap">
                  {selectedRec.status !== 'proposed' && (
                    <button
                      onClick={() => handleUpdateStatus(selectedRec.id, 'proposed')}
                      disabled={saving}
                      className="flex items-center gap-2 px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg transition-colors disabled:opacity-50"
                    >
                      <Zap className="w-4 h-4" />
                      Propose
                    </button>
                  )}
                  {selectedRec.status !== 'accepted' && (
                    <button
                      onClick={() => handleUpdateStatus(selectedRec.id, 'accepted')}
                      disabled={saving}
                      className="flex items-center gap-2 px-4 py-2 bg-green-600 hover:bg-green-700 text-white rounded-lg transition-colors disabled:opacity-50"
                    >
                      <Check className="w-4 h-4" />
                      Accept
                    </button>
                  )}
                  {selectedRec.status !== 'rejected' && (
                    <button
                      onClick={() => handleUpdateStatus(selectedRec.id, 'rejected')}
                      disabled={saving}
                      className="flex items-center gap-2 px-4 py-2 bg-red-600 hover:bg-red-700 text-white rounded-lg transition-colors disabled:opacity-50"
                    >
                      <X className="w-4 h-4" />
                      Reject
                    </button>
                  )}
                  {selectedRec.status === 'accepted' && (
                    <button
                      onClick={() => handleUpdateStatus(selectedRec.id, 'implemented')}
                      disabled={saving}
                      className="flex items-center gap-2 px-4 py-2 bg-purple-600 hover:bg-purple-700 text-white rounded-lg transition-colors disabled:opacity-50"
                    >
                      <CheckCircle className="w-4 h-4" />
                      Implement
                    </button>
                  )}
                </div>
              </div>
            </div>
          )}
        </div>
      )}

      {/* Create/Edit Recommendation Modal */}
      {showForm && (
        <div className="fixed inset-0 bg-black/50 dark:bg-black/70 flex items-center justify-center z-50">
          <div className="bg-white dark:bg-gray-800 rounded-lg shadow-xl max-w-2xl w-full mx-4 max-h-[90vh] overflow-y-auto">
            <div className="sticky top-0 flex items-center justify-between p-6 border-b border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800">
              <h2 className="text-2xl font-bold text-gray-900 dark:text-white">New Recommendation</h2>
              <button
                onClick={() => setShowForm(false)}
                className="p-2 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
                title="Close form"
                aria-label="Close form"
              >
                <X className="w-6 h-6 text-gray-600 dark:text-gray-400" />
              </button>
            </div>

            <div className="p-6 space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Title *</label>
                <input
                  type="text"
                  value={formData.title}
                  onChange={(e) => setFormData({ ...formData, title: e.target.value })}
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
                  placeholder="e.g., Increase Bond Allocation"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Type *</label>
                <select
                  value={formData.type}
                  onChange={(e) => setFormData({ ...formData, type: e.target.value as 'rebalance' | 'tactical' | 'strategic' })}
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
                  title="Select recommendation type"
                  aria-label="Select recommendation type"
                >
                  <option value="rebalance">Rebalance</option>
                  <option value="tactical">Tactical</option>
                  <option value="strategic">Strategic</option>
                </select>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Description</label>
                <textarea
                  value={formData.description}
                  onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
                  placeholder="Describe the recommendation..."
                  rows={3}
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Rationale</label>
                <textarea
                  value={formData.rationale}
                  onChange={(e) => setFormData({ ...formData, rationale: e.target.value })}
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
                  placeholder="Explain the reasoning..."
                  rows={2}
                />
              </div>

              <div className="grid grid-cols-3 gap-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Expected Return %</label>
                  <input
                    type="number"
                    value={formData.expectedReturn}
                    onChange={(e) => setFormData({ ...formData, expectedReturn: parseFloat(e.target.value) })}
                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
                    step="0.1"
                    title="Expected return percentage"
                    aria-label="Expected return percentage"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Time Horizon</label>
                  <input
                    type="text"
                    value={formData.timeHorizon}
                    onChange={(e) => setFormData({ ...formData, timeHorizon: e.target.value })}
                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
                    placeholder="e.g., 6 months"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Priority</label>
                  <select
                    value={formData.priority}
                    onChange={(e) => setFormData({ ...formData, priority: e.target.value as 'low' | 'medium' | 'high' })}
                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
                    title="Select priority"
                    aria-label="Select priority"
                  >
                    <option value="low">Low</option>
                    <option value="medium">Medium</option>
                    <option value="high">High</option>
                  </select>
                </div>
              </div>
            </div>

            <div className="flex items-center justify-end gap-3 p-6 border-t border-gray-200 dark:border-gray-700">
              <button
                onClick={() => setShowForm(false)}
                className="px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={handleCreateRecommendation}
                disabled={saving}
                className="flex items-center gap-2 px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
              >
                <Plus className="w-5 h-5" />
                {saving ? 'Creating...' : 'Create'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default RecommendationReviewPage;
