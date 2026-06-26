/**
 * RecommendationCard.tsx
 *
 * Individual NBA Recommendation Card Component
 *
 * Features:
 * - AI confidence visualization with animated gauges
 * - Expected value calculation display
 * - Signal context with decay indicators
 * - Quick action buttons (Execute, Dismiss, Schedule)
 * - Time-sensitive urgency indicators
 * - Client tier and relationship context
 */

import React, { useState } from 'react';
import {
  Phone,
  Mail,
  Video,
  User,
  Clock,
  CheckCircle,
  AlertTriangle,
  XCircle,
  TrendingUp,
  DollarSign,
  Target,
  ChevronDown,
  ChevronUp,
  Calendar,
  Zap,
  Shield,
  ThumbsDown,
  Bell,
  MessageSquare,
  ExternalLink,
  Award,
} from 'lucide-react';
import type {
  NextBestAction,
  ActionChannel,
  ActionPriority,
} from '../../types/nba';

// ====================
// Types
// ====================

interface RecommendationCardProps {
  recommendation: NextBestAction;
  onExecute: () => void;
  onDismiss: (reason: string) => void;
  onSchedule?: (time: Date) => void;
  onViewDetails?: () => void;
  compact?: boolean;
}

// ====================
// Channel & Priority Configuration
// ====================

const CHANNEL_CONFIG: Record<ActionChannel, {
  icon: React.ElementType;
  label: string;
  color: string;
}> = {
  EMAIL: { icon: Mail, label: 'Email', color: 'text-blue-600' },
  PHONE: { icon: Phone, label: 'Phone', color: 'text-green-600' },
  VIDEO_CALL: { icon: Video, label: 'Video', color: 'text-purple-600' },
  IN_PERSON: { icon: User, label: 'Meeting', color: 'text-amber-600' },
  AUTOMATED_MESSAGE: { icon: Bell, label: 'Auto', color: 'text-slate-600' },
  PORTAL_NOTIFICATION: { icon: MessageSquare, label: 'Portal', color: 'text-cyan-600' },
};

const PRIORITY_CONFIG: Record<ActionPriority, {
  label: string;
  color: string;
  bgColor: string;
  borderColor: string;
  icon: React.ElementType;
}> = {
  CRITICAL: {
    label: 'Critical',
    color: 'text-red-700',
    bgColor: 'bg-red-50',
    borderColor: 'border-red-200',
    icon: AlertTriangle,
  },
  HIGH: {
    label: 'High',
    color: 'text-orange-700',
    bgColor: 'bg-orange-50',
    borderColor: 'border-orange-200',
    icon: TrendingUp,
  },
  MEDIUM: {
    label: 'Medium',
    color: 'text-amber-700',
    bgColor: 'bg-amber-50',
    borderColor: 'border-amber-200',
    icon: Target,
  },
  LOW: {
    label: 'Low',
    color: 'text-slate-600',
    bgColor: 'bg-slate-50',
    borderColor: 'border-slate-200',
    icon: Clock,
  },
  OPTIONAL: {
    label: 'Optional',
    color: 'text-slate-500',
    bgColor: 'bg-slate-50',
    borderColor: 'border-slate-200',
    icon: CheckCircle,
  },
};

const CLIENT_TIER_CONFIG: Record<string, {
  label: string;
  color: string;
  bgColor: string;
  icon: React.ElementType;
}> = {
  VIP: { label: 'VIP', color: 'text-purple-700', bgColor: 'bg-purple-100', icon: Award },
  HIGH_NET_WORTH: { label: 'HNW', color: 'text-indigo-700', bgColor: 'bg-indigo-100', icon: Shield },
  STANDARD: { label: 'Standard', color: 'text-slate-600', bgColor: 'bg-slate-100', icon: User },
};

// ====================
// Helper Components
// ====================

interface ConfidenceGaugeProps {
  confidence: number;
  size?: 'sm' | 'md' | 'lg';
}

function ConfidenceGauge({ confidence, size = 'md' }: ConfidenceGaugeProps) {
  const radius = size === 'sm' ? 20 : size === 'md' ? 28 : 36;
  const strokeWidth = size === 'sm' ? 4 : size === 'md' ? 5 : 6;
  const circumference = 2 * Math.PI * radius;
  const progress = confidence * circumference;

  const getColor = () => {
    if (confidence >= 0.8) return '#10B981'; // green
    if (confidence >= 0.6) return '#F59E0B'; // amber
    return '#EF4444'; // red
  };

  return (
    <div className="relative inline-flex items-center justify-center">
      <svg
        className="transform -rotate-90"
        width={radius * 2 + strokeWidth * 2}
        height={radius * 2 + strokeWidth * 2}
      >
        <circle
          className="text-slate-200"
          stroke="currentColor"
          strokeWidth={strokeWidth}
          fill="transparent"
          r={radius}
          cx={radius + strokeWidth}
          cy={radius + strokeWidth}
        />
        <circle
          stroke={getColor()}
          strokeWidth={strokeWidth}
          strokeLinecap="round"
          fill="transparent"
          r={radius}
          cx={radius + strokeWidth}
          cy={radius + strokeWidth}
          strokeDasharray={circumference}
          strokeDashoffset={circumference - progress}
          className="transition-all duration-500"
        />
      </svg>
      <span className={`absolute font-bold ${
        size === 'sm' ? 'text-xs' : size === 'md' ? 'text-sm' : 'text-base'
      }`}>
        {Math.round(confidence * 100)}%
      </span>
    </div>
  );
}

interface TimeRemainingProps {
  expiresAt?: string;
}

function TimeRemaining({ expiresAt }: TimeRemainingProps) {
  if (!expiresAt) return null;

  const now = new Date();
  const expires = new Date(expiresAt);
  const diffMs = expires.getTime() - now.getTime();
  const diffHours = Math.floor(diffMs / (1000 * 60 * 60));
  const diffDays = Math.floor(diffHours / 24);

  if (diffMs < 0) {
    return (
      <span className="flex items-center gap-1 text-xs text-red-600">
        <XCircle className="w-3 h-3" />
        Expired
      </span>
    );
  }

  if (diffHours < 24) {
    return (
      <span className="flex items-center gap-1 text-xs text-red-600 animate-pulse">
        <Clock className="w-3 h-3" />
        {diffHours}h remaining
      </span>
    );
  }

  return (
    <span className="flex items-center gap-1 text-xs text-amber-600">
      <Clock className="w-3 h-3" />
      {diffDays}d remaining
    </span>
  );
}

interface DismissMenuProps {
  onDismiss: (reason: string) => void;
  onClose: () => void;
}

function DismissMenu({ onDismiss, onClose }: DismissMenuProps) {
  const reasons = [
    { code: 'NOT_RELEVANT', label: 'Not relevant to client' },
    { code: 'ALREADY_DONE', label: 'Already completed' },
    { code: 'WRONG_TIMING', label: 'Wrong timing' },
    { code: 'CLIENT_OPTED_OUT', label: 'Client opted out' },
    { code: 'OTHER', label: 'Other reason' },
  ];

  return (
    <div className="absolute right-0 top-full mt-1 bg-white rounded-lg shadow-lg border border-slate-200 py-1 z-10 min-w-48">
      <div className="px-3 py-2 text-xs font-medium text-slate-500 border-b border-slate-100">
        Dismiss reason
      </div>
      {reasons.map(reason => (
        <button
          key={reason.code}
          onClick={() => {
            onDismiss(reason.code);
            onClose();
          }}
          className="w-full px-3 py-2 text-left text-sm text-slate-700 hover:bg-slate-50"
          title={`Dismiss: ${reason.label}`}
        >
          {reason.label}
        </button>
      ))}
    </div>
  );
}

// ====================
// Main Component
// ====================

export function RecommendationCard({
  recommendation,
  onExecute,
  onDismiss,
  onSchedule,
  onViewDetails,
  compact = false,
}: RecommendationCardProps) {
  const [showDetails, setShowDetails] = useState(false);
  const [showDismissMenu, setShowDismissMenu] = useState(false);

  const priorityConfig = PRIORITY_CONFIG[recommendation.priority];
  const channelConfig = CHANNEL_CONFIG[recommendation.recommendedChannel];
  const tierConfig = CLIENT_TIER_CONFIG[recommendation.clientTier];
  const PriorityIcon = priorityConfig.icon;
  const ChannelIcon = channelConfig.icon;
  const TierIcon = tierConfig.icon;

  if (compact) {
    return (
      <div className={`bg-white rounded-lg border ${priorityConfig.borderColor} p-3 hover:shadow-md transition-shadow`}>
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <ConfidenceGauge confidence={recommendation.confidence} size="sm" />
            <div>
              <h3 className="font-medium text-slate-900 text-sm">{recommendation.actionName}</h3>
              <div className="flex items-center gap-2 text-xs text-slate-500">
                <span>{recommendation.clientName}</span>
                <span>•</span>
                <span className={channelConfig.color}>{channelConfig.label}</span>
              </div>
            </div>
          </div>
          <div className="flex items-center gap-2">
            <span className="text-sm font-bold text-green-600">
              ${recommendation.expectedValue.toLocaleString()}
            </span>
            <button
              onClick={onExecute}
              className="px-3 py-1 bg-indigo-600 text-white text-xs rounded-md hover:bg-indigo-700"
              title="Execute action"
            >
              Execute
            </button>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className={`bg-white rounded-xl border-2 ${priorityConfig.borderColor} overflow-hidden hover:shadow-lg transition-shadow`}>
      {/* Priority Header */}
      <div className={`px-4 py-2 ${priorityConfig.bgColor} flex items-center justify-between`}>
        <div className="flex items-center gap-2">
          <PriorityIcon className={`w-4 h-4 ${priorityConfig.color}`} />
          <span className={`text-sm font-medium ${priorityConfig.color}`}>
            {priorityConfig.label} Priority
          </span>
        </div>
        <TimeRemaining expiresAt={recommendation.expiresAt} />
      </div>

      {/* Main Content */}
      <div className="p-4">
        {/* Action & Client Info */}
        <div className="flex items-start justify-between mb-4">
          <div className="flex-1">
            <h3 className="font-bold text-slate-900 text-lg mb-1">{recommendation.actionName}</h3>
            <div className="flex items-center gap-3">
              <div className="flex items-center gap-1.5">
                <User className="w-4 h-4 text-slate-400" />
                <span className="text-sm text-slate-700">{recommendation.clientName}</span>
              </div>
              <div className={`flex items-center gap-1 px-2 py-0.5 rounded-full ${tierConfig.bgColor}`}>
                <TierIcon className={`w-3 h-3 ${tierConfig.color}`} />
                <span className={`text-xs font-medium ${tierConfig.color}`}>{tierConfig.label}</span>
              </div>
            </div>
          </div>
          <ConfidenceGauge confidence={recommendation.confidence} size="lg" />
        </div>

        {/* Key Metrics */}
        <div className="grid grid-cols-3 gap-3 mb-4">
          <div className="bg-green-50 rounded-lg p-3 text-center">
            <div className="flex items-center justify-center gap-1 text-green-600 mb-1">
              <DollarSign className="w-4 h-4" />
              <span className="text-xs font-medium">Expected Value</span>
            </div>
            <div className="text-xl font-bold text-green-700">
              ${recommendation.expectedValue.toLocaleString()}
            </div>
          </div>
          <div className="bg-blue-50 rounded-lg p-3 text-center">
            <div className="flex items-center justify-center gap-1 text-blue-600 mb-1">
              <Target className="w-4 h-4" />
              <span className="text-xs font-medium">Success Prob.</span>
            </div>
            <div className="text-xl font-bold text-blue-700">
              {Math.round(recommendation.successProbability * 100)}%
            </div>
          </div>
          <div className="bg-purple-50 rounded-lg p-3 text-center">
            <div className="flex items-center justify-center gap-1 text-purple-600 mb-1">
              <Clock className="w-4 h-4" />
              <span className="text-xs font-medium">Duration</span>
            </div>
            <div className="text-xl font-bold text-purple-700">
              {recommendation.estimatedDurationMinutes}m
            </div>
          </div>
        </div>

        {/* Channel & Trigger Signal */}
        <div className="flex items-center justify-between mb-4 pb-4 border-b border-slate-100">
          <div className="flex items-center gap-4">
            <div className="flex items-center gap-2">
              <ChannelIcon className={`w-4 h-4 ${channelConfig.color}`} />
              <span className="text-sm text-slate-600">{channelConfig.label}</span>
            </div>
            <div className="flex items-center gap-2 px-2 py-1 bg-indigo-50 rounded-md">
              <Zap className="w-3 h-3 text-indigo-600" />
              <span className="text-xs text-indigo-700 font-medium">{recommendation.triggerSignal}</span>
              <span className="text-xs text-indigo-500">
                ({Math.round(recommendation.triggerSignalStrength * 100)}%)
              </span>
            </div>
          </div>
        </div>

        {/* Expandable Details */}
        <button
          onClick={() => setShowDetails(!showDetails)}
          className="w-full flex items-center justify-between py-2 text-sm text-slate-600 hover:text-slate-900"
          title={showDetails ? 'Hide reasoning' : 'Show reasoning'}
        >
          <span>AI Reasoning</span>
          {showDetails ? (
            <ChevronUp className="w-4 h-4" />
          ) : (
            <ChevronDown className="w-4 h-4" />
          )}
        </button>

        {showDetails && (
          <div className="bg-slate-50 rounded-lg p-3 mb-4">
            <p className="text-sm text-slate-700">{recommendation.reasoning}</p>
          </div>
        )}

        {/* Action Buttons */}
        <div className="flex items-center gap-2 pt-2">
          <button
            onClick={onExecute}
            className="flex-1 flex items-center justify-center gap-2 px-4 py-2.5 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors font-medium"
            title="Execute this action"
          >
            <CheckCircle className="w-4 h-4" />
            Execute Now
          </button>

          {onSchedule && (
            <button
              onClick={() => {
                const scheduledTime = new Date(Date.now() + 24 * 60 * 60 * 1000);
                onSchedule(scheduledTime);
              }}
              className="p-2.5 border border-slate-300 rounded-lg hover:bg-slate-50 transition-colors"
              title="Schedule for later"
            >
              <Calendar className="w-4 h-4 text-slate-600" />
            </button>
          )}

          <div className="relative">
            <button
              onClick={() => setShowDismissMenu(!showDismissMenu)}
              className="p-2.5 border border-slate-300 rounded-lg hover:bg-slate-50 transition-colors"
              title="Dismiss action"
            >
              <ThumbsDown className="w-4 h-4 text-slate-600" />
            </button>
            {showDismissMenu && (
              <DismissMenu
                onDismiss={onDismiss}
                onClose={() => setShowDismissMenu(false)}
              />
            )}
          </div>

          {onViewDetails && (
            <button
              onClick={onViewDetails}
              className="p-2.5 border border-slate-300 rounded-lg hover:bg-slate-50 transition-colors"
              title="View full details"
            >
              <ExternalLink className="w-4 h-4 text-slate-600" />
            </button>
          )}
        </div>
      </div>

      {/* Urgency Indicator */}
      {(recommendation.priority === 'CRITICAL' || recommendation.priority === 'HIGH') && (
        <div className={`px-4 py-2 ${
          recommendation.priority === 'CRITICAL' ? 'bg-red-100' : 'bg-orange-100'
        } flex items-center gap-2`}>
          <AlertTriangle className={`w-4 h-4 ${
            recommendation.priority === 'CRITICAL' ? 'text-red-600' : 'text-orange-600'
          }`} />
          <span className={`text-xs font-medium ${
            recommendation.priority === 'CRITICAL' ? 'text-red-700' : 'text-orange-700'
          }`}>
            {recommendation.priority === 'CRITICAL'
              ? 'Requires immediate attention'
              : 'Time-sensitive action recommended'}
          </span>
        </div>
      )}
    </div>
  );
}

export default RecommendationCard;
