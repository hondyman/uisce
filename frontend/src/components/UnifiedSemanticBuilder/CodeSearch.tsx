// React default import removed — using named imports where needed
import { ProfessionalSearchInput } from '../common/ProfessionalSearchInput';
import { useGlobalSearch } from '../../contexts/GlobalSearchContext';
import { IconButton, Tooltip } from '@mui/material';
import * as PaletteIcons from './icons';

interface Props {
  matchIndex: number;
  matchCount: number;
  setMatchIndex: (i: number) => void;
  setMatchCount: (c: number) => void;
  onCopy: () => void;
  onDownload: () => void;
}

const CodeSearch: React.FC<Props> = ({
  matchIndex,
  matchCount,
  setMatchIndex,
  setMatchCount,
  onCopy,
  onDownload,
}) => {
  const { searchTerm: ctxTerm, setSearchTerm: ctxSet } = useGlobalSearch();
  const term = ctxTerm;
  const setTerm = ctxSet;
  const handleNavigate = (direction: number) => {
    // Fire an event that the CodePanel can listen to
    window.dispatchEvent(new CustomEvent('semlayer.navigateMatch', { detail: { direction } }));
  };

  return (
    <div className="prism-toolbar">
      <div className="prism-search-container">
        <ProfessionalSearchInput
          value={term}
          onChange={(value) => {
            setTerm && setTerm(value);
            setMatchIndex(0);
            setMatchCount(0); // Reset on new search
          }}
          onSuggestionSelect={() => {}}
          suggestions={[]}
          placeholder="Search in code..."
          showSuggestions={false}
          currentMatch={matchIndex + 1}
          totalMatches={matchCount}
          onNavigateMatch={handleNavigate}
          navigationEnabled={true}
        />
      </div>
      <div className="prism-actions">
        <Tooltip title="Copy" placement="bottom" arrow>
          <IconButton size="small" onClick={onCopy} aria-label="Copy code">
            <PaletteIcons.IconCopy size={16} />
          </IconButton>
        </Tooltip>
        <Tooltip title="Download" placement="bottom" arrow>
          <IconButton size="small" onClick={onDownload} aria-label="Download code">
            <PaletteIcons.IconDownload size={16} />
          </IconButton>
        </Tooltip>
      </div>
    </div>
  );
};

export default CodeSearch;