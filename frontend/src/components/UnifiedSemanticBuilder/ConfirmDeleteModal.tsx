import { FC, useEffect } from 'react';
import * as TablerIcons from '@tabler/icons-react';
import './ConfirmDeleteModal.css';

interface Props {
  open: boolean;
  title?: string;
  message?: string;
  associated?: { id: string; display_name?: string; model_key?: string }[];
  onConfirm: () => void;
  onCancel: () => void;
}

const ConfirmDeleteModal: FC<Props> = ({ open, title = 'Confirm delete', message, associated = [], onConfirm, onCancel }) => {
  // Always register hooks in the same order. The effect will no-op when the
  // modal is closed to avoid calling hooks conditionally across renders.
  useEffect(() => {
    if (!open) return;
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onCancel();
    };

    document.addEventListener('keydown', handleEscape);
    return () => document.removeEventListener('keydown', handleEscape);
  }, [open, onCancel]);

  if (!open) return null;

  return (
    <div 
      className="confirm-modal-backdrop" 
      role="dialog" 
      aria-modal="true"
      aria-labelledby="confirm-dialog-title"
      onClick={(e) => {
        // Close on backdrop click
        if (e.target === e.currentTarget) onCancel();
      }}
    >
      <div className="confirm-modal">
        <div className="confirm-header">
          <h3 id="confirm-dialog-title">{title}</h3>
          <button 
            aria-label="Close" 
            onClick={onCancel} 
            className="btn-close"
          >
            <TablerIcons.IconX size={16} />
          </button>
        </div>
        <div className="confirm-body">
          <div className="confirm-icon">
            <TablerIcons.IconAlertTriangle size={28} color="#dc2626" />
          </div>
          <p>{message}</p>
          {associated.length > 0 && (
            <div className="associated-list">
              <strong>Associated custom models that will be deleted:</strong>
              <ul>
                {associated.map(a => (
                  <li key={a.id}>{a.display_name || a.model_key || a.id}</li>
                ))}
              </ul>
            </div>
          )}
        </div>
        <div className="confirm-actions">
          <button className="btn-cancel" onClick={onCancel}>
            Cancel
          </button>
          <button 
            className="btn-confirm" 
            onClick={onConfirm}
            autoFocus
          >
            <TablerIcons.IconTrash size={16} style={{ marginRight: '6px' }} />
            Delete
          </button>
        </div>
      </div>
    </div>
  );
};

export default ConfirmDeleteModal;
