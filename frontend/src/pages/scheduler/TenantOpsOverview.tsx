import React, { useState, useEffect } from 'react';
import { useActor } from '../../contexts/ActorContext';
import './SchedulerConsole.css';

interface TenantStats {
  totalJobs: number;
  succeeded: number;
  failed: number;
  sloBreaches: number;
  successRate: number;
  avgDuration: number;
}

interface UpcomingJob {
  id: string;
  name: string;
  nextRun: string;
  type: string;
  riskScore: number;
}

interface AIInsight {
  id: string;
  type: 'warning' | 'suggestion' | 'info';
  title: string;
  description: string;
  jobId?: string;
  action?: string;
}

/**
 * TenantOpsOverview - Single-tenant dashboard for Tenant Ops users
 * Shows tenant-specific job health, upcoming jobs, and AI insights
 */
const TenantOpsOverview: React.FC = () => {
  const { tenantId, tenantName } = useActor();
  const [stats, setStats] = useState<TenantStats | null>(null);
  const [upcomingJobs, setUpcomingJobs] = useState<UpcomingJob[]>([]);
  const [aiInsights, setAIInsights] = useState<AIInsight[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchTenantData();
  }, [tenantId]);

  const fetchTenantData = async () => {
    setLoading(true);
    try {
      // Simulated data - would fetch from API
      setStats({
        totalJobs: 42,
        succeeded: 39,
        failed: 3,
        sloBreaches: 1,
        successRate: 92.8,
        avgDuration: 1.8,
      });

      setUpcomingJobs([
        { id: '1', name: 'Positions Pre-Agg Refresh', nextRun: '2:00 AM', type: 'pre-agg', riskScore: 0.2 },
        { id: '2', name: 'Risk Batch', nextRun: '2:15 AM', type: 'batch', riskScore: 0.1 },
        { id: '3', name: 'Data Quality Scan', nextRun: '3:00 AM', type: 'data_quality', riskScore: 0.05 },
        { id: '4', name: 'Client Report Generation', nextRun: '6:00 AM', type: 'report', riskScore: 0.3 },
      ]);

      setAIInsights([
        {
          id: '1',
          type: 'warning',
          title: 'Pre-Agg job close to SLO threshold',
          description: 'Consider scheduling 15 minutes earlier to reduce deadline pressure',
          jobId: '1',
          action: 'Reschedule',
        },
        {
          id: '2',
          type: 'warning',
          title: 'Integration X token expiring',
          description: 'Token refresh needed within 48 hours',
          action: 'Refresh Token',
        },
        {
          id: '3',
          type: 'suggestion',
          title: 'Parallelize data load steps',
          description: 'Could reduce DAG execution time by 30%',
          action: 'View DAG',
        },
      ]);
    } finally {
      setLoading(false);
    }
  };

  if (loading) {
    return (
      <div className="overview-loading">
        <div className="skeleton-header" />
        <div className="skeleton-stats" />
        <div className="skeleton-list" />
      </div>
    );
  }

  return (
    <div className="tenant-ops-overview">
      {/* Tenant Header */}
      <div className="tenant-header">
        <h2>Tenant: {tenantName || 'Unknown'}</h2>
        <span className="tenant-id">{tenantId}</span>
      </div>

      {/* Status Strip */}
      <div className="status-strip tenant-status">
        <div className="status-item success">
          <span className="value">{stats?.succeeded}</span>
          <span className="label">Succeeded</span>
        </div>
        <div className="status-item failed">
          <span className="value">{stats?.failed}</span>
          <span className="label">Failed</span>
        </div>
        <div className="status-item slo">
          <span className="value">{stats?.sloBreaches}</span>
          <span className="label">SLO Breaches</span>
        </div>
        <div className="status-item rate">
          <span className="value">{stats?.successRate}%</span>
          <span className="label">Success Rate</span>
        </div>
      </div>

      {/* Main Content Grid */}
      <div className="overview-grid tenant-grid">
        {/* Job Health Card */}
        <div className="overview-card job-health-card">
          <h3>Job Health</h3>
          <div className="health-metrics">
            <div className="metric">
              <span className="metric-value">{stats?.totalJobs}</span>
              <span className="metric-label">Total Jobs</span>
            </div>
            <div className="metric">
              <span className="metric-value">{stats?.avgDuration}s</span>
              <span className="metric-label">Avg Duration</span>
            </div>
          </div>
          <div className="health-bar">
            <div 
              className="health-fill" 
              style={{ width: `${stats?.successRate}%` }}
            />
          </div>
        </div>

        {/* Upcoming Jobs Card */}
        <div className="overview-card upcoming-jobs-card">
          <h3>Upcoming Jobs</h3>
          <div className="upcoming-list">
            {upcomingJobs.map(job => (
              <div key={job.id} className="upcoming-item">
                <div className="job-time">{job.nextRun}</div>
                <div className="job-info">
                  <span className="job-name">{job.name}</span>
                  <span className={`job-type type-${job.type}`}>{job.type}</span>
                </div>
                {job.riskScore > 0.2 && (
                  <span className="risk-badge">⚠️ At Risk</span>
                )}
              </div>
            ))}
          </div>
        </div>

        {/* AI Insights Card */}
        <div className="overview-card ai-insights-card">
          <h3>
            <span className="ai-icon">🤖</span>
            AI Insights
          </h3>
          <div className="insights-list">
            {aiInsights.map(insight => (
              <div key={insight.id} className={`insight-item insight-${insight.type}`}>
                <div className="insight-content">
                  <span className="insight-icon">
                    {insight.type === 'warning' ? '⚠️' : insight.type === 'suggestion' ? '💡' : 'ℹ️'}
                  </span>
                  <div className="insight-text">
                    <strong>{insight.title}</strong>
                    <p>{insight.description}</p>
                  </div>
                </div>
                {insight.action && (
                  <button className="insight-action">{insight.action}</button>
                )}
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
};

export default TenantOpsOverview;
