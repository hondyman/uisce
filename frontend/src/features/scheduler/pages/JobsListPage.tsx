/**
 * World-Class Enterprise Scheduler - Jobs List Page
 * Comprehensive job listing with filtering, sorting, and bulk actions
 */

import React, { useState, useEffect, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { Link, useNavigate, useSearchParams } from 'react-router-dom';
import * as schedulerService from '../services/schedulerService';
import {
  Job,
  JobStatus,
  JobPriority,
  JobListFilters,
  PaginatedResponse,
} from '../../../types/scheduler';
import '../styles/SchedulerDashboard.css';

// ============================================================================
// Jobs List Page
// ============================================================================

export function JobsListPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  
  const [jobs, setJobs] = useState<PaginatedResponse<Job> | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selectedJobs, setSelectedJobs] = useState<Set<string>>(new Set());
  
  // Filters
  const [search, setSearch] = useState(searchParams.get('search') || '');
  const [statusFilter, setStatusFilter] = useState<string[]>(
    searchParams.get('status')?.split(',').filter(Boolean) || []
  );
  const [priorityFilter, setPriorityFilter] = useState<string[]>(
    searchParams.get('priority')?.split(',').filter(Boolean) || []
  );
  const [enabledFilter, setEnabledFilter] = useState<string>(
    searchParams.get('enabled') || 'all'
  );
  
  // Pagination
  const [page, setPage] = useState(parseInt(searchParams.get('page') || '1'));
  const pageSize = 20;
  
  // Load jobs
  const loadJobs = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      
      const filters: JobListFilters = {};
      if (search) filters.search = search;
      if (statusFilter.length) filters.status = statusFilter as JobStatus[];
      if (priorityFilter.length) filters.priority = priorityFilter as JobPriority[];
      if (enabledFilter !== 'all') filters.enabled = enabledFilter === 'enabled';
      
      const result = await schedulerService.listJobs(filters, page, pageSize);
      setJobs(result);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load jobs');
    } finally {
      setLoading(false);
    }
  }, [search, statusFilter, priorityFilter, enabledFilter, page]);
  
  useEffect(() => {
    loadJobs();
  }, [loadJobs]);
  
  // Update URL params
  useEffect(() => {
    const params = new URLSearchParams();
    if (search) params.set('search', search);
    if (statusFilter.length) params.set('status', statusFilter.join(','));
    if (priorityFilter.length) params.set('priority', priorityFilter.join(','));
    if (enabledFilter !== 'all') params.set('enabled', enabledFilter);
    if (page > 1) params.set('page', String(page));
    setSearchParams(params, { replace: true });
  }, [search, statusFilter, priorityFilter, enabledFilter, page, setSearchParams]);
  
  // Handlers
  const handleSearch = (e: React.ChangeEvent<HTMLInputElement>) => {
    setSearch(e.target.value);
    setPage(1);
  };
  
  const handleStatusChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    const value = e.target.value;
    setStatusFilter(value ? [value] : []);
    setPage(1);
  };
  
  const handlePriorityChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    const value = e.target.value;
    setPriorityFilter(value ? [value] : []);
    setPage(1);
  };
  
  const handleEnabledChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    setEnabledFilter(e.target.value);
    setPage(1);
  };
  
  const handleSelectJob = (jobId: string) => {
    setSelectedJobs(prev => {
      const next = new Set(prev);
      if (next.has(jobId)) {
        next.delete(jobId);
      } else {
        next.add(jobId);
      }
      return next;
    });
  };
  
  const handleSelectAll = () => {
    if (selectedJobs.size === (jobs?.data.length || 0)) {
      setSelectedJobs(new Set());
    } else {
      setSelectedJobs(new Set(jobs?.data.map(j => j.id) || []));
    }
  };
  
  const handleBulkAction = async (action: 'pause' | 'resume' | 'delete') => {
    if (selectedJobs.size === 0) return;
    
    try {
      for (const jobId of selectedJobs) {
        if (action === 'pause') {
          await schedulerService.pauseJob(jobId);
        } else if (action === 'resume') {
          await schedulerService.resumeJob(jobId);
        } else if (action === 'delete') {
          await schedulerService.deleteJob(jobId);
        }
      }
      setSelectedJobs(new Set());
      loadJobs();
    } catch (err) {
      console.error(`Bulk ${action} failed:`, err);
    }
  };
  
  const handleTriggerJob = async (jobId: string) => {
    try {
      await schedulerService.triggerJob(jobId);
      // Could show a toast notification here
    } catch (err) {
      console.error('Failed to trigger job:', err);
    }
  };
  
  return (
    <div className="scheduler-dashboard">
      {/* Header */}
      <div className="scheduler-header">
        <h1>
          📋 {t('scheduler.jobs.title', 'Jobs')}
        </h1>
        <div className="scheduler-header-actions">
          <button className="btn btn-secondary" onClick={loadJobs}>
            🔄 {t('scheduler.refresh', 'Refresh')}
          </button>
          <Link to="/scheduler/jobs/create" className="btn btn-primary">
            ➕ {t('scheduler.createJob', 'Create Job')}
          </Link>
        </div>
      </div>
      
      {/* Filters */}
      <div className="filters-bar">
        <input
          type="text"
          aria-label={t('scheduler.jobs.searchPlaceholder', 'Search jobs by name...')}
          className="search-input"
          placeholder={t('scheduler.jobs.searchPlaceholder', 'Search jobs by name...')}
          value={search}
          onChange={handleSearch}
        />
        
        <select
          aria-label={t('scheduler.filters.status', 'Status filter')}
          className="filter-select"
          value={statusFilter[0] || ''}
          onChange={handleStatusChange}
        >
          <option value="">{t('scheduler.filters.allStatuses', 'All Statuses')}</option>
          <option value="pending">{t('scheduler.status.pending', 'Pending')}</option>
          <option value="running">{t('scheduler.status.running', 'Running')}</option>
          <option value="completed">{t('scheduler.status.completed', 'Completed')}</option>
          <option value="failed">{t('scheduler.status.failed', 'Failed')}</option>
          <option value="paused">{t('scheduler.status.paused', 'Paused')}</option>
        </select>
        
        <select
          aria-label={t('scheduler.filters.priority', 'Priority filter')}
          className="filter-select"
          value={priorityFilter[0] || ''}
          onChange={handlePriorityChange}
        >
          <option value="">{t('scheduler.filters.allPriorities', 'All Priorities')}</option>
          <option value="critical">{t('scheduler.priority.critical', 'Critical')}</option>
          <option value="high">{t('scheduler.priority.high', 'High')}</option>
          <option value="medium">{t('scheduler.priority.medium', 'Medium')}</option>
          <option value="low">{t('scheduler.priority.low', 'Low')}</option>
        </select>
        
        <select
          aria-label={t('scheduler.filters.enabled', 'Enabled/disabled filter')}
          className="filter-select"
          value={enabledFilter}
          onChange={handleEnabledChange}
        >
          <option value="all">{t('scheduler.filters.allJobs', 'All Jobs')}</option>
          <option value="enabled">{t('scheduler.filters.enabled', 'Enabled Only')}</option>
          <option value="disabled">{t('scheduler.filters.disabled', 'Disabled Only')}</option>
        </select>
      </div>
      
      {/* Bulk Actions */}
      {selectedJobs.size > 0 && (
        <div className="bulk-actions" style={{
          display: 'flex',
          gap: 12,
          padding: 12,
          background: 'var(--card-bg)',
          borderRadius: 8,
          marginBottom: 16,
          alignItems: 'center',
        }}>
          <span style={{ fontWeight: 500 }}>
            {selectedJobs.size} {t('scheduler.selected', 'selected')}
          </span>
          <button className="btn btn-sm btn-secondary" onClick={() => handleBulkAction('pause')}>
            ⏸️ {t('scheduler.pause', 'Pause')}
          </button>
          <button className="btn btn-sm btn-secondary" onClick={() => handleBulkAction('resume')}>
            ▶️ {t('scheduler.resume', 'Resume')}
          </button>
          <button className="btn btn-sm btn-danger" onClick={() => handleBulkAction('delete')}>
            🗑️ {t('scheduler.delete', 'Delete')}
          </button>
        </div>
      )}
      
      {/* Jobs Table */}
      <div className="dashboard-card">
        <div className="card-content" style={{ padding: 0 }}>
          {loading ? (
            <div className="loading-spinner">
              <div className="spinner" />
            </div>
          ) : error ? (
            <div className="empty-state">
              <div className="empty-state-icon">⚠️</div>
              <div className="empty-state-text">{error}</div>
            </div>
          ) : jobs?.data.length === 0 ? (
            <div className="empty-state">
              <div className="empty-state-icon">📋</div>
              <div className="empty-state-text">
                {t('scheduler.jobs.noJobs', 'No jobs found')}
              </div>
              <Link to="/scheduler/jobs/create" className="btn btn-primary" style={{ marginTop: 16 }}>
                {t('scheduler.createJob', 'Create Job')}
              </Link>
            </div>
          ) : (
            <table className="jobs-table">
              <thead>
                <tr>
                    <th style={{ width: 40 }}>
                    <input
                      type="checkbox"
                      aria-label={t('scheduler.jobs.selectAll', 'Select all jobs')}
                      checked={selectedJobs.size === jobs?.data.length}
                      onChange={handleSelectAll}
                    />
                  </th>
                  <th>{t('scheduler.jobs.name', 'Name')}</th>
                  <th>{t('scheduler.jobs.type', 'Type')}</th>
                  <th>{t('scheduler.jobs.priority', 'Priority')}</th>
                  <th>{t('scheduler.jobs.schedule', 'Schedule')}</th>
                  <th>{t('scheduler.jobs.lastRun', 'Last Run')}</th>
                  <th>{t('scheduler.jobs.status', 'Status')}</th>
                  <th style={{ width: 120 }}>{t('scheduler.jobs.actions', 'Actions')}</th>
                </tr>
              </thead>
              <tbody>
                {jobs?.data.map(job => (
                  <tr key={job.id}>
                    <td>
                      <input
                        type="checkbox"
                        aria-label={`Select job ${job.name}`}
                        checked={selectedJobs.has(job.id)}
                        onChange={() => handleSelectJob(job.id)}
                      />
                    </td>
                    <td>
                      <div className="job-name-cell">
                        <div className="job-icon">
                          {getJobIcon(job.job_type)}
                        </div>
                        <div>
                          <Link
                            to={`/scheduler/jobs/${job.id}`}
                            className="job-name-text"
                            style={{ color: 'inherit', textDecoration: 'none' }}
                          >
                            {job.name}
                          </Link>
                          {job.description && (
                            <div className="job-type-text">{job.description.substring(0, 50)}...</div>
                          )}
                        </div>
                      </div>
                    </td>
                    <td>
                      <span className="job-type-text">{job.job_type}</span>
                    </td>
                    <td>
                      <PriorityBadge priority={job.priority} />
                    </td>
                    <td>
                      {job.schedule ? (
                        <span className="job-type-text">
                          {formatSchedule(job.schedule)}
                        </span>
                      ) : (
                        <span style={{ color: 'var(--text-secondary)' }}>—</span>
                      )}
                    </td>
                    <td>
                      <span className="job-type-text">
                        {job.schedule?.last_run_at
                          ? formatRelativeTime(job.schedule.last_run_at)
                          : '—'}
                      </span>
                    </td>
                    <td>
                      <StatusBadge enabled={job.enabled} />
                    </td>
                    <td>
                      <div style={{ display: 'flex', gap: 4 }}>
                        <button
                          className="btn btn-sm btn-secondary btn-icon"
                          onClick={() => handleTriggerJob(job.id)}
                          title={t('scheduler.runNow', 'Run Now')}
                        >
                          ▶️
                        </button>
                        <button
                          className="btn btn-sm btn-secondary btn-icon"
                          onClick={() => navigate(`/scheduler/jobs/${job.id}/edit`)}
                          title={t('scheduler.edit', 'Edit')}
                        >
                          ✏️
                        </button>
                        <button
                          className="btn btn-sm btn-secondary btn-icon"
                          onClick={() => navigate(`/scheduler/jobs/${job.id}`)}
                          title={t('scheduler.view', 'View')}
                        >
                          👁️
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
        
        {/* Pagination */}
        {jobs && jobs.total_pages > 1 && (
          <div style={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            padding: 16,
            borderTop: '1px solid var(--border-color)',
          }}>
            <span style={{ fontSize: 13, color: 'var(--text-secondary)' }}>
              {t('scheduler.pagination.showing', 'Showing')} {(page - 1) * pageSize + 1} -{' '}
              {Math.min(page * pageSize, jobs.total)} {t('scheduler.pagination.of', 'of')}{' '}
              {jobs.total}
            </span>
            <div style={{ display: 'flex', gap: 8 }}>
              <button
                className="btn btn-sm btn-secondary"
                disabled={page === 1}
                onClick={() => setPage(p => p - 1)}
              >
                ← {t('scheduler.pagination.prev', 'Previous')}
              </button>
              <button
                className="btn btn-sm btn-secondary"
                disabled={page === jobs.total_pages}
                onClick={() => setPage(p => p + 1)}
              >
                {t('scheduler.pagination.next', 'Next')} →
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

// ============================================================================
// Helper Components
// ============================================================================

function PriorityBadge({ priority }: { priority: JobPriority }) {
  const colors: Record<JobPriority, { bg: string; color: string }> = {
    [JobPriority.CRITICAL]: { bg: '#fee2e2', color: '#991b1b' },
    [JobPriority.HIGH]: { bg: '#fef3c7', color: '#92400e' },
    [JobPriority.MEDIUM]: { bg: '#dbeafe', color: '#1e40af' },
    [JobPriority.LOW]: { bg: '#f3f4f6', color: '#4b5563' },
  };
  
  const style = colors[priority] || colors[JobPriority.MEDIUM];
  
  return (
    <span
      style={{
        padding: '4px 10px',
        borderRadius: 9999,
        fontSize: 11,
        fontWeight: 500,
        background: style.bg,
        color: style.color,
        textTransform: 'uppercase',
      }}
    >
      {priority}
    </span>
  );
}

function StatusBadge({ enabled }: { enabled: boolean }) {
  return (
    <span
      className={`status-badge ${enabled ? 'completed' : 'pending'}`}
      style={{ textTransform: 'capitalize' }}
    >
      {enabled ? '✓ Active' : '⏸ Disabled'}
    </span>
  );
}

// ============================================================================
// Utility Functions
// ============================================================================

function getJobIcon(jobType: string): string {
  const icons: Record<string, string> = {
    etl: '🔄',
    report: '📊',
    export: '📤',
    import: '📥',
    notification: '🔔',
    cleanup: '🧹',
    backup: '💾',
    sync: '🔁',
    calculation: '🧮',
    validation: '✅',
  };
  return icons[jobType.toLowerCase()] || '📋';
}

function formatSchedule(schedule: any): string {
  if (!schedule) return '—';
  
  if (schedule.schedule_type === 'once' && schedule.run_at) {
    return `Once at ${new Date(schedule.run_at).toLocaleDateString()}`;
  }
  
  if (schedule.cron_expression) {
    return `Cron: ${schedule.cron_expression}`;
  }
  
  if (schedule.recurrence) {
    const { pattern, interval, days_of_week, time_of_day } = schedule.recurrence;
    let str = pattern.charAt(0).toUpperCase() + pattern.slice(1);
    if (interval > 1) str = `Every ${interval} ${pattern}`;
    if (days_of_week?.length) str += ` (${days_of_week.map((d: string) => d.substring(0, 3)).join(', ')})`;
    if (time_of_day) str += ` at ${time_of_day}`;
    return str;
  }
  
  return schedule.schedule_type || '—';
}

function formatRelativeTime(dateStr: string): string {
  const date = new Date(dateStr);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMins = Math.round(diffMs / 60000);
  const diffHours = Math.round(diffMs / 3600000);
  const diffDays = Math.round(diffMs / 86400000);
  
  if (diffMins < 60) return `${diffMins}m ago`;
  if (diffHours < 24) return `${diffHours}h ago`;
  if (diffDays < 30) return `${diffDays}d ago`;
  return date.toLocaleDateString();
}

export default JobsListPage;
