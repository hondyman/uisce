// React import removed — JSX runtime handles createElement
import { useState } from 'react';
import { ViewMeta } from './types';

export default function ViewList({ views, onSelect }: { views: ViewMeta[], onSelect: (v: ViewMeta) => void }) {
  const [search, setSearch] = useState('');
  const filtered = views.filter(v => v.name.toLowerCase().includes(search.toLowerCase()) || v.tags?.some(t => t.includes(search)));
  return (
    <div className="view-list">
      <input
        type="search"
        aria-label="Search views"
        placeholder="Search views..."
        value={search}
        onChange={e => setSearch(e.target.value)}
      />
      <ul>
        {filtered.map(v => (
          <li key={v.name}>
            <button onClick={() => onSelect(v)}>
              <strong>{v.name}</strong>
              <small>{v.description}</small>
            </button>
          </li>
        ))}
      </ul>
    </div>
  );
}