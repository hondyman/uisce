import React, { useState } from 'react';
import { useMutation } from '@tanstack/react-query';

const TAX_HARVEST_MUTATION = `
  mutation TaxHarvest($umaID: String!) {
    taxHarvest(umaID: $umaID) {
      status
      workflowId
      estimatedSavings
    }
  }
`;

const TaxHarvest: React.FC = () => {
  const [umaID, setUmaID] = useState('uma-123');
  const [results, setResults] = useState<any>(null);

  const mutation = useMutation({
    mutationFn: async (umaID: string) => {
      const response = await fetch('/api/uma/' + umaID + '/tax', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
      });
      if (!response.ok) {
        throw new Error('Tax harvest failed');
      }
      return response.json();
    },
    onSuccess: (data) => {
      setResults(data);
    },
  });

  return (
    <div className="tax-harvest-container">
      <h1>🚀 AI Tax Harvest Optimization</h1>
      <p>Save $1M+ in taxes per $1B AUM with AI-powered optimization</p>

      <div className="tax-harvest-form">
        <label>
          UMA ID:
          <input
            type="text"
            value={umaID}
            onChange={(e) => setUmaID(e.target.value)}
            placeholder="Enter UMA ID"
          />
        </label>

        <button
          onClick={() => mutation.mutate(umaID)}
          disabled={mutation.isPending}
          className="tax-harvest-button"
        >
          {mutation.isPending ? 'Optimizing...' : '🚀 AI Tax Harvest'}
        </button>
      </div>

      {mutation.isError && (
        <div className="error-message">
          Error: {mutation.error.message}
        </div>
      )}

      {results && (
        <div className="tax-results">
          <h3>Tax Optimization Results</h3>
          <p><strong>Status:</strong> {results.status}</p>
          <p><strong>Workflow ID:</strong> {results.workflow_id}</p>
          <p><strong>Estimated Savings:</strong> ${results.estimated_savings?.toLocaleString()}</p>
        </div>
      )}

      <div className="tax-strategies">
        <h3>AI Tax Optimization Strategies</h3>
        <div className="strategy-grid">
          <div className="strategy-card">
            <h4>Lot-Level Harvesting</h4>
            <p>xAI selects optimal lots based on basis, gains, and ESG scores</p>
          </div>
          <div className="strategy-card">
            <h4>Wash Sale Avoidance</h4>
            <p>xAI predicts 30-day conflicts and prevents wash sales</p>
          </div>
          <div className="strategy-card">
            <h4>ESG + Tax Alignment</h4>
            <p>xAI balances tax efficiency with ESG impact scoring</p>
          </div>
          <div className="strategy-card">
            <h4>Household Optimization</h4>
            <p>xAI aggregates UMAs for comprehensive tax planning</p>
          </div>
        </div>
      </div>
    </div>
  );
};

export default TaxHarvest;