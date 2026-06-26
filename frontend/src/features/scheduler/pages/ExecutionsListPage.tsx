/**
 * World-Class Enterprise Scheduler - Executions List Page
 * View and manage all job executions with filtering and bulk actions
 */

import React, { useState, useEffect, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { Link, useSearchParams } from 'react-router-dom';
import * as schedulerService from '../services/schedulerService';
import { JobExecution, JobStatus } from '../../../types/scheduler';
import '../styles/SchedulerDashboard.css';

export function ExecutionsListPage() {
  const { t } = useTranslation();
  const [searchParams, setSearchParams] = useSearchParams();
  
  const [executions, setExecutions] = useState<JobExecution[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [totalCount, setTotalCount] = useState(0);
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
  
  // Filters from URL params
  const page = parseInt(searchParams.get('page') || '1');
  const limit = parseInt(searchParams.get('limit') || '25');
  const statusFilter = searchParams.get('status') || '';
  const jobFilter = searchParams.get('job') || '';
  const dateFrom = searchParams.get('from') || '';
  const dateTo = searchParams.get('to') || '';
  
  // Load executions
  const loadExecutions = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      
      const response = await schedulerService.listAllExecutions({
        page,
        limit,
        status: statusFilter as JobStatus | undefined,
        job_id: jobFilter || undefined,
        start_date: dateFrom || undefined,
        end_date: dateTo || undefined,
      });
      
      setExecutions(response.executions);
      setTotalCount(response.total);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load executions');
    } finally {
      setLoading(false);
    }
  }, [page, limit, statusFilter, jobFilter, dateFrom, dateTo]);
  
  useEffect(() => {
    loadExecutions();
  }, [loadExecutions]);
  
  // Update filters
  const updateFilter = (key: string, value: string) => {
    const params = new URLSearchParams(searchParams);
    if (value) {
      params.set(key, value);
    } else {
      params.delete(key);
    }
    params.set('page', '1'); // Reset to first page
    setSearchParams(params);
  };
  
  // Handle selection
  const handleSelectAll = () => {
    if (selectedIds.size === executions.length) {
      setSelectedIds(new Set());
    } else {
      setSelectedIds(new Set(executions.map(e => e.id)));
    }
  };
  
  const handleSelect = (id: string) => {
    const newSet = new Set(selectedIds);
    if (newSet.has(id)) {
      newSet.delete(id);
    } else {
      newSet.add(id);
    }
    setSelectedIds(newSet);
  };
  
  // Bulk actions
  const handleBulkCancel = async () => {
    if (!confirm(t('scheduler.confirmBulkCancel', 'Cancel selected executions?'))) return;
    
    try {
      await Promise.all(
        Array.from(selectedIds).map(id => 
          schedulerService.cancelExecution(id, 'Bulk cancelled by user')
        )
      );
      setSelectedIds(new Set());
      loadExecutions();
    } catch (err) {
      console.error('Bulk cancel failed:', err);
    }
  };
  
  const handleBulkResubmit = async () => {
    if (!confirm(t('scheduler.confirmBulkResubmit', 'Resubmit selected executions?'))) return;
    
    try {
      await Promise.all(
        Array.from(selectedIds).map(id => schedulerService.resubmitExecution(id))
      );
      setSelectedIds(new Set());
      loadExecutions();
    } catch (err) {
      console.error('Bulk resubmit failed:', err);
    }
  };
  
  const totalPages = Math.ceil(totalCount / limit);
  
  return (
    <div className="scheduler-dashboard">
      {/* Header */}
      <div className="scheduler-header">
        <div>
          <h1>📋 {t('scheduler.executions', 'Executions')}</h1>
          <p className="header-subtitle">
            {t('scheduler.executionsDesc', 'View and manage all job executions')}
          </p>
        </div>
        <div className="scheduler-header-actions">
          <button className="btn btn-secondary" onClick={loadExecutions}>
            🔄 {t('scheduler.refresh', 'Refresh')}
          </button>
        </div>
      </div>
      
      {/* Filters */}
      <div className="filters-bar">
        <select
          className="filter-select"
          value={statusFilter}
          onChange={e => updateFilter('status', e.target.value)}
          aria-label={t('scheduler.filterByStatus', 'Filter by status')}
        >
          <option value="">{t('scheduler.allStatuses', 'All Statuses')}</option>
          {Object.values(JobStatus).map(status => (
            <option key={status} value={status}>{status}</option>
          ))}
        </select>
        
        <input
          type="date"
          className="form-control filter-date"
          value={dateFrom}
          onChange={e => updateFilter('from', e.target.value)}
          placeholder={t('scheduler.from', 'From')}
        />
        <input
          type="date"
          className="form-control filter-date"
          value={dateTo}
          onChange={e => updateFilter('to', e.target.value)}
          placeholder={t('scheduler.to', 'To')}
        />
        
        {(statusFilter || dateFrom || dateTo) && (
          <button
            className="btn btn-sm btn-secondary"
            onClick={() => setSearchParams(new URLSearchParams())}
          >
            ✕ {t('scheduler.clearFilters', 'Clear')}
          </button>
        )}
      </div>
      
      {/* Bulk Actions */}
      {selectedIds.size > 0 && (
        <div className="bulk-actions-bar">
          <span className="selection-count">
            {selectedIds.size} {t('scheduler.selected', 'selected')}
          </span>
          <button className="btn btn-sm btn-danger" onClick={handleBulkCancel}>
            ⏹️ {t('scheduler.cancel', 'Cancel')}
          </button>
          <button className="btn btn-sm btn-primary" onClick={handleBulkResubmit}>
            🔄 {t('scheduler.resubmit', 'Resubmit')}
          </button>
          <button className="btn btn-sm btn-secondary" onClick={() => setSelectedIds(new Set())}>
            ✕ {t('scheduler.clearSelection', 'Clear Selection')}
          </button>
        </div>
      )}
      
      {/* Error State */}
      {error && (
        <div className="error-banner">
          <span>⚠️ {error}</span>
          <button onClick={loadExecutions}>{t('scheduler.retry', 'Retry')}</button>
        </div>
      )}
      
      {/* Loading State */}
      {loading ? (
        <div className="loading-spinner">
          <div className="spinner" />
        </div>
      ) : executions.length === 0 ? (
        <div className="empty-state">
          <div className="empty-state-icon">📋</div>
          <div className="empty-state-text">
            {statusFilter || dateFrom || dateTo
              ? t('scheduler.noExecutionsMatch', 'No executions match your filters')
              : t('scheduler.noExecutions', 'No executions yet')}
          </div>
        </div>
      ) : (
        <>
          {/* Executions Table */}
          <div className="dashboard-card">
            <table className="data-table">
              <thead>
                <tr>
                  <th className="checkbox-cell">
                    <input
                      type="checkbox"
                      checked={selectedIds.size === executions.length && executions.length > 0}
                      onChange={handleSelectAll}
                      aria-label={t('scheduler.selectAll', 'Select all')}
                    />
                  </th>
                  <th>{t('scheduler.fields.job', 'Job')}</th>
                  <th>{t('scheduler.fields.status', 'Status')}</th>
                  <th>{t('scheduler.fields.started', 'Started')}</th>
                  <th>{t('scheduler.fields.duration', 'Duration')}</th>
                  <th>{t('scheduler.fields.attempt', 'Attempt')}</th>
                  <th>{t('scheduler.fields.triggeredBy', 'Triggered By')}</th>
                  <th className="actions-cell">{t('scheduler.actions', 'Actions')}</th>
                </tr>
              </thead>
              <tbody>
                {executions.map(execution => (
                  <tr
                    key={execution.id}
                    className={selectedIds.has(execution.id) ? 'selected' : ''}
                  >
                    <td className="checkbox-cell">
                      <input
                        type="checkbox"
                        checked={selectedIds.has(execution.id)}
                        onChange={() => handleSelect(execution.id)}
                        aria-label={t('scheduler.select', 'Select')}
                      />
                    </td>
                    <td>
                      <Link to={`/scheduler/jobs/${execution.job_id}`} className="job-link">
                        {execution.job_name}
                      </Link>
                      <div className="execution-id">#{execution.run_number}</div>
                    </td>
                    <td>
                      <StatusBadge status={execution.status} />
                    </td>
                    <td>
                      {execution.started_at
                        ? new Date(execution.started_at).toLocaleString()
                        : execution.scheduled_at
                          ? new Date(execution.scheduled_at).toLocaleString()
                          : '—'}
                    </td>
                    <td>
                      {execution.duration_ms ? formatDuration(execution.duration_ms) : '—'}
                    </td>
                    <td>
                      {execution.attempt_number}/{execution.max_attempts}
                    </td>
                    <td>
                      <TriggerBadge trigger={execution.triggered_by} />
                    </td>
                    <td className="actions-cell">
                      <Link
                        to={`/scheduler/executions/${execution.id}`}
                        className="btn btn-sm btn-ghost"
                        title={t('scheduler.viewDetails', 'View Details')}
                      >
                        👁️
                      </Link>
                      {execution.status === JobStatus.RUNNING && (
                        <button
                          className="btn btn-sm btn-ghost"
                          onClick={() => cancelExecution(execution.id, loadExecutions)}
                          title={t('scheduler.cancel', 'Cancel')}
                        >
                          ⏹️
                        </button>
                      )}
                      {(execution.status === JobStatus.FAILED || execution.status === JobStatus.CANCELLED) && (
                        <button
                          className="btn btn-sm btn-ghost"
                          onClick={() => resubmitExecution(execution.id, loadExecutions)}
                          title={t('scheduler.resubmit', 'Resubmit')}
                        >
                          🔄
                        </button>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          
          {/* Pagination */}
          <div className="pagination">
            <button
              className="btn btn-sm btn-secondary"
              disabled={page === 1}
              onClick={() => updateFilter('page', '1')}
            >
              ⏮️
            </button>
            <button
              className="btn btn-sm btn-secondary"
              disabled={page === 1}
              onClick={() => updateFilter('page', String(page - 1))}
            >
              ◀️
            </button>
            <span className="pagination-info">
              {t('scheduler.page', 'Page')} {page} {t('scheduler.of', 'of')} {totalPages}
              <span className="pagination-total">
                ({totalCount} {t('scheduler.total', 'total')})
              </span>
            </span>
            <button
              className="btn btn-sm btn-secondary"
              disabled={page === totalPages}
              onClick={() => updateFilter('page', String(page + 1))}
            >
              ▶️
            </button>
            <button
              className="btn btn-sm btn-secondary"
              disabled={page === totalPages}
              onClick={() => updateFilter('page', String(totalPages))}
            >
              ⏭️
            </button>
          </div>
        </>
      )}
    </div>
  );
}

// ============================================================================
// Helper Components
// ============================================================================

function StatusBadge({ status }: { status: JobStatus }) {
  const statusConfig: Record<JobStatus, { className: string; icon: string }> = {
    [JobStatus.PENDING]: { className: 'badge-warning', icon: '⏳' },
    [JobStatus.QUEUED]: { className: 'badge-info', icon: '📥' },
    [JobStatus.RUNNING]: { className: 'badge-info', icon: '▶️' },
    [JobStatus.COMPLETED]: { className: 'badge-success', icon: '✅' },
    [JobStatus.FAILED]: { className: 'badge-danger', icon: '❌' },
    [JobStatus.CANCELLED]: { className: 'badge-secondary', icon: '⏹️' },
    [JobStatus.PAUSED]: { className: 'badge-warning', icon: '⏸️' },
    [JobStatus.WAITING_DEPENDENCY]: { className: 'badge-info', icon: '🔗' },
    [JobStatus.WAITING_CALENDAR]: { className: 'badge-info', icon: '📅' },
    [JobStatus.RETRYING]: { className: 'badge-warning', icon: '🔄' },
    [JobStatus.SKIPPED]: { className: 'badge-secondary', icon: '⏭️' },
  };
  
  const config = statusConfig[status] || { className: 'badge-secondary', icon: '❓' };
  
  return (
    <span className={`badge ${config.className}`}>
      {config.icon} {status}
    </span>
  );
}

function TriggerBadge({ trigger }: { trigger: string }) {
  const triggerIcons: Record<string, string> = {
    schedule: '📅',
    manual: '👤',
    api: '🔌',
    dependency: '🔗',
    event: '⚡',
    resubmit: '🔄',
  };
  
  return (
    <span className="trigger-badge">
      {triggerIcons[trigger] || '❓'} {trigger}
    </span>
  );
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

async function cancelExecution(id: string, onSuccess: () => void) {
  try {
    await schedulerService.cancelExecution(id, 'Cancelled by user');
    onSuccess();
  } catch (err) {
    console.error('Failed to cancel:', err);
  }
}

async function resubmitExecution(id: string, onSuccess: () => void) {
  try {
    await schedulerService.resubmitExecution(id);
    onSuccess();
  } catch (err) {
    console.error('Failed to resubmit:', err);
  }
}

export default ExecutionsListPage;
