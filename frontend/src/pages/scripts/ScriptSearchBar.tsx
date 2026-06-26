import { Dispatch, SetStateAction } from 'react';
import { ScriptSearchFilters, ScriptState } from '../../types/scripts';

export function ScriptSearchBar({
  query, setQuery,
  filters, setFilters
}: {
  query: string;
  setQuery: (q: string) => void;
  filters: ScriptSearchFilters;
  setFilters: Dispatch<SetStateAction<ScriptSearchFilters>>;
}) {
  const handleStateChange = (value: string) => {
    setFilters((prev) => ({ ...prev, state: (value || undefined) as ScriptState | undefined }));
  };

  const handleScopeChange = (value: string) => {
    setFilters((prev) => ({ ...prev, scope: value || undefined }));
  };

  const handleTagChange = (value: string) => {
    setFilters((prev) => ({ ...prev, tag: value || undefined }));
  };

  const handleStewardChange = (value: string) => {
    setFilters((prev) => ({ ...prev, steward: value || undefined }));
  };

  return (
    <div className="searchBar">
      <input placeholder="Search scripts..." value={query} onChange={e => setQuery(e.target.value)} />
      <select
        aria-label="Filter by script state"
        value={filters.state || ''}
        onChange={e => handleStateChange(e.target.value)}
      >
        <option value="">All states</option>
        <option value="draft">Draft</option>
        <option value="certified">Certified</option>
        <option value="published">Published</option>
        <option value="deprecated">Deprecated</option>
      </select>
      <select
        aria-label="Filter by script scope"
        value={filters.scope || ''}
        onChange={e => handleScopeChange(e.target.value)}
      >
        <option value="">All scopes</option>
        <option value="table">Table</option>
        <option value="semantic">Semantic</option>
      </select>
      <input placeholder="Tag" value={filters.tag || ''} onChange={e => handleTagChange(e.target.value)} />
      <input placeholder="Steward" value={filters.steward || ''} onChange={e => handleStewardChange(e.target.value)} />
    </div>
  );
}
