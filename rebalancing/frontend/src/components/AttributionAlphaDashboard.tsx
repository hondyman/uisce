import React, { useState } from 'react';
import { useSubscription, gql } from '@apollo/client';
import { BarChart, Zap } from 'lucide-react';

const PORTFOLIOS_ATTR_SUB = gql`
  subscription PortfoliosAttribution {
    portfolios(order_by: {aum: desc}) {
      id
      name
      aum
      alpha
      sector_attribution
      rebalance_status
    }
  }
`;

const AttributionAlphaDashboard = () => {
  const { data, loading } = useSubscription(PORTFOLIOS_ATTR_SUB);
  const [triggered, setTriggered] = useState({});

  const handleAttribute = async (portfolioId) => {
    setTriggered(prev => ({ ...prev, [portfolioId]: 'triggered' }));
    try {
      await fetch(`/api/portfolio/${portfolioId}/attribute`, { method: 'POST' });
      setTimeout(() => setTriggered(prev => ({ ...prev, [portfolioId]: 'complete' })), 5000);
    } catch (error) {
      console.error("Attribution trigger error:", error);
      setTriggered(prev => ({ ...prev, [portfolioId]: 'failed' }));
    }
  };

  if (loading) return <div>Loading...</div>;

  return (
    <div>
      <h1 className="text-4xl font-bold mb-2 bg-gradient-to-r from-teal-400 to-sky-500 bg-clip-text text-transparent">
        Attribution Alpha
      </h1>
      <p className="text-slate-400 mb-8">Attribute portfolio performance to key drivers with AI-powered models.</p>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {data?.portfolios.map(p => {
          const alpha = p.alpha || 0;
          const alphaColor = alpha > 0 ? 'text-green-400' : 'text-red-400';

          return (
            <div key={p.id} className="bg-slate-800 rounded-xl border border-slate-700 p-6">
              <div className="flex justify-between items-start mb-4">
                <div>
                  <h3 className="font-bold text-lg">{p.name}</h3>
                  <p className="text-sm text-slate-400">{`$${(p.aum / 1e6).toFixed(1)}M AUM`}</p>
                </div>
                <div className={`flex items-center gap-2 font-bold text-2xl ${alphaColor}`}>
                  <BarChart />
                  <span>{`${alpha.toFixed(2)}%`}</span>
                </div>
              </div>
              <p className="text-sm text-slate-400 mb-4">Status: <span className="font-medium text-slate-300">{p.rebalance_status}</span></p>
              <button
                onClick={() => handleAttribute(p.id)}
                disabled={triggered[p.id] === 'triggered'}
                className="w-full py-3 rounded-lg font-medium transition flex items-center justify-center gap-2 bg-gradient-to-r from-teal-600 to-sky-600 hover:from-teal-700 hover:to-sky-700 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                <Zap className="w-4 h-4" />
                AI Alpha Attribute
              </button>
            </div>
          );
        })}
      </div>
    </div>
  );
};

export default AttributionAlphaDashboard;
