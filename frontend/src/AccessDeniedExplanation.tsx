import { useState } from 'react';
import { devError } from './utils/devLogger';
import { getDecisionTrace } from './api';
import type { AccessDecisionTrace } from './types';

interface AccessDeniedExplanationProps {
  reason: string;
  decisionId: string;
  onClose: () => void;
  onRequestAccess: () => void;
}

export default function AccessDeniedExplanation({ reason, decisionId, onClose, onRequestAccess }: AccessDeniedExplanationProps) {
  const [trace, setTrace] = useState<AccessDecisionTrace | null>(null);
  const [loadingTrace, setLoadingTrace] = useState(false);

  const handleShowTrace = async () => {
    setLoadingTrace(true);
    try {
      const traceData = await getDecisionTrace(decisionId);
      setTrace(traceData);
    } catch (error) {
      devError("Failed to fetch trace", error);
    } finally {
      setLoadingTrace(false);
    }
  };

  return (
    <div className="access-denied-explanation">
      <h4>🚫 Access Denied</h4>
      <p>{reason}</p>
      <div className="actions">
        <button onClick={onRequestAccess}>Request Access</button>
        <button onClick={handleShowTrace} disabled={loadingTrace}>
          {loadingTrace ? 'Loading Trace...' : 'Show Technical Trace'}
        </button>
        <button onClick={onClose}>Dismiss</button>
      </div>
      {trace && (
        <div className="technical-trace">
          <h5>Technical Trace</h5>
          <pre><code>{JSON.stringify(trace, null, 2)}</code></pre>
        </div>
      )}
    </div>
  );
}