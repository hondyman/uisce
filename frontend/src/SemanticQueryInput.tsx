import { useState } from 'react';
import { devError } from './utils/devLogger';
import type { SemanticQuery, NLQueryRequest, NLQueryResponse } from './types';
import { compileNLQuery } from './api';

interface SemanticQueryInputProps {
  // These parameter names are type-only in some call sites; disable the unused-var rule only for the type line
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    onQuery: (_viewName: string, _query: SemanticQuery) => void;
  // In a real app, these would come from context
  currentDatasource: string;
  currentUser: string;
}

export default function SemanticQueryInput({ onQuery, currentDatasource, currentUser }: SemanticQueryInputProps) {
  const [text, setText] = useState('');
  const [loading, setLoading] = useState(false);
  const [lastResponse, setLastResponse] = useState<NLQueryResponse | null>(null);

  const handleTranslate = async () => {
    if (!text.trim()) return;
    setLoading(true);
    try {
      const request: NLQueryRequest = {
        text: text.trim(),
        user_id: currentUser,
        datasource: currentDatasource,
      };

      const response = await compileNLQuery(request);
      setLastResponse(response);

      // Convert NL response to SemanticQuery format
      const semanticQuery: SemanticQuery = {
        dimensions: response.generated_query.dimensions,
        metrics: response.generated_query.measures,
        filters: response.generated_query.filters.map(f => ({
          field: f.field,
          op: f.operator,
          values: [f.value]
        })),
        order: response.generated_query.order_by?.map(o => [o.field, o.dir]),
      };

      onQuery('generated_view', semanticQuery);
      setText(''); // Clear input on success
    } catch (error) {
      devError("NL Query compilation failed:", error);
      // TODO: Show an error to the user
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="semantic-query-input">
      <input
        type="text"
        placeholder="Ask a question… e.g., 'average order value by region last quarter'"
        value={text}
        onChange={e => setText(e.target.value)}
        onKeyDown={e => e.key === 'Enter' && !loading && handleTranslate()}
        disabled={loading}
      />
      <button onClick={handleTranslate} disabled={loading || !text.trim()}>
        {loading ? 'Thinking...' : 'Ask'}
      </button>

      {lastResponse && (
        <div className="nl-query-results">
          <h4>Generated Query:</h4>
          <pre>{lastResponse.generated_query.sql}</pre>

          {lastResponse.governance_diff.blocked_metrics && lastResponse.governance_diff.blocked_metrics.length > 0 && (
            <div className="governance-warnings">
              <h5>Governance Applied:</h5>
              <ul>
                {lastResponse.governance_diff.blocked_metrics.map((metric) => (
                  <li key={metric}>Blocked metric: {metric}</li>
                ))}
              </ul>
            </div>
          )}

          {lastResponse.compliance_notes.length > 0 && (
            <div className="compliance-notes">
              <h5>Compliance Notes:</h5>
              <ul>
                {lastResponse.compliance_notes.map((note) => (
                  <li key={note}>{note}</li>
                ))}
              </ul>
            </div>
          )}
        </div>
      )}
    </div>
  );
}