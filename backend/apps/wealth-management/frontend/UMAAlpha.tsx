import React from 'react';
import { useSubscription, gql } from '@apollo/client';

const UMA_ALPHA_QUERY = gql`
  subscription UMAAlphaSubscription {
    uma_accounts {
      id
      aum
      tax_saved
      status
    }
  }
`;

const UMAAlpha: React.FC = () => {
  const { data, loading, error } = useSubscription(UMA_ALPHA_QUERY);

  if (loading) return <div>Loading UMA Alpha Dashboard...</div>;
  if (error) return <div>Error: {error.message}</div>;

  return (
    <div className="uma-alpha-dashboard">
      <h1>UMA Alpha - AI-Powered Rebalancing</h1>
      <div className="uma-grid">
        {data?.uma_accounts?.map((uma: any) => (
          <div key={uma.id} className="uma-card">
            <h3>UMA {uma.id}</h3>
            <p>AUM: <strong>${uma.aum?.toLocaleString()}</strong></p>
            <p>Tax Saved: <strong>${uma.tax_saved}</strong></p>
            <p>Status: <strong>{uma.status}</strong></p>
            <button
              onClick={() => fetch(`/api/uma/${uma.id}/alpha`, { method: 'POST' })}
              className="alpha-button"
            >
              🚀 AI Alpha Rebalance
            </button>
          </div>
        ))}
      </div>
    </div>
  );
};

export default UMAAlpha;