import type { Explain } from './types';
import ImpactPanel from './ImpactPanel';
import { LineageGraph } from './LineageGraph';

// A simple utility to format bytes
function formatBytes(bytes: number, decimals = 2) {
  if (!+bytes) return '0 Bytes';
  const k = 1024;
  const dm = decimals < 0 ? 0 : decimals;
  const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB', 'PB', 'EB', 'ZB', 'YB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(dm))} ${sizes[i]}`;
}

// Simple components for styling, can be replaced with a real component library
const Section = ({ title, children }: { title: string, children: React.ReactNode }) => (
  <div className="explain-section">
    <h4>{title}</h4>
    {children}
  </div>
);

const Badge = ({ children }: { children: React.ReactNode }) => <span className="badge">{children}</span>;
const Warning = ({ children }: { children: React.ReactNode }) => <div className="warning-badge">{children}</div>;

export default function ExplainTab({ explain }: { explain?: Explain | null }) {
  if (!explain) {
    return <div className="explain-tab-placeholder">No explanation available. Run a query to see details.</div>;
  }

  return (
    <div className="explain-tab">
      <Section title="Routing & Optimization">
        {explain.preagg_hit ? (
          <Badge>Pre-Agg Hit: {explain.preagg_name}</Badge>
        ) : (
          <Warning>Fallback to base tables</Warning>
        )}
        {explain.routing_reason && <p>{explain.routing_reason}</p>}
        {!explain.preagg_hit && explain.fallback_reason && <p><strong>Reason for fallback:</strong> {explain.fallback_reason}</p>}
        {explain.scan_size_estimate != null && <p>Estimated scan size: {formatBytes(explain.scan_size_estimate)}</p>}
        {explain.partitions_pruned && explain.partitions_pruned.length > 0 && <p>Partitions pruned: {explain.partitions_pruned.join(', ')}</p>}
        {explain.freshness && <p>Data freshness: {explain.freshness}</p>}
      </Section>

      {explain.optimization_suggestions && explain.optimization_suggestions.length > 0 && (
        <Section title="Optimization Tips">
          <ul>
            {explain.optimization_suggestions.map((s, i) => (
              <li key={i}>{s}</li>
            ))}
          </ul>
        </Section>
      )}

  {/* For demonstration, we'll show lineage for a specific metric */}
      <Section title="Lineage & Impact">
        <p>Showing lineage for metric: <strong>total_revenue</strong></p>
  {/* In a real app, you'd pass the relevant asset ID from the query */}
  <LineageGraph apis={[]} businessTerms={[]} onNodeClick={() => {}} />
  <ImpactPanel assetId="total_revenue" />
      </Section>
    </div>
  );
}