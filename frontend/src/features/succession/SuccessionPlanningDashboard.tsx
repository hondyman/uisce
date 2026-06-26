import React, { useState, useEffect } from 'react';
import { Users, TrendingUp, Calendar, AlertCircle, DollarSign, ArrowRight } from 'lucide-react';

interface SuccessionPlan {
  planId: string;
  advisorName: string;
  successorName: string | null;
  targetTransitionDate: string | null;
  transitionStatus: string;
  readinessScore: number;
  practiceValue: number;
  clientsInTransition: number;
  totalClients: number;
}

interface PracticeMetrics {
  advisorId: string;
  advisorName: string;
  totalAUM: number;
  annualRevenue: number;
  clientCount: number;
  avgClientRelationship: number;
  practiceMultiple: number;
  estimatedValue: number;
}

export const SuccessionPlanningDashboard: React.FC = () => {
  const [plans, setPlans] = useState<SuccessionPlan[]>([]);
  const [metrics, setMetrics] = useState<PracticeMetrics[]>([]);
  const [activeTab, setActiveTab] = useState<'plans' | 'valuations'>('plans');

  useEffect(() => {
    fetchData();
  }, []);

  const fetchData = async () => {
    try {
      const [plansRes, metricsRes] = await Promise.all([
        fetch('/api/succession/plans'),
        fetch('/api/succession/metrics'),
      ]);

      setPlans(await plansRes.json());
      setMetrics(await metricsRes.json());
    } catch (error) {
      console.error('Failed to fetch succession data:', error);
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

  const formatAUM = (value: number) => {
    if (value >= 1000000000) return `$${(value / 1000000000).toFixed(1)}B`;
    if (value >= 1000000) return `$${(value / 1000000).toFixed(1)}M`;
    return formatCurrency(value);
  };

  const getReadinessColor = (score: number) => {
    if (score >= 80) return 'text-green-600 bg-green-100';
    if (score >= 60) return 'text-yellow-600 bg-yellow-100';
    return 'text-orange-600 bg-orange-100';
  };

  return (
    <div className="max-w-7xl mx-auto p-6">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-3xl font-bold text-gray-900">Succession Planning & Continuity</h1>
          <p className="text-gray-600 mt-1">Manage advisor transitions and practice valuations</p>
        </div>
        <button className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors">
          Create New Plan
        </button>
      </div>

      {/* Summary Stats */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-6 mb-8">
        <div className="bg-white rounded-xl border border-gray-200 p-6">
          <div className="flex items-center gap-3 mb-2">
            <div className="p-2 bg-indigo-100 rounded-lg">
              <Users className="w-6 h-6 text-indigo-600" />
            </div>
          </div>
          <p className="text-3xl font-bold text-gray-900">{plans.length}</p>
          <p className="text-sm text-gray-600 mt-1">Active Succession Plans</p>
        </div>

        <div className="bg-white rounded-xl border border-gray-200 p-6">
          <div className="flex items-center gap-3 mb-2">
            <div className="p-2 bg-green-100 rounded-lg">
              <DollarSign className="w-6 h-6 text-green-600" />
            </div>
          </div>
          <p className="text-3xl font-bold text-gray-900">
            {formatCurrency(metrics.reduce((sum, m) => sum + m.estimatedValue, 0))}
          </p>
          <p className="text-sm text-gray-600 mt-1">Total Practice Value</p>
        </div>

        <div className="bg-white rounded-xl border border-gray-200 p-6">
          <div className="flex items-center gap-3 mb-2">
            <div className="p-2 bg-orange-100 rounded-lg">
              <Calendar className="w-6 h-6 text-orange-600" />
            </div>
          </div>
          <p className="text-3xl font-bold text-gray-900">
            {plans.filter((p) => p.transitionStatus === 'IN_PROGRESS').length}
          </p>
          <p className="text-sm text-gray-600 mt-1">Transitions in Progress</p>
        </div>

        <div className="bg-white rounded-xl border border-gray-200 p-6">
          <div className="flex items-center gap-3 mb-2">
            <div className="p-2 bg-purple-100 rounded-lg">
              <TrendingUp className="w-6 h-6 text-purple-600" />
            </div>
          </div>
          <p className="text-3xl font-bold text-gray-900">
            {(metrics.reduce((sum, m) => sum + m.practiceMultiple, 0) / (metrics.length || 1)).toFixed(1)}x
          </p>
          <p className="text-sm text-gray-600 mt-1">Avg Practice Multiple</p>
        </div>
      </div>

      {/* Tabs */}
      <div className="flex gap-4 border-b border-gray-200 mb-6">
        <button
          onClick={() => setActiveTab('plans')}
          className={`px-4 py-3 border-b-2 transition-colors font-medium ${
            activeTab === 'plans'
              ? 'border-indigo-600 text-indigo-600'
              : 'border-transparent text-gray-600 hover:text-gray-900'
          }`}
        >
          Succession Plans
        </button>
        <button
          onClick={() => setActiveTab('valuations')}
          className={`px-4 py-3 border-b-2 transition-colors font-medium ${
            activeTab === 'valuations'
              ? 'border-indigo-600 text-indigo-600'
              : 'border-transparent text-gray-600 hover:text-gray-900'
          }`}
        >
          Practice Valuations
        </button>
      </div>

      {/* Plans Tab */}
      {activeTab === 'plans' && (
        <div className="space-y-4">
          {plans.map((plan) => (
            <div key={plan.planId} className="bg-white rounded-xl border border-gray-200 p-6">
              <div className="flex items-start justify-between mb-4">
                <div>
                  <h3 className="text-lg font-semibold text-gray-900">{plan.advisorName}</h3>
                  <p className="text-sm text-gray-600">
                    {plan.successorName ? `Transitioning to ${plan.successorName}` : 'No successor assigned'}
                  </p>
                </div>
                <span className={`px-3 py-1 rounded-full text-sm font-medium ${getReadinessColor(plan.readinessScore)}`}>
                  {plan.readinessScore}% Ready
                </span>
              </div>

              <div className="grid grid-cols-4 gap-6 mb-4">
                <div>
                  <p className="text-sm text-gray-600">Practice Value</p>
                  <p className="text-xl font-bold text-gray-900">{formatCurrency(plan.practiceValue)}</p>
                </div>
                <div>
                  <p className="text-sm text-gray-600">Client Transition</p>
                  <p className="text-xl font-bold text-gray-900">
                    {plan.clientsInTransition}/{plan.totalClients}
                  </p>
                  <div className="w-full bg-gray-200 rounded-full h-2 mt-1">
                    <div
                      className="bg-indigo-600 h-2 rounded-full"
                      style={{ width: `${(plan.clientsInTransition / plan.totalClients) * 100}%` }}
                    />
                  </div>
                </div>
                <div>
                  <p className="text-sm text-gray-600">Target Date</p>
                  <p className="text-xl font-bold text-gray-900">
                    {plan.targetTransitionDate
                      ? new Date(plan.targetTransitionDate).toLocaleDateString()
                      : 'TBD'}
                  </p>
                </div>
                <div>
                  <p className="text-sm text-gray-600">Status</p>
                  <p className="text-xl font-bold text-gray-900 capitalize">{plan.transitionStatus.replace('_', ' ')}</p>
                </div>
              </div>

              <button className="flex items-center gap-2 text-indigo-600 hover:text-indigo-700 font-medium text-sm">
                View Full Plan
                <ArrowRight className="w-4 h-4" />
              </button>
            </div>
          ))}
        </div>
      )}

      {/* Valuations Tab */}
      {activeTab === 'valuations' && (
        <div className="bg-white rounded-xl border border-gray-200">
          <table className="w-full">
            <thead className="bg-gray-50 border-b border-gray-200">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Advisor</th>
                <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase">AUM</th>
                <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase">Revenue</th>
                <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase">Clients</th>
                <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase">Avg Relationship</th>
                <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase">Multiple</th>
                <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase">Est. Value</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-200">
              {metrics.map((metric) => (
                <tr key={metric.advisorId} className="hover:bg-gray-50">
                  <td className="px-6 py-4 font-medium text-gray-900">{metric.advisorName}</td>
                  <td className="px-6 py-4 text-right">{formatAUM(metric.totalAUM)}</td>
                  <td className="px-6 py-4 text-right">{formatCurrency(metric.annualRevenue)}</td>
                  <td className="px-6 py-4 text-right">{metric.clientCount}</td>
                  <td className="px-6 py-4 text-right">{metric.avgClientRelationship.toFixed(1)} yrs</td>
                  <td className="px-6 py-4 text-right font-semibold">{metric.practiceMultiple.toFixed(1)}x</td>
                  <td className="px-6 py-4 text-right font-bold text-green-600">
                    {formatCurrency(metric.estimatedValue)}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
};
