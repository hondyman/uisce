import { useState, useEffect } from 'react';
import { HistoryEntry, QueryState } from './types';
import { listHistory } from './api';

export default function HistoryPanel({ onLoadQuery }: { onLoadQuery: (_q: QueryState) => void }) {
  const [history, setHistory] = useState<HistoryEntry[]>([]);
  useEffect(() => { 
    listHistory().then(setHistory).catch(err => import('./utils/devLogger').then(({ devError }) => devError("Failed to load history:", err)).catch(() => {}));
  }, []);
  return (
    <div className="history-panel">
      <h4>History</h4>
      <ul>
        {history.map(h => (
          <li key={h.id}>
            <button onClick={() => h.request && onLoadQuery(h.request)}>
              {h.name} <small>{h.last_run_at ? new Date(h.last_run_at).toLocaleDateString() : ''}</small>
            </button>
          </li>
        ))}
      </ul>
    </div>
  );
}