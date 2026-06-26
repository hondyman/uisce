/**
 * Governance Console Page - ChangeSet management
 */

import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Policy as PolicyIcon,
  CheckCircle as CheckCircleIcon,
  Cancel as CancelIcon,
  Warning as WarningIcon,
  Schedule as ScheduleIcon,
  Visibility as VisibilityIcon,
} from '@mui/icons-material';

interface ChangeSet {
  id: string;
  title: string;
  type: 'job_update' | 'dag_update' | 'policy_change' | 'schema_change';
  status: 'pending' | 'approved' | 'rejected' | 'applied';
  createdBy: string;
  createdAt: string;
  riskScore: number;
  description: string;
  affectedItems: string[];
  requiredApprovers: number;
  currentApprovals: number;
  semanticImpact?: {
    affected_bos?: string[];
    affected_apis?: string[];
    downstream_jobs?: string[];
  };
}

const GovernanceConsolePage: React.FC = () => {
  const navigate = useNavigate();
  const [changeSets, setChangeSets] = useState<ChangeSet[]>([]);
  const [filter, setFilter] = useState<string>('pending');
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchChangeSets();
  }, []);

  const fetchChangeSets = async () => {
    setLoading(true);
    try {
      const data: ChangeSet[] = [
        {
          id: 'cs-001',
          title: 'Update Pre-Agg job timeout',
          type: 'job_update',
          status: 'pending',
          createdBy: 'john.smith@example.com',
          createdAt: '2026-01-17T14:30:00Z',
          riskScore: 0.3,
          description: 'Increase timeout from 300s to 600s to prevent timeouts on large datasets',
          affectedItems: ['EU Pre-Agg', 'APAC Pre-Agg'],
          requiredApprovers: 2,
          currentApprovals: 1,
          semanticImpact: {
            affected_bos: ['Customer', 'Account'],
            downstream_jobs: ['Daily Balance Sync'],
          },
        },
        {
          id: 'cs-002',
          title: 'Add Data Quality scan job',
          type: 'job_update',
          status: 'pending',
          createdBy: 'jane.doe@example.com',
          createdAt: '2026-01-17T12:00:00Z',
          riskScore: 0.1,
          description: 'New scheduled job to scan for data quality issues',
          affectedItems: ['positions', 'transactions'],
          requiredApprovers: 1,
          currentApprovals: 0,
        },
        {
          id: 'cs-003',
          title: 'Restructure Risk DAG parallelization',
          type: 'dag_update',
          status: 'approved',
          createdBy: 'bob.wilson@example.com',
          createdAt: '2026-01-16T10:00:00Z',
          riskScore: 0.5,
          description: 'Refactor DAG to parallelize data load steps for 30% faster execution',
          affectedItems: ['Risk Batch DAG'],
          requiredApprovers: 2,
          currentApprovals: 2,
        },
        {
          id: 'cs-004',
          title: 'Update data retention policy',
          type: 'policy_change',
          status: 'rejected',
          createdBy: 'alice.johnson@example.com',
          createdAt: '2026-01-15T09:00:00Z',
          riskScore: 0.7,
          description: 'Change retention period from 7 years to 5 years',
          affectedItems: ['audit_logs', 'transactions', 'positions'],
          requiredApprovers: 3,
          currentApprovals: 1,
        },
      ];
      setChangeSets(data);
    } finally {
      setLoading(false);
    }
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'approved':
        return <CheckCircleIcon style={{ color: '#16a34a' }} />;
      case 'rejected':
        return <CancelIcon style={{ color: '#dc2626' }} />;
      case 'applied':
        return <CheckCircleIcon style={{ color: '#3b82f6' }} />;
      default:
        return <ScheduleIcon style={{ color: '#ca8a04' }} />;
    }
  };

  const getRiskColor = (score: number): string => {
    if (score >= 0.7) return '#dc2626';
    if (score >= 0.4) return '#ca8a04';
    return '#16a34a';
  };

  const filteredChangeSets = changeSets.filter(cs =>
    filter === 'all' || cs.status === filter
  );

  const handleApprove = (id: string) => {
    setChangeSets(prev =>
      prev.map(cs => {
        if (cs.id === id) {
          const newApprovals = cs.currentApprovals + 1;
          return {
            ...cs,
            currentApprovals: newApprovals,
            status: newApprovals >= cs.requiredApprovers ? 'approved' : cs.status,
          } as ChangeSet;
        }
        return cs;
      })
    );
  };

  if (loading) {
    return <div className="loading-state">Loading change sets...</div>;
  }

  return (
    <div className="governance-console">
      <style>{`
        .governance-console {
          max-width: 1200px;
          margin: 0 auto;
          padding: 2rem;
        }
        .governance-header {
          display: flex;
          align-items: center;
          gap: 1rem;
          margin-bottom: 2rem;
        }
        .governance-header h1 {
          margin: 0;
          font-size: 1.5rem;
          font-weight: 600;
        }
        .governance-header p {
          margin: 0;
          color: #6b7280;
        }
        .filter-tabs {
          display: flex;
          gap: 0.5rem;
          margin-bottom: 1.5rem;
          border-bottom: 1px solid #e5e7eb;
          padding-bottom: 0.5rem;
        }
        .filter-tabs button {
          padding: 0.5rem 1rem;
          border: none;
          background: transparent;
          cursor: pointer;
          font-weight: 500;
          color: #6b7280;
          border-radius: 6px 6px 0 0;
        }
        .filter-tabs button.active {
          color: #673AB7;
          background: rgba(103, 58, 183, 0.1);
          border-bottom: 2px solid #673AB7;
        }
        .changeset-list {
          display: flex;
          flex-direction: column;
          gap: 1rem;
        }
        .changeset-card {
          background: white;
          border: 1px solid #e5e7eb;
          border-radius: 12px;
          padding: 1.25rem;
        }
        .cs-header {
          display: flex;
          align-items: center;
          gap: 0.75rem;
          margin-bottom: 0.75rem;
        }
        .cs-title {
          font-weight: 600;
          font-size: 1.1rem;
          flex: 1;
        }
        .cs-type {
          font-size: 0.75rem;
          padding: 0.25rem 0.5rem;
          background: #f3f4f6;
          border-radius: 4px;
          color: #6b7280;
        }
        .cs-description {
          color: #4b5563;
          margin-bottom: 1rem;
          font-size: 0.9rem;
        }
        .cs-meta {
          display: flex;
          gap: 2rem;
          font-size: 0.85rem;
          color: #6b7280;
          margin-bottom: 1rem;
        }
        .cs-affected {
          display: flex;
          flex-wrap: wrap;
          gap: 0.5rem;
          margin-bottom: 1rem;
        }
        .affected-tag {
          font-size: 0.75rem;
          padding: 0.25rem 0.5rem;
          background: #EDE7F6;
          color: #673AB7;
          border-radius: 4px;
        }
        .cs-footer {
          display: flex;
          justify-content: space-between;
          align-items: center;
          padding-top: 0.75rem;
          border-top: 1px solid #f3f4f6;
        }
        .approval-status {
          font-size: 0.85rem;
        }
        .cs-actions {
          display: flex;
          gap: 0.5rem;
        }
        .approve-btn {
          padding: 0.5rem 1rem;
          background: #16a34a;
          color: white;
          border: none;
          border-radius: 6px;
          cursor: pointer;
          font-weight: 500;
        }
        .reject-btn {
          padding: 0.5rem 1rem;
          background: #dc2626;
          color: white;
          border: none;
          border-radius: 6px;
          cursor: pointer;
          font-weight: 500;
        }
        .view-btn {
          padding: 0.5rem 1rem;
          background: transparent;
          color: #673AB7;
          border: 1px solid #673AB7;
          border-radius: 6px;
          cursor: pointer;
          font-weight: 500;
        }
        .risk-score {
          display: inline-flex;
          align-items: center;
          gap: 0.25rem;
          font-weight: 500;
        }
        .cs-semantic-impact {
          background: #f8fafc;
          border: 1px dashed #e2e8f0;
          border-radius: 8px;
          padding: 0.75rem;
          margin-bottom: 1rem;
        }
        .impact-label {
          font-size: 0.75rem;
          font-weight: 600;
          color: #64748b;
          text-transform: uppercase;
          margin-bottom: 0.5rem;
        }
        .impact-details {
          display: flex;
          gap: 0.5rem;
        }
        .impact-badge {
          font-size: 0.7rem;
          padding: 0.15rem 0.4rem;
          border-radius: 4px;
          font-weight: 500;
        }
        .impact-badge.bo { background: #dcfce7; color: #166534; }
        .impact-badge.job { background: #fee2e2; color: #991b1b; }
        .impact-badge.api { background: #e0f2fe; color: #075985; }
      `}</style>

      <header className="governance-header">
        <PolicyIcon style={{ fontSize: '2rem', color: '#673AB7' }} />
        <div>
          <h1>Governance Console</h1>
          <p>Review and approve change requests</p>
        </div>
      </header>

      <div className="filter-tabs">
        {['pending', 'approved', 'rejected', 'all'].map(f => (
          <button
            key={f}
            className={filter === f ? 'active' : ''}
            onClick={() => setFilter(f)}
          >
            {f.charAt(0).toUpperCase() + f.slice(1)}
          </button>
        ))}
      </div>

      <div className="changeset-list">
        {filteredChangeSets.length === 0 ? (
          <div className="empty-state">
            <CheckCircleIcon style={{ fontSize: '3rem', color: '#16a34a' }} />
            <p>No change sets in this category</p>
          </div>
        ) : (
          filteredChangeSets.map(cs => (
            <div key={cs.id} className="changeset-card">
              <div className="cs-header">
                {getStatusIcon(cs.status)}
                <span className="cs-title">{cs.title}</span>
                <span className="cs-type">{cs.type.replace('_', ' ')}</span>
                <span className="risk-score" style={{ color: getRiskColor(cs.riskScore) }}>
                  <WarningIcon style={{ fontSize: '1rem' }} />
                  {Math.round(cs.riskScore * 100)}% risk
                </span>
              </div>

              <p className="cs-description">{cs.description}</p>

              <div className="cs-affected">
                {cs.affectedItems.map(item => (
                  <span key={item} className="affected-tag">{item}</span>
                ))}
              </div>

              <div className="cs-meta">
                <span>By: {cs.createdBy}</span>
                <span>{new Date(cs.createdAt).toLocaleString()}</span>
              </div>

              {cs.semanticImpact && (
                <div className="cs-semantic-impact">
                  <div className="impact-label">Semantic Impact Analysis:</div>
                  <div className="impact-details">
                    {cs.semanticImpact.affected_bos && cs.semanticImpact.affected_bos.length > 0 && (
                      <span className="impact-badge bo">
                        {cs.semanticImpact.affected_bos.length} BOs
                      </span>
                    )}
                    {cs.semanticImpact.downstream_jobs && cs.semanticImpact.downstream_jobs.length > 0 && (
                      <span className="impact-badge job">
                        {cs.semanticImpact.downstream_jobs.length} Downstream Jobs
                      </span>
                    )}
                    {cs.semanticImpact.affected_apis && cs.semanticImpact.affected_apis.length > 0 && (
                      <span className="impact-badge api">
                        {cs.semanticImpact.affected_apis.length} APIs
                      </span>
                    )}
                  </div>
                </div>
              )}

              <div className="cs-footer">
                <span className="approval-status">
                  {cs.currentApprovals} / {cs.requiredApprovers} approvals
                </span>
                <div className="cs-actions">
                  <button className="view-btn">
                    <VisibilityIcon style={{ fontSize: '1rem' }} /> View Details
                  </button>
                  {cs.status === 'pending' && (
                    <>
                      <button className="approve-btn" onClick={() => handleApprove(cs.id)}>
                        Approve
                      </button>
                      <button className="reject-btn">Reject</button>
                    </>
                  )}
                </div>
              </div>
            </div>
          ))
        )}
      </div>
    </div>
  );
};

export default GovernanceConsolePage;
