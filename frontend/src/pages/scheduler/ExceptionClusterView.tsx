import React, { useState, useEffect } from 'react';
import { useActor } from '../../contexts/ActorContext';
import './SchedulerConsole.css';

interface ExceptionCluster {
  id: string;
  name: string;
  pattern: string;
  category: 'timeout' | 'connectivity' | 'auth' | 'data' | 'resource' | 'code';
  occurrenceCount: number;
  affectedJobs: string[];
  affectedTenants: string[];
  severity: 'critical' | 'high' | 'medium' | 'low';
  trend: 'increasing' | 'stable' | 'decreasing';
  firstSeen: string;
  lastSeen: string;
  rootCauseSummary: string;
  remediation: string[];
}

interface ExceptionClusterViewProps {
  onClusterSelect?: (clusterId: string) => void;
  onApplyFix?: (clusterId: string) => void;
}

/**
 * ExceptionClusterView - Groups similar failures for pattern recognition
 * Shows tenant-scoped clusters for Tenant Ops, cross-tenant for Global Ops
 */
const ExceptionClusterView: React.FC<ExceptionClusterViewProps> = ({ 
  onClusterSelect, 
  onApplyFix 
}) => {
  const { role, tenantId, permissions } = useActor();
  const [clusters, setClusters] = useState<ExceptionCluster[]>([]);
  const [selectedCluster, setSelectedCluster] = useState<ExceptionCluster | null>(null);
  const [filterCategory, setFilterCategory] = useState<string>('all');
  const [filterSeverity, setFilterSeverity] = useState<string>('all');
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchClusters();
  }, [tenantId, role]);

  const fetchClusters = async () => {
    setLoading(true);
    try {
      // Simulated data - would fetch from API
      const data: ExceptionCluster[] = [
        {
          id: 'cls-001',
          name: 'Pre-Agg Timeout Cluster',
          pattern: 'context deadline exceeded after 300s',
          category: 'timeout',
          occurrenceCount: 12,
          affectedJobs: ['EU Pre-Agg', 'APAC Pre-Agg', 'NAM Pre-Agg'],
          affectedTenants: role === 'GLOBAL_OPS' ? ['T-002', 'T-005', 'T-015'] : [tenantId || ''],
          severity: 'high',
          trend: 'increasing',
          firstSeen: '2026-01-15T02:15:00Z',
          lastSeen: '2026-01-17T02:18:00Z',
          rootCauseSummary: 'Database query timeout due to increased data volume',
          remediation: [
            'Increase job timeout from 300s to 600s',
            'Add query optimization for large datasets',
            'Consider partitioning by date range',
          ],
        },
        {
          id: 'cls-002',
          name: 'Integration Auth Failures',
          pattern: 'unauthorized: token expired',
          category: 'auth',
          occurrenceCount: 7,
          affectedJobs: ['Data Sync', 'API Integration'],
          affectedTenants: role === 'GLOBAL_OPS' ? ['T-001', 'T-003', 'T-008', 'T-012'] : [tenantId || ''],
          severity: 'medium',
          trend: 'stable',
          firstSeen: '2026-01-16T14:30:00Z',
          lastSeen: '2026-01-17T08:45:00Z',
          rootCauseSummary: 'OAuth token refresh failing due to rate limiting',
          remediation: [
            'Implement token refresh before expiration',
            'Add retry with exponential backoff',
            'Contact API provider for rate limit increase',
          ],
        },
        {
          id: 'cls-003',
          name: 'Data Quality Anomalies',
          pattern: 'null value in required field: account_id',
          category: 'data',
          occurrenceCount: 5,
          affectedJobs: ['Validation Job', 'Quality Scan'],
          affectedTenants: role === 'GLOBAL_OPS' ? ['T-005'] : [tenantId || ''],
          severity: 'high',
          trend: 'stable',
          firstSeen: '2026-01-17T03:00:00Z',
          lastSeen: '2026-01-17T06:00:00Z',
          rootCauseSummary: 'Upstream data source missing required fields',
          remediation: [
            'Add null check with default value',
            'Create data quality alert for upstream',
            'Implement graceful degradation',
          ],
        },
      ];
      
      // Filter for tenant ops
      if (role === 'TENANT_OPS') {
        setClusters(data.filter(c => c.affectedTenants.includes(tenantId || '')));
      } else {
        setClusters(data);
      }
    } finally {
      setLoading(false);
    }
  };

  const getCategoryIcon = (category: string): string => {
    switch (category) {
      case 'timeout': return '⏱️';
      case 'connectivity': return '🔌';
      case 'auth': return '🔐';
      case 'data': return '📊';
      case 'resource': return '💾';
      case 'code': return '🐛';
      default: return '❓';
    }
  };

  const getSeverityColor = (severity: string): string => {
    switch (severity) {
      case 'critical': return '#dc2626';
      case 'high': return '#ea580c';
      case 'medium': return '#ca8a04';
      case 'low': return '#16a34a';
      default: return '#6b7280';
    }
  };

  const getTrendIcon = (trend: string): string => {
    switch (trend) {
      case 'increasing': return '📈';
      case 'decreasing': return '📉';
      default: return '➡️';
    }
  };

  const handleClusterClick = (cluster: ExceptionCluster) => {
    setSelectedCluster(cluster);
    onClusterSelect?.(cluster.id);
  };

  const handleApplyFix = (clusterId: string) => {
    onApplyFix?.(clusterId);
  };

  const filteredClusters = clusters
    .filter(c => filterCategory === 'all' || c.category === filterCategory)
    .filter(c => filterSeverity === 'all' || c.severity === filterSeverity)
    .sort((a, b) => {
      const severityOrder = { critical: 0, high: 1, medium: 2, low: 3 };
      return severityOrder[a.severity] - severityOrder[b.severity];
    });

  const categories = ['all', ...Array.from(new Set(clusters.map(c => c.category)))];
  const severities = ['all', 'critical', 'high', 'medium', 'low'];

  if (loading) {
    return <div className="clusters-loading">Analyzing exception patterns...</div>;
  }

  return (
    <div className="exception-cluster-view">
      {/* Header */}
      <div className="cluster-header">
        <h3>
          <span className="header-icon">🔍</span>
          Exception Clusters
          {role === 'GLOBAL_OPS' && <span className="header-badge">Cross-Tenant</span>}
        </h3>
        <div className="cluster-filters">
          <select value={filterCategory} onChange={e => setFilterCategory(e.target.value)}>
            {categories.map(c => (
              <option key={c} value={c}>{c === 'all' ? 'All Categories' : c}</option>
            ))}
          </select>
          <select value={filterSeverity} onChange={e => setFilterSeverity(e.target.value)}>
            {severities.map(s => (
              <option key={s} value={s}>{s === 'all' ? 'All Severities' : s}</option>
            ))}
          </select>
        </div>
      </div>

      <div className="cluster-content">
        {/* Cluster List */}
        <div className="cluster-list">
          {filteredClusters.length === 0 ? (
            <div className="empty-state">
              <span className="empty-icon">✅</span>
              <p>No exception clusters detected</p>
            </div>
          ) : (
            filteredClusters.map(cluster => (
              <div
                key={cluster.id}
                className={`cluster-card ${selectedCluster?.id === cluster.id ? 'selected' : ''}`}
                onClick={() => handleClusterClick(cluster)}
              >
                <div className="cluster-icon">{getCategoryIcon(cluster.category)}</div>
                <div className="cluster-info">
                  <h4>{cluster.name}</h4>
                  <p className="cluster-pattern">{cluster.pattern}</p>
                  <div className="cluster-meta">
                    <span className="occurrence-count">{cluster.occurrenceCount} occurrences</span>
                    <span className="affected-jobs">{cluster.affectedJobs.length} jobs</span>
                    {permissions.canViewCrossTenant && (
                      <span className="affected-tenants">{cluster.affectedTenants.length} tenants</span>
                    )}
                  </div>
                </div>
                <div className="cluster-indicators">
                  <span 
                    className="severity-badge"
                    style={{ backgroundColor: getSeverityColor(cluster.severity) }}
                  >
                    {cluster.severity}
                  </span>
                  <span className="trend-icon">{getTrendIcon(cluster.trend)}</span>
                </div>
              </div>
            ))
          )}
        </div>

        {/* Cluster Detail Panel */}
        {selectedCluster && (
          <div className="cluster-detail">
            <div className="detail-header">
              <h3>{selectedCluster.name}</h3>
              <button className="close-btn" onClick={() => setSelectedCluster(null)}>×</button>
            </div>

            <div className="detail-section">
              <h4>Error Pattern</h4>
              <code className="error-pattern">{selectedCluster.pattern}</code>
            </div>

            <div className="detail-section">
              <h4>Root Cause Analysis</h4>
              <p className="root-cause">{selectedCluster.rootCauseSummary}</p>
            </div>

            <div className="detail-section">
              <h4>Affected Jobs</h4>
              <div className="affected-list">
                {selectedCluster.affectedJobs.map(job => (
                  <span key={job} className="affected-item">{job}</span>
                ))}
              </div>
            </div>

            {permissions.canViewCrossTenant && (
              <div className="detail-section">
                <h4>Affected Tenants</h4>
                <div className="affected-list">
                  {selectedCluster.affectedTenants.map(tenant => (
                    <span key={tenant} className="affected-item tenant">{tenant}</span>
                  ))}
                </div>
              </div>
            )}

            <div className="detail-section">
              <h4>Recommended Remediation</h4>
              <ul className="remediation-list">
                {selectedCluster.remediation.map((step, i) => (
                  <li key={i}>{step}</li>
                ))}
              </ul>
            </div>

            <div className="detail-actions">
              <button 
                className="action-btn primary"
                onClick={() => handleApplyFix(selectedCluster.id)}
              >
                Apply Fix → ChangeSet
              </button>
              <button className="action-btn secondary">View Jobs</button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
};

export default ExceptionClusterView;
