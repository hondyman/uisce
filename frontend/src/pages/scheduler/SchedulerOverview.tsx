/**
 * Scheduler Overview Dashboard
 * Displays key metrics, job status distribution, and recent activity
 */

import React, { useMemo } from 'react';
import { 
  CheckCircle, 
  XCircle, 
  Clock, 
  AlertTriangle,
  TrendingUp,
  Calendar,
  Zap,
  Shield
} from 'lucide-react';
import { useTenantContext } from '../../hooks/useTenantContext';
import { useSchedulerStats, useJobs, Job } from '../../api/schedulerApi';

const SchedulerOverview: React.FC = () => {
  const { selectedTenant } = useTenantContext();
  const tenantId = selectedTenant?.id;
  const { stats, loading: statsLoading } = useSchedulerStats(tenantId || '');
  const { jobs, loading: jobsLoading } = useJobs(tenantId || '', { limit: 10 });

  const categoryBreakdown = useMemo(() => {
    if (!jobs) return [];
    const counts: Record<string, number> = {};
    jobs.forEach((job: Job) => {
      counts[job.category] = (counts[job.category] || 0) + 1;
    });
    return Object.entries(counts).map(([category, count]) => ({ category, count }));
  }, [jobs]);

  const upcomingJobs = useMemo(() => {
    if (!jobs) return [];
    return jobs
      .filter((job: Job) => job.next_run_at && job.is_active)
      .sort((a: Job, b: Job) => {
        if (!a.next_run_at || !b.next_run_at) return 0;
        return new Date(a.next_run_at).getTime() - new Date(b.next_run_at).getTime();
      })
      .slice(0, 5);
  }, [jobs]);

  if (statsLoading || jobsLoading) {
    return <div className="loading-state">Loading overview...</div>;
  }

  return (
    <div className="scheduler-overview">
      {/* Stats Cards */}
      <section className="stats-grid">
        <div className="stat-card primary">
          <div className="stat-icon-wrapper">
            <Calendar />
          </div>
          <div className="stat-content">
            <span className="stat-number">{stats?.total_jobs || 0}</span>
            <span className="stat-title">Total Jobs</span>
          </div>
        </div>

        <div className="stat-card success">
          <div className="stat-icon-wrapper">
            <CheckCircle />
          </div>
          <div className="stat-content">
            <span className="stat-number">{stats?.succeeded_last_24h || 0}</span>
            <span className="stat-title">Succeeded (24h)</span>
          </div>
        </div>

        <div className="stat-card danger">
          <div className="stat-icon-wrapper">
            <XCircle />
          </div>
          <div className="stat-content">
            <span className="stat-number">{stats?.failed_last_24h || 0}</span>
            <span className="stat-title">Failed (24h)</span>
          </div>
        </div>


        <div className="stat-card warning">
          <div className="stat-icon-wrapper">
            <Shield />
          </div>
          <div className="stat-content">
            <span className="stat-number">{stats?.slo_critical_jobs || 0}</span>
            <span className="stat-title">SLO Critical</span>
          </div>
        </div>

        <div className={`stat-card ${stats?.slo_breached_jobs ? 'danger' : 'success'}`}>
          <div className="stat-icon-wrapper">
            <AlertTriangle />
          </div>
          <div className="stat-content">
            <span className="stat-number">{stats?.slo_breached_jobs || 0}</span>
            <span className="stat-title">SLO Breaches</span>
          </div>
        </div>

        <div className="stat-card primary">
          <div className="stat-icon-wrapper">
            <Zap />
          </div>
          <div className="stat-content">
             {/* Show remaining budget roughly */}
            <span className="stat-number">{100 - (stats?.error_budget_consumed || 0)}%</span> 
            <span className="stat-title">Error Budget</span>
          </div>
        </div>
      </section>

      {/* Two Column Layout */}
      <div className="overview-grid">
        {/* Upcoming Jobs */}
        <section className="overview-section">
          <div className="section-header">
            <h3><Clock className="section-icon" /> Upcoming Jobs</h3>
          </div>
          <div className="jobs-list">
            {upcomingJobs.length === 0 ? (
              <div className="empty-list">No scheduled jobs</div>
            ) : (
              upcomingJobs.map((job: Job) => (
                <div key={job.id} className="job-item">
                  <div className="job-info">
                    <span className="job-name">{job.name}</span>
                    <span className="job-category">{job.category}</span>
                  </div>
                  <div className="job-meta">
                    <span className="job-time">
                      {job.next_run_at && formatRelativeTime(new Date(job.next_run_at))}
                    </span>
                    {job.slo_critical && <span className="slo-badge">SLO</span>}
                  </div>
                </div>
              ))
            )}
          </div>
        </section>

        {/* Category Breakdown */}
        <section className="overview-section">
          <div className="section-header">
            <h3><TrendingUp className="section-icon" /> Jobs by Category</h3>
          </div>
          <div className="category-chart">
            {categoryBreakdown.length === 0 ? (
              <div className="empty-list">No categories</div>
            ) : (
              categoryBreakdown.map(({ category, count }) => (
                <div key={category} className="category-bar">
                  <div className="category-label">
                    <span className="category-name">{category}</span>
                    <span className="category-count">{count}</span>
                  </div>
                  <div className="bar-container">
                    <div 
                      className="bar-fill" 
                      style={{ width: `${(count / (stats?.total_jobs || 1)) * 100}%` }}
                    />
                  </div>
                </div>
              ))
            )}
          </div>
        </section>

        {/* Running Jobs */}
        <section className="overview-section wide">
          <div className="section-header">
            <h3><Zap className="section-icon" /> Currently Running</h3>
            <span className="running-count">{stats?.running_jobs || 0} active</span>
          </div>
          {stats?.running_jobs === 0 ? (
            <div className="empty-list">No jobs currently running</div>
          ) : (
            <div className="running-indicator">
              <div className="pulse-animation" />
              <span>{stats?.running_jobs} job(s) in progress</span>
            </div>
          )}
        </section>
      </div>
    </div>
  );
};

// Helper function
function formatRelativeTime(date: Date): string {
  const now = new Date();
  const diffMs = date.getTime() - now.getTime();
  const diffMins = Math.round(diffMs / 60000);
  
  if (diffMins < 0) return 'Overdue';
  if (diffMins < 1) return 'Now';
  if (diffMins < 60) return `in ${diffMins}m`;
  
  const diffHours = Math.round(diffMins / 60);
  if (diffHours < 24) return `in ${diffHours}h`;
  
  const diffDays = Math.round(diffHours / 24);
  return `in ${diffDays}d`;
}

export default SchedulerOverview;
