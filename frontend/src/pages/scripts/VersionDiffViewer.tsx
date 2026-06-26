import { useMemo } from 'react';

function simpleLineDiff(left: string, right: string): { type: 'add'|'del'|'same'; text: string }[] {
  const l = left.split('\n');
  const r = right.split('\n');
  const max = Math.max(l.length, r.length);
  const rows: { type: 'add'|'del'|'same'; text: string }[] = [];
  for (let i = 0; i < max; i++) {
    if (l[i] === r[i]) rows.push({ type: 'same', text: r[i] ?? '' });
    else {
      if (l[i] !== undefined) rows.push({ type: 'del', text: l[i] });
      if (r[i] !== undefined) rows.push({ type: 'add', text: r[i] });
    }
  }
  return rows;
}

export function VersionDiffViewer({ left, right }: { left: string; right: string }) {
  const rows = useMemo(() => simpleLineDiff(left, right), [left, right]);
  return (
    <div className="diffViewer">
      {rows.map((row, idx) => (
        <pre key={idx} className={`row ${row.type}`}>
          {row.type === 'add' ? '+ ' : row.type === 'del' ? '- ' : '  '}
          {row.text}
        </pre>
      ))}
    </div>
  );
}

