/**
 * World-Class Enterprise Scheduler - Execution Monitor Page
 * Real-time monitoring of job executions with logs, progress tracking, and resubmit capability
 */

import React, { useState, useEffect, useCallback, useRef } from 'react';
import { useTranslation } from 'react-i18next';
import { useParams, useNavigate, Link } from 'react-router-dom';
import * as schedulerService from '../services/schedulerService';
import {
  JobExecution,
  ExecutionLog,
  JobStatus,
} from '../../../types/scheduler';
import '../styles/SchedulerDashboard.css';

// ============================================================================
// Execution Detail Page
// ============================================================================

export function ExecutionDetailPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { executionId } = useParams<{ executionId: string }>();
  
  const [execution, setExecution] = useState<JobExecution | null>(null);
  const [logs, setLogs] = useState<ExecutionLog[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [logFilter, setLogFilter] = useState<string>('all');
  const [autoScroll, setAutoScroll] = useState(true);
  const [resubmitting, setResubmitting] = useState(false);
  
  const logsEndRef = useRef<HTMLDivElement>(null);
  
  // Load execution details
  const loadExecution = useCallback(async () => {
    if (!executionId) return;
    
    try {
      setLoading(true);
      const [exec, execLogs] = await Promise.all([
        schedulerService.getExecution(executionId),
        schedulerService.getExecutionLogs(executionId),
      ]);
      setExecution(exec);
      setLogs(execLogs);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load execution');
    } finally {
      setLoading(false);
    }
  }, [executionId]);
  
  useEffect(() => {
    loadExecution();
  }, [loadExecution]);
  
  // Subscribe to real-time updates if execution is running
  useEffect(() => {
    if (!executionId || !execution) return;
    if (execution.status !== JobStatus.RUNNING && execution.status !== JobStatus.QUEUED) return;
    
    const unsubscribe = schedulerService.subscribeToExecutionUpdates(
      executionId,
      (updatedExecution) => {
        setExecution(updatedExecution);
      },
      (newLog) => {
        setLogs(prev => [...prev, newLog]);
      },
      (err) => {
        console.error('WebSocket error:', err);
      }
    );
    
    return unsubscribe;
  }, [executionId, execution?.status]);
  
  // Auto-scroll to bottom of logs
  useEffect(() => {
    if (autoScroll && logsEndRef.current) {
      logsEndRef.current.scrollIntoView({ behavior: 'smooth' });
    }
  }, [logs, autoScroll]);
  
  // Handlers
  const handleCancel = async () => {
    if (!executionId) return;
    try {
      await schedulerService.cancelExecution(executionId, 'Cancelled by user');
      loadExecution();
    } catch (err) {
      console.error('Failed to cancel:', err);
    }
  };
  
  const handleResubmit = async () => {
    if (!executionId) return;
    try {
      setResubmitting(true);
      const newExecution = await schedulerService.resubmitExecution(executionId);
      navigate(`/scheduler/executions/${newExecution.id}`);
    } catch (err) {
      console.error('Failed to resubmit:', err);
    } finally {
      setResubmitting(false);
    }
  };
  
  // Filter logs
  const filteredLogs = logs.filter(log => {
    if (logFilter === 'all') return true;
    return log.level === logFilter;
  });
  
  if (loading) {
    return (
      <div className="scheduler-dashboard">
        <div className="loading-spinner">
          <div className="spinner" />
        </div>
      </div>
    );
  }
  
  if (error || !execution) {
    return (
      <div className="scheduler-dashboard">
        <div className="empty-state">
          <div className="empty-state-icon">⚠️</div>
          <div className="empty-state-text">{error || 'Execution not found'}</div>
          <Link to="/scheduler/executions" className="btn btn-primary" style={{ marginTop: 16 }}>
            {t('scheduler.backToExecutions', 'Back to Executions')}
          </Link>
        </div>
      </div>
    );
  }
  
  return (
    <div className="scheduler-dashboard">
      {/* Header */}
      <div className="scheduler-header">
        <div>
          <h1>
            <ExecutionStatusIcon status={execution.status} />
            {execution.job_name}
          </h1>
          <p style={{ color: 'var(--text-secondary)', margin: '4px 0 0' }}>
            {t('scheduler.execution.run', 'Run')} #{execution.run_number} •{' '}
            {t('scheduler.execution.attempt', 'Attempt')} {execution.attempt_number}/{execution.max_attempts}
          </p>
        </div>
        <div className="scheduler-header-actions">
          {(execution.status === JobStatus.RUNNING || execution.status === JobStatus.QUEUED) && (
            <button className="btn btn-danger" onClick={handleCancel}>
              ⏹️ {t('scheduler.cancel', 'Cancel')}
            </button>
          )}
          {(execution.status === JobStatus.FAILED || execution.status === JobStatus.CANCELLED) && (
            <button
              className="btn btn-primary"
              onClick={handleResubmit}
              disabled={resubmitting}
            >
              🔄 {resubmitting ? t('scheduler.resubmitting', 'Resubmitting...') : t('scheduler.resubmit', 'Resubmit')}
            </button>
          )}
          <Link to={`/scheduler/jobs/${execution.job_id}`} className="btn btn-secondary">
            {t('scheduler.viewJob', 'View Job')}
          </Link>
        </div>
      </div>
      
      {/* Status Banner */}
      <StatusBanner execution={execution} t={t} />
      
      {/* Progress (if running) */}
      {execution.status === JobStatus.RUNNING && (
        <ProgressCard execution={execution} t={t} />
      )}
      
      {/* Error Details (if failed) */}
      {execution.status === JobStatus.FAILED && execution.error_message && (
        <ErrorCard execution={execution} t={t} />
      )}
      
      {/* Main Content Grid */}
      <div className="dashboard-grid">
        {/* Execution Details */}
        <div>
          <ExecutionDetailsCard execution={execution} t={t} />
          
          {/* Execution Logs */}
          <div className="dashboard-card" style={{ marginTop: 24 }}>
            <div className="card-header">
              <h3>📜 {t('scheduler.execution.logs', 'Execution Logs')}</h3>
              <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
                <select
                  className="filter-select"
                  value={logFilter}
                  onChange={e => setLogFilter(e.target.value)}
                  style={{ width: 120 }}
                  title={t('scheduler.execution.logFilter', 'Log Filter')}
                >
                  <option value="all">{t('scheduler.logs.all', 'All Levels')}</option>
                  <option value="debug">{t('scheduler.logs.debug', 'Debug')}</option>
                  <option value="info">{t('scheduler.logs.info', 'Info')}</option>
                  <option value="warn">{t('scheduler.logs.warn', 'Warning')}</option>
                  <option value="error">{t('scheduler.logs.error', 'Error')}</option>
                </select>
                <label style={{ display: 'flex', alignItems: 'center', gap: 4, fontSize: 13 }}>
                  <input
                    type="checkbox"
                    checked={autoScroll}
                    onChange={e => setAutoScroll(e.target.checked)}
                  />
                  {t('scheduler.autoScroll', 'Auto-scroll')}
                </label>
              </div>
            </div>
            <div className="logs-container">
              {filteredLogs.length === 0 ? (
                <div className="empty-state">
                  <div className="empty-state-icon">📜</div>
                  <div className="empty-state-text">
                    {t('scheduler.noLogs', 'No logs available')}
                  </div>
                </div>
              ) : (
                <div className="logs-list">
                  {filteredLogs.map((log, index) => (
                    <LogEntry key={log.id || index} log={log} />
                  ))}
                  <div ref={logsEndRef} />
                </div>
              )}
            </div>
          </div>
        </div>
        
        {/* Right Sidebar */}
        <div>
          {/* Timeline */}
          <TimelineCard execution={execution} t={t} />
          
          {/* Output (if available) */}
          {execution.output && Object.keys(execution.output).length > 0 && (
            <OutputCard output={execution.output} t={t} />
          )}
        </div>
      </div>
    </div>
  );
}

// ============================================================================
// Status Banner
// ============================================================================

function StatusBanner({ execution, t }: { execution: JobExecution; t: any }) {
  const statusConfig: Record<JobStatus, { bg: string; color: string; icon: string; label: string }> = {
    [JobStatus.PENDING]: { bg: '#fef3c7', color: '#92400e', icon: '⏳', label: t('scheduler.status.pending', 'Pending') },
    [JobStatus.QUEUED]: { bg: '#e0e7ff', color: '#3730a3', icon: '📥', label: t('scheduler.status.queued', 'Queued') },
    [JobStatus.RUNNING]: { bg: '#dbeafe', color: '#1e40af', icon: '▶️', label: t('scheduler.status.running', 'Running') },
    [JobStatus.COMPLETED]: { bg: '#d1fae5', color: '#065f46', icon: '✅', label: t('scheduler.status.completed', 'Completed') },
    [JobStatus.FAILED]: { bg: '#fee2e2', color: '#991b1b', icon: '❌', label: t('scheduler.status.failed', 'Failed') },
    [JobStatus.CANCELLED]: { bg: '#f3f4f6', color: '#4b5563', icon: '⏹️', label: t('scheduler.status.cancelled', 'Cancelled') },
    [JobStatus.PAUSED]: { bg: '#fef3c7', color: '#92400e', icon: '⏸️', label: t('scheduler.status.paused', 'Paused') },
    [JobStatus.WAITING_DEPENDENCY]: { bg: '#e0e7ff', color: '#3730a3', icon: '🔗', label: t('scheduler.status.waitingDependency', 'Waiting for Dependency') },
    [JobStatus.WAITING_CALENDAR]: { bg: '#e0e7ff', color: '#3730a3', icon: '📅', label: t('scheduler.status.waitingCalendar', 'Waiting for Calendar') },
    [JobStatus.RETRYING]: { bg: '#fef3c7', color: '#92400e', icon: '🔄', label: t('scheduler.status.retrying', 'Retrying') },
    [JobStatus.SKIPPED]: { bg: '#f3f4f6', color: '#4b5563', icon: '⏭️', label: t('scheduler.status.skipped', 'Skipped') },
  };
  
  const config = statusConfig[execution.status] || statusConfig[JobStatus.PENDING];
  
  return (
    <div
      className="status-banner"
      style={{
        background: config.bg,
        color: config.color,
        padding: 16,
        borderRadius: 12,
        marginBottom: 24,
        display: 'flex',
        alignItems: 'center',
        gap: 12,
      }}
    >
      <span style={{ fontSize: 24 }}>{config.icon}</span>
      <div>
        <div style={{ fontWeight: 600, fontSize: 16 }}>{config.label}</div>
        <div style={{ fontSize: 13, opacity: 0.8 }}>
          {execution.status === JobStatus.RUNNING && execution.current_step && (
            <span>{execution.current_step}</span>
          )}
          {execution.status === JobStatus.COMPLETED && execution.duration_ms && (
            <span>{t('scheduler.completedIn', 'Completed in')} {formatDuration(execution.duration_ms)}</span>
          )}
          {execution.status === JobStatus.FAILED && (
            <span>{t('scheduler.failedAfter', 'Failed after')} {execution.attempt_number} {t('scheduler.attempts', 'attempts')}</span>
          )}
          {execution.status === JobStatus.RETRYING && execution.next_retry_at && (
            <span>{t('scheduler.nextRetryAt', 'Next retry at')} {new Date(execution.next_retry_at).toLocaleTimeString()}</span>
          )}
        </div>
      </div>
    </div>
  );
}

// ============================================================================
// Progress Card
// ============================================================================

function ProgressCard({ execution, t }: { execution: JobExecution; t: any }) {
  const progress = execution.progress_percent || 0;
  
  return (
    <div className="dashboard-card" style={{ marginBottom: 24 }}>
      <div className="card-content">
        <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 8 }}>
          <span style={{ fontWeight: 600 }}>
            {execution.current_step || t('scheduler.processing', 'Processing...')}
          </span>
          <span style={{ fontWeight: 600 }}>{progress}%</span>
        </div>
        <div className="progress-bar" style={{ height: 12, borderRadius: 6 }}>
          <div
            className="progress-fill"
            style={{
              width: `${progress}%`,
              height: '100%',
              borderRadius: 6,
              transition: 'width 0.5s ease',
            }}
          />
        </div>
        <div style={{ display: 'flex', justifyContent: 'space-between', marginTop: 8, fontSize: 13, color: 'var(--text-secondary)' }}>
          <span>
            {t('scheduler.step', 'Step')} {execution.completed_steps || 0} / {execution.total_steps || '?'}
          </span>
          <span>
            {execution.started_at && formatDuration(Date.now() - new Date(execution.started_at).getTime())} {t('scheduler.elapsed', 'elapsed')}
          </span>
        </div>
      </div>
    </div>
  );
}

// ============================================================================
// Error Card
// ============================================================================

function ErrorCard({ execution, t }: { execution: JobExecution; t: any }) {
  const [showStack, setShowStack] = useState(false);
  
  return (
    <div className="dashboard-card error-card" style={{ marginBottom: 24, borderColor: '#fecaca', background: '#fef2f2' }}>
      <div className="card-header" style={{ background: '#fee2e2', borderColor: '#fecaca' }}>
        <h3 style={{ color: '#991b1b' }}>
          ❌ {t('scheduler.errorDetails', 'Error Details')}
        </h3>
        {execution.error_code && (
          <span style={{ fontSize: 12, background: '#991b1b', color: 'white', padding: '4px 8px', borderRadius: 4 }}>
            {execution.error_code}
          </span>
        )}
      </div>
      <div className="card-content">
        <p style={{ color: '#991b1b', fontWeight: 500, marginBottom: 12 }}>
          {execution.error_message}
        </p>
        {execution.error_stack && (
          <>
            <button
              className="btn btn-sm btn-secondary"
              onClick={() => setShowStack(!showStack)}
            >
              {showStack ? t('scheduler.hideStack', 'Hide Stack Trace') : t('scheduler.showStack', 'Show Stack Trace')}
            </button>
            {showStack && (
              <pre style={{
                marginTop: 12,
                padding: 12,
                background: '#1a1a1a',
                color: '#f1f1f1',
                borderRadius: 8,
                overflow: 'auto',
                fontSize: 12,
                maxHeight: 300,
              }}>
                {execution.error_stack}
              </pre>
            )}
          </>
        )}
      </div>
    </div>
  );
}

// ============================================================================
// Execution Details Card
// ============================================================================

function ExecutionDetailsCard({ execution, t }: { execution: JobExecution; t: any }) {
  const details = [
    { label: t('scheduler.fields.executionId', 'Execution ID'), value: execution.id },
    { label: t('scheduler.fields.jobId', 'Job ID'), value: execution.job_id },
    { label: t('scheduler.fields.triggeredBy', 'Triggered By'), value: formatTrigger(execution.triggered_by, execution.triggered_by_user_id) },
    { label: t('scheduler.fields.correlationId', 'Correlation ID'), value: execution.correlation_id || '—' },
    { label: t('scheduler.fields.workerId', 'Worker ID'), value: execution.worker_id || '—' },
    { label: t('scheduler.fields.workerHost', 'Worker Host'), value: execution.worker_host || '—' },
  ];
  
  if (execution.memory_used_mb) {
    details.push({ label: t('scheduler.fields.memoryUsed', 'Memory Used'), value: `${execution.memory_used_mb} MB` });
  }
  if (execution.cpu_seconds) {
    details.push({ label: t('scheduler.fields.cpuTime', 'CPU Time'), value: `${execution.cpu_seconds}s` });
  }
  
  return (
    <div className="dashboard-card">
      <div className="card-header">
        <h3>📋 {t('scheduler.executionDetails', 'Execution Details')}</h3>
      </div>
      <div className="card-content">
        <dl className="details-list">
          {details.map((detail, index) => (
            <div key={index} className="detail-row">
              <dt>{detail.label}</dt>
              <dd>{detail.value}</dd>
            </div>
          ))}
        </dl>
      </div>
    </div>
  );
}

// ============================================================================
// Timeline Card
// ============================================================================

function TimelineCard({ execution, t }: { execution: JobExecution; t: any }) {
  const events = [
    { label: t('scheduler.timeline.scheduled', 'Scheduled'), time: execution.scheduled_at, icon: '📅' },
    { label: t('scheduler.timeline.queued', 'Queued'), time: execution.queued_at, icon: '📥' },
    { label: t('scheduler.timeline.started', 'Started'), time: execution.started_at, icon: '▶️' },
    { label: t('scheduler.timeline.completed', 'Completed'), time: execution.completed_at, icon: execution.status === JobStatus.COMPLETED ? '✅' : execution.status === JobStatus.FAILED ? '❌' : '⏳' },
  ].filter(e => e.time);
  
  return (
    <div className="dashboard-card" style={{ marginBottom: 24 }}>
      <div className="card-header">
        <h3>⏱️ {t('scheduler.timeline.title', 'Timeline')}</h3>
      </div>
      <div className="card-content">
        <div className="timeline">
          {events.map((event, index) => (
            <div key={index} className="timeline-event">
              <div className="timeline-icon">{event.icon}</div>
              <div className="timeline-content">
                <div className="timeline-label">{event.label}</div>
                <div className="timeline-time">
                  {new Date(event.time!).toLocaleString()}
                </div>
              </div>
            </div>
          ))}
        </div>
        
        {execution.duration_ms && (
          <div className="duration-summary" style={{ marginTop: 16, paddingTop: 16, borderTop: '1px solid var(--border-color)' }}>
            <strong>{t('scheduler.totalDuration', 'Total Duration')}:</strong> {formatDuration(execution.duration_ms)}
          </div>
        )}
      </div>
    </div>
  );
}

// ============================================================================
// Output Card
// ============================================================================

function OutputCard({ output, t }: { output: Record<string, unknown>; t: any }) {
  return (
    <div className="dashboard-card">
      <div className="card-header">
        <h3>📤 {t('scheduler.output', 'Output')}</h3>
      </div>
      <div className="card-content">
        <pre style={{
          background: 'var(--code-bg, #f3f4f6)',
          padding: 12,
          borderRadius: 8,
          overflow: 'auto',
          fontSize: 12,
          maxHeight: 300,
        }}>
          {JSON.stringify(output, null, 2)}
        </pre>
      </div>
    </div>
  );
}

// ============================================================================
// Log Entry Component
// ============================================================================

function LogEntry({ log }: { log: ExecutionLog }) {
  const levelColors: Record<string, string> = {
    debug: '#6b7280',
    info: '#3b82f6',
    warn: '#f59e0b',
    error: '#ef4444',
  };
  
  return (
    <div className="log-entry" style={{
      display: 'flex',
      gap: 12,
      padding: '8px 0',
      borderBottom: '1px solid var(--border-color, #e5e7eb)',
      fontSize: 13,
      fontFamily: 'monospace',
    }}>
      <span style={{ color: 'var(--text-secondary)', minWidth: 80 }}>
        {new Date(log.timestamp).toLocaleTimeString()}
      </span>
      <span style={{
        color: levelColors[log.level] || '#6b7280',
        fontWeight: 600,
        minWidth: 50,
        textTransform: 'uppercase',
      }}>
        {log.level}
      </span>
      {log.step && (
        <span style={{ color: 'var(--text-secondary)', minWidth: 100 }}>
          [{log.step}]
        </span>
      )}
      <span style={{ flex: 1, color: log.level === 'error' ? '#ef4444' : 'inherit' }}>
        {log.message}
      </span>
    </div>
  );
}

// ============================================================================
// Execution Status Icon
// ============================================================================

function ExecutionStatusIcon({ status }: { status: JobStatus }) {
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
  
  return <span style={{ marginRight: 8 }}>{icons[status] || '❓'}</span>;
}

// ============================================================================
// Utility Functions
// ============================================================================

function formatDuration(ms: number): string {
  if (ms < 1000) return `${ms}ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
  if (ms < 3600000) return `${Math.floor(ms / 60000)}m ${Math.floor((ms % 60000) / 1000)}s`;
  const hours = Math.floor(ms / 3600000);
  const mins = Math.floor((ms % 3600000) / 60000);
  return `${hours}h ${mins}m`;
}

function formatTrigger(triggeredBy: string, userId?: string): string {
  const labels: Record<string, string> = {
    schedule: '📅 Scheduled',
    manual: '👤 Manual',
    api: '🔌 API',
    dependency: '🔗 Dependency',
    event: '⚡ Event',
    resubmit: '🔄 Resubmit',
  };
  
  let label = labels[triggeredBy] || triggeredBy;
  if (userId) {
    label += ` (${userId})`;
  }
  return label;
}

export default ExecutionDetailPage;
