// React default import removed — file uses only JSX and no React namespace types
import * as TablerIcons from '@tabler/icons-react';

interface Props {
  mode: 'override' | 'custom' | null;
  kind: string;
  displayName: string;
  onClose: () => void;
}

const AddElementModalHeader: React.FC<Props> = ({ mode, kind, displayName, onClose }) => {
  return (
    <div className="modal-header">
      <div className="modal-header-left">
        <h3>{mode === 'override' ? 'Override' : 'Add'} {kind && kind.charAt(0).toUpperCase() + kind.slice(1)}</h3>
        {displayName && (
          <div className="modal-model-name" title={displayName}>
            {displayName}
          </div>
        )}
      </div>
      <button className="close-btn" onClick={onClose} title="Close"><TablerIcons.IconX size={18} /></button>
    </div>
  );
};

export default AddElementModalHeader;
