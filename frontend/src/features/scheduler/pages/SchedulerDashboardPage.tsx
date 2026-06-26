/**
 * World-Class Enterprise Scheduler - Dashboard Page
 * Main scheduler dashboard with real-time metrics, job monitoring, and quick actions
 */

import React, { useState, useEffect, useCallback, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { Link, useNavigate } from 'react-router-dom';
import * as schedulerService from '../services/schedulerService';
import {
  Job,
  JobExecution,
  JobStatus,
  SchedulerDashboardMetrics,
} from '../../../types/scheduler';
import '../styles/SchedulerDashboard.css';

// ============================================================================
// Dashboard Component
// ============================================================================

export function SchedulerDashboardPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  
  const [metrics, setMetrics] = useState<SchedulerDashboardMetrics | null>(null);
  const [recentExecutions, setRecentExecutions] = useState<JobExecution[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [refreshKey, setRefreshKey] = useState(0);
  
  // Load dashboard data
  const loadDashboardData = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      
      const [metricsData, executionsData] = await Promise.all([
        schedulerService.getDashboardMetrics(),
        schedulerService.listExecutions({ has_errors: false }, 1, 10),
      ]);
      
      setMetrics(metricsData);
      setRecentExecutions(executionsData.data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load dashboard data');
    } finally {
      setLoading(false);
    }
  }, []);
  
  useEffect(() => {
    loadDashboardData();
  }, [loadDashboardData, refreshKey]);
  
  // Auto-refresh every 30 seconds
  useEffect(() => {
    const interval = setInterval(() => {
      setRefreshKey(k => k + 1);
    }, 30000);
    return () => clearInterval(interval);
  }, []);
  
  // Subscribe to real-time updates
  useEffect(() => {
    const unsubscribe = schedulerService.subscribeToJobUpdates(
      () => setRefreshKey(k => k + 1),
      () => setRefreshKey(k => k + 1)
    );
    return unsubscribe;
  }, []);
  
  const handleRefresh = () => setRefreshKey(k => k + 1);
  
  const handleResubmit = async (executionId: string) => {
    try {
      await schedulerService.resubmitExecution(executionId);
      setRefreshKey(k => k + 1);
    } catch (err) {
      console.error('Failed to resubmit:', err);
    }
  };
  
  if (loading && !metrics) {
    return (
      <div className="scheduler-dashboard">
        <div className="loading-spinner">
          <div className="spinner" />
        </div>
      </div>
    );
  }
  
  if (error) {
    return (
      <div className="scheduler-dashboard">
        <div className="empty-state">
          <div className="empty-state-icon">⚠️</div>
          <div className="empty-state-text">{error}</div>
          <button className="btn btn-primary" onClick={handleRefresh} style={{ marginTop: 16 }}>
            {t('scheduler.retry', 'Retry')}
          </button>
        </div>
      </div>
    );
  }
  
  return (
    <div className="scheduler-dashboard">
      {/* Header */}
      <div className="scheduler-header">
        <h1>
          📅 {t('scheduler.dashboard.title', 'Job Scheduler')}
        </h1>
        <div className="scheduler-header-actions">
          <button className="btn btn-secondary" onClick={handleRefresh}>
            🔄 {t('scheduler.refresh', 'Refresh')}
          </button>
          <Link to="/scheduler/jobs/create" className="btn btn-primary">
            ➕ {t('scheduler.createJob', 'Create Job')}
          </Link>
        </div>
      </div>
      
      {/* Metrics Grid */}
      <MetricsGrid metrics={metrics} />
      
      {/* Quick Actions */}
      <QuickActions />
      
      {/* Main Dashboard Grid */}
      <div className="dashboard-grid">
        {/* Left Column */}
        <div>
          {/* Running Jobs */}
          <RunningJobsCard
            runningCount={metrics?.running_now || 0}
            queuedCount={metrics?.queued_now || 0}
          />
          
          {/* Recent Failures */}
          <RecentFailuresCard
            failures={metrics?.recent_failures || []}
            onResubmit={handleResubmit}
          />
        </div>
        
        {/* Right Column */}
        <div>
          {/* Upcoming Jobs */}
          <UpcomingJobsCard
            jobs={metrics?.next_scheduled_jobs || []}
          />
          
          {/* SLA Status */}
          <SLAStatusCard
            complianceRate={metrics?.sla_compliance_rate || 0}
            breachesToday={metrics?.sla_breaches_today || 0}
          />
        </div>
      </div>
    </div>
  );
}

// ============================================================================
// Metrics Grid Component
// ============================================================================

interface MetricsGridProps {
  metrics: SchedulerDashboardMetrics | null;
}

function MetricsGrid({ metrics }: MetricsGridProps) {
  const { t } = useTranslation();
  
  const metricCards = useMemo(() => [
    {
      label: t('scheduler.metrics.totalJobs', 'Total Jobs'),
      value: metrics?.total_jobs || 0,
      type: 'info',
      icon: '📋',
    },
    {
      label: t('scheduler.metrics.activeJobs', 'Active Jobs'),
      value: metrics?.active_jobs || 0,
      type: 'info',
      icon: '✅',
    },
    {
      label: t('scheduler.metrics.runningNow', 'Running Now'),
      value: metrics?.running_now || 0,
      type: 'info',
      icon: '▶️',
    },
    {
      label: t('scheduler.metrics.successfulToday', 'Successful Today'),
      value: metrics?.successful_today || 0,
      type: 'success',
      icon: '✓',
    },
    {
      label: t('scheduler.metrics.failedToday', 'Failed Today'),
      value: metrics?.failed_today || 0,
      type: metrics?.failed_today ? 'error' : 'success',
      icon: '✗',
    },
    {
      label: t('scheduler.metrics.successRate', 'Success Rate (7d)'),
      value: `${(metrics?.success_rate_7d || 0).toFixed(1)}%`,
      type: (metrics?.success_rate_7d || 0) >= 95 ? 'success' : 'warning',
      icon: '📈',
    },
    {
      label: t('scheduler.metrics.avgDuration', 'Avg Duration'),
      value: formatDuration(metrics?.average_duration_ms || 0),
      type: 'info',
      icon: '⏱️',
    },
    {
      label: t('scheduler.metrics.queueDepth', 'Queue Depth'),
      value: metrics?.queue_depth || 0,
      type: (metrics?.queue_depth || 0) > 100 ? 'warning' : 'info',
      icon: '📥',
    },
  ], [metrics, t]);
  
  return (
    <div className="metrics-grid">
      {metricCards.map((card, index) => (
        <div key={index} className={`metric-card ${card.type}`}>
          <div className="metric-value">{card.value}</div>
          <div className="metric-label">
            <span style={{ marginRight: 4 }}>{card.icon}</span>
            {card.label}
          </div>
        </div>
      ))}
    </div>
  );
}

// ============================================================================
// Quick Actions Component
// ============================================================================

function QuickActions() {
  const { t } = useTranslation();
  
  const actions = [
    { icon: '➕', label: t('scheduler.actions.createJob', 'Create Job'), to: '/scheduler/jobs/create' },
    { icon: '📊', label: t('scheduler.actions.viewAll', 'All Jobs'), to: '/scheduler/jobs' },
    { icon: '🔗', label: t('scheduler.actions.chains', 'Job Chains'), to: '/scheduler/chains' },
    { icon: '📅', label: t('scheduler.actions.calendars', 'Calendars'), to: '/scheduler/calendars' },
    { icon: '🔔', label: t('scheduler.actions.notifications', 'Notifications'), to: '/scheduler/notifications' },
    { icon: '📋', label: t('scheduler.actions.audit', 'Audit Log'), to: '/scheduler/audit' },
    { icon: '📈', label: t('scheduler.actions.compliance', 'Compliance'), to: '/scheduler/compliance' },
    { icon: '⚙️', label: t('scheduler.actions.settings', 'Settings'), to: '/scheduler/settings' },
  ];
  
  return (
    <div className="quick-actions">
      {actions.map((action, index) => (
        <Link key={index} to={action.to} className="quick-action-btn">
          <span className="quick-action-icon">{action.icon}</span>
          <span className="quick-action-label">{action.label}</span>
        </Link>
      ))}
    </div>
  );
}

// ============================================================================
// Running Jobs Card
// ============================================================================

interface RunningJobsCardProps {
  runningCount: number;
  queuedCount: number;
}

function RunningJobsCard({ runningCount, queuedCount }: RunningJobsCardProps) {
  const { t } = useTranslation();
  const [executions, setExecutions] = useState<JobExecution[]>([]);
  const [loading, setLoading] = useState(true);
  
  useEffect(() => {
    const loadRunning = async () => {
      try {
        const result = await schedulerService.listExecutions(
          { status: [JobStatus.RUNNING] },
          1,
          5
        );
        setExecutions(result.data);
      } catch (err) {
        console.error('Failed to load running jobs:', err);
      } finally {
        setLoading(false);
      }
    };
    loadRunning();
    
    const interval = setInterval(loadRunning, 5000);
    return () => clearInterval(interval);
  }, []);
  
  return (
    <div className="dashboard-card" style={{ marginBottom: 24 }}>
      <div className="card-header">
        <h3>
          ▶️ {t('scheduler.runningJobs', 'Running Jobs')}
          <span className="status-badge running">{runningCount}</span>
        </h3>
        <span style={{ fontSize: 12, color: 'var(--text-secondary)' }}>
          {queuedCount} {t('scheduler.queued', 'queued')}
        </span>
      </div>
      <div className="card-content">
        {loading ? (
          <div className="loading-spinner">
            <div className="spinner" />
          </div>
        ) : executions.length === 0 ? (
          <div className="empty-state">
            <div className="empty-state-icon">😴</div>
            <div className="empty-state-text">
              {t('scheduler.noRunningJobs', 'No jobs currently running')}
            </div>
          </div>
        ) : (
          <div className="running-jobs-list">
            {executions.map(exec => (
              <div key={exec.id} className="running-job-item">
                <div className="running-job-header">
                  <span className="running-job-name">{exec.job_name}</span>
                  <span className="running-job-status running">
                    <span className="status-dot running" />
                    {t('scheduler.status.running', 'Running')}
                  </span>
                </div>
                <div className="running-job-progress">
                  <div className="progress-bar">
                    <div
                      className="progress-fill"
                      style={{ width: `${exec.progress_percent || 0}%` }}
                    />
                  </div>
                </div>
                <div className="running-job-meta">
                  <span>
                    {exec.current_step || t('scheduler.step', 'Step')} {exec.completed_steps || 0}/{exec.total_steps || '?'}
                  </span>
                  <span>{formatDuration(Date.now() - new Date(exec.started_at || '').getTime())}</span>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

// ============================================================================
// Recent Failures Card
// ============================================================================

interface RecentFailuresCardProps {
  failures: Array<{
    execution_id: string;
    job_id: string;
    job_name: string;
    failed_at: string;
    error_message?: string;
    attempt_number: number;
  }>;
  onResubmit: (executionId: string) => void;
}

function RecentFailuresCard({ failures, onResubmit }: RecentFailuresCardProps) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  
  return (
    <div className="dashboard-card">
      <div className="card-header">
        <h3>
          ⚠️ {t('scheduler.recentFailures', 'Recent Failures')}
          {failures.length > 0 && (
            <span className="status-badge failed">{failures.length}</span>
          )}
        </h3>
        <Link to="/scheduler/executions?status=failed" className="btn btn-sm btn-secondary">
          {t('scheduler.viewAll', 'View All')}
        </Link>
      </div>
      <div className="card-content">
        {failures.length === 0 ? (
          <div className="empty-state">
            <div className="empty-state-icon">✅</div>
            <div className="empty-state-text">
              {t('scheduler.noRecentFailures', 'No recent failures')}
            </div>
          </div>
        ) : (
          <div className="failures-list">
            {failures.slice(0, 5).map(failure => (
              <div key={failure.execution_id} className="failure-item">
                <div className="failure-icon">✗</div>
                <div className="failure-details">
                  <div className="failure-job-name">{failure.job_name}</div>
                  <div className="failure-message">
                    {failure.error_message || t('scheduler.unknownError', 'Unknown error')}
                  </div>
                  <div className="failure-time">
                    {formatRelativeTime(failure.failed_at)} • 
                    {t('scheduler.attempt', 'Attempt')} #{failure.attempt_number}
                  </div>
                </div>
                <div className="failure-actions">
                  <button
                    className="btn btn-sm btn-secondary"
                    onClick={() => navigate(`/scheduler/executions/${failure.execution_id}`)}
                  >
                    {t('scheduler.details', 'Details')}
                  </button>
                  <button
                    className="btn btn-sm btn-primary"
                    onClick={() => onResubmit(failure.execution_id)}
                  >
                    {t('scheduler.resubmit', 'Resubmit')}
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

// ============================================================================
// Upcoming Jobs Card
// ============================================================================

interface UpcomingJobsCardProps {
  jobs: Array<{
    job_id: string;
    job_name: string;
    scheduled_at: string;
    schedule_name?: string;
  }>;
}

function UpcomingJobsCard({ jobs }: UpcomingJobsCardProps) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  
  return (
    <div className="dashboard-card" style={{ marginBottom: 24 }}>
      <div className="card-header">
        <h3>
          ⏰ {t('scheduler.upcomingJobs', 'Upcoming Jobs')}
        </h3>
        <Link to="/scheduler/schedules" className="btn btn-sm btn-secondary">
          {t('scheduler.viewSchedules', 'View Schedules')}
        </Link>
      </div>
      <div className="card-content">
        {jobs.length === 0 ? (
          <div className="empty-state">
            <div className="empty-state-icon">📅</div>
            <div className="empty-state-text">
              {t('scheduler.noUpcoming', 'No upcoming scheduled jobs')}
            </div>
          </div>
        ) : (
          <div className="upcoming-jobs-list">
            {jobs.slice(0, 5).map((job, index) => (
              <div
                key={index}
                className="upcoming-job-item"
                onClick={() => navigate(`/scheduler/jobs/${job.job_id}`)}
              >
                <div className="upcoming-job-icon">📋</div>
                <div className="upcoming-job-details">
                  <div className="upcoming-job-name">{job.job_name}</div>
                  <div className="upcoming-job-time">
                    {formatRelativeTime(job.scheduled_at)}
                    {job.schedule_name && ` • ${job.schedule_name}`}
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

// ============================================================================
// SLA Status Card
// ============================================================================

interface SLAStatusCardProps {
  complianceRate: number;
  breachesToday: number;
}

function SLAStatusCard({ complianceRate, breachesToday }: SLAStatusCardProps) {
  const { t } = useTranslation();
  
  const statusColor = complianceRate >= 99 ? '#10b981' : complianceRate >= 95 ? '#f59e0b' : '#ef4444';
  
  return (
    <div className="dashboard-card">
      <div className="card-header">
        <h3>
          📊 {t('scheduler.slaStatus', 'SLA Compliance')}
        </h3>
        <Link to="/scheduler/compliance" className="btn btn-sm btn-secondary">
          {t('scheduler.details', 'Details')}
        </Link>
      </div>
      <div className="card-content">
        <div style={{ textAlign: 'center', padding: '20px 0' }}>
          <div
            style={{
              fontSize: 48,
              fontWeight: 700,
              color: statusColor,
              marginBottom: 8,
            }}
          >
            {complianceRate.toFixed(1)}%
          </div>
          <div style={{ color: 'var(--text-secondary)', fontSize: 14 }}>
            {t('scheduler.slaComplianceRate', 'Compliance Rate')}
          </div>
          {breachesToday > 0 && (
            <div
              style={{
                marginTop: 16,
                padding: '8px 16px',
                background: '#fef2f2',
                borderRadius: 8,
                color: '#991b1b',
                fontSize: 13,
              }}
            >
              ⚠️ {breachesToday} {t('scheduler.breachesToday', 'breaches today')}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

// ============================================================================
// Utility Functions
// ============================================================================

function formatDuration(ms: number): string {
  if (ms < 1000) return `${ms}ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
  if (ms < 3600000) return `${Math.floor(ms / 60000)}m ${Math.floor((ms % 60000) / 1000)}s`;
  return `${Math.floor(ms / 3600000)}h ${Math.floor((ms % 3600000) / 60000)}m`;
}

function formatRelativeTime(dateStr: string): string {
  const date = new Date(dateStr);
  const now = new Date();
  const diffMs = date.getTime() - now.getTime();
  const diffMins = Math.round(diffMs / 60000);
  const diffHours = Math.round(diffMs / 3600000);
  const diffDays = Math.round(diffMs / 86400000);
  
  if (diffMs < 0) {
    // Past
    const absMins = Math.abs(diffMins);
    const absHours = Math.abs(diffHours);
    const absDays = Math.abs(diffDays);
    
    if (absMins < 60) return `${absMins}m ago`;
    if (absHours < 24) return `${absHours}h ago`;
    return `${absDays}d ago`;
  } else {
    // Future
    if (diffMins < 60) return `in ${diffMins}m`;
    if (diffHours < 24) return `in ${diffHours}h`;
    return `in ${diffDays}d`;
  }
}

export default SchedulerDashboardPage;
