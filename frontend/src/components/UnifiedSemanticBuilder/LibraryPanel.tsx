// React default import removed — using automatic JSX runtime
import * as TablerIcons from '@tabler/icons-react';

// Minimal CoreOption shape used by the library panel
interface CoreOption {
  name: string;
  title?: string;
  sql?: string;
  description?: string;
  category?: string;
  type?: string;
}

interface Props {
  open: boolean;
  libraryByCategory: Record<string, CoreOption[]>;
  librarySearch: string;
  setLibrarySearch: (s: string) => void;
  onClose: () => void;
  onSelect: (opt: CoreOption) => void;
}

const LibraryPanel: React.FC<Props> = ({ open, libraryByCategory, librarySearch, setLibrarySearch, onClose, onSelect }) => {
  if (!open) return null;
  return (
    <div className={`library-panel open`}>
      <div className="panel-header">
        <div className="panel-title">
          <TablerIcons.IconChartBar size={20} />
          <h4>Financial Calculation Library</h4>
        </div>
        <button className="close-btn" onClick={onClose} title="Close">
          <TablerIcons.IconX size={18} />
        </button>
      </div>

      <div className="panel-search">
        <div className="search-input-wrapper">
          <TablerIcons.IconSearch size={16} className="search-icon" />
          <input 
            value={librarySearch} 
            onChange={(e) => setLibrarySearch(e.target.value)} 
            placeholder="Search calculations, formulas, or descriptions..." 
            className="search-input"
          />
        </div>
      </div>

      <div className="panel-content">
        {Object.keys(libraryByCategory).map(cat => (
          <div key={cat} className="category-section">
            <div className="category-header">
              <h5>{cat.charAt(0).toUpperCase() + cat.slice(1)}</h5>
              <span className="calc-count">{libraryByCategory[cat].length}</span>
            </div>
            <div className="category-calculations">
              {libraryByCategory[cat].map(opt => (
                <div 
                  key={opt.name} 
                  className="calculation-card"
                  onClick={() => onSelect(opt)}
                >
                  <div className="calc-header">
                    <div className="calc-name">{opt.title || opt.name}</div>
                    <div className="calc-type-badge">{cat}</div>
                  </div>
                  <div className="calc-formula">{opt.sql}</div>
                </div>
              ))}
            </div>
          </div>
        ))}
        {Object.keys(libraryByCategory).length === 0 && (
          <div className="empty-state">
            <TablerIcons.IconSearch size={48} className="empty-icon" />
            <h6>No calculations found</h6>
            <p>Try adjusting your search terms</p>
          </div>
        )}
      </div>
    </div>
  );
};

export default LibraryPanel;
