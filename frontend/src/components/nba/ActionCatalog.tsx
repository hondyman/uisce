/**
 * Action Catalog Component
 * 
 * Browse and configure the pre-defined action library for NBA recommendations.
 * Shows all available action templates with their configurations, success metrics,
 * required skills, and estimated impact.
 */

import React, { useState, useMemo } from 'react';
import {
  BookOpen,
  Search,
  Filter,
  Phone,
  Mail,
  Video,
  User,
  Bell,
  CheckCircle,
  Clock,
  DollarSign,
  Target,
  Copy,
  Edit,
  Shield,
  BarChart3,
  TrendingUp,
  AlertCircle,
  ChevronRight,
  Zap,
} from 'lucide-react';
import type {
  NBAAction,
  ActionCategory,
  ActionChannel,
} from '../../types/nba';

// ============================================================================
// Props
// ============================================================================

export interface ActionCatalogProps {
  tenantId: string;
  datasourceId: string;
  onSelectAction?: (action: NBAAction) => void;
  onEditAction?: (action: NBAAction) => void;
}

// ============================================================================
// Constants
// ============================================================================

const CHANNEL_CONFIG: Record<ActionChannel, {
  icon: React.ElementType;
  label: string;
  color: string;
  bgColor: string;
}> = {
  PHONE: { icon: Phone, label: 'Phone', color: 'text-green-600', bgColor: 'bg-green-100' },
  EMAIL: { icon: Mail, label: 'Email', color: 'text-blue-600', bgColor: 'bg-blue-100' },
  VIDEO_CALL: { icon: Video, label: 'Video Call', color: 'text-purple-600', bgColor: 'bg-purple-100' },
  IN_PERSON: { icon: User, label: 'In-Person', color: 'text-orange-600', bgColor: 'bg-orange-100' },
  AUTOMATED_MESSAGE: { icon: Bell, label: 'Automated', color: 'text-slate-600', bgColor: 'bg-slate-100' },
  PORTAL_NOTIFICATION: { icon: Bell, label: 'Portal', color: 'text-indigo-600', bgColor: 'bg-indigo-100' },
};

const CATEGORY_CONFIG: Record<ActionCategory, {
  label: string;
  icon: React.ElementType;
  color: string;
  bgColor: string;
  description: string;
}> = {
  PROACTIVE_OUTREACH: {
    label: 'Proactive Outreach',
    icon: Phone,
    color: 'text-green-600',
    bgColor: 'bg-green-100',
    description: 'Initiating contact based on detected signals',
  },
  SERVICE_DELIVERY: {
    label: 'Service Delivery',
    icon: CheckCircle,
    color: 'text-blue-600',
    bgColor: 'bg-blue-100',
    description: 'Fulfilling client service needs',
  },
  PORTFOLIO_MANAGEMENT: {
    label: 'Portfolio Management',
    icon: BarChart3,
    color: 'text-purple-600',
    bgColor: 'bg-purple-100',
    description: 'Investment and allocation adjustments',
  },
  RELATIONSHIP_BUILDING: {
    label: 'Relationship Building',
    icon: User,
    color: 'text-pink-600',
    bgColor: 'bg-pink-100',
    description: 'Strengthening client relationships',
  },
  COMPLIANCE: {
    label: 'Compliance',
    icon: Shield,
    color: 'text-red-600',
    bgColor: 'bg-red-100',
    description: 'Regulatory and compliance actions',
  },
  TAX_PLANNING: {
    label: 'Tax Planning',
    icon: DollarSign,
    color: 'text-amber-600',
    bgColor: 'bg-amber-100',
    description: 'Tax optimization strategies',
  },
};

// ============================================================================
// Mock Data
// ============================================================================

const MOCK_ACTIONS: NBAAction[] = [
  {
    actionId: 'act1',
    actionCode: 'PROACTIVE_TAX_LOSS_HARVEST',
    actionName: 'Initiate Tax-Loss Harvesting Review',
    actionCategory: 'TAX_PLANNING',
    description: 'Proactively reach out to discuss tax-loss harvesting opportunities based on unrealized losses detected in portfolio.',
    defaultChannel: 'PHONE',
    estimatedDurationMinutes: 30,
    estimatedRevenueImpact: 2500,
    clientValueImpact: 0.15,
    automationEligible: false,
    templateContent: {
      emailSubject: 'Opportunity to Reduce Your 2025 Tax Bill',
      emailBody: 'Hi {client_first_name},\n\nI noticed some unrealized losses in your portfolio that could save you approximately ${estimated_tax_savings:,.0f} in taxes this year through strategic tax-loss harvesting.\n\nWould you have 20 minutes this week to discuss this opportunity?\n\nBest regards,\n{advisor_name}',
      callScript: 'Opening: I wanted to reach out because our system flagged a potential tax savings opportunity in your account...\n\nKey Points:\n- Current unrealized losses: ${total_loss}\n- Estimated tax savings: ${tax_benefit}\n- Recommended action: Harvest losses and reinvest in similar securities\n\nClose: Can we schedule 20 minutes to walk through the specific positions?',
    },
    requiredAdvisorSkills: ['TAX_PLANNING'],
    complianceReviewRequired: false,
    successMetrics: { successMetric: 'tax_loss_harvested_amount', targetValue: 10000 },
  },
  {
    actionId: 'act2',
    actionCode: 'REENGAGEMENT_OUTREACH',
    actionName: 'Client Re-engagement Call',
    actionCategory: 'RELATIONSHIP_BUILDING',
    description: 'Reach out to client showing signs of disengagement (low portal logins, low email opens).',
    defaultChannel: 'PHONE',
    estimatedDurationMinutes: 20,
    estimatedRevenueImpact: 5000,
    clientValueImpact: 0.25,
    automationEligible: false,
    templateContent: {
      callScript: 'Hi {client_first_name}, I realized we haven\'t connected in a while and wanted to check in. How have things been going for you?\n\n[Listen actively]\n\nI want to make sure we\'re providing the level of service and communication that works best for you. Is there anything we could be doing differently?\n\n[Adjust communication preferences if needed]\n\nLet\'s schedule a portfolio review in the next couple weeks. What works better for you - morning or afternoon?',
      followUpEmail: 'Great talking with you today! As discussed, I\'m scheduling our portfolio review for {meeting_date}. Looking forward to it.',
    },
    requiredAdvisorSkills: ['RELATIONSHIP_MANAGEMENT'],
    complianceReviewRequired: false,
    successMetrics: { successMetric: 'engagement_score_increase', targetValue: 0.3 },
  },
  {
    actionId: 'act3',
    actionCode: 'CONCENTRATED_POSITION_REVIEW',
    actionName: 'Diversification Strategy Discussion',
    actionCategory: 'PORTFOLIO_MANAGEMENT',
    description: 'Schedule meeting to discuss concentrated position risk and diversification options.',
    defaultChannel: 'VIDEO_CALL',
    estimatedDurationMinutes: 45,
    estimatedRevenueImpact: 3500,
    clientValueImpact: 0.20,
    automationEligible: false,
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
    requiredAdvisorSkills: ['PORTFOLIO_MANAGEMENT', 'RISK_MANAGEMENT'],
    complianceReviewRequired: true,
    successMetrics: { successMetric: 'position_concentration_reduction', targetValue: 0.15 },
  },
  {
    actionId: 'act4',
    actionCode: 'RETIREMENT_READINESS_REVIEW',
    actionName: 'Retirement Readiness Assessment',
    actionCategory: 'PROACTIVE_OUTREACH',
    description: 'Comprehensive review for clients approaching retirement within 12 months.',
    defaultChannel: 'IN_PERSON',
    estimatedDurationMinutes: 90,
    estimatedRevenueImpact: 15000,
    clientValueImpact: 0.40,
    automationEligible: false,
    templateContent: {
      meetingAgenda: '1. Review retirement income needs\n2. Social Security optimization\n3. Healthcare planning (Medicare)\n4. Tax-efficient withdrawal strategy\n5. Estate planning review\n6. Legacy and gifting goals',
      emailSubject: 'Your Retirement Journey - Let\'s Plan Together',
      emailBody: 'Dear {client_name},\n\nWith your planned retirement date approaching, I believe it\'s the perfect time to review your retirement readiness and ensure all the pieces are in place for a smooth transition.\n\nI\'d like to schedule a comprehensive planning session to cover income planning, Social Security timing, healthcare, and withdrawal strategies.\n\nPlease let me know your availability over the next two weeks.\n\nBest regards,\n{advisor_name}',
    },
    requiredAdvisorSkills: ['RETIREMENT_PLANNING', 'TAX_PLANNING', 'ESTATE_PLANNING'],
    complianceReviewRequired: false,
    successMetrics: { successMetric: 'retirement_readiness_score', targetValue: 0.85 },
  },
  {
    actionId: 'act5',
    actionCode: 'INHERITANCE_PLANNING',
    actionName: 'Inheritance Integration Meeting',
    actionCategory: 'SERVICE_DELIVERY',
    description: 'Sensitive outreach for clients who have received a significant inheritance.',
    defaultChannel: 'IN_PERSON',
    estimatedDurationMinutes: 60,
    estimatedRevenueImpact: 25000,
    clientValueImpact: 0.35,
    automationEligible: false,
    templateContent: {
      emailSubject: 'Thinking of You - Planning Conversation',
      emailBody: 'Dear {client_name},\n\nI noticed a significant transfer into your account. I wanted to reach out to ensure we handle this thoughtfully and take advantage of all available planning opportunities.\n\nIf this is related to a loss, please accept my sincere condolences. I\'m here to help however I can.\n\nWhen you\'re ready, I\'d welcome the opportunity to meet and discuss how to integrate this into your overall financial plan.\n\nWarmly,\n{advisor_name}',
      meetingAgenda: '1. Understand the source and emotional context\n2. Review step-up in basis implications\n3. Investment allocation strategy\n4. Tax planning opportunities\n5. Update overall financial plan',
    },
    requiredAdvisorSkills: ['ESTATE_PLANNING', 'TAX_PLANNING', 'RELATIONSHIP_MANAGEMENT'],
    complianceReviewRequired: false,
    successMetrics: { successMetric: 'assets_retained', targetValue: 0.95 },
  },
  {
    actionId: 'act6',
    actionCode: 'EXCESS_CASH_DEPLOYMENT',
    actionName: 'Cash Deployment Strategy',
    actionCategory: 'PORTFOLIO_MANAGEMENT',
    description: 'Discuss investment options for clients with excess cash holdings.',
    defaultChannel: 'PHONE',
    estimatedDurationMinutes: 25,
    estimatedRevenueImpact: 1500,
    clientValueImpact: 0.10,
    automationEligible: true,
    templateContent: {
      emailSubject: 'Maximizing Your Cash Holdings',
      emailBody: 'Hi {client_name},\n\nI noticed you have a significant cash position ({cash_percentage}% of your portfolio). While some cash is important for liquidity, this level may be creating opportunity cost.\n\nI\'d like to discuss options to put this cash to work while maintaining appropriate liquidity for your needs.\n\nCan we schedule a quick call this week?\n\nBest,\n{advisor_name}',
      callScript: 'Hi {client_name}, I wanted to discuss your current cash position...\n\nKey Points:\n- Current cash: {cash_percentage}%\n- Estimated opportunity cost: {opportunity_cost_annual} annually\n- Options: Money market, short-term bonds, systematic investment\n\nClose: Which approach sounds most aligned with your goals?',
    },
    requiredAdvisorSkills: ['PORTFOLIO_MANAGEMENT'],
    complianceReviewRequired: false,
    successMetrics: { successMetric: 'cash_deployed_amount', targetValue: 50000 },
  },
  {
    actionId: 'act7',
    actionCode: 'VOLATILITY_COMFORT_CHECK',
    actionName: 'Market Volatility Check-In',
    actionCategory: 'RELATIONSHIP_BUILDING',
    description: 'Proactive outreach during high volatility to reassure clients and reinforce strategy.',
    defaultChannel: 'EMAIL',
    estimatedDurationMinutes: 10,
    estimatedRevenueImpact: 3000,
    clientValueImpact: 0.20,
    automationEligible: true,
    templateContent: {
      emailSubject: 'Checking In During Market Volatility',
      emailBody: 'Dear {client_name},\n\nWith recent market volatility, I wanted to check in and reassure you that your portfolio is designed to weather periods like this.\n\nKey points to remember:\n- Your allocation reflects your long-term goals and risk tolerance\n- Market corrections are normal and expected\n- Staying the course has historically been rewarded\n\nI\'m here if you have any questions or would like to discuss your portfolio.\n\nBest regards,\n{advisor_name}',
    },
    requiredAdvisorSkills: ['RELATIONSHIP_MANAGEMENT'],
    complianceReviewRequired: false,
    successMetrics: { successMetric: 'client_satisfaction_maintained', targetValue: 0.90 },
  },
  {
    actionId: 'act8',
    actionCode: 'ANNUAL_REVIEW_SCHEDULING',
    actionName: 'Annual Review Invitation',
    actionCategory: 'SERVICE_DELIVERY',
    description: 'Schedule annual comprehensive portfolio and planning review.',
    defaultChannel: 'EMAIL',
    estimatedDurationMinutes: 15,
    estimatedRevenueImpact: 500,
    clientValueImpact: 0.15,
    automationEligible: true,
    templateContent: {
      emailSubject: 'Time for Your Annual Review',
      emailBody: 'Dear {client_name},\n\nIt\'s been a year since our last comprehensive review, and I\'d like to schedule our annual meeting to ensure your financial plan remains on track.\n\nDuring our meeting, we\'ll cover:\n- Portfolio performance review\n- Goal progress check\n- Life changes and updates\n- Strategy adjustments if needed\n\nPlease click here to book a time that works for you: [Scheduling Link]\n\nLooking forward to connecting,\n{advisor_name}',
    },
    requiredAdvisorSkills: [],
    complianceReviewRequired: false,
    successMetrics: { successMetric: 'meeting_scheduled', targetValue: 1 },
  },
];

// ============================================================================
// Helper Components
// ============================================================================

interface ImpactIndicatorProps {
  value: number;
  type: 'revenue' | 'satisfaction';
}

const ImpactIndicator: React.FC<ImpactIndicatorProps> = ({ value, type }) => {
  const barRef = React.useRef<HTMLDivElement>(null);
  
  const normalizedValue = type === 'revenue' 
    ? Math.min(value / 25000, 1) // Max $25k scale
    : value; // Already 0-1
  
  React.useEffect(() => {
    if (barRef.current) {
      barRef.current.style.width = `${normalizedValue * 100}%`;
    }
  }, [normalizedValue]);

  return (
    <div className="flex items-center gap-2">
      <div className="flex-1 h-1.5 bg-slate-200 rounded-full overflow-hidden">
        <div
          ref={barRef}
          className={`h-full ${type === 'revenue' ? 'bg-green-500' : 'bg-blue-500'} transition-all duration-300`}
        />
      </div>
      <span className="text-xs font-medium text-slate-600 w-16 text-right">
        {type === 'revenue' ? `$${value.toLocaleString()}` : `+${(value * 100).toFixed(0)}%`}
      </span>
    </div>
  );
};

// ============================================================================
// Action Detail Modal
// ============================================================================

interface ActionDetailModalProps {
  action: NBAAction;
  onClose: () => void;
  onEdit?: () => void;
}

const ActionDetailModal: React.FC<ActionDetailModalProps> = ({ action, onClose, onEdit }) => {
  const categoryConfig = CATEGORY_CONFIG[action.actionCategory];
  const channelConfig = CHANNEL_CONFIG[action.defaultChannel];
  const CategoryIcon = categoryConfig.icon;
  const ChannelIcon = channelConfig.icon;

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <div className="bg-white rounded-xl shadow-xl w-full max-w-3xl max-h-[90vh] overflow-y-auto">
        {/* Header */}
        <div className="p-6 border-b border-slate-200">
          <div className="flex items-start justify-between">
            <div className="flex items-center gap-4">
              <div className={`p-3 rounded-lg ${categoryConfig.bgColor}`}>
                <CategoryIcon className={`w-6 h-6 ${categoryConfig.color}`} />
              </div>
              <div>
                <h2 className="text-xl font-bold text-slate-900">{action.actionName}</h2>
                <p className="text-sm text-slate-500 mt-1">{action.actionCode}</p>
              </div>
            </div>
            <button
              onClick={onClose}
              className="p-2 hover:bg-slate-100 rounded-lg"
              title="Close"
            >
              <AlertCircle className="w-5 h-5 text-slate-400" />
            </button>
          </div>
        </div>

        <div className="p-6 space-y-6">
          {/* Description */}
          <div>
            <h3 className="font-semibold text-slate-900 mb-2">Description</h3>
            <p className="text-slate-600">{action.description}</p>
          </div>

          {/* Key Metrics */}
          <div className="grid grid-cols-2 gap-4">
            <div className="p-4 bg-slate-50 rounded-lg">
              <div className="flex items-center gap-2 mb-2">
                <DollarSign className="w-4 h-4 text-green-600" />
                <span className="text-sm font-medium text-slate-700">Revenue Impact</span>
              </div>
              <ImpactIndicator value={action.estimatedRevenueImpact} type="revenue" />
            </div>
            <div className="p-4 bg-slate-50 rounded-lg">
              <div className="flex items-center gap-2 mb-2">
                <TrendingUp className="w-4 h-4 text-blue-600" />
                <span className="text-sm font-medium text-slate-700">Client Value Impact</span>
              </div>
              <ImpactIndicator value={action.clientValueImpact} type="satisfaction" />
            </div>
          </div>

          {/* Channel & Duration */}
          <div className="flex items-center gap-6">
            <div className="flex items-center gap-2">
              <ChannelIcon className={`w-5 h-5 ${channelConfig.color}`} />
              <span className="text-sm text-slate-600">Default: {channelConfig.label}</span>
            </div>
            <div className="flex items-center gap-2">
              <Clock className="w-5 h-5 text-slate-400" />
              <span className="text-sm text-slate-600">{action.estimatedDurationMinutes} minutes</span>
            </div>
            {action.automationEligible && (
              <div className="flex items-center gap-2">
                <Zap className="w-5 h-5 text-amber-500" />
                <span className="text-sm text-amber-600 font-medium">Automation Eligible</span>
              </div>
            )}
            {action.complianceReviewRequired && (
              <div className="flex items-center gap-2">
                <Shield className="w-5 h-5 text-red-500" />
                <span className="text-sm text-red-600 font-medium">Compliance Review Required</span>
              </div>
            )}
          </div>

          {/* Required Skills */}
          {action.requiredAdvisorSkills.length > 0 && (
            <div>
              <h3 className="font-semibold text-slate-900 mb-2">Required Skills</h3>
              <div className="flex flex-wrap gap-2">
                {action.requiredAdvisorSkills.map(skill => (
                  <span key={skill} className="px-3 py-1 text-sm bg-indigo-100 text-indigo-700 rounded-full">
                    {skill.replace(/_/g, ' ')}
                  </span>
                ))}
              </div>
            </div>
          )}

          {/* Templates */}
          {action.templateContent.emailBody && (
            <div>
              <div className="flex items-center justify-between mb-2">
                <h3 className="font-semibold text-slate-900 flex items-center gap-2">
                  <Mail className="w-4 h-4" />
                  Email Template
                </h3>
                <button className="text-sm text-indigo-600 hover:text-indigo-700 flex items-center gap-1" title="Copy Template">
                  <Copy className="w-4 h-4" />
                  Copy
                </button>
              </div>
              <div className="bg-slate-50 rounded-lg p-4 border border-slate-200">
                <p className="text-sm font-medium text-slate-700 mb-2">
                  Subject: {action.templateContent.emailSubject}
                </p>
                <p className="text-sm text-slate-600 whitespace-pre-wrap">
                  {action.templateContent.emailBody}
                </p>
              </div>
            </div>
          )}

          {action.templateContent.callScript && (
            <div>
              <div className="flex items-center justify-between mb-2">
                <h3 className="font-semibold text-slate-900 flex items-center gap-2">
                  <Phone className="w-4 h-4" />
                  Call Script
                </h3>
                <button className="text-sm text-indigo-600 hover:text-indigo-700 flex items-center gap-1" title="Copy Script">
                  <Copy className="w-4 h-4" />
                  Copy
                </button>
              </div>
              <div className="bg-slate-50 rounded-lg p-4 border border-slate-200">
                <p className="text-sm text-slate-600 whitespace-pre-wrap">
                  {action.templateContent.callScript}
                </p>
              </div>
            </div>
          )}

          {action.templateContent.meetingAgenda && (
            <div>
              <h3 className="font-semibold text-slate-900 mb-2 flex items-center gap-2">
                <Target className="w-4 h-4" />
                Meeting Agenda
              </h3>
              <div className="bg-slate-50 rounded-lg p-4 border border-slate-200">
                <p className="text-sm text-slate-600 whitespace-pre-wrap">
                  {action.templateContent.meetingAgenda}
                </p>
              </div>
            </div>
          )}

          {/* Success Metrics */}
          <div>
            <h3 className="font-semibold text-slate-900 mb-2">Success Metrics</h3>
            <div className="p-4 bg-green-50 rounded-lg border border-green-200">
              <div className="flex items-center gap-2">
                <CheckCircle className="w-5 h-5 text-green-600" />
                <span className="text-sm text-slate-700">
                  <strong>{action.successMetrics.successMetric.replace(/_/g, ' ')}</strong> ≥{' '}
                  {typeof action.successMetrics.targetValue === 'number' && action.successMetrics.targetValue < 1
                    ? `${(action.successMetrics.targetValue * 100).toFixed(0)}%`
                    : action.successMetrics.targetValue.toLocaleString()}
                </span>
              </div>
            </div>
          </div>
        </div>

        {/* Footer */}
        <div className="p-6 border-t border-slate-200 flex gap-3">
          {onEdit && (
            <button
              onClick={onEdit}
              className="flex-1 px-4 py-2 text-sm font-medium text-indigo-600 bg-indigo-50 rounded-lg hover:bg-indigo-100 flex items-center justify-center gap-2"
              title="Edit Action"
            >
              <Edit className="w-4 h-4" />
              Edit Action
            </button>
          )}
          <button
            onClick={onClose}
            className="px-4 py-2 text-sm font-medium text-slate-700 bg-slate-100 rounded-lg hover:bg-slate-200"
            title="Close"
          >
            Close
          </button>
        </div>
      </div>
    </div>
  );
};

// ============================================================================
// Main Component
// ============================================================================

export const ActionCatalog: React.FC<ActionCatalogProps> = ({
  tenantId,
  datasourceId,
  onSelectAction,
  onEditAction,
}) => {
  // Suppress unused variable warnings
  void tenantId;
  void datasourceId;

  const [searchQuery, setSearchQuery] = useState('');
  const [selectedCategory, setSelectedCategory] = useState<ActionCategory | 'ALL'>('ALL');
  const [selectedChannel, setSelectedChannel] = useState<ActionChannel | 'ALL'>('ALL');
  const [selectedAction, setSelectedAction] = useState<NBAAction | null>(null);
  const [sortBy, setSortBy] = useState<'name' | 'revenue' | 'duration'>('name');

  // Filtered and sorted actions
  const filteredActions = useMemo(() => {
    let actions = MOCK_ACTIONS.filter(action => {
      if (selectedCategory !== 'ALL' && action.actionCategory !== selectedCategory) {
        return false;
      }
      if (selectedChannel !== 'ALL' && action.defaultChannel !== selectedChannel) {
        return false;
      }
      if (searchQuery) {
        const query = searchQuery.toLowerCase();
        return (
          action.actionName.toLowerCase().includes(query) ||
          action.description.toLowerCase().includes(query) ||
          action.actionCode.toLowerCase().includes(query)
        );
      }
      return true;
    });

    // Sort
    actions.sort((a, b) => {
      switch (sortBy) {
        case 'revenue':
          return b.estimatedRevenueImpact - a.estimatedRevenueImpact;
        case 'duration':
          return a.estimatedDurationMinutes - b.estimatedDurationMinutes;
        default:
          return a.actionName.localeCompare(b.actionName);
      }
    });

    return actions;
  }, [searchQuery, selectedCategory, selectedChannel, sortBy]);

  // Stats
  const stats = useMemo(() => ({
    totalActions: MOCK_ACTIONS.length,
    automationEligible: MOCK_ACTIONS.filter(a => a.automationEligible).length,
    avgDuration: Math.round(MOCK_ACTIONS.reduce((sum, a) => sum + a.estimatedDurationMinutes, 0) / MOCK_ACTIONS.length),
    avgRevenue: Math.round(MOCK_ACTIONS.reduce((sum, a) => sum + a.estimatedRevenueImpact, 0) / MOCK_ACTIONS.length),
  }), []);

  return (
    <div className="bg-slate-50 min-h-screen">
      {/* Header */}
      <div className="bg-white border-b border-slate-200 px-6 py-4">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-xl font-semibold text-slate-900 flex items-center gap-2">
              <BookOpen className="w-6 h-6 text-indigo-600" />
              Action Catalog
            </h1>
            <p className="text-sm text-slate-500 mt-1">
              Pre-configured action templates for NBA recommendations
            </p>
          </div>
          <div className="flex items-center gap-4 text-sm">
            <div className="flex items-center gap-2 text-slate-600">
              <Target className="w-4 h-4" />
              <span>{stats.totalActions} actions</span>
            </div>
            <div className="flex items-center gap-2 text-slate-600">
              <Zap className="w-4 h-4 text-amber-500" />
              <span>{stats.automationEligible} automatable</span>
            </div>
            <div className="flex items-center gap-2 text-slate-600">
              <Clock className="w-4 h-4" />
              <span>Avg {stats.avgDuration} min</span>
            </div>
            <div className="flex items-center gap-2 text-slate-600">
              <DollarSign className="w-4 h-4 text-green-500" />
              <span>Avg ${stats.avgRevenue.toLocaleString()}</span>
            </div>
          </div>
        </div>
      </div>

      {/* Filters */}
      <div className="bg-white border-b border-slate-200 px-6 py-3">
        <div className="flex items-center gap-4">
          {/* Search */}
          <div className="relative flex-1 max-w-md">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" />
            <input
              type="text"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder="Search actions..."
              className="w-full pl-10 pr-4 py-2 border border-slate-300 rounded-lg text-sm"
              title="Search actions"
            />
          </div>

          {/* Category filter */}
          <div className="flex items-center gap-2">
            <Filter className="w-4 h-4 text-slate-400" />
            <select
              value={selectedCategory}
              onChange={(e) => setSelectedCategory(e.target.value as ActionCategory | 'ALL')}
              className="px-3 py-2 border border-slate-300 rounded-lg text-sm"
              title="Filter by category"
            >
              <option value="ALL">All Categories</option>
              {Object.entries(CATEGORY_CONFIG).map(([key, config]) => (
                <option key={key} value={key}>{config.label}</option>
              ))}
            </select>
          </div>

          {/* Channel filter */}
          <select
            value={selectedChannel}
            onChange={(e) => setSelectedChannel(e.target.value as ActionChannel | 'ALL')}
            className="px-3 py-2 border border-slate-300 rounded-lg text-sm"
            title="Filter by channel"
          >
            <option value="ALL">All Channels</option>
            {Object.entries(CHANNEL_CONFIG).map(([key, config]) => (
              <option key={key} value={key}>{config.label}</option>
            ))}
          </select>

          {/* Sort */}
          <select
            value={sortBy}
            onChange={(e) => setSortBy(e.target.value as 'name' | 'revenue' | 'duration')}
            className="px-3 py-2 border border-slate-300 rounded-lg text-sm"
            title="Sort by"
          >
            <option value="name">Sort by Name</option>
            <option value="revenue">Sort by Revenue</option>
            <option value="duration">Sort by Duration</option>
          </select>
        </div>
      </div>

      {/* Action Grid */}
      <div className="px-6 py-4">
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {filteredActions.map(action => {
            const categoryConfig = CATEGORY_CONFIG[action.actionCategory];
            const channelConfig = CHANNEL_CONFIG[action.defaultChannel];
            const CategoryIcon = categoryConfig.icon;
            const ChannelIcon = channelConfig.icon;

            return (
              <div
                key={action.actionId}
                className="bg-white rounded-lg border border-slate-200 p-4 hover:shadow-md transition-shadow"
              >
                {/* Clickable Header Area */}
                <div
                  className="cursor-pointer"
                  onClick={() => setSelectedAction(action)}
                  onKeyDown={(e) => e.key === 'Enter' && setSelectedAction(action)}
                  tabIndex={0}
                  role="button"
                  aria-label={`View ${action.actionName} details`}
                >
                  {/* Header */}
                  <div className="flex items-start justify-between mb-3">
                    <div className="flex items-center gap-3">
                      <div className={`p-2 rounded-lg ${categoryConfig.bgColor}`}>
                        <CategoryIcon className={`w-5 h-5 ${categoryConfig.color}`} />
                      </div>
                      <div>
                        <h3 className="font-semibold text-slate-900 text-sm">{action.actionName}</h3>
                        <span className={`text-xs ${categoryConfig.color}`}>{categoryConfig.label}</span>
                      </div>
                    </div>
                    {action.automationEligible && (
                      <Zap className="w-4 h-4 text-amber-500" />
                    )}
                  </div>

                  {/* Description */}
                  <p className="text-xs text-slate-600 mb-3 line-clamp-2">{action.description}</p>

                  {/* Metrics */}
                  <div className="grid grid-cols-2 gap-2 mb-3">
                    <div className="text-xs">
                      <span className="text-slate-500">Revenue:</span>
                      <span className="ml-1 font-medium text-green-600">
                        ${action.estimatedRevenueImpact.toLocaleString()}
                      </span>
                    </div>
                    <div className="text-xs">
                      <span className="text-slate-500">Duration:</span>
                      <span className="ml-1 font-medium">
                        {action.estimatedDurationMinutes}m
                      </span>
                    </div>
                  </div>
                </div>

                {/* Footer - Outside clickable area */}
                <div className="flex items-center justify-between pt-3 border-t border-slate-100">
                  <div className={`flex items-center gap-1 text-xs ${channelConfig.color}`}>
                    <ChannelIcon className="w-3 h-3" />
                    {channelConfig.label}
                  </div>
                  <button
                    onClick={() => {
                      if (onSelectAction) onSelectAction(action);
                    }}
                    className="text-xs text-indigo-600 hover:text-indigo-700 flex items-center gap-1"
                    title="View Details"
                  >
                    View Details
                    <ChevronRight className="w-3 h-3" />
                  </button>
                </div>
              </div>
            );
          })}
        </div>

        {filteredActions.length === 0 && (
          <div className="text-center py-12">
            <BookOpen className="w-12 h-12 text-slate-300 mx-auto mb-3" />
            <p className="text-slate-600">No actions match your search</p>
            <p className="text-sm text-slate-400">Try adjusting your filters</p>
          </div>
        )}
      </div>

      {/* Action Detail Modal */}
      {selectedAction && (
        <ActionDetailModal
          action={selectedAction}
          onClose={() => setSelectedAction(null)}
          onEdit={onEditAction ? () => {
            if (onEditAction) onEditAction(selectedAction);
            setSelectedAction(null);
          } : undefined}
        />
      )}
    </div>
  );
};

export default ActionCatalog;
