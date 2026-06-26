import React, { useState, useEffect } from 'react';
import { TrendingUp, AlertCircle, Calendar, DollarSign, Target, ChevronRight, Check, X } from 'lucide-react';

interface RecommendedAction {
  actionId: string;
  actionType: string;
  priority: number;
  estimatedImpactDollars: number | null;
  estimatedImpactDescription: string;
  timeSensitivity: string;
  actionDetails: any;
  rationale: string;
  status: string;
  presentedAt: string | null;
  createdAt: string;
}

const ACTION_TYPE_CONFIG: Record<string, { icon: any; label: string; color: string }> = {
  REBALANCE_PORTFOLIO: { icon: TrendingUp, label: 'Rebalance Portfolio', color: 'blue' },
  TAX_LOSS_HARVEST: { icon: DollarSign, label: 'Tax-Loss Harvesting', color: 'green' },
  INCREASE_CONTRIBUTION: { icon: Target, label: 'Increase Contributions', color: 'purple' },
  REDUCE_EXPENSES: { icon: AlertCircle, label: 'Reduce Expenses', color: 'orange' },
  SCHEDULE_REVIEW: { icon: Calendar, label: 'Schedule Review', color: 'indigo' },
  UPDATE_BENEFICIARIES: { icon: AlertCircle, label: 'Update Beneficiaries', color: 'red' },
  REVIEW_INSURANCE: { icon: AlertCircle, label: 'Review Insurance', color: 'yellow' },
  ROTH_CONVERSION: { icon: DollarSign, label: 'Roth Conversion', color: 'emerald' },
  ESTATE_PLANNING: { icon: AlertCircle, label: 'Estate Planning', color: 'violet' },
  CHARITABLE_GIVING: { icon: Target, label: 'Charitable Giving', color: 'pink' },
};

const URGENCY_CONFIG: Record<string, { label: string; color: string; bgColor: string }> = {
  IMMEDIATE: { label: 'Act Now', color: 'text-red-700', bgColor: 'bg-red-100' },
  THIS_WEEK: { label: 'This Week', color: 'text-orange-700', bgColor: 'bg-orange-100' },
  THIS_MONTH: { label: 'This Month', color: 'text-yellow-700', bgColor: 'bg-yellow-100' },
  THIS_QUARTER: { label: 'This Quarter', color: 'text-blue-700', bgColor: 'bg-blue-100' },
  ANYTIME: { label: 'No Rush', color: 'text-gray-700', bgColor: 'bg-gray-100' },
};

export const ActionTrackerWidget: React.FC = () => {
  const [actions, setActions] = useState<RecommendedAction[]>([]);
  const [expandedAction, setExpandedAction] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    fetchActions();
  }, []);

  const fetchActions = async () => {
    setIsLoading(true);
    try {
      const response = await fetch('/api/actions/recommended');
      const data = await response.json();
      setActions(data);
    } catch (error) {
      console.error('Failed to fetch actions:', error);
    } finally {
      setIsLoading(false);
    }
  };

  const handleAction = async (actionId: string, accept: boolean) => {
    try {
      await fetch(`/api/actions/${actionId}/${accept ? 'accept' : 'decline'}`, {
        method: 'POST',
      });
      fetchActions();
    } catch (error) {
      console.error('Failed to update action:', error);
    }
  };

  const formatCurrency = (value: number) => {
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: 'USD',
      minimumFractionDigits: 0,
      maximumFractionDigits: 0,
    }).format(value);
  };

  if (isLoading) {
    return (
      <div className="widget-card animate-pulse">
        <div className="h-6 bg-gray-200 rounded w-1/3 mb-4"></div>
        <div className="space-y-3">
          {[1, 2, 3].map(i => (
            <div key={i} className="h-20 bg-gray-100 rounded"></div>
          ))}
        </div>
      </div>
    );
  }

  const pendingActions = actions.filter(a => a.status === 'PENDING' || a.status === 'PRESENTED');
  const sortedActions = pendingActions.sort((a, b) => b.priority - a.priority);

  return (
    <div className="widget-card">
      <div className="flex items-center justify-between mb-4">
        <div>
          <h3 className="text-lg font-semibold text-gray-900">Recommended Actions</h3>
          <p className="text-sm text-gray-600">AI-powered insights to optimize your wealth</p>
        </div>
        <div className="bg-indigo-100 text-indigo-700 px-3 py-1 rounded-full text-sm font-semibold">
          {pendingActions.length} pending
        </div>
      </div>

      {sortedActions.length === 0 ? (
        <div className="text-center py-8 bg-gradient-to-br from-green-50 to-emerald-50 rounded-xl">
          <div className="w-16 h-16 bg-green-500 rounded-full flex items-center justify-center mx-auto mb-3">
            <Check className="w-8 h-8 text-white" />
          </div>
          <p className="text-gray-700 font-medium">You're all caught up!</p>
          <p className="text-sm text-gray-600 mt-1">No pending actions at this time</p>
        </div>
      ) : (
        <div className="space-y-3">
          {sortedActions.slice(0, 5).map((action) => {
            const config = ACTION_TYPE_CONFIG[action.actionType] || {
              icon: AlertCircle,
              label: action.actionType,
              color: 'gray',
            };
            const urgency = URGENCY_CONFIG[action.timeSensitivity] || URGENCY_CONFIG.ANYTIME;
            const Icon = config.icon;
            const isExpanded = expandedAction === action.actionId;

            return (
              <div
                key={action.actionId}
                className="border border-gray-200 rounded-xl hover:shadow-md transition-all overflow-hidden"
              >
                <button
                  onClick={() => setExpandedAction(isExpanded ? null : action.actionId)}
                  className="w-full p-4 text-left"
                >
                  <div className="flex items-start gap-3">
                    <div className={`p-2 rounded-lg bg-${config.color}-100`}>
                      <Icon className={`w-5 h-5 text-${config.color}-600`} />
                    </div>

                    <div className="flex-1 min-w-0">
                      <div className="flex items-start justify-between gap-2 mb-1">
                        <h4 className="font-semibold text-gray-900">{config.label}</h4>
                        <span className={`px-2 py-1 rounded-full text-xs font-medium ${urgency.bgColor} ${urgency.color}`}>
                          {urgency.label}
                        </span>
                      </div>

                      <p className="text-sm text-gray-600 mb-2">{action.estimatedImpactDescription}</p>

                      {action.estimatedImpactDollars && (
                        <div className="flex items-center gap-2 text-sm">
                          <span className="text-green-600 font-semibold">
                            {formatCurrency(action.estimatedImpactDollars)}
                          </span>
                          <span className="text-gray-500">potential benefit</span>
                        </div>
                      )}

                      <div className="flex items-center gap-2 mt-2">
                        <div className="flex-1 bg-gray-200 rounded-full h-1.5 overflow-hidden">
                          <div
                            className="bg-indigo-600 h-full"
                            style={{ width: `${action.priority * 10}%` }}
                          />
                        </div>
                        <span className="text-xs text-gray-500">Priority: {action.priority}/10</span>
                      </div>
                    </div>

                    <ChevronRight
                      className={`w-5 h-5 text-gray-400 transition-transform ${
                        isExpanded ? 'rotate-90' : ''
                      }`}
                    />
                  </div>
                </button>

                {isExpanded && (
                  <div className="px-4 pb-4 border-t border-gray-100 pt-4 bg-gray-50">
                    <div className="mb-4">
                      <h5 className="font-medium text-gray-900 mb-2">Why this matters:</h5>
                      <p className="text-sm text-gray-700">{action.rationale}</p>
                    </div>

                    {action.actionDetails && (
                      <div className="mb-4 bg-white p-3 rounded-lg border border-gray-200">
                        <h5 className="font-medium text-gray-900 mb-2 text-sm">Details:</h5>
                        <div className="text-xs text-gray-600 space-y-1">
                          {Object.entries(action.actionDetails).map(([key, value]) => (
                            <div key={key} className="flex justify-between">
                              <span className="font-medium">{key.replace(/_/g, ' ')}:</span>
                              <span>{String(value)}</span>
                            </div>
                          ))}
                        </div>
                      </div>
                    )}

                    <div className="flex gap-2">
                      <button
                        onClick={() => handleAction(action.actionId, true)}
                        className="flex-1 bg-gradient-to-r from-indigo-600 to-purple-600 text-white px-4 py-2 rounded-lg hover:from-indigo-700 hover:to-purple-700 transition-all font-medium text-sm"
                      >
                        Accept & Schedule
                      </button>
                      <button
                        onClick={() => handleAction(action.actionId, false)}
                        className="px-4 py-2 border border-gray-300 rounded-lg hover:bg-gray-100 transition-colors font-medium text-sm text-gray-700"
                      >
                        Decline
                      </button>
                    </div>
                  </div>
                )}
              </div>
            );
          })}

          {sortedActions.length > 5 && (
            <button className="w-full text-indigo-600 hover:text-indigo-700 text-sm font-medium py-2">
              View All {sortedActions.length} Actions →
            </button>
          )}
        </div>
      )}
    </div>
  );
};
