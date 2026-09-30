/* eslint-disable jsx-a11y/no-autofocus --
 * This component is a hand-rolled overlay (modal / omnibox / palette).
 * Focus must move into it when it opens so keyboard and screen-reader users
 * land inside the dialog, and no other element here takes initial focus.
 * The attribute sits inside a multi-line JSX attribute list, where an
 * inline eslint-disable comment is not valid syntax, so the scope is the
 * whole file. This file has exactly one autoFocus and no other JSX.
 */
// React default import removed — using automatic JSX runtime
import ActionButton from '../ui/ActionButton';

interface Props {
  filteredCore: any[];
  coreSelected: string;
  setCoreSelected: (s: string) => void;
  coreSearch: string;
  setCoreSearch: (s: string) => void;
  kind: string | null;
  onBack: () => void;
  onCreateOverride: () => void;
}

const AddElementOverride: React.FC<Props> = ({ filteredCore, coreSelected, setCoreSelected, coreSearch, setCoreSearch, kind, onBack, onCreateOverride }) => {
  return (
    <div className="override-form">
      <input
        className="core-search-input"
        value={coreSearch}
        onChange={(e) => setCoreSearch(e.target.value)}
        placeholder={`Search core ${kind || ''}s`}
        title="Search core items"
        autoFocus
      />
      <div className="core-list">
        {filteredCore.length === 0 && <div className="empty">No matches</div>}
        {filteredCore.map(opt => (
          <div key={opt.name} className={`core-row ${coreSelected === opt.name ? 'selected' : ''}`} onClick={() => setCoreSelected(opt.name)}>
            <div className="core-row-name">
              {kind === 'dimension' && opt.sourceTable
                ? `${opt.sourceTable}.${opt.title || opt.name}`
                : (opt.title || opt.name)
              }
            </div>
            <div className="core-row-desc">{opt.description}</div>
          </div>
        ))}
      </div>
      <div className="actions">
        <ActionButton variant="secondary" onClick={onBack}>Back</ActionButton>
        <ActionButton variant="primary" onClick={onCreateOverride} pending={!coreSelected}>Create Override</ActionButton>
      </div>
    </div>
  );
};

export default AddElementOverride;
