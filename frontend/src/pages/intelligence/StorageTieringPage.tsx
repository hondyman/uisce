/**
 * Storage Tiering Page - AI-driven data lifecycle management
 */

import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Layers as LayersIcon,
  TrendingDown as TrendingDownIcon,
  CheckCircle as CheckCircleIcon,
  Warning as WarningIcon,
  ArrowBack as ArrowBackIcon,
  Bolt as BoltIcon,
  AcUnit as AcUnitIcon,
  CloudQueue as CloudQueueIcon,
} from '@mui/icons-material';
import './IntelligenceDashboard.css';

import { tieringApi, TieringPlan, TieringRule, PlanStatus, StorageTier } from '../../api/tieringApi';

const StorageTieringPage: React.FC = () => {
  const navigate = useNavigate();
  const [plans, setPlans] = useState<TieringPlan[]>([]);
  const [activePlan, setActivePlan] = useState<TieringPlan | null>(null);
  const [loading, setLoading] = useState(true);

  // For demo, we use a fixed tenant ID
  const tenantId = '00000000-0000-0000-0000-000000000001';

  useEffect(() => {
    fetchData();
  }, []);

  const fetchData = async () => {
    setLoading(true);
    try {
      const existingPlans = await tieringApi.listPlans(tenantId);
      if (existingPlans.length > 0) {
        setPlans(existingPlans);
        setActivePlan(existingPlans[0]);
      } else {
        // Generate initial plan if none exist
        const newPlan = await tieringApi.generatePlan(tenantId);
        setPlans([newPlan]);
        setActivePlan(newPlan);
      }
    } catch (error) {
      console.error('Failed to fetch storage tiering plans', error);
    } finally {
      setLoading(false);
    }
  };

  const handleGenerateNew = async () => {
    setLoading(true);
    try {
      const newPlan = await tieringApi.generatePlan(tenantId);
      setPlans(prev => [newPlan, ...prev]);
      setActivePlan(newPlan);
    } finally {
      setLoading(false);
    }
  };


  const handleMigrate = async (planId: string) => {
    try {
      setPlans(prev =>
        prev.map(p => p.id === planId ? { ...p, status: 'migrating' as PlanStatus } : p)
      );
      if (activePlan?.id === planId) {
        setActivePlan({ ...activePlan, status: 'migrating' as PlanStatus });
      }

      await tieringApi.executePlan(planId);

      setPlans(prev =>
        prev.map(p => p.id === planId ? { ...p, status: 'completed' as PlanStatus } : p)
      );
      if (activePlan?.id === planId) {
        setActivePlan({ ...activePlan, status: 'completed' as PlanStatus });
      }
    } catch (error) {
      console.error('Migration failed', error);
      fetchData(); // Refresh on error
    }
  };

  const getTierIcon = (tier: StorageTier) => {
    switch (tier) {
      case 'hot': return <BoltIcon style={{ color: '#ef4444' }} />;
      case 'warm': return <CloudQueueIcon style={{ color: '#f59e0b' }} />;
      case 'cold': return <AcUnitIcon style={{ color: '#3b82f6' }} />;
      case 'archive': return <LayersIcon style={{ color: '#6b7280' }} />;
      default: return null;
    }
  };

  if (loading) {
    return <div className="intelligence-loading"><div className="loading-spinner" /><p>Scanning data storage utilization...</p></div>;
  }

  const totalSavings = activePlan?.rules
    .reduce((acc, r) => {
      const savings = parseInt(r.costSavings.replace('$', '').replace(',', '').replace('/month', ''));
      return isNaN(savings) ? acc : acc + savings;
    }, 0) || 0;

  return (
    <div className="intelligence-dashboard">
      <header className="dashboard-header">
        <button className="back-btn" onClick={() => navigate('/intelligence')}>
          <ArrowBackIcon /> Back
        </button>
        <div className="header-content">
          <LayersIcon className="header-icon" />
          <div>
            <h1>Storage Tiering</h1>
            <p>AI-driven recommendations for data lifecycle and cost optimization</p>
          </div>
        </div>
        <button className="generate-btn" onClick={handleGenerateNew} disabled={loading}>
          {loading ? 'Analyzing...' : 'Refresh Analysis'}
        </button>
      </header>

      <div className="summary-grid">
        <div className="summary-card savings">
          <TrendingDownIcon className="card-icon" />
          <div className="card-content">
            <span className="value">${totalSavings.toLocaleString()}/mo</span>
            <span className="label">Total Potential Savings</span>
          </div>
        </div>
        <div className="summary-card">
          <LayersIcon className="card-icon" />
          <div className="card-content">
            <span className="value">{activePlan?.rules.length || 0}</span>
            <span className="label">Storage Objects Analyzed</span>
          </div>
        </div>
      </div>

      <div className="suggestions-list">
        {!activePlan && <p>No tiering recommendations found.</p>}
        {activePlan?.rules.map((rule, idx) => (
          <div key={`${activePlan.id}-${idx}`} className="suggestion-card">
            <style>{`
              .suggestion-card {
                background: white;
                border: 1px solid #e5e7eb;
                border-radius: 12px;
                padding: 1.5rem;
                margin-bottom: 1rem;
                display: flex;
                flex-direction: column;
                gap: 1rem;
              }
              .suggestion-header {
                display: flex;
                justify-content: space-between;
                align-items: center;
              }
              .suggestion-title {
                display: flex;
                flex-direction: column;
              }
              .dataset-name {
                font-weight: 600;
                font-size: 1.1rem;
              }
              .table-path {
                font-size: 0.85rem;
                color: #6b7280;
              }
              .tier-transition {
                display: flex;
                align-items: center;
                gap: 0.75rem;
                background: #f9fafb;
                padding: 0.75rem 1rem;
                border-radius: 8px;
              }
              .tier-box {
                display: flex;
                align-items: center;
                gap: 0.5rem;
                font-weight: 500;
                font-size: 0.9rem;
                text-transform: capitalize;
              }
              .arrow-divider {
                color: #9ca3af;
              }
              .suggestion-meta {
                display: grid;
                grid-template-columns: repeat(4, 1fr);
                gap: 1.5rem;
                padding: 1rem 0;
                border-top: 1px solid #f3f4f6;
                border-bottom: 1px solid #f3f4f6;
              }
              .meta-item {
                display: flex;
                flex-direction: column;
                gap: 0.25rem;
              }
              .meta-label {
                font-size: 0.75rem;
                color: #6b7280;
                text-transform: uppercase;
                letter-spacing: 0.05em;
              }
              .meta-value {
                font-weight: 600;
                font-size: 1rem;
              }
              .meta-value.savings {
                color: #10b981;
              }
              .suggestion-actions {
                display: flex;
                justify-content: flex-end;
                gap: 0.75rem;
                align-items: center;
              }
              .rationale-text {
                font-size: 0.9rem;
                color: #4b5563;
                background: #f3f4f6;
                padding: 0.75rem;
                border-radius: 6px;
                border-left: 3px solid #673AB7;
              }
              .status-badge.completed {
                background: #ecfdf5;
                color: #059669;
                padding: 0.5rem 1rem;
                border-radius: 6px;
                font-weight: 500;
                display: flex;
                align-items: center;
                gap: 0.5rem;
              }
              .status-badge.migrating {
                background: #f5f3ff;
                color: #673AB7;
                padding: 0.5rem 1rem;
                border-radius: 6px;
                font-weight: 500;
                display: flex;
                align-items: center;
                gap: 0.5rem;
              }
              .migrating-spinner {
                width: 16px;
                height: 16px;
                border: 2px solid rgba(103, 58, 183, 0.2);
                border-top-color: #673AB7;
                border-radius: 50%;
                animation: spin 1s linear infinite;
              }
              @keyframes spin {
                from { transform: rotate(0deg); }
                to { transform: rotate(360deg); }
              }
              .generate-btn {
                background: #673AB7;
                color: white;
                border: none;
                padding: 0.6rem 1.2rem;
                border-radius: 8px;
                font-weight: 600;
                cursor: pointer;
                display: flex;
                align-items: center;
                gap: 0.5rem;
                transition: all 0.2s;
              }
              .generate-btn:hover:not(:disabled) {
                background: #5E35B1;
                transform: translateY(-1px);
                box-shadow: 0 4px 12px rgba(103, 58, 183, 0.2);
              }
              .generate-btn:disabled {
                opacity: 0.6;
                cursor: not-allowed;
              }
            `}</style>
            
            <div className="suggestion-header">
              <div className="suggestion-title">
                <span className="dataset-name">{rule.tableName}</span>
                <span className="table-path">{rule.condition}</span>
              </div>
              <div className="tier-transition">
                <div className="tier-box">
                  {getTierIcon('hot')}
                  Hot
                </div>
                <span className="arrow-divider">→</span>
                <div className="tier-box">
                  {getTierIcon(rule.targetTier)}
                  {rule.targetTier}
                </div>
              </div>
            </div>

            <div className="rationale-text">
              <strong>Rationale:</strong> {rule.rationale}
            </div>

            <div className="suggestion-meta">
              <div className="meta-item">
                <span className="meta-label">Est. Savings</span>
                <span className="meta-value savings">{rule.costSavings}</span>
              </div>
              <div className="meta-item">
                <span className="meta-label">Data Volume</span>
                <span className="meta-value">{rule.dataVolume}</span>
              </div>
              <div className="meta-item">
                <span className="meta-label">Analysis Date</span>
                <span className="meta-value">{new Date(activePlan.created_at).toLocaleDateString()}</span>
              </div>
              <div className="meta-item">
                <span className="meta-label">Plan ID</span>
                <span className="meta-value">{activePlan.id.split('-')[0]}</span>
              </div>
            </div>

            <div className="suggestion-actions">
              {activePlan.status === 'pending' && (
                <>
                  <button className="dismiss-btn">Ignore</button>
                  <button className="apply-btn" onClick={() => handleMigrate(activePlan.id)}>
                    Execute Tiering Plan
                  </button>
                </>
              )}
              {activePlan.status === 'migrating' && (
                <div className="status-badge migrating">
                  <div className="migrating-spinner" />
                  Moving Data...
                </div>
              )}
              {activePlan.status === 'completed' && (
                <div className="status-badge completed">
                  <CheckCircleIcon fontSize="small" />
                  Migration Complete
                </div>
              )}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
};

export default StorageTieringPage;
