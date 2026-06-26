import React, { useState, useEffect } from 'react';
import { AlertCircle, CheckCircle2, XCircle, Clock, Zap, Download, Trash2, RefreshCw } from 'lucide-react';
import { useTenant } from '../contexts/TenantContext';
import { useConfirm } from '../components/ConfirmProvider';
import { useNotification } from '../hooks/useNotification';

interface ApprovalWorkflow {
  workflowId: string;
  status: 'PENDING' | 'APPROVED' | 'REJECTED' | 'ESCALATED' | 'EXPIRED';
  type: string;
  entityId: string;
  amount?: number;
  riskLevel?: string;
  initiatedBy: string;
  initiatedAt: Date | string;
  approvalChain: string[];
  escalationPath: string[];
  totalDurationSeconds?: number;
  decisions: ApprovalDecision[];
}

interface ApprovalDecision {
  approverId: string;
  route: string;
  decision: 'APPROVED' | 'REJECTED' | 'NEEDS_INFO';
  reason?: string;
  decidedAt: Date | string;
}

interface SeedingResult {
  validationRules: number;
  approvalRules: number;
  approverAssignments: number;
}

export const ApprovalWorkflowDashboard: React.FC = () => {
  const [activeTab, setActiveTab] = useState<'workflows' | 'seeding' | 'config'>('workflows');
  const [workflows, setWorkflows] = useState<ApprovalWorkflow[]>([]);
  const [loading, setLoading] = useState(false);
  const [seeding, setSeeding] = useState(false);
  const [seedResult, setSeedResult] = useState<SeedingResult | null>(null);
  const [toast, setToast] = useState<{ type: 'success' | 'error'; message: string } | null>(null);
  const [selectedWorkflow, setSelectedWorkflow] = useState<ApprovalWorkflow | null>(null);

  // Get tenant and datasource from TenantContext (preferred) or URL/localStorage fallback
  const { tenant, datasource } = useTenant();

  const [tenantId, setTenantId] = useState('');
  const [datasourceId, setDatasourceId] = useState('');

  useEffect(() => {
    // Prefer TenantContext values when available
    if (tenant && datasource) {
      setTenantId(tenant.id);
      setDatasourceId(datasource.id);
      return;
    }

    // Fallback: try reading from localStorage keys used by TenantContext
    try {
      const rawTenant = window.localStorage.getItem('selected_tenant');
      const rawDatasource = window.localStorage.getItem('selected_datasource');
      if (rawTenant) {
        try {
          const parsed = JSON.parse(rawTenant);
          if (parsed && parsed.id) setTenantId(parsed.id);
        } catch (e) {
          // ignore parse errors
        }
      }
      if (rawDatasource) {
        try {
          const parsedDs = JSON.parse(rawDatasource);
          if (parsedDs && parsedDs.id) setDatasourceId(parsedDs.id);
        } catch (e) {
          // ignore parse errors
        }
      }
    } catch (e) {
      // ignore localStorage issues
    }

    // Final fallback: URL params
    const params = new URLSearchParams(window.location.search);
    if (!tenantId) setTenantId(params.get('tenantId') || '');
    if (!datasourceId) setDatasourceId(params.get('datasourceId') || '');
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tenant, datasource]);

  useEffect(() => {
    if (activeTab === 'workflows') {
      loadWorkflows();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [activeTab, tenantId, datasourceId]);

  const showToast = (type: 'success' | 'error', message: string) => {
    setToast({ type, message });
    setTimeout(() => setToast(null), 4000);
  };

  const loadWorkflows = async () => {
    setLoading(true);
    try {
      const response = await fetch(`/api/approvals?tenantId=${tenantId}&datasourceId=${datasourceId}`);
      if (!response.ok) throw new Error('Failed to load workflows');
      const data = await response.json();
      setWorkflows(data.workflows || []);
    } catch (error) {
      showToast('error', 'Failed to load workflows');
    } finally {
      setLoading(false);
    }
  };

  const handleSeedAll = async () => {
    if (!tenantId || !datasourceId) {
      showToast('error', 'Tenant and datasource required');
      return;
    }

    setSeeding(true);
    try {
      const response = await fetch('/api/admin/seed', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ tenantId, datasourceId }),
      });

      if (!response.ok) throw new Error('Seeding failed');
      const data = await response.json();
      setSeedResult(data.data);
      showToast('success', 'Seeding completed successfully');
    } catch (error) {
      showToast('error', error instanceof Error ? error.message : 'Seeding failed');
    } finally {
      setSeeding(false);
    }
  };

  const handleSeedValidationRules = async () => {
    if (!tenantId || !datasourceId) {
      showToast('error', 'Tenant and datasource required');
      return;
    }

    setSeeding(true);
    try {
      const response = await fetch('/api/admin/seed/validation-rules', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ tenantId, datasourceId }),
      });

      if (!response.ok) throw new Error('Seeding failed');
      const data = await response.json();
      showToast('success', `Seeded ${data.count} validation rules`);
    } catch (error) {
      showToast('error', error instanceof Error ? error.message : 'Seeding failed');
    } finally {
      setSeeding(false);
    }
  };

  const handleSeedApprovalRules = async () => {
    if (!tenantId) {
      showToast('error', 'Tenant required');
      return;
    }

    setSeeding(true);
    try {
      const response = await fetch('/api/admin/seed/approval-rules', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ tenantId }),
      });

      if (!response.ok) throw new Error('Seeding failed');
      const data = await response.json();
      showToast('success', `Seeded ${data.count} approval rules`);
    } catch (error) {
      showToast('error', error instanceof Error ? error.message : 'Seeding failed');
    } finally {
      setSeeding(false);
    }
  };

  const handleClearSeed = async () => {
    const confirm = useConfirm();
    const notification = useNotification();
    if (!(await confirm({ title: 'Clear seed', description: 'Are you sure? This will delete all seeded rules.' }))) return;

    setSeeding(true);
    try {
      const response = await fetch('/api/admin/seed', {
        method: 'DELETE',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ tenantId, datasourceId }),
      });

      if (!response.ok) throw new Error('Clear failed');
      setSeedResult(null);
      showToast('success', 'All seeded rules cleared');
      notification.success('All seeded rules cleared');
    } catch (error) {
      showToast('error', error instanceof Error ? error.message : 'Clear failed');
    } finally {
      setSeeding(false);
    }
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'APPROVED':
        return 'bg-green-50 dark:bg-green-900/20';
      case 'REJECTED':
        return 'bg-red-50 dark:bg-red-900/20';
      case 'ESCALATED':
        return 'bg-orange-50 dark:bg-orange-900/20';
      case 'EXPIRED':
        return 'bg-gray-50 dark:bg-gray-900/20';
      default:
        return 'bg-blue-50 dark:bg-blue-900/20';
    }
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'APPROVED':
        return <CheckCircle2 className="w-5 h-5 text-green-600 dark:text-green-400" />;
      case 'REJECTED':
        return <XCircle className="w-5 h-5 text-red-600 dark:text-red-400" />;
      case 'ESCALATED':
        return <Zap className="w-5 h-5 text-orange-600 dark:text-orange-400" />;
      case 'EXPIRED':
        return <Clock className="w-5 h-5 text-gray-600 dark:text-gray-400" />;
      default:
        return <Clock className="w-5 h-5 text-blue-600 dark:text-blue-400" />;
    }
  };

  return (
    <div className="min-h-screen bg-gradient-to-br from-gray-50 to-gray-100 dark:from-gray-900 dark:to-gray-800">
      <div className="max-w-7xl mx-auto p-6">
        {/* Header */}
        <div className="mb-8">
          <h1 className="text-4xl font-bold text-gray-900 dark:text-white mb-2">Approval Workflow Manager</h1>
          <p className="text-gray-600 dark:text-gray-400">Manage dynamic approval workflows and seed business rules</p>
        </div>

        {/* Toast */}
        {toast && (
          <div className={`mb-6 p-4 rounded-lg flex items-center gap-3 ${
            toast.type === 'success'
              ? 'bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800'
              : 'bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800'
          }`}>
            {toast.type === 'success' ? (
              <CheckCircle2 className="w-5 h-5 text-green-600 dark:text-green-400" />
            ) : (
              <AlertCircle className="w-5 h-5 text-red-600 dark:text-red-400" />
            )}
            <span className={toast.type === 'success' ? 'text-green-800 dark:text-green-200' : 'text-red-800 dark:text-red-200'}>
              {toast.message}
            </span>
          </div>
        )}

        {/* Tenant/Datasource Input */}
        <div className="bg-white dark:bg-gray-800 rounded-lg shadow-sm p-6 mb-6">
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Tenant ID</label>
              <input
                type="text"
                value={tenantId}
                onChange={(e) => setTenantId(e.target.value)}
                className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500"
                placeholder="e.g., tenant-1"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Datasource ID</label>
              <input
                type="text"
                value={datasourceId}
                onChange={(e) => setDatasourceId(e.target.value)}
                className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500"
                placeholder="e.g., datasource-1"
              />
            </div>
          </div>
        </div>

        {/* Tab Navigation */}
        <div className="flex gap-4 mb-6 border-b border-gray-200 dark:border-gray-700">
          <button onClick={() => setActiveTab('workflows')} className={`px-4 py-3 font-medium border-b-2 transition-colors ${
            activeTab === 'workflows' ? 'border-blue-600 text-blue-600 dark:text-blue-400' : 'border-transparent text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-gray-300'
          }`}>Active Workflows</button>
          <button onClick={() => setActiveTab('seeding')} className={`px-4 py-3 font-medium border-b-2 transition-colors ${
            activeTab === 'seeding' ? 'border-blue-600 text-blue-600 dark:text-blue-400' : 'border-transparent text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-gray-300'
          }`}>Data Seeding</button>
          <button onClick={() => setActiveTab('config')} className={`px-4 py-3 font-medium border-b-2 transition-colors ${
            activeTab === 'config' ? 'border-blue-600 text-blue-600 dark:text-blue-400' : 'border-transparent text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-gray-300'
          }`}>Configuration</button>
        </div>

        {/* Workflows Tab */}
        {activeTab === 'workflows' && (
          <div className="space-y-4">
            <div className="flex justify-between items-center mb-4">
              <h2 className="text-2xl font-bold text-gray-900 dark:text-white">Active Approval Workflows ({workflows.length})</h2>
              <button onClick={loadWorkflows} disabled={loading} className="flex items-center gap-2 px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg disabled:opacity-50">
                <RefreshCw className="w-4 h-4" />
                Refresh
              </button>
            </div>

            {loading ? (
              <div className="text-center py-12">
                <div className="animate-spin inline-block w-8 h-8 border-4 border-gray-300 dark:border-gray-600 border-t-blue-600 rounded-full"></div>
                <p className="text-gray-600 dark:text-gray-400 mt-4">Loading workflows...</p>
              </div>
            ) : workflows.length === 0 ? (
              <div className="text-center py-12 bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
                <Clock className="w-12 h-12 text-gray-400 dark:text-gray-500 mx-auto mb-4" />
                <p className="text-gray-600 dark:text-gray-400">No active workflows</p>
              </div>
            ) : (
              <div className="space-y-4">
                {workflows.map((workflow) => (
                  <div key={workflow.workflowId} className={`p-6 rounded-lg border border-gray-200 dark:border-gray-700 cursor-pointer transition-all hover:shadow-md ${getStatusColor(workflow.status)}`} onClick={() => setSelectedWorkflow(selectedWorkflow?.workflowId === workflow.workflowId ? null : workflow)}>
                    <div className="flex items-start justify-between">
                      <div className="flex-1">
                        <div className="flex items-center gap-3 mb-2">
                          {getStatusIcon(workflow.status)}
                          <h3 className="text-lg font-semibold text-gray-900 dark:text-white">{workflow.type} Approval</h3>
                          <span className="px-3 py-1 bg-white dark:bg-gray-700 rounded-full text-sm font-medium text-gray-700 dark:text-gray-300">{workflow.status}</span>
                        </div>
                        <p className="text-sm text-gray-600 dark:text-gray-400 mb-3">Entity: {workflow.entityId} • Amount: {workflow.amount ? `${workflow.amount.toLocaleString()}` : 'N/A'} • Risk: {workflow.riskLevel || 'N/A'}</p>
                        <div className="flex flex-wrap gap-2">
                          {workflow.approvalChain.map((route, idx) => (
                            <div key={`${route}-${idx}`} className="flex items-center gap-1">
                              <span className="px-2 py-1 bg-white dark:bg-gray-700 rounded text-xs font-medium text-gray-700 dark:text-gray-300">{route}</span>
                              {idx < workflow.approvalChain.length - 1 && (<span className="text-gray-400">→</span>)}
                            </div>
                          ))}
                        </div>
                      </div>
                      <div className="text-right ml-4">
                        <p className="text-sm text-gray-600 dark:text-gray-400">{workflow.totalDurationSeconds} sec</p>
                        <p className="text-xs text-gray-500 dark:text-gray-500 mt-1">{new Date(workflow.initiatedAt).toLocaleString()}</p>
                      </div>
                    </div>

                    {/* Expanded Details */}
                    {selectedWorkflow?.workflowId === workflow.workflowId && (
                      <div className="mt-6 pt-6 border-t border-gray-300 dark:border-gray-600 space-y-4">
                        <div>
                          <h4 className="font-semibold text-gray-900 dark:text-white mb-3">Approval Decisions</h4>
                          <div className="space-y-2">
                            {workflow.decisions.map((decision, idx) => (
                              <div key={idx} className="p-3 bg-white dark:bg-gray-700 rounded text-sm">
                                <div className="flex items-center justify-between">
                                  <div>
                                    <p className="font-medium text-gray-900 dark:text-white">{decision.route}</p>
                                    <p className="text-gray-600 dark:text-gray-400">by {decision.approverId}</p>
                                  </div>
                                  <span className={`px-2 py-1 rounded text-xs font-medium ${
                                    decision.decision === 'APPROVED' ? 'bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-200' : decision.decision === 'REJECTED' ? 'bg-red-100 dark:bg-red-900/30 text-red-800 dark:text-red-200' : 'bg-yellow-100 dark:bg-yellow-900/30 text-yellow-800 dark:text-yellow-200'
                                  }`}>{decision.decision}</span>
                                </div>
                                {decision.reason && (<p className="text-gray-600 dark:text-gray-400 mt-2">{decision.reason}</p>)}
                                <p className="text-gray-500 dark:text-gray-500 text-xs mt-1">{new Date(decision.decidedAt).toLocaleString()}</p>
                              </div>
                            ))}
                          </div>
                        </div>
                      </div>
                    )}
                  </div>
                ))}
              </div>
            )}
          </div>
        )}

        {/* Seeding Tab */}
        {activeTab === 'seeding' && (
          <div className="space-y-6">
            <div className="bg-white dark:bg-gray-800 rounded-lg shadow-sm p-6">
              <h2 className="text-2xl font-bold text-gray-900 dark:text-white mb-4">Database Seeding</h2>
              <p className="text-gray-600 dark:text-gray-400 mb-6">Seed validation rules, approval rules, and approver assignments into your Hasura instance.</p>

              <div className="space-y-3">
                <button onClick={handleSeedAll} disabled={seeding || !tenantId || !datasourceId} className="w-full px-6 py-3 bg-green-600 hover:bg-green-700 text-white rounded-lg font-medium disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center gap-2">
                  <Download className="w-5 h-5" />
                  {seeding ? 'Seeding...' : 'Seed All Rules'}
                </button>

                <div className="grid grid-cols-2 gap-3">
                  <button onClick={handleSeedValidationRules} disabled={seeding || !tenantId || !datasourceId} className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg font-medium disabled:opacity-50">{seeding ? 'Seeding...' : 'Validation Rules'}</button>
                  <button onClick={handleSeedApprovalRules} disabled={seeding || !tenantId} className="px-4 py-2 bg-purple-600 hover:bg-purple-700 text-white rounded-lg font-medium disabled:opacity-50">{seeding ? 'Seeding...' : 'Approval Rules'}</button>
                </div>

                <button onClick={handleClearSeed} disabled={seeding || !tenantId || !datasourceId} className="w-full px-6 py-3 bg-red-600 hover:bg-red-700 text-white rounded-lg font-medium disabled:opacity-50 flex items-center justify-center gap-2">
                  <Trash2 className="w-5 h-5" />
                  Clear All Seeded Data
                </button>
              </div>

              {seedResult && (
                <div className="mt-6 p-4 bg-green-50 dark:bg-green-900/20 rounded-lg border border-green-200 dark:border-green-800">
                  <h3 className="font-semibold text-green-800 dark:text-green-200 mb-2">Last Seed Result</h3>
                  <ul className="text-sm text-green-700 dark:text-green-300 space-y-1">
                    <li>✓ Validation Rules: {seedResult.validationRules}</li>
                    <li>✓ Approval Rules: {seedResult.approvalRules}</li>
                    <li>✓ Approver Assignments: {seedResult.approverAssignments}</li>
                  </ul>
                </div>
              )}
            </div>
          </div>
        )}

        {/* Configuration Tab */}
        {activeTab === 'config' && (
          <div className="space-y-6">
            <div className="bg-white dark:bg-gray-800 rounded-lg shadow-sm p-6">
              <h2 className="text-2xl font-bold text-gray-900 dark:text-white mb-4">Environment Configuration</h2>
              
              <div className="space-y-4">
                <div className="p-4 bg-gray-50 dark:bg-gray-700 rounded-lg font-mono text-sm">
                  <p className="text-gray-600 dark:text-gray-400">Required environment variables:</p>
                  <pre className="text-gray-900 dark:text-gray-100 mt-2 whitespace-pre-wrap break-words">{`HASURA_GRAPHQL_ENDPOINT=http://localhost:8080/v1/graphql
HASURA_ADMIN_SECRET=dev-secret
TEMPORAL_SERVER_HOST=localhost:7233
RABBITMQ_URL=amqp://guest:guest@localhost:5672
DATABASE_URL=postgresql://user:pass@localhost:5432/wealth_db`}</pre>
                </div>

                <div className="p-4 bg-blue-50 dark:bg-blue-900/20 rounded-lg border border-blue-200 dark:border-blue-800">
                  <h3 className="font-semibold text-blue-900 dark:text-blue-100 mb-2">Quick Start</h3>
                  <ol className="text-sm text-blue-800 dark:text-blue-200 space-y-2 list-decimal list-inside">
                    <li>Ensure Hasura is running at the configured endpoint</li>
                    <li>Fill in Tenant ID and Datasource ID above</li>
                    <li>Click "Seed All Rules" to populate database</li>
                    <li>Rules will appear in Validation Rules Builder</li>
                    <li>Approval workflows automatically route based on rules</li>
                  </ol>
                </div>

                <div className="p-4 bg-amber-50 dark:bg-amber-900/20 rounded-lg border border-amber-200 dark:border-amber-800">
                  <h3 className="font-semibold text-amber-900 dark:text-amber-100 mb-2">API Endpoints</h3>
                  <ul className="text-sm text-amber-800 dark:text-amber-200 space-y-1 font-mono">
                    <li>POST /api/admin/seed - Seed all</li>
                    <li>POST /api/admin/seed/validation-rules - Validation only</li>
                    <li>POST /api/admin/seed/approval-rules - Approval only</li>
                    <li>DELETE /api/admin/seed - Clear all</li>
                    <li>GET /api/approvals - List workflows</li>
                    <li>POST /api/approvals/start - Start workflow</li>
                    <li>POST /api/approvals/:id/decide - Submit decision</li>
                  </ul>
                </div>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
};

export default ApprovalWorkflowDashboard;
