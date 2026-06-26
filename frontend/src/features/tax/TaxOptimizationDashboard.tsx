import React, { useState, useEffect } from 'react';
import { TrendingDown, AlertTriangle, DollarSign, Calendar, CheckCircle, X, TrendingUp } from 'lucide-react';

interface TaxOpportunity {
  opportunityId: string;
  opportunityType: string;
  clientName: string;
  estimatedSavings: number;
  complexity: number;
  taxYear: number;
  deadline: string | null;
  description: string;
  actionRequired: string;
  status: string;
  identifiedAt: string;
}

const OPPORTUNITY_TYPES = {
  TAX_LOSS_HARVEST: { label: 'Tax-Loss Harvesting', icon: TrendingDown, color: 'green' },
  ROTH_CONVERSION: { label: 'Roth Conversion', icon: TrendingUp, color: 'blue' },
  CHARITABLE_GIVING: { label: 'Charitable Giving', icon: DollarSign, color: 'purple' },
  ESTATE_PLANNING: { label: 'Estate Planning', icon: AlertTriangle, color: 'orange' },
  GAIN_DEFERRAL: { label: 'Gain Deferral', icon: Calendar, color: 'indigo' },
  RETIREMENT_CONTRIBUTION: { label: 'Retirement Contribution', icon: TrendingUp, color: 'emerald' },
  BUSINESS_DEDUCTION: { label: 'Business Deduction', icon: DollarSign, color: 'violet' },
  KIDDIE_TAX_PLANNING: { label: 'Kiddie Tax Planning', icon: AlertTriangle, color: 'pink' },
};

export const TaxOptimizationDashboard: React.FC = () => {
  const [opportunities, setOpportunities] = useState<TaxOpportunity[]>([]);
  const [filter, setFilter] = useState<'all' | 'pending' | 'approved' | 'completed'>('all');
  const [stats, setStats] = useState({
    totalSavings: 0,
    pendingOpportunities: 0,
    avgComplexity: 0,
    deadlinesSoon: 0,
  });

  useEffect(() => {
    fetchOpportunities();
  }, []);

  const fetchOpportunities = async () => {
    try {
      const response = await fetch('/api/tax/opportunities');
      const data = await response.json();
      setOpportunities(data);

      // Calculate stats
      const pending = data.filter((o: TaxOpportunity) => o.status === 'PENDING');
      const totalSavings = data.reduce((sum: number, o: TaxOpportunity) => sum + o.estimatedSavings, 0);
      const avgComplexity = data.reduce((sum: number, o: TaxOpportunity) => sum + o.complexity, 0) / (data.length || 1);
      
      const now = new Date();
      const deadlinesSoon = data.filter((o: TaxOpportunity) => {
        if (!o.deadline) return false;
        const deadline = new Date(o.deadline);
        const daysUntil = (deadline.getTime() - now.getTime()) / (1000 * 60 * 60 * 24);
        return daysUntil > 0 && daysUntil <= 30;
      }).length;

      setStats({
        totalSavings,
        pendingOpportunities: pending.length,
        avgComplexity,
        deadlinesSoon,
      });
    } catch (error) {
      console.error('Failed to fetch tax opportunities:', error);
    }
  };

  const updateOpportunityStatus = async (opportunityId: string, status: string) => {
    try {
      await fetch(`/api/tax/opportunities/${opportunityId}/status`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ status }),
      });
      fetchOpportunities();
    } catch (error) {
      console.error('Failed to update opportunity:', error);
    }
  };

  const filteredOpportunities = opportunities.filter(opp => {
    if (filter === 'all') return true;
    return opp.status.toLowerCase() === filter;
  });

  const formatCurrency = (value: number) => {
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: 'USD',
      minimumFractionDigits: 0,
      maximumFractionDigits: 0,
    }).format(value);
  };

  return (
    <div className="max-w-7xl mx-auto p-6">
      <div className="mb-8">
        <h1 className="text-3xl font-bold text-gray-900">Tax Optimization Engine</h1>
        <p className="text-gray-600 mt-1">AI-powered tax savings opportunities across your client base</p>
      </div>

      {/* Stats Dashboard */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-6 mb-8">
        <div className="bg-gradient-to-br from-green-500 to-emerald-600 text-white rounded-xl p-6">
          <div className="flex items-center justify-between mb-2">
            <DollarSign className="w-8 h-8 opacity-80" />
            <TrendingDown className="w-6 h-6" />
          </div>
          <p className="text-3xl font-bold">{formatCurrency(stats.totalSavings)}</p>
          <p className="text-green-100 text-sm mt-1">Total Tax Savings Identified</p>
        </div>

        <div className="bg-white rounded-xl border border-gray-200 p-6">
          <div className="flex items-center justify-between mb-2">
            <div className="p-2 bg-orange-100 rounded-lg">
              <AlertTriangle className="w-6 h-6 text-orange-600" />
            </div>
            <span className="text-sm font-medium text-orange-600">ACTION NEEDED</span>
          </div>
          <p className="text-3xl font-bold text-gray-900">{stats.pendingOpportunities}</p>
          <p className="text-sm text-gray-600 mt-1">Pending Opportunities</p>
        </div>

        <div className="bg-white rounded-xl border border-gray-200 p-6">
          <div className="flex items-center justify-between mb-2">
            <div className="p-2 bg-indigo-100 rounded-lg">
              <Calendar className="w-6 h-6 text-indigo-600" />
            </div>
          </div>
          <p className="text-3xl font-bold text-gray-900">{stats.deadlinesSoon}</p>
          <p className="text-sm text-gray-600 mt-1">Deadlines Within 30 Days</p>
        </div>

        <div className="bg-white rounded-xl border border-gray-200 p-6">
          <div className="flex items-center justify-between mb-2">
            <div className="p-2 bg-purple-100 rounded-lg">
              <TrendingUp className="w-6 h-6 text-purple-600" />
            </div>
          </div>
          <p className="text-3xl font-bold text-gray-900">{stats.avgComplexity.toFixed(1)}/10</p>
          <p className="text-sm text-gray-600 mt-1">Avg Complexity Score</p>
        </div>
      </div>

      {/* Filter Tabs */}
      <div className="flex gap-2 mb-6">
        {(['all', 'pending', 'approved', 'completed'] as const).map((f) => (
          <button
            key={f}
            onClick={() => setFilter(f)}
            className={`px-4 py-2 rounded-lg font-medium transition-colors ${
              filter === f
                ? 'bg-indigo-600 text-white'
                : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
            }`}
          >
            {f.charAt(0).toUpperCase() + f.slice(1)}
          </button>
        ))}
      </div>

      {/* Opportunities List */}
      <div className="space-y-4">
        {filteredOpportunities.length === 0 ? (
          <div className="text-center py-12 bg-gray-50 rounded-xl">
            <CheckCircle className="w-16 h-16 text-green-500 mx-auto mb-4" />
            <p className="text-gray-700 font-medium">No {filter !== 'all' ? filter : ''} opportunities found</p>
            <p className="text-sm text-gray-500 mt-1">The AI engine scans quarterly for new opportunities</p>
          </div>
        ) : (
          filteredOpportunities.map((opp) => {
            const config = OPPORTUNITY_TYPES[opp.opportunityType as keyof typeof OPPORTUNITY_TYPES] || {
              label: opp.opportunityType,
              icon: AlertTriangle,
              color: 'gray',
            };
            const Icon = config.icon;

            const urgency = opp.deadline ? getDaysUntil(opp.deadline) : null;

            return (
              <div
                key={opp.opportunityId}
                className="bg-white rounded-xl border border-gray-200 p-6 hover:shadow-md transition-shadow"
              >
                <div className="flex items-start gap-4">
                  <div className={`p-3 rounded-xl bg-${config.color}-100`}>
                    <Icon className={`w-6 h-6 text-${config.color}-600`} />
                  </div>

                  <div className="flex-1 min-w-0">
                    <div className="flex items-start justify-between mb-2">
                      <div>
                        <h3 className="text-lg font-semibold text-gray-900">{config.label}</h3>
                        <p className="text-sm text-gray-600">{opp.clientName}</p>
                      </div>
                      <div className="text-right">
                        <p className="text-2xl font-bold text-green-600">
                          {formatCurrency(opp.estimatedSavings)}
                        </p>
                        <p className="text-xs text-gray-500">potential savings</p>
                      </div>
                    </div>

                    <p className="text-sm text-gray-700 mb-3">{opp.description}</p>

                    <div className="flex items-center gap-4 text-sm mb-3">
                      <div className="flex items-center gap-2">
                        <span className="text-gray-600">Tax Year:</span>
                        <span className="font-medium">{opp.taxYear}</span>
                      </div>
                      
                      <div className="flex items-center gap-2">
                        <span className="text-gray-600">Complexity:</span>
                        <div className="flex gap-1">
                          {Array.from({ length: 10 }).map((_, i) => (
                            <div
                              key={i}
                              className={`w-2 h-4 rounded-sm ${
                                i < opp.complexity ? 'bg-indigo-600' : 'bg-gray-200'
                              }`}
                            />
                          ))}
                        </div>
                        <span className="font-medium">{opp.complexity}/10</span>
                      </div>

                      {urgency !== null && (
                        <div className={`flex items-center gap-1 ${
                          urgency <= 7 ? 'text-red-600' : urgency <= 30 ? 'text-orange-600' : 'text-gray-600'
                        }`}>
                          <Calendar className="w-4 h-4" />
                          <span className="font-medium">{urgency} days left</span>
                        </div>
                      )}
                    </div>

                    <div className="bg-blue-50 border border-blue-100 p-3 rounded-lg mb-4">
                      <p className="text-sm text-blue-900">
                        <strong>Action Required:</strong> {opp.actionRequired}
                      </p>
                    </div>

                    <div className="flex gap-2">
                      {opp.status === 'PENDING' && (
                        <>
                          <button
                            onClick={() => updateOpportunityStatus(opp.opportunityId, 'APPROVED')}
                            className="px-4 py-2 bg-green-600 text-white rounded-lg hover:bg-green-700 transition-colors text-sm font-medium"
                          >
                            <CheckCircle className="w-4 h-4 inline mr-2" />
                            Approve & Execute
                          </button>
                          <button
                            onClick={() => updateOpportunityStatus(opp.opportunityId, 'REJECTED')}
                            className="px-4 py-2 border border-gray-300 text-gray-700 rounded-lg hover:bg-gray-50 transition-colors text-sm font-medium"
                          >
                            <X className="w-4 h-4 inline mr-2" />
                            Decline
                          </button>
                        </>
                      )}
                      {opp.status === 'APPROVED' && (
                        <span className="px-4 py-2 bg-blue-100 text-blue-700 rounded-lg text-sm font-medium">
                          ✓ Approved - In Progress
                        </span>
                      )}
                      {opp.status === 'COMPLETED' && (
                        <span className="px-4 py-2 bg-green-100 text-green-700 rounded-lg text-sm font-medium">
                          ✓ Completed - Savings Realized
                        </span>
                      )}
                    </div>
                  </div>
                </div>
              </div>
            );
          })
        )}
      </div>
    </div>
  );
};

const getDaysUntil = (dateString: string): number => {
  const deadline = new Date(dateString);
  const now = new Date();
  const diffMs = deadline.getTime() - now.getTime();
  return Math.ceil(diffMs / (1000 * 60 * 60 * 24));
};
