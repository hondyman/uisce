/**
 * OutcomeTracker.tsx
 *
 * Track Action Execution Outcomes for ML Feedback Loop
 *
 * Features:
 * - Track execution outcomes (success, partial, failed)
 * - Client response tracking (positive, neutral, negative)
 * - Revenue attribution for ROI analysis
 * - Advisor feedback and ratings
 * - Historical outcome visualization
 * - ML model performance insights
 *
 * Purpose: Feeds outcome data back to NBA ML models for
 * continuous learning and recommendation improvement.
 */

import React, { useState, useMemo } from 'react';
import {
  CheckCircle,
  XCircle,
  AlertCircle,
  Clock,
  TrendingUp,
  TrendingDown,
  DollarSign,
  BarChart3,
  ThumbsUp,
  Minus,
  Calendar,
  Target,
  Award,
  Zap,
  Filter,
  Download,
  RefreshCw,
  ChevronDown,
  ChevronUp,
  Star,
  MessageSquare,
  Activity,
} from 'lucide-react';
import type {
  NBAActionOutcome,
  SignalType,
  ActionChannel,
  OutcomeStats,
} from '../../types/nba';

// ====================
// Types
// ====================

interface OutcomeTrackerProps {
  advisorId?: string;
  clientId?: string;
  period?: 'DAY' | 'WEEK' | 'MONTH' | 'QUARTER';
  onRefresh?: () => void;
}

type OutcomeFilter = 'ALL' | 'SUCCESS' | 'PARTIAL' | 'FAILED' | 'PENDING';
type TimeFilter = 'ALL' | 'TODAY' | 'WEEK' | 'MONTH';

// ====================
// Configuration
// ====================

const OUTCOME_CONFIG: Record<string, {
  icon: React.ElementType;
  label: string;
  color: string;
  bgColor: string;
}> = {
  SUCCESS: {
    icon: CheckCircle,
    label: 'Successful',
    color: 'text-green-600',
    bgColor: 'bg-green-100',
  },
  PARTIAL: {
    icon: AlertCircle,
    label: 'Partial',
    color: 'text-amber-600',
    bgColor: 'bg-amber-100',
  },
  FAILED: {
    icon: XCircle,
    label: 'Failed',
    color: 'text-red-600',
    bgColor: 'bg-red-100',
  },
  PENDING: {
    icon: Clock,
    label: 'Pending',
    color: 'text-slate-500',
    bgColor: 'bg-slate-100',
  },
};

const CHANNEL_CONFIG: Record<ActionChannel, { label: string; color: string }> = {
  EMAIL: { label: 'Email', color: 'text-blue-600' },
  PHONE: { label: 'Phone', color: 'text-green-600' },
  VIDEO_CALL: { label: 'Video', color: 'text-purple-600' },
  IN_PERSON: { label: 'Meeting', color: 'text-amber-600' },
  AUTOMATED_MESSAGE: { label: 'Auto', color: 'text-slate-600' },
  PORTAL_NOTIFICATION: { label: 'Portal', color: 'text-cyan-600' },
};

// ====================
// Mock Data
// ====================

function generateMockOutcomes(): NBAActionOutcome[] {
  const signalTypes: SignalType[] = [
    'LARGE_WITHDRAWAL_PENDING',
    'EMAIL_ENGAGEMENT_DROP',
    'TAX_LOSS_HARVEST_OPPORTUNITY',
    'RETIREMENT_APPROACHING',
    'REBALANCING_DUE',
  ];

  const channels: ActionChannel[] = ['EMAIL', 'PHONE', 'VIDEO_CALL', 'IN_PERSON'];

  return Array.from({ length: 15 }, (_, i) => {
    const isSuccess = Math.random() > 0.3;
    const isPartial = !isSuccess && Math.random() > 0.5;
    const now = Date.now();
    const daysAgo = Math.floor(Math.random() * 30);

    return {
      outcomeId: `outcome-${i + 1}`,
      actionId: `action-${i + 1}`,
      clientId: `client-${Math.floor(Math.random() * 10) + 1}`,
      advisorId: 'advisor-1',
      triggerSignalType: signalTypes[Math.floor(Math.random() * signalTypes.length)],
      recommendedAt: new Date(now - daysAgo * 24 * 60 * 60 * 1000).toISOString(),
      executedAt: new Date(now - (daysAgo - 1) * 24 * 60 * 60 * 1000).toISOString(),
      completedAt: isSuccess || isPartial
        ? new Date(now - (daysAgo - 2) * 24 * 60 * 60 * 1000).toISOString()
        : undefined,
      executionChannel: channels[Math.floor(Math.random() * channels.length)],
      clientResponded: isSuccess || isPartial,
      responseTimeHours: isSuccess ? Math.floor(Math.random() * 48) + 1 : undefined,
      actionSuccessful: isSuccess,
      revenueGenerated: isSuccess ? Math.floor(Math.random() * 50000) + 5000 : undefined,
      clientSatisfactionChange: isSuccess ? Math.random() * 0.3 : -Math.random() * 0.1,
      aumChange: isSuccess ? Math.floor(Math.random() * 200000) : undefined,
      advisorFeedback: isSuccess ? 'Client was very receptive to the recommendation.' : undefined,
      advisorRating: isSuccess ? Math.floor(Math.random() * 2) + 4 : Math.floor(Math.random() * 3) + 1,
      dismissReason: !isSuccess && !isPartial ? 'WRONG_TIMING' : undefined,
    };
  });
}

function calculateStats(outcomes: NBAActionOutcome[]): OutcomeStats {
  const executed = outcomes.filter(o => o.executedAt);
  const successful = executed.filter(o => o.actionSuccessful);
  const responseTimes = outcomes.filter(o => o.responseTimeHours).map(o => o.responseTimeHours!);
  const revenues = outcomes.filter(o => o.revenueGenerated).map(o => o.revenueGenerated!);
  const satChanges = outcomes.filter(o => o.clientSatisfactionChange !== undefined).map(o => o.clientSatisfactionChange!);

  return {
    totalActions: outcomes.length,
    executedActions: executed.length,
    successfulActions: successful.length,
    successRate: executed.length > 0 ? successful.length / executed.length : 0,
    avgResponseTimeHours: responseTimes.length > 0
      ? responseTimes.reduce((a, b) => a + b, 0) / responseTimes.length
      : 0,
    totalRevenueGenerated: revenues.reduce((a, b) => a + b, 0),
    avgSatisfactionChange: satChanges.length > 0
      ? satChanges.reduce((a, b) => a + b, 0) / satChanges.length
      : 0,
  };
}

// ====================
// Helper Components
// ====================

interface StatCardProps {
  title: string;
  value: string | number;
  subtitle?: string;
  icon: React.ElementType;
  trend?: 'up' | 'down' | 'neutral';
  trendValue?: string;
  color?: string;
}

function StatCard({ title, value, subtitle, icon: Icon, trend, trendValue, color = 'text-indigo-600' }: StatCardProps) {
  return (
    <div className="bg-white rounded-xl border border-slate-200 p-4">
      <div className="flex items-center justify-between mb-2">
        <span className="text-sm text-slate-500">{title}</span>
        <Icon className={`w-5 h-5 ${color}`} />
      </div>
      <div className="text-2xl font-bold text-slate-900">{value}</div>
      {(subtitle || trend) && (
        <div className="flex items-center gap-2 mt-1">
          {subtitle && <span className="text-xs text-slate-500">{subtitle}</span>}
          {trend && trendValue && (
            <span className={`flex items-center gap-0.5 text-xs ${
              trend === 'up' ? 'text-green-600' : trend === 'down' ? 'text-red-600' : 'text-slate-500'
            }`}>
              {trend === 'up' ? <TrendingUp className="w-3 h-3" /> :
               trend === 'down' ? <TrendingDown className="w-3 h-3" /> :
               <Minus className="w-3 h-3" />}
              {trendValue}
            </span>
          )}
        </div>
      )}
    </div>
  );
}

interface ProgressBarProps {
  percentage: number;
  colorClass: string;
}

function ProgressBar({ percentage, colorClass }: ProgressBarProps) {
  // Use discrete width classes to avoid inline styles
  const getWidthClass = () => {
    if (percentage >= 95) return 'w-full';
    if (percentage >= 90) return 'w-11/12';
    if (percentage >= 85) return 'w-10/12';
    if (percentage >= 75) return 'w-3/4';
    if (percentage >= 65) return 'w-2/3';
    if (percentage >= 50) return 'w-1/2';
    if (percentage >= 35) return 'w-1/3';
    if (percentage >= 25) return 'w-1/4';
    if (percentage >= 15) return 'w-1/6';
    if (percentage >= 10) return 'w-1/12';
    return 'w-1';
  };

  return (
    <div className="h-2 bg-slate-100 rounded-full overflow-hidden">
      <div className={`h-full rounded-full ${colorClass} ${getWidthClass()}`} />
    </div>
  );
}

interface OutcomeRowProps {
  outcome: NBAActionOutcome;
  onViewDetails?: () => void;
}

function OutcomeRow({ outcome, onViewDetails }: OutcomeRowProps) {
  const [expanded, setExpanded] = useState(false);
  const status = outcome.actionSuccessful ? 'SUCCESS' : outcome.completedAt ? 'PARTIAL' : outcome.executedAt ? 'PENDING' : 'FAILED';
  const config = OUTCOME_CONFIG[status];
  const StatusIcon = config.icon;

  return (
    <div className="border-b border-slate-100 last:border-b-0">
      <div
        className="flex items-center gap-4 p-4 hover:bg-slate-50 cursor-pointer"
        onClick={() => setExpanded(!expanded)}
        role="button"
        tabIndex={0}
        onKeyDown={(e) => e.key === 'Enter' && setExpanded(!expanded)}
        aria-label={`View outcome details for ${outcome.actionId}`}
      >
        <div className={`p-2 rounded-lg ${config.bgColor}`}>
          <StatusIcon className={`w-4 h-4 ${config.color}`} />
        </div>

        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <span className="text-sm font-medium text-slate-900 truncate">
              Action #{outcome.actionId.split('-')[1]}
            </span>
            <span className={`text-xs px-2 py-0.5 rounded-full ${config.bgColor} ${config.color}`}>
              {config.label}
            </span>
          </div>
          <div className="flex items-center gap-3 text-xs text-slate-500 mt-1">
            <span className="flex items-center gap-1">
              <Zap className="w-3 h-3" />
              {outcome.triggerSignalType.replace(/_/g, ' ')}
            </span>
            <span className="flex items-center gap-1">
              <Calendar className="w-3 h-3" />
              {new Date(outcome.recommendedAt).toLocaleDateString()}
            </span>
            {outcome.executionChannel && (
              <span className={CHANNEL_CONFIG[outcome.executionChannel].color}>
                {CHANNEL_CONFIG[outcome.executionChannel].label}
              </span>
            )}
          </div>
        </div>

        <div className="text-right">
          {outcome.revenueGenerated && (
            <div className="text-sm font-bold text-green-600">
              +${outcome.revenueGenerated.toLocaleString()}
            </div>
          )}
          {outcome.responseTimeHours && (
            <div className="text-xs text-slate-500">
              Response: {outcome.responseTimeHours}h
            </div>
          )}
        </div>

        <div className="flex items-center gap-2">
          {outcome.advisorRating && (
            <div className="flex items-center gap-0.5">
              {[1, 2, 3, 4, 5].map(star => (
                <Star
                  key={star}
                  className={`w-3 h-3 ${
                    star <= outcome.advisorRating! ? 'text-amber-400 fill-amber-400' : 'text-slate-300'
                  }`}
                />
              ))}
            </div>
          )}
          {expanded ? (
            <ChevronUp className="w-4 h-4 text-slate-400" />
          ) : (
            <ChevronDown className="w-4 h-4 text-slate-400" />
          )}
        </div>
      </div>

      {expanded && (
        <div className="px-4 pb-4 ml-12 bg-slate-50 rounded-b-lg">
          <div className="grid grid-cols-3 gap-4 mb-3">
            <div>
              <span className="text-xs text-slate-500 block mb-1">Client Response</span>
              <span className={`flex items-center gap-1 text-sm font-medium ${
                outcome.clientResponded ? 'text-green-600' : 'text-slate-500'
              }`}>
                {outcome.clientResponded ? (
                  <><ThumbsUp className="w-4 h-4" /> Responded</>
                ) : (
                  <><Minus className="w-4 h-4" /> No Response</>
                )}
              </span>
            </div>
            <div>
              <span className="text-xs text-slate-500 block mb-1">Satisfaction Change</span>
              <span className={`flex items-center gap-1 text-sm font-medium ${
                (outcome.clientSatisfactionChange ?? 0) >= 0 ? 'text-green-600' : 'text-red-600'
              }`}>
                {(outcome.clientSatisfactionChange ?? 0) >= 0 ? (
                  <TrendingUp className="w-4 h-4" />
                ) : (
                  <TrendingDown className="w-4 h-4" />
                )}
                {((outcome.clientSatisfactionChange ?? 0) * 100).toFixed(1)}%
              </span>
            </div>
            <div>
              <span className="text-xs text-slate-500 block mb-1">AUM Change</span>
              <span className="text-sm font-medium text-slate-700">
                {outcome.aumChange
                  ? `+$${outcome.aumChange.toLocaleString()}`
                  : '-'}
              </span>
            </div>
          </div>

          {outcome.advisorFeedback && (
            <div className="bg-white rounded-lg p-3">
              <div className="flex items-center gap-2 text-xs text-slate-500 mb-1">
                <MessageSquare className="w-3 h-3" />
                Advisor Notes
              </div>
              <p className="text-sm text-slate-700">{outcome.advisorFeedback}</p>
            </div>
          )}

          {outcome.dismissReason && (
            <div className="bg-amber-50 rounded-lg p-3 mt-2">
              <div className="flex items-center gap-2 text-xs text-amber-600">
                <AlertCircle className="w-3 h-3" />
                Dismiss Reason: {outcome.dismissReason.replace(/_/g, ' ')}
              </div>
            </div>
          )}

          {onViewDetails && (
            <button
              onClick={onViewDetails}
              className="mt-3 text-xs text-indigo-600 hover:text-indigo-700"
              title="View full details"
            >
              View Full Details →
            </button>
          )}
        </div>
      )}
    </div>
  );
}

interface FilterButtonProps {
  label: string;
  isActive: boolean;
  onClick: () => void;
  count?: number;
}

function FilterButton({ label, isActive, onClick, count }: FilterButtonProps) {
  return (
    <button
      onClick={onClick}
      className={`px-3 py-1.5 rounded-lg text-sm transition-colors ${
        isActive
          ? 'bg-indigo-100 text-indigo-700 font-medium'
          : 'text-slate-600 hover:bg-slate-100'
      }`}
      title={`Filter by ${label}`}
    >
      {label}
      {count !== undefined && (
        <span className={`ml-1.5 px-1.5 py-0.5 rounded-full text-xs ${
          isActive ? 'bg-indigo-200' : 'bg-slate-200'
        }`}>
          {count}
        </span>
      )}
    </button>
  );
}

// ====================
// Main Component
// ====================

export function OutcomeTracker({
  onRefresh,
}: OutcomeTrackerProps) {
  const [outcomes] = useState<NBAActionOutcome[]>(generateMockOutcomes);
  const [outcomeFilter, setOutcomeFilter] = useState<OutcomeFilter>('ALL');
  const [timeFilter, setTimeFilter] = useState<TimeFilter>('ALL');

  const stats = useMemo(() => calculateStats(outcomes), [outcomes]);

  const filteredOutcomes = useMemo(() => {
    let filtered = [...outcomes];

    // Apply outcome filter
    if (outcomeFilter !== 'ALL') {
      filtered = filtered.filter(o => {
        const status = o.actionSuccessful ? 'SUCCESS' : o.completedAt ? 'PARTIAL' : o.executedAt ? 'PENDING' : 'FAILED';
        return status === outcomeFilter;
      });
    }

    // Apply time filter
    if (timeFilter !== 'ALL') {
      const now = new Date();
      const cutoff = timeFilter === 'TODAY' ? 1 :
                     timeFilter === 'WEEK' ? 7 : 30;
      filtered = filtered.filter(o => {
        const date = new Date(o.recommendedAt);
        const diffDays = (now.getTime() - date.getTime()) / (1000 * 60 * 60 * 24);
        return diffDays <= cutoff;
      });
    }

    return filtered.sort((a, b) =>
      new Date(b.recommendedAt).getTime() - new Date(a.recommendedAt).getTime()
    );
  }, [outcomes, outcomeFilter, timeFilter]);

  const outcomeCounts = useMemo(() => {
    const counts = { SUCCESS: 0, PARTIAL: 0, FAILED: 0, PENDING: 0, ALL: outcomes.length };
    outcomes.forEach(o => {
      const status = o.actionSuccessful ? 'SUCCESS' : o.completedAt ? 'PARTIAL' : o.executedAt ? 'PENDING' : 'FAILED';
      counts[status]++;
    });
    return counts;
  }, [outcomes]);

  return (
    <div className="bg-slate-50 min-h-screen p-6">
      <div className="max-w-7xl mx-auto">
        {/* Header */}
        <div className="flex items-center justify-between mb-6">
          <div>
            <h1 className="text-2xl font-bold text-slate-900">Outcome Tracker</h1>
            <p className="text-slate-600">Track action outcomes for ML model improvement</p>
          </div>
          <div className="flex items-center gap-3">
            <button
              onClick={onRefresh}
              className="flex items-center gap-2 px-4 py-2 border border-slate-300 rounded-lg hover:bg-white"
              title="Refresh outcomes"
            >
              <RefreshCw className="w-4 h-4" />
              Refresh
            </button>
            <button
              className="flex items-center gap-2 px-4 py-2 border border-slate-300 rounded-lg hover:bg-white"
              title="Export outcomes"
            >
              <Download className="w-4 h-4" />
              Export
            </button>
          </div>
        </div>

        {/* Stats Cards */}
        <div className="grid grid-cols-4 gap-4 mb-6">
          <StatCard
            title="Total Actions"
            value={stats.totalActions}
            subtitle={`${stats.executedActions} executed`}
            icon={Activity}
            color="text-indigo-600"
          />
          <StatCard
            title="Success Rate"
            value={`${Math.round(stats.successRate * 100)}%`}
            icon={Target}
            trend={stats.successRate >= 0.7 ? 'up' : 'down'}
            trendValue={stats.successRate >= 0.7 ? 'Above target' : 'Below target'}
            color="text-green-600"
          />
          <StatCard
            title="Total Revenue"
            value={`$${stats.totalRevenueGenerated.toLocaleString()}`}
            icon={DollarSign}
            trend="up"
            trendValue="+12% vs last period"
            color="text-emerald-600"
          />
          <StatCard
            title="Avg Response Time"
            value={`${stats.avgResponseTimeHours.toFixed(1)}h`}
            icon={Clock}
            trend={stats.avgResponseTimeHours < 24 ? 'up' : 'down'}
            trendValue={stats.avgResponseTimeHours < 24 ? 'Fast' : 'Needs improvement'}
            color="text-amber-600"
          />
        </div>

        {/* Performance Insights */}
        <div className="grid grid-cols-3 gap-6 mb-6">
          {/* Success by Channel */}
          <div className="bg-white rounded-xl border border-slate-200 p-4">
            <h3 className="font-semibold text-slate-900 mb-4 flex items-center gap-2">
              <BarChart3 className="w-4 h-4 text-indigo-600" />
              Success by Channel
            </h3>
            <div className="space-y-3">
              {(['EMAIL', 'PHONE', 'VIDEO_CALL', 'IN_PERSON'] as ActionChannel[]).map(channel => {
                const channelOutcomes = outcomes.filter(o => o.executionChannel === channel);
                const successCount = channelOutcomes.filter(o => o.actionSuccessful).length;
                const rate = channelOutcomes.length > 0 ? successCount / channelOutcomes.length : 0;
                const colorClass = rate >= 0.7 ? 'bg-green-500' :
                                   rate >= 0.5 ? 'bg-amber-500' : 'bg-red-500';

                return (
                  <div key={channel}>
                    <div className="flex items-center justify-between text-sm mb-1">
                      <span className={CHANNEL_CONFIG[channel].color}>
                        {CHANNEL_CONFIG[channel].label}
                      </span>
                      <span className="text-slate-600">{Math.round(rate * 100)}%</span>
                    </div>
                    <ProgressBar percentage={rate * 100} colorClass={colorClass} />
                  </div>
                );
              })}
            </div>
          </div>

          {/* Top Performing Signals */}
          <div className="bg-white rounded-xl border border-slate-200 p-4">
            <h3 className="font-semibold text-slate-900 mb-4 flex items-center gap-2">
              <Zap className="w-4 h-4 text-amber-600" />
              Top Performing Signals
            </h3>
            <div className="space-y-3">
              {(['TAX_LOSS_HARVEST_OPPORTUNITY', 'RETIREMENT_APPROACHING', 'REBALANCING_DUE'] as SignalType[]).map((signal, index) => {
                const signalOutcomes = outcomes.filter(o => o.triggerSignalType === signal);
                const successCount = signalOutcomes.filter(o => o.actionSuccessful).length;
                const rate = signalOutcomes.length > 0 ? successCount / signalOutcomes.length : 0;

                return (
                  <div key={signal} className="flex items-center gap-3">
                    <div className={`w-6 h-6 rounded-full flex items-center justify-center text-xs font-bold ${
                      index === 0 ? 'bg-amber-100 text-amber-700' :
                      index === 1 ? 'bg-slate-200 text-slate-700' :
                      'bg-orange-100 text-orange-700'
                    }`}>
                      {index + 1}
                    </div>
                    <div className="flex-1">
                      <div className="text-sm text-slate-700 truncate">
                        {signal.replace(/_/g, ' ')}
                      </div>
                      <div className="text-xs text-slate-500">
                        {Math.round(rate * 100)}% success rate
                      </div>
                    </div>
                  </div>
                );
              })}
            </div>
          </div>

          {/* Recent Feedback */}
          <div className="bg-white rounded-xl border border-slate-200 p-4">
            <h3 className="font-semibold text-slate-900 mb-4 flex items-center gap-2">
              <MessageSquare className="w-4 h-4 text-purple-600" />
              Recent Advisor Feedback
            </h3>
            <div className="space-y-3">
              {outcomes
                .filter(o => o.advisorFeedback)
                .slice(0, 3)
                .map((outcome, index) => (
                  <div key={index} className="bg-slate-50 rounded-lg p-3">
                    <div className="flex items-center gap-2 mb-1">
                      <div className="flex items-center gap-0.5">
                        {[1, 2, 3, 4, 5].map(star => (
                          <Star
                            key={star}
                            className={`w-3 h-3 ${
                              star <= (outcome.advisorRating || 0)
                                ? 'text-amber-400 fill-amber-400'
                                : 'text-slate-300'
                            }`}
                          />
                        ))}
                      </div>
                    </div>
                    <p className="text-xs text-slate-600 line-clamp-2">
                      {outcome.advisorFeedback}
                    </p>
                  </div>
                ))}
            </div>
          </div>
        </div>

        {/* Outcomes List */}
        <div className="bg-white rounded-xl border border-slate-200 overflow-hidden">
          {/* Filters */}
          <div className="px-4 py-3 border-b border-slate-200 flex items-center justify-between">
            <div className="flex items-center gap-2">
              <Filter className="w-4 h-4 text-slate-400" />
              <span className="text-sm text-slate-600 mr-2">Status:</span>
              {(['ALL', 'SUCCESS', 'PARTIAL', 'PENDING', 'FAILED'] as OutcomeFilter[]).map(filter => (
                <FilterButton
                  key={filter}
                  label={filter === 'ALL' ? 'All' : OUTCOME_CONFIG[filter]?.label || filter}
                  isActive={outcomeFilter === filter}
                  onClick={() => setOutcomeFilter(filter)}
                  count={outcomeCounts[filter]}
                />
              ))}
            </div>
            <div className="flex items-center gap-2">
              <span className="text-sm text-slate-600 mr-2">Time:</span>
              {(['ALL', 'TODAY', 'WEEK', 'MONTH'] as TimeFilter[]).map(filter => (
                <FilterButton
                  key={filter}
                  label={filter === 'ALL' ? 'All Time' :
                         filter === 'TODAY' ? 'Today' :
                         filter === 'WEEK' ? 'This Week' : 'This Month'}
                  isActive={timeFilter === filter}
                  onClick={() => setTimeFilter(filter)}
                />
              ))}
            </div>
          </div>

          {/* Outcomes */}
          <div className="divide-y divide-slate-100">
            {filteredOutcomes.length > 0 ? (
              filteredOutcomes.map(outcome => (
                <OutcomeRow
                  key={outcome.outcomeId}
                  outcome={outcome}
                />
              ))
            ) : (
              <div className="p-12 text-center">
                <Award className="w-12 h-12 text-slate-300 mx-auto mb-3" />
                <p className="text-slate-600">No outcomes match your filters</p>
                <p className="text-sm text-slate-400">Try adjusting your filter criteria</p>
              </div>
            )}
          </div>
        </div>

        {/* ML Model Insights */}
        <div className="mt-6 bg-gradient-to-r from-indigo-50 to-purple-50 rounded-xl border border-indigo-100 p-6">
          <div className="flex items-center gap-3 mb-4">
            <div className="p-2 bg-indigo-100 rounded-lg">
              <BarChart3 className="w-5 h-5 text-indigo-600" />
            </div>
            <div>
              <h3 className="font-semibold text-slate-900">ML Model Insights</h3>
              <p className="text-sm text-slate-600">Performance metrics feeding into model training</p>
            </div>
          </div>
          <div className="grid grid-cols-4 gap-4">
            <div className="bg-white rounded-lg p-3">
              <div className="text-xs text-slate-500 mb-1">Prediction Accuracy</div>
              <div className="text-xl font-bold text-green-600">87.3%</div>
            </div>
            <div className="bg-white rounded-lg p-3">
              <div className="text-xs text-slate-500 mb-1">F1 Score</div>
              <div className="text-xl font-bold text-blue-600">0.84</div>
            </div>
            <div className="bg-white rounded-lg p-3">
              <div className="text-xs text-slate-500 mb-1">Model Version</div>
              <div className="text-xl font-bold text-purple-600">v2.4.1</div>
            </div>
            <div className="bg-white rounded-lg p-3">
              <div className="text-xs text-slate-500 mb-1">Last Training</div>
              <div className="text-xl font-bold text-slate-700">2h ago</div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

export default OutcomeTracker;
