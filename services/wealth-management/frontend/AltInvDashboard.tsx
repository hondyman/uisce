import React, { useState, useCallback } from 'react';
import { useSubscription, useQuery, useMutation, gql } from '@apollo/client';
import type {
  AlternativesAdvisorDashboard,
  InvestmentOpportunity,
  NextAction,
  RiskAlert,
  PipelineUpdateEvent,
  AllocationRecommendation,
  InvestmentStage,
  ActionPriority,
} from './types/altinv.types';

// ============================================================
// GraphQL Queries and Mutations
// ============================================================

const DASHBOARD_QUERY = gql`
  query AlternativesDashboard($portfolioId: ID!) {
    alternativesDashboard(portfolioId: $portfolioId) {
      pipeline_overview {
        total_opportunities
        by_stage {
          stage
          count
          value_millions
        }
        total_pipeline_value_millions
        avg_time_to_decision_days
        conversion_rate
      }
      portfolio_health {
        total_aum_millions
        alternatives_allocation_percent
        target_allocation_percent
        unfunded_commitments
        available_liquidity
        concentration_metrics {
          largest_position_percent
          top_5_concentration_percent
        }
      }
      performance_attribution {
        total_portfolio_irr
        alternatives_irr
        benchmark_irr
        alpha
      }
      risk_monitoring {
        overall_risk_score
        risk_trend
        alerts {
          id
          alert_type
          severity
          title
          description
          acknowledged
        }
        liquidity_coverage {
          coverage_ratio
          status
        }
      }
      next_actions {
        id
        action_type
        priority
        title
        description
        due_date
        status
      }
      last_updated
    }
  }
`;

const PIPELINE_SUBSCRIPTION = gql`
  subscription PipelineUpdates($portfolioId: ID!) {
    pipelineUpdate(portfolioId: $portfolioId) {
      event_type
      opportunity_id
      opportunity_name
      current_stage
      timestamp
      actor
    }
  }
`;

const OPPORTUNITIES_QUERY = gql`
  query Opportunities($stage: String, $assetClass: String) {
    investmentOpportunities(stage: $stage, assetClass: $assetClass) {
      id
      fund_name
      asset_class
      vintage_year
      target_size_millions
      expected_irr
      expected_moic
      stage
      created_at
    }
  }
`;

const APPROVE_ALLOCATION = gql`
  mutation ApproveAllocation($recommendationId: ID!, $amount: Float!, $notes: String) {
    approveAllocation(recommendationId: $recommendationId, amount: $amount, notes: $notes) {
      id
      status
    }
  }
`;

const ACKNOWLEDGE_ALERT = gql`
  mutation AcknowledgeAlert($alertId: ID!) {
    acknowledgeRiskAlert(alertId: $alertId) {
      id
      acknowledged
    }
  }
`;

// ============================================================
// Helper Components
// ============================================================

const StageIndicator: React.FC<{ stage: InvestmentStage }> = ({ stage }) => {
  const stageColors: Record<InvestmentStage, string> = {
    sourced: '#6b7280',
    screening: '#3b82f6',
    due_diligence: '#f59e0b',
    committee_review: '#8b5cf6',
    approved: '#10b981',
    rejected: '#ef4444',
    invested: '#059669',
  };

  const stageLabels: Record<InvestmentStage, string> = {
    sourced: 'Sourced',
    screening: 'Screening',
    due_diligence: 'Due Diligence',
    committee_review: 'Committee',
    approved: 'Approved',
    rejected: 'Rejected',
    invested: 'Invested',
  };

  return (
    <span
      className="stage-badge"
      style={{
        backgroundColor: stageColors[stage],
        color: 'white',
        padding: '2px 8px',
        borderRadius: '4px',
        fontSize: '12px',
        fontWeight: 500,
      }}
    >
      {stageLabels[stage]}
    </span>
  );
};

const PriorityBadge: React.FC<{ priority: ActionPriority }> = ({ priority }) => {
  const colors: Record<ActionPriority, { bg: string; text: string }> = {
    urgent: { bg: '#fef2f2', text: '#dc2626' },
    high: { bg: '#fef3c7', text: '#d97706' },
    normal: { bg: '#dbeafe', text: '#2563eb' },
    low: { bg: '#f3f4f6', text: '#6b7280' },
  };

  return (
    <span
      style={{
        backgroundColor: colors[priority].bg,
        color: colors[priority].text,
        padding: '2px 6px',
        borderRadius: '4px',
        fontSize: '11px',
        fontWeight: 600,
        textTransform: 'uppercase',
      }}
    >
      {priority}
    </span>
  );
};

const MetricCard: React.FC<{
  title: string;
  value: string | number;
  subtitle?: string;
  trend?: 'up' | 'down' | 'neutral';
  format?: 'currency' | 'percent' | 'number';
}> = ({ title, value, subtitle, trend, format }) => {
  const formatValue = (v: string | number): string => {
    if (typeof v === 'string') return v;
    switch (format) {
      case 'currency':
        return `$${v.toLocaleString()}M`;
      case 'percent':
        return `${v.toFixed(1)}%`;
      default:
        return v.toLocaleString();
    }
  };

  const trendIcon = trend === 'up' ? '↑' : trend === 'down' ? '↓' : '';
  const trendColor = trend === 'up' ? '#10b981' : trend === 'down' ? '#ef4444' : '#6b7280';

  return (
    <div className="metric-card" style={{
      background: 'white',
      borderRadius: '8px',
      padding: '16px',
      boxShadow: '0 1px 3px rgba(0,0,0,0.1)',
    }}>
      <div style={{ color: '#6b7280', fontSize: '12px', marginBottom: '4px' }}>{title}</div>
      <div style={{ fontSize: '24px', fontWeight: 600, display: 'flex', alignItems: 'center', gap: '8px' }}>
        {formatValue(value)}
        {trend && <span style={{ color: trendColor, fontSize: '14px' }}>{trendIcon}</span>}
      </div>
      {subtitle && <div style={{ color: '#9ca3af', fontSize: '11px', marginTop: '4px' }}>{subtitle}</div>}
    </div>
  );
};

const RiskScoreGauge: React.FC<{ score: number; trend: 'improving' | 'stable' | 'deteriorating' }> = ({ score, trend }) => {
  const getColor = (s: number): string => {
    if (s <= 30) return '#10b981';
    if (s <= 60) return '#f59e0b';
    return '#ef4444';
  };

  const trendIcon = trend === 'improving' ? '📈' : trend === 'deteriorating' ? '📉' : '➡️';

  return (
    <div style={{ textAlign: 'center' }}>
      <div style={{
        width: '100px',
        height: '100px',
        borderRadius: '50%',
        border: `8px solid ${getColor(score)}`,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        margin: '0 auto',
        background: 'white',
      }}>
        <span style={{ fontSize: '28px', fontWeight: 700 }}>{score}</span>
      </div>
      <div style={{ marginTop: '8px', fontSize: '14px' }}>
        {trendIcon} {trend.charAt(0).toUpperCase() + trend.slice(1)}
      </div>
    </div>
  );
};

// ============================================================
// Section Components
// ============================================================

const PipelineSection: React.FC<{
  overview: AlternativesAdvisorDashboard['pipeline_overview'];
  onViewOpportunity: (id: string) => void;
}> = ({ overview, onViewOpportunity }) => {
  return (
    <div className="pipeline-section" style={{ marginBottom: '24px' }}>
      <h2 style={{ fontSize: '18px', fontWeight: 600, marginBottom: '16px' }}>📊 Pipeline Overview</h2>
      
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: '16px', marginBottom: '20px' }}>
        <MetricCard title="Active Opportunities" value={overview.total_opportunities} />
        <MetricCard title="Pipeline Value" value={overview.total_pipeline_value_millions} format="currency" />
        <MetricCard title="Avg. Decision Time" value={`${overview.avg_time_to_decision_days} days`} />
        <MetricCard title="Conversion Rate" value={overview.conversion_rate * 100} format="percent" />
      </div>

      <div style={{ background: 'white', borderRadius: '8px', padding: '16px', boxShadow: '0 1px 3px rgba(0,0,0,0.1)' }}>
        <h3 style={{ fontSize: '14px', fontWeight: 600, marginBottom: '12px' }}>Stage Distribution</h3>
        <div style={{ display: 'flex', gap: '8px', flexWrap: 'wrap' }}>
          {overview.by_stage.map((stage) => (
            <div
              key={stage.stage}
              style={{
                padding: '12px 16px',
                background: '#f9fafb',
                borderRadius: '6px',
                textAlign: 'center',
                minWidth: '120px',
              }}
            >
              <StageIndicator stage={stage.stage} />
              <div style={{ marginTop: '8px', fontSize: '20px', fontWeight: 600 }}>{stage.count}</div>
              <div style={{ color: '#6b7280', fontSize: '11px' }}>${stage.value_millions}M</div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
};

const PortfolioHealthSection: React.FC<{
  health: AlternativesAdvisorDashboard['portfolio_health'];
}> = ({ health }) => {
  const allocationVariance = health.alternatives_allocation_percent - health.target_allocation_percent;
  const varianceTrend = Math.abs(allocationVariance) < 2 ? 'neutral' : allocationVariance > 0 ? 'up' : 'down';

  return (
    <div className="portfolio-health-section" style={{ marginBottom: '24px' }}>
      <h2 style={{ fontSize: '18px', fontWeight: 600, marginBottom: '16px' }}>💼 Portfolio Health</h2>
      
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: '16px' }}>
        <MetricCard 
          title="Total AUM" 
          value={health.total_aum_millions} 
          format="currency" 
        />
        <MetricCard 
          title="Alternatives Allocation" 
          value={health.alternatives_allocation_percent} 
          format="percent"
          subtitle={`Target: ${health.target_allocation_percent}%`}
          trend={varianceTrend}
        />
        <MetricCard 
          title="Unfunded Commitments" 
          value={health.unfunded_commitments / 1_000_000} 
          format="currency"
        />
        <MetricCard 
          title="Available Liquidity" 
          value={health.available_liquidity / 1_000_000} 
          format="currency"
        />
      </div>

      <div style={{ 
        marginTop: '16px', 
        background: 'white', 
        borderRadius: '8px', 
        padding: '16px',
        boxShadow: '0 1px 3px rgba(0,0,0,0.1)',
      }}>
        <h3 style={{ fontSize: '14px', fontWeight: 600, marginBottom: '12px' }}>Concentration Metrics</h3>
        <div style={{ display: 'flex', gap: '24px' }}>
          <div>
            <span style={{ color: '#6b7280', fontSize: '12px' }}>Largest Position:</span>
            <strong style={{ marginLeft: '8px' }}>{health.concentration_metrics.largest_position_percent}%</strong>
          </div>
          <div>
            <span style={{ color: '#6b7280', fontSize: '12px' }}>Top 5 Concentration:</span>
            <strong style={{ marginLeft: '8px' }}>{health.concentration_metrics.top_5_concentration_percent}%</strong>
          </div>
        </div>
      </div>
    </div>
  );
};

const PerformanceSection: React.FC<{
  attribution: AlternativesAdvisorDashboard['performance_attribution'];
}> = ({ attribution }) => {
  return (
    <div className="performance-section" style={{ marginBottom: '24px' }}>
      <h2 style={{ fontSize: '18px', fontWeight: 600, marginBottom: '16px' }}>📈 Performance Attribution</h2>
      
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: '16px' }}>
        <MetricCard 
          title="Portfolio IRR" 
          value={attribution.total_portfolio_irr} 
          format="percent"
        />
        <MetricCard 
          title="Alternatives IRR" 
          value={attribution.alternatives_irr} 
          format="percent"
        />
        <MetricCard 
          title="Benchmark IRR" 
          value={attribution.benchmark_irr} 
          format="percent"
        />
        <MetricCard 
          title="Alpha" 
          value={attribution.alpha} 
          format="percent"
          trend={attribution.alpha > 0 ? 'up' : attribution.alpha < 0 ? 'down' : 'neutral'}
        />
      </div>
    </div>
  );
};

const RiskMonitoringSection: React.FC<{
  monitoring: AlternativesAdvisorDashboard['risk_monitoring'];
  onAcknowledgeAlert: (alertId: string) => void;
}> = ({ monitoring, onAcknowledgeAlert }) => {
  const criticalAlerts = monitoring.alerts.filter(a => a.severity === 'critical' && !a.acknowledged);
  const warningAlerts = monitoring.alerts.filter(a => a.severity === 'warning' && !a.acknowledged);

  return (
    <div className="risk-monitoring-section" style={{ marginBottom: '24px' }}>
      <h2 style={{ fontSize: '18px', fontWeight: 600, marginBottom: '16px' }}>⚠️ Risk Monitoring</h2>
      
      <div style={{ display: 'grid', gridTemplateColumns: '200px 1fr', gap: '24px' }}>
        <div style={{ 
          background: 'white', 
          borderRadius: '8px', 
          padding: '20px',
          boxShadow: '0 1px 3px rgba(0,0,0,0.1)',
        }}>
          <h3 style={{ fontSize: '14px', fontWeight: 600, marginBottom: '16px', textAlign: 'center' }}>
            Risk Score
          </h3>
          <RiskScoreGauge score={monitoring.overall_risk_score} trend={monitoring.risk_trend} />
          <div style={{ marginTop: '16px', textAlign: 'center' }}>
            <div style={{ fontSize: '12px', color: '#6b7280' }}>
              Liquidity: {monitoring.liquidity_coverage.status.toUpperCase()}
            </div>
            <div style={{ fontSize: '11px', color: '#9ca3af' }}>
              Coverage: {monitoring.liquidity_coverage.coverage_ratio.toFixed(1)}x
            </div>
          </div>
        </div>

        <div style={{ 
          background: 'white', 
          borderRadius: '8px', 
          padding: '16px',
          boxShadow: '0 1px 3px rgba(0,0,0,0.1)',
        }}>
          <h3 style={{ fontSize: '14px', fontWeight: 600, marginBottom: '12px' }}>
            Active Alerts ({criticalAlerts.length + warningAlerts.length})
          </h3>
          
          {criticalAlerts.length === 0 && warningAlerts.length === 0 ? (
            <div style={{ color: '#10b981', padding: '20px', textAlign: 'center' }}>
              ✅ No active alerts
            </div>
          ) : (
            <div style={{ maxHeight: '200px', overflowY: 'auto' }}>
              {[...criticalAlerts, ...warningAlerts].map((alert) => (
                <div
                  key={alert.id}
                  style={{
                    padding: '12px',
                    borderLeft: `4px solid ${alert.severity === 'critical' ? '#ef4444' : '#f59e0b'}`,
                    background: alert.severity === 'critical' ? '#fef2f2' : '#fffbeb',
                    marginBottom: '8px',
                    borderRadius: '0 4px 4px 0',
                  }}
                >
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'start' }}>
                    <div>
                      <div style={{ fontWeight: 600, fontSize: '13px' }}>{alert.title}</div>
                      <div style={{ fontSize: '12px', color: '#6b7280', marginTop: '4px' }}>
                        {alert.description}
                      </div>
                    </div>
                    <button
                      onClick={() => onAcknowledgeAlert(alert.id)}
                      style={{
                        background: 'white',
                        border: '1px solid #d1d5db',
                        borderRadius: '4px',
                        padding: '4px 8px',
                        fontSize: '11px',
                        cursor: 'pointer',
                      }}
                    >
                      Acknowledge
                    </button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

const NextActionsSection: React.FC<{
  actions: NextAction[];
  onActionClick: (action: NextAction) => void;
}> = ({ actions, onActionClick }) => {
  const sortedActions = [...actions].sort((a, b) => {
    const priorityOrder: Record<ActionPriority, number> = { urgent: 0, high: 1, normal: 2, low: 3 };
    return priorityOrder[a.priority] - priorityOrder[b.priority];
  });

  const getActionIcon = (type: NextAction['action_type']): string => {
    const icons: Record<NextAction['action_type'], string> = {
      review_opportunity: '🔍',
      committee_meeting: '👥',
      capital_call: '💰',
      compliance_filing: '📋',
      rebalance: '⚖️',
      client_meeting: '🤝',
    };
    return icons[type] || '📌';
  };

  return (
    <div className="next-actions-section">
      <h2 style={{ fontSize: '18px', fontWeight: 600, marginBottom: '16px' }}>📋 Next Actions</h2>
      
      <div style={{ 
        background: 'white', 
        borderRadius: '8px',
        boxShadow: '0 1px 3px rgba(0,0,0,0.1)',
        overflow: 'hidden',
      }}>
        {sortedActions.length === 0 ? (
          <div style={{ padding: '40px', textAlign: 'center', color: '#6b7280' }}>
            No pending actions
          </div>
        ) : (
          sortedActions.slice(0, 10).map((action, index) => (
            <div
              key={action.id}
              onClick={() => onActionClick(action)}
              style={{
                padding: '16px',
                borderBottom: index < sortedActions.length - 1 ? '1px solid #f3f4f6' : 'none',
                cursor: 'pointer',
                transition: 'background 0.15s',
              }}
              onMouseEnter={(e) => (e.currentTarget.style.background = '#f9fafb')}
              onMouseLeave={(e) => (e.currentTarget.style.background = 'white')}
            >
              <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                <span style={{ fontSize: '20px' }}>{getActionIcon(action.action_type)}</span>
                <div style={{ flex: 1 }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                    <span style={{ fontWeight: 600, fontSize: '14px' }}>{action.title}</span>
                    <PriorityBadge priority={action.priority} />
                  </div>
                  <div style={{ color: '#6b7280', fontSize: '12px', marginTop: '4px' }}>
                    {action.description}
                  </div>
                </div>
                <div style={{ textAlign: 'right' }}>
                  <div style={{ fontSize: '12px', color: '#6b7280' }}>
                    Due: {new Date(action.due_date).toLocaleDateString()}
                  </div>
                  {action.status === 'overdue' && (
                    <span style={{ color: '#ef4444', fontSize: '11px', fontWeight: 600 }}>OVERDUE</span>
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

// ============================================================
// Main Dashboard Component
// ============================================================

interface AltInvDashboardProps {
  portfolioId: string;
}

const AltInvDashboard: React.FC<AltInvDashboardProps> = ({ portfolioId }) => {
  const [selectedView, setSelectedView] = useState<'dashboard' | 'pipeline' | 'allocations'>('dashboard');

  // Main dashboard query
  const { data, loading, error, refetch } = useQuery<{ alternativesDashboard: AlternativesAdvisorDashboard }>(
    DASHBOARD_QUERY,
    { variables: { portfolioId }, pollInterval: 30000 }
  );

  // Real-time pipeline updates
  const { data: pipelineUpdate } = useSubscription<{ pipelineUpdate: PipelineUpdateEvent }>(
    PIPELINE_SUBSCRIPTION,
    { variables: { portfolioId } }
  );

  // Mutations
  const [acknowledgeAlert] = useMutation(ACKNOWLEDGE_ALERT, {
    onCompleted: () => refetch(),
  });

  const handleAcknowledgeAlert = useCallback((alertId: string) => {
    acknowledgeAlert({ variables: { alertId } });
  }, [acknowledgeAlert]);

  const handleViewOpportunity = useCallback((opportunityId: string) => {
    // Navigate to opportunity detail
    window.location.href = `/alternatives/opportunities/${opportunityId}`;
  }, []);

  const handleActionClick = useCallback((action: NextAction) => {
    // Navigate based on action type
    if (action.related_entity_id) {
      switch (action.action_type) {
        case 'review_opportunity':
          window.location.href = `/alternatives/opportunities/${action.related_entity_id}`;
          break;
        case 'capital_call':
          window.location.href = `/alternatives/capital-calls/${action.related_entity_id}`;
          break;
        case 'compliance_filing':
          window.location.href = `/compliance/filings/${action.related_entity_id}`;
          break;
        default:
          console.log('Action clicked:', action);
      }
    }
  }, []);

  if (loading) {
    return (
      <div style={{ padding: '40px', textAlign: 'center' }}>
        <div style={{ fontSize: '24px', marginBottom: '16px' }}>⏳</div>
        <div>Loading Alternative Investments Dashboard...</div>
      </div>
    );
  }

  if (error) {
    return (
      <div style={{ padding: '40px', textAlign: 'center', color: '#ef4444' }}>
        <div style={{ fontSize: '24px', marginBottom: '16px' }}>❌</div>
        <div>Error loading dashboard: {error.message}</div>
        <button
          onClick={() => refetch()}
          style={{
            marginTop: '16px',
            padding: '8px 16px',
            background: '#3b82f6',
            color: 'white',
            border: 'none',
            borderRadius: '6px',
            cursor: 'pointer',
          }}
        >
          Retry
        </button>
      </div>
    );
  }

  const dashboard = data?.alternativesDashboard;
  if (!dashboard) return null;

  return (
    <div className="altinv-dashboard" style={{ padding: '24px', background: '#f9fafb', minHeight: '100vh' }}>
      {/* Header */}
      <div style={{ 
        display: 'flex', 
        justifyContent: 'space-between', 
        alignItems: 'center',
        marginBottom: '24px',
      }}>
        <div>
          <h1 style={{ fontSize: '24px', fontWeight: 700, margin: 0 }}>
            🏛️ Alternative Investments Dashboard
          </h1>
          <div style={{ color: '#6b7280', fontSize: '12px', marginTop: '4px' }}>
            Last updated: {new Date(dashboard.last_updated).toLocaleString()}
          </div>
        </div>

        <div style={{ display: 'flex', gap: '8px' }}>
          {(['dashboard', 'pipeline', 'allocations'] as const).map((view) => (
            <button
              key={view}
              onClick={() => setSelectedView(view)}
              style={{
                padding: '8px 16px',
                background: selectedView === view ? '#3b82f6' : 'white',
                color: selectedView === view ? 'white' : '#374151',
                border: '1px solid #d1d5db',
                borderRadius: '6px',
                cursor: 'pointer',
                fontWeight: 500,
                textTransform: 'capitalize',
              }}
            >
              {view}
            </button>
          ))}
        </div>
      </div>

      {/* Real-time update toast */}
      {pipelineUpdate?.pipelineUpdate && (
        <div style={{
          background: '#dbeafe',
          border: '1px solid #93c5fd',
          borderRadius: '8px',
          padding: '12px 16px',
          marginBottom: '16px',
          display: 'flex',
          alignItems: 'center',
          gap: '12px',
        }}>
          <span>🔔</span>
          <span>
            <strong>{pipelineUpdate.pipelineUpdate.opportunity_name}</strong> moved to{' '}
            <StageIndicator stage={pipelineUpdate.pipelineUpdate.current_stage} />
          </span>
          <span style={{ color: '#6b7280', fontSize: '12px', marginLeft: 'auto' }}>
            by {pipelineUpdate.pipelineUpdate.actor}
          </span>
        </div>
      )}

      {/* Dashboard Sections */}
      <PipelineSection 
        overview={dashboard.pipeline_overview} 
        onViewOpportunity={handleViewOpportunity}
      />
      
      <PortfolioHealthSection health={dashboard.portfolio_health} />
      
      <PerformanceSection attribution={dashboard.performance_attribution} />
      
      <RiskMonitoringSection 
        monitoring={dashboard.risk_monitoring}
        onAcknowledgeAlert={handleAcknowledgeAlert}
      />
      
      <NextActionsSection 
        actions={dashboard.next_actions}
        onActionClick={handleActionClick}
      />

      {/* Quick Actions FAB */}
      <div style={{
        position: 'fixed',
        bottom: '24px',
        right: '24px',
        display: 'flex',
        flexDirection: 'column',
        gap: '8px',
      }}>
        <button
          onClick={() => window.location.href = '/alternatives/opportunities/new'}
          style={{
            width: '56px',
            height: '56px',
            borderRadius: '50%',
            background: '#3b82f6',
            color: 'white',
            border: 'none',
            fontSize: '24px',
            cursor: 'pointer',
            boxShadow: '0 4px 12px rgba(59, 130, 246, 0.4)',
          }}
          title="Add New Opportunity"
        >
          +
        </button>
      </div>
    </div>
  );
};

export default AltInvDashboard;
