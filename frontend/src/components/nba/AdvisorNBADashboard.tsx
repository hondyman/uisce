/**
 * Advisor NBA Dashboard Component
 * 
 * Main dashboard for AI-Driven Next Best Action recommendations.
 * Displays prioritized action recommendations with confidence scores,
 * expected value, and success probability. Enables one-click execution.
 */

import React, { useState, useEffect, useMemo, useCallback } from 'react';
import {
  AlertCircle,
  TrendingUp,
  Clock,
  DollarSign,
  Filter,
  RefreshCw,
  CheckCircle,
  XCircle,
  ChevronDown,
  User,
  Phone,
  Mail,
  Video,
  Bell,
  Zap,
  Target,
  BarChart3,
  Calendar,
  Star,
  ThumbsUp,
  ThumbsDown,
} from 'lucide-react';
import type {
  NextBestAction,
  ActionCategory,
  ActionPriority,
  ActionChannel,
  CompleteActionRequest,
} from '../../types/nba';

// ============================================================================
// Props
// ============================================================================

export interface AdvisorNBADashboardProps {
  advisorId: string;
  tenantId: string;
  datasourceId: string;
  onActionExecute?: (action: NextBestAction) => void;
  onActionDismiss?: (actionId: string, reason: string) => void;
}

// ============================================================================
// Constants
// ============================================================================

const CHANNEL_CONFIG: Record<ActionChannel, {
  icon: React.ElementType;
  label: string;
  color: string;
}> = {
  PHONE: { icon: Phone, label: 'Phone', color: 'text-green-600' },
  EMAIL: { icon: Mail, label: 'Email', color: 'text-blue-600' },
  VIDEO_CALL: { icon: Video, label: 'Video', color: 'text-purple-600' },
  IN_PERSON: { icon: User, label: 'In-Person', color: 'text-orange-600' },
  AUTOMATED_MESSAGE: { icon: Bell, label: 'Auto', color: 'text-slate-600' },
  PORTAL_NOTIFICATION: { icon: Bell, label: 'Portal', color: 'text-indigo-600' },
};

const PRIORITY_CONFIG: Record<ActionPriority, {
  label: string;
  bgColor: string;
  textColor: string;
}> = {
  CRITICAL: { label: 'Critical', bgColor: 'bg-red-100', textColor: 'text-red-700' },
  HIGH: { label: 'High', bgColor: 'bg-orange-100', textColor: 'text-orange-700' },
  MEDIUM: { label: 'Medium', bgColor: 'bg-yellow-100', textColor: 'text-yellow-700' },
  LOW: { label: 'Low', bgColor: 'bg-green-100', textColor: 'text-green-700' },
  OPTIONAL: { label: 'Optional', bgColor: 'bg-slate-100', textColor: 'text-slate-700' },
};

const CATEGORY_CONFIG: Record<ActionCategory, {
  label: string;
  icon: React.ElementType;
  color: string;
}> = {
  PROACTIVE_OUTREACH: { label: 'Outreach', icon: Phone, color: 'text-green-600' },
  SERVICE_DELIVERY: { label: 'Service', icon: CheckCircle, color: 'text-blue-600' },
  PORTFOLIO_MANAGEMENT: { label: 'Portfolio', icon: BarChart3, color: 'text-purple-600' },
  RELATIONSHIP_BUILDING: { label: 'Relationship', icon: User, color: 'text-pink-600' },
  COMPLIANCE: { label: 'Compliance', icon: AlertCircle, color: 'text-red-600' },
  TAX_PLANNING: { label: 'Tax', icon: DollarSign, color: 'text-amber-600' },
};

// ============================================================================
// Mock Data
// ============================================================================

const generateMockActions = (): NextBestAction[] => {
  const now = new Date();
  return [
    {
      actionId: 'a1',
      clientId: 'c1',
      clientName: 'John & Sarah Mitchell',
      clientTier: 'VIP',
      actionType: 'PROACTIVE_TAX_LOSS_HARVEST',
      actionName: 'Initiate Tax-Loss Harvesting Review',
      actionCategory: 'TAX_PLANNING',
      confidence: 0.92,
      urgencyScore: 0.88,
      expectedValue: 16650,
      successProbability: 0.85,
      triggerSignal: 'TAX_LOSS_HARVEST_OPPORTUNITY',
      triggerSignalStrength: 0.92,
      reasoning: 'Portfolio has $45,000 in unrealized losses across TSLA, NVDA, and META. Tax-loss harvesting could save approximately $16,650 in taxes at the 37% bracket. Market volatility suggests these losses may recover soon—act now to capture the benefit.',
      recommendedChannel: 'PHONE',
      estimatedDurationMinutes: 30,
      templateContent: {
        emailSubject: 'Opportunity to Reduce Your 2025 Tax Bill',
        emailBody: 'Hi John & Sarah,\n\nI noticed some unrealized losses in your portfolio that could save you approximately $16,650 in taxes this year through strategic tax-loss harvesting.\n\nWould you have 20 minutes this week to discuss this opportunity?\n\nBest regards,\nYour Advisor',
        callScript: 'Opening: I wanted to reach out because our system flagged a potential tax savings opportunity in your account...\n\nKey Points:\n- Current unrealized losses: $45,000\n- Estimated tax savings: $16,650\n- Recommended action: Harvest losses and reinvest in similar securities\n\nClose: Can we schedule 20 minutes to walk through the specific positions?',
      },
      recommendedAt: new Date(now.getTime() - 2 * 60 * 60 * 1000).toISOString(),
      priority: 'HIGH',
    },
    {
      actionId: 'a2',
      clientId: 'c2',
      clientName: 'Robert Chen',
      clientTier: 'HIGH_NET_WORTH',
      actionType: 'RETIREMENT_PLANNING_REVIEW',
      actionName: 'Schedule Retirement Readiness Review',
      actionCategory: 'PROACTIVE_OUTREACH',
      confidence: 0.88,
      urgencyScore: 0.95,
      expectedValue: 25000,
      successProbability: 0.78,
      triggerSignal: 'RETIREMENT_APPROACHING',
      triggerSignalStrength: 0.88,
      reasoning: 'Robert\'s retirement is 120 days away. Portfolio readiness score is 72%—below the 85% target. Key gaps: income planning not finalized, Social Security optimization not discussed, and healthcare bridge strategy needed.',
      recommendedChannel: 'VIDEO_CALL',
      estimatedDurationMinutes: 60,
      templateContent: {
        meetingAgenda: '1. Review current portfolio allocation\n2. Income planning strategy\n3. Social Security optimization\n4. Healthcare bridge (Medicare gap)\n5. Tax-efficient withdrawal strategy',
        emailSubject: 'Your Retirement is 120 Days Away - Let\'s Finalize Your Plan',
        emailBody: 'Hi Robert,\n\nWith your retirement just 120 days away, I wanted to schedule a comprehensive review to ensure everything is in place for a smooth transition.\n\nI\'d like to cover income planning, Social Security timing, healthcare, and your withdrawal strategy.\n\nWould next Tuesday at 2pm work for a video call?\n\nBest,\nYour Advisor',
      },
      recommendedAt: new Date(now.getTime() - 4 * 60 * 60 * 1000).toISOString(),
      priority: 'CRITICAL',
    },
    {
      actionId: 'a3',
      clientId: 'c3',
      clientName: 'Jennifer Williams',
      clientTier: 'STANDARD',
      actionType: 'REENGAGEMENT_OUTREACH',
      actionName: 'Client Re-engagement Call',
      actionCategory: 'RELATIONSHIP_BUILDING',
      confidence: 0.75,
      urgencyScore: 0.65,
      expectedValue: 5000,
      successProbability: 0.70,
      triggerSignal: 'ENGAGEMENT_DECLINE',
      triggerSignalStrength: 0.75,
      reasoning: 'Jennifer\'s portal logins dropped 75% (from 8 to 2 per month). Email open rate is below 20%. Early indicator of potential disengagement—proactive outreach can prevent attrition.',
      recommendedChannel: 'PHONE',
      estimatedDurationMinutes: 20,
      templateContent: {
        callScript: 'Hi Jennifer, I realized we haven\'t connected in a while and wanted to check in. How have things been going for you?\n\n[Listen actively]\n\nI want to make sure we\'re providing the level of service and communication that works best for you. Is there anything we could be doing differently?\n\n[Adjust communication preferences if needed]\n\nLet\'s schedule a portfolio review in the next couple weeks. What works better for you - morning or afternoon?',
        followUpEmail: 'Great talking with you today! As discussed, I\'m scheduling our portfolio review for [DATE]. Looking forward to it.',
      },
      recommendedAt: new Date(now.getTime() - 6 * 60 * 60 * 1000).toISOString(),
      priority: 'MEDIUM',
    },
    {
      actionId: 'a4',
      clientId: 'c4',
      clientName: 'Michael & Lisa Davis',
      clientTier: 'VIP',
      actionType: 'CONCENTRATED_POSITION_REVIEW',
      actionName: 'Diversification Strategy Discussion',
      actionCategory: 'PORTFOLIO_MANAGEMENT',
      confidence: 0.85,
      urgencyScore: 0.78,
      expectedValue: 8500,
      successProbability: 0.82,
      triggerSignal: 'CONCENTRATED_POSITION_RISK',
      triggerSignalStrength: 0.85,
      reasoning: 'AAPL position has grown to 32% of portfolio due to recent rally. Concentration risk exceeds policy limit of 10%. Tax-efficient diversification strategies available including qualified opportunity zones and charitable giving.',
      recommendedChannel: 'VIDEO_CALL',
      estimatedDurationMinutes: 45,
      templateContent: {
        meetingAgenda: '1. Review current portfolio concentration\n2. Discuss risks of single-position overweight\n3. Present diversification strategies\n4. Address tax implications\n5. Create implementation timeline',
        presentationSlides: [
          'Current Portfolio Allocation',
          'Concentration Risk Analysis',
          'Diversification Options',
          'Tax-Efficient Implementation',
          'Expected Risk Reduction',
        ],
      },
      recommendedAt: new Date(now.getTime() - 8 * 60 * 60 * 1000).toISOString(),
      priority: 'HIGH',
    },
    {
      actionId: 'a5',
      clientId: 'c6',
      clientName: 'Patricia Anderson',
      clientTier: 'HIGH_NET_WORTH',
      actionType: 'INHERITANCE_PLANNING',
      actionName: 'Inheritance Integration Meeting',
      actionCategory: 'SERVICE_DELIVERY',
      confidence: 0.95,
      urgencyScore: 0.92,
      expectedValue: 42500,
      successProbability: 0.90,
      triggerSignal: 'INHERITANCE_DETECTED',
      triggerSignalStrength: 0.95,
      reasoning: 'Large estate transfer of $850,000 detected. Critical to discuss: investment strategy, tax implications of inherited assets, step-up in basis opportunities, and integration with existing financial plan.',
      recommendedChannel: 'IN_PERSON',
      estimatedDurationMinutes: 90,
      templateContent: {
        emailSubject: 'Planning for Your Recent Inheritance',
        emailBody: 'Hi Patricia,\n\nI noticed a significant transfer into your account. I wanted to reach out to ensure we\'re handling this thoughtfully and taking advantage of all available planning opportunities.\n\nThis is an important moment, and there are several tax and investment considerations we should discuss. Would you be available for an in-person meeting this week?\n\nMy condolences if this is related to a loss. I\'m here to help however I can.\n\nWarmly,\nYour Advisor',
        meetingAgenda: '1. Understand the source and any emotional considerations\n2. Review step-up in basis implications\n3. Discuss investment allocation strategy\n4. Tax planning opportunities\n5. Update overall financial plan',
      },
      recommendedAt: new Date(now.getTime() - 24 * 60 * 60 * 1000).toISOString(),
      priority: 'CRITICAL',
    },
  ];
};

// ============================================================================
// Helper Components
// ============================================================================

interface StatCardProps {
  icon: React.ReactNode;
  label: string;
  value: string | number;
  subtext?: string;
}

const StatCard: React.FC<StatCardProps> = ({ icon, label, value, subtext }) => (
  <div className="bg-white rounded-lg border border-slate-200 p-4">
    <div className="flex items-center gap-3">
      <div className="p-2 bg-slate-100 rounded-lg">{icon}</div>
      <div>
        <p className="text-sm text-slate-500">{label}</p>
        <p className="text-xl font-bold text-slate-900">{value}</p>
        {subtext && <p className="text-xs text-slate-400">{subtext}</p>}
      </div>
    </div>
  </div>
);

interface ConfidenceBarProps {
  value: number;
  label?: string;
}

const ConfidenceBar: React.FC<ConfidenceBarProps> = ({ value, label }) => {
  const barRef = React.useRef<HTMLDivElement>(null);
  
  React.useEffect(() => {
    if (barRef.current) {
      barRef.current.style.width = `${value * 100}%`;
    }
  }, [value]);

  const getColor = () => {
    if (value >= 0.8) return 'bg-green-500';
    if (value >= 0.6) return 'bg-yellow-500';
    return 'bg-red-500';
  };

  return (
    <div>
      {label && <span className="text-xs text-slate-500">{label}</span>}
      <div className="flex items-center gap-2">
        <div className="flex-1 h-1.5 bg-slate-200 rounded-full overflow-hidden">
          <div
            ref={barRef}
            className={`h-full ${getColor()} transition-all duration-300`}
          />
        </div>
        <span className="text-xs font-medium text-slate-600 w-10 text-right">
          {(value * 100).toFixed(0)}%
        </span>
      </div>
    </div>
  );
};

interface UrgencyBadgeProps {
  score: number;
}

const UrgencyBadge: React.FC<UrgencyBadgeProps> = ({ score }) => {
  if (score >= 0.9) {
    return <span className="px-2 py-0.5 text-xs font-medium rounded bg-red-100 text-red-700">Urgent</span>;
  }
  if (score >= 0.7) {
    return <span className="px-2 py-0.5 text-xs font-medium rounded bg-orange-100 text-orange-700">High</span>;
  }
  if (score >= 0.5) {
    return <span className="px-2 py-0.5 text-xs font-medium rounded bg-yellow-100 text-yellow-700">Medium</span>;
  }
  return <span className="px-2 py-0.5 text-xs font-medium rounded bg-green-100 text-green-700">Low</span>;
};

interface TierBadgeProps {
  tier: 'VIP' | 'HIGH_NET_WORTH' | 'STANDARD';
}

const TierBadge: React.FC<TierBadgeProps> = ({ tier }) => {
  const config = {
    VIP: { bg: 'bg-purple-100', text: 'text-purple-700', icon: Star },
    HIGH_NET_WORTH: { bg: 'bg-blue-100', text: 'text-blue-700', icon: Target },
    STANDARD: { bg: 'bg-slate-100', text: 'text-slate-700', icon: User },
  };

  const { bg, text, icon: Icon } = config[tier];

  return (
    <span className={`inline-flex items-center gap-1 px-2 py-0.5 text-xs font-medium rounded ${bg} ${text}`}>
      <Icon className="w-3 h-3" />
      {tier.replace('_', ' ')}
    </span>
  );
};

// ============================================================================
// Action Card Component
// ============================================================================

interface ActionCardComponentProps {
  action: NextBestAction;
  onExecute: () => void;
  onDismiss: (reason: string) => void;
  onViewDetails: () => void;
}

const ActionCard: React.FC<ActionCardComponentProps> = ({
  action,
  onExecute,
  onDismiss,
  onViewDetails,
}) => {
  const [showDismissMenu, setShowDismissMenu] = useState(false);
  const channelConfig = CHANNEL_CONFIG[action.recommendedChannel];
  const priorityConfig = PRIORITY_CONFIG[action.priority];
  const categoryConfig = CATEGORY_CONFIG[action.actionCategory];
  const ChannelIcon = channelConfig.icon;
  const CategoryIcon = categoryConfig.icon;

  const formatTimeAgo = (dateStr: string) => {
    const date = new Date(dateStr);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffHours = Math.floor(diffMs / (1000 * 60 * 60));
    if (diffHours >= 24) return `${Math.floor(diffHours / 24)}d ago`;
    if (diffHours >= 1) return `${diffHours}h ago`;
    return `${Math.floor(diffMs / (1000 * 60))}m ago`;
  };

  return (
    <div className="bg-white border border-slate-200 rounded-lg p-4 hover:shadow-md transition-shadow">
      {/* Header */}
      <div className="flex items-start justify-between mb-3">
        <div className="flex items-center gap-3">
          <div className={`p-2 rounded-lg ${categoryConfig.color.replace('text-', 'bg-').replace('600', '100')}`}>
            <CategoryIcon className={`w-5 h-5 ${categoryConfig.color}`} />
          </div>
          <div>
            <div className="flex items-center gap-2">
              <h3 className="font-semibold text-slate-900">{action.clientName}</h3>
              <TierBadge tier={action.clientTier} />
            </div>
            <p className="text-sm text-slate-500">{action.actionName}</p>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <span className={`px-2 py-0.5 text-xs font-medium rounded ${priorityConfig.bgColor} ${priorityConfig.textColor}`}>
            {priorityConfig.label}
          </span>
          <span className="text-xs text-slate-400">{formatTimeAgo(action.recommendedAt)}</span>
        </div>
      </div>

      {/* Key Metrics */}
      <div className="grid grid-cols-4 gap-4 mb-4 p-3 bg-slate-50 rounded-lg">
        <div>
          <span className="text-xs text-slate-500">Confidence</span>
          <ConfidenceBar value={action.confidence} />
        </div>
        <div>
          <span className="text-xs text-slate-500">Urgency</span>
          <div className="flex items-center gap-1 mt-1">
            <UrgencyBadge score={action.urgencyScore} />
          </div>
        </div>
        <div>
          <span className="text-xs text-slate-500">Expected Value</span>
          <p className="text-sm font-semibold text-green-600">${action.expectedValue.toLocaleString()}</p>
        </div>
        <div>
          <span className="text-xs text-slate-500">Success Rate</span>
          <p className="text-sm font-semibold">{(action.successProbability * 100).toFixed(0)}%</p>
        </div>
      </div>

      {/* Signal & Channel */}
      <div className="flex items-center gap-4 mb-4 text-sm">
        <span className="flex items-center gap-1 text-slate-600">
          <Zap className="w-4 h-4 text-amber-500" />
          {action.triggerSignal.replace(/_/g, ' ')}
        </span>
        <span className={`flex items-center gap-1 ${channelConfig.color}`}>
          <ChannelIcon className="w-4 h-4" />
          {channelConfig.label}
        </span>
        <span className="flex items-center gap-1 text-slate-500">
          <Clock className="w-4 h-4" />
          {action.estimatedDurationMinutes} min
        </span>
      </div>

      {/* Reasoning */}
      <div className="mb-4 p-3 bg-blue-50 border border-blue-100 rounded-lg">
        <p className="text-sm text-slate-700">
          <strong className="text-blue-700">AI Reasoning:</strong> {action.reasoning}
        </p>
      </div>

      {/* Actions */}
      <div className="flex items-center gap-2">
        <button
          onClick={onExecute}
          className="flex-1 px-4 py-2 text-sm font-medium text-white bg-indigo-600 rounded-lg hover:bg-indigo-700 flex items-center justify-center gap-2"
          title="Execute Action"
        >
          <Zap className="w-4 h-4" />
          Execute
        </button>
        <button
          onClick={onViewDetails}
          className="px-4 py-2 text-sm font-medium text-slate-700 bg-slate-100 rounded-lg hover:bg-slate-200"
          title="View Details"
        >
          Details
        </button>
        <div className="relative">
          <button
            onClick={() => setShowDismissMenu(!showDismissMenu)}
            className="px-3 py-2 text-sm text-slate-500 hover:bg-slate-100 rounded-lg flex items-center gap-1"
            title="Dismiss Options"
          >
            <XCircle className="w-4 h-4" />
            <ChevronDown className="w-3 h-3" />
          </button>
          {showDismissMenu && (
            <div className="absolute right-0 mt-1 w-48 bg-white border border-slate-200 rounded-lg shadow-lg z-10">
              {['NOT_RELEVANT', 'ALREADY_DONE', 'WRONG_TIMING', 'CLIENT_OPTED_OUT'].map(reason => (
                <button
                  key={reason}
                  onClick={() => {
                    onDismiss(reason);
                    setShowDismissMenu(false);
                  }}
                  className="w-full px-4 py-2 text-left text-sm text-slate-700 hover:bg-slate-50 first:rounded-t-lg last:rounded-b-lg"
                  title={reason.replace(/_/g, ' ')}
                >
                  {reason.replace(/_/g, ' ')}
                </button>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

// ============================================================================
// Main Component
// ============================================================================

export const AdvisorNBADashboard: React.FC<AdvisorNBADashboardProps> = ({
  advisorId,
  tenantId,
  datasourceId,
  onActionExecute,
  onActionDismiss,
}) => {
  // Suppress unused variable warnings
  void advisorId;
  void tenantId;
  void datasourceId;

  const [actions, setActions] = useState<NextBestAction[]>([]);
  const [filterCategory, setFilterCategory] = useState<ActionCategory | 'ALL'>('ALL');
  const [filterPriority, setFilterPriority] = useState<'ALL' | 'CRITICAL' | 'HIGH_VALUE'>('ALL');
  const [selectedAction, setSelectedAction] = useState<NextBestAction | null>(null);
  const [showExecutionModal, setShowExecutionModal] = useState(false);
  const [isRefreshing, setIsRefreshing] = useState(false);

  // Load mock data
  useEffect(() => {
    setActions(generateMockActions());
  }, []);

  // Filtered actions
  const filteredActions = useMemo(() => {
    return actions.filter(action => {
      if (filterCategory !== 'ALL' && action.actionCategory !== filterCategory) {
        return false;
      }
      if (filterPriority === 'CRITICAL' && action.urgencyScore < 0.8) {
        return false;
      }
      if (filterPriority === 'HIGH_VALUE' && action.expectedValue < 10000) {
        return false;
      }
      return true;
    });
  }, [actions, filterCategory, filterPriority]);

  // Stats
  const stats = useMemo(() => {
    const criticalCount = actions.filter(a => a.urgencyScore > 0.8).length;
    const totalValue = actions.reduce((sum, a) => sum + a.expectedValue, 0);
    const avgSuccess = actions.length > 0
      ? actions.reduce((sum, a) => sum + a.successProbability, 0) / actions.length
      : 0;
    const totalTime = actions.reduce((sum, a) => sum + a.estimatedDurationMinutes, 0);

    return { criticalCount, totalValue, avgSuccess, totalTime };
  }, [actions]);

  const handleRefresh = useCallback(async () => {
    setIsRefreshing(true);
    await new Promise(resolve => setTimeout(resolve, 1000));
    setActions(generateMockActions());
    setIsRefreshing(false);
  }, []);

  const handleExecute = useCallback((action: NextBestAction) => {
    setSelectedAction(action);
    setShowExecutionModal(true);
    if (onActionExecute) {
      onActionExecute(action);
    }
  }, [onActionExecute]);

  const handleDismiss = useCallback((actionId: string, reason: string) => {
    setActions(prev => prev.filter(a => a.actionId !== actionId));
    if (onActionDismiss) {
      onActionDismiss(actionId, reason);
    }
  }, [onActionDismiss]);

  const handleCompleteAction = useCallback((_request: CompleteActionRequest) => {
    setShowExecutionModal(false);
    setSelectedAction(null);
    // Would send to API
  }, []);

  // ============================================================================
  // Render
  // ============================================================================

  return (
    <div className="bg-slate-50 min-h-screen">
      {/* Header */}
      <div className="bg-white border-b border-slate-200 px-6 py-4">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold text-slate-900 flex items-center gap-2">
              <Zap className="w-7 h-7 text-indigo-600" />
              AI-Recommended Actions
            </h1>
            <p className="text-sm text-slate-500 mt-1">
              Proactive client engagement powered by multi-signal intelligence
            </p>
          </div>
          <div className="flex items-center gap-3">
            <button
              onClick={handleRefresh}
              disabled={isRefreshing}
              className="px-4 py-2 text-sm font-medium text-slate-700 bg-white border border-slate-300 rounded-lg hover:bg-slate-50 flex items-center gap-2"
              title="Refresh Recommendations"
            >
              <RefreshCw className={`w-4 h-4 ${isRefreshing ? 'animate-spin' : ''}`} />
              Refresh
            </button>
          </div>
        </div>
      </div>

      {/* Stats */}
      <div className="px-6 py-4">
        <div className="grid grid-cols-4 gap-4">
          <StatCard
            icon={<AlertCircle className="w-5 h-5 text-red-500" />}
            label="Critical Actions"
            value={stats.criticalCount}
            subtext="Requires immediate attention"
          />
          <StatCard
            icon={<DollarSign className="w-5 h-5 text-green-500" />}
            label="Potential Revenue"
            value={`$${stats.totalValue.toLocaleString()}`}
            subtext="Total expected value"
          />
          <StatCard
            icon={<TrendingUp className="w-5 h-5 text-blue-500" />}
            label="Avg Success Rate"
            value={`${(stats.avgSuccess * 100).toFixed(0)}%`}
            subtext="Based on historical data"
          />
          <StatCard
            icon={<Clock className="w-5 h-5 text-purple-500" />}
            label="Time Required"
            value={`${stats.totalTime} min`}
            subtext="For all actions"
          />
        </div>
      </div>

      {/* Filters */}
      <div className="px-6 py-3 flex items-center gap-4 border-b border-slate-200 bg-white">
        <Filter className="w-4 h-4 text-slate-400" />
        <div className="flex gap-2">
          <button
            onClick={() => setFilterPriority('ALL')}
            className={`px-3 py-1.5 text-sm rounded-lg ${filterPriority === 'ALL' ? 'bg-indigo-100 text-indigo-700 font-medium' : 'text-slate-600 hover:bg-slate-100'}`}
            title="Show all actions"
          >
            All ({actions.length})
          </button>
          <button
            onClick={() => setFilterPriority('CRITICAL')}
            className={`px-3 py-1.5 text-sm rounded-lg ${filterPriority === 'CRITICAL' ? 'bg-red-100 text-red-700 font-medium' : 'text-slate-600 hover:bg-slate-100'}`}
            title="Show critical actions only"
          >
            Critical ({actions.filter(a => a.urgencyScore > 0.8).length})
          </button>
          <button
            onClick={() => setFilterPriority('HIGH_VALUE')}
            className={`px-3 py-1.5 text-sm rounded-lg ${filterPriority === 'HIGH_VALUE' ? 'bg-green-100 text-green-700 font-medium' : 'text-slate-600 hover:bg-slate-100'}`}
            title="Show high value actions only"
          >
            High Value ({actions.filter(a => a.expectedValue > 10000).length})
          </button>
        </div>

        <div className="ml-auto">
          <select
            value={filterCategory}
            onChange={(e) => setFilterCategory(e.target.value as ActionCategory | 'ALL')}
            className="px-3 py-1.5 text-sm border border-slate-300 rounded-lg"
            title="Filter by category"
          >
            <option value="ALL">All Categories</option>
            {Object.entries(CATEGORY_CONFIG).map(([key, config]) => (
              <option key={key} value={key}>{config.label}</option>
            ))}
          </select>
        </div>
      </div>

      {/* Action Cards */}
      <div className="px-6 py-4">
        <div className="space-y-4">
          {filteredActions.map(action => (
            <ActionCard
              key={action.actionId}
              action={action}
              onExecute={() => handleExecute(action)}
              onDismiss={(reason) => handleDismiss(action.actionId, reason)}
              onViewDetails={() => setSelectedAction(action)}
            />
          ))}
        </div>

        {filteredActions.length === 0 && (
          <div className="text-center py-12">
            <CheckCircle className="w-12 h-12 text-green-400 mx-auto mb-3" />
            <p className="text-slate-600 font-medium">All caught up!</p>
            <p className="text-sm text-slate-400">No actions match your current filters</p>
          </div>
        )}
      </div>

      {/* Execution Modal */}
      {showExecutionModal && selectedAction && (
        <ActionExecutionModal
          action={selectedAction}
          onComplete={handleCompleteAction}
          onClose={() => {
            setShowExecutionModal(false);
            setSelectedAction(null);
          }}
        />
      )}
    </div>
  );
};

// ============================================================================
// Action Execution Modal Component
// ============================================================================

interface ActionExecutionModalProps {
  action: NextBestAction;
  onComplete: (request: CompleteActionRequest) => void;
  onClose: () => void;
}

const ActionExecutionModal: React.FC<ActionExecutionModalProps> = ({
  action,
  onComplete,
  onClose,
}) => {
  const [notes, setNotes] = useState('');
  const [outcome, setOutcome] = useState<'SUCCESS' | 'PARTIAL' | 'FAILED' | null>(null);
  const [clientResponded, setClientResponded] = useState(true);
  const [advisorRating, setAdvisorRating] = useState<number | null>(null);

  const handleComplete = () => {
    if (!outcome) return;

    onComplete({
      actionId: action.actionId,
      outcome,
      notes,
      clientResponded,
      advisorRating: advisorRating || undefined,
    });
  };

  const channelConfig = CHANNEL_CONFIG[action.recommendedChannel];

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <div className="bg-white rounded-xl shadow-xl w-full max-w-2xl max-h-[90vh] overflow-y-auto">
        {/* Header */}
        <div className="p-6 border-b border-slate-200">
          <div className="flex items-center justify-between">
            <div>
              <h2 className="text-xl font-bold text-slate-900">{action.actionName}</h2>
              <p className="text-sm text-slate-500 mt-1">Client: {action.clientName}</p>
            </div>
            <button
              onClick={onClose}
              className="p-2 hover:bg-slate-100 rounded-lg"
              title="Close"
            >
              <XCircle className="w-5 h-5 text-slate-400" />
            </button>
          </div>
        </div>

        <div className="p-6 space-y-6">
          {/* Recommended Channel */}
          <div className="flex items-center gap-3 p-3 bg-indigo-50 rounded-lg">
            <div className={`p-2 bg-white rounded-lg ${channelConfig.color}`}>
              {React.createElement(channelConfig.icon, { className: 'w-5 h-5' })}
            </div>
            <div>
              <p className="font-medium text-slate-900">Recommended: {channelConfig.label}</p>
              <p className="text-sm text-slate-500">Est. {action.estimatedDurationMinutes} minutes</p>
            </div>
          </div>

          {/* Email Template */}
          {action.templateContent.emailBody && (
            <div>
              <h3 className="font-semibold text-slate-900 mb-2 flex items-center gap-2">
                <Mail className="w-4 h-4" />
                Suggested Email
              </h3>
              <div className="bg-slate-50 rounded-lg p-4 border border-slate-200">
                <p className="text-sm font-medium text-slate-700 mb-2">
                  Subject: {action.templateContent.emailSubject}
                </p>
                <p className="text-sm text-slate-600 whitespace-pre-wrap">
                  {action.templateContent.emailBody}
                </p>
              </div>
              <button className="mt-2 px-4 py-2 text-sm font-medium text-indigo-600 bg-indigo-50 rounded-lg hover:bg-indigo-100" title="Copy to Clipboard">
                Copy to Clipboard
              </button>
            </div>
          )}

          {/* Call Script */}
          {action.templateContent.callScript && (
            <div>
              <h3 className="font-semibold text-slate-900 mb-2 flex items-center gap-2">
                <Phone className="w-4 h-4" />
                Call Script
              </h3>
              <div className="bg-slate-50 rounded-lg p-4 border border-slate-200">
                <p className="text-sm text-slate-600 whitespace-pre-wrap">
                  {action.templateContent.callScript}
                </p>
              </div>
            </div>
          )}

          {/* Meeting Agenda */}
          {action.templateContent.meetingAgenda && (
            <div>
              <h3 className="font-semibold text-slate-900 mb-2 flex items-center gap-2">
                <Calendar className="w-4 h-4" />
                Meeting Agenda
              </h3>
              <div className="bg-slate-50 rounded-lg p-4 border border-slate-200">
                <p className="text-sm text-slate-600 whitespace-pre-wrap">
                  {action.templateContent.meetingAgenda}
                </p>
              </div>
            </div>
          )}

          {/* Outcome Tracking */}
          <div className="border-t border-slate-200 pt-6">
            <h3 className="font-semibold text-slate-900 mb-3">Record Outcome</h3>
            
            <div className="flex gap-2 mb-4">
              <button
                onClick={() => setOutcome('SUCCESS')}
                className={`flex-1 px-4 py-3 rounded-lg border-2 flex items-center justify-center gap-2 ${
                  outcome === 'SUCCESS' ? 'border-green-500 bg-green-50 text-green-700' : 'border-slate-200 text-slate-600 hover:border-slate-300'
                }`}
                title="Mark as Success"
              >
                <ThumbsUp className="w-5 h-5" />
                Success
              </button>
              <button
                onClick={() => setOutcome('PARTIAL')}
                className={`flex-1 px-4 py-3 rounded-lg border-2 flex items-center justify-center gap-2 ${
                  outcome === 'PARTIAL' ? 'border-yellow-500 bg-yellow-50 text-yellow-700' : 'border-slate-200 text-slate-600 hover:border-slate-300'
                }`}
                title="Mark as Partial"
              >
                <Target className="w-5 h-5" />
                Partial
              </button>
              <button
                onClick={() => setOutcome('FAILED')}
                className={`flex-1 px-4 py-3 rounded-lg border-2 flex items-center justify-center gap-2 ${
                  outcome === 'FAILED' ? 'border-red-500 bg-red-50 text-red-700' : 'border-slate-200 text-slate-600 hover:border-slate-300'
                }`}
                title="Mark as Failed"
              >
                <ThumbsDown className="w-5 h-5" />
                Failed
              </button>
            </div>

            <div className="flex items-center gap-4 mb-4">
              <label className="flex items-center gap-2">
                <input
                  type="checkbox"
                  checked={clientResponded}
                  onChange={(e) => setClientResponded(e.target.checked)}
                  className="w-4 h-4 text-indigo-600 rounded"
                />
                <span className="text-sm text-slate-700">Client responded</span>
              </label>
            </div>

            <div className="mb-4">
              <label className="block text-sm font-medium text-slate-700 mb-1">
                Rate this recommendation (optional)
              </label>
              <div className="flex gap-1">
                {[1, 2, 3, 4, 5].map(rating => (
                  <button
                    key={rating}
                    onClick={() => setAdvisorRating(rating)}
                    className={`p-2 rounded ${advisorRating && advisorRating >= rating ? 'text-yellow-500' : 'text-slate-300'}`}
                    title={`Rate ${rating} stars`}
                  >
                    <Star className="w-6 h-6 fill-current" />
                  </button>
                ))}
              </div>
            </div>

            <div className="mb-4">
              <label className="block text-sm font-medium text-slate-700 mb-1">
                Notes (optional)
              </label>
              <textarea
                value={notes}
                onChange={(e) => setNotes(e.target.value)}
                className="w-full border border-slate-300 rounded-lg p-3 text-sm"
                rows={3}
                placeholder="What happened? Any feedback on this recommendation?"
              />
            </div>
          </div>
        </div>

        {/* Footer */}
        <div className="p-6 border-t border-slate-200 flex gap-3">
          <button
            onClick={handleComplete}
            disabled={!outcome}
            className="flex-1 px-4 py-2 text-sm font-medium text-white bg-indigo-600 rounded-lg hover:bg-indigo-700 disabled:bg-slate-300 disabled:cursor-not-allowed"
            title="Complete Action"
          >
            Complete Action
          </button>
          <button
            onClick={onClose}
            className="px-4 py-2 text-sm font-medium text-slate-700 bg-slate-100 rounded-lg hover:bg-slate-200"
            title="Cancel"
          >
            Cancel
          </button>
        </div>
      </div>
    </div>
  );
};

export default AdvisorNBADashboard;
