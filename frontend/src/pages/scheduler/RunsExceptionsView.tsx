/**
 * Runs & Exceptions View - Monitor job runs, exceptions, and failures
 */

import React, { useState, useMemo, useEffect } from 'react';
import { 
  Search, 
  Filter, 
  RefreshCw,
  Clock,
  CheckCircle,
  XCircle,
  AlertTriangle,
  Pause,
  PlayCircle,
  Eye,
  RotateCcw
} from 'lucide-react';
import { useTenantContext } from '../../hooks/useTenantContext';
import { getJobRuns, getDAGRuns, JobRun, DAGRun } from '../../api/schedulerApi';

type RunFilter = 'all' | 'running' | 'completed' | 'failed' | 'exceptions';

const RunsExceptionsView: React.FC = () => {
  const { selectedTenant } = useTenantContext();
  const tenantId = selectedTenant?.id;
  const [runs, setRuns] = useState<JobRun[]>([]);
  const [loading, setLoading] = useState(true);
  const [filter, setFilter] = useState<RunFilter>('all');
  const [searchQuery, setSearchQuery] = useState('');
  const [autoRefresh, setAutoRefresh] = useState(true);
  const [selectedRun, setSelectedRun] = useState<JobRun | null>(null);

  // Mock data for demonstration since we don't have a list-all-runs endpoint
  useEffect(() => {
    const mockRuns: JobRun[] = [
      {
        id: '1',
        job_id: 'job-1',
        tenant_id: tenantId || '',
        status: 'running',
        attempt_number: 1,
        trigger_type: 'scheduled',
        started_at: new Date().toISOString(),
        slo_breached: false,
        created_at: new Date().toISOString(),
        semantic_bindings: { bo_ids: ['Customer'] }
      },
      {
        id: '2',
        job_id: 'job-2',
        tenant_id: tenantId || '',
        status: 'completed',
        attempt_number: 1,
        trigger_type: 'scheduled',
        started_at: new Date(Date.now() - 3600000).toISOString(),
        completed_at: new Date(Date.now() - 3000000).toISOString(),
        duration_ms: 600000,
        slo_breached: false,
        created_at: new Date(Date.now() - 3600000).toISOString()
      },
      {
        id: '3',
        job_id: 'job-3',
        tenant_id: tenantId || '',
        status: 'failed',
        attempt_number: 2,
        trigger_type: 'manual',
        started_at: new Date(Date.now() - 7200000).toISOString(),
        completed_at: new Date(Date.now() - 7000000).toISOString(),
        error_message: 'Connection timeout',
        slo_breached: true,
        slo_target_ms: 300000,
        created_at: new Date(Date.now() - 7200000).toISOString()
      },
      {
        id: '4',
        job_id: 'job-4',
        tenant_id: tenantId || '',
        status: 'paused',
        attempt_number: 1,
        trigger_type: 'api',
        started_at: new Date(Date.now() - 1800000).toISOString(),
        slo_breached: false,
        created_at: new Date(Date.now() - 1800000).toISOString()
      }
    ];
    
    setRuns(mockRuns);
    setLoading(false);
  }, [tenantId]);

  // Auto-refresh
  useEffect(() => {
    if (!autoRefresh) return;
    const interval = setInterval(() => {
      // Would refetch here
    }, 30000);
    return () => clearInterval(interval);
  }, [autoRefresh]);

  const filteredRuns = useMemo(() => {
    let filtered = runs;
    
    // Apply status filter
    if (filter === 'running') {
      filtered = filtered.filter(r => r.status === 'running' || r.status === 'paused');
    } else if (filter === 'completed') {
      filtered = filtered.filter(r => r.status === 'completed');
    } else if (filter === 'failed') {
      filtered = filtered.filter(r => r.status === 'failed' || r.status === 'cancelled');
    } else if (filter === 'exceptions') {
      filtered = filtered.filter(r => r.slo_breached || r.status === 'failed');
    }
    
    // Apply search
    if (searchQuery) {
      filtered = filtered.filter(r => 
        r.job_id.toLowerCase().includes(searchQuery.toLowerCase()) ||
        r.id.toLowerCase().includes(searchQuery.toLowerCase())
      );
    }
    
    return filtered;
  }, [runs, filter, searchQuery]);

  const stats = useMemo(() => ({
    total: runs.length,
    running: runs.filter(r => r.status === 'running').length,
    completed: runs.filter(r => r.status === 'completed').length,
    failed: runs.filter(r => r.status === 'failed').length,
    sloBreaches: runs.filter(r => r.slo_breached).length
  }), [runs]);

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'running': return <PlayCircle className="status-icon running" />;
      case 'completed': return <CheckCircle className="status-icon success" />;
      case 'failed': return <XCircle className="status-icon danger" />;
      case 'cancelled': return <XCircle className="status-icon warning" />;
      case 'paused': return <Pause className="status-icon warning" />;
      default: return <Clock className="status-icon" />;
    }
  };

  if (loading) {
    return <div className="loading-state">Loading runs...</div>;
  }

  return (
    <div className="runs-view">
      {/* Stats Bar */}
      <div className="runs-stats-bar">
        <button 
          className={`stat-button ${filter === 'all' ? 'active' : ''}`}
          onClick={() => setFilter('all')}
        >
          <span className="stat-count">{stats.total}</span>
          <span className="stat-label">Total</span>
        </button>
        <button 
          className={`stat-button running ${filter === 'running' ? 'active' : ''}`}
          onClick={() => setFilter('running')}
        >
          <span className="stat-count">{stats.running}</span>
          <span className="stat-label">Running</span>
        </button>
        <button 
          className={`stat-button success ${filter === 'completed' ? 'active' : ''}`}
          onClick={() => setFilter('completed')}
        >
          <span className="stat-count">{stats.completed}</span>
          <span className="stat-label">Completed</span>
        </button>
        <button 
          className={`stat-button danger ${filter === 'failed' ? 'active' : ''}`}
          onClick={() => setFilter('failed')}
        >
          <span className="stat-count">{stats.failed}</span>
          <span className="stat-label">Failed</span>
        </button>
        <button 
          className={`stat-button warning ${filter === 'exceptions' ? 'active' : ''}`}
          onClick={() => setFilter('exceptions')}
        >
          <AlertTriangle className="stat-icon" />
          <span className="stat-count">{stats.sloBreaches}</span>
          <span className="stat-label">SLO Breaches</span>
        </button>
      </div>

      {/* Toolbar */}
      <div className="toolbar">
        <div className="toolbar-left">
          <div className="search-box">
            <Search className="search-icon" />
            <input
              type="text"
              placeholder="Search by job ID..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
            />
          </div>
        </div>
        
        <div className="toolbar-right">
          <label className="checkbox-filter">
            <input 
              type="checkbox" 
              checked={autoRefresh}
              onChange={(e) => setAutoRefresh(e.target.checked)}
            />
            Auto-refresh
          </label>
          <button className="btn-secondary">
            <RefreshCw /> Refresh
          </button>
        </div>
      </div>

      {/* Runs List */}
      <div className="runs-list">
        {filteredRuns.length === 0 ? (
          <div className="empty-state">
            <Clock className="empty-icon" />
            <h3>No Runs Found</h3>
            <p>No job runs match your current filters</p>
          </div>
        ) : (
          filteredRuns.map((run) => (
            <div 
              key={run.id} 
              className={`run-card ${run.status} ${run.slo_breached ? 'slo-breached' : ''}`}
              onClick={() => setSelectedRun(run)}
            >
              <div className="run-status">
                {getStatusIcon(run.status)}
              </div>
              
              <div className="run-info">
                <div className="run-header">
                  <span className="run-id">Run {run.id.slice(0, 8)}</span>
                  <span className="job-id">Job: {run.job_id}</span>
                </div>
                <div className="run-meta">
                  <span className="trigger-type">{run.trigger_type}</span>
                  <span className="attempt">Attempt #{run.attempt_number}</span>
                  {run.started_at && (
                    <span className="start-time">
                      Started: {formatRelativeTime(new Date(run.started_at))}
                    </span>
                  )}
                </div>
                {run.error_message && (
                  <div className="error-message">
                    <AlertTriangle className="error-icon" />
                    {run.error_message}
                  </div>
                )}
              </div>
              
              <div className="run-metrics">
                {run.duration_ms && (
                  <span className="duration">{formatDuration(run.duration_ms)}</span>
                )}
                {run.slo_breached && (
                  <span className="slo-breach-badge">SLO Breached</span>
                )}
              </div>
              
              <div className="run-actions">
                <button className="action-btn" title="View Details">
                  <Eye />
                </button>
                {run.status === 'failed' && (
                  <button className="action-btn" title="Retry">
                    <RotateCcw />
                  </button>
                )}
              </div>
            </div>
          ))
        )}
      </div>

      {/* Run Detail Modal */}
      {selectedRun && (
        <RunDetailModal 
          run={selectedRun} 
          onClose={() => setSelectedRun(null)} 
        />
      )}
    </div>
  );
};

interface RunDetailModalProps {
  run: JobRun;
  onClose: () => void;
}

const RunDetailModal: React.FC<RunDetailModalProps> = ({ run, onClose }) => {
  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="run-detail-modal" onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <h2>Run Details</h2>
          <button className="close-btn" onClick={onClose}>&times;</button>
        </div>
        
        <div className="modal-content">
          <div className="detail-grid">
            <div className="detail-item">
              <label>Run ID</label>
              <span>{run.id}</span>
            </div>
            <div className="detail-item">
              <label>Job ID</label>
              <span>{run.job_id}</span>
            </div>
            <div className="detail-item">
              <label>Status</label>
              <span className={`status-badge ${run.status}`}>{run.status}</span>
            </div>
            <div className="detail-item">
              <label>Trigger Type</label>
              <span>{run.trigger_type}</span>
            </div>
            <div className="detail-item">
              <label>Attempt</label>
              <span>#{run.attempt_number}</span>
            </div>
            {run.temporal_workflow_id && (
              <div className="detail-item">
                <label>Temporal Workflow</label>
                <span className="code">{run.temporal_workflow_id}</span>
              </div>
            )}
            {run.started_at && (
              <div className="detail-item">
                <label>Started At</label>
                <span>{new Date(run.started_at).toLocaleString()}</span>
              </div>
            )}
            {run.completed_at && (
              <div className="detail-item">
                <label>Completed At</label>
                <span>{new Date(run.completed_at).toLocaleString()}</span>
              </div>
            )}
            {run.duration_ms && (
              <div className="detail-item">
                <label>Duration</label>
                <span>{formatDuration(run.duration_ms)}</span>
              </div>
            )}

          {run.semantic_bindings && (
            <div className="semantic-section">
              <h4>Semantic Context</h4>
              <div className="semantic-bindings-list">
                {run.semantic_bindings.bo_ids && run.semantic_bindings.bo_ids.map(id => (
                  <div key={id} className="semantic-item">
                    <span className="semantic-label bo">BO</span>
                    <span className="semantic-value">{id}</span>
                  </div>
                ))}
                {run.semantic_bindings.api_ids && run.semantic_bindings.api_ids.map(id => (
                  <div key={id} className="semantic-item">
                    <span className="semantic-label api">API</span>
                    <span className="semantic-value">{id}</span>
                  </div>
                ))}
              </div>
            </div>
          )}
          </div>
          
          {run.error_message && (
            <div className="error-section">
              <h4>Error</h4>
              <div className="error-content">{run.error_message}</div>
            </div>
          )}
          
          {run.slo_breached && (
            <div className="slo-section">
              <h4>SLO Information</h4>
              <p>Target: {run.slo_target_ms}ms</p>
              <p>Actual: {run.duration_ms}ms</p>
              <p className="breach-indicator">⚠️ SLO Breached</p>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

function formatRelativeTime(date: Date): string {
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMins = Math.round(diffMs / 60000);
  
  if (diffMins < 1) return 'just now';
  if (diffMins < 60) return `${diffMins}m ago`;
  
  const diffHours = Math.round(diffMins / 60);
  if (diffHours < 24) return `${diffHours}h ago`;
  
  const diffDays = Math.round(diffHours / 24);
  return `${diffDays}d ago`;
}

function formatDuration(ms: number): string {
  if (ms < 1000) return `${ms}ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
  if (ms < 3600000) return `${Math.round(ms / 60000)}m`;
  return `${(ms / 3600000).toFixed(1)}h`;
}

export default RunsExceptionsView;
