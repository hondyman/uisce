import React, { useState, useEffect } from 'react';
import { useActor } from '../../contexts/ActorContext';
import './SchedulerConsole.css';

interface TenantHealth {
  id: string;
  name: string;
  region: string;
  jobCount: number;
  successRate: number;
  failedJobs: number;
  sloBreaches: number;
  riskScore: number;
  trend: 'up' | 'down' | 'stable';
}

interface TenantHeatmapProps {
  onTenantSelect?: (tenantId: string) => void;
}

/**
 * TenantHeatmap - Visual representation of cross-tenant health
 * Only visible to Global Ops users
 */
const TenantHeatmap: React.FC<TenantHeatmapProps> = ({ onTenantSelect }) => {
  const { permissions } = useActor();
  const [tenants, setTenants] = useState<TenantHealth[]>([]);
  const [selectedTenant, setSelectedTenant] = useState<string | null>(null);
  const [sortBy, setSortBy] = useState<'name' | 'successRate' | 'riskScore'>('riskScore');
  const [filterRegion, setFilterRegion] = useState<string>('all');
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchTenantHealth();
  }, []);

  const fetchTenantHealth = async () => {
    setLoading(true);
    try {
      // Simulated data - would fetch from API
      const data: TenantHealth[] = [
        { id: 'T-001', name: 'Acme Corp', region: 'NA', jobCount: 45, successRate: 98.2, failedJobs: 1, sloBreaches: 0, riskScore: 0.1, trend: 'stable' },
        { id: 'T-002', name: 'GlobalFin', region: 'EU', jobCount: 62, successRate: 94.5, failedJobs: 3, sloBreaches: 1, riskScore: 0.4, trend: 'down' },
        { id: 'T-003', name: 'TechStart', region: 'NA', jobCount: 28, successRate: 100, failedJobs: 0, sloBreaches: 0, riskScore: 0.05, trend: 'up' },
        { id: 'T-005', name: 'BioMed Inc', region: 'EU', jobCount: 51, successRate: 88.2, failedJobs: 6, sloBreaches: 2, riskScore: 0.6, trend: 'down' },
        { id: 'T-008', name: 'DataFlow', region: 'APAC', jobCount: 33, successRate: 96.9, failedJobs: 1, sloBreaches: 0, riskScore: 0.15, trend: 'stable' },
        { id: 'T-012', name: 'CloudSys', region: 'NA', jobCount: 78, successRate: 97.4, failedJobs: 2, sloBreaches: 0, riskScore: 0.2, trend: 'up' },
        { id: 'T-015', name: 'FinServe', region: 'EU', jobCount: 89, successRate: 92.1, failedJobs: 7, sloBreaches: 3, riskScore: 0.55, trend: 'down' },
        { id: 'T-018', name: 'RetailMax', region: 'APAC', jobCount: 41, successRate: 99.5, failedJobs: 0, sloBreaches: 0, riskScore: 0.08, trend: 'up' },
        { id: 'T-021', name: 'LogiTech', region: 'NA', jobCount: 55, successRate: 95.8, failedJobs: 2, sloBreaches: 1, riskScore: 0.25, trend: 'stable' },
        { id: 'T-024', name: 'EuroBank', region: 'EU', jobCount: 112, successRate: 99.1, failedJobs: 1, sloBreaches: 0, riskScore: 0.12, trend: 'up' },
      ];
      setTenants(data);
    } finally {
      setLoading(false);
    }
  };

  const getHealthColor = (successRate: number): string => {
    if (successRate >= 98) return '#22c55e'; // Green
    if (successRate >= 95) return '#84cc16'; // Lime
    if (successRate >= 90) return '#eab308'; // Yellow
    if (successRate >= 85) return '#f97316'; // Orange
    return '#ef4444'; // Red
  };

  const getRiskBadge = (riskScore: number): string => {
    if (riskScore >= 0.5) return 'risk-high';
    if (riskScore >= 0.25) return 'risk-medium';
    return 'risk-low';
  };

  const getTrendIcon = (trend: string): string => {
    switch (trend) {
      case 'up': return '↑';
      case 'down': return '↓';
      default: return '→';
    }
  };

  const filteredTenants = tenants
    .filter(t => filterRegion === 'all' || t.region === filterRegion)
    .sort((a, b) => {
      switch (sortBy) {
        case 'name': return a.name.localeCompare(b.name);
        case 'successRate': return a.successRate - b.successRate;
        case 'riskScore': return b.riskScore - a.riskScore;
        default: return 0;
      }
    });

  const regions = ['all', ...Array.from(new Set(tenants.map(t => t.region)))];

  const handleTenantClick = (tenantId: string) => {
    setSelectedTenant(tenantId);
    onTenantSelect?.(tenantId);
  };

  if (!permissions.canViewCrossTenant) {
    return null;
  }

  if (loading) {
    return <div className="heatmap-loading">Loading tenant data...</div>;
  }

  return (
    <div className="tenant-heatmap">
      {/* Controls */}
      <div className="heatmap-controls">
        <div className="control-group">
          <label>Region:</label>
          <select value={filterRegion} onChange={e => setFilterRegion(e.target.value)}>
            {regions.map(r => (
              <option key={r} value={r}>{r === 'all' ? 'All Regions' : r}</option>
            ))}
          </select>
        </div>
        <div className="control-group">
          <label>Sort by:</label>
          <select value={sortBy} onChange={e => setSortBy(e.target.value as any)}>
            <option value="riskScore">Risk Score</option>
            <option value="successRate">Success Rate</option>
            <option value="name">Name</option>
          </select>
        </div>
      </div>

      {/* Heatmap Grid */}
      <div className="heatmap-grid">
        {filteredTenants.map(tenant => (
          <div
            key={tenant.id}
            className={`heatmap-cell ${selectedTenant === tenant.id ? 'selected' : ''}`}
            onClick={() => handleTenantClick(tenant.id)}
            style={{
              borderColor: getHealthColor(tenant.successRate),
              boxShadow: `0 0 8px ${getHealthColor(tenant.successRate)}40`,
            }}
          >
            <div className="cell-header">
              <span className="tenant-name">{tenant.name}</span>
              <span className="tenant-region">{tenant.region}</span>
            </div>
            
            <div className="cell-metrics">
              <div className="metric-bar">
                <div 
                  className="metric-fill"
                  style={{ 
                    width: `${tenant.successRate}%`,
                    backgroundColor: getHealthColor(tenant.successRate)
                  }}
                />
              </div>
              <span className="metric-value">{tenant.successRate.toFixed(1)}%</span>
            </div>

            <div className="cell-footer">
              <span className="job-count">{tenant.jobCount} jobs</span>
              {tenant.failedJobs > 0 && (
                <span className="failed-count">❌ {tenant.failedJobs}</span>
              )}
              {tenant.sloBreaches > 0 && (
                <span className="slo-count">⚠️ {tenant.sloBreaches} SLO</span>
              )}
              <span className={`trend trend-${tenant.trend}`}>
                {getTrendIcon(tenant.trend)}
              </span>
            </div>

            <div className={`risk-badge ${getRiskBadge(tenant.riskScore)}`}>
              {Math.round(tenant.riskScore * 100)}%
            </div>
          </div>
        ))}
      </div>

      {/* Legend */}
      <div className="heatmap-legend">
        <span className="legend-item">
          <span className="legend-color" style={{ backgroundColor: '#22c55e' }} />
          ≥98%
        </span>
        <span className="legend-item">
          <span className="legend-color" style={{ backgroundColor: '#84cc16' }} />
          95-98%
        </span>
        <span className="legend-item">
          <span className="legend-color" style={{ backgroundColor: '#eab308' }} />
          90-95%
        </span>
        <span className="legend-item">
          <span className="legend-color" style={{ backgroundColor: '#f97316' }} />
          85-90%
        </span>
        <span className="legend-item">
          <span className="legend-color" style={{ backgroundColor: '#ef4444' }} />
          &lt;85%
        </span>
      </div>
    </div>
  );
};

export default TenantHeatmap;
