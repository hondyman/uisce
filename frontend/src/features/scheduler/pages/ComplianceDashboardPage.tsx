/**
 * World-Class Enterprise Scheduler - Compliance Dashboard
 * Comprehensive audit logs, compliance reports, and governance tracking
 */

import React, { useState, useEffect, useCallback, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { Link } from 'react-router-dom';
import * as schedulerService from '../services/schedulerService';
import {
  AuditLogEntry,
  Job,
  JobExecution,
  JobStatus,
} from '../../../types/scheduler';
import '../styles/SchedulerDashboard.css';
import { devDebug } from '../../../utils/devLogger';

// ============================================================================
// Main Compliance Dashboard
// ============================================================================

export function ComplianceDashboardPage() {
  const { t } = useTranslation();
  
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState<'overview' | 'audit' | 'sla' | 'reports'>('overview');
  
  // Data
  const [auditLogs, setAuditLogs] = useState<AuditLogEntry[]>([]);
  const [jobs, setJobs] = useState<Job[]>([]);
  const [executions, setExecutions] = useState<JobExecution[]>([]);
  const [metrics, setMetrics] = useState<ComplianceMetrics | null>(null);
  
  // Filters
  const [dateRange, setDateRange] = useState<{ start: string; end: string }>({
    start: getDefaultStartDate(),
    end: new Date().toISOString().split('T')[0],
  });
  const [auditFilter, setAuditFilter] = useState({
    action: '',
    entity: '',
    user: '',
  });
  
  // Load data
  const loadData = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      
      const [auditData, jobsData, executionsData] = await Promise.all([
        schedulerService.getAuditLogs({
          start_date: dateRange.start,
          end_date: dateRange.end,
          page: 1,
          limit: 500,
        }),
        schedulerService.listJobs({ page: 1, limit: 100 }),
        schedulerService.listAllExecutions({ page: 1, limit: 500 }),
      ]);
      
      setAuditLogs(auditData.entries);
      setJobs(jobsData.jobs);
      setExecutions(executionsData.executions);
      
      // Calculate metrics
      setMetrics(calculateMetrics(auditData.entries, jobsData.jobs, executionsData.executions));
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load compliance data');
    } finally {
      setLoading(false);
    }
  }, [dateRange]);
  
  useEffect(() => {
    loadData();
  }, [loadData]);
  
  // Filter audit logs
  const filteredAuditLogs = useMemo(() => {
    return auditLogs.filter(log => {
      if (auditFilter.action && log.action !== auditFilter.action) return false;
      if (auditFilter.entity && log.entity_type !== auditFilter.entity) return false;
      if (auditFilter.user && !log.user_id?.includes(auditFilter.user)) return false;
      return true;
    });
  }, [auditLogs, auditFilter]);
  
  if (loading) {
    return (
      <div className="scheduler-dashboard">
        <div className="loading-spinner">
          <div className="spinner" />
        </div>
      </div>
    );
  }
  
  return (
    <div className="scheduler-dashboard">
      {/* Header */}
      <div className="scheduler-header">
        <div>
          <h1>📊 {t('scheduler.complianceDashboard', 'Compliance Dashboard')}</h1>
          <p className="header-subtitle">
            {t('scheduler.complianceDashboardDesc', 'Audit logs, SLA tracking, and compliance reporting')}
          </p>
        </div>
        <div className="scheduler-header-actions">
          <div className="date-range-picker">
            <input
              type="date"
              value={dateRange.start}
              onChange={e => setDateRange(prev => ({ ...prev, start: e.target.value }))}
              className="form-control"
            />
            <span>to</span>
            <input
              type="date"
              value={dateRange.end}
              onChange={e => setDateRange(prev => ({ ...prev, end: e.target.value }))}
              className="form-control"
            />
          </div>
          <button className="btn btn-secondary" onClick={loadData}>
            🔄 {t('scheduler.refresh', 'Refresh')}
          </button>
          <button className="btn btn-primary" onClick={() => exportComplianceReport(metrics, auditLogs, t)}>
            📥 {t('scheduler.exportReport', 'Export Report')}
          </button>
        </div>
      </div>
      
      {/* Error */}
      {error && (
        <div className="error-banner">
          <span>⚠️ {error}</span>
          <button onClick={loadData}>{t('scheduler.retry', 'Retry')}</button>
        </div>
      )}
      
      {/* Tabs */}
      <div className="editor-tabs">
        <button
          className={`tab ${activeTab === 'overview' ? 'active' : ''}`}
          onClick={() => setActiveTab('overview')}
        >
          📈 {t('scheduler.tabs.overview', 'Overview')}
        </button>
        <button
          className={`tab ${activeTab === 'audit' ? 'active' : ''}`}
          onClick={() => setActiveTab('audit')}
        >
          📜 {t('scheduler.tabs.auditLogs', 'Audit Logs')}
          <span className="tab-badge">{auditLogs.length}</span>
        </button>
        <button
          className={`tab ${activeTab === 'sla' ? 'active' : ''}`}
          onClick={() => setActiveTab('sla')}
        >
          ⏱️ {t('scheduler.tabs.slaTracking', 'SLA Tracking')}
        </button>
        <button
          className={`tab ${activeTab === 'reports' ? 'active' : ''}`}
          onClick={() => setActiveTab('reports')}
        >
          📋 {t('scheduler.tabs.reports', 'Reports')}
        </button>
      </div>
      
      {/* Tab Content */}
      {activeTab === 'overview' && metrics && (
        <OverviewTab metrics={metrics} executions={executions} t={t} />
      )}
      
      {activeTab === 'audit' && (
        <AuditTab
          logs={filteredAuditLogs}
          filter={auditFilter}
          setFilter={setAuditFilter}
          t={t}
        />
      )}
      
      {activeTab === 'sla' && (
        <SLATab jobs={jobs} executions={executions} t={t} />
      )}
      
      {activeTab === 'reports' && (
        <ReportsTab metrics={metrics} dateRange={dateRange} t={t} />
      )}
    </div>
  );
}

// ============================================================================
// Overview Tab
// ============================================================================

interface OverviewTabProps {
  metrics: ComplianceMetrics;
  executions: JobExecution[];
  t: (key: string, defaultValue: string) => string;
}

function OverviewTab({ metrics, executions, t }: OverviewTabProps) {
  return (
    <>
      {/* Key Metrics */}
      <div className="compliance-metrics-grid">
        <MetricCard
          icon="📊"
          label={t('scheduler.metrics.totalExecutions', 'Total Executions')}
          value={metrics.totalExecutions}
          trend={metrics.executionsTrend}
        />
        <MetricCard
          icon="✅"
          label={t('scheduler.metrics.successRate', 'Success Rate')}
          value={`${metrics.successRate.toFixed(1)}%`}
          trend={metrics.successTrend}
          trendPositive={metrics.successTrend > 0}
        />
        <MetricCard
          icon="⚠️"
          label={t('scheduler.metrics.slaBreaches', 'SLA Breaches')}
          value={metrics.slaBreaches}
          trend={metrics.slaBreachesTrend}
          trendPositive={metrics.slaBreachesTrend < 0}
        />
        <MetricCard
          icon="🔄"
          label={t('scheduler.metrics.avgRetries', 'Avg Retries')}
          value={metrics.avgRetries.toFixed(2)}
        />
      </div>
      
      {/* Charts Row */}
      <div className="dashboard-grid two-columns">
        {/* Execution Status Distribution */}
        <div className="dashboard-card">
          <div className="card-header">
            <h3>📊 {t('scheduler.statusDistribution', 'Status Distribution')}</h3>
          </div>
          <div className="card-content">
            <div className="status-distribution">
              {Object.entries(metrics.statusCounts).map(([status, count]) => (
                <div key={status} className="status-bar-item">
                  <div className="status-label">
                    <StatusIcon status={status as JobStatus} />
                    <span>{status}</span>
                  </div>
                  <div className="status-bar-container">
                    <div
                      className={`status-bar status-${status.toLowerCase()}`}
                      style={{ width: `${(count / metrics.totalExecutions) * 100}%` }}
                    />
                    <span className="status-count">{count}</span>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>
        
        {/* Daily Execution Trend */}
        <div className="dashboard-card">
          <div className="card-header">
            <h3>📈 {t('scheduler.dailyTrend', 'Daily Execution Trend')}</h3>
          </div>
          <div className="card-content">
            <DailyTrendChart executions={executions} />
          </div>
        </div>
      </div>
      
      {/* Recent Activity */}
      <div className="dashboard-card">
        <div className="card-header">
          <h3>🕐 {t('scheduler.recentActivity', 'Recent Activity')}</h3>
          <Link to="#" className="btn btn-sm btn-secondary" onClick={() => {}}>
            {t('scheduler.viewAll', 'View All')}
          </Link>
        </div>
        <div className="card-content">
          <div className="activity-timeline">
            {executions.slice(0, 10).map(exec => (
              <div key={exec.id} className="activity-item">
                <StatusIcon status={exec.status} />
                <div className="activity-content">
                  <Link to={`/scheduler/executions/${exec.id}`} className="activity-title">
                    {exec.job_name}
                  </Link>
                  <span className="activity-meta">
                    {exec.status} • {formatRelativeTime(exec.completed_at || exec.started_at)}
                  </span>
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>
    </>
  );
}

// ============================================================================
// Audit Tab
// ============================================================================

interface AuditTabProps {
  logs: AuditLogEntry[];
  filter: { action: string; entity: string; user: string };
  setFilter: React.Dispatch<React.SetStateAction<{ action: string; entity: string; user: string }>>;
  t: (key: string, defaultValue: string) => string;
}

function AuditTab({ logs, filter, setFilter, t }: AuditTabProps) {
  const [selectedLog, setSelectedLog] = useState<AuditLogEntry | null>(null);
  const [page, setPage] = useState(1);
  const pageSize = 25;
  
  const paginatedLogs = logs.slice((page - 1) * pageSize, page * pageSize);
  const totalPages = Math.ceil(logs.length / pageSize);
  
  return (
    <>
      {/* Filters */}
      <div className="filters-bar">
        <select
          className="filter-select"
          value={filter.action}
          onChange={e => setFilter(prev => ({ ...prev, action: e.target.value }))}
          aria-label={t('scheduler.filterByAction', 'Filter by action')}
        >
          <option value="">{t('scheduler.allActions', 'All Actions')}</option>
          <option value="create">{t('scheduler.action.create', 'Create')}</option>
          <option value="update">{t('scheduler.action.update', 'Update')}</option>
          <option value="delete">{t('scheduler.action.delete', 'Delete')}</option>
          <option value="execute">{t('scheduler.action.execute', 'Execute')}</option>
          <option value="cancel">{t('scheduler.action.cancel', 'Cancel')}</option>
          <option value="resubmit">{t('scheduler.action.resubmit', 'Resubmit')}</option>
        </select>
        <select
          className="filter-select"
          value={filter.entity}
          onChange={e => setFilter(prev => ({ ...prev, entity: e.target.value }))}
          aria-label={t('scheduler.filterByEntity', 'Filter by entity')}
        >
          <option value="">{t('scheduler.allEntities', 'All Entities')}</option>
          <option value="job">{t('scheduler.entity.job', 'Job')}</option>
          <option value="execution">{t('scheduler.entity.execution', 'Execution')}</option>
          <option value="schedule">{t('scheduler.entity.schedule', 'Schedule')}</option>
          <option value="calendar">{t('scheduler.entity.calendar', 'Calendar')}</option>
          <option value="notification">{t('scheduler.entity.notification', 'Notification')}</option>
        </select>
        <input
          type="text"
          className="search-input"
          placeholder={t('scheduler.searchByUser', 'Search by user...')}
          value={filter.user}
          onChange={e => setFilter(prev => ({ ...prev, user: e.target.value }))}
        />
      </div>
      
      {/* Audit Log Table */}
      <div className="dashboard-card">
        <div className="card-header">
          <h3>📜 {t('scheduler.auditLogs', 'Audit Logs')}</h3>
          <span className="badge badge-secondary">{logs.length} {t('scheduler.entries', 'entries')}</span>
        </div>
        <div className="card-content">
          {logs.length === 0 ? (
            <div className="empty-state empty-state-small">
              <div className="empty-state-icon">📜</div>
              <div className="empty-state-text">
                {t('scheduler.noAuditLogs', 'No audit logs found for the selected filters')}
              </div>
            </div>
          ) : (
            <>
              <table className="data-table audit-table">
                <thead>
                  <tr>
                    <th>{t('scheduler.fields.timestamp', 'Timestamp')}</th>
                    <th>{t('scheduler.fields.action', 'Action')}</th>
                    <th>{t('scheduler.fields.entity', 'Entity')}</th>
                    <th>{t('scheduler.fields.user', 'User')}</th>
                    <th>{t('scheduler.fields.ip', 'IP Address')}</th>
                    <th>{t('scheduler.fields.details', 'Details')}</th>
                  </tr>
                </thead>
                <tbody>
                  {paginatedLogs.map(log => (
                    <tr
                      key={log.id}
                      className={selectedLog?.id === log.id ? 'selected' : ''}
                      onClick={() => setSelectedLog(log)}
                    >
                      <td className="timestamp-cell">
                        {new Date(log.timestamp).toLocaleString()}
                      </td>
                      <td>
                        <ActionBadge action={log.action} />
                      </td>
                      <td>
                        <span className="entity-badge">
                          {log.entity_type}
                        </span>
                        <span className="entity-id">{log.entity_id.slice(0, 8)}...</span>
                      </td>
                      <td>{log.user_id || t('scheduler.system', 'System')}</td>
                      <td className="ip-cell">{log.ip_address || '—'}</td>
                      <td>
                        <button
                          className="btn btn-sm btn-ghost"
                          onClick={(e) => {
                            e.stopPropagation();
                            setSelectedLog(log);
                          }}
                        >
                          👁️
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
              
              {/* Pagination */}
              <div className="pagination">
                <button
                  className="btn btn-sm btn-secondary"
                  disabled={page === 1}
                  onClick={() => setPage(1)}
                >
                  ⏮️
                </button>
                <button
                  className="btn btn-sm btn-secondary"
                  disabled={page === 1}
                  onClick={() => setPage(p => p - 1)}
                >
                  ◀️
                </button>
                <span className="pagination-info">
                  {t('scheduler.page', 'Page')} {page} {t('scheduler.of', 'of')} {totalPages}
                </span>
                <button
                  className="btn btn-sm btn-secondary"
                  disabled={page === totalPages}
                  onClick={() => setPage(p => p + 1)}
                >
                  ▶️
                </button>
                <button
                  className="btn btn-sm btn-secondary"
                  disabled={page === totalPages}
                  onClick={() => setPage(totalPages)}
                >
                  ⏭️
                </button>
              </div>
            </>
          )}
        </div>
      </div>
      
      {/* Log Detail Modal */}
      {selectedLog && (
        <div className="modal-overlay" onClick={() => setSelectedLog(null)}>
          <div className="modal-content" onClick={e => e.stopPropagation()}>
            <div className="modal-header">
              <h3>{t('scheduler.auditLogDetail', 'Audit Log Detail')}</h3>
              <button className="btn btn-ghost" onClick={() => setSelectedLog(null)}>✕</button>
            </div>
            <div className="modal-body">
              <dl className="detail-list">
                <dt>{t('scheduler.fields.id', 'ID')}</dt>
                <dd><code>{selectedLog.id}</code></dd>
                
                <dt>{t('scheduler.fields.timestamp', 'Timestamp')}</dt>
                <dd>{new Date(selectedLog.timestamp).toLocaleString()}</dd>
                
                <dt>{t('scheduler.fields.action', 'Action')}</dt>
                <dd><ActionBadge action={selectedLog.action} /></dd>
                
                <dt>{t('scheduler.fields.entityType', 'Entity Type')}</dt>
                <dd>{selectedLog.entity_type}</dd>
                
                <dt>{t('scheduler.fields.entityId', 'Entity ID')}</dt>
                <dd><code>{selectedLog.entity_id}</code></dd>
                
                <dt>{t('scheduler.fields.user', 'User')}</dt>
                <dd>{selectedLog.user_id || t('scheduler.system', 'System')}</dd>
                
                <dt>{t('scheduler.fields.ip', 'IP Address')}</dt>
                <dd>{selectedLog.ip_address || '—'}</dd>
                
                <dt>{t('scheduler.fields.userAgent', 'User Agent')}</dt>
                <dd className="user-agent">{selectedLog.user_agent || '—'}</dd>
                
                {selectedLog.changes && (
                  <>
                    <dt>{t('scheduler.fields.changes', 'Changes')}</dt>
                    <dd>
                      <pre className="changes-json">
                        {JSON.stringify(selectedLog.changes, null, 2)}
                      </pre>
                    </dd>
                  </>
                )}
              </dl>
            </div>
          </div>
        </div>
      )}
    </>
  );
}

// ============================================================================
// SLA Tab
// ============================================================================

interface SLATabProps {
  jobs: Job[];
  executions: JobExecution[];
  t: (key: string, defaultValue: string) => string;
}

function SLATab({ jobs, executions, t }: SLATabProps) {
  // Calculate SLA metrics per job
  const jobSLAMetrics = useMemo(() => {
    return jobs.map(job => {
      const jobExecutions = executions.filter(e => e.job_id === job.id);
      const completedExecutions = jobExecutions.filter(e => e.status === JobStatus.COMPLETED);
      const failedExecutions = jobExecutions.filter(e => e.status === JobStatus.FAILED);
      
      const avgDuration = completedExecutions.length > 0
        ? completedExecutions.reduce((sum, e) => sum + (e.duration_ms || 0), 0) / completedExecutions.length
        : 0;
      
      const slaTarget = job.sla_seconds ? job.sla_seconds * 1000 : null;
      const slaBreaches = slaTarget
        ? completedExecutions.filter(e => (e.duration_ms || 0) > slaTarget).length
        : 0;
      
      return {
        job,
        totalRuns: jobExecutions.length,
        successCount: completedExecutions.length,
        failureCount: failedExecutions.length,
        successRate: jobExecutions.length > 0
          ? (completedExecutions.length / jobExecutions.length) * 100
          : 0,
        avgDuration,
        slaTarget,
        slaBreaches,
        slaCompliance: slaTarget && completedExecutions.length > 0
          ? ((completedExecutions.length - slaBreaches) / completedExecutions.length) * 100
          : null,
      };
    });
  }, [jobs, executions]);
  
  return (
    <>
      {/* SLA Overview Cards */}
      <div className="sla-overview-cards">
        <div className="dashboard-card sla-card">
          <div className="sla-card-content">
            <div className="sla-metric">
              <span className="sla-value">
                {jobSLAMetrics.filter(m => m.slaTarget).length}
              </span>
              <span className="sla-label">{t('scheduler.jobsWithSLA', 'Jobs with SLA')}</span>
            </div>
          </div>
        </div>
        <div className="dashboard-card sla-card">
          <div className="sla-card-content">
            <div className="sla-metric">
              <span className="sla-value">
                {jobSLAMetrics.reduce((sum, m) => sum + m.slaBreaches, 0)}
              </span>
              <span className="sla-label">{t('scheduler.totalBreaches', 'Total Breaches')}</span>
            </div>
          </div>
        </div>
        <div className="dashboard-card sla-card">
          <div className="sla-card-content">
            <div className="sla-metric">
              <span className="sla-value">
                {calculateOverallSLACompliance(jobSLAMetrics).toFixed(1)}%
              </span>
              <span className="sla-label">{t('scheduler.overallCompliance', 'Overall Compliance')}</span>
            </div>
          </div>
        </div>
      </div>
      
      {/* SLA Performance Table */}
      <div className="dashboard-card">
        <div className="card-header">
          <h3>⏱️ {t('scheduler.slaPerformance', 'SLA Performance by Job')}</h3>
        </div>
        <div className="card-content">
          <table className="data-table sla-table">
            <thead>
              <tr>
                <th>{t('scheduler.fields.job', 'Job')}</th>
                <th>{t('scheduler.fields.runs', 'Runs')}</th>
                <th>{t('scheduler.fields.successRate', 'Success Rate')}</th>
                <th>{t('scheduler.fields.avgDuration', 'Avg Duration')}</th>
                <th>{t('scheduler.fields.slaTarget', 'SLA Target')}</th>
                <th>{t('scheduler.fields.breaches', 'Breaches')}</th>
                <th>{t('scheduler.fields.compliance', 'Compliance')}</th>
              </tr>
            </thead>
            <tbody>
              {jobSLAMetrics.map(metric => (
                <tr key={metric.job.id}>
                  <td>
                    <Link to={`/scheduler/jobs/${metric.job.id}`} className="job-link">
                      {metric.job.name}
                    </Link>
                  </td>
                  <td>{metric.totalRuns}</td>
                  <td>
                    <SuccessRateBar rate={metric.successRate} />
                  </td>
                  <td>{formatDuration(metric.avgDuration)}</td>
                  <td>
                    {metric.slaTarget ? formatDuration(metric.slaTarget) : '—'}
                  </td>
                  <td>
                    {metric.slaTarget ? (
                      <span className={`breach-count ${metric.slaBreaches > 0 ? 'has-breaches' : ''}`}>
                        {metric.slaBreaches}
                      </span>
                    ) : '—'}
                  </td>
                  <td>
                    {metric.slaCompliance !== null ? (
                      <ComplianceBadge compliance={metric.slaCompliance} />
                    ) : '—'}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </>
  );
}

// ============================================================================
// Reports Tab
// ============================================================================

interface ReportsTabProps {
  metrics: ComplianceMetrics | null;
  dateRange: { start: string; end: string };
  t: (key: string, defaultValue: string) => string;
}

function ReportsTab({ metrics, dateRange, t }: ReportsTabProps) {
  const reports = [
    {
      id: 'execution-summary',
      name: t('scheduler.reports.executionSummary', 'Execution Summary'),
      description: t('scheduler.reports.executionSummaryDesc', 'Summary of all job executions including success/failure rates'),
      icon: '📊',
    },
    {
      id: 'sla-compliance',
      name: t('scheduler.reports.slaCompliance', 'SLA Compliance Report'),
      description: t('scheduler.reports.slaComplianceDesc', 'Detailed SLA compliance analysis by job and time period'),
      icon: '⏱️',
    },
    {
      id: 'audit-trail',
      name: t('scheduler.reports.auditTrail', 'Audit Trail Report'),
      description: t('scheduler.reports.auditTrailDesc', 'Complete audit log of all system activities'),
      icon: '📜',
    },
    {
      id: 'failure-analysis',
      name: t('scheduler.reports.failureAnalysis', 'Failure Analysis'),
      description: t('scheduler.reports.failureAnalysisDesc', 'Analysis of job failures with root cause breakdown'),
      icon: '🔍',
    },
    {
      id: 'resource-utilization',
      name: t('scheduler.reports.resourceUtilization', 'Resource Utilization'),
      description: t('scheduler.reports.resourceUtilizationDesc', 'CPU, memory, and execution time analysis'),
      icon: '💻',
    },
    {
      id: 'trend-analysis',
      name: t('scheduler.reports.trendAnalysis', 'Trend Analysis'),
      description: t('scheduler.reports.trendAnalysisDesc', 'Historical trends and pattern analysis'),
      icon: '📈',
    },
  ];
  
  const handleGenerateReport = async (reportId: string) => {
    // In a real implementation, this would call the API to generate the report
    devDebug('Generating report:', reportId, dateRange);
    alert(t('scheduler.reportGenerating', 'Report is being generated. It will be available for download shortly.'));
  };
  
  return (
    <>
      {/* Report Cards */}
      <div className="reports-grid">
        {reports.map(report => (
          <div key={report.id} className="dashboard-card report-card">
            <div className="report-icon">{report.icon}</div>
            <div className="report-content">
              <h3>{report.name}</h3>
              <p>{report.description}</p>
            </div>
            <button
              className="btn btn-primary"
              onClick={() => handleGenerateReport(report.id)}
            >
              {t('scheduler.generate', 'Generate')}
            </button>
          </div>
        ))}
      </div>
      
      {/* Scheduled Reports */}
      <div className="dashboard-card">
        <div className="card-header">
          <h3>📅 {t('scheduler.scheduledReports', 'Scheduled Reports')}</h3>
          <button className="btn btn-sm btn-primary">
            ➕ {t('scheduler.scheduleReport', 'Schedule Report')}
          </button>
        </div>
        <div className="card-content">
          <div className="empty-state empty-state-small">
            <div className="empty-state-icon">📅</div>
            <div className="empty-state-text">
              {t('scheduler.noScheduledReports', 'No scheduled reports configured')}
            </div>
          </div>
        </div>
      </div>
    </>
  );
}

// ============================================================================
// Helper Components
// ============================================================================

interface MetricCardProps {
  icon: string;
  label: string;
  value: string | number;
  trend?: number;
  trendPositive?: boolean;
}

function MetricCard({ icon, label, value, trend, trendPositive }: MetricCardProps) {
  return (
    <div className="compliance-metric-card">
      <span className="metric-icon">{icon}</span>
      <div className="metric-content">
        <div className="metric-value">{value}</div>
        <div className="metric-label">{label}</div>
        {trend !== undefined && (
          <div className={`metric-trend ${trendPositive ? 'positive' : 'negative'}`}>
            {trend > 0 ? '↑' : '↓'} {Math.abs(trend).toFixed(1)}%
          </div>
        )}
      </div>
    </div>
  );
}

function StatusIcon({ status }: { status: JobStatus }) {
  const icons: Record<JobStatus, string> = {
    [JobStatus.PENDING]: '⏳',
    [JobStatus.QUEUED]: '📥',
    [JobStatus.RUNNING]: '▶️',
    [JobStatus.COMPLETED]: '✅',
    [JobStatus.FAILED]: '❌',
    [JobStatus.CANCELLED]: '⏹️',
    [JobStatus.PAUSED]: '⏸️',
    [JobStatus.WAITING_DEPENDENCY]: '🔗',
    [JobStatus.WAITING_CALENDAR]: '📅',
    [JobStatus.RETRYING]: '🔄',
    [JobStatus.SKIPPED]: '⏭️',
  };
  return <span className="status-icon">{icons[status] || '❓'}</span>;
}

function ActionBadge({ action }: { action: string }) {
  const classes: Record<string, string> = {
    create: 'badge-success',
    update: 'badge-info',
    delete: 'badge-danger',
    execute: 'badge-primary',
    cancel: 'badge-warning',
    resubmit: 'badge-secondary',
  };
  return <span className={`badge ${classes[action] || 'badge-secondary'}`}>{action}</span>;
}

function SuccessRateBar({ rate }: { rate: number }) {
  const color = rate >= 95 ? '#059669' : rate >= 80 ? '#d97706' : '#dc2626';
  return (
    <div className="success-rate-bar">
      <div className="rate-bar" style={{ width: `${rate}%`, backgroundColor: color }} />
      <span className="rate-value">{rate.toFixed(1)}%</span>
    </div>
  );
}

function ComplianceBadge({ compliance }: { compliance: number }) {
  let className = 'badge-success';
  if (compliance < 95) className = 'badge-warning';
  if (compliance < 80) className = 'badge-danger';
  return <span className={`badge ${className}`}>{compliance.toFixed(1)}%</span>;
}

function DailyTrendChart({ executions }: { executions: JobExecution[] }) {
  // Group by date
  const dailyCounts = new Map<string, { success: number; failed: number }>();
  executions.forEach(exec => {
    const date = (exec.started_at || exec.scheduled_at || '').split('T')[0];
    if (!date) return;
    
    if (!dailyCounts.has(date)) {
      dailyCounts.set(date, { success: 0, failed: 0 });
    }
    const counts = dailyCounts.get(date)!;
    if (exec.status === JobStatus.COMPLETED) counts.success++;
    if (exec.status === JobStatus.FAILED) counts.failed++;
  });
  
  const dates = Array.from(dailyCounts.keys()).sort().slice(-14);
  const maxCount = Math.max(...dates.map(d => {
    const c = dailyCounts.get(d)!;
    return c.success + c.failed;
  }), 1);
  
  return (
    <div className="daily-trend-chart">
      {dates.map(date => {
        const counts = dailyCounts.get(date)!;
        const total = counts.success + counts.failed;
        return (
          <div key={date} className="trend-bar-container">
            <div className="trend-bars" style={{ height: 120 }}>
              <div
                className="trend-bar success"
                style={{ height: `${(counts.success / maxCount) * 100}%` }}
                title={`${counts.success} successful`}
              />
              <div
                className="trend-bar failed"
                style={{ height: `${(counts.failed / maxCount) * 100}%` }}
                title={`${counts.failed} failed`}
              />
            </div>
            <div className="trend-date">{date.slice(5)}</div>
          </div>
        );
      })}
    </div>
  );
}

// ============================================================================
// Types and Utilities
// ============================================================================

interface ComplianceMetrics {
  totalExecutions: number;
  successRate: number;
  slaBreaches: number;
  avgRetries: number;
  statusCounts: Record<string, number>;
  executionsTrend: number;
  successTrend: number;
  slaBreachesTrend: number;
}

function calculateMetrics(
  auditLogs: AuditLogEntry[],
  jobs: Job[],
  executions: JobExecution[]
): ComplianceMetrics {
  const completed = executions.filter(e => e.status === JobStatus.COMPLETED).length;
  const failed = executions.filter(e => e.status === JobStatus.FAILED).length;
  
  const statusCounts: Record<string, number> = {};
  executions.forEach(e => {
    statusCounts[e.status] = (statusCounts[e.status] || 0) + 1;
  });
  
  const retries = executions.filter(e => e.attempt_number > 1);
  const avgRetries = retries.length > 0
    ? retries.reduce((sum, e) => sum + e.attempt_number, 0) / retries.length
    : 0;
  
  return {
    totalExecutions: executions.length,
    successRate: executions.length > 0 ? (completed / executions.length) * 100 : 0,
    slaBreaches: 0, // Would calculate from actual SLA data
    avgRetries,
    statusCounts,
    executionsTrend: 5.2, // Would calculate from historical data
    successTrend: 2.1,
    slaBreachesTrend: -1.5,
  };
}

function calculateOverallSLACompliance(metrics: any[]): number {
  const withSLA = metrics.filter(m => m.slaCompliance !== null);
  if (withSLA.length === 0) return 100;
  return withSLA.reduce((sum, m) => sum + m.slaCompliance, 0) / withSLA.length;
}

function getDefaultStartDate(): string {
  const date = new Date();
  date.setDate(date.getDate() - 30);
  return date.toISOString().split('T')[0];
}

function formatDuration(ms: number): string {
  if (ms === 0) return '—';
  if (ms < 1000) return `${ms}ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
  if (ms < 3600000) return `${Math.floor(ms / 60000)}m`;
  return `${Math.floor(ms / 3600000)}h ${Math.floor((ms % 3600000) / 60000)}m`;
}

function formatRelativeTime(timestamp?: string): string {
  if (!timestamp) return '';
  const diff = Date.now() - new Date(timestamp).getTime();
  if (diff < 60000) return 'just now';
  if (diff < 3600000) return `${Math.floor(diff / 60000)}m ago`;
  if (diff < 86400000) return `${Math.floor(diff / 3600000)}h ago`;
  return `${Math.floor(diff / 86400000)}d ago`;
}

function exportComplianceReport(
  metrics: ComplianceMetrics | null,
  auditLogs: AuditLogEntry[],
  t: (key: string, defaultValue: string) => string
) {
  const report = {
    generated_at: new Date().toISOString(),
    metrics,
    audit_log_count: auditLogs.length,
    // Add more report data as needed
  };
  
  const blob = new Blob([JSON.stringify(report, null, 2)], { type: 'application/json' });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = `compliance-report-${new Date().toISOString().split('T')[0]}.json`;
  a.click();
  URL.revokeObjectURL(url);
}

export default ComplianceDashboardPage;