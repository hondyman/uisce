import React, { useState } from 'react';
import { evaluateRuleWasm } from './rules/wasmRuntime';

interface RuleSimulatorProps {
  rule: any;
}

export function RuleSimulator({ rule }: RuleSimulatorProps) {
  const [context, setContext] = useState<string>('{}');
  const [result, setResult] = useState<boolean | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const simulate = async () => {
    setLoading(true);
    setError(null);
    try {
      const ctx = JSON.parse(context);
      const res = await evaluateRuleWasm(rule, ctx);
      setResult(res);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
      setResult(null);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="rule-simulator">
      <h3>Rule Simulator (WASM)</h3>
      <div>
        <label>Context (JSON):</label>
        <textarea
          value={context}
          onChange={(e) => setContext(e.target.value)}
          rows={4}
          cols={50}
        />
      </div>
      <button onClick={simulate} disabled={loading}>
        {loading ? 'Simulating...' : 'Simulate'}
      </button>
      {error && <div className="error">Error: {error}</div>}
      {result !== null && (
        <div className="result">
          Result: {result ? 'TRUE' : 'FALSE'}
        </div>
      )}
    </div>
  );
}