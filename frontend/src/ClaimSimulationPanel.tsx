import { useState } from 'react';
import { devError } from './utils/devLogger';
import { useNotification } from './hooks/useNotification';
import { simulateClaims } from './api';
import type { ClaimSimulationRequest, ClaimSimulationResult, ProposedClaim } from './types';

interface ClaimSimulationPanelProps {
  // In a real app, you'd have a way to look up models
  availableModels: { id: string; name: string }[];
}

export default function ClaimSimulationPanel({ availableModels }: ClaimSimulationPanelProps) {
  const [simulateFor, setSimulateFor] = useState('');
  const [isRole, setIsRole] = useState(true);
  const [proposedClaims, setProposedClaims] = useState<ProposedClaim[]>([]);
  const [result, setResult] = useState<ClaimSimulationResult | null>(null);
  const [loading, setLoading] = useState(false);
  const notification = useNotification();

  const handleAddClaim = () => {
    setProposedClaims([...proposedClaims, { model_id: availableModels[0].id, permission: 'read' }]);
  };

  const handleClaimChange = (index: number, field: keyof ProposedClaim, value: string) => {
    const newClaims = [...proposedClaims];
    // keep the ProposedClaim shape but allow updating fields via index/key
    newClaims[index] = { ...newClaims[index], [field]: value } as ProposedClaim;
    setProposedClaims(newClaims);
  };

  const handleRunSimulation = async () => {
    if (!simulateFor.trim()) {
      notification.error('Please enter a user or role to simulate for.');
      return;
    }
    setLoading(true);
    setResult(null);
    try {
      const req: ClaimSimulationRequest = {
        simulate_for: simulateFor,
        is_role: isRole,
        proposed_claims: proposedClaims,
      };
      const simResult = await simulateClaims(req);
      setResult(simResult);
    } catch (error) {
      devError('Simulation failed:', error);
      notification.error('Simulation failed. See console for details.');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="claim-simulation-panel">
      <h4>Claim Simulation & Impact Preview</h4>
      <div className="simulation-form">
        <div className="form-group">
          <label>Simulate for:</label>
          <input type="text" value={simulateFor} onChange={e => setSimulateFor(e.target.value)} placeholder="User ID or Role Name" />
          <label><input type="checkbox" checked={isRole} onChange={e => setIsRole(e.target.checked)} /> Is Role</label>
        </div>
        <div className="form-group">
          <label>Proposed Claims:</label>
          {proposedClaims.map((claim: ProposedClaim, index: number) => (
            <div key={index} className="claim-row">
              <select aria-label={`Permission for claim ${index + 1}`} value={claim.permission} onChange={e => handleClaimChange(index, 'permission', e.target.value)}>
                <option value="read">read</option><option value="write">write</option><option value="create">create</option><option value="delete">delete</option>
              </select>
              <span>on</span>
              <select aria-label={`Model for claim ${index + 1}`} value={claim.model_id} onChange={e => handleClaimChange(index, 'model_id', e.target.value)}>
                {availableModels.map((m: { id: string; name: string }) => <option key={m.id} value={m.id}>{m.name}</option>)}
              </select>
            </div>
          ))}
          <button onClick={handleAddClaim}>+ Add Claim</button>
        </div>
        <button onClick={handleRunSimulation} disabled={loading || !simulateFor.trim()}>{loading ? 'Simulating...' : 'Run Simulation'}</button>
      </div>

      {result && (
        <div className="simulation-result">
          <h5>Simulation Result for '{result.simulated_for ?? 'unknown'}'</h5>
          {(result.risk_flags && result.risk_flags.length > 0) && (
            <div className="risk-flags">
              <strong>⚠️ Risk Flags:</strong>
              <ul>{result.risk_flags.map((flag: string, i: number) => <li key={i}>{flag}</li>)}</ul>
            </div>
          )}
          <div className="affected-models">
            <strong>Affected Models:</strong>
            <ul>
              {(result.affected_models || []).map((model: any) => (
                <li key={model.model_id}>{model.model_name} ({model.change}) {model.certified && <span className="badge">Certified</span>}</li>
              ))}
            </ul>
          </div>
        </div>
      )}
    </div>
  );
}