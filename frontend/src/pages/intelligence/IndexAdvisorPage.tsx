/**
 * Index Advisor Page - AI-driven index recommendations
 */

import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Storage as StorageIcon,
  TrendingUp as TrendingUpIcon,
  CheckCircle as CheckCircleIcon,
  Warning as WarningIcon,
  ArrowBack as ArrowBackIcon,
  AutoFixHigh as AutoFixHighIcon,
} from '@mui/icons-material';
import './IntelligenceDashboard.css';

interface IndexRecommendation {
  id: string;
  table: string;
  schema: string;
  columns: string[];
  indexType: 'btree' | 'hash' | 'composite' | 'covering';
  impact: {
    queryTimeReduction: number;
    queriesAffected: number;
    estimatedCostSavings: string;
  };
  riskLevel: 'low' | 'medium' | 'high';
  status: 'pending' | 'applied' | 'dismissed';
  createdAt: string;
  query: string;
}

const IndexAdvisorPage: React.FC = () => {
  const navigate = useNavigate();
  const [recommendations, setRecommendations] = useState<IndexRecommendation[]>([]);
  const [filter, setFilter] = useState<string>('all');
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchRecommendations();
  }, []);

  const fetchRecommendations = async () => {
    setLoading(true);
    try {
      // Simulated data
      const data: IndexRecommendation[] = [
        {
          id: '1',
          table: 'positions',
          schema: 'public',
          columns: ['account_id', 'as_of_date'],
          indexType: 'btree',
          impact: { queryTimeReduction: 65, queriesAffected: 142, estimatedCostSavings: '$2,400/mo' },
          riskLevel: 'low',
          status: 'pending',
          createdAt: '2026-01-17T10:00:00Z',
          query: 'SELECT * FROM positions WHERE account_id = ? AND as_of_date >= ?',
        },
        {
          id: '2',
          table: 'transactions',
          schema: 'public',
          columns: ['tenant_id', 'created_at'],
          indexType: 'btree',
          impact: { queryTimeReduction: 40, queriesAffected: 89, estimatedCostSavings: '$1,200/mo' },
          riskLevel: 'low',
          status: 'pending',
          createdAt: '2026-01-17T09:30:00Z',
          query: 'SELECT * FROM transactions WHERE tenant_id = ? ORDER BY created_at DESC',
        },
        {
          id: '3',
          table: 'holdings',
          schema: 'public',
          columns: ['security_id', 'portfolio_id'],
          indexType: 'composite',
          impact: { queryTimeReduction: 55, queriesAffected: 67, estimatedCostSavings: '$800/mo' },
          riskLevel: 'medium',
          status: 'pending',
          createdAt: '2026-01-17T08:00:00Z',
          query: 'SELECT h.*, s.symbol FROM holdings h JOIN securities s ON h.security_id = s.id WHERE portfolio_id = ?',
        },
        {
          id: '4',
          table: 'audit_logs',
          schema: 'public',
          columns: ['entity_type', 'entity_id', 'timestamp'],
          indexType: 'composite',
          impact: { queryTimeReduction: 72, queriesAffected: 234, estimatedCostSavings: '$3,100/mo' },
          riskLevel: 'low',
          status: 'applied',
          createdAt: '2026-01-16T14:00:00Z',
          query: 'SELECT * FROM audit_logs WHERE entity_type = ? AND entity_id = ? ORDER BY timestamp DESC',
        },
      ];
      setRecommendations(data);
    } finally {
      setLoading(false);
    }
  };

  const getRiskColor = (level: string): string => {
    switch (level) {
      case 'high': return '#dc2626';
      case 'medium': return '#ca8a04';
      case 'low': return '#16a34a';
      default: return '#6b7280';
    }
  };

  const getStatusBadge = (status: string) => {
    switch (status) {
      case 'applied':
        return <span className="status-badge applied"><CheckCircleIcon /> Applied</span>;
      case 'dismissed':
        return <span className="status-badge dismissed">Dismissed</span>;
      default:
        return <span className="status-badge pending">Pending</span>;
    }
  };

  const handleApply = (id: string) => {
    setRecommendations(prev => 
      prev.map(r => r.id === id ? { ...r, status: 'applied' } : r) as IndexRecommendation[]
    );
  };

  const filteredRecs = recommendations.filter(r => 
    filter === 'all' || r.status === filter
  );

  if (loading) {
    return <div className="intelligence-loading"><div className="loading-spinner" /><p>Analyzing query patterns...</p></div>;
  }

  return (
    <div className="intelligence-dashboard">
      <header className="dashboard-header">
        <button className="back-btn" onClick={() => navigate('/intelligence')}>
          <ArrowBackIcon /> Back
        </button>
        <div className="header-content">
          <StorageIcon className="header-icon" />
          <div>
            <h1>Index Advisor</h1>
            <p>AI-powered index recommendations based on query patterns</p>
          </div>
        </div>
      </header>

      {/* Filters */}
      <div className="filter-bar">
        <select value={filter} onChange={e => setFilter(e.target.value)}>
          <option value="all">All Recommendations</option>
          <option value="pending">Pending</option>
          <option value="applied">Applied</option>
          <option value="dismissed">Dismissed</option>
        </select>
        <span className="result-count">{filteredRecs.length} recommendations</span>
      </div>

      {/* Recommendations List */}
      <div className="recommendations-list">
        {filteredRecs.map(rec => (
          <div key={rec.id} className="recommendation-card">
            <div className="rec-header">
              <div className="rec-title">
                <span className="table-name">{rec.schema}.{rec.table}</span>
                <span className="index-type">{rec.indexType} index</span>
              </div>
              {getStatusBadge(rec.status)}
            </div>

            <div className="rec-columns">
              <strong>Suggested Index:</strong>
              <code>{rec.columns.join(', ')}</code>
            </div>

            <div className="rec-query">
              <strong>Sample Query:</strong>
              <pre>{rec.query}</pre>
            </div>

            <div className="rec-impact">
              <div className="impact-item">
                <TrendingUpIcon />
                <span>{rec.impact.queryTimeReduction}% faster</span>
              </div>
              <div className="impact-item">
                <span>{rec.impact.queriesAffected} queries affected</span>
              </div>
              <div className="impact-item savings">
                <span>{rec.impact.estimatedCostSavings}</span>
              </div>
              <div className="impact-item">
                <span style={{ color: getRiskColor(rec.riskLevel) }}>
                  {rec.riskLevel} risk
                </span>
              </div>
            </div>

            {rec.status === 'pending' && (
              <div className="rec-actions">
                <button className="apply-btn" onClick={() => handleApply(rec.id)}>
                  <AutoFixHighIcon /> Apply Index
                </button>
                <button className="dismiss-btn">Dismiss</button>
              </div>
            )}
          </div>
        ))}
      </div>
    </div>
  );
};

export default IndexAdvisorPage;
