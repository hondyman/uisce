import React, { useEffect, useState } from 'react';
import { LineChart, Line, BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer, PieChart, Pie, Cell } from 'recharts';
import { useQuery, useSubscription } from '@apollo/client';
import gql from 'graphql-tag';
import './RebalanceDashboard.css';
import { devDebug } from '../utils/devLogger';

// GraphQL Queries & Subscriptions
const GET_PORTFOLIO_HOLDINGS = gql`
  query GetPortfolioHoldings($portfolioId: uuid!, $tenantId: uuid!) {
    proposed_trades(
      where: { portfolio_id: { _eq: $portfolioId }, tenant_id: { _eq: $tenantId } }
      order_by: { created_at: desc }
      limit: 100
    ) {
      id
      symbol
      action
      shares
      price
      unrealized_gain
      is_tax_harvest
      status
      created_at
    }
  }
`;

const GET_ALLOCATION_MODEL = gql`
  query GetAllocationModel($modelId: uuid!, $tenantId: uuid!) {
    allocation_models_by_pk(id: $modelId, tenant_id: $tenantId) {
      id
      name
      model_type
      allocations
      created_at
    }
  }
`;

const GET_REBALANCE_SUMMARY = gql`
  query GetRebalanceSummary($portfolioId: uuid!, $tenantId: uuid!) {
    v_rebalance_summary(
      where: { portfolio_id: { _eq: $portfolioId }, tenant_id: { _eq: $tenantId } }
      order_by: { created_at: desc }
      limit: 1
    ) {
      workflow_id
      status
      drift_before
      drift_after
      tax_saved
      trade_count
      gross_trade_value
      triggered_by
      created_at
    }
  }
`;

const ON_PROPOSED_TRADES = gql`
  subscription OnProposedTrades($portfolioId: uuid!, $tenantId: uuid!) {
    proposed_trades(
      where: { portfolio_id: { _eq: $portfolioId }, tenant_id: { _eq: $tenantId } }
      order_by: { created_at: desc }
    ) {
      id
      symbol
      action
      shares
      price
      unrealized_gain
      is_tax_harvest
      status
      created_at
    }
  }
`;

const ON_EXECUTION_UPDATES = gql`
  subscription OnExecutionUpdates($portfolioId: uuid!, $tenantId: uuid!) {
    trade_execution_log(
      where: { proposed_trade: { portfolio_id: { _eq: $portfolioId }, tenant_id: { _eq: $tenantId } } }
      order_by: { executed_at: desc }
    ) {
      id
      custodian
      order_id
      symbol
      action
      shares
      price
      status
      executed_at
    }
  }
`;

const ON_REBALANCE_SUMMARY = gql`
  subscription OnRebalanceSummary($portfolioId: uuid!, $tenantId: uuid!) {
    v_rebalance_summary(
      where: { portfolio_id: { _eq: $portfolioId }, tenant_id: { _eq: $tenantId } }
      order_by: { created_at: desc }
      limit: 1
    ) {
      workflow_id
      status
      drift_before
      drift_after
      tax_saved
      trade_count
      gross_trade_value
    }
  }
`;

interface RebalanceDashboardProps {
  portfolioId: string;
  modelId: string;
  tenantId: string;
  onExecute?: (trades: any[], taxImpact: any) => void;
}

export const RebalanceDashboard: React.FC<RebalanceDashboardProps> = ({
  portfolioId,
  modelId,
  tenantId,
  onExecute,
}) => {
  const [dryRunMode, setDryRunMode] = useState(false);
  const [driftData, setDriftData] = useState<any[]>([]);
  const [allocationComparison, setAllocationComparison] = useState<any[]>([]);
  const [taxMetrics, setTaxMetrics] = useState({
    taxSaved: 0,
    estimatedTaxDebt: 0,
    netImpact: 0,
    harvested: 0,
  });
  const [selectedTrade, setSelectedTrade] = useState<any>(null);

  // Portfolio holdings query
  const { data: holdingsData } = useQuery(GET_PORTFOLIO_HOLDINGS, {
    variables: { portfolioId, tenantId },
    pollInterval: 5000,
  });

  // Allocation model query
  const { data: modelData } = useQuery(GET_ALLOCATION_MODEL, {
    variables: { modelId, tenantId },
  });

  // Rebalance summary query
  const { data: summaryData } = useQuery(GET_REBALANCE_SUMMARY, {
    variables: { portfolioId, tenantId },
  });

  // Real-time subscriptions
  const { data: tradesData } = useSubscription(ON_PROPOSED_TRADES, {
    variables: { portfolioId, tenantId },
  });

  const { data: executionData } = useSubscription(ON_EXECUTION_UPDATES, {
    variables: { portfolioId, tenantId },
  });

  const { data: summarySubscription } = useSubscription(ON_REBALANCE_SUMMARY, {
    variables: { portfolioId, tenantId },
  });

  // Process drift data
  useEffect(() => {
    if (summaryData?.v_rebalance_summary?.[0]) {
      const summary = summaryData.v_rebalance_summary[0];
      setDriftData([
        {
          name: 'Before',
          drift: (summary.drift_before * 100).toFixed(2),
          type: 'Drift %',
        },
        {
          name: 'After',
          drift: (summary.drift_after * 100).toFixed(2),
          type: 'Drift %',
        },
      ]);

      setTaxMetrics({
        taxSaved: summary.tax_saved || 0,
        estimatedTaxDebt: summary.tax_saved ? (summary.tax_saved * 2) : 0, // Simplified: 2x tax saved as debt
        netImpact: -(summary.tax_saved || 0),
        harvested: summary.trade_count || 0,
      });
    }
  }, [summaryData, summarySubscription]);

  // Process allocation comparison
  useEffect(() => {
    if (modelData?.allocation_models_by_pk) {
      const model = modelData.allocation_models_by_pk;
      const allocations = Array.isArray(model.allocations) ? model.allocations : [];
      
      setAllocationComparison(
        allocations.map((alloc: any) => ({
          name: alloc.asset_class || 'Other',
          target: alloc.target_percent * 100,
          current: (alloc.target_percent * 0.95) * 100, // Mock current (5% drift)
          min: alloc.min_percent * 100,
          max: alloc.max_percent * 100,
        }))
      );
    }
  }, [modelData]);

  const COLORS = ['#3b82f6', '#10b981', '#f59e0b', '#ef4444', '#8b5cf6', '#ec4899'];

  const trades = tradesData?.proposed_trades || holdingsData?.proposed_trades || [];
  const executions = executionData?.trade_execution_log || [];

  const completedTrades = trades.filter((t: any) => t.status === 'executed');
  const completionRate = trades.length > 0 ? (completedTrades.length / trades.length) * 100 : 0;

  const handleExecuteRebalance = async () => {
    if (onExecute) {
      onExecute(trades, taxMetrics);
    }
    // In real implementation, would call backend API to start workflow
    devDebug('Starting rebalance workflow for portfolio:', portfolioId, 'Dry run:', dryRunMode);
  };

  const handlePreviewTrade = (trade: any) => {
    setSelectedTrade(trade);
  };

  return (
    <div className="rebalance-dashboard">
      {/* Header */}
      <div className="dashboard-header">
        <h1>Portfolio Rebalancing Dashboard</h1>
        <div className="header-controls">
          <label className="dry-run-toggle">
            <input
              type="checkbox"
              checked={dryRunMode}
              onChange={(e) => setDryRunMode(e.target.checked)}
            />
            <span>Dry Run Mode</span>
          </label>
          <button 
            className={`btn-execute ${dryRunMode ? 'btn-preview' : 'btn-execute-live'}`}
            onClick={handleExecuteRebalance}
          >
            {dryRunMode ? '👁 Preview Rebalance' : '▶ Execute Rebalance'}
          </button>
        </div>
      </div>

      {/* Key Metrics Row */}
      <div className="metrics-grid">
        <div className="metric-card">
          <h3>Drift Reduction</h3>
          <div className="metric-value">
            {driftData[0]?.drift}% → {driftData[1]?.drift}%
          </div>
          <div className="metric-subtext">Portfolio drift before/after</div>
        </div>

        <div className="metric-card metric-positive">
          <h3>Tax Saved</h3>
          <div className="metric-value">${taxMetrics.taxSaved.toFixed(2)}</div>
          <div className="metric-subtext">Via loss harvesting</div>
        </div>

        <div className="metric-card">
          <h3>Est. Tax Debt</h3>
          <div className="metric-value">${taxMetrics.estimatedTaxDebt.toFixed(2)}</div>
          <div className="metric-subtext">Realized gains (~20% rate)</div>
        </div>

        <div className="metric-card metric-neutral">
          <h3>Net Tax Impact</h3>
          <div className="metric-value">${taxMetrics.netImpact.toFixed(2)}</div>
          <div className="metric-subtext">Savings minus debt</div>
        </div>

        <div className="metric-card">
          <h3>Trades Proposed</h3>
          <div className="metric-value">{trades.length}</div>
          <div className="metric-subtext">{completedTrades.length} completed</div>
        </div>

        <div className="metric-card">
          <h3>Completion Rate</h3>
          <div className="metric-value">{completionRate.toFixed(0)}%</div>
          <div className="metric-subtext">{completedTrades.length}/{trades.length} trades</div>
        </div>
      </div>

      {/* Charts Row */}
      <div className="charts-grid">
        {/* Drift Comparison Chart */}
        <div className="chart-container">
          <h3>Drift Before & After</h3>
          <ResponsiveContainer width="100%" height={250}>
            <BarChart data={driftData}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey="name" />
              <YAxis />
              <Tooltip 
                formatter={(value) => `${value}%`}
                contentStyle={{ backgroundColor: '#1f2937', border: 'none', borderRadius: '8px', color: '#fff' }}
              />
              <Bar dataKey="drift" fill="#3b82f6" />
            </BarChart>
          </ResponsiveContainer>
        </div>

        {/* Allocation Comparison Chart */}
        <div className="chart-container">
          <h3>Target vs Current Allocation</h3>
          <ResponsiveContainer width="100%" height={250}>
            <BarChart data={allocationComparison}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey="name" />
              <YAxis />
              <Tooltip 
                formatter={(value) => `${(value as number).toFixed(1)}%`}
                contentStyle={{ backgroundColor: '#1f2937', border: 'none', borderRadius: '8px', color: '#fff' }}
              />
              <Legend />
              <Bar dataKey="target" fill="#10b981" name="Target %" />
              <Bar dataKey="current" fill="#3b82f6" name="Current %" />
            </BarChart>
          </ResponsiveContainer>
        </div>
      </div>

      {/* Proposed Trades Table */}
      <div className="trades-section">
        <h3>Proposed Trades ({trades.length} total)</h3>
        <div className="trades-table-wrapper">
          <table className="trades-table">
            <thead>
              <tr>
                <th>Symbol</th>
                <th>Action</th>
                <th>Shares</th>
                <th>Price</th>
                <th>Unrealized Gain</th>
                <th>Tax Harvest?</th>
                <th>Status</th>
                <th>Details</th>
              </tr>
            </thead>
            <tbody>
              {trades.map((trade: any, idx: number) => (
                <tr key={idx} className={`trade-row status-${trade.status}`}>
                  <td className="symbol-col">
                    <strong>{trade.symbol}</strong>
                  </td>
                  <td>
                    <span className={`badge badge-${trade.action.toLowerCase()}`}>
                      {trade.action.toUpperCase()}
                    </span>
                  </td>
                  <td>{trade.shares.toLocaleString()}</td>
                  <td>${trade.price.toFixed(2)}</td>
                  <td className={trade.unrealized_gain >= 0 ? 'positive' : 'negative'}>
                    ${trade.unrealized_gain.toFixed(2)}
                  </td>
                  <td>
                    {trade.is_tax_harvest ? (
                      <span className="badge badge-harvest">Harvesting</span>
                    ) : (
                      <span className="badge-na">—</span>
                    )}
                  </td>
                  <td>
                    <span className={`status-badge status-${trade.status}`}>
                      {trade.status.charAt(0).toUpperCase() + trade.status.slice(1)}
                    </span>
                  </td>
                  <td>
                    <button 
                      className="btn-detail"
                      onClick={() => handlePreviewTrade(trade)}
                    >
                      View
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {/* Trade Detail Modal */}
      {selectedTrade && (
        <div className="modal-overlay" onClick={() => setSelectedTrade(null)}>
          <div className="modal-content" onClick={(e) => e.stopPropagation()}>
            <div className="modal-header">
              <h3>Trade Details: {selectedTrade.symbol}</h3>
              <button className="btn-close" onClick={() => setSelectedTrade(null)}>×</button>
            </div>
            <div className="modal-body">
              <div className="detail-row">
                <label>Symbol:</label>
                <span>{selectedTrade.symbol}</span>
              </div>
              <div className="detail-row">
                <label>Action:</label>
                <span className={`badge badge-${selectedTrade.action.toLowerCase()}`}>
                  {selectedTrade.action}
                </span>
              </div>
              <div className="detail-row">
                <label>Shares:</label>
                <span>{selectedTrade.shares.toLocaleString()}</span>
              </div>
              <div className="detail-row">
                <label>Price:</label>
                <span>${selectedTrade.price.toFixed(2)}</span>
              </div>
              <div className="detail-row">
                <label>Total Value:</label>
                <span>${(selectedTrade.shares * selectedTrade.price).toFixed(2)}</span>
              </div>
              <div className="detail-row">
                <label>Unrealized Gain/Loss:</label>
                <span className={selectedTrade.unrealized_gain >= 0 ? 'positive' : 'negative'}>
                  ${selectedTrade.unrealized_gain.toFixed(2)}
                </span>
              </div>
              <div className="detail-row">
                <label>Tax Harvest:</label>
                <span>{selectedTrade.is_tax_harvest ? '✓ Yes' : '✗ No'}</span>
              </div>
              <div className="detail-row">
                <label>Status:</label>
                <span className={`status-badge status-${selectedTrade.status}`}>
                  {selectedTrade.status}
                </span>
              </div>
              <div className="detail-row">
                <label>Created:</label>
                <span>{new Date(selectedTrade.created_at).toLocaleString()}</span>
              </div>
            </div>
            <div className="modal-footer">
              <button className="btn-secondary" onClick={() => setSelectedTrade(null)}>
                Close
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Execution Timeline */}
      {executions.length > 0 && (
        <div className="execution-timeline">
          <h3>Execution Timeline</h3>
          <div className="timeline">
            {executions.slice(0, 5).map((exec: any, idx: number) => (
              <div key={idx} className="timeline-item">
                <div className="timeline-dot"></div>
                <div className="timeline-content">
                  <div className="timeline-header">
                    <strong>{exec.symbol}</strong>
                    <span className={`status-badge status-${exec.status}`}>
                      {exec.status}
                    </span>
                  </div>
                  <div className="timeline-details">
                    <span>{exec.action} {exec.shares} @ ${exec.price}</span>
                    <span className="timeline-time">
                      {new Date(exec.executed_at).toLocaleTimeString()}
                    </span>
                  </div>
                  <div className="timeline-custodian">
                    {exec.custodian} • Order: {exec.order_id}
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
};

export default RebalanceDashboard;