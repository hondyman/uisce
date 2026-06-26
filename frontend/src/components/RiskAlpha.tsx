
import type { FC } from 'react';
import { gql, useSubscription } from '@apollo/client';

const PORTFOLIOS_SUBSCRIPTION = gql`
  subscription { 
    portfolios {
      id
      aum
      risk_score
      status
    }
  }
`;

const RiskAlpha: FC = () => {
  const { data, loading, error } = useSubscription(PORTFOLIOS_SUBSCRIPTION);

  if (loading) return <p>Loading...</p>;
  if (error) return <p>Error: {error.message}</p>;

  return (
    <div>
      {data?.portfolios.map((p: any) => (
        <div key={p.id} className="risk-card">
          <h3>Portfolio {p.id} — ${p.aum.toLocaleString()}</h3>
          <p>Risk Score: <strong>{p.risk_score}</strong></p>
          <p>Status: <strong>{p.status}</strong></p>
          <button onClick={() => fetch(`/api/portfolio/${p.id}/risk`, {method: 'POST'})}>
            AI Risk Manage
          </button>
        </div>
      ))}
    </div>
  );
};

export default RiskAlpha;
