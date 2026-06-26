import React from 'react';
import { useSubscription, gql } from '@apollo/client';

const ATTRIBUTION_ALPHA_QUERY = gql`
  subscription AttributionAlphaSubscription {
    portfolios {
      id
      aum
      alpha
      sector
      status
    }
  }
`;

const AttributionAlpha: React.FC = () => {
  const { data, loading, error } = useSubscription(ATTRIBUTION_ALPHA_QUERY);

  if (loading) return <div>Loading Attribution Alpha Dashboard...</div>;
  if (error) return <div>Error: {error.message}</div>;

  return (
    <div className="attribution-alpha-dashboard">
      <h1>Attribution Alpha - AI-Powered Performance Analysis</h1>
      <div className="portfolio-grid">
        {data?.portfolios?.map((portfolio: any) => (
          <div key={portfolio.id} className="attr-card">
            <h3>Portfolio {portfolio.id}</h3>
            <p>AUM: <strong>${portfolio.aum?.toLocaleString()}</strong></p>
            <p>Alpha: <strong>{portfolio.alpha}%</strong></p>
            <p>Sector Contribution: <strong>{portfolio.sector}%</strong></p>
            <p>Status: <strong>{portfolio.status}</strong></p>
            <button
              onClick={() => fetch(`/api/portfolio/${portfolio.id}/attribute`, { method: 'POST' })}
              className="alpha-button"
            >
              🚀 AI Alpha Attribute
            </button>
          </div>
        ))}
      </div>
    </div>
  );
};

export default AttributionAlpha;