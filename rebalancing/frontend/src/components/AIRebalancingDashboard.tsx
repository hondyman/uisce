import React, { useState } from 'react';
import { useSubscription, useMutation, gql } from '@apollo/client';
import { Zap, TrendingUp, DollarSign, AlertTriangle, CheckCircle, Clock, Activity } from 'lucide-react';

// GraphQL Subscriptions
const PORTFOLIOS_SUB = gql`
  subscription Portfolios {
    portfolios(order_by: {aum: desc}) {
      id name aum drift last_rebalance tax_saved rebalance_status
      target_model constraints policy_document
      holdings_aggregate { aggregate { count } }
    }
  }
`;

const REBALANCE_PLANS_SUB = gql`
  subscription Plans($portfolio_id: uuid!) {
    rebalance_plans(
      where: {portfolio_id: {_eq: $portfolio_id}}
      order_by: {timestamp: desc}
      limit: 10
    ) {
      id timestamp current_drift expected_drift tax_savings confidence status rationale summary
      proposed_trades
    }
  }
`;

const TRIGGER_REBALANCE = gql`
  mutation Trigger($portfolio_id: String!) {
    triggerRebalance(portfolioId: $portfolio_id) {
      workflow_id
      status
    }
  }
`;

const AIRebalancingDashboard = () => {
  const [selected, setSelected] = useState(null);
  const { data, loading } = useSubscription(PORTFOLIOS_SUB);
  const { data: plans } = useSubscription(REBALANCE_PLANS_SUB, {
    variables: { portfolio_id: selected?.id },
    skip: !selected
  });

  const [triggerRebalance] = useMutation(TRIGGER_REBALANCE, {
    onCompleted: (data) => {
      console.log('Rebalance initiated:', data.triggerRebalance.workflow_id);
    }
  });

  const portfolios = data?.portfolios || [];

  const handleRebalance = async (id) => {
    try {
      await triggerRebalance({ variables: { portfolio_id: id } });
    } catch (err) {
      console.error('Rebalance failed:', err);
    }
  };

  if (loading) return <Loading />;

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-900 via-slate-800 to-slate-900 text-white p-8">
      {/* Header */}
      <div className="mb-8">
        <h1 className="text-4xl font-bold mb-2 bg-gradient-to-r from-blue-400 to-purple-500 bg-clip-text text-transparent">
          AI Portfolio Rebalancing Alpha
        </h1>
        <p className="text-slate-400">3-second rebalancing • AI-optimized • Tax-efficient • ABAC-secured</p>
      </div>

      {/* Metrics */}
      <div className="grid grid-cols-4 gap-4 mb-8">
        <MetricCard
          icon={<DollarSign className="w-6 h-6 text-blue-400" />}
          title="Total AUM"
          value={`$${(portfolios.reduce((s, p) => s + p.aum, 0) / 1e9).toFixed(2)}B`}
        />
        <MetricCard
          icon={<TrendingUp className="w-6 h-6 text-green-400" />}
          title="Avg Drift"
          value={`${(portfolios.reduce((s, p) => s + p.drift, 0) / portfolios.length || 0).toFixed(2)}%`}
        />
        <MetricCard
          icon={<Zap className="w-6 h-6 text-yellow-400" />}
          title="Tax Saved YTD"
          value={`$${(portfolios.reduce((s, p) => s + (p.tax_saved || 0), 0) / 1e6).toFixed(2)}M`}
        />
        <MetricCard
          icon={<CheckCircle className="w-6 h-6 text-purple-400" />}
          title="Portfolios"
          value={portfolios.length}
          subtitle={`${portfolios.filter(p => p.drift > 5).length} need rebalance`}
        />
      </div>

      {/* Portfolio Cards */}
      <div className="grid grid-cols-3 gap-6">
        {portfolios.map(p => (
          <PortfolioCard
            key={p.id}
            portfolio={p}
            onSelect={() => setSelected(p)}
            onRebalance={() => handleRebalance(p.id)}
            selected={selected?.id === p.id}
          />
        ))}
      </div>

      {/* Plan Detail Modal */}
      {selected && plans && (
        <PlanModal
          portfolio={selected}
          plans={plans.rebalance_plans}
          onClose={() => setSelected(null)}
        />
      )}
    </div>
  );
};

const MetricCard = ({ icon, title, value, subtitle }) => (
  <div className="bg-slate-800 rounded-xl border border-slate-700 p-6">
    <div className="flex items-center justify-between mb-4">
      {icon}
    </div>
    <div className="text-3xl font-bold mb-1">{value}</div>
    <div className="text-sm text-slate-400">{subtitle || title}</div>
  </div>
);

const PortfolioCard = ({ portfolio: p, onSelect, onRebalance, selected }) => {
  const needsRebalance = p.drift > 5;
  const driftColor = p.drift > 10 ? 'text-red-400' : p.drift > 5 ? 'text-yellow-400' : 'text-green-400';
  const [lastUpdated, setLastUpdated] = useState(new Date());

  React.useEffect(() => {
    setLastUpdated(new Date());
  }, [p]);

  return (
    <div
      onClick={onSelect}
      className={`bg-slate-800 rounded-xl border-2 p-6 cursor-pointer transition ${
        selected ? 'border-blue-500 shadow-lg shadow-blue-500/20' : 'border-slate-700 hover:border-slate-600'
      }`}
    >
      <div className="flex items-start justify-between mb-4">
        <div>
          <h3 className="font-bold text-lg">{p.name}</h3>
          <p className="text-sm text-slate-400">{p.holdings_aggregate.aggregate.count} holdings</p>
        </div>
        <div className="text-right">
          {p.rebalance_status === 'in_progress' ? (
            <div className="flex items-center gap-1 text-blue-400 text-xs">
              <Clock className="w-3 h-3 animate-spin" />
              Rebalancing...
            </div>
          ) : (
            <div className="text-xs text-slate-500">Live: {lastUpdated.toLocaleTimeString()}</div>
          )}
        </div>
      </div>

      <div className="space-y-3 mb-4">
        <div className="flex justify-between">
          <span className="text-sm text-slate-400">AUM</span>
          <span className="font-bold">{`$${(p.aum / 1e6).toFixed(1)}M`}</span>
        </div>
        <div className="flex justify-between">
          <span className="text-sm text-slate-400">Drift</span>
          <span className={`font-bold ${driftColor}`}>{p.drift.toFixed(2)}%</span>
        </div>
        <div className="flex justify-between">
          <span className="text-sm text-slate-400">Tax Saved</span>
          <span className="font-bold text-green-400">{`$${((p.tax_saved || 0) / 1000).toFixed(1)}K`}</span>
        </div>
      </div>

      {p.constraints && (p.constraints.esg_preference || p.constraints.risk_appetite) && (
        <div className="mb-4 border-t border-slate-700 pt-4">
            <h4 className="text-sm font-bold text-slate-300 mb-2">Personalization</h4>
            <div className="flex flex-wrap gap-2">
                {p.constraints.esg_preference && <span className="text-xs bg-blue-900/50 text-blue-300 px-2 py-1 rounded-full">ESG: {p.constraints.esg_preference}</span>}
                {p.constraints.risk_appetite && <span className="text-xs bg-purple-900/50 text-purple-300 px-2 py-1 rounded-full">Risk: {p.constraints.risk_appetite}</span>}
            </div>
        </div>
      )}

      <button
        onClick={(e) => { e.stopPropagation(); onRebalance(); }}
        disabled={!needsRebalance || p.rebalance_status === 'in_progress'}
        className={`w-full py-3 rounded-lg font-medium transition flex items-center justify-center gap-2 ${
          needsRebalance && p.rebalance_status !== 'in_progress'
            ? 'bg-gradient-to-r from-blue-600 to-purple-600 hover:from-blue-700 hover:to-purple-700'
            : 'bg-slate-700 text-slate-500 cursor-not-allowed'
        }`}
      >
        <Zap className="w-4 h-4" />
        AI Alpha Rebalance
      </button>

      {needsRebalance && (
        <div className="mt-3 flex items-center gap-2 text-xs text-yellow-400">
          <AlertTriangle className="w-3 h-3" />
          Drift exceeds 5% threshold
        </div>
      )}
    </div>
  );
};

const PlanModal = ({ portfolio, plans, onClose }) => (
  <div className="fixed inset-0 bg-black/80 flex items-center justify-center z-50 p-8">
    <div className="bg-slate-800 rounded-xl border border-slate-700 max-w-6xl w-full max-h-[90vh] overflow-y-auto">
      <div className="sticky top-0 bg-slate-800 border-b border-slate-700 p-6 flex justify-between">
        <div>
          <h2 className="text-2xl font-bold">{portfolio.name}</h2>
          <p className="text-slate-400">Rebalance History & Compliance</p>
        </div>
        <button onClick={onClose} className="text-2xl text-slate-400 hover:text-white">✕</button>
      </div>

      <div className="p-6 space-y-6">
        {portfolio.policy_document && (
            <div className="bg-slate-900/50 border border-slate-700 rounded-lg p-4">
                <h3 className="text-lg font-bold text-slate-300 mb-2">Compliance Policy</h3>
                <pre className="text-sm text-slate-400 whitespace-pre-wrap font-sans">{portfolio.policy_document}</pre>
            </div>
        )}
        <div>
            <h3 className="text-lg font-bold text-slate-300 mb-4">Recent Plans</h3>
            {plans.length === 0 ? (
              <div className="text-center py-12 text-slate-400">No plans yet</div>
            ) : (
              <div className="space-y-4">
                {plans.map(plan => <PlanCard key={plan.id} plan={plan} />)}
              </div>
            )}
        </div>
      </div>
    </div>
  </div>
);

const PlanCard = ({ plan }) => {
  const statusColor = {
    proposed: 'bg-blue-900/30 text-blue-400 border-blue-500/30',
    completed: 'bg-green-900/30 text-green-400 border-green-500/30',
    failed: 'bg-red-900/30 text-red-400 border-red-500/30'
  }[plan.status] || 'bg-slate-700 text-slate-400';

  const trades = JSON.parse(plan.proposed_trades || '[]');

  return (
    <div className="bg-slate-700 rounded-lg border border-slate-600 p-6">
      <div className="flex justify-between mb-4">
        <div className="flex items-center gap-3">
          <span className={`px-3 py-1 rounded-full text-xs font-medium border ${statusColor}`}>
            {plan.status.toUpperCase()}
          </span>
          <span className="text-sm text-slate-400">
            {new Date(plan.timestamp).toLocaleString()}
          </span>
        </div>
        <div className="text-sm">
          <span className="text-slate-400">Confidence: </span>
          <span className="font-bold text-green-400">{plan.confidence.toFixed(0)}%</span>
        </div>
      </div>

      <div className="grid grid-cols-3 gap-4 mb-4">
        <div className="bg-slate-800 rounded-lg p-4">
          <div className="text-xs text-slate-400 mb-1">Current Drift</div>
          <div className="text-2xl font-bold text-red-400">{plan.current_drift.toFixed(2)}%</div>
        </div>
        <div className="bg-slate-800 rounded-lg p-4">
          <div className="text-xs text-slate-400 mb-1">Expected Drift</div>
          <div className="text-2xl font-bold text-green-400">{plan.expected_drift.toFixed(2)}%</div>
        </div>
        <div className="bg-slate-800 rounded-lg p-4">
          <div className="text-xs text-slate-400 mb-1">Tax Savings</div>
          <div className="text-2xl font-bold text-green-400">{`$${(plan.tax_savings / 1000).toFixed(1)}K`}</div>
        </div>
      </div>

      {plan.rationale && (
        <div className="bg-slate-800 rounded-lg p-4 mb-4">
          <div className="text-xs text-slate-400 mb-2">AI Rationale</div>
          <div className="text-sm text-slate-300">{plan.rationale}</div>
        </div>
      )}

      {plan.summary && (
        <div className="bg-green-900/20 border border-green-500/30 rounded-lg p-4 mb-4">
          <div className="text-xs text-green-400 mb-2">AI Generated Summary</div>
          <div className="text-sm text-slate-300">{plan.summary}</div>
        </div>
      )}

      {trades.length > 0 && (
        <div className="bg-slate-800 rounded-lg p-4">
          <div className="text-xs text-slate-400 mb-3">Proposed Trades ({trades.length})</div>
          <div className="space-y-2 max-h-40 overflow-y-auto">
            {trades.slice(0, 5).map((t, i) => (
              <div key={i} className="flex justify-between text-sm">
                <span className={t.action === 'BUY' ? 'text-green-400' : 'text-red-400'}>
                  {t.action} {t.symbol}
                </span>
                <span className="text-slate-300">
                  {t.shares.toLocaleString()} @ ${t.estimated_price.toFixed(2)}
                </span>
              </div>
            ))}
            {trades.length > 5 && (
              <div className="text-xs text-slate-500 text-center pt-2">
                +{trades.length - 5} more trades
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
};

const Loading = () => (
  <div className="min-h-screen bg-slate-900 flex items-center justify-center">
    <div className="text-center">
      <Activity className="w-16 h-16 text-blue-500 animate-spin mx-auto mb-4" />
      <p className="text-slate-400">Loading portfolios...</p>
    </div>
  </div>
);

export default AIRebalancingDashboard;
