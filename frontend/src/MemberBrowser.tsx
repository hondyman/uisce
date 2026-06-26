import { ViewMeta, QueryState, ViewMember } from './types';

export default function MemberBrowser({ view, query, onChange }: { view: ViewMeta, query: QueryState, onChange: (_q: QueryState) => void }) {
  const toggle = (type: 'measures' | 'dimensions', name: string) => {
  const current = query[type] || [];
  const arr = current.includes(name) ? current.filter((m: string) => m !== name) : [...current, name];
    onChange({ ...query, [type]: arr });
  };

  const buildTitle = (member: ViewMember) => {
    const parts = [member.description || member.label, `Type: ${member.type}`];
    if (member.pii) {
      parts.push('Contains PII');
    }
    return parts.join('\n');
  };

  return (
    <div className="member-browser">
      <h4>Measures</h4>
  {(view.measures || []).map((m: ViewMember) => (
        <label key={m.name} title={buildTitle(m)}>
          <input type="checkbox" checked={Array.isArray(query.measures) && query.measures.includes(m.name)} onChange={() => toggle('measures', m.name)} />
          {m.label} {m.pii && <span aria-label="Contains PII">🔒</span>}
        </label>
      ))}
      <h4>Dimensions</h4>
  {(view.dimensions || []).map((d: ViewMember) => (
        <label key={d.name} title={buildTitle(d)}>
          <input type="checkbox" checked={Array.isArray(query.dimensions) && query.dimensions.includes(d.name)} onChange={() => toggle('dimensions', d.name)} />
          {d.label} {d.pii && <span aria-label="Contains PII">🔒</span>}
        </label>
      ))}
    </div>
  );
}