import React, { useState, useEffect, useCallback } from 'react';
import { AlertCircle, CheckCircle, XCircle, Clock, Play, Pause, StopCircle, RotateCcw, Terminal } from 'lucide-react';
import './TemporalAdminDashboard.css';
import { devError } from '../utils/devLogger';
import ActionButton from '../components/ui/ActionButton';

// Types
interface WorkflowExecution {
  id: string;
  runId: string;
  startTime: string;
  endTime?: string;
  status: string;
  workflowType: string;
  businessUnit?: string;
  priority?: number;
  slaDeadline?: string;
}

interface SearchAttribute {
  name: string;
  type: string;
  description: string;
}

interface AdminAction {
  action: string;
  status: 'success' | 'failed' | 'pending';
  message: string;
  timestamp: string;
}

// Temporal Admin Dashboard Component
import { useTenant } from '../contexts/TenantContext';
import { getSelectedRegion } from '../lib/region';

// ... (inside component)
export const TemporalAdminDashboard: React.FC = () => {
  const { tenant, datasource } = useTenant();
  const [workflows, setWorkflows] = useState<WorkflowExecution[]>([]);
  // some hooks may be unused in trimmed/demo flows; reference to avoid unused-import lint noise
  void useCallback;
  const [filteredWorkflows, setFilteredWorkflows] = useState<WorkflowExecution[]>([]);
  const [searchAttributes, setSearchAttributes] = useState<SearchAttribute[]>([]);
  const [selectedWorkflow, setSelectedWorkflow] = useState<WorkflowExecution | null>(null);
  const [actionHistory, setActionHistory] = useState<AdminAction[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  // Filter state
  const [filters, setFilters] = useState({
    status: '',
    businessUnit: '',
    priority: '',
    searchText: '',
  });

  // Signal/Action dialog state
  const [showActionDialog, setShowActionDialog] = useState(false);
  const [selectedAction, setSelectedAction] = useState('signal');
  const [actionInput, setActionInput] = useState({
    signalName: '',
    reason: '',
    input: '{}',
  });

  // Saved views
  const [savedViews, _setSavedViews] = useState([
    { name: 'Failed Last 24h', query: "status = 'failed' AND start_time > '-24h'" },
    { name: 'Pending > 2h', query: "status = 'pending' AND elapsed_time > 7200" },
    { name: 'High Priority', query: 'Priority > 2' },
  ]);

  // Load Search Attributes on mount
  useEffect(() => {
    loadSearchAttributes();
    // In production, load workflows from your backend
    loadMockWorkflows();
  }, []);

  // Apply filters whenever they change
  useEffect(() => {
    applyFilters();
  }, [filters, workflows]);

  const loadSearchAttributes = async () => {
    try {
      const response = await fetch('/api/temporal/search-attributes');
      if (response.ok) {
        const data = await response.json();
        const attrList = Object.values(data.search_attributes || {}).map((attr: any) => ({
          name: attr.name,
          type: attr.type,
          description: attr.desc,
        }));
        setSearchAttributes(attrList);
      }
    } catch (err) {
      devError('Error loading search attributes:', err);
    }
  };

  const loadMockWorkflows = () => {
    // Mock data for demo; replace with actual API call
    setWorkflows([
      {
        id: 'order-123',
        runId: 'run-001',
        startTime: new Date(Date.now() - 3600000).toISOString(),
        status: 'running',
        workflowType: 'OrderProcessing',
        businessUnit: 'Retail',
        priority: 1,
      },
      {
        id: 'order-124',
        runId: 'run-002',
        startTime: new Date(Date.now() - 7200000).toISOString(),
        endTime: new Date(Date.now() - 3600000).toISOString(),
        status: 'completed',
        workflowType: 'OrderProcessing',
        businessUnit: 'Wholesale',
        priority: 2,
      },
      {
        id: 'order-125',
        runId: 'run-003',
        startTime: new Date(Date.now() - 86400000).toISOString(),
        status: 'failed',
        workflowType: 'OrderProcessing',
        businessUnit: 'Retail',
        priority: 3,
      },
    ]);
  };

  const applyFilters = () => {
    let filtered = workflows;

    if (filters.status) {
      filtered = filtered.filter(w => w.status === filters.status);
    }

    if (filters.businessUnit) {
      filtered = filtered.filter(w => w.businessUnit === filters.businessUnit);
    }

    if (filters.priority) {
      filtered = filtered.filter(w => w.priority?.toString() === filters.priority);
    }

    if (filters.searchText) {
      const text = filters.searchText.toLowerCase();
      filtered = filtered.filter(w => w.id.toLowerCase().includes(text) || w.workflowType.toLowerCase().includes(text));
    }

    setFilteredWorkflows(filtered);
  };

  const performAction = async (workflowId: string, runId: string, action: string) => {
    setLoading(true);
    setError('');

    try {
      const endpoint = `/api/temporal/workflows/${workflowId}/${action}`;
      const payload = {
        workflow_id: workflowId,
        run_id: runId,
        ...(action === 'signal' && { signal_name: actionInput.signalName, input: JSON.parse(actionInput.input) }),
        ...(action === 'update' && { update_name: actionInput.signalName, input: JSON.parse(actionInput.input) }),
        reason: actionInput.reason,
      };

      const response = await fetch(endpoint, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Tenant-ID': tenant?.id || '',
          'X-Tenant-Datasource-ID': datasource?.id || '',
          'X-Tenant-Region': getSelectedRegion(),
        },
        body: JSON.stringify(payload),
      });

      if (response.ok) {
        const result = await response.json();
        addActionToHistory({
          action,
          status: 'success',
          message: result.message,
          timestamp: new Date().toISOString(),
        });
        setShowActionDialog(false);
        setActionInput({ signalName: '', reason: '', input: '{}' });
      } else {
        const err = await response.json();
        throw new Error(err.error || 'Action failed');
      }
    } catch (err) {
      setError((err as Error).message);
      addActionToHistory({
        action,
        status: 'failed',
        message: (err as Error).message,
        timestamp: new Date().toISOString(),
      });
    } finally {
      setLoading(false);
    }
  };

  const addActionToHistory = (action: AdminAction) => {
    setActionHistory(prev => [action, ...prev].slice(0, 10));
  };

  const applyView = (query: string) => {
    setFilters(prev => ({
      ...prev,
      searchText: query,
    }));
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'completed':
        return <CheckCircle className="status-icon completed" />;
      case 'failed':
        return <XCircle className="status-icon failed" />;
      case 'running':
        return <Clock className="status-icon running" />;
      default:
        return <AlertCircle className="status-icon" />;
    }
  };

  return (
    <div className="temporal-admin-dashboard">
      <header className="dashboard-header">
        <h1>Temporal Workflow Admin Dashboard</h1>
        <p>Governance, monitoring, and operational controls</p>
      </header>

      <div className="dashboard-container">
        {/* Left Sidebar: Saved Views & Search Attributes */}
        <aside className="sidebar">
          <div className="sidebar-section">
            <h3>Saved Views</h3>
            {savedViews.map(view => (
              <button
                key={view.name}
                className="view-button"
                onClick={() => applyView(view.query)}
              >
                {view.name}
              </button>
            ))}
          </div>

          <div className="sidebar-section">
            <h3>Search Attributes</h3>
            <div className="attributes-list">
              {searchAttributes.map(attr => (
                <div key={attr.name} className="attribute-item">
                  <div className="attr-name">{attr.name}</div>
                  <div className="attr-type">{attr.type}</div>
                </div>
              ))}
            </div>
          </div>

          <div className="sidebar-section">
            <h3>CLI Setup</h3>
            <a href="/api/temporal/setup-cli-script" target="_blank" rel="noreferrer" className="setup-link">
              <Terminal size={16} />
              Download Setup Script
            </a>
          </div>
        </aside>

        {/* Main Content */}
        <main className="main-content">
          {/* Filters */}
          <div className="filters-section">
            <input
              type="text"
              placeholder="Search workflows..."
              value={filters.searchText}
              onChange={e => setFilters(prev => ({ ...prev, searchText: e.target.value }))}
              className="filter-input"
            />

            <select
              aria-label="Filter by status"
              value={filters.status}
              onChange={e => setFilters(prev => ({ ...prev, status: e.target.value }))}
              className="filter-select"
            >
              <option value="">All Status</option>
              <option value="running">Running</option>
              <option value="completed">Completed</option>
              <option value="failed">Failed</option>
            </select>

            <select
              aria-label="Filter by business unit"
              value={filters.businessUnit}
              onChange={e => setFilters(prev => ({ ...prev, businessUnit: e.target.value }))}
              className="filter-select"
            >
              <option value="">All Business Units</option>
              <option value="Retail">Retail</option>
              <option value="Wholesale">Wholesale</option>
            </select>

            <select
              aria-label="Filter by priority"
              value={filters.priority}
              onChange={e => setFilters(prev => ({ ...prev, priority: e.target.value }))}
              className="filter-select"
            >
              <option value="">All Priorities</option>
              <option value="1">High (1)</option>
              <option value="2">Medium (2)</option>
              <option value="3">Low (3)</option>
            </select>
          </div>

          {/* Error Message */}
          {error && (
            <div className="error-banner">
              <AlertCircle size={18} />
              <span>{error}</span>
            </div>
          )}

          {/* Workflow List */}
          <div className="workflows-section">
            <h2>Workflows ({filteredWorkflows.length})</h2>

            {filteredWorkflows.length === 0 ? (
              <div className="empty-state">No workflows match your filters.</div>
            ) : (
              <div className="workflows-table">
                <div className="table-header">
                  <div className="col-id">Workflow ID</div>
                  <div className="col-type">Type</div>
                  <div className="col-status">Status</div>
                  <div className="col-unit">Business Unit</div>
                  <div className="col-priority">Priority</div>
                  <div className="col-actions">Actions</div>
                </div>

                {filteredWorkflows.map(workflow => (
                  <div
                    key={workflow.id}
                    className={`table-row ${selectedWorkflow?.id === workflow.id ? 'selected' : ''}`}
                    onClick={() => setSelectedWorkflow(workflow)}
                  >
                    <div className="col-id">{workflow.id}</div>
                    <div className="col-type">{workflow.workflowType}</div>
                    <div className="col-status">
                      {getStatusIcon(workflow.status)}
                      <span>{workflow.status}</span>
                    </div>
                    <div className="col-unit">{workflow.businessUnit}</div>
                    <div className="col-priority">
                      <span className={`priority-${workflow.priority}`}>{workflow.priority}</span>
                    </div>
                    <div className="col-actions">
                      <button
                        className="action-btn signal"
                        onClick={e => {
                          e.stopPropagation();
                          setSelectedAction('signal');
                          setShowActionDialog(true);
                        }}
                        title="Send Signal"
                      >
                        <Play size={16} />
                      </button>
                      <button
                        className="action-btn cancel"
                        onClick={e => {
                          e.stopPropagation();
                          performAction(workflow.id, workflow.runId, 'cancel');
                        }}
                        title="Cancel"
                      >
                        <Pause size={16} />
                      </button>
                      <button
                        className="action-btn terminate"
                        onClick={e => {
                          e.stopPropagation();
                          performAction(workflow.id, workflow.runId, 'terminate');
                        }}
                        title="Terminate"
                      >
                        <StopCircle size={16} />
                      </button>
                      <button
                        className="action-btn reset"
                        onClick={e => {
                          e.stopPropagation();
                          setSelectedAction('reset');
                          setShowActionDialog(true);
                        }}
                        title="Reset"
                      >
                        <RotateCcw size={16} />
                      </button>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>

          {/* Action History */}
          <div className="action-history-section">
            <h3>Recent Admin Actions</h3>
            <div className="action-history-list">
              {actionHistory.length === 0 ? (
                <div className="empty-state">No admin actions yet.</div>
              ) : (
                actionHistory.map((action, idx) => (
                  <div key={idx} className={`history-item ${action.status}`}>
                    <div className="action-details">
                      <span className="action-type">{action.action}</span>
                      <span className="action-message">{action.message}</span>
                    </div>
                    <span className="action-time">
                      {new Date(action.timestamp).toLocaleTimeString()}
                    </span>
                  </div>
                ))
              )}
            </div>
          </div>
        </main>

        {/* Right Sidebar: Details */}
        {selectedWorkflow && (
          <aside className="details-sidebar">
            <h3>Workflow Details</h3>

            <div className="detail-section">
              <div className="detail-row">
                <span className="label">ID:</span>
                <span className="value">{selectedWorkflow.id}</span>
              </div>
              <div className="detail-row">
                <span className="label">Run ID:</span>
                <span className="value">{selectedWorkflow.runId}</span>
              </div>
              <div className="detail-row">
                <span className="label">Type:</span>
                <span className="value">{selectedWorkflow.workflowType}</span>
              </div>
              <div className="detail-row">
                <span className="label">Status:</span>
                <span className="value">{selectedWorkflow.status}</span>
              </div>
              <div className="detail-row">
                <span className="label">Started:</span>
                <span className="value">{new Date(selectedWorkflow.startTime).toLocaleString()}</span>
              </div>
              {selectedWorkflow.endTime && (
                <div className="detail-row">
                  <span className="label">Ended:</span>
                  <span className="value">{new Date(selectedWorkflow.endTime).toLocaleString()}</span>
                </div>
              )}
              <div className="detail-row">
                <span className="label">Business Unit:</span>
                <span className="value">{selectedWorkflow.businessUnit}</span>
              </div>
              <div className="detail-row">
                <span className="label">Priority:</span>
                <span className="value">{selectedWorkflow.priority}</span>
              </div>
            </div>

            <div className="action-buttons">
              <ActionButton variant="primary" onClick={() => { setSelectedAction('signal'); setShowActionDialog(true); }}>
                Send Signal
              </ActionButton>
              <ActionButton variant="warning" onClick={() => performAction(selectedWorkflow.id, selectedWorkflow.runId, 'cancel')}>
                Cancel
              </ActionButton>
              <ActionButton variant="danger" onClick={() => performAction(selectedWorkflow.id, selectedWorkflow.runId, 'terminate')}>
                Terminate
              </ActionButton>
            </div>
          </aside>
        )}
      </div>

      {/* Action Dialog */}
      {showActionDialog && (
        <div className="modal-overlay" onClick={() => setShowActionDialog(false)}>
          <div className="modal-content" onClick={e => e.stopPropagation()}>
            <h3>{selectedAction.toUpperCase()} Workflow</h3>

            {selectedAction === 'signal' && (
              <>
                <input
                  type="text"
                  placeholder="Signal name"
                  value={actionInput.signalName}
                  onChange={e => setActionInput(prev => ({ ...prev, signalName: e.target.value }))}
                  className="modal-input"
                />
                <textarea
                  placeholder="Input (JSON)"
                  value={actionInput.input}
                  onChange={e => setActionInput(prev => ({ ...prev, input: e.target.value }))}
                  className="modal-textarea"
                />
              </>
            )}

            <input
              type="text"
              placeholder="Reason"
              value={actionInput.reason}
              onChange={e => setActionInput(prev => ({ ...prev, reason: e.target.value }))}
              className="modal-input"
            />

            <div className="modal-actions">
              <button
                className="btn btn-primary"
                disabled={loading}
                onClick={() => {
                  if (selectedWorkflow) {
                    performAction(selectedWorkflow.id, selectedWorkflow.runId, selectedAction);
                  }
                }}
              >
                {loading ? 'Processing...' : 'Confirm'}
              </button>
              <button className="btn btn-secondary" onClick={() => setShowActionDialog(false)}>
                Cancel
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default TemporalAdminDashboard;
