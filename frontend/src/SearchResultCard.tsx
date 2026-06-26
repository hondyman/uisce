import type { ReactNode } from 'react';
import { useNotification } from './hooks/useNotification';
import type { SemanticSearchResult } from './types';
import MiniEChart from './MiniEChart';
import CodeBlock from './CodeBlock';
import TabPreview from './TabPreview';

const Badge = ({ children, type = 'default' }: { children: ReactNode, type?: 'default' | 'reason' | 'restricted' | 'certified' }) => (
  <span className={`badge badge-${type}`}>
    {type === 'restricted' && '🔒 '}
    {type === 'certified' && '✅ '}
    {children}
  </span>
);

interface SearchResultCardProps {
  result: SemanticSearchResult;
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  onOpen: (_r: SemanticSearchResult) => void;
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  onExplain: (_r: SemanticSearchResult) => void;
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  onFeedback: (_r: SemanticSearchResult, _action: 'favorited' | 'ignored') => void;
}

export default function SearchResultCard({ result, onOpen, onExplain, onFeedback }: SearchResultCardProps) {
  const notification = useNotification();
  const normalizeChartType = (t: unknown): 'bar' | 'line' | 'pie' => {
    if (t === 'bar' || t === 'line' || t === 'pie') return t;
    return 'bar';
  };
  return (
    <div className="search-result-card">
      <div className="header">
        <div className="title-line">
          <strong>{result.name}</strong>
          <div className="badges">
            {result.reason && <Badge type="reason">{result.reason}</Badge>}
            {result.certified && <Badge type="certified">Certified</Badge>}
            {result.is_restricted && <Badge type="restricted">Restricted</Badge>}
            {result.popular && <Badge>Popular</Badge>}
          </div>
        </div>
        <small className="score" title={`Score: ${(result.score * 100).toFixed(1)}%`}>
          {result.type} • Score: {(result.score * 100).toFixed(1)}%
        </small>
      </div>
      <div className="preview">
        {result.preview?.kind === 'chart' && result.preview.chart && result.preview.chart.data && (
            <MiniEChart
              data={result.preview.chart.data}
              x={result.preview.chart.x}
              y={result.preview.chart.y}
              type={normalizeChartType(result.preview.chart.type)}
            />
        )}
        {result.preview?.kind === 'sql' && result.preview.sql && <CodeBlock language="sql" content={result.preview.sql.slice(0, 400)} />}
        {result.preview?.kind === 'tabs' && <TabPreview tabs={result.preview.tabs || []} />}
      </div>
      {result.has_access ? (
        <div className="actions">
          <button onClick={() => onFeedback(result, 'ignored')} title="Not relevant">👎</button>
          <button onClick={() => onFeedback(result, 'favorited')} title="Favorite">👍</button>
          <button onClick={() => onExplain(result)}>Why this matched</button>
          <button onClick={() => onOpen(result)}>Explore</button>
        </div>
      ) : (
        <div className="actions access-denied-overlay">
            <p>You don't have access. <button onClick={() => notification.info('Requesting access...')}>Request Access</button></p>
        </div>
      )}
    </div>
  );
}