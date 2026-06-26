// React default import removed — using automatic JSX runtime

interface Props {
  filteredNodesCount?: number;
  onSearch?: (q: string) => void;
}

const SearchPanel: React.FC<Props> = ({ filteredNodesCount = 0, onSearch }) => {
  return (
    <aside className="workspace-search-panel">
      <div className="workspace-search-inner">
        <label htmlFor="builder-search">Search catalog</label>
        <input id="builder-search" className="input input-sm" placeholder="Search tables or columns" onChange={(e) => onSearch?.(e.target.value)} />
        <div className="workspace-search-results"><strong>Results:</strong> {filteredNodesCount}</div>
      </div>
    </aside>
  );
};

export default SearchPanel;
