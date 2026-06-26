/**
 * Process Analytics Dashboard - Enterprise Predictive Analytics
 * 
 * Features:
 * - Real-time workflow monitoring
 * - ML-powered bottleneck detection
 * - Predictive duration estimation
 * - AI-generated optimization recommendations
 * - Beautiful charts and visualizations
 */

import React, { useState, useEffect } from 'react';
import {
  TrendingUp,
  TrendingDown,
  Activity,
  AlertTriangle,
  CheckCircle2,
  Clock,
  Zap,
  Target,
  Brain,
  LineChart,
  PieChart,
  RefreshCw,
  Trophy,
} from 'lucide-react';
import { ProcessBenchmarking } from './ProcessBenchmarking';

// ============================================================================
// TYPE DEFINITIONS
// ============================================================================

interface DashboardStats {
  total_workflows: number;
  active_workflows: number;
  completed_workflows: number;
  failed_workflows: number;
  avg_duration_minutes: number;
  success_rate: number;
  active_bottlenecks: number;
  pending_optimizations: number;
  trend_data: TrendDataPoint[];
  top_bottlenecks: BottleneckAnalysis[];
}

interface TrendDataPoint {
  date: string;
  total_workflows: number;
  success_rate: number;
  avg_duration: number;
}

interface BottleneckAnalysis {
  id: string;
  workflow_type: string;
  step_name: string;
  bottleneck_type: string;
  severity: number;
  avg_duration: string;
  failure_rate: number;
  recommendation: string;
  confidence: number;
  detected_at: string;
}

interface OptimizationRecommendation {
  id: string;
  workflow_type: string;
  title: string;
  description: string;
  priority: 'high' | 'medium' | 'low';
  expected_impact: number;
  implementation: any;
  status: string;
  created_at: string;
}

interface StepPerformance {
  step_name: string;
  step_type: string;
  execution_count: number;
  avg_duration_minutes: number;
  success_rate: number;
  is_bottleneck: boolean;
}

interface PredictedDuration {
  workflow_type: string;
  predicted_minutes: number;
  confidence_interval: {
    lower: number;
    upper: number;
  };
  factors: PredictionFactor[];
}

interface PredictionFactor {
  name: string;
  impact: number;
}

interface ProcessAnalyticsDashboardProps {
  tenant: { id: string; display_name: string };
  datasource: { id: string; source_name: string };
}

// ============================================================================
// MAIN COMPONENT
// ============================================================================

export const ProcessAnalyticsDashboard: React.FC<ProcessAnalyticsDashboardProps> = ({
  tenant,
  datasource,
}) => {
  const [stats, setStats] = useState<DashboardStats | null>(null);
  const [recommendations, setRecommendations] = useState<OptimizationRecommendation[]>([]);
  const [selectedWorkflow, setSelectedWorkflow] = useState<string>('');
  const [stepPerformance, setStepPerformance] = useState<StepPerformance[]>([]);
  const [prediction, setPrediction] = useState<PredictedDuration | null>(null);
  const [loading, setLoading] = useState(true);
  const [autoRefresh, setAutoRefresh] = useState(true);
  const [viewMode, setViewMode] = useState<'overview' | 'bottlenecks' | 'recommendations' | 'predictions' | 'benchmarking'>('overview');

  // Fetch dashboard data
  const fetchDashboardData = async () => {
    try {
      setLoading(true);
      const response = await fetch(
        `/api/process-analytics/dashboard?tenant_id=${tenant.id}&tenant_instance_id=${datasource.id}`
      );
      const data = await response.json();
      setStats(data);
    } catch (error) {
      console.error('Failed to fetch dashboard stats:', error);
    } finally {
      setLoading(false);
    }
  };

  // Fetch optimization recommendations
  const fetchRecommendations = async () => {
    try {
      const response = await fetch(
        `/api/process-analytics/recommendations?tenant_id=${tenant.id}&status=pending`
      );
      const data = await response.json();
      setRecommendations(data || []);
    } catch (error) {
      console.error('Failed to fetch recommendations:', error);
    }
  };

  // Fetch step performance for selected workflow
  const fetchStepPerformance = async (workflowType: string) => {
    try {
      const response = await fetch(
        `/api/process-analytics/step-performance?tenant_id=${tenant.id}&workflow_type=${workflowType}`
      );
      const data = await response.json();
      setStepPerformance(data || []);
    } catch (error) {
      console.error('Failed to fetch step performance:', error);
    }
  };

  // Fetch duration prediction
  const fetchPrediction = async (workflowType: string) => {
    try {
      const response = await fetch(
        `/api/process-analytics/predict-duration?tenant_id=${tenant.id}&workflow_type=${workflowType}`
      );
      const data = await response.json();
      setPrediction(data);
    } catch (error) {
      console.error('Failed to fetch prediction:', error);
    }
  };

  // Run bottleneck analysis
  const runBottleneckAnalysis = async () => {
    try {
      await fetch(
        `/api/process-analytics/analyze-bottlenecks?tenant_id=${tenant.id}`,
        { method: 'POST' }
      );
      fetchDashboardData();
    } catch (error) {
      console.error('Failed to run bottleneck analysis:', error);
    }
  };

  // Initial load
  useEffect(() => {
    fetchDashboardData();
    fetchRecommendations();
  }, [tenant.id]);

  // Auto refresh every 30 seconds
  useEffect(() => {
    if (!autoRefresh) return;
    const interval = setInterval(() => {
      fetchDashboardData();
    }, 30000);
    return () => clearInterval(interval);
  }, [autoRefresh, tenant.id]);

  // Load workflow-specific data when selected
  useEffect(() => {
    if (selectedWorkflow) {
      fetchStepPerformance(selectedWorkflow);
      fetchPrediction(selectedWorkflow);
    }
  }, [selectedWorkflow]);

  // Extract workflow types from stats
  const workflowTypes = stats?.top_bottlenecks 
    ? Array.from(new Set(stats.top_bottlenecks.map(b => b.workflow_type)))
    : [];

  if (loading && !stats) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="flex flex-col items-center gap-4">
          <RefreshCw className="w-12 h-12 animate-spin text-blue-500" />
          <p className="text-gray-600">Loading analytics dashboard...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-50 via-blue-50 to-indigo-50 p-6">
      {/* Header */}
      <div className="mb-8">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-bold text-gray-900 flex items-center gap-3">
              <Brain className="w-8 h-8 text-purple-600" />
              Process Analytics Dashboard
            </h1>
            <p className="text-gray-600 mt-2">
              AI-powered insights and predictive analytics for {tenant.display_name}
            </p>
          </div>
          <div className="flex items-center gap-3">
            <button
              onClick={() => setAutoRefresh(!autoRefresh)}
              className={`px-4 py-2 rounded-lg font-medium transition-all flex items-center gap-2 ${
                autoRefresh
                  ? 'bg-green-100 text-green-700 border-2 border-green-300'
                  : 'bg-gray-100 text-gray-600 border-2 border-gray-300'
              }`}
            >
              <Activity className="w-4 h-4" />
              {autoRefresh ? 'Live' : 'Paused'}
            </button>
            <button
              onClick={fetchDashboardData}
              className="px-4 py-2 bg-blue-600 text-white rounded-lg font-medium hover:bg-blue-700 transition-all flex items-center gap-2"
            >
              <RefreshCw className="w-4 h-4" />
              Refresh
            </button>
            <button
              onClick={runBottleneckAnalysis}
              className="px-4 py-2 bg-purple-600 text-white rounded-lg font-medium hover:bg-purple-700 transition-all flex items-center gap-2"
            >
              <Zap className="w-4 h-4" />
              Analyze
            </button>
          </div>
        </div>

        {/* View Mode Tabs */}
        <div className="flex gap-2 mt-6">
          {(['overview', 'bottlenecks', 'recommendations', 'predictions', 'benchmarking'] as const).map((mode) => (
            <button
              key={mode}
              onClick={() => setViewMode(mode)}
              className={`px-6 py-3 rounded-lg font-medium transition-all flex items-center gap-2 ${
                viewMode === mode
                  ? 'bg-white text-blue-600 shadow-lg border-2 border-blue-200'
                  : 'bg-white/50 text-gray-600 hover:bg-white hover:shadow'
              }`}
            >
              {mode === 'benchmarking' && <Trophy className="w-4 h-4" />}
              {mode.charAt(0).toUpperCase() + mode.slice(1)}
            </button>
          ))}
        </div>
      </div>

      {/* KPI Cards */}
      {viewMode === 'overview' && stats && (
        <>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
            <KPICard
              title="Total Workflows"
              value={stats.total_workflows}
              icon={<Activity className="w-6 h-6" />}
              color="blue"
              trend={stats.trend_data.length > 1 ? calculateTrend(stats.trend_data.map(d => d.total_workflows)) : null}
            />
            <KPICard
              title="Success Rate"
              value={`${(stats.success_rate * 100).toFixed(1)}%`}
              icon={<CheckCircle2 className="w-6 h-6" />}
              color="green"
              trend={stats.trend_data.length > 1 ? calculateTrend(stats.trend_data.map(d => d.success_rate)) : null}
            />
            <KPICard
              title="Avg Duration"
              value={`${stats.avg_duration_minutes.toFixed(1)} min`}
              icon={<Clock className="w-6 h-6" />}
              color="purple"
              trend={stats.trend_data.length > 1 ? calculateTrend(stats.trend_data.map(d => d.avg_duration)) * -1 : null}
            />
            <KPICard
              title="Active Bottlenecks"
              value={stats.active_bottlenecks}
              icon={<AlertTriangle className="w-6 h-6" />}
              color="red"
              alert={stats.active_bottlenecks > 3}
            />
          </div>

          {/* Charts Row */}
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-8">
            <TrendChart data={stats.trend_data} />
            <SuccessRateChart
              completed={stats.completed_workflows}
              failed={stats.failed_workflows}
              active={stats.active_workflows}
            />
          </div>

          {/* Top Bottlenecks */}
          <BottlenecksSection bottlenecks={stats.top_bottlenecks} />
        </>
      )}

      {/* Bottlenecks View */}
      {viewMode === 'bottlenecks' && stats && (
        <div className="space-y-6">
          <BottlenecksSection bottlenecks={stats.top_bottlenecks} detailed />
          
          {workflowTypes.length > 0 && (
            <div className="bg-white rounded-2xl shadow-xl p-6">
              <h3 className="text-xl font-bold text-gray-900 mb-4">Step Performance Analysis</h3>
              <select
                value={selectedWorkflow}
                onChange={(e) => setSelectedWorkflow(e.target.value)}
                className="w-full p-3 border-2 border-gray-200 rounded-lg focus:outline-none focus:border-blue-500 mb-4"
                aria-label="Select workflow type for step performance analysis"
              >
                <option value="">Select a workflow type</option>
                {workflowTypes.map(type => (
                  <option key={type} value={type}>{type}</option>
                ))}
              </select>
              
              {stepPerformance.length > 0 && (
                <StepPerformanceTable steps={stepPerformance} />
              )}
            </div>
          )}
        </div>
      )}

      {/* Recommendations View */}
      {viewMode === 'recommendations' && (
        <RecommendationsSection recommendations={recommendations} onRefresh={fetchRecommendations} />
      )}

      {/* Predictions View */}
      {viewMode === 'predictions' && (
        <PredictionsSection
          workflowTypes={workflowTypes}
          selectedWorkflow={selectedWorkflow}
          onSelectWorkflow={setSelectedWorkflow}
          prediction={prediction}
        />
      )}

      {/* Benchmarking View */}
      {viewMode === 'benchmarking' && (
        <ProcessBenchmarking
          tenant={tenant}
          datasource={datasource}
          processType={selectedWorkflow}
        />
      )}
    </div>
  );
};

// ============================================================================
// KPI CARD COMPONENT
// ============================================================================

interface KPICardProps {
  title: string;
  value: string | number;
  icon: React.ReactNode;
  color: 'blue' | 'green' | 'purple' | 'red';
  trend?: number | null;
  alert?: boolean;
}

const KPICard: React.FC<KPICardProps> = ({ title, value, icon, color, trend, alert }) => {
  const colorClasses = {
    blue: 'from-blue-500 to-blue-600',
    green: 'from-green-500 to-green-600',
    purple: 'from-purple-500 to-purple-600',
    red: 'from-red-500 to-red-600',
  };

  return (
    <div className="bg-white rounded-2xl shadow-xl p-6 hover:shadow-2xl transition-all">
      <div className="flex items-center justify-between mb-4">
        <div className={`p-3 rounded-xl bg-gradient-to-br ${colorClasses[color]} text-white`}>
          {icon}
        </div>
        {trend !== null && trend !== undefined && (
          <div className={`flex items-center gap-1 text-sm font-medium ${trend > 0 ? 'text-green-600' : 'text-red-600'}`}>
            {trend > 0 ? <TrendingUp className="w-4 h-4" /> : <TrendingDown className="w-4 h-4" />}
            {Math.abs(trend).toFixed(1)}%
          </div>
        )}
      </div>
      <h3 className="text-sm font-medium text-gray-600 mb-1">{title}</h3>
      <p className={`text-3xl font-bold ${alert ? 'text-red-600' : 'text-gray-900'}`}>
        {value}
      </p>
    </div>
  );
};

// ============================================================================
// TREND CHART COMPONENT
// ============================================================================

const TrendChart: React.FC<{ data: TrendDataPoint[] }> = ({ data }) => {
  if (data.length === 0) {
    return (
      <div className="bg-white rounded-2xl shadow-xl p-6">
        <h3 className="text-xl font-bold text-gray-900 mb-4 flex items-center gap-2">
          <LineChart className="w-5 h-5 text-blue-600" />
          Workflow Trends (14 Days)
        </h3>
        <p className="text-gray-500 text-center py-8">No trend data available</p>
      </div>
    );
  }

  const maxWorkflows = Math.max(...data.map(d => d.total_workflows));

  return (
    <div className="bg-white rounded-2xl shadow-xl p-6">
      <h3 className="text-xl font-bold text-gray-900 mb-4 flex items-center gap-2">
        <LineChart className="w-5 h-5 text-blue-600" />
        Workflow Trends (14 Days)
      </h3>
      <div className="space-y-4">
        {data.map((point, index) => {
          const barHeight = (point.total_workflows / maxWorkflows) * 100;
          const date = new Date(point.date);
          const dayLabel = date.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
          
          return (
            <div key={index} className="flex items-center gap-3">
              <span className="text-xs font-medium text-gray-600 w-16">{dayLabel}</span>
              <div className="flex-1 bg-gray-100 rounded-full h-8 overflow-hidden">
                <div
                  className="bg-gradient-to-r from-blue-500 to-purple-500 h-full rounded-full flex items-center justify-end pr-3 text-white text-sm font-medium transition-all"
                  style={{ width: `${barHeight}%` }}
                >
                  {point.total_workflows > 0 && point.total_workflows}
                </div>
              </div>
              <span className="text-xs font-medium text-green-600 w-12">
                {(point.success_rate * 100).toFixed(0)}%
              </span>
            </div>
          );
        })}
      </div>
    </div>
  );
};

// ============================================================================
// SUCCESS RATE CHART COMPONENT
// ============================================================================

const SuccessRateChart: React.FC<{ completed: number; failed: number; active: number }> = ({
  completed,
  failed,
  active,
}) => {
  const total = completed + failed + active;
  const completedPercent = total > 0 ? (completed / total) * 100 : 0;
  const failedPercent = total > 0 ? (failed / total) * 100 : 0;
  const activePercent = total > 0 ? (active / total) * 100 : 0;

  return (
    <div className="bg-white rounded-2xl shadow-xl p-6">
      <h3 className="text-xl font-bold text-gray-900 mb-4 flex items-center gap-2">
        <PieChart className="w-5 h-5 text-green-600" />
        Workflow Status Distribution
      </h3>
      <div className="space-y-4">
        <div className="flex items-center gap-3">
          <div className="w-4 h-4 bg-green-500 rounded"></div>
          <span className="text-sm font-medium text-gray-700 flex-1">Completed</span>
          <span className="text-sm font-bold text-gray-900">{completed}</span>
          <span className="text-xs text-gray-500">({completedPercent.toFixed(1)}%)</span>
        </div>
        <div className="h-8 w-full bg-gray-100 rounded-full overflow-hidden flex">
          {completedPercent > 0 && (
            <div className="bg-green-500" style={{ width: `${completedPercent}%` }}></div>
          )}
          {activePercent > 0 && (
            <div className="bg-blue-500" style={{ width: `${activePercent}%` }}></div>
          )}
          {failedPercent > 0 && (
            <div className="bg-red-500" style={{ width: `${failedPercent}%` }}></div>
          )}
        </div>
        <div className="flex items-center gap-3">
          <div className="w-4 h-4 bg-blue-500 rounded"></div>
          <span className="text-sm font-medium text-gray-700 flex-1">Active</span>
          <span className="text-sm font-bold text-gray-900">{active}</span>
          <span className="text-xs text-gray-500">({activePercent.toFixed(1)}%)</span>
        </div>
        <div className="flex items-center gap-3">
          <div className="w-4 h-4 bg-red-500 rounded"></div>
          <span className="text-sm font-medium text-gray-700 flex-1">Failed</span>
          <span className="text-sm font-bold text-gray-900">{failed}</span>
          <span className="text-xs text-gray-500">({failedPercent.toFixed(1)}%)</span>
        </div>
      </div>
    </div>
  );
};

// ============================================================================
// BOTTLENECKS SECTION
// ============================================================================

const BottlenecksSection: React.FC<{ bottlenecks: BottleneckAnalysis[]; detailed?: boolean }> = ({
  bottlenecks,
  detailed = false,
}) => {
  if (bottlenecks.length === 0) {
    return (
      <div className="bg-white rounded-2xl shadow-xl p-8 text-center">
        <CheckCircle2 className="w-16 h-16 text-green-500 mx-auto mb-4" />
        <h3 className="text-xl font-bold text-gray-900 mb-2">No Bottlenecks Detected!</h3>
        <p className="text-gray-600">All workflows are performing optimally.</p>
      </div>
    );
  }

  return (
    <div className="bg-white rounded-2xl shadow-xl p-6">
      <h3 className="text-xl font-bold text-gray-900 mb-4 flex items-center gap-2">
        <AlertTriangle className="w-5 h-5 text-orange-600" />
        {detailed ? 'All Detected Bottlenecks' : 'Top Bottlenecks'}
      </h3>
      <div className="space-y-4">
        {bottlenecks.slice(0, detailed ? undefined : 5).map((bottleneck) => (
          <div
            key={bottleneck.id}
            className="border-2 border-gray-200 rounded-xl p-4 hover:border-orange-300 transition-all"
          >
            <div className="flex items-start justify-between mb-2">
              <div className="flex-1">
                <h4 className="font-bold text-gray-900">{bottleneck.step_name}</h4>
                <p className="text-sm text-gray-600">{bottleneck.workflow_type}</p>
              </div>
              <div className="flex items-center gap-2">
                <span
                  className={`px-3 py-1 rounded-full text-xs font-bold ${
                    bottleneck.severity > 0.7
                      ? 'bg-red-100 text-red-700'
                      : bottleneck.severity > 0.4
                      ? 'bg-orange-100 text-orange-700'
                      : 'bg-yellow-100 text-yellow-700'
                  }`}
                >
                  {(bottleneck.severity * 100).toFixed(0)}% Severity
                </span>
                <span className="px-3 py-1 rounded-full text-xs font-bold bg-blue-100 text-blue-700">
                  {bottleneck.bottleneck_type}
                </span>
              </div>
            </div>
            <p className="text-sm text-gray-700 mb-3">{bottleneck.recommendation}</p>
            <div className="flex items-center gap-4 text-xs text-gray-500">
              <span>Failure Rate: {(bottleneck.failure_rate * 100).toFixed(1)}%</span>
              <span>Confidence: {(bottleneck.confidence * 100).toFixed(0)}%</span>
              <span>Detected: {new Date(bottleneck.detected_at).toLocaleDateString()}</span>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
};

// ============================================================================
// STEP PERFORMANCE TABLE
// ============================================================================

const StepPerformanceTable: React.FC<{ steps: StepPerformance[] }> = ({ steps }) => {
  return (
    <div className="overflow-x-auto">
      <table className="w-full">
        <thead>
          <tr className="border-b-2 border-gray-200">
            <th className="text-left py-3 px-4 font-bold text-gray-700">Step Name</th>
            <th className="text-left py-3 px-4 font-bold text-gray-700">Type</th>
            <th className="text-right py-3 px-4 font-bold text-gray-700">Executions</th>
            <th className="text-right py-3 px-4 font-bold text-gray-700">Avg Duration</th>
            <th className="text-right py-3 px-4 font-bold text-gray-700">Success Rate</th>
            <th className="text-center py-3 px-4 font-bold text-gray-700">Status</th>
          </tr>
        </thead>
        <tbody>
          {steps.map((step, index) => (
            <tr key={index} className="border-b border-gray-100 hover:bg-gray-50">
              <td className="py-3 px-4 font-medium text-gray-900">{step.step_name}</td>
              <td className="py-3 px-4 text-gray-600">{step.step_type}</td>
              <td className="py-3 px-4 text-right text-gray-900">{step.execution_count}</td>
              <td className="py-3 px-4 text-right text-gray-900">
                {step.avg_duration_minutes.toFixed(1)} min
              </td>
              <td className="py-3 px-4 text-right">
                <span
                  className={`font-medium ${
                    step.success_rate > 0.9
                      ? 'text-green-600'
                      : step.success_rate > 0.7
                      ? 'text-yellow-600'
                      : 'text-red-600'
                  }`}
                >
                  {(step.success_rate * 100).toFixed(1)}%
                </span>
              </td>
              <td className="py-3 px-4 text-center">
                {step.is_bottleneck ? (
                  <span className="px-2 py-1 rounded-full text-xs font-bold bg-red-100 text-red-700">
                    Bottleneck
                  </span>
                ) : (
                  <span className="px-2 py-1 rounded-full text-xs font-bold bg-green-100 text-green-700">
                    Healthy
                  </span>
                )}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
};

// ============================================================================
// RECOMMENDATIONS SECTION
// ============================================================================

const RecommendationsSection: React.FC<{
  recommendations: OptimizationRecommendation[];
  onRefresh: () => void;
}> = ({ recommendations, onRefresh }) => {
  if (recommendations.length === 0) {
    return (
      <div className="bg-white rounded-2xl shadow-xl p-8 text-center">
        <Target className="w-16 h-16 text-blue-500 mx-auto mb-4" />
        <h3 className="text-xl font-bold text-gray-900 mb-2">All Optimized!</h3>
        <p className="text-gray-600">No pending optimization recommendations.</p>
      </div>
    );
  }

  return (
    <div className="bg-white rounded-2xl shadow-xl p-6">
      <div className="flex items-center justify-between mb-6">
        <h3 className="text-xl font-bold text-gray-900 flex items-center gap-2">
          <Brain className="w-5 h-5 text-purple-600" />
          AI-Generated Recommendations
        </h3>
        <button
          onClick={onRefresh}
          className="px-4 py-2 bg-purple-600 text-white rounded-lg font-medium hover:bg-purple-700 transition-all flex items-center gap-2"
        >
          <RefreshCw className="w-4 h-4" />
          Refresh
        </button>
      </div>
      <div className="space-y-4">
        {recommendations.map((rec) => (
          <div
            key={rec.id}
            className="border-2 border-gray-200 rounded-xl p-5 hover:border-purple-300 transition-all"
          >
            <div className="flex items-start justify-between mb-3">
              <div className="flex-1">
                <h4 className="font-bold text-gray-900 text-lg">{rec.title}</h4>
                <p className="text-sm text-gray-600 mt-1">{rec.workflow_type}</p>
              </div>
              <div className="flex items-center gap-2">
                <span
                  className={`px-3 py-1 rounded-full text-xs font-bold ${
                    rec.priority === 'high'
                      ? 'bg-red-100 text-red-700'
                      : rec.priority === 'medium'
                      ? 'bg-yellow-100 text-yellow-700'
                      : 'bg-blue-100 text-blue-700'
                  }`}
                >
                  {rec.priority.toUpperCase()} PRIORITY
                </span>
                <span className="px-3 py-1 rounded-full text-xs font-bold bg-green-100 text-green-700">
                  {(rec.expected_impact * 100).toFixed(0)}% Impact
                </span>
              </div>
            </div>
            <p className="text-gray-700 mb-4">{rec.description}</p>
            <div className="flex items-center justify-between">
              <span className="text-xs text-gray-500">
                Created: {new Date(rec.created_at).toLocaleDateString()}
              </span>
              <div className="flex gap-2">
                <button className="px-4 py-2 bg-green-600 text-white rounded-lg text-sm font-medium hover:bg-green-700 transition-all">
                  Implement
                </button>
                <button className="px-4 py-2 bg-gray-200 text-gray-700 rounded-lg text-sm font-medium hover:bg-gray-300 transition-all">
                  Dismiss
                </button>
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
};

// ============================================================================
// PREDICTIONS SECTION
// ============================================================================

const PredictionsSection: React.FC<{
  workflowTypes: string[];
  selectedWorkflow: string;
  onSelectWorkflow: (workflow: string) => void;
  prediction: PredictedDuration | null;
}> = ({ workflowTypes, selectedWorkflow, onSelectWorkflow, prediction }) => {
  return (
    <div className="space-y-6">
      <div className="bg-white rounded-2xl shadow-xl p-6">
        <h3 className="text-xl font-bold text-gray-900 mb-4 flex items-center gap-2">
          <Zap className="w-5 h-5 text-yellow-600" />
          Predictive Duration Analysis
        </h3>
        <p className="text-gray-600 mb-6">
          Using machine learning to predict workflow completion times based on historical data, parallel execution patterns, and current system load.
        </p>
        <select
          value={selectedWorkflow}
          onChange={(e) => onSelectWorkflow(e.target.value)}
          className="w-full p-3 border-2 border-gray-200 rounded-lg focus:outline-none focus:border-blue-500"
          aria-label="Select workflow type for duration prediction"
        >
          <option value="">Select a workflow type</option>
          {workflowTypes.map((type) => (
            <option key={type} value={type}>
              {type}
            </option>
          ))}
        </select>
      </div>

      {prediction && (
        <div className="bg-gradient-to-br from-purple-50 to-blue-50 rounded-2xl shadow-xl p-8">
          <h4 className="text-2xl font-bold text-gray-900 mb-6">
            Predicted Duration: {prediction.predicted_minutes.toFixed(1)} minutes
          </h4>
          
          <div className="bg-white rounded-xl p-6 mb-6">
            <h5 className="font-bold text-gray-900 mb-3">Confidence Interval (95%)</h5>
            <div className="flex items-center gap-4">
              <div className="flex-1">
                <div className="h-4 bg-gray-200 rounded-full overflow-hidden">
                  <div
                    className="h-full bg-gradient-to-r from-blue-500 to-purple-500"
                    style={{
                      width: '60%',
                      marginLeft: '20%',
                    }}
                  ></div>
                </div>
                <div className="flex justify-between mt-2 text-sm text-gray-600">
                  <span>{prediction.confidence_interval.lower.toFixed(1)} min</span>
                  <span className="font-bold text-purple-600">
                    {prediction.predicted_minutes.toFixed(1)} min
                  </span>
                  <span>{prediction.confidence_interval.upper.toFixed(1)} min</span>
                </div>
              </div>
            </div>
          </div>

          {prediction.factors.length > 0 && (
            <div className="bg-white rounded-xl p-6">
              <h5 className="font-bold text-gray-900 mb-4">Prediction Factors</h5>
              <div className="space-y-3">
                {prediction.factors.map((factor, index) => (
                  <div key={index} className="flex items-center justify-between">
                    <span className="text-gray-700">{factor.name}</span>
                    <span
                      className={`font-bold ${
                        factor.impact < 0 ? 'text-green-600' : 'text-red-600'
                      }`}
                    >
                      {factor.impact > 0 ? '+' : ''}
                      {(factor.impact * 100).toFixed(0)}%
                    </span>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
};

// ============================================================================
// UTILITY FUNCTIONS
// ============================================================================

function calculateTrend(values: number[]): number {
  if (values.length < 2) return 0;
  const recent = values.slice(-3).reduce((a, b) => a + b, 0) / Math.min(3, values.length);
  const older = values.slice(0, -3).reduce((a, b) => a + b, 0) / Math.max(1, values.length - 3);
  if (older === 0) return 0;
  return ((recent - older) / older) * 100;
}
