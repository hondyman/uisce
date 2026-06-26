import { useState } from 'react';
import { Bar, Radar } from 'react-chartjs-2';
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  BarElement,
  RadialLinearScale,
  PointElement,
  LineElement,
  Filler,
  Title,
  Tooltip,
  Legend,
} from 'chart.js';
import styles from './AdvisorDashboard.module.css';

// Register Chart.js components
ChartJS.register(
  CategoryScale,
  LinearScale,
  BarElement,
  RadialLinearScale,
  PointElement,
  LineElement,
  Filler,
  Title,
  Tooltip,
  Legend
);

// Types
export interface MonteCarloSummary {
  mean: number;
  median: number;
  pct05: number;
  pct95: number;
  confidence80_min: number;
  confidence80_max: number;
  runs: number;
}

export interface FactorVector {
  symbol: string;
  factors: number[];
}

export interface FactorSimilarity {
  target: FactorVector;
  replacements: FactorVector[];
}

export interface Order {
  side: string;
  symbol: string;
  qty: number;
  est_value_usd: number;
  reason: string;
  lots?: { lot_id: string; term: string; unrealized_pnl: number }[];
}

export interface Citation {
  id: string;
  source: string;
  snapshot_id: string;
  excerpt: string;
}

export interface AdvisorView {
  title: string;
  summary: string;
  tracking_error_before: number;
  tracking_error_after: number;
  tax_impact_usd: number;
  disclosures: string[];
  monte_carlo: MonteCarloSummary;
  factor_similarity?: FactorSimilarity;
}

export interface RebalanceProposal {
  proposal_id: string;
  portfolio_id: string;
  generated_at: string;
  advisor_view: AdvisorView;
  orders: Order[];
  citations: Citation[];
  actions: {
    approve: { label: string };
    reject: { label: string };
    clarify: { label: string };
  };
}

// Utility functions
const formatUSD = (value: number): string => {
  const absValue = Math.abs(value);
  const prefix = value < 0 ? '-$' : '$';
  return `${prefix}${absValue.toLocaleString('en-US', { minimumFractionDigits: 0, maximumFractionDigits: 0 })}`;
};

// Monte Carlo Histogram Component
export function MonteCarloHistogram({ summary }: { summary: MonteCarloSummary }) {
  const data = {
    labels: ['5th %', 'Median', '95th %'],
    datasets: [
      {
        label: 'Tax Impact Distribution',
        data: [summary.pct05, summary.median, summary.pct95],
        backgroundColor: ['#f87171', '#60a5fa', '#34d399'],
      },
    ],
  };

  const options = {
    responsive: true,
    plugins: {
      legend: { display: false },
      tooltip: {
        callbacks: {
          label: (ctx: { raw: number }) => formatUSD(ctx.raw),
        },
      },
    },
    scales: {
      y: {
        ticks: {
          callback: (value: number) => formatUSD(value),
        },
      },
    },
  };

  return (
    <div className={styles.chartContainer}>
      <h4>Tax Impact Distribution</h4>
      <Bar data={data} options={options as any} />
      <p className={styles.confidenceBand}>
        80% confidence interval: {formatUSD(summary.confidence80_min)} – {formatUSD(summary.confidence80_max)}
      </p>
    </div>
  );
}

// Factor Similarity Chart Component
export function FactorSimilarityChart({ target, replacements }: {
  target: FactorVector;
  replacements: FactorVector[];
}) {
  const labels = ['Size', 'Value', 'Momentum', 'Quality', 'Volatility'];
  
  const datasets = [
    {
      label: target.symbol,
      data: target.factors,
      borderColor: '#2563eb',
      backgroundColor: 'rgba(37, 99, 235, 0.2)',
    },
    ...replacements.map((r, i) => ({
      label: r.symbol,
      data: r.factors,
      borderColor: ['#f97316', '#10b981', '#9333ea'][i % 3],
      backgroundColor: ['rgba(249, 115, 22, 0.2)', 'rgba(16, 185, 129, 0.2)', 'rgba(147, 51, 234, 0.2)'][i % 3],
    })),
  ];

  const data = { labels, datasets };
  
  const options = {
    responsive: true,
    scales: {
      r: {
        min: 0,
        max: 1,
      },
    },
  };

  return (
    <div className={styles.chartContainer}>
      <h4>Factor Similarity</h4>
      <Radar data={data} options={options} />
    </div>
  );
}

// Metric Component
function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div className={styles.metric}>
      <span className={styles.metricLabel}>{label}</span>
      <span className={styles.metricValue}>{value}</span>
    </div>
  );
}

// Proposal Card Component
export function ProposalCard({
  data,
  onApprove,
  onReject,
  onClarify,
}: {
  data: RebalanceProposal;
  onApprove: () => void;
  onReject: () => void;
  onClarify: () => void;
}) {
  const { advisor_view, orders, citations, actions } = data;

  return (
    <div className={styles.card}>
      <header className={styles.cardHeader}>
        <h3>{advisor_view.title}</h3>
        <p>{advisor_view.summary}</p>
      </header>

      <section className={styles.metrics}>
        <Metric
          label="Tracking error before"
          value={`${advisor_view.tracking_error_before.toFixed(2)}%`}
        />
        <Metric
          label="Tracking error after"
          value={`${advisor_view.tracking_error_after.toFixed(2)}%`}
        />
        <Metric
          label="Median tax impact"
          value={formatUSD(advisor_view.monte_carlo.median)}
        />
        <Metric
          label="80% confidence band"
          value={`${formatUSD(advisor_view.monte_carlo.confidence80_min)} – ${formatUSD(advisor_view.monte_carlo.confidence80_max)}`}
        />
      </section>

      <section className={styles.disclosures}>
        <h4>Disclosures</h4>
        <ul>
          {advisor_view.disclosures.map((d, i) => (
            <li key={i}>{d}</li>
          ))}
        </ul>
      </section>

      <section className={styles.orders}>
        <h4>Orders</h4>
        <table className={styles.ordersTable}>
          <thead>
            <tr>
              <th>Side</th>
              <th>Symbol</th>
              <th>Qty</th>
              <th>Est. Value</th>
              <th>Reason</th>
            </tr>
          </thead>
          <tbody>
            {orders.map((o, i) => (
              <tr key={i} className={o.side === 'SELL' ? styles.sellRow : styles.buyRow}>
                <td className={styles[o.side.toLowerCase()]}>{o.side}</td>
                <td>{o.symbol}</td>
                <td>{o.qty}</td>
                <td>{formatUSD(o.est_value_usd)}</td>
                <td>{o.reason.replace(/_/g, ' ')}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>

      <section className={styles.citations}>
        <h4>Citations</h4>
        <ul>
          {citations.map((c) => (
            <li key={c.id}>
              <strong>{c.source}</strong> • {c.excerpt}
            </li>
          ))}
        </ul>
      </section>

      <footer className={styles.cardFooter}>
        <button className={styles.approveButton} onClick={onApprove}>
          {actions.approve.label}
        </button>
        <button className={styles.rejectButton} onClick={onReject}>
          {actions.reject.label}
        </button>
        <button className={styles.clarifyButton} onClick={onClarify}>
          {actions.clarify.label}
        </button>
      </footer>
    </div>
  );
}

// Proposal List Component
function ProposalList({
  proposals,
  selected,
  onSelect,
}: {
  proposals: RebalanceProposal[];
  selected: RebalanceProposal | null;
  onSelect: (p: RebalanceProposal) => void;
}) {
  return (
    <div className={styles.proposalList}>
      <h2>Proposals</h2>
      <ul>
        {proposals.map((p) => (
          <li
            key={p.proposal_id}
            className={`${styles.proposalItem} ${selected?.proposal_id === p.proposal_id ? styles.selected : ''}`}
            onClick={() => onSelect(p)}
          >
            <strong>{p.advisor_view.title}</strong>
            <div className={styles.proposalMeta}>
              <span>
                TE Δ {p.advisor_view.tracking_error_before.toFixed(2)} → {p.advisor_view.tracking_error_after.toFixed(2)}
              </span>
              <span>Tax impact: {formatUSD(p.advisor_view.monte_carlo.median)}</span>
            </div>
          </li>
        ))}
      </ul>
    </div>
  );
}

// Proposal Detail Component
function ProposalDetail({
  proposal,
  onApprove,
  onReject,
  onClarify,
}: {
  proposal: RebalanceProposal;
  onApprove: () => void;
  onReject: () => void;
  onClarify: () => void;
}) {
  return (
    <div className={styles.proposalDetail}>
      <ProposalCard
        data={proposal}
        onApprove={onApprove}
        onReject={onReject}
        onClarify={onClarify}
      />
      
      <div className={styles.visualizations}>
        <MonteCarloHistogram summary={proposal.advisor_view.monte_carlo} />
        
        {proposal.advisor_view.factor_similarity && (
          <FactorSimilarityChart
            target={proposal.advisor_view.factor_similarity.target}
            replacements={proposal.advisor_view.factor_similarity.replacements}
          />
        )}
      </div>
    </div>
  );
}

// Server action for advisor decisions
export async function sendAdvisorDecision(
  proposal: RebalanceProposal,
  action: 'approve' | 'reject' | 'clarify',
  rationale?: string
): Promise<void> {
  const response = await fetch(`/api/workflow/${proposal.proposal_id}/signal`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      signal: 'AdvisorApproval',
      payload: {
        approved: action === 'approve',
        advisor_id: 'current_user', // In production, get from auth context
        rationale,
        action,
        time: new Date().toISOString(),
      },
    }),
  });

  if (!response.ok) {
    throw new Error('Failed to send decision');
  }
}

// Main Dashboard Component
export function AdvisorDashboard({ proposals: initialProposals }: { proposals?: RebalanceProposal[] }) {
  const [proposals] = useState<RebalanceProposal[]>(initialProposals || SAMPLE_PROPOSALS);
  const [selected, setSelected] = useState<RebalanceProposal | null>(null);
  const [loading, setLoading] = useState(false);

  const handleApprove = async () => {
    if (!selected) return;
    setLoading(true);
    try {
      await sendAdvisorDecision(selected, 'approve');
      alert('Proposal approved! Trades will be executed.');
    } catch (e) {
      alert('Failed to approve proposal');
    }
    setLoading(false);
  };

  const handleReject = async () => {
    if (!selected) return;
    setLoading(true);
    try {
      await sendAdvisorDecision(selected, 'reject', 'Manual rejection by advisor');
      alert('Proposal rejected.');
    } catch (e) {
      alert('Failed to reject proposal');
    }
    setLoading(false);
  };

  const handleClarify = async () => {
    if (!selected) return;
    const rationale = prompt('Enter clarification request:');
    if (!rationale) return;
    setLoading(true);
    try {
      await sendAdvisorDecision(selected, 'clarify', rationale);
      alert('Clarification requested.');
    } catch (e) {
      alert('Failed to request clarification');
    }
    setLoading(false);
  };

  return (
    <div className={styles.dashboard}>
      <ProposalList proposals={proposals} selected={selected} onSelect={setSelected} />
      
      {selected ? (
        <ProposalDetail
          proposal={selected}
          onApprove={handleApprove}
          onReject={handleReject}
          onClarify={handleClarify}
        />
      ) : (
        <div className={styles.placeholder}>
          <p>Select a proposal to review</p>
        </div>
      )}
      
      {loading && <div className={styles.loadingOverlay}>Processing...</div>}
    </div>
  );
}

// Sample data for development
const SAMPLE_PROPOSALS: RebalanceProposal[] = [
  {
    proposal_id: 'prop_20251123_001',
    portfolio_id: 'pf_892',
    generated_at: '2025-11-23T21:40:00Z',
    advisor_view: {
      title: 'Reduce IVV overweight; harvest BND losses; buy VOO/SPY replacements',
      summary: 'Sell 50 IVV to lower US Equity drift; sell 100 BND (loss) to offset gains; buy VOO and SPY to preserve exposure.',
      tracking_error_before: 2.0,
      tracking_error_after: 1.3,
      tax_impact_usd: -1800,
      disclosures: ['Wash-sale rules enforced', 'Factor exposures preserved'],
      monte_carlo: {
        mean: -1750,
        median: -1800,
        pct05: -1200,
        pct95: -2200,
        confidence80_min: -1500,
        confidence80_max: -2000,
        runs: 1000,
      },
      factor_similarity: {
        target: { symbol: 'IVV', factors: [0.5, 0.3, 0.2, 0.6, 0.4] },
        replacements: [
          { symbol: 'VOO', factors: [0.5, 0.3, 0.2, 0.6, 0.4] },
          { symbol: 'SPY', factors: [0.5, 0.3, 0.2, 0.6, 0.4] },
        ],
      },
    },
    orders: [
      { side: 'SELL', symbol: 'IVV', qty: 50, est_value_usd: 22500, reason: 'reduce_overweight' },
      { side: 'SELL', symbol: 'BND', qty: 100, est_value_usd: 7000, reason: 'harvest_loss', lots: [{ lot_id: 'bnd_l1', term: 'long', unrealized_pnl: -1200 }] },
      { side: 'BUY', symbol: 'VOO', qty: 20, est_value_usd: 8000, reason: 'factor_aware_replacement' },
      { side: 'BUY', symbol: 'SPY', qty: 15, est_value_usd: 7500, reason: 'factor_aware_replacement' },
    ],
    citations: [
      { id: 'C1', source: 'positions_snapshot', snapshot_id: 'snap_pos_20251123', excerpt: 'IVV overweight 35% vs target 30%' },
      { id: 'C2', source: 'factor_metadata', snapshot_id: 'factor_snap_20251123', excerpt: 'VOO/SPY factor vectors closely match IVV' },
    ],
    actions: {
      approve: { label: 'Approve and execute' },
      reject: { label: 'Reject' },
      clarify: { label: 'Request clarification' },
    },
  },
];

export default AdvisorDashboard;
