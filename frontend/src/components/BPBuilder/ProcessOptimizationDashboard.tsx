import React, { useState, useEffect } from 'react';
import {
  Zap,
  TrendingUp,
  Clock,
  DollarSign,
  CheckCircle2,
  X,
  AlertTriangle,
  Info,
  BarChart3,
  Settings,
  Play,
  RefreshCw,
  ThumbsUp,
  ThumbsDown,
  Sparkles,
  Target,
} from 'lucide-react';

interface ProcessOptimizationDashboardProps {
  tenant: { id: string; display_name: string };
  datasource: { id: string; source_name: string };
}

interface OptimizationSuggestion {
  id: string;
  workflow_type: string;
  suggestion_type: string;
  title: string;
  description: string;
  confidence_score: number;
  expected_improvement: string;
  impact_metrics: Record<string, any>;
  target_steps: string[];
  action_details: Record<string, any>;
  based_on_executions: number;
  status: string;
  priority: string;
  created_at: string;
}

interface AppliedOptimization {
  id: string;
  suggestion_id: string;
  workflow_type: string;
  applied_at: string;
  applied_by: string;
  before_metrics: Record<string, any>;
  after_metrics: Record<string, any>;
  actual_improvement: number;
  rollback_available: boolean;
}

export const ProcessOptimizationDashboard: React.FC<ProcessOptimizationDashboardProps> = ({
  tenant,
  datasource,
}) => {
  const [suggestions, setSuggestions] = useState<OptimizationSuggestion[]>([]);
  const [appliedOptimizations, setAppliedOptimizations] = useState<AppliedOptimization[]>([]);
  const [selectedSuggestion, setSelectedSuggestion] = useState<OptimizationSuggestion | null>(null);
  const [viewMode, setViewMode] = useState<'suggestions' | 'applied' | 'auto-tune'>('suggestions');
  const [isAnalyzing, setIsAnalyzing] = useState(false);
  const [autoTuneEnabled, setAutoTuneEnabled] = useState(false);
  const [confidenceThreshold, setConfidenceThreshold] = useState(80);

  async function fetchSuggestions() {
    try {
      const params = new URLSearchParams({
        tenant_id: tenant.id,
        tenant_instance_id: datasource.id,
        status: 'pending',
      });

      const response = await fetch(`/api/process-optimization/suggestions?${params}`);
      if (!response.ok) throw new Error('Failed to fetch suggestions');

      const data = await response.json();
      setSuggestions(data || []);
    } catch (error) {
      console.error('Error fetching suggestions:', error);
    }
  }

  async function fetchApplied() {
    try {
      const params = new URLSearchParams({
        tenant_id: tenant.id,
        tenant_instance_id: datasource.id,
      });

      const response = await fetch(`/api/process-optimization/applied?${params}`);
      if (!response.ok) throw new Error('Failed to fetch applied');

      const data = await response.json();
      setAppliedOptimizations(data || []);
    } catch (error) {
      console.error('Error fetching applied:', error);
    }
  }

  async function runAnalysis() {
    setIsAnalyzing(true);
    try {
      const params = new URLSearchParams({
        tenant_id: tenant.id,
        tenant_instance_id: datasource.id,
      });

      const response = await fetch(`/api/process-optimization/analyze?${params}`, {
        method: 'POST',
      });

      if (!response.ok) throw new Error('Analysis failed');

      const data = await response.json();
      console.log(`Generated ${data.suggestions_generated} suggestions`);
      
      await fetchSuggestions();
    } catch (error) {
      console.error('Error running analysis:', error);
      alert('Failed to run analysis: ' + error);
    } finally {
      setIsAnalyzing(false);
    }
  }

  async function applySuggestion(suggestionId: string) {
    if (!confirm('Apply this optimization? This will modify your workflow definition.')) {
      return;
    }

    try {
      const params = new URLSearchParams({
        tenant_id: tenant.id,
        tenant_instance_id: datasource.id,
      });

      const response = await fetch(`/api/process-optimization/apply/${suggestionId}?${params}`, {
        method: 'POST',
      });

      if (!response.ok) throw new Error('Failed to apply optimization');

      const data = await response.json();
      console.log('Applied:', data);

      await fetchSuggestions();
      await fetchApplied();
      setSelectedSuggestion(null);
    } catch (error) {
      console.error('Error applying suggestion:', error);
      alert('Failed to apply optimization: ' + error);
    }
  }

  async function dismissSuggestion(suggestionId: string) {
    try {
      const params = new URLSearchParams({
        tenant_id: tenant.id,
        tenant_instance_id: datasource.id,
      });

      const response = await fetch(`/api/process-optimization/dismiss/${suggestionId}?${params}`, {
        method: 'POST',
      });

      if (!response.ok) throw new Error('Failed to dismiss suggestion');

      await fetchSuggestions();
      setSelectedSuggestion(null);
    } catch (error) {
      console.error('Error dismissing suggestion:', error);
    }
  }

  async function toggleAutoTune() {
    try {
      const params = new URLSearchParams({
        tenant_id: tenant.id,
        tenant_instance_id: datasource.id,
      });

      const response = await fetch(`/api/process-optimization/auto-tune/enable?${params}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          enabled: !autoTuneEnabled,
          confidence_threshold: confidenceThreshold,
          auto_apply_types: ['sla_adjustment'],
        }),
      });

      if (!response.ok) throw new Error('Failed to toggle auto-tune');

      setAutoTuneEnabled(!autoTuneEnabled);
    } catch (error) {
      console.error('Error toggling auto-tune:', error);
    }
  }

  useEffect(() => {
    fetchSuggestions();
    fetchApplied();

    const interval = setInterval(() => {
      if (viewMode === 'suggestions') fetchSuggestions();
      else if (viewMode === 'applied') fetchApplied();
    }, 60000);

    return () => clearInterval(interval);
  }, [tenant.id, datasource.id, viewMode]);

  const getPriorityColor = (priority: string) => {
    switch (priority) {
      case 'critical':
        return 'text-red-600 bg-red-100 border-red-300';
      case 'high':
        return 'text-orange-600 bg-orange-100 border-orange-300';
      case 'medium':
        return 'text-yellow-600 bg-yellow-100 border-yellow-300';
      default:
        return 'text-gray-600 bg-gray-100 border-gray-300';
    }
  };

  const getSuggestionIcon = (type: string) => {
    switch (type) {
      case 'parallel_execution':
        return <Zap size={20} className="text-purple-600" />;
      case 'reorder_steps':
        return <RefreshCw size={20} className="text-blue-600" />;
      case 'remove_step':
        return <X size={20} className="text-red-600" />;
      case 'sla_adjustment':
        return <Clock size={20} className="text-green-600" />;
      case 'resource_allocation':
        return <Settings size={20} className="text-orange-600" />;
      default:
        return <Sparkles size={20} className="text-gray-600" />;
    }
  };

  return (
    <div className="h-full flex flex-col bg-gray-50">
      {/* Header */}
      <div className="bg-white border-b px-6 py-4">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold text-gray-900 flex items-center gap-2">
              <Sparkles size={28} className="text-purple-600" />
              AI Process Optimization
            </h1>
            <p className="text-sm text-gray-500 mt-1">
              ML-powered suggestions to improve workflow performance
            </p>
          </div>
          <div className="flex items-center gap-3">
            <button
              onClick={runAnalysis}
              disabled={isAnalyzing}
              className="px-4 py-2 bg-purple-600 text-white rounded-lg hover:bg-purple-700 disabled:opacity-50 flex items-center gap-2 font-medium transition-all"
            >
              {isAnalyzing ? (
                <>
                  <RefreshCw size={18} className="animate-spin" />
                  Analyzing...
                </>
              ) : (
                <>
                  <Target size={18} />
                  Run Analysis
                </>
              )}
            </button>
          </div>
        </div>

        {/* View Tabs */}
        <div className="flex gap-2 mt-4">
          <button
            onClick={() => setViewMode('suggestions')}
            className={`px-4 py-2 rounded-lg font-medium transition-all ${
              viewMode === 'suggestions'
                ? 'bg-purple-600 text-white'
                : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
            }`}
          >
            Suggestions ({suggestions.length})
          </button>
          <button
            onClick={() => setViewMode('applied')}
            className={`px-4 py-2 rounded-lg font-medium transition-all ${
              viewMode === 'applied'
                ? 'bg-purple-600 text-white'
                : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
            }`}
          >
            Applied ({appliedOptimizations.length})
          </button>
          <button
            onClick={() => setViewMode('auto-tune')}
            className={`px-4 py-2 rounded-lg font-medium transition-all ${
              viewMode === 'auto-tune'
                ? 'bg-purple-600 text-white'
                : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
            }`}
          >
            Auto-Tune
          </button>
        </div>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-hidden flex">
        {viewMode === 'suggestions' && (
          <>
            {/* Suggestions List */}
            <div className="w-1/2 border-r overflow-y-auto p-4">
              {suggestions.length === 0 ? (
                <div className="text-center py-12 text-gray-500">
                  <Sparkles size={48} className="mx-auto mb-4 text-gray-400" />
                  <p className="text-lg font-medium">No suggestions yet</p>
                  <p className="text-sm mt-2">Click "Run Analysis" to generate optimization suggestions</p>
                </div>
              ) : (
                <div className="space-y-3">
                  {suggestions.map((suggestion) => (
                    <div
                      key={suggestion.id}
                      onClick={() => setSelectedSuggestion(suggestion)}
                      className={`bg-white rounded-lg border-2 p-4 cursor-pointer transition-all hover:shadow-md ${
                        selectedSuggestion?.id === suggestion.id
                          ? 'border-purple-500 shadow-lg'
                          : 'border-gray-200'
                      }`}
                    >
                      <div className="flex items-start gap-3">
                        <div className="p-2 rounded-lg bg-gray-50">
                          {getSuggestionIcon(suggestion.suggestion_type)}
                        </div>
                        <div className="flex-1">
                          <div className="flex items-center gap-2 mb-2">
                            <span className={`px-2 py-1 rounded-full text-xs font-medium border ${getPriorityColor(suggestion.priority)}`}>
                              {suggestion.priority}
                            </span>
                            <span className="text-xs text-gray-500">
                              {suggestion.suggestion_type.replace(/_/g, ' ')}
                            </span>
                          </div>
                          <h3 className="text-sm font-semibold text-gray-900 mb-1">
                            {suggestion.title}
                          </h3>
                          <p className="text-xs text-gray-600 line-clamp-2 mb-2">
                            {suggestion.description}
                          </p>
                          <div className="flex items-center gap-4 text-xs text-gray-500">
                            <span className="flex items-center gap-1">
                              <TrendingUp size={14} />
                              {suggestion.confidence_score}% confidence
                            </span>
                            <span className="flex items-center gap-1">
                              <BarChart3 size={14} />
                              {suggestion.based_on_executions} samples
                            </span>
                          </div>
                        </div>
                      </div>
                      <div className="mt-3 p-2 bg-green-50 rounded text-xs text-green-700 font-medium">
                        💡 {suggestion.expected_improvement}
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>

            {/* Suggestion Details */}
            <div className="w-1/2 overflow-y-auto p-4">
              {!selectedSuggestion ? (
                <div className="text-center py-12 text-gray-500">
                  <Info size={48} className="mx-auto mb-4 text-gray-400" />
                  <p className="text-lg font-medium">Select a suggestion</p>
                  <p className="text-sm mt-2">Click a suggestion to view details and take action</p>
                </div>
              ) : (
                <div>
                  <div className="bg-white rounded-lg border p-6 mb-4">
                    <div className="flex items-start justify-between mb-4">
                      <div className="flex items-center gap-3">
                        <div className="p-3 rounded-xl bg-purple-100">
                          {getSuggestionIcon(selectedSuggestion.suggestion_type)}
                        </div>
                        <div>
                          <h2 className="text-xl font-bold text-gray-900">
                            {selectedSuggestion.title}
                          </h2>
                          <p className="text-sm text-gray-500 mt-1">
                            {selectedSuggestion.workflow_type}
                          </p>
                        </div>
                      </div>
                      <button
                        onClick={() => setSelectedSuggestion(null)}
                        className="text-gray-400 hover:text-gray-600"
                      >
                        <X size={20} />
                      </button>
                    </div>

                    <div className="space-y-4">
                      <div>
                        <h4 className="text-sm font-semibold text-gray-700 mb-2">Description</h4>
                        <p className="text-sm text-gray-600">{selectedSuggestion.description}</p>
                      </div>

                      <div className="grid grid-cols-2 gap-4">
                        <div>
                          <h4 className="text-xs font-semibold text-gray-700 mb-1">Confidence Score</h4>
                          <div className="flex items-center gap-2">
                            <div className="flex-1 bg-gray-200 rounded-full h-2">
                              <div
                                className="bg-purple-600 h-2 rounded-full"
                                style={{ width: `${selectedSuggestion.confidence_score}%` }}
                              />
                            </div>
                            <span className="text-sm font-bold text-purple-600">
                              {selectedSuggestion.confidence_score}%
                            </span>
                          </div>
                        </div>
                        <div>
                          <h4 className="text-xs font-semibold text-gray-700 mb-1">Based On</h4>
                          <p className="text-sm text-gray-900">
                            {selectedSuggestion.based_on_executions} executions
                          </p>
                        </div>
                      </div>

                      {selectedSuggestion.target_steps.length > 0 && (
                        <div>
                          <h4 className="text-sm font-semibold text-gray-700 mb-2">Affected Steps</h4>
                          <div className="flex flex-wrap gap-2">
                            {selectedSuggestion.target_steps.map((step, idx) => (
                              <span
                                key={idx}
                                className="px-3 py-1 bg-blue-100 text-blue-700 rounded-full text-xs font-medium"
                              >
                                {step}
                              </span>
                            ))}
                          </div>
                        </div>
                      )}

                      <div>
                        <h4 className="text-sm font-semibold text-gray-700 mb-2">Expected Impact</h4>
                        <div className="bg-gradient-to-r from-green-50 to-emerald-50 border border-green-200 rounded-lg p-4">
                          <div className="flex items-center gap-2 mb-3">
                            <TrendingUp size={20} className="text-green-600" />
                            <span className="text-sm font-semibold text-green-800">
                              {selectedSuggestion.expected_improvement}
                            </span>
                          </div>
                          {Object.entries(selectedSuggestion.impact_metrics).map(([key, value]) => (
                            <div key={key} className="flex justify-between text-xs text-gray-600 mb-1">
                              <span>{key.replace(/_/g, ' ')}:</span>
                              <span className="font-medium">{JSON.stringify(value)}</span>
                            </div>
                          ))}
                        </div>
                      </div>

                      <div className="border-t pt-4">
                        <h4 className="text-sm font-semibold text-gray-700 mb-2">Action Details</h4>
                        <pre className="text-xs bg-gray-50 p-3 rounded overflow-x-auto">
                          {JSON.stringify(selectedSuggestion.action_details, null, 2)}
                        </pre>
                      </div>
                    </div>
                  </div>

                  {/* Action Buttons */}
                  <div className="flex gap-3">
                    <button
                      onClick={() => applySuggestion(selectedSuggestion.id)}
                      className="flex-1 px-6 py-3 bg-gradient-to-r from-green-600 to-emerald-600 text-white rounded-lg hover:from-green-700 hover:to-emerald-700 font-semibold flex items-center justify-center gap-2 shadow-lg hover:shadow-xl transition-all"
                    >
                      <CheckCircle2 size={20} />
                      Apply Optimization
                    </button>
                    <button
                      onClick={() => dismissSuggestion(selectedSuggestion.id)}
                      className="flex-1 px-6 py-3 bg-gray-200 text-gray-700 rounded-lg hover:bg-gray-300 font-semibold flex items-center justify-center gap-2 transition-all"
                    >
                      <ThumbsDown size={20} />
                      Dismiss
                    </button>
                  </div>
                </div>
              )}
            </div>
          </>
        )}

        {viewMode === 'applied' && (
          <div className="flex-1 overflow-y-auto p-4">
            {appliedOptimizations.length === 0 ? (
              <div className="text-center py-12 text-gray-500">
                <CheckCircle2 size={48} className="mx-auto mb-4 text-gray-400" />
                <p className="text-lg font-medium">No applied optimizations yet</p>
                <p className="text-sm mt-2">Applied optimizations will appear here</p>
              </div>
            ) : (
              <div className="grid grid-cols-2 gap-4">
                {appliedOptimizations.map((opt) => (
                  <div key={opt.id} className="bg-white rounded-lg border p-4">
                    <div className="flex items-center gap-2 mb-3">
                      <CheckCircle2 size={18} className="text-green-600" />
                      <h3 className="font-semibold text-gray-900">{opt.workflow_type}</h3>
                    </div>
                    <div className="space-y-2 text-sm text-gray-600">
                      <div>
                        <span className="font-medium">Applied:</span>{' '}
                        {new Date(opt.applied_at).toLocaleString()}
                      </div>
                      <div>
                        <span className="font-medium">By:</span> {opt.applied_by}
                      </div>
                      {opt.actual_improvement > 0 && (
                        <div className="flex items-center gap-2 text-green-600 font-semibold">
                          <TrendingUp size={16} />
                          {opt.actual_improvement.toFixed(1)}% improvement
                        </div>
                      )}
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        )}

        {viewMode === 'auto-tune' && (
          <div className="flex-1 overflow-y-auto p-6">
            <div className="max-w-2xl mx-auto">
              <div className="bg-white rounded-lg border p-6 mb-6">
                <h2 className="text-xl font-bold text-gray-900 mb-4">Auto-Tune Configuration</h2>
                
                <div className="space-y-6">
                  <div className="flex items-center justify-between">
                    <div>
                      <h4 className="font-semibold text-gray-900">Enable Auto-Tune</h4>
                      <p className="text-sm text-gray-600 mt-1">
                        Automatically apply low-risk optimizations
                      </p>
                    </div>
                    <button
                      onClick={toggleAutoTune}
                      className={`relative inline-flex h-8 w-14 items-center rounded-full transition-colors ${
                        autoTuneEnabled ? 'bg-purple-600' : 'bg-gray-300'
                      }`}
                    >
                      <span
                        className={`inline-block h-6 w-6 transform rounded-full bg-white transition-transform ${
                          autoTuneEnabled ? 'translate-x-7' : 'translate-x-1'
                        }`}
                      />
                    </button>
                  </div>

                  <div>
                    <label className="block text-sm font-semibold text-gray-900 mb-2">
                      Confidence Threshold: {confidenceThreshold}%
                    </label>
                    <input
                      type="range"
                      min="50"
                      max="95"
                      step="5"
                      value={confidenceThreshold}
                      onChange={(e) => setConfidenceThreshold(parseInt(e.target.value))}
                      className="w-full"
                    />
                    <p className="text-xs text-gray-600 mt-1">
                      Only apply suggestions with confidence above this threshold
                    </p>
                  </div>

                  <div className="border-t pt-6">
                    <h4 className="font-semibold text-gray-900 mb-3">Auto-Apply Types</h4>
                    <div className="space-y-2">
                      {['SLA Adjustment', 'Resource Allocation'].map((type) => (
                        <label key={type} className="flex items-center gap-2">
                          <input
                            type="checkbox"
                            defaultChecked={type === 'SLA Adjustment'}
                            className="rounded"
                          />
                          <span className="text-sm text-gray-700">{type}</span>
                        </label>
                      ))}
                    </div>
                  </div>
                </div>
              </div>

              <div className="bg-gradient-to-r from-blue-50 to-purple-50 border border-purple-200 rounded-lg p-6">
                <div className="flex items-start gap-3">
                  <Info size={24} className="text-purple-600 flex-shrink-0 mt-1" />
                  <div>
                    <h4 className="font-semibold text-purple-900 mb-2">About Auto-Tune</h4>
                    <p className="text-sm text-purple-800 mb-3">
                      Auto-Tune continuously monitors your workflows and applies safe, proven optimizations automatically.
                    </p>
                    <ul className="text-sm text-purple-700 space-y-1 list-disc list-inside">
                      <li>Only applies optimizations above confidence threshold</li>
                      <li>Rollback available for all changes</li>
                      <li>Weekly reports sent to your email</li>
                      <li>Can be disabled anytime</li>
                    </ul>
                  </div>
                </div>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
};
