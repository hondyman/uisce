import React, { useEffect } from 'react';
import type { ModelCatalogNode } from '../../types/model';
import { devLog } from '../../utils/devLogger';

interface Props {
  model: ModelCatalogNode | null;
  onClose: () => void;
}

const ModelInfoModal: React.FC<Props> = ({ model, onClose }) => {
  useEffect(() => {
    if (model) {
      // Analytics/logging hook: record that the info modal was opened
      try {
        devLog('model_info_open', { modelId: model.id, modelKey: model.model_key });
      } catch (e) {
        // non-fatal
      }
    }
  }, [model]);

  if (!model) return null;

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal-content" onClick={(e) => e.stopPropagation()}>
        <header className="modal-header">
          <h3>{model.display_name || model.model_key}</h3>
        </header>
        <div className="modal-body">
          <p>{model.description || 'No description available.'}</p>
          <div className="modal-meta">
            <small>Status: {model.status}</small>
            <small>Version: v{model.version}</small>
          </div>
        </div>
        <footer className="modal-actions">
          <button className="btn" onClick={onClose}>Close</button>
        </footer>
      </div>
    </div>
  );
};

export default ModelInfoModal;
