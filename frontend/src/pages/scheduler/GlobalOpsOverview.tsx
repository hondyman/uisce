import React, { useState, useEffect } from 'react';
import { useActor } from '../../contexts/ActorContext';
import TenantHeatmap from './TenantHeatmap';
import './SchedulerConsole.css';

interface GlobalStats {
  totalJobs: number;
  succeeded: number;
  failed: number;
  sloCritical: number;
  activeTenants: number;
}

interface CategoryBreakdown {
  category: string;
  count: number;
  successRate: number;
}

interface CriticalJob {
  id: string;
  name: string;
  tenant: string;
  nextRun: string;
  type: string;
  riskScore: number;
  sloCritical: boolean;
}

interface GlobalAIInsight {
  id: string;
  type: 'cross_tenant' | 'optimization' | 'alert' | 'consolidation';
  title: string;
  description: string;
  affectedTenants: string[];
  impact: string;
  action?: string;
}

/**
 * GlobalOpsOverview - Cross-tenant dashboard for Global Ops users
 * Shows global status, tenant heatmap, category breakdown, and cross-tenant AI insights
 */
const GlobalOpsOverview: React.FC = () => {
  const { permissions } = useActor();
  const [stats, setStats] = useState<GlobalStats | null>(null);
  const [categories, setCategories] = useState<CategoryBreakdown[]>([]);
  const [criticalJobs, setCriticalJobs] = useState<CriticalJob[]>([]);
  const [aiInsights, setAIInsights] = useState<GlobalAIInsight[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchGlobalData();
  }, []);

  const fetchGlobalData = async () => {
    setLoading(true);
    try {
      // Simulated data - would fetch from API
      setStats({
        totalJobs: 1204,
        succeeded: 1172,
        failed: 32,
        sloCritical: 7,
        activeTenants: 24,
      });

      setCategories([
        { category: 'Pre-Aggregate', count: 142, successRate: 94.5 },
        { category: 'Reports', count: 89, successRate: 98.2 },
        { category: 'Integrations', count: 67, successRate: 91.0 },
        { category: 'Data Quality', count: 45, successRate: 99.5 },
        { category: 'Compliance', count: 23, successRate: 100 },
      ]);

      setCriticalJobs([
        { id: '1', name: 'EU Positions Pre-Agg', tenant: 'Global', nextRun: '2:00 AM', type: 'pre-agg', riskScore: 0.4, sloCritical: true },
        { id: '2', name: 'Risk Batch DAG', tenant: 'T-002', nextRun: '2:15 AM', type: 'batch', riskScore: 0.3, sloCritical: true },
        { id: '3', name: 'Client NAV Calculation', tenant: 'T-005', nextRun: '2:30 AM', type: 'calculation', riskScore: 0.5, sloCritical: true },
        { id: '4', name: 'Regulatory Report', tenant: 'Global', nextRun: '6:00 AM', type: 'report', riskScore: 0.2, sloCritical: true },
      ]);

      setAIInsights([
        {
          id: '1',
          type: 'cross_tenant',
          title: '3 DAGs failing at Step 4 across 2 tenants',
          description: 'Common pattern detected: timeout on external API call',
          affectedTenants: ['T-002', 'T-005'],
          impact: '4 downstream jobs blocked',
          action: 'View Pattern',
        },
        {
          id: '2',
          type: 'cross_tenant',
          title: 'Integration X failing for 4 tenants',
          description: 'Authentication errors suggest token rotation needed',
          affectedTenants: ['T-001', 'T-003', 'T-008', 'T-012'],
          impact: '12 jobs affected',
          action: 'Rotate Tokens',
        },
        {
          id: '3',
          type: 'consolidation',
          title: 'Suggest consolidating 2 global DAGs',
          description: 'Pre-Agg pipelines share 80% of steps',
          affectedTenants: ['All'],
          impact: '20% resource reduction',
          action: 'View Comparison',
        },
        {
          id: '4',
          type: 'optimization',
          title: 'Stagger heavy jobs across regions',
          description: 'Peak load at 2:00 AM causing contention',
          affectedTenants: ['T-001', 'T-002', 'T-003'],
          impact: '35% latency improvement',
          action: 'Apply Schedule',
        },
      ]);
    } finally {
      setLoading(false);
    }
  };

  const getInsightIcon = (type: string) => {
    switch (type) {
      case 'cross_tenant': return '🌐';
      case 'optimization': return '⚡';
      case 'alert': return '🚨';
      case 'consolidation': return '🔗';
      default: return '💡';
    }
  };

  if (loading) {
    return (
      <div className="overview-loading">
        <div className="skeleton-header" />
        <div className="skeleton-stats" />
        <div className="skeleton-heatmap" />
      </div>
    );
  }

  return (
    <div className="global-ops-overview">
      {/* Global Status Strip */}
      <div className="status-strip global-status">
        <div className="status-item total">
          <span className="value">{stats?.totalJobs.toLocaleString()}</span>
          <span className="label">Total Jobs</span>
        </div>
        <div className="status-item success">
          <span className="value">{stats?.succeeded.toLocaleString()}</span>
          <span className="label">Succeeded</span>
        </div>
        <div className="status-item failed">
          <span className="value">{stats?.failed}</span>
          <span className="label">Failed</span>
        </div>
        <div className="status-item slo-critical">
          <span className="value">{stats?.sloCritical}</span>
          <span className="label">SLO-Critical</span>
        </div>
        <div className="status-item tenants">
          <span className="value">{stats?.activeTenants}</span>
          <span className="label">Active Tenants</span>
        </div>
      </div>

      {/* Main Content Grid */}
      <div className="overview-grid global-grid">
        {/* Tenant Heatmap */}
        <div className="overview-card heatmap-card full-width">
          <h3>Tenant Health Heatmap</h3>
          <TenantHeatmap />
        </div>

        {/* Category Breakdown */}
        <div className="overview-card category-card">
          <h3>Category Breakdown</h3>
          <div className="category-list">
            {categories.map(cat => (
              <div key={cat.category} className="category-item">
                <div className="category-info">
                  <span className="category-name">{cat.category}</span>
                  <span className="category-count">{cat.count} jobs</span>
                </div>
                <div className="category-bar">
                  <div 
                    className="category-fill"
                    style={{ 
                      width: `${cat.successRate}%`,
                      backgroundColor: cat.successRate >= 95 ? '#4ade80' : 
                                      cat.successRate >= 90 ? '#fbbf24' : '#ef4444'
                    }}
                  />
                </div>
                <span className="category-rate">{cat.successRate}%</span>
              </div>
            ))}
          </div>
        </div>

        {/* Critical Jobs */}
        <div className="overview-card critical-jobs-card">
          <h3>
            <span className="critical-icon">⚡</span>
            Upcoming SLO-Critical Jobs
          </h3>
          <div className="critical-list">
            {criticalJobs.map(job => (
              <div key={job.id} className="critical-item">
                <div className="job-time">{job.nextRun}</div>
                <div className="job-info">
                  <span className="job-name">{job.name}</span>
                  <div className="job-meta">
                    <span className="job-tenant">{job.tenant}</span>
                    <span className={`job-type type-${job.type}`}>{job.type}</span>
                  </div>
                </div>
                <div className={`risk-indicator risk-${job.riskScore > 0.3 ? 'high' : 'medium'}`}>
                  {Math.round(job.riskScore * 100)}%
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* Global AI Insights */}
        <div className="overview-card ai-insights-card global-ai full-width">
          <h3>
            <span className="ai-icon">🤖</span>
            Cross-Tenant AI Insights
          </h3>
          <div className="insights-grid">
            {aiInsights.map(insight => (
              <div key={insight.id} className={`insight-card insight-${insight.type}`}>
                <div className="insight-header">
                  <span className="insight-icon">{getInsightIcon(insight.type)}</span>
                  <span className="insight-type">{insight.type.replace('_', ' ')}</span>
                </div>
                <h4>{insight.title}</h4>
                <p>{insight.description}</p>
                <div className="insight-meta">
                  <span className="affected-tenants">
                    {insight.affectedTenants.length === 1 && insight.affectedTenants[0] === 'All' 
                      ? 'All tenants' 
                      : `${insight.affectedTenants.length} tenants`}
                  </span>
                  <span className="impact">{insight.impact}</span>
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

export default GlobalOpsOverview;
