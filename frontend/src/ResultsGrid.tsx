// React default import not required
import type { ColumnMeta, PageInfo } from './types';

/* eslint-disable no-unused-vars */
/* eslint-disable @typescript-eslint/no-unused-vars */
export default function ResultsGrid({ rows, columns, page, onPageChange }: { rows: Record<string, unknown>[], columns: ColumnMeta[], page: PageInfo, onPageChange: (p: PageInfo) => void }) {
/* eslint-enable @typescript-eslint/no-unused-vars */
/* eslint-enable no-unused-vars */
  return (
    <div className="results-grid">
      <table>
        <thead>
          <tr>{columns.map(c => <th key={c.name}>{c.name}</th>)}</tr>
        </thead>
        <tbody>
          {rows.map((r, i) => (
            <tr key={i}>{columns.map(c => <td key={c.name}>{String(r[c.name])}</td>)}</tr>
          ))}
        </tbody>
      </table>
      <div className="pagination">
        <button disabled={!page.offset || page.offset === 0} onClick={() => onPageChange({ ...page, offset: (page.offset || 0) - (page.limit || 50) })}>Prev</button>
        <button disabled={!page.hasNext} onClick={() => onPageChange({ ...page, offset: (page.offset || 0) + (page.limit || 50) })}>Next</button>
      </div>
    </div>
  );
}