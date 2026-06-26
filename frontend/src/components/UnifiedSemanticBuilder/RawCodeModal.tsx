import React, { useMemo } from 'react';
import './RawCodeModal.css';

interface RawCodeModalProps {
  open: boolean;
  onClose: () => void;
  format: 'json' | 'yaml';
  onFormatChange: (f: 'json' | 'yaml') => void;
  generateJSON: () => string;
  generateYAML: () => string;
  title?: string;
}

const RawCodeModal: React.FC<RawCodeModalProps> = ({ open, onClose, format, onFormatChange, generateJSON, generateYAML, title = 'Raw Model Code' }) => {
  const text = useMemo(() => (format === 'json' ? generateJSON() : generateYAML()), [format, generateJSON, generateYAML]);
  if (!open) return null;
  return (
    <div className="raw-code-modal-backdrop" onClick={onClose}>
      <div className="raw-code-modal" onClick={e => e.stopPropagation()}>
        <div className="raw-code-modal-header">
          <h3>{title}</h3>
          <div className="raw-code-format-switch">
            <button className={format === 'json' ? 'active' : ''} onClick={() => onFormatChange('json')} disabled={format === 'json'}>JSON</button>
            <button className={format === 'yaml' ? 'active' : ''} onClick={() => onFormatChange('yaml')} disabled={format === 'yaml'}>YAML</button>
          </div>
          <button className="raw-code-close" onClick={onClose}>×</button>
        </div>
  <textarea className="raw-code-textarea" value={text} readOnly spellCheck={false} aria-label="Raw code" />
        <div className="raw-code-footer">
          <button onClick={onClose}>Close</button>
        </div>
      </div>
    </div>
  );
};

export default RawCodeModal;
