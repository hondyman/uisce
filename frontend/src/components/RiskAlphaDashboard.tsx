import React, { useMemo } from 'react';
import { useSubscription, useMutation, gql } from '@apollo/client';
import {
  AlertTriangle,
  Shield,
  TrendingDown,
  Activity,
  CheckCircle,
  Clock,
  Zap,
  BarChart3,
  TrendingUp,
} from 'lucide-react';

/**
 * RiskAlphaDashboard Component
 * Real-time portfolio risk monitoring with AI-powered detection and auto-mitigation
 * 
 * Integrates with:
 * - Hasura GraphQL subscriptions (real-time risk_events)
 * - Temporal workflows (Risk Alpha business process)
 * - xAI Grok (comprehensive risk analysis)
 * - ABAC (authorization for mitigation)
 */

// GraphQL Queries & Subscriptions
const PORTFOLIO_RISK_DASHBOARD_SUBSCRIPTION = gql`
  subscription PortfolioRiskDashboard($tenantId: uuid!) {
    v_portfolio_risk_dashboard(
      where: { tenant_id: { _eq: $tenantId } }
      order_by: { current_risk_score: desc_nulls_last }
    ) {
      portfolio_entity_id
      portfolio_name
      current_risk_score
      active_alerts
      critical_alerts
      mitigated_last_30d
      auto_mitigation_rate
      var_95
      cvar_95
      sharpe_ratio
      liquidity_ratio
      top_10_concentration
      herfindahl_index
      latest_risk_event
    }
  }
`;

const RISK_EVENTS_SUBSCRIPTION = gql`
  subscription RiskEvents($tenantId: uuid!) {
    risk_events(
      where: {
        tenant_id: { _eq: $tenantId }
        status: { _in: ["DETECTED", "ACKNOWLEDGED", "MITIGATING"] }
      }
      order_by: { detected_at: desc }
      limit: 50
    ) {
      id
      portfolio_entity_id
      event_type
      severity
      risk_score
      confidence_score
      var_95
      cvar_95
      status
      ai_reasoning
      ai_recommendations
      detected_at
      auto_mitigated
      mitigation_actions
      workflow_id
    }
  }
`;

const TRIGGER_RISK_ANALYSIS = gql`
  mutation TriggerRiskAnalysis($businessProcessId: uuid!, $portfolioId: uuid!) {
    executeBusinessProcess(processId: $businessProcessId, input: { portfolio_id: $portfolioId }) {
      execution_id
      status
    }
  }
`;

interface Portfolio {
  portfolio_entity_id: string;
  portfolio_name: string;
  current_risk_score: number;
  active_alerts: number;
  critical_alerts: number;
  mitigated_last_30d: number;
  auto_mitigation_rate: number;
  var_95?: number;
  cvar_95?: number;
  sharpe_ratio?: number;
  liquidity_ratio?: number;
  top_10_concentration?: number;
  herfindahl_index?: number;
  latest_risk_event?: any;
}

interface RiskEvent {
  id: string;
  portfolio_entity_id: string;
  event_type: string;
  severity: 'CRITICAL' | 'HIGH' | 'MEDIUM' | 'LOW';
  risk_score: number;
  confidence_score: number;
  status: string;
  ai_reasoning: string;
  detected_at: string;
  auto_mitigated: boolean;
  mitigation_actions?: string[];
  workflow_id?: string;
}

export const RiskAlphaDashboard: React.FC<{ tenantId: string }> = ({ tenantId }) => {
  const [selectedPortfolio, setSelectedPortfolio] = React.useState<string | null>(null);

  // Real-time subscriptions
  const { data: dashboardData, loading: dashboardLoading } = useSubscription(
    PORTFOLIO_RISK_DASHBOARD_SUBSCRIPTION,
    { variables: { tenantId } }
  );

  const { data: eventsData, loading: eventsLoading } = useSubscription(
    RISK_EVENTS_SUBSCRIPTION,
    { variables: { tenantId } }
  );

  const [triggerAnalysis, { loading: triggering }] = useMutation(TRIGGER_RISK_ANALYSIS);

  const portfolios = (dashboardData?.v_portfolio_risk_dashboard || []) as Portfolio[];
  const riskEvents = (eventsData?.risk_events || []) as RiskEvent[];

  // Compute dashboard metrics
  const metrics = useMemo(() => {
    return {
      totalPortfolios: portfolios.length,
      avgRiskScore: portfolios.reduce((sum, p) => sum + (p.current_risk_score || 0), 0) / (portfolios.length || 1),
      totalAlerts: riskEvents.length,
      criticalAlerts: riskEvents.filter(e => e.severity === 'CRITICAL').length,
      highAlerts: riskEvents.filter(e => e.severity === 'HIGH').length,
      autoMitigatedCount: riskEvents.filter(e => e.auto_mitigated).length,
      autoMitigationRate: portfolios.reduce((sum, p) => sum + (p.auto_mitigation_rate || 0), 0) / (portfolios.length || 1),
    };
  }, [portfolios, riskEvents]);

  const handleTriggerAnalysis = async (portfolioId: string) => {
    try {
      await triggerAnalysis({
        variables: {
          businessProcessId: 'risk_alpha_v1',
          portfolioId,
        },
      });
      // Toast notification here
    } catch (error) {
      console.error('Failed to trigger analysis:', error);
    }
  };

  if (dashboardLoading || eventsLoading) {
    return <div className="p-4 text-center">Loading risk dashboard...</div>;
  }

  return (
    <div className="risk-alpha-dashboard p-6 bg-gray-50">
      {/* Header */}
      <div className="mb-8">
        <h1 className="text-3xl font-bold text-gray-900 flex items-center gap-2">
          <Shield className="text-blue-600" size={32} />
          Risk Management Alpha
        </h1>
        <p className="text-gray-600 mt-1">AI-powered portfolio risk detection & automated mitigation</p>
      </div>

      {/* Key Metrics Cards */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-8">
        <MetricCard
          icon={<BarChart3 className="text-blue-500" size={24} />}
          label="Avg Risk Score"
          value={metrics.avgRiskScore.toFixed(2)}
          suffix="/10"
          status={metrics.avgRiskScore > 7 ? 'critical' : metrics.avgRiskScore > 5 ? 'warning' : 'normal'}
        />

        <MetricCard
          icon={<AlertTriangle className="text-red-500" size={24} />}
          label="Active Alerts"
          value={metrics.totalAlerts}
          status={metrics.totalAlerts > 10 ? 'critical' : 'normal'}
        />

        <MetricCard
          icon={<Zap className="text-yellow-500" size={24} />}
          label="Critical Alerts"
          value={metrics.criticalAlerts}
          status={metrics.criticalAlerts > 0 ? 'critical' : 'normal'}
        />

        <MetricCard
          icon={<CheckCircle className="text-green-500" size={24} />}
          label="Auto-Mitigation Rate"
          value={`${(metrics.autoMitigationRate * 100).toFixed(0)}%`}
          status="success"
        />
      </div>

      {/* Portfolio Grid */}
      <div className="mb-8">
        <h2 className="text-xl font-semibold text-gray-900 mb-4">Portfolios at Risk</h2>
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
          {portfolios.map((portfolio) => (
            <PortfolioCard
              key={portfolio.portfolio_entity_id}
              portfolio={portfolio}
              onAnalyze={() => handleTriggerAnalysis(portfolio.portfolio_entity_id)}
              isSelected={selectedPortfolio === portfolio.portfolio_entity_id}
              onSelect={() => setSelectedPortfolio(portfolio.portfolio_entity_id)}
              isAnalyzing={triggering}
            />
          ))}
        </div>
      </div>

      {/* Risk Events List */}
      <div>
        <h2 className="text-xl font-semibold text-gray-900 mb-4">Active Risk Events</h2>
        <div className="space-y-3 max-h-96 overflow-y-auto">
          {riskEvents.length === 0 ? (
            <div className="p-6 bg-green-50 border border-green-200 rounded-lg text-center">
              <CheckCircle className="mx-auto text-green-600 mb-2" size={32} />
              <p className="text-green-800 font-medium">No Active Risk Events</p>
              <p className="text-green-700 text-sm">All portfolios are within acceptable risk thresholds</p>
            </div>
          ) : (
            riskEvents.map((event) => (
              <RiskEventRow key={event.id} event={event} />
            ))
          )}
        </div>
      </div>
    </div>
  );
};

// Portfolio Risk Card
interface PortfolioCardProps {
  portfolio: Portfolio;
  onAnalyze: () => void;
  isSelected: boolean;
  onSelect: () => void;
  isAnalyzing: boolean;
}

const PortfolioCard: React.FC<PortfolioCardProps> = ({
  portfolio,
  onAnalyze,
  isSelected,
  onSelect,
  isAnalyzing,
}) => {
  const riskLevel = getRiskLevel(portfolio.current_risk_score || 0);
  const riskColors = {
    critical: 'bg-red-50 border-red-300',
    high: 'bg-orange-50 border-orange-300',
    medium: 'bg-yellow-50 border-yellow-300',
    low: 'bg-green-50 border-green-300',
  };

  return (
    <div
      className={`p-4 border-2 rounded-lg cursor-pointer transition ${
        isSelected ? 'ring-2 ring-blue-500' : ''
      } ${riskColors[riskLevel] || riskColors.low}`}
      onClick={onSelect}
    >
      <div className="flex justify-between items-start mb-3">
        <div>
          <h3 className="font-semibold text-gray-900">{portfolio.portfolio_name}</h3>
          <p className="text-sm text-gray-600">{portfolio.portfolio_entity_id}</p>
        </div>
        <span className={`text-lg font-bold px-3 py-1 rounded-full ${getRiskBadgeColor(riskLevel)}`}>
          {(portfolio.current_risk_score || 0).toFixed(1)}/10
        </span>
      </div>

      <div className="grid grid-cols-2 gap-2 mb-3 text-sm">
        <StatItem label="Active Alerts" value={portfolio.active_alerts} />
        <StatItem label="Critical" value={portfolio.critical_alerts} highlight={portfolio.critical_alerts > 0} />
        <StatItem label="VaR 95%" value={portfolio.var_95 ? `${portfolio.var_95.toFixed(2)}%` : 'N/A'} />
        <StatItem label="Liquidity" value={portfolio.liquidity_ratio ? `${(portfolio.liquidity_ratio * 100).toFixed(1)}%` : 'N/A'} />
      </div>

      {portfolio.latest_risk_event && (
        <div className="bg-white bg-opacity-50 p-2 rounded mb-3 text-xs">
          <p className="font-medium text-gray-800">Latest: {portfolio.latest_risk_event.event_type}</p>
          <p className="text-gray-700 line-clamp-2">{portfolio.latest_risk_event.ai_reasoning?.substring(0, 80)}...</p>
        </div>
      )}

      <button
        onClick={(e) => {
          e.stopPropagation();
          onAnalyze();
        }}
        disabled={isAnalyzing}
        className="w-full bg-blue-600 hover:bg-blue-700 text-white font-medium py-2 px-4 rounded flex items-center justify-center gap-2 disabled:opacity-50"
      >
        <Zap size={16} />
        {isAnalyzing ? 'Analyzing...' : 'Run AI Analysis'}
      </button>
    </div>
  );
};

// Risk Event Row
interface RiskEventRowProps {
  event: RiskEvent;
}

const RiskEventRow: React.FC<RiskEventRowProps> = ({ event }) => {
  const severityColors = {
    CRITICAL: 'bg-red-100 text-red-800 border-red-300',
    HIGH: 'bg-orange-100 text-orange-800 border-orange-300',
    MEDIUM: 'bg-yellow-100 text-yellow-800 border-yellow-300',
    LOW: 'bg-blue-100 text-blue-800 border-blue-300',
  };

  return (
    <div className={`p-3 border-l-4 rounded ${severityColors[event.severity] || severityColors.LOW}`}>
      <div className="flex items-start justify-between">
        <div className="flex-1">
          <div className="flex items-center gap-2 mb-1">
            <AlertTriangle size={16} />
            <span className="font-semibold text-sm">{event.event_type}</span>
            <span className="text-xs px-2 py-1 bg-white bg-opacity-50 rounded">
              Risk: {event.risk_score.toFixed(1)}/10
            </span>
          </div>
          <p className="text-xs text-gray-700 line-clamp-2 mb-1">{event.ai_reasoning?.substring(0, 100)}</p>
          <div className="flex items-center gap-2 text-xs">
            <Clock size={12} />
            <span>{formatTime(event.detected_at)}</span>
            {event.auto_mitigated && (
              <span className="ml-auto flex items-center gap-1 text-green-700">
                <CheckCircle size={12} />
                Auto-Mitigated
              </span>
            )}
          </div>
        </div>
      </div>
    </div>
  );
};

// Helper Components
interface MetricCardProps {
  icon: React.ReactNode;
  label: string;
  value: string | number;
  suffix?: string;
  status: 'normal' | 'warning' | 'critical' | 'success';
}

const MetricCard: React.FC<MetricCardProps> = ({ icon, label, value, suffix, status }) => {
  const statusColors = {
    normal: 'bg-blue-50 border-blue-200',
    warning: 'bg-yellow-50 border-yellow-200',
    critical: 'bg-red-50 border-red-200',
    success: 'bg-green-50 border-green-200',
  };

  return (
    <div className={`p-4 border rounded-lg ${statusColors[status]}`}>
      <div className="flex items-center gap-3">
        <div className="flex-shrink-0">{icon}</div>
        <div className="flex-1">
          <p className="text-xs text-gray-600 font-medium">{label}</p>
          <p className="text-2xl font-bold text-gray-900">
            {value}
            {suffix && <span className="text-lg ml-1">{suffix}</span>}
          </p>
        </div>
      </div>
    </div>
  );
};

interface StatItemProps {
  label: string;
  value: string | number;
  highlight?: boolean;
}

const StatItem: React.FC<StatItemProps> = ({ label, value, highlight }) => (
  <div>
    <span className="text-gray-600">{label}:</span>
    <span className={`ml-1 font-semibold ${highlight ? 'text-red-600' : 'text-gray-900'}`}>{value}</span>
  </div>
);

// Helper Functions
function getRiskLevel(score: number): 'critical' | 'high' | 'medium' | 'low' {
  if (score >= 9) return 'critical';
  if (score >= 7) return 'high';
  if (score >= 5) return 'medium';
  return 'low';
}

function getRiskBadgeColor(level: string): string {
  const colors = {
    critical: 'bg-red-600 text-white',
    high: 'bg-orange-600 text-white',
    medium: 'bg-yellow-600 text-white',
    low: 'bg-green-600 text-white',
  };
  return colors[level as keyof typeof colors] || colors.low;
}

function formatTime(timestamp: string): string {
  const date = new Date(timestamp);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMins = Math.floor(diffMs / 60000);
  const diffHours = Math.floor(diffMs / 3600000);
  const diffDays = Math.floor(diffMs / 86400000);

  if (diffMins < 1) return 'now';
  if (diffMins < 60) return `${diffMins}m ago`;
  if (diffHours < 24) return `${diffHours}h ago`;
  if (diffDays < 7) return `${diffDays}d ago`;
  return date.toLocaleDateString();
}

export default RiskAlphaDashboard;
