import React, { useState } from 'react';
import { useSubscription, gql } from '@apollo/client';
import { Shield, Zap } from 'lucide-react';

const PORTFOLIOS_RISK_SUB = gql`
  subscription PortfoliosRisk {
    portfolios(order_by: {aum: desc}) {
      id
      name
      aum
      risk_score
      rebalance_status
    }
  }
`;

const RiskAlphaDashboard = () => {
  const { data, loading } = useSubscription(PORTFOLIOS_RISK_SUB);
  const [triggered, setTriggered] = useState({});

  const handleManageRisk = async (portfolioId) => {
    setTriggered(prev => ({ ...prev, [portfolioId]: 'triggered' }));
    try {
      await fetch(`/api/portfolio/${portfolioId}/risk`, { method: 'POST' });
      // In a real app, you'd poll a status endpoint or get updates via subscription
      setTimeout(() => setTriggered(prev => ({ ...prev, [portfolioId]: 'complete' })), 5000);
    } catch (error) {
      console.error("Risk management trigger error:", error);
      setTriggered(prev => ({ ...prev, [portfolioId]: 'failed' }));
    }
  };

  if (loading) return <div>Loading...</div>;

  return (
    <div>
      <h1 className="text-4xl font-bold mb-2 bg-gradient-to-r from-red-500 to-orange-500 bg-clip-text text-transparent">
        Risk Alpha
      </h1>
      <p className="text-slate-400 mb-8">Detect and mitigate portfolio risk in seconds with AI-driven analysis.</p>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {data?.portfolios.map(p => {
          const riskScore = p.risk_score || 0;
          const riskColor = riskScore > 7.5 ? 'text-red-400' : riskScore > 5 ? 'text-orange-400' : 'text-green-400';

          return (
            <div key={p.id} className="bg-slate-800 rounded-xl border border-slate-700 p-6">
              <div className="flex justify-between items-start mb-4">
                <div>
                  <h3 className="font-bold text-lg">{p.name}</h3>
                  <p className="text-sm text-slate-400">{`$${(p.aum / 1e6).toFixed(1)}M AUM`}</p>
                </div>
                <div className={`flex items-center gap-2 font-bold text-2xl ${riskColor}`}>
                  <Shield />
                  <span>{riskScore.toFixed(1)}</span>
                </div>
              </div>
              <p className="text-sm text-slate-400 mb-4">Status: <span className="font-medium text-slate-300">{p.rebalance_status}</span></p>
              <button
                onClick={() => handleManageRisk(p.id)}
                disabled={triggered[p.id] === 'triggered'}
                className="w-full py-3 rounded-lg font-medium transition flex items-center justify-center gap-2 bg-gradient-to-r from-red-600 to-orange-600 hover:from-red-700 hover:to-orange-700 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                <Zap className="w-4 h-4" />
                AI Risk Manage
              </button>
            </div>
          );
        })}
      </div>
    </div>
  );
};

export default RiskAlphaDashboard;
