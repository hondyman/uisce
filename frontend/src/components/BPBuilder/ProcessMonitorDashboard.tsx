import React, { useState, useEffect, useMemo } from 'react';
import {
  Activity,
  AlertCircle,
  CheckCircle2,
  Clock,
  Filter,
  Play,
  Pause,
  X,
  SkipForward,
  UserPlus,
  RefreshCw,
  Wifi,
  WifiOff,
} from 'lucide-react';
import { useProcessMonitorWebSocket, ProcessEvent } from '../hooks/useProcessMonitorWebSocket';

interface ProcessMonitorDashboardProps {
  tenant: { id: string; display_name: string };
  datasource: { id: string; source_name: string };
}

interface ProcessInstance {
  workflow_id: string;
  workflow_type: string;
  status: string; // running, completed, failed
  current_step: string | null;
  started_at: string;
  last_activity_at: string;
  steps_completed: number;
  steps_total: number;
  sla_deadline: string | null;
  time_remaining: number | null; // minutes
  health_score: number; // 0-100
  tenant_id: string;
  tenant_instance_id: string;
  owner: string | null;
  metadata: Record<string, any>;
  execution_history?: StepExecution[];
}

interface StepExecution {
  step_name: string;
  status: string;
  started_at: string;
  completed_at: string | null;
  duration: number | null; // seconds
  error_msg: string | null;
  metadata: Record<string, any>;
}

interface MonitoringStats {
  total_active: number;
  running_count: number;
  completed_count: number;
  failed_count: number;
  workflow_types: number;
}

export const ProcessMonitorDashboard: React.FC<ProcessMonitorDashboardProps> = ({
  tenant,
  datasource,
}) => {
  const [instances, setInstances] = useState<ProcessInstance[]>([]);
  const [selectedInstance, setSelectedInstance] = useState<ProcessInstance | null>(null);
  const [stats, setStats] = useState<MonitoringStats>({
    total_active: 0,
    running_count: 0,
    completed_count: 0,
    failed_count: 0,
    workflow_types: 0,
  });
  const [filters, setFilters] = useState({
    workflow_type: '',
    status: '',
  });
  const [showInterventionModal, setShowInterventionModal] = useState(false);
  const [interventionAction, setInterventionAction] = useState<'skip_step' | 'reassign' | 'cancel' | 'retry'>('skip_step');
  const [interventionReason, setInterventionReason] = useState('');

  // WebSocket connection for real-time updates
  const { isConnected, lastEvent, updateFilters } = useProcessMonitorWebSocket({
    tenantId: tenant.id,
    datasourceId: datasource.id,
    filters,
    onEvent: handleProcessEvent,
    autoReconnect: true,
  });

  function handleProcessEvent(event: ProcessEvent) {
    console.log('Received process event:', event);

    // Update instances based on event type
    if (event.type === 'workflow_started') {
      fetchActiveInstances(); // Refresh to get new workflow
    } else if (event.type === 'step_completed' || event.type === 'step_failed') {
      // Update specific instance
      setInstances((prev) =>
        prev.map((inst) =>
          inst.workflow_id === event.workflow_id
            ? {
                ...inst,
                current_step: event.step_name || inst.current_step,
                last_activity_at: event.timestamp,
                status: event.status,
              }
            : inst
        )
      );
    } else if (event.type === 'workflow_completed') {
      // Mark workflow as completed
      setInstances((prev) =>
        prev.map((inst) =>
          inst.workflow_id === event.workflow_id
            ? { ...inst, status: 'completed', last_activity_at: event.timestamp }
            : inst
        )
      );
    }
  }

  async function fetchActiveInstances() {
    try {
      const params = new URLSearchParams({
        tenant_id: tenant.id,
        tenant_instance_id: datasource.id,
      });

      if (filters.workflow_type) params.append('workflow_type', filters.workflow_type);
      if (filters.status) params.append('status', filters.status);

      const response = await fetch(`/api/process-monitor/active-instances?${params}`);
      if (!response.ok) throw new Error('Failed to fetch active instances');

      const data = await response.json();
      setInstances(data || []);
    } catch (error) {
      console.error('Error fetching active instances:', error);
    }
  }

  async function fetchStats() {
    try {
      const params = new URLSearchParams({
        tenant_id: tenant.id,
        tenant_instance_id: datasource.id,
      });

      const response = await fetch(`/api/process-monitor/stats?${params}`);
      if (!response.ok) throw new Error('Failed to fetch stats');

      const data = await response.json();
      setStats(data.stats || {});
    } catch (error) {
      console.error('Error fetching stats:', error);
    }
  }

  async function fetchInstanceDetails(workflowId: string) {
    try {
      const params = new URLSearchParams({
        tenant_id: tenant.id,
        tenant_instance_id: datasource.id,
      });

      const response = await fetch(`/api/process-monitor/instance/${workflowId}?${params}`);
      if (!response.ok) throw new Error('Failed to fetch instance details');

      const data = await response.json();
      setSelectedInstance(data);
    } catch (error) {
      console.error('Error fetching instance details:', error);
    }
  }

  async function handleIntervention() {
    if (!selectedInstance) return;

    try {
      const params = new URLSearchParams({
        tenant_id: tenant.id,
        tenant_instance_id: datasource.id,
      });

      const response = await fetch(`/api/process-monitor/intervene?${params}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          action: interventionAction,
          workflow_id: selectedInstance.workflow_id,
          step_name: selectedInstance.current_step,
          reason: interventionReason,
        }),
      });

      if (!response.ok) throw new Error('Intervention failed');

      const result = await response.json();
      console.log('Intervention result:', result);

      // Refresh data
      fetchActiveInstances();
      setShowInterventionModal(false);
      setInterventionReason('');
    } catch (error) {
      console.error('Error executing intervention:', error);
      alert('Failed to execute intervention: ' + error);
    }
  }

  useEffect(() => {
    fetchActiveInstances();
    fetchStats();

    const interval = setInterval(() => {
      fetchActiveInstances();
      fetchStats();
    }, 30000); // Refresh every 30 seconds

    return () => clearInterval(interval);
  }, [tenant.id, datasource.id, filters]);

  useEffect(() => {
    if (filters.workflow_type || filters.status) {
      updateFilters(filters);
    }
  }, [filters, updateFilters]);

  const workflowTypes = useMemo(() => {
    const types = new Set(instances.map((i) => i.workflow_type));
    return Array.from(types);
  }, [instances]);

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'running':
        return 'text-blue-600 bg-blue-100';
      case 'completed':
        return 'text-green-600 bg-green-100';
      case 'failed':
        return 'text-red-600 bg-red-100';
      default:
        return 'text-gray-600 bg-gray-100';
    }
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'running':
        return <Activity className="animate-pulse" size={16} />;
      case 'completed':
        return <CheckCircle2 size={16} />;
      case 'failed':
        return <AlertCircle size={16} />;
      default:
        return <Clock size={16} />;
    }
  };

  const formatDuration = (seconds: number | null) => {
    if (!seconds) return 'N/A';
    if (seconds < 60) return `${Math.round(seconds)}s`;
    if (seconds < 3600) return `${Math.round(seconds / 60)}m`;
    return `${Math.round(seconds / 3600)}h`;
  };

  const formatTimeRemaining = (minutes: number | null) => {
    if (!minutes) return 'No SLA';
    if (minutes < 0) return <span className="text-red-600 font-semibold">SLA Violated</span>;
    if (minutes < 60) return `${Math.round(minutes)}m remaining`;
    return `${Math.round(minutes / 60)}h remaining`;
  };

  return (
    <div className="h-full flex flex-col bg-gray-50">
      {/* Header */}
      <div className="bg-white border-b px-6 py-4">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold text-gray-900">Live Process Monitor</h1>
            <p className="text-sm text-gray-500 mt-1">
              Real-time visibility into running workflow instances
            </p>
          </div>
          <div className="flex items-center gap-3">
            <div className={`flex items-center gap-2 px-3 py-1.5 rounded-lg ${isConnected ? 'bg-green-100 text-green-700' : 'bg-red-100 text-red-700'}`}>
              {isConnected ? <Wifi size={16} /> : <WifiOff size={16} />}
              <span className="text-sm font-medium">
                {isConnected ? 'Live' : 'Disconnected'}
              </span>
            </div>
          </div>
        </div>

        {/* Stats Row */}
        <div className="grid grid-cols-5 gap-4 mt-4">
          <div className="bg-gradient-to-br from-blue-50 to-blue-100 rounded-lg px-4 py-3">
            <div className="text-sm text-blue-600 font-medium">Total Active</div>
            <div className="text-2xl font-bold text-blue-700 mt-1">{stats.total_active}</div>
          </div>
          <div className="bg-gradient-to-br from-purple-50 to-purple-100 rounded-lg px-4 py-3">
            <div className="text-sm text-purple-600 font-medium">Running</div>
            <div className="text-2xl font-bold text-purple-700 mt-1 flex items-center gap-2">
              {stats.running_count}
              <Activity className="animate-pulse text-purple-500" size={20} />
            </div>
          </div>
          <div className="bg-gradient-to-br from-green-50 to-green-100 rounded-lg px-4 py-3">
            <div className="text-sm text-green-600 font-medium">Completed</div>
            <div className="text-2xl font-bold text-green-700 mt-1">{stats.completed_count}</div>
          </div>
          <div className="bg-gradient-to-br from-red-50 to-red-100 rounded-lg px-4 py-3">
            <div className="text-sm text-red-600 font-medium">Failed</div>
            <div className="text-2xl font-bold text-red-700 mt-1">{stats.failed_count}</div>
          </div>
          <div className="bg-gradient-to-br from-gray-50 to-gray-100 rounded-lg px-4 py-3">
            <div className="text-sm text-gray-600 font-medium">Workflow Types</div>
            <div className="text-2xl font-bold text-gray-700 mt-1">{stats.workflow_types}</div>
          </div>
        </div>

        {/* Filters */}
        <div className="flex items-center gap-3 mt-4">
          <Filter size={18} className="text-gray-500" />
          <select
            value={filters.workflow_type}
            onChange={(e) => setFilters({ ...filters, workflow_type: e.target.value })}
            className="px-3 py-1.5 border rounded-lg text-sm"
          >
            <option value="">All Workflow Types</option>
            {workflowTypes.map((type) => (
              <option key={type} value={type}>
                {type}
              </option>
            ))}
          </select>
          <select
            value={filters.status}
            onChange={(e) => setFilters({ ...filters, status: e.target.value })}
            className="px-3 py-1.5 border rounded-lg text-sm"
          >
            <option value="">All Statuses</option>
            <option value="running">Running</option>
            <option value="completed">Completed</option>
            <option value="failed">Failed</option>
          </select>
          {(filters.workflow_type || filters.status) && (
            <button
              onClick={() => setFilters({ workflow_type: '', status: '' })}
              className="text-sm text-gray-600 hover:text-gray-800 underline"
            >
              Clear Filters
            </button>
          )}
        </div>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-hidden flex">
        {/* Instance List */}
        <div className="w-1/2 border-r overflow-y-auto p-4">
          {instances.length === 0 ? (
            <div className="text-center py-12 text-gray-500">
              <Activity size={48} className="mx-auto mb-4 text-gray-400" />
              <p className="text-lg font-medium">No active process instances</p>
              <p className="text-sm mt-2">Instances will appear here when workflows are running</p>
            </div>
          ) : (
            <div className="space-y-3">
              {instances.map((instance) => (
                <div
                  key={instance.workflow_id}
                  onClick={() => fetchInstanceDetails(instance.workflow_id)}
                  className={`bg-white rounded-lg border-2 p-4 cursor-pointer transition-all hover:shadow-md ${
                    selectedInstance?.workflow_id === instance.workflow_id
                      ? 'border-blue-500 shadow-lg'
                      : 'border-gray-200'
                  }`}
                >
                  <div className="flex items-start justify-between">
                    <div className="flex-1">
                      <div className="flex items-center gap-2 mb-2">
                        <span className={`px-2 py-1 rounded-full text-xs font-medium flex items-center gap-1 ${getStatusColor(instance.status)}`}>
                          {getStatusIcon(instance.status)}
                          {instance.status}
                        </span>
                        <span className="text-xs text-gray-500">{instance.workflow_type}</span>
                      </div>
                      <div className="text-sm font-mono text-gray-600 mb-1">
                        {instance.workflow_id.substring(0, 12)}...
                      </div>
                      {instance.current_step && (
                        <div className="text-sm text-gray-700">
                          <span className="font-medium">Current:</span> {instance.current_step}
                        </div>
                      )}
                      <div className="flex items-center gap-4 mt-2 text-xs text-gray-500">
                        <span>{instance.steps_completed} / {instance.steps_total} steps</span>
                        <span>Health: {instance.health_score}%</span>
                      </div>
                    </div>
                    <div className="text-right">
                      <div className="text-xs text-gray-500 mb-1">
                        {formatTimeRemaining(instance.time_remaining)}
                      </div>
                      <div className="text-xs text-gray-400">
                        {new Date(instance.last_activity_at).toLocaleTimeString()}
                      </div>
                    </div>
                  </div>
                  <div className="mt-2 bg-gray-100 rounded-full h-2 overflow-hidden">
                    <div
                      className={`h-full transition-all ${
                        instance.status === 'failed' ? 'bg-red-500' :
                        instance.status === 'completed' ? 'bg-green-500' :
                        'bg-blue-500'
                      }`}
                      style={{ width: `${(instance.steps_completed / instance.steps_total) * 100}%` }}
                    />
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Instance Details */}
        <div className="w-1/2 overflow-y-auto p-4">
          {!selectedInstance ? (
            <div className="text-center py-12 text-gray-500">
              <Play size={48} className="mx-auto mb-4 text-gray-400" />
              <p className="text-lg font-medium">Select an instance</p>
              <p className="text-sm mt-2">Click an instance to view details and execution history</p>
            </div>
          ) : (
            <div>
              <div className="bg-white rounded-lg border p-4 mb-4">
                <div className="flex items-start justify-between mb-4">
                  <div>
                    <h3 className="text-lg font-bold text-gray-900">{selectedInstance.workflow_type}</h3>
                    <p className="text-sm font-mono text-gray-500 mt-1">{selectedInstance.workflow_id}</p>
                  </div>
                  <button
                    onClick={() => setSelectedInstance(null)}
                    className="text-gray-400 hover:text-gray-600"
                  >
                    <X size={20} />
                  </button>
                </div>

                <div className="grid grid-cols-2 gap-4 mb-4">
                  <div>
                    <div className="text-xs text-gray-500">Status</div>
                    <div className={`mt-1 px-2 py-1 rounded-full text-sm font-medium inline-flex items-center gap-1 ${getStatusColor(selectedInstance.status)}`}>
                      {getStatusIcon(selectedInstance.status)}
                      {selectedInstance.status}
                    </div>
                  </div>
                  <div>
                    <div className="text-xs text-gray-500">Health Score</div>
                    <div className="text-2xl font-bold text-gray-900 mt-1">{selectedInstance.health_score}%</div>
                  </div>
                  <div>
                    <div className="text-xs text-gray-500">Steps Progress</div>
                    <div className="text-sm font-medium text-gray-900 mt-1">
                      {selectedInstance.steps_completed} / {selectedInstance.steps_total}
                    </div>
                  </div>
                  <div>
                    <div className="text-xs text-gray-500">Started</div>
                    <div className="text-sm font-medium text-gray-900 mt-1">
                      {new Date(selectedInstance.started_at).toLocaleString()}
                    </div>
                  </div>
                </div>

                {selectedInstance.status === 'running' && (
                  <div className="border-t pt-4">
                    <h4 className="text-sm font-semibold text-gray-700 mb-3">Intervention Actions</h4>
                    <div className="flex gap-2">
                      <button
                        onClick={() => {
                          setInterventionAction('skip_step');
                          setShowInterventionModal(true);
                        }}
                        className="flex items-center gap-2 px-3 py-2 bg-yellow-100 text-yellow-700 rounded-lg hover:bg-yellow-200 text-sm font-medium"
                      >
                        <SkipForward size={16} />
                        Skip Step
                      </button>
                      <button
                        onClick={() => {
                          setInterventionAction('reassign');
                          setShowInterventionModal(true);
                        }}
                        className="flex items-center gap-2 px-3 py-2 bg-blue-100 text-blue-700 rounded-lg hover:bg-blue-200 text-sm font-medium"
                      >
                        <UserPlus size={16} />
                        Reassign
                      </button>
                      <button
                        onClick={() => {
                          setInterventionAction('retry');
                          setShowInterventionModal(true);
                        }}
                        className="flex items-center gap-2 px-3 py-2 bg-green-100 text-green-700 rounded-lg hover:bg-green-200 text-sm font-medium"
                      >
                        <RefreshCw size={16} />
                        Retry
                      </button>
                      <button
                        onClick={() => {
                          setInterventionAction('cancel');
                          setShowInterventionModal(true);
                        }}
                        className="flex items-center gap-2 px-3 py-2 bg-red-100 text-red-700 rounded-lg hover:bg-red-200 text-sm font-medium"
                      >
                        <X size={16} />
                        Cancel
                      </button>
                    </div>
                  </div>
                )}
              </div>

              {/* Execution History */}
              {selectedInstance.execution_history && selectedInstance.execution_history.length > 0 && (
                <div className="bg-white rounded-lg border p-4">
                  <h4 className="text-sm font-semibold text-gray-700 mb-3">Execution History</h4>
                  <div className="space-y-2">
                    {selectedInstance.execution_history.map((step, idx) => (
                      <div key={idx} className="flex items-center gap-3 p-3 bg-gray-50 rounded-lg">
                        <div className={`p-2 rounded-full ${getStatusColor(step.status)}`}>
                          {getStatusIcon(step.status)}
                        </div>
                        <div className="flex-1">
                          <div className="text-sm font-medium text-gray-900">{step.step_name}</div>
                          <div className="text-xs text-gray-500 mt-1">
                            {step.completed_at ? (
                              <>Duration: {formatDuration(step.duration)}</>
                            ) : (
                              <>Started: {new Date(step.started_at).toLocaleTimeString()}</>
                            )}
                          </div>
                          {step.error_msg && (
                            <div className="text-xs text-red-600 mt-1">{step.error_msg}</div>
                          )}
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>
          )}
        </div>
      </div>

      {/* Intervention Modal */}
      {showInterventionModal && selectedInstance && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg p-6 max-w-md w-full">
            <h3 className="text-lg font-bold text-gray-900 mb-4">
              Confirm Intervention: {interventionAction.replace('_', ' ').toUpperCase()}
            </h3>
            <p className="text-sm text-gray-600 mb-4">
              This will {interventionAction.replace('_', ' ')} for workflow: {selectedInstance.workflow_id}
            </p>
            <textarea
              value={interventionReason}
              onChange={(e) => setInterventionReason(e.target.value)}
              placeholder="Reason for intervention (required)"
              className="w-full px-3 py-2 border rounded-lg text-sm mb-4"
              rows={3}
              required
            />
            <div className="flex gap-3">
              <button
                onClick={handleIntervention}
                disabled={!interventionReason.trim()}
                className="flex-1 px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 disabled:opacity-50 disabled:cursor-not-allowed font-medium"
              >
                Execute
              </button>
              <button
                onClick={() => {
                  setShowInterventionModal(false);
                  setInterventionReason('');
                }}
                className="flex-1 px-4 py-2 bg-gray-200 text-gray-700 rounded-lg hover:bg-gray-300 font-medium"
              >
                Cancel
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
