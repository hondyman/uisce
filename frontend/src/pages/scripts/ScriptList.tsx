import { ScriptSummary } from '../../types/scripts';

export function ScriptList({ scripts, onOpen }: { scripts: ScriptSummary[]; onOpen: (id: string) => void }) {
  return (
    <div className="list">
      {scripts.map(s => (
        <div key={s.id} className="card">
          <div className="cardHeader">
            <span className={`badge state-${s.state}`}>{s.state}</span>
            <span className="name clickable" onClick={() => onOpen(s.id)}>{s.name}</span>
          </div>
          <div className="meta">
            <div>Scope: {s.scope}</div>
            <div>Version: {s.latestVersion}</div>
            <div>Steward: {s.steward || '—'}</div>
            <div>Tags: {(s.domainTags ?? []).join(', ')}</div>
          </div>
        </div>
      ))}
    </div>
  );
}
