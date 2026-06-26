/**
 * Jobs View - List and manage scheduled jobs
 */

import React, { useState, useMemo } from 'react';
import { 
  Plus, 
  Search, 
  Filter, 
  Play, 
  Pause, 
  Trash2, 
  Edit2,
  Clock,
  CheckCircle,
  XCircle,
  AlertTriangle,
  ExternalLink
} from 'lucide-react';
import { useTenantContext } from '../../hooks/useTenantContext';
import { useJobs, Job, triggerJob, deleteJob, updateJob, getJobRuns, JobRun } from '../../api/schedulerApi';
import JobRunTimeline from '../../components/scheduler/JobRunTimeline';
import SemanticBindingsList from '../../components/scheduler/SemanticBindingsList';

const JobsView: React.FC = () => {
  const { selectedTenant } = useTenantContext();
  const tenantId = selectedTenant?.id;
  const [searchQuery, setSearchQuery] = useState('');
  const [categoryFilter, setCategoryFilter] = useState<string>('all');
  const [statusFilter, setStatusFilter] = useState<string>('all');
  const [selectedJob, setSelectedJob] = useState<Job | null>(null);
  
  const { jobs, loading, error, refetch } = useJobs(tenantId || '', { limit: 100 });

  const categories = useMemo(() => {
    if (!jobs) return [];
    const cats = new Set(jobs.map((j: Job) => j.category));
    return Array.from(cats);
  }, [jobs]);

  const filteredJobs = useMemo(() => {
    if (!jobs) return [];
    return jobs.filter((job: Job) => {
      const matchesSearch = job.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
        job.description?.toLowerCase().includes(searchQuery.toLowerCase());
      const matchesCategory = categoryFilter === 'all' || job.category === categoryFilter;
      const matchesStatus = statusFilter === 'all' || 
        (statusFilter === 'active' && job.is_active) ||
        (statusFilter === 'inactive' && !job.is_active);
      return matchesSearch && matchesCategory && matchesStatus;
    });
  }, [jobs, searchQuery, categoryFilter, statusFilter]);

  const handleTrigger = async (job: Job) => {
    try {
      await triggerJob(job.id);
      refetch();
    } catch (err) {
      console.error('Failed to trigger job:', err);
    }
  };

  const handleToggleActive = async (job: Job) => {
    try {
      await updateJob(job.id, { is_active: !job.is_active });
      refetch();
    } catch (err) {
      console.error('Failed to update job:', err);
    }
  };

  const handleDelete = async (job: Job) => {
    if (!confirm(`Delete job "${job.name}"?`)) return;
    try {
      await deleteJob(job.id);
      refetch();
    } catch (err) {
      console.error('Failed to delete job:', err);
    }
  };

  if (loading) {
    return <div className="loading-state">Loading jobs...</div>;
  }

  if (error) {
    return <div className="error-state">Error loading jobs: {error.message}</div>;
  }

  return (
    <div className="jobs-view">
      {/* Toolbar */}
      <div className="toolbar">
        <div className="toolbar-left">
          <div className="search-box">
            <Search className="search-icon" />
            <input
              type="text"
              placeholder="Search jobs..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
            />
          </div>
          
          <select 
            aria-label="Filter by category"
            title="Filter by category"
            value={categoryFilter} 
            onChange={(e) => setCategoryFilter(e.target.value)}
            className="filter-select"
          >
            <option value="all">All Categories</option>
            {categories.map((cat) => (
              <option key={cat} value={cat}>{cat}</option>
            ))}
          </select>
          
          <select 
            aria-label="Filter by status"
            title="Filter by status"
            value={statusFilter} 
            onChange={(e) => setStatusFilter(e.target.value)}
            className="filter-select"
          >
            <option value="all">All Status</option>
            <option value="active">Active</option>
            <option value="inactive">Inactive</option>
          </select>
        </div>
        
        <div className="toolbar-right">
          <button className="btn-primary">
            <Plus /> New Job
          </button>
        </div>
      </div>

      {/* Jobs Table */}
      <div className="jobs-table-container">
        <table className="jobs-table">
          <thead>
            <tr>
              <th>Name</th>
              <th>Category</th>
              <th>Schedule</th>
              <th>Next Run</th>
              <th>Last Run</th>
              <th>Status</th>
              <th>SLO</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            {filteredJobs.length === 0 ? (
              <tr>
                <td colSpan={8} className="empty-row">No jobs found</td>
              </tr>
            ) : (
              filteredJobs.map((job: Job) => (
                <tr key={job.id} className={!job.is_active ? 'inactive' : ''}>
                  <td>
                    <div className="job-name-cell">
                      <span className="job-name" onClick={() => setSelectedJob(job)}>
                        {job.name}
                      </span>
                      {job.description && (
                        <span className="job-desc">{job.description}</span>
                      )}
                    </div>
                  </td>
                  <td>
                    <span className={`category-badge ${job.category}`}>
                      {job.category}
                    </span>
                  </td>
                  <td>
                    <span className="schedule-type">{job.schedule_type}</span>
                    {job.cron_expression && (
                      <span className="cron-expr">{job.cron_expression}</span>
                    )}
                  </td>
                  <td>
                    {job.next_run_at ? (
                      <span className="time-cell">
                        <Clock className="time-icon" />
                        {formatDateTime(job.next_run_at)}
                      </span>
                    ) : '-'}
                  </td>
                  <td>
                    {job.last_run_at ? formatDateTime(job.last_run_at) : 'Never'}
                  </td>
                  <td>
                    <span className={`status-badge ${job.is_active ? 'active' : 'inactive'}`}>
                      {job.is_active ? 'Active' : 'Inactive'}
                    </span>
                  </td>
                  <td>
                    {job.slo_critical ? (
                      <span className="slo-badge critical">SLO</span>
                    ) : (
                      <span className="slo-badge">-</span>
                    )}
                  </td>
                  <td>
                    <div className="action-buttons">
                      <button 
                        className="action-btn" 
                        title="Run Now"
                        onClick={() => handleTrigger(job)}
                      >
                        <Play />
                      </button>
                      <button 
                        className="action-btn"
                        title={job.is_active ? 'Pause' : 'Resume'}
                        onClick={() => handleToggleActive(job)}
                      >
                        <Pause />
                      </button>
                      <button 
                        className="action-btn"
                        title="Edit"
                      >
                        <Edit2 />
                      </button>
                      <button 
                        className="action-btn danger"
                        title="Delete"
                        onClick={() => handleDelete(job)}
                      >
                        <Trash2 />
                      </button>
                    </div>
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      {/* Job Detail Sidebar */}
      {selectedJob && (
        <JobDetailSidebar 
          job={selectedJob} 
          onClose={() => setSelectedJob(null)} 
        />
      )}
    </div>
  );
};

interface JobDetailSidebarProps {
  job: Job;
  onClose: () => void;
}

const JobDetailSidebar: React.FC<JobDetailSidebarProps> = ({ job, onClose }) => {
  const [runs, setRuns] = useState<JobRun[]>([]);
  const [loadingRuns, setLoadingRuns] = useState(false);

  React.useEffect(() => {
    const fetchRuns = async () => {
      setLoadingRuns(true);
      try {
        const data = await getJobRuns(job.id, 10);
        setRuns(data);
      } catch (e) {
        console.error("Failed to fetch runs", e);
      } finally {
        setLoadingRuns(false);
      }
    };
    if (job.id) {
      fetchRuns();
    }
  }, [job.id]);

  return (
    <div className="job-detail-sidebar">
      <div className="sidebar-header">
        <h3>{job.name}</h3>
        <button className="close-btn" onClick={onClose}>&times;</button>
      </div>
      
      <div className="sidebar-content">
        <div className="detail-section">
          <h4>Details</h4>
          <dl>
            <dt>Category</dt>
            <dd>{job.category}</dd>
            <dt>Type</dt>
            <dd>{job.job_type}</dd>
            <dt>Schedule</dt>
            <dd>{job.schedule_type} {job.cron_expression && `(${job.cron_expression})`}</dd>
            <dt>Timezone</dt>
            <dd>{job.timezone}</dd>
            <dt>Timeout</dt>
            <dd>{job.timeout_seconds}s</dd>
            <dt>Priority</dt>
            <dd>{job.priority}</dd>
          </dl>
        </div>

        <div className="detail-section">
          <JobRunTimeline runs={runs} loading={loadingRuns} />
        </div>
        
        <div className="detail-section">
          {job.slo_critical && (
            <div className="slo-alert">
              <span className="slo-badge critical">SLO Critical</span>
              <a href={`/intelligence/slo?job=${job.id}`} className="slo-link">
                View Forecast <ExternalLink size={12} />
              </a>
            </div>
          )}
          {(() => {
            // Determine cold BOs from compliance tags
            // In a real implementation this would come from a more specific API field or the bindings themselves
            const isColdStorage = job.compliance_tags?.some(t => t.includes('TIER:COLD') || t.includes('STORAGE:COLD'));
            const coldBOs = isColdStorage ? job.semantic_bindings?.bo_ids || [] : [];
            return <SemanticBindingsList bindings={job.semantic_bindings} coldBOs={coldBOs} />;
          })()}
        </div>
        
        {/* Compliance Section */}
        {job.compliance && (
          <div className="detail-section">
            <h4>Compliance Governance</h4>
            <div className="compliance-grid" style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: '10px' }}>
              <div className={`compliance-card ${job.compliance.pii ? 'danger' : 'success'}`} style={{ padding: '8px', border: '1px solid #eee', borderRadius: '4px', textAlign: 'center' }}>
                <span className="label" style={{ display: 'block', fontSize: '10px', color: '#666' }}>PII DATA</span>
                <span className="value" style={{ fontWeight: 600, color: job.compliance.pii ? '#d32f2f' : '#388e3c' }}>
                  {job.compliance.pii ? 'DETECTED' : 'NONE'}
                </span>
              </div>
              <div className="compliance-card" style={{ padding: '8px', border: '1px solid #eee', borderRadius: '4px', textAlign: 'center' }}>
                <span className="label" style={{ display: 'block', fontSize: '10px', color: '#666' }}>RESIDENCY</span>
                <span className="value" style={{ fontWeight: 600 }}>{job.compliance.residency || 'GLOBAL'}</span>
              </div>
              <div className="compliance-card" style={{ padding: '8px', border: '1px solid #eee', borderRadius: '4px', textAlign: 'center' }}>
                <span className="label" style={{ display: 'block', fontSize: '10px', color: '#666' }}>SENSITIVITY</span>
                <span className="value" style={{ fontWeight: 600 }}>{job.compliance.sensitivity || 'LOW'}</span>
              </div>
            </div>
          </div>
        )}
        
        {job.compliance_tags && job.compliance_tags.length > 0 && (
          <div className="detail-section">
            <h4>Compliance Tags</h4>
            <div className="tags-list">
              {job.compliance_tags.map((tag) => (
                <span key={tag} className="compliance-tag">{tag}</span>
              ))}
            </div>
          </div>
        )}

      </div>
    </div>
  );
};

function formatDateTime(isoString: string): string {
  const date = new Date(isoString);
  return date.toLocaleString(undefined, {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit'
  });
}

export default JobsView;
