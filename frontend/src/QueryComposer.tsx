// React default import not required
import type { QueryState } from './types';

/* eslint-disable no-unused-vars */
/* eslint-disable @typescript-eslint/no-unused-vars */
interface QueryComposerProps {
  query: QueryState;
  onChange: (_q: QueryState) => void;
  onCompile: () => void;
  onExecute: () => void;
}
/* eslint-enable @typescript-eslint/no-unused-vars */
/* eslint-enable no-unused-vars */

export default function QueryComposer({ query, onCompile, onExecute }: QueryComposerProps) {
  // A real implementation would have UI for filters, ordering, etc.
  // This is a simplified display.
  return (
    <div className="query-composer">
      <h3>Query Composer</h3>
      <div>
        <strong>Measures:</strong> {query.measures.join(', ') || 'None'}
      </div>
      <div>
        <strong>Dimensions:</strong> {query.dimensions.join(', ') || 'None'}
      </div>
      <div className="query-composer-actions">
        <button onClick={onCompile}>Compile</button>
        <button onClick={onExecute}>Execute</button>
      </div>
    </div>
  );
}