/**
 * Data Quality Monitor - AI-driven quality checks and anomaly detection
 */

import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  CheckCircle as CheckCircleIcon,
  Warning as WarningIcon,
  Error as ErrorIcon,
  TrendingUp as TrendingUpIcon,
  TrendingDown as TrendingDownIcon,
  ArrowBack as ArrowBackIcon,
  Search as SearchIcon,
  FilterList as FilterListIcon,
} from '@mui/icons-material';
import './IntelligenceDashboard.css';

interface QualityMetric {
  id: string;
  name: string;
  table: string;
  schema: string;
  score: number;
  trend: 'up' | 'down' | 'stable';
  lastRun: string;
  issuesFound: number;
  status: 'healthy' | 'warning' | 'critical';
}

const DataQualityMonitorPage: React.FC = () => {
  const navigate = useNavigate();
  const [metrics, setMetrics] = useState<QualityMetric[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchMetrics();
  }, []);

  const fetchMetrics = async () => {
    setLoading(true);
    try {
      // Simulated data
      const data: QualityMetric[] = [
        {
          id: '1',
          name: 'Tax ID Completeness',
          table: 'clients',
          schema: 'public',
          score: 94.5,
          trend: 'down',
          lastRun: '1 hour ago',
          issuesFound: 245,
          status: 'warning',
        },
        {
          id: '2',
          name: 'Transaction Date Integrity',
          table: 'transactions',
          schema: 'public',
          score: 99.8,
          trend: 'stable',
          lastRun: '2 hours ago',
          issuesFound: 12,
          status: 'healthy',
        },
        {
          id: '3',
          name: 'Account Status Enums',
          table: 'accounts',
          schema: 'public',
          score: 99.9,
          trend: 'up',
          lastRun: '30 mins ago',
          issuesFound: 8,
          status: 'healthy',
        },
        {
          id: '4',
          name: 'Price Feed Accuracy',
          table: 'market_data',
          schema: 'market',
          score: 82.1,
          trend: 'down',
          lastRun: '5 mins ago',
          issuesFound: 1420,
          status: 'critical',
        },
      ];
      setMetrics(data);
    } finally {
      setLoading(false);
    }
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'healthy': return <CheckCircleIcon style={{ color: '#10b981' }} />;
      case 'warning': return <WarningIcon style={{ color: '#f59e0b' }} />;
      case 'critical': return <ErrorIcon style={{ color: '#ef4444' }} />;
      default: return null;
    }
  };

  const getTrendIcon = (trend: string) => {
    switch (trend) {
      case 'up': return <TrendingUpIcon style={{ color: '#10b981', fontSize: '1rem' }} />;
      case 'down': return <TrendingDownIcon style={{ color: '#ef4444', fontSize: '1rem' }} />;
      default: return null;
    }
  };

  if (loading) {
    return <div className="intelligence-loading"><div className="loading-spinner" /><p>Running quality sanity checks...</p></div>;
  }

  return (
    <div className="intelligence-dashboard">
      <header className="dashboard-header">
        <button className="back-btn" onClick={() => navigate('/intelligence')}>
          <ArrowBackIcon /> Back
        </button>
        <div className="header-content">
          <CheckCircleIcon className="header-icon" />
          <div>
            <h1>Data Quality Monitor</h1>
            <p>AI-powered observability and quality assurance for your data layer</p>
          </div>
        </div>
      </header>

      <div className="quality-summary-grid">
        <style>{`
          .quality-summary-grid {
            display: grid;
            grid-template-columns: repeat(3, 1fr);
            gap: 1rem;
            margin-bottom: 2rem;
          }
          .metric-overview-card {
            background: white;
            padding: 1.5rem;
            border-radius: 12px;
            box-shadow: 0 2px 8px rgba(0,0,0,0.05);
            border: 1px solid #e5e7eb;
            display: flex;
            flex-direction: column;
            gap: 0.5rem;
          }
          .metric-label {
            font-size: 0.85rem;
            color: #6b7280;
            font-weight: 500;
          }
          .metric-value-row {
            display: flex;
            align-items: baseline;
            gap: 0.5rem;
          }
          .metric-value {
            font-size: 1.75rem;
            font-weight: 700;
          }
          .metric-unit {
            color: #6b7280;
            font-size: 1rem;
          }
          .quality-table-container {
            background: white;
            border-radius: 12px;
            border: 1px solid #e5e7eb;
            overflow: hidden;
          }
          .quality-table {
            width: 100%;
            border-collapse: collapse;
          }
          .quality-table th {
            text-align: left;
            padding: 1rem;
            background: #f9fafb;
            font-size: 0.75rem;
            text-transform: uppercase;
            color: #6b7280;
            letter-spacing: 0.05em;
            border-bottom: 1px solid #e5e7eb;
          }
          .quality-table td {
            padding: 1rem;
            border-bottom: 1px solid #f3f4f6;
            vertical-align: middle;
          }
          .metric-name-cell {
            display: flex;
            flex-direction: column;
          }
          .name-label {
            font-weight: 600;
            font-size: 0.95rem;
          }
          .table-label {
            font-size: 0.8rem;
            color: #6b7280;
          }
          .score-cell {
            display: flex;
            align-items: center;
            gap: 0.5rem;
            font-weight: 600;
          }
          .score-bar-bg {
            width: 80px;
            height: 6px;
            background: #f3f4f6;
            border-radius: 3px;
            overflow: hidden;
          }
          .score-bar-fg {
            height: 100%;
            border-radius: 3px;
          }
        `}</style>
        <div className="metric-overview-card">
          <span className="metric-label">Global Quality Score</span>
          <div className="metric-value-row">
            <span className="metric-value">94.8</span>
            <span className="metric-unit">/ 100</span>
          </div>
          <span className="status-label" style={{ color: '#f59e0b', fontSize: '0.85rem', fontWeight: 600 }}>Needs Attention (3 warnings)</span>
        </div>
        <div className="metric-overview-card">
          <span className="metric-label">Checks Running</span>
          <div className="metric-value-row">
            <span className="metric-value">124</span>
          </div>
          <span className="status-label" style={{ color: '#6b7280', fontSize: '0.85rem' }}>Active monitoring</span>
        </div>
        <div className="metric-overview-card">
          <span className="metric-label">Anomalies Detected</span>
          <div className="metric-value-row">
            <span className="metric-value" style={{ color: '#ef4444' }}>14</span>
          </div>
          <span className="status-label" style={{ color: '#ef4444', fontSize: '0.85rem', fontWeight: 600 }}>Immediate review required</span>
        </div>
      </div>

      <div className="quality-table-container">
        <table className="quality-table">
          <thead>
            <tr>
              <th>Status</th>
              <th>Metric Name</th>
              <th>Quality Score</th>
              <th>Issues</th>
              <th>Last Run</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            {metrics.map(metric => (
              <tr key={metric.id}>
                <td>{getStatusIcon(metric.status)}</td>
                <td>
                  <div className="metric-name-cell">
                    <span className="name-label">{metric.name}</span>
                    <span className="table-label">{metric.schema}.{metric.table}</span>
                  </div>
                </td>
                <td>
                  <div className="score-cell">
                    <span>{metric.score}%</span>
                    {getTrendIcon(metric.trend)}
                    <div className="score-bar-bg">
                      <div 
                        className="score-bar-fg" 
                        style={{ 
                          width: `${metric.score}%`,
                          backgroundColor: metric.score > 95 ? '#10b981' : metric.score > 85 ? '#f59e0b' : '#ef4444'
                        }} 
                      />
                    </div>
                  </div>
                </td>
                <td>
                  <span style={{ fontWeight: 600, color: metric.issuesFound > 0 ? '#ef4444' : 'inherit' }}>
                    {metric.issuesFound}
                  </span>
                </td>
                <td style={{ fontSize: '0.85rem', color: '#6b7280' }}>{metric.lastRun}</td>
                <td>
                  <button className="view-btn">Inspect</button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
};

export default DataQualityMonitorPage;
