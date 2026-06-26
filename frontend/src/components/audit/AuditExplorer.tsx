import React, { useState, useEffect } from 'react';
import { AlertCircle, TrendingUp, Database, CheckCircle } from 'lucide-react';

interface AuditExplorerProps {
  tenantId: string;
  tenantName: string;
}

interface JobRun {
  run_id: string;
  job_id: string;
  tenant_id: string;
  start_ts: string;
  end_ts: string;
  status: string;
  error_message?: string;
  semantic_context?: any;
  compliance_context?: any;
  slo_context?: any;
  ai_narrative?: any;
}

interface ComplianceViolation {
  violation_id: string;
  tenant_id: string;
  violated_at: string;
  remediated_at?: string;
  violation_type: string;
  severity: string;
  pii_exposed: boolean;
  affected_records: number;
  narrative: string;
}

interface ChangeSet {
  changeset_id: string;
  type: string;
  actor: string;
  created_at: string;
  status: string;
  semantic_impact?: any;
  compliance_impact?: any;
  ai_summary?: any;
  ai_risk?: any;
}

type TabType = 'jobs' | 'violations' | 'changesets' | 'dashboards';

export const AuditExplorer: React.FC<AuditExplorerProps> = ({ tenantId, tenantName }) => {
  const [activeTab, setActiveTab] = useState<TabType>('jobs');
  const [jobRuns, setJobRuns] = useState<JobRun[]>([]);
  const [violations, setViolations] = useState<ComplianceViolation[]>([]);
  const [changeSets, setChangeSets] = useState<ChangeSet[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [selectedRecord, setSelectedRecord] = useState<any | null>(null);
  
  // Filters
  const [statusFilter, setStatusFilter] = useState<string>('');
  const [dateFilter, setDateFilter] = useState<string>('7d');

  useEffect(() => {
    loadData();
  }, [activeTab, tenantId, statusFilter, dateFilter]);

  const loadData = async () => {
    setLoading(true);
    setError(null);
    
    try {
      const headers = {
        'X-Tenant-ID': tenantId,
        'Content-Type': 'application/json',
      };

      switch (activeTab) {
        case 'jobs':
          const jobsResponse = await fetch(
            `/api/audit/job-runs?status=${statusFilter}&limit=100`,
            { headers }
          );
          const jobsData = await jobsResponse.json();
          setJobRuns(jobsData.data || []);
          break;

        case 'violations':
          const violationsResponse = await fetch(
            `/api/audit/violations?limit=100`,
            { headers }
          );
          const violationsData = await violationsResponse.json();
          setViolations(violationsData.data || []);
          break;

        case 'changesets':
          const changesetsResponse = await fetch(
            `/api/audit/changesets?limit=100`,
            { headers }
          );
          const changesetsData = await changesetsResponse.json();
          setChangeSets(changesetsData.data || []);
          break;
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load data');
    } finally {
      setLoading(false);
    }
  };

  const explainWithAI = async (recordType: string, recordId: string) => {
    try {
      const response = await fetch(`/api/audit/ai-narratives`, {
        method: 'POST',
        headers: {
          'X-Tenant-ID': tenantId,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ record_type: recordType, record_id: recordId }),
      });
      
      const data = await response.json();
      setSelectedRecord({ ...selectedRecord, aiNarrative: data });
    } catch (err) {
      console.error('Failed to generate AI narrative:', err);
    }
  };

  const getStatusColor = (status: string) => {
    switch (status.toUpperCase()) {
      case 'SUCCESS': return 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200';
      case 'FAILED': return 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200';
      case 'COMPLIANCE_BLOCK': return 'bg-orange-100 text-orange-800 dark:bg-orange-900 dark:text-orange-200';
      case 'RUNNING': return 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200';
      default: return 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-200';
    }
  };

  const getSeverityColor = (severity: string) => {
    switch (severity.toUpperCase()) {
      case 'CRITICAL': return 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200';
      case 'HIGH': return 'bg-orange-100 text-orange-800 dark:bg-orange-900 dark:text-orange-200';
      case 'MEDIUM': return 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200';
      case 'LOW': return 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200';
      default: return 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-200';
    }
  };

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-900">
      {/* Header */}
      <div className="bg-white dark:bg-gray-800 shadow">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-4">
          <div className="flex items-center justify-between">
            <div>
              <h1 className="text-2xl font-bold text-gray-900 dark:text-white">Audit Explorer</h1>
              <p className="text-sm text-gray-500 dark:text-gray-400">
                Tenant: {tenantName} ({tenantId})
              </p>
            </div>
            <div className="flex gap-2">
              <button
                onClick={loadData}
                className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700"
              >
                Refresh
              </button>
            </div>
          </div>

          {/* Tabs */}
          <div className="mt-4 border-b border-gray-200 dark:border-gray-700">
            <nav className="-mb-px flex space-x-8">
              {[
                { id: 'jobs', label: 'Job Runs', icon: Database },
                { id: 'violations', label: 'Compliance', icon: AlertCircle },
                { id: 'changesets', label: 'Governance', icon: TrendingUp },
                { id: 'dashboards', label: 'Dashboards', icon: CheckCircle },
              ].map(({ id, label, icon: Icon }) => (
                <button
                  key={id}
                  onClick={() => setActiveTab(id as TabType)}
                  className={`
                    flex items-center gap-2 py-4 px-1 border-b-2 font-medium text-sm
                    ${activeTab === id
                      ? 'border-blue-500 text-blue-600 dark:text-blue-400'
                      : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300 dark:text-gray-400 dark:hover:text-gray-300'
                    }
                  `}
                >
                  <Icon className="w-5 h-5" />
                  {label}
                </button>
              ))}
            </nav>
          </div>
        </div>
      </div>

      {/* Filters */}
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-4">
        <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-4 mb-4">
          <div className="flex gap-4">
            {activeTab === 'jobs' && (
              <>
                <label htmlFor="status-filter" className="sr-only">Filter by Status</label>
                <select
                  id="status-filter"
                  value={statusFilter}
                  onChange={(e) => setStatusFilter(e.target.value)}
                  className="px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
                  aria-label="Filter by Status"
                >
                  <option value="">All Statuses</option>
                  <option value="SUCCESS">Success</option>
                  <option value="FAILED">Failed</option>
                  <option value="COMPLIANCE_BLOCK">Compliance Block</option>
                </select>
              </>
            )}
            <label htmlFor="date-filter" className="sr-only">Filter by Date</label>
            <select
              id="date-filter"
              value={dateFilter}
              onChange={(e) => setDateFilter(e.target.value)}
              className="px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
              aria-label="Filter by Date Range"
            >
              <option value="1d">Last 24 Hours</option>
              <option value="7d">Last 7 Days</option>
              <option value="30d">Last 30 Days</option>
              <option value="90d">Last 90 Days</option>
            </select>
          </div>
        </div>

        {/* Content Area */}
        {loading ? (
          <div className="flex justify-center items-center h-64">
            <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
          </div>
        ) : error ? (
          <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-4">
            <p className="text-red-800 dark:text-red-200">{error}</p>
          </div>
        ) : (
          <>
            {activeTab === 'jobs' && (
              <JobRunsTable
                jobRuns={jobRuns}
                onExplain={(runId) => explainWithAI('JOB_RUN', runId)}
                onSelect={setSelectedRecord}
                getStatusColor={getStatusColor}
              />
            )}

            {activeTab === 'violations' && (
              <ViolationsTable
                violations={violations}
                onSelect={setSelectedRecord}
                getSeverityColor={getSeverityColor}
              />
            )}

            {activeTab === 'changesets' && (
              <ChangeSetsTable
                changeSets={changeSets}
                onExplain={(csId) => explainWithAI('CHANGESET', csId)}
                onSelect={setSelectedRecord}
                getStatusColor={getStatusColor}
              />
            )}

            {activeTab === 'dashboards' && (
              <DashboardsView tenantId={tenantId} />
            )}
          </>
        )}
      </div>

      {/* Detail Panel */}
      {selectedRecord && (
        <DetailPanel
          record={selectedRecord}
          onClose={() => setSelectedRecord(null)}
        />
      )}
    </div>
  );
};

// Job Runs Table Component
const JobRunsTable: React.FC<{
  jobRuns: JobRun[];
  onExplain: (runId: string) => void;
  onSelect: (record: any) => void;
  getStatusColor: (status: string) => string;
}> = ({ jobRuns, onExplain, onSelect, getStatusColor }) => {
  return (
    <div className="bg-white dark:bg-gray-800 shadow rounded-lg overflow-hidden">
      <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
        <thead className="bg-gray-50 dark:bg-gray-700">
          <tr>
            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
              Job ID
            </th>
            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
              Status
            </th>
            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
              Duration
            </th>
            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
              Start Time
            </th>
            <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
              Actions
            </th>
          </tr>
        </thead>
        <tbody className="bg-white dark:bg-gray-800 divide-y divide-gray-200 dark:divide-gray-700">
          {jobRuns.map((run) => (
            <tr key={run.run_id} className="hover:bg-gray-50 dark:hover:bg-gray-700 cursor-pointer">
              <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900 dark:text-white">
                {run.job_id}
              </td>
              <td className="px-6 py-4 whitespace-nowrap">
                <span className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${getStatusColor(run.status)}`}>
                  {run.status}
                </span>
              </td>
              <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
                {calculateDuration(run.start_ts, run.end_ts)}
              </td>
              <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
                {formatTimestamp(run.start_ts)}
              </td>
              <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                <button
                  onClick={() => onExplain(run.run_id)}
                  className="text-blue-600 hover:text-blue-900 dark:text-blue-400 dark:hover:text-blue-300 mr-4"
                >
                  Explain with AI
                </button>
                <button
                  onClick={() => onSelect(run)}
                  className="text-gray-600 hover:text-gray-900 dark:text-gray-400 dark:hover:text-gray-300"
                >
                  View Details
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
};

// Violations Table Component
const ViolationsTable: React.FC<{
  violations: ComplianceViolation[];
  onSelect: (record: any) => void;
  getSeverityColor: (severity: string) => string;
}> = ({ violations, onSelect, getSeverityColor }) => {
  return (
    <div className="bg-white dark:bg-gray-800 shadow rounded-lg overflow-hidden">
      <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
        <thead className="bg-gray-50 dark:bg-gray-700">
          <tr>
            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
              Type
            </th>
            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
              Severity
            </th>
            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
              PII Exposed
            </th>
            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
              Records Affected
            </th>
            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
              Status
            </th>
            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
              Violated At
            </th>
          </tr>
        </thead>
        <tbody className="bg-white dark:bg-gray-800 divide-y divide-gray-200 dark:divide-gray-700">
          {violations.map((violation) => (
            <tr
              key={violation.violation_id}
              onClick={() => onSelect(violation)}
              className="hover:bg-gray-50 dark:hover:bg-gray-700 cursor-pointer"
            >
              <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 dark:text-white">
                {violation.violation_type}
              </td>
              <td className="px-6 py-4 whitespace-nowrap">
                <span className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${getSeverityColor(violation.severity)}`}>
                  {violation.severity}
                </span>
              </td>
              <td className="px-6 py-4 whitespace-nowrap text-sm">
                {violation.pii_exposed ? (
                  <span className="text-red-600 dark:text-red-400 font-semibold">YES</span>
                ) : (
                  <span className="text-green-600 dark:text-green-400">No</span>
                )}
              </td>
              <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
                {violation.affected_records.toLocaleString()}
              </td>
              <td className="px-6 py-4 whitespace-nowrap text-sm">
                {violation.remediated_at ? (
                  <span className="text-green-600 dark:text-green-400">Remediated</span>
                ) : (
                  <span className="text-orange-600 dark:text-orange-400 font-semibold">Open</span>
                )}
              </td>
              <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
                {formatTimestamp(violation.violated_at)}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
};

// ChangeSets Table Component
const ChangeSetsTable: React.FC<{
  changeSets: ChangeSet[];
  onExplain: (csId: string) => void;
  onSelect: (record: any) => void;
  getStatusColor: (status: string) => string;
}> = ({ changeSets, onExplain, onSelect, getStatusColor }) => {
  return (
    <div className="bg-white dark:bg-gray-800 shadow rounded-lg overflow-hidden">
      <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
        <thead className="bg-gray-50 dark:bg-gray-700">
          <tr>
            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
              Type
            </th>
            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
              Actor
            </th>
            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
              Status
            </th>
            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
              Created
            </th>
            <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
              Actions
            </th>
          </tr>
        </thead>
        <tbody className="bg-white dark:bg-gray-800 divide-y divide-gray-200 dark:divide-gray-700">
          {changeSets.map((cs) => (
            <tr key={cs.changeset_id} className="hover:bg-gray-50 dark:hover:bg-gray-700 cursor-pointer">
              <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 dark:text-white">
                {cs.type}
              </td>
              <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
                {cs.actor}
              </td>
              <td className="px-6 py-4 whitespace-nowrap">
                <span className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${getStatusColor(cs.status)}`}>
                  {cs.status}
                </span>
              </td>
              <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
                {formatTimestamp(cs.created_at)}
              </td>
              <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                <button
                  onClick={() => onExplain(cs.changeset_id)}
                  className="text-blue-600 hover:text-blue-900 dark:text-blue-400 dark:hover:text-blue-300 mr-4"
                >
                  Explain with AI
                </button>
                <button
                  onClick={() => onSelect(cs)}
                  className="text-gray-600 hover:text-gray-900 dark:text-gray-400 dark:hover:text-gray-300"
                >
                  View Details
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
};

// Dashboards View Component
const DashboardsView: React.FC<{ tenantId: string }> = ({ tenantId }) => {
  const [sloData, setSloData] = useState<any[]>([]);
  const [complianceData, setComplianceData] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadDashboardData();
  }, [tenantId]);

  const loadDashboardData = async () => {
    try {
      const headers = { 'X-Tenant-ID': tenantId };
      
      const [sloResponse, complianceResponse] = await Promise.all([
        fetch('/api/audit/dashboard/slo', { headers }),
        fetch('/api/audit/dashboard/compliance', { headers }),
      ]);

      const slo = await sloResponse.json();
      const compliance = await complianceResponse.json();

      setSloData(slo.data || []);
      setComplianceData(compliance.data || []);
    } catch (err) {
      console.error('Failed to load dashboard data:', err);
    } finally {
      setLoading(false);
    }
  };

  if (loading) {
    return <div className="text-center py-8">Loading dashboards...</div>;
  }

  return (
    <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
      {/* SLO Dashboard */}
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
        <h3 className="text-lg font-semibold mb-4 text-gray-900 dark:text-white">SLO Performance</h3>
        <div className="space-y-4">
          {sloData.slice(0, 7).map((day, idx) => (
            <div key={idx} className="flex items-center justify-between">
              <span className="text-sm text-gray-600 dark:text-gray-400">
                {new Date(day.run_date).toLocaleDateString()}
              </span>
              <div className="flex items-center gap-4">
                <span className="text-sm text-gray-900 dark:text-white">
                  {((day.successful_runs / day.total_runs) * 100).toFixed(1)}% success
                </span>
                <span className="text-xs text-gray-500 dark:text-gray-400">
                  {day.total_runs} runs
                </span>
              </div>
            </div>
          ))}
        </div>
      </div>

      {/* Compliance Dashboard */}
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
        <h3 className="text-lg font-semibold mb-4 text-gray-900 dark:text-white">Compliance Status</h3>
        <div className="space-y-4">
          {complianceData.slice(0, 7).map((day, idx) => (
            <div key={idx} className="flex items-center justify-between">
              <span className="text-sm text-gray-600 dark:text-gray-400">
                {new Date(day.violation_date).toLocaleDateString()}
              </span>
              <div className="flex items-center gap-4">
                <span className={`text-sm font-semibold ${day.violation_count > 0 ? 'text-red-600 dark:text-red-400' : 'text-green-600 dark:text-green-400'}`}>
                  {day.violation_count} violations
                </span>
                {day.pii_exposure_count > 0 && (
                  <span className="text-xs text-red-600 dark:text-red-400 font-semibold">
                    {day.pii_exposure_count} PII
                  </span>
                )}
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
};

// Detail Panel Component
const DetailPanel: React.FC<{
  record: any;
  onClose: () => void;
}> = ({ record, onClose }) => {
  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow-xl max-w-4xl w-full max-h-[90vh] overflow-y-auto">
        <div className="sticky top-0 bg-white dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700 px-6 py-4 flex items-center justify-between">
          <h2 className="text-xl font-semibold text-gray-900 dark:text-white">Record Details</h2>
          <button
            onClick={onClose}
            className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
          >
            ✕
          </button>
        </div>
        
        <div className="p-6">
          <pre className="bg-gray-50 dark:bg-gray-900 p-4 rounded-lg overflow-x-auto text-sm">
            {JSON.stringify(record, null, 2)}
          </pre>
        </div>
      </div>
    </div>
  );
};

// Utility functions
const calculateDuration = (start: string, end: string): string => {
  const duration = new Date(end).getTime() - new Date(start).getTime();
  const seconds = Math.floor(duration / 1000);
  if (seconds < 60) return `${seconds}s`;
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m ${seconds % 60}s`;
  const hours = Math.floor(minutes / 60);
  return `${hours}h ${minutes % 60}m`;
};

const formatTimestamp = (ts: string): string => {
  return new Date(ts).toLocaleString();
};

export default AuditExplorer;
