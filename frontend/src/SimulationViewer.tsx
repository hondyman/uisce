import { useState, useEffect, useCallback } from 'react';
import { listClaimSimulations } from '../../api';
import { ClaimSimulationResult } from './types';

function SimulationResultRow({ result }: { result: ClaimSimulationResult }) {
  const [expanded, setExpanded] = useState(false);

  const injectedRow = `
    .simulation-row{ cursor: pointer }
    .risk-text{ color: #b91c1c; font-weight: bold }
    .no-risk-text{ color: #166534 }
    .simulation-expanded{ padding: 1rem; background-color: #f3f4f6 }
    .risk-flag-item{ color: #b91c1c }
  `;

  return (
    <>
      <style dangerouslySetInnerHTML={{ __html: injectedRow }} />
      <tr onClick={() => setExpanded(!expanded)} className="simulation-row">
        <td>{result.simulated_at ? new Date(result.simulated_at).toLocaleString() : '—'}</td>
        <td>{result.simulated_by ?? 'system'}</td>
        <td>{result.simulated_for ?? '—'}</td>
        <td>
          {(result.risk_flags && result.risk_flags.length > 0) ? (
            <span className="risk-text">⚠️ {result.risk_flags.length} risk(s)</span>
          ) : (
            <span className="no-risk-text">✅ No risks</span>
          )}
        </td>
      </tr>
      {expanded && (
        <tr>
          <td colSpan={4} className="simulation-expanded">
            <div className="simulation-details">
              <h4>Simulation Details (ID: <code>{result.id ?? ''}</code>)</h4>
              
              <h5>Proposed Claims</h5>
              <pre><code>{JSON.stringify(typeof result.proposed_claims === 'string' ? JSON.parse(result.proposed_claims) : (result.proposed_claims ?? []), null, 2)}</code></pre>
              
              <h5>Risk Flags</h5>
              {(result.risk_flags && result.risk_flags.length > 0) ? (
                <ul>
                  {result.risk_flags.map((flag: string, i: number) => <li key={i} className="risk-flag-item">{flag}</li>)}
                </ul>
              ) : <p>None</p>}

              <h5>Affected Models</h5>
              {((result.affected_models || []) as Array<{ model_name?: string; model_id?: string; change?: string; certified?: boolean }>).length > 0 ? (
                <ul>
                  {((result.affected_models || []) as Array<{ model_name?: string; model_id?: string; change?: string; certified?: boolean }>).map((model, i) => (
                    <li key={i}>
                      <strong>{model.model_name}</strong> (<code>{model.model_id}</code>) - Change: {model.change} {model.certified && '(Certified)'}
                    </li>
                  ))}
                </ul>
              ) : <p>None</p>}
            </div>
          </td>
        </tr>
      )}
    </>
  );
}

export default function SimulationViewer() {
  const [simulations, setSimulations] = useState<ClaimSimulationResult[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchSimulations = useCallback(async () => {
    try {
      setLoading(true);
      const data = await listClaimSimulations();
      setSimulations(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch simulations');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchSimulations();
  }, [fetchSimulations]);

  if (loading) return <div>Loading simulation results...</div>;
  if (error) return <div className="error-text">Error: {error}</div>;

  return (
    <div>
      <h2>Claim Simulation History</h2>
      <p>Review past claim simulations and their potential impact.</p>
      <table className="governance-table">
        <thead>
          <tr>
            <th>Simulated At</th>
            <th>Simulated By</th>
            <th>For User/Role</th>
            <th>Risk Level</th>
          </tr>
        </thead>
        <tbody>
          {simulations.map(sim => (
            <SimulationResultRow key={sim.id} result={sim} />
          ))}
        </tbody>
      </table>
    </div>
  );
}