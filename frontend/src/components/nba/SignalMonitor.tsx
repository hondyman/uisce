/**
 * Signal Monitor Component
 * 
 * Real-time multi-signal intelligence visualization for the NBA Engine.
 * Displays detected signals across 8 categories: CRM activity, portfolio events,
 * market conditions, life events, behavioral patterns, competitor intelligence,
 * social signals, and regulatory triggers.
 */

import React, { useState, useEffect, useMemo, useCallback } from 'react';
import {
  Activity,
  TrendingUp,
  AlertTriangle,
  Bell,
  Clock,
  Filter,
  RefreshCw,
  ChevronRight,
  Eye,
  Zap,
  Target,
  Users,
  BarChart3,
  Shield,
  Briefcase,
  Heart,
  Globe,
} from 'lucide-react';
import type {
  DetectedSignal,
  SignalCategory,
  SignalType,
  SignalSource,
} from '../../types/nba';

// ============================================================================
// Props
// ============================================================================

export interface SignalMonitorProps {
  tenantId: string;
  datasourceId: string;
  clientId?: string;
  limit?: number;
  onSignalClick?: (signal: DetectedSignal) => void;
  onGenerateAction?: (signal: DetectedSignal) => void;
}

// ============================================================================
// Constants
// ============================================================================

const SIGNAL_CATEGORY_CONFIG: Record<SignalCategory, {
  label: string;
  icon: React.ElementType;
  color: string;
  bgColor: string;
  description: string;
}> = {
  BEHAVIORAL: {
    label: 'Behavioral',
    icon: Activity,
    color: 'text-purple-700',
    bgColor: 'bg-purple-100',
    description: 'Communication patterns and engagement changes',
  },
  MARKET: {
    label: 'Market',
    icon: TrendingUp,
    color: 'text-blue-700',
    bgColor: 'bg-blue-100',
    description: 'Volatility, sector rotation, rate changes',
  },
  LIFECYCLE: {
    label: 'Lifecycle',
    icon: Heart,
    color: 'text-pink-700',
    bgColor: 'bg-pink-100',
    description: 'Life events like retirement, inheritance, job change',
  },
  PORTFOLIO: {
    label: 'Portfolio',
    icon: Briefcase,
    color: 'text-green-700',
    bgColor: 'bg-green-100',
    description: 'Large deposits/withdrawals, concentrated positions',
  },
  ENGAGEMENT: {
    label: 'Engagement',
    icon: Users,
    color: 'text-orange-700',
    bgColor: 'bg-orange-100',
    description: 'Portal logins, email opens, document views',
  },
};

// Signal source configuration (for future use in detailed views)
const _SIGNAL_SOURCE_CONFIG: Record<SignalSource, {
  label: string;
  icon: React.ElementType;
}> = {
  CRM_ACTIVITY: { label: 'CRM', icon: Users },
  PORTFOLIO_EVENTS: { label: 'Portfolio', icon: Briefcase },
  MARKET_CONDITIONS: { label: 'Market', icon: TrendingUp },
  LIFE_EVENTS: { label: 'Life Events', icon: Heart },
  BEHAVIORAL_PATTERNS: { label: 'Behavior', icon: Activity },
  COMPETITOR_INTELLIGENCE: { label: 'Competitor', icon: Target },
  SOCIAL_SIGNALS: { label: 'Social', icon: Globe },
  REGULATORY_TRIGGERS: { label: 'Regulatory', icon: Shield },
};

const SIGNAL_TYPE_CONFIG: Record<SignalType, {
  label: string;
  urgency: 'critical' | 'high' | 'medium' | 'low';
  description: string;
}> = {
  LARGE_WITHDRAWAL_PENDING: {
    label: 'Large Withdrawal Pending',
    urgency: 'critical',
    description: 'Client has a pending withdrawal over $50,000',
  },
  EMAIL_ENGAGEMENT_DROP: {
    label: 'Email Engagement Drop',
    urgency: 'medium',
    description: 'Open rate dropped below 20% in last 90 days',
  },
  CONCENTRATED_POSITION_ALERT: {
    label: 'Concentrated Position',
    urgency: 'high',
    description: 'Single position exceeds 25% of portfolio',
  },
  EXCESS_CASH_DRAG: {
    label: 'Excess Cash Drag',
    urgency: 'medium',
    description: 'Over 15% cash creating opportunity cost',
  },
  TAX_LOSS_HARVEST_OPPORTUNITY: {
    label: 'Tax Loss Harvest',
    urgency: 'high',
    description: 'Unrealized losses over $10,000 available',
  },
  CONCENTRATED_POSITION_RISK: {
    label: 'Position Risk',
    urgency: 'high',
    description: 'Single position exceeds 20% of portfolio',
  },
  ENGAGEMENT_DECLINE: {
    label: 'Engagement Decline',
    urgency: 'medium',
    description: 'Portal login frequency dropped 50%+',
  },
  LOW_EMAIL_ENGAGEMENT: {
    label: 'Low Email Open Rate',
    urgency: 'low',
    description: 'Email open rate below 20%',
  },
  VOLATILITY_EXPOSURE: {
    label: 'Volatility Exposure',
    urgency: 'high',
    description: 'High VIX with significant equity exposure',
  },
  RETIREMENT_APPROACHING: {
    label: 'Retirement Approaching',
    urgency: 'high',
    description: 'Client retirement date within 12 months',
  },
  INHERITANCE_DETECTED: {
    label: 'Inheritance Detected',
    urgency: 'critical',
    description: 'Large inflow detected, likely inheritance',
  },
  JOB_CHANGE_DETECTED: {
    label: 'Job Change Detected',
    urgency: 'medium',
    description: 'LinkedIn or CRM indicates job change',
  },
  ANNIVERSARY_UPCOMING: {
    label: 'Anniversary Upcoming',
    urgency: 'low',
    description: 'Client relationship anniversary soon',
  },
  REBALANCING_DUE: {
    label: 'Rebalancing Due',
    urgency: 'medium',
    description: 'Portfolio drift exceeds threshold',
  },
  COMPLIANCE_DEADLINE: {
    label: 'Compliance Deadline',
    urgency: 'critical',
    description: 'Regulatory filing or review deadline',
  },
};

// ============================================================================
// Mock Data
// ============================================================================

const generateMockSignals = (): DetectedSignal[] => {
  const now = new Date();
  return [
    {
      signalId: 's1',
      clientId: 'c1',
      clientName: 'John & Sarah Mitchell',
      signalType: 'TAX_LOSS_HARVEST_OPPORTUNITY',
      signalCategory: 'PORTFOLIO',
      signalSource: 'PORTFOLIO_EVENTS',
      detectedAt: new Date(now.getTime() - 2 * 60 * 60 * 1000).toISOString(),
      strength: 0.92,
      rawData: {
        totalUnrealizedLoss: -45000,
        estimatedTaxSavings: 16650,
        topLossPositions: ['TSLA', 'NVDA', 'META'],
      },
    },
    {
      signalId: 's2',
      clientId: 'c2',
      clientName: 'Robert Chen',
      signalType: 'RETIREMENT_APPROACHING',
      signalCategory: 'LIFECYCLE',
      signalSource: 'LIFE_EVENTS',
      detectedAt: new Date(now.getTime() - 4 * 60 * 60 * 1000).toISOString(),
      strength: 0.88,
      rawData: {
        retirementDate: '2026-03-15',
        daysUntil: 120,
        portfolioReadiness: 0.72,
      },
    },
    {
      signalId: 's3',
      clientId: 'c3',
      clientName: 'Jennifer Williams',
      signalType: 'ENGAGEMENT_DECLINE',
      signalCategory: 'BEHAVIORAL',
      signalSource: 'BEHAVIORAL_PATTERNS',
      detectedAt: new Date(now.getTime() - 6 * 60 * 60 * 1000).toISOString(),
      strength: 0.75,
      rawData: {
        recentLogins: 2,
        priorLogins: 8,
        declinePct: 75,
      },
    },
    {
      signalId: 's4',
      clientId: 'c4',
      clientName: 'Michael & Lisa Davis',
      signalType: 'CONCENTRATED_POSITION_RISK',
      signalCategory: 'PORTFOLIO',
      signalSource: 'PORTFOLIO_EVENTS',
      detectedAt: new Date(now.getTime() - 8 * 60 * 60 * 1000).toISOString(),
      strength: 0.85,
      rawData: {
        ticker: 'AAPL',
        positionPercent: 32,
        diversificationGap: 22,
      },
    },
    {
      signalId: 's5',
      clientId: 'c5',
      clientName: 'David Thompson',
      signalType: 'VOLATILITY_EXPOSURE',
      signalCategory: 'MARKET',
      signalSource: 'MARKET_CONDITIONS',
      detectedAt: new Date(now.getTime() - 12 * 60 * 60 * 1000).toISOString(),
      strength: 0.78,
      rawData: {
        currentVix: 35,
        equityExposure: 78,
        riskLevel: 'HIGH',
      },
    },
    {
      signalId: 's6',
      clientId: 'c6',
      clientName: 'Patricia Anderson',
      signalType: 'INHERITANCE_DETECTED',
      signalCategory: 'LIFECYCLE',
      signalSource: 'LIFE_EVENTS',
      detectedAt: new Date(now.getTime() - 24 * 60 * 60 * 1000).toISOString(),
      strength: 0.95,
      rawData: {
        incomingAmount: 850000,
        sourceType: 'estate_transfer',
        taxImplications: true,
      },
    },
  ];
};

// ============================================================================
// Helper Components
// ============================================================================

interface SignalStrengthBarProps {
  strength: number;
}

const SignalStrengthBar: React.FC<SignalStrengthBarProps> = ({ strength }) => {
  const barRef = React.useRef<HTMLDivElement>(null);
  
  React.useEffect(() => {
    if (barRef.current) {
      barRef.current.style.width = `${strength * 100}%`;
    }
  }, [strength]);

  const getColor = () => {
    if (strength >= 0.9) return 'bg-red-500';
    if (strength >= 0.75) return 'bg-orange-500';
    if (strength >= 0.5) return 'bg-yellow-500';
    return 'bg-green-500';
  };

  return (
    <div className="flex items-center gap-2">
      <div className="flex-1 h-1.5 bg-slate-200 rounded-full overflow-hidden">
        <div
          ref={barRef}
          className={`h-full ${getColor()} transition-all duration-300`}
        />
      </div>
      <span className="text-xs font-medium text-slate-600 w-10 text-right">
        {(strength * 100).toFixed(0)}%
      </span>
    </div>
  );
};

interface UrgencyBadgeProps {
  urgency: 'critical' | 'high' | 'medium' | 'low';
}

const UrgencyBadge: React.FC<UrgencyBadgeProps> = ({ urgency }) => {
  const config = {
    critical: { bg: 'bg-red-100', text: 'text-red-700', label: 'Critical' },
    high: { bg: 'bg-orange-100', text: 'text-orange-700', label: 'High' },
    medium: { bg: 'bg-yellow-100', text: 'text-yellow-700', label: 'Medium' },
    low: { bg: 'bg-green-100', text: 'text-green-700', label: 'Low' },
  };

  const { bg, text, label } = config[urgency];

  return (
    <span className={`px-2 py-0.5 text-xs font-medium rounded ${bg} ${text}`}>
      {label}
    </span>
  );
};

interface CategoryBadgeProps {
  category: SignalCategory;
  size?: 'sm' | 'md';
}

const CategoryBadge: React.FC<CategoryBadgeProps> = ({ category, size = 'sm' }) => {
  const config = SIGNAL_CATEGORY_CONFIG[category];
  const Icon = config.icon;

  return (
    <span className={`inline-flex items-center gap-1 px-2 py-0.5 rounded ${config.bgColor} ${config.color} ${size === 'sm' ? 'text-xs' : 'text-sm'}`}>
      <Icon className={size === 'sm' ? 'w-3 h-3' : 'w-4 h-4'} />
      {config.label}
    </span>
  );
};

// ============================================================================
// Signal Card Component
// ============================================================================

interface SignalCardProps {
  signal: DetectedSignal;
  onViewDetails: () => void;
  onGenerateAction: () => void;
}

const SignalCard: React.FC<SignalCardProps> = ({
  signal,
  onViewDetails,
  onGenerateAction,
}) => {
  const typeConfig = SIGNAL_TYPE_CONFIG[signal.signalType];
  const categoryConfig = SIGNAL_CATEGORY_CONFIG[signal.signalCategory];
  const CategoryIcon = categoryConfig.icon;

  const formatTimeAgo = (dateStr: string) => {
    const date = new Date(dateStr);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffHours = Math.floor(diffMs / (1000 * 60 * 60));
    const diffMins = Math.floor(diffMs / (1000 * 60));

    if (diffHours >= 24) return `${Math.floor(diffHours / 24)}d ago`;
    if (diffHours >= 1) return `${diffHours}h ago`;
    return `${diffMins}m ago`;
  };

  const renderInsights = () => {
    const data = signal.rawData;
    
    switch (signal.signalType) {
      case 'TAX_LOSS_HARVEST_OPPORTUNITY':
        return (
          <div className="grid grid-cols-2 gap-2 text-sm">
            <div>
              <span className="text-slate-500">Unrealized Loss:</span>
              <span className="ml-1 font-medium text-red-600">
                ${Math.abs(data.totalUnrealizedLoss as number).toLocaleString()}
              </span>
            </div>
            <div>
              <span className="text-slate-500">Est. Tax Savings:</span>
              <span className="ml-1 font-medium text-green-600">
                ${(data.estimatedTaxSavings as number).toLocaleString()}
              </span>
            </div>
          </div>
        );
      case 'CONCENTRATED_POSITION_RISK':
        return (
          <div className="grid grid-cols-2 gap-2 text-sm">
            <div>
              <span className="text-slate-500">Position:</span>
              <span className="ml-1 font-medium">{data.ticker as string}</span>
            </div>
            <div>
              <span className="text-slate-500">Concentration:</span>
              <span className="ml-1 font-medium text-orange-600">
                {data.positionPercent as number}%
              </span>
            </div>
          </div>
        );
      case 'RETIREMENT_APPROACHING':
        return (
          <div className="grid grid-cols-2 gap-2 text-sm">
            <div>
              <span className="text-slate-500">Days Until:</span>
              <span className="ml-1 font-medium">{data.daysUntil as number}</span>
            </div>
            <div>
              <span className="text-slate-500">Readiness:</span>
              <span className="ml-1 font-medium">
                {((data.portfolioReadiness as number) * 100).toFixed(0)}%
              </span>
            </div>
          </div>
        );
      case 'INHERITANCE_DETECTED':
        return (
          <div className="text-sm">
            <span className="text-slate-500">Incoming Amount:</span>
            <span className="ml-1 font-medium text-green-600">
              ${(data.incomingAmount as number).toLocaleString()}
            </span>
          </div>
        );
      default:
        return null;
    }
  };

  return (
    <div className="bg-white rounded-lg border border-slate-200 p-4 hover:shadow-md transition-shadow">
      <div className="flex items-start justify-between mb-3">
        <div className="flex items-center gap-3">
          <div className={`p-2 rounded-lg ${categoryConfig.bgColor}`}>
            <CategoryIcon className={`w-5 h-5 ${categoryConfig.color}`} />
          </div>
          <div>
            <h3 className="font-semibold text-slate-900">{signal.clientName}</h3>
            <p className="text-sm text-slate-500">{typeConfig.label}</p>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <UrgencyBadge urgency={typeConfig.urgency} />
          <span className="text-xs text-slate-400 flex items-center gap-1">
            <Clock className="w-3 h-3" />
            {formatTimeAgo(signal.detectedAt)}
          </span>
        </div>
      </div>

      <p className="text-sm text-slate-600 mb-3">{typeConfig.description}</p>

      {/* Signal-specific insights */}
      <div className="mb-3 p-2 bg-slate-50 rounded">
        {renderInsights()}
      </div>

      {/* Signal strength */}
      <div className="mb-3">
        <div className="flex items-center justify-between text-xs text-slate-500 mb-1">
          <span>Signal Strength</span>
          <CategoryBadge category={signal.signalCategory} />
        </div>
        <SignalStrengthBar strength={signal.strength} />
      </div>

      {/* Actions */}
      <div className="flex gap-2">
        <button
          onClick={onGenerateAction}
          className="flex-1 px-3 py-2 text-sm font-medium text-white bg-indigo-600 rounded-lg hover:bg-indigo-700 flex items-center justify-center gap-1"
          title="Generate Action"
        >
          <Zap className="w-4 h-4" />
          Generate Action
        </button>
        <button
          onClick={onViewDetails}
          className="px-3 py-2 text-sm font-medium text-slate-600 bg-slate-100 rounded-lg hover:bg-slate-200 flex items-center gap-1"
          title="View Details"
        >
          <Eye className="w-4 h-4" />
          Details
        </button>
      </div>
    </div>
  );
};

// ============================================================================
// Main Component
// ============================================================================

export const SignalMonitor: React.FC<SignalMonitorProps> = ({
  tenantId,
  datasourceId,
  clientId,
  limit = 20,
  onSignalClick,
  onGenerateAction,
}) => {
  // Suppress unused variable warnings
  void tenantId;
  void datasourceId;
  void clientId;
  void limit;

  const [signals, setSignals] = useState<DetectedSignal[]>([]);
  const [selectedCategory, setSelectedCategory] = useState<SignalCategory | 'ALL'>('ALL');
  const [selectedUrgency, setSelectedUrgency] = useState<'ALL' | 'critical' | 'high'>('ALL');
  const [isRefreshing, setIsRefreshing] = useState(false);
  const [viewMode, setViewMode] = useState<'cards' | 'list'>('cards');

  // Load mock data
  useEffect(() => {
    setSignals(generateMockSignals());
  }, []);

  // Filter signals
  const filteredSignals = useMemo(() => {
    return signals.filter(signal => {
      if (selectedCategory !== 'ALL' && signal.signalCategory !== selectedCategory) {
        return false;
      }
      if (selectedUrgency !== 'ALL') {
        const typeConfig = SIGNAL_TYPE_CONFIG[signal.signalType];
        if (selectedUrgency === 'critical' && typeConfig.urgency !== 'critical') return false;
        if (selectedUrgency === 'high' && !['critical', 'high'].includes(typeConfig.urgency)) return false;
      }
      return true;
    });
  }, [signals, selectedCategory, selectedUrgency]);

  // Stats
  const stats = useMemo(() => {
    const criticalCount = signals.filter(s => 
      SIGNAL_TYPE_CONFIG[s.signalType].urgency === 'critical'
    ).length;
    const highCount = signals.filter(s => 
      SIGNAL_TYPE_CONFIG[s.signalType].urgency === 'high'
    ).length;
    const avgStrength = signals.length > 0
      ? signals.reduce((sum, s) => sum + s.strength, 0) / signals.length
      : 0;

    return { criticalCount, highCount, avgStrength, totalCount: signals.length };
  }, [signals]);

  const handleRefresh = useCallback(async () => {
    setIsRefreshing(true);
    await new Promise(resolve => setTimeout(resolve, 1000));
    setSignals(generateMockSignals());
    setIsRefreshing(false);
  }, []);

  const handleSignalClick = useCallback((signal: DetectedSignal) => {
    if (onSignalClick) {
      onSignalClick(signal);
    }
  }, [onSignalClick]);

  const handleGenerateAction = useCallback((signal: DetectedSignal) => {
    if (onGenerateAction) {
      onGenerateAction(signal);
    }
  }, [onGenerateAction]);

  // ============================================================================
  // Render
  // ============================================================================

  return (
    <div className="bg-slate-50 min-h-screen">
      {/* Header */}
      <div className="bg-white border-b border-slate-200 px-6 py-4">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-xl font-semibold text-slate-900 flex items-center gap-2">
              <Activity className="w-6 h-6 text-indigo-600" />
              Signal Monitor
            </h1>
            <p className="text-sm text-slate-500 mt-1">
              Real-time multi-signal intelligence detection
            </p>
          </div>
          <div className="flex items-center gap-3">
            <button
              onClick={handleRefresh}
              disabled={isRefreshing}
              className="px-4 py-2 text-sm font-medium text-slate-700 bg-white border border-slate-300 rounded-lg hover:bg-slate-50 flex items-center gap-2"
              title="Refresh Signals"
            >
              <RefreshCw className={`w-4 h-4 ${isRefreshing ? 'animate-spin' : ''}`} />
              Refresh
            </button>
          </div>
        </div>
      </div>

      {/* Stats Bar */}
      <div className="bg-white border-b border-slate-200 px-6 py-3">
        <div className="flex items-center gap-6">
          <div className="flex items-center gap-2">
            <span className="w-8 h-8 bg-red-100 rounded-full flex items-center justify-center">
              <AlertTriangle className="w-4 h-4 text-red-600" />
            </span>
            <div>
              <p className="text-xs text-slate-500">Critical</p>
              <p className="text-lg font-semibold text-slate-900">{stats.criticalCount}</p>
            </div>
          </div>
          <div className="flex items-center gap-2">
            <span className="w-8 h-8 bg-orange-100 rounded-full flex items-center justify-center">
              <Bell className="w-4 h-4 text-orange-600" />
            </span>
            <div>
              <p className="text-xs text-slate-500">High Priority</p>
              <p className="text-lg font-semibold text-slate-900">{stats.highCount}</p>
            </div>
          </div>
          <div className="flex items-center gap-2">
            <span className="w-8 h-8 bg-blue-100 rounded-full flex items-center justify-center">
              <BarChart3 className="w-4 h-4 text-blue-600" />
            </span>
            <div>
              <p className="text-xs text-slate-500">Avg Strength</p>
              <p className="text-lg font-semibold text-slate-900">
                {(stats.avgStrength * 100).toFixed(0)}%
              </p>
            </div>
          </div>
          <div className="flex items-center gap-2">
            <span className="w-8 h-8 bg-slate-100 rounded-full flex items-center justify-center">
              <Zap className="w-4 h-4 text-slate-600" />
            </span>
            <div>
              <p className="text-xs text-slate-500">Total Signals</p>
              <p className="text-lg font-semibold text-slate-900">{stats.totalCount}</p>
            </div>
          </div>
        </div>
      </div>

      {/* Filters */}
      <div className="px-6 py-4 flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Filter className="w-4 h-4 text-slate-400" />
          <span className="text-sm text-slate-500">Filter:</span>
          
          {/* Category filter */}
          <select
            value={selectedCategory}
            onChange={(e) => setSelectedCategory(e.target.value as SignalCategory | 'ALL')}
            className="px-3 py-1.5 text-sm border border-slate-300 rounded-lg"
            title="Filter by category"
          >
            <option value="ALL">All Categories</option>
            {Object.entries(SIGNAL_CATEGORY_CONFIG).map(([key, config]) => (
              <option key={key} value={key}>{config.label}</option>
            ))}
          </select>

          {/* Urgency filter */}
          <select
            value={selectedUrgency}
            onChange={(e) => setSelectedUrgency(e.target.value as 'ALL' | 'critical' | 'high')}
            className="px-3 py-1.5 text-sm border border-slate-300 rounded-lg"
            title="Filter by urgency"
          >
            <option value="ALL">All Urgency</option>
            <option value="critical">Critical Only</option>
            <option value="high">High & Critical</option>
          </select>
        </div>

        <div className="flex items-center gap-2">
          <span className="text-sm text-slate-500">
            {filteredSignals.length} signals
          </span>
          <button
            onClick={() => setViewMode(viewMode === 'cards' ? 'list' : 'cards')}
            className="p-2 hover:bg-slate-100 rounded-lg"
            title={`Switch to ${viewMode === 'cards' ? 'list' : 'cards'} view`}
          >
            {viewMode === 'cards' ? (
              <BarChart3 className="w-4 h-4 text-slate-500" />
            ) : (
              <Target className="w-4 h-4 text-slate-500" />
            )}
          </button>
        </div>
      </div>

      {/* Signal Grid */}
      <div className="px-6 pb-6">
        {viewMode === 'cards' ? (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {filteredSignals.map(signal => (
              <SignalCard
                key={signal.signalId}
                signal={signal}
                onViewDetails={() => handleSignalClick(signal)}
                onGenerateAction={() => handleGenerateAction(signal)}
              />
            ))}
          </div>
        ) : (
          <div className="bg-white rounded-lg border border-slate-200">
            <table className="w-full">
              <thead>
                <tr className="border-b border-slate-200 bg-slate-50">
                  <th className="text-left py-3 px-4 text-sm font-medium text-slate-600">Client</th>
                  <th className="text-left py-3 px-4 text-sm font-medium text-slate-600">Signal</th>
                  <th className="text-left py-3 px-4 text-sm font-medium text-slate-600">Category</th>
                  <th className="text-left py-3 px-4 text-sm font-medium text-slate-600">Strength</th>
                  <th className="text-left py-3 px-4 text-sm font-medium text-slate-600">Urgency</th>
                  <th className="text-left py-3 px-4 text-sm font-medium text-slate-600">Detected</th>
                  <th className="text-right py-3 px-4 text-sm font-medium text-slate-600">Action</th>
                </tr>
              </thead>
              <tbody>
                {filteredSignals.map(signal => {
                  const typeConfig = SIGNAL_TYPE_CONFIG[signal.signalType];
                  return (
                    <tr key={signal.signalId} className="border-b border-slate-100 hover:bg-slate-50">
                      <td className="py-3 px-4">
                        <span className="font-medium text-slate-900">{signal.clientName}</span>
                      </td>
                      <td className="py-3 px-4">
                        <span className="text-sm text-slate-700">{typeConfig.label}</span>
                      </td>
                      <td className="py-3 px-4">
                        <CategoryBadge category={signal.signalCategory} />
                      </td>
                      <td className="py-3 px-4 w-32">
                        <SignalStrengthBar strength={signal.strength} />
                      </td>
                      <td className="py-3 px-4">
                        <UrgencyBadge urgency={typeConfig.urgency} />
                      </td>
                      <td className="py-3 px-4">
                        <span className="text-sm text-slate-500">
                          {new Date(signal.detectedAt).toLocaleString()}
                        </span>
                      </td>
                      <td className="py-3 px-4 text-right">
                        <button
                          onClick={() => handleGenerateAction(signal)}
                          className="px-3 py-1 text-sm font-medium text-indigo-600 hover:bg-indigo-50 rounded"
                          title="Generate Action"
                        >
                          <ChevronRight className="w-4 h-4" />
                        </button>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}

        {filteredSignals.length === 0 && (
          <div className="text-center py-12">
            <Activity className="w-12 h-12 text-slate-300 mx-auto mb-3" />
            <p className="text-slate-600">No signals match your filters</p>
            <p className="text-sm text-slate-400">Try adjusting your filter criteria</p>
          </div>
        )}
      </div>
    </div>
  );
};

export default SignalMonitor;
