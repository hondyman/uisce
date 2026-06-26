/**
 * Intelligence Dashboard - AI optimization hub
 * Central location for all AI-powered optimization and monitoring features
 */

import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Speed as SpeedIcon,
  Storage as StorageIcon,
  CheckCircle as CheckCircleIcon,
  TrendingUp as TrendingUpIcon,
  AutoFixHigh as AutoFixHighIcon,
  Layers as LayersIcon,
  QueryStats as QueryStatsIcon,
  Warning as WarningIcon,
} from '@mui/icons-material';
import './IntelligenceDashboard.css';

interface OptimizationSummary {
  totalOptimizations: number;
  pendingActions: number;
  appliedThisWeek: number;
  estimatedSavings: string;
}

interface IndexRecommendation {
  id: string;
  table: string;
  suggestedIndex: string;
  impact: string;
  riskLevel: 'low' | 'medium' | 'high';
  estimatedImprovement: number;
}

interface StorageTierSuggestion {
  id: string;
  dataSet: string;
  currentTier: string;
  suggestedTier: string;
  savingsPerMonth: string;
  dataAge: string;
}

interface DataQualityIssue {
  id: string;
  table: string;
  column: string;
  issueType: string;
  severity: 'critical' | 'high' | 'medium' | 'low';
  recordsAffected: number;
}

const IntelligenceDashboard: React.FC = () => {
  const navigate = useNavigate();
  const [summary, setSummary] = useState<OptimizationSummary | null>(null);
  const [indexRecs, setIndexRecs] = useState<IndexRecommendation[]>([]);
  const [storageSuggestions, setStorageSuggestions] = useState<StorageTierSuggestion[]>([]);
  const [qualityIssues, setQualityIssues] = useState<DataQualityIssue[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchDashboardData();
  }, []);

  const fetchDashboardData = async () => {
    setLoading(true);
    try {
      // Simulated data - would fetch from API
      setSummary({
        totalOptimizations: 47,
        pendingActions: 12,
        appliedThisWeek: 8,
        estimatedSavings: '$12,500/mo',
      });

      setIndexRecs([
        { id: '1', table: 'positions', suggestedIndex: 'account_id, as_of_date', impact: 'Query time -65%', riskLevel: 'low', estimatedImprovement: 65 },
        { id: '2', table: 'transactions', suggestedIndex: 'tenant_id, created_at DESC', impact: 'Query time -40%', riskLevel: 'low', estimatedImprovement: 40 },
        { id: '3', table: 'holdings', suggestedIndex: 'security_id, portfolio_id', impact: 'Query time -55%', riskLevel: 'medium', estimatedImprovement: 55 },
      ]);

      setStorageSuggestions([
        { id: '1', dataSet: 'historical_positions', currentTier: 'Hot', suggestedTier: 'Cold', savingsPerMonth: '$2,400', dataAge: '2+ years' },
        { id: '2', dataSet: 'audit_logs_2022', currentTier: 'Hot', suggestedTier: 'Archive', savingsPerMonth: '$800', dataAge: '3+ years' },
        { id: '3', dataSet: 'market_data_snapshots', currentTier: 'Warm', suggestedTier: 'Cold', savingsPerMonth: '$1,200', dataAge: '1+ year' },
      ]);

      setQualityIssues([
        { id: '1', table: 'clients', column: 'tax_id', issueType: 'Null values', severity: 'high', recordsAffected: 245 },
        { id: '2', table: 'transactions', column: 'settlement_date', issueType: 'Future dates', severity: 'medium', recordsAffected: 12 },
        { id: '3', table: 'accounts', column: 'status', issueType: 'Invalid enum', severity: 'low', recordsAffected: 8 },
      ]);
    } finally {
      setLoading(false);
    }
  };

  const getRiskColor = (level: string): string => {
    switch (level) {
      case 'critical': return '#dc2626';
      case 'high': return '#ea580c';
      case 'medium': return '#ca8a04';
      case 'low': return '#16a34a';
      default: return '#6b7280';
    }
  };

  if (loading) {
    return (
      <div className="intelligence-loading">
        <div className="loading-spinner" />
        <p>Analyzing optimization opportunities...</p>
      </div>
    );
  }

  return (
    <div className="intelligence-dashboard">
      {/* Header */}
      <header className="dashboard-header">
        <div className="header-content">
          <AutoFixHighIcon className="header-icon" />
          <div>
            <h1>Intelligence Dashboard</h1>
            <p>AI-powered optimization and monitoring hub</p>
          </div>
        </div>
      </header>

      {/* Summary Cards */}
      <div className="summary-grid">
        <div className="summary-card">
          <SpeedIcon className="card-icon optimization" />
          <div className="card-content">
            <span className="value">{summary?.totalOptimizations}</span>
            <span className="label">Total Optimizations</span>
          </div>
        </div>
        <div className="summary-card pending">
          <WarningIcon className="card-icon pending" />
          <div className="card-content">
            <span className="value">{summary?.pendingActions}</span>
            <span className="label">Pending Actions</span>
          </div>
        </div>
        <div className="summary-card applied">
          <CheckCircleIcon className="card-icon applied" />
          <div className="card-content">
            <span className="value">{summary?.appliedThisWeek}</span>
            <span className="label">Applied This Week</span>
          </div>
        </div>
        <div className="summary-card savings">
          <TrendingUpIcon className="card-icon savings" />
          <div className="card-content">
            <span className="value">{summary?.estimatedSavings}</span>
            <span className="label">Est. Savings</span>
          </div>
        </div>
      </div>

      {/* Feature Cards Grid */}
      <div className="feature-grid">
        {/* Index Advisor */}
        <div className="feature-card">
          <div className="feature-header">
            <StorageIcon className="feature-icon" />
            <h2>Index Advisor</h2>
            <button className="view-all" onClick={() => navigate('/intelligence/index-advisor')}>
              View All →
            </button>
          </div>
          <div className="feature-content">
            {indexRecs.map(rec => (
              <div key={rec.id} className="recommendation-item">
                <div className="rec-info">
                  <span className="rec-table">{rec.table}</span>
                  <span className="rec-index">{rec.suggestedIndex}</span>
                </div>
                <div className="rec-metrics">
                  <span className="rec-impact">{rec.impact}</span>
                  <span 
                    className="rec-risk"
                    style={{ color: getRiskColor(rec.riskLevel) }}
                  >
                    {rec.riskLevel}
                  </span>
                </div>
                <button className="apply-btn">Apply</button>
              </div>
            ))}
          </div>
        </div>

        {/* Storage Tiering */}
        <div className="feature-card">
          <div className="feature-header">
            <LayersIcon className="feature-icon" />
            <h2>Storage Tiering</h2>
            <button className="view-all" onClick={() => navigate('/intelligence/storage')}>
              View All →
            </button>
          </div>
          <div className="feature-content">
            {storageSuggestions.map(suggestion => (
              <div key={suggestion.id} className="tier-item">
                <div className="tier-info">
                  <span className="tier-dataset">{suggestion.dataSet}</span>
                  <span className="tier-migration">
                    {suggestion.currentTier} → {suggestion.suggestedTier}
                  </span>
                </div>
                <div className="tier-metrics">
                  <span className="tier-savings">{suggestion.savingsPerMonth}</span>
                  <span className="tier-age">{suggestion.dataAge}</span>
                </div>
                <button className="apply-btn">Migrate</button>
              </div>
            ))}
          </div>
        </div>

        {/* Data Quality */}
        <div className="feature-card full-width">
          <div className="feature-header">
            <CheckCircleIcon className="feature-icon" />
            <h2>Data Quality Monitor</h2>
            <button className="view-all" onClick={() => navigate('/intelligence/data-quality')}>
              View All →
            </button>
          </div>
          <div className="feature-content quality-grid">
            {qualityIssues.map(issue => (
              <div key={issue.id} className="quality-item">
                <div 
                  className="severity-indicator"
                  style={{ backgroundColor: getRiskColor(issue.severity) }}
                />
                <div className="quality-info">
                  <span className="quality-table">{issue.table}.{issue.column}</span>
                  <span className="quality-issue">{issue.issueType}</span>
                </div>
                <div className="quality-metrics">
                  <span className="quality-affected">{issue.recordsAffected.toLocaleString()} records</span>
                  <span 
                    className="quality-severity"
                    style={{ color: getRiskColor(issue.severity) }}
                  >
                    {issue.severity}
                  </span>
                </div>
                <button className="fix-btn">Fix</button>
              </div>
            ))}
          </div>
        </div>

        {/* Quick Actions */}
        <div className="quick-actions">
          <h3>Quick Actions</h3>
          <div className="action-buttons">
            <button onClick={() => navigate('/optimization')}>
              <SpeedIcon />
              <span>ASO Center</span>
            </button>
            <button onClick={() => navigate('/observability')}>
              <QueryStatsIcon />
              <span>Metrics</span>
            </button>
            <button onClick={() => navigate('/nlq')}>
              <AutoFixHighIcon />
              <span>AI Copilot</span>
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};

export default IntelligenceDashboard;
