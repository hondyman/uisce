
import type { FC } from 'react';
import { gql, useSubscription } from '@apollo/client';

const PORTFOLIOS_SUBSCRIPTION = gql`
  subscription { 
    portfolios {
      id
      aum
      sharpe
      risk
      status
    }
  }
`;

const OptimizeAlpha: FC = () => {
  const { data, loading, error } = useSubscription(PORTFOLIOS_SUBSCRIPTION);

  if (loading) return <p>Loading...</p>;
  if (error) return <p>Error: {error.message}</p>;

  return (
    <div>
      {data?.portfolios.map((p: any) => (
        <div key={p.id} className="opt-card">
          <h3>Portfolio {p.id} — ${p.aum.toLocaleString()}</h3>
          <p>Sharpe: <strong>{p.sharpe}</strong></p>
          <p>Risk: <strong>{p.risk}%</strong></p>
          <p>Status: <strong>{p.status}</strong></p>
          <button onClick={() => fetch(`/api/portfolio/${p.id}/optimize`, {method: 'POST'})}>
            AI Alpha Optimize
          </button>
        </div>
      ))}
    </div>
  );
};

export default OptimizeAlpha;
