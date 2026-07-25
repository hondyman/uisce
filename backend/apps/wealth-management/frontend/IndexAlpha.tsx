import React, { useState } from 'react';
import { useSubscription, gql } from '@apollo/client';

const DIRECT_INDEXES_SUBSCRIPTION = gql`
  subscription {
    direct_indexes {
      id
      aum
      drift
      tax_saved
      status
      esg_score
    }
  }
`;

const IndexAlpha: React.FC = () => {
  const [selectedIndex, setSelectedIndex] = useState<string>('');
  const [results, setResults] = useState<any>(null);

  const { data, loading, error } = useSubscription(DIRECT_INDEXES_SUBSCRIPTION);

  const handleOptimize = async (indexID: string) => {
    try {
      const response = await fetch(`/api/index/${indexID}/alpha`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
      });
      if (!response.ok) {
        throw new Error('Index optimization failed');
      }
      const data = await response.json();
      setResults(data);
    } catch (error) {
      console.error('Optimization error:', error);
    }
  };

  if (loading) return <div>Loading direct indexes...</div>;
  if (error) return <div>Error: {error.message}</div>;

  return (
    <div className="index-alpha-container">
      <h1>🚀 AI Direct Indexing Optimization</h1>
      <p>Optimize $10B direct index in 3 seconds with AI, ABAC, and zero code</p>

      <div className="index-grid">
        {data?.direct_indexes?.map((index: any) => (
          <div key={index.id} className="index-card">
            <h3>Index {index.id}</h3>
            <div className="index-metrics">
              <p><strong>AUM:</strong> ${index.aum?.toLocaleString()}</p>
              <p><strong>Drift:</strong> <span className={index.drift > 5 ? 'high-drift' : 'low-drift'}>{index.drift}%</span></p>
              <p><strong>Tax Saved:</strong> ${index.tax_saved?.toLocaleString()}</p>
              <p><strong>ESG Score:</strong> {index.esg_score}/100</p>
              <p><strong>Status:</strong> <span className={`status-${index.status}`}>{index.status}</span></p>
            </div>
            <button
              onClick={() => handleOptimize(index.id)}
              disabled={index.status === 'optimizing'}
              className="optimize-btn"
            >
              {index.status === 'optimizing' ? 'Optimizing...' : 'AI Alpha Optimize'}
            </button>
          </div>
        ))}
      </div>

      {results && (
        <div className="optimization-results">
          <h3>Optimization Results</h3>
          <pre>{JSON.stringify(results, null, 2)}</pre>
        </div>
      )}
    </div>
  );
};

export default IndexAlpha;