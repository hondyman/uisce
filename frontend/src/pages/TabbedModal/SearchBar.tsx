// React default import not required with new JSX transform

interface SearchResult {
  id: string;
  type: 'table' | 'column';
  label: string;
  nodeId: string;
  tableName: string;
}

/* eslint-disable no-unused-vars */
// eslint-disable-next-line @typescript-eslint/no-unused-vars
interface SearchBarProps {
  searchTerm: string;
  searchResults: SearchResult[];
  onSearchChange: (_term: string) => void;
  onSearchSelect: (_item: SearchResult) => void;
}
/* eslint-enable no-unused-vars */

const SearchBar: React.FC<SearchBarProps> = ({
  searchTerm,
  searchResults,
  onSearchChange,
  onSearchSelect
}) => {
  const ariaExpanded = searchResults.length > 0 ? 'true' : 'false';
  const [highlighted, setHighlighted] = React.useState<number | null>(null);
  return (
    <div className="search-container">
      // eslint-disable-next-line jsx-a11y/aria-proptypes
      <input
        type="text"
        placeholder="Search tables and columns..."
        value={searchTerm}
        onChange={(e) => onSearchChange(e.target.value)}
        onKeyDown={(e) => {
          if (!searchResults || searchResults.length === 0) return;
          if (e.key === 'ArrowDown') {
            e.preventDefault();
            setHighlighted((h) => Math.min((h ?? -1) + 1, searchResults.length - 1));
          } else if (e.key === 'ArrowUp') {
            e.preventDefault();
            setHighlighted((h) => Math.max((h ?? 0) - 1, -1));
          } else if (e.key === 'Enter') {
            if (highlighted !== null && highlighted >= 0 && highlighted < searchResults.length) {
              onSearchSelect(searchResults[highlighted]);
            }
          } else if (e.key === 'Escape') {
            setHighlighted(null);
          }
        }}
        aria-controls="search-results-dropdown"
        aria-controls="search-results-dropdown"
        aria-autocomplete="list"
        className="search-input"
      />
      <span className="search-icon">🔍</span>
      {searchResults.length > 0 && (
        <div id="search-results-dropdown" className="search-results-dropdown" role="listbox" aria-label="Search results" aria-activedescendant={highlighted !== null && highlighted >= 0 ? `search-result-${searchResults[highlighted].id}` : undefined}>
          {searchResults.map((item) => (
            <div 
              key={item.id} 
              id={`search-result-${item.id}`}
              className={`search-result-item ${item.type} ${highlighted !== null && searchResults[highlighted]?.id === item.id ? 'highlighted' : ''}`}
              role="option"
              onClick={() => onSearchSelect(item)}
              onMouseEnter={() => setHighlighted(searchResults.findIndex(r => r.id === item.id))}
            >
              <span className="search-result-icon">
                {item.type === 'table' ? '📋' : '📄'}
              </span>
              <span className="search-result-text">{item.label}</span>
            </div>
          ))}
        </div>
      )}
      {searchTerm && searchResults.length === 0 && (
        <div className="search-no-results">No results found</div>
      )}
    </div>
  );
};

export default SearchBar;