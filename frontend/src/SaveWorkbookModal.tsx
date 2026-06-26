import { useState } from 'react';
import { useNotification } from './hooks/useNotification';
import { createWorkbook } from './api';
import type { TabState, WorkbookTab } from './types';
import { getViewIdentifier } from './types/views';

interface SaveWorkbookModalProps {
  tabs: TabState[];
  onClose: () => void;
  onSaved: (workbookId: string) => void;
}

export default function SaveWorkbookModal({ tabs, onClose, onSaved }: SaveWorkbookModalProps) {
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');

  const handleSave = async () => {
    if (!name) {
      const notification = useNotification();
      notification.error('Workbook name is required.');
      return;
    }

    const workbookTabs: WorkbookTab[] = tabs
      .filter(t => t.view) // Only save tabs that have a view
      .map((t, i) => ({
        title: t.title,
        view_name: getViewIdentifier(t.view as any),
        query: t.query,
        viz_config: t.viz,
        position: i,
      }));

    try {
      const savedWorkbook = await createWorkbook({ name, description, tabs: workbookTabs });
      const notification = useNotification();
      notification.success(`Workbook "${savedWorkbook.name}" saved!`);
      onSaved(savedWorkbook.id);
      onClose();
    } catch (error) {
      const notification = useNotification();
      notification.error(`Failed to save workbook: ${(error as Error).message}`);
    }
  };

  return (
    <div className="modal-overlay">
      <div className="modal-content">
        <h2>Save Workbook</h2>
        <input autoFocus placeholder="Workbook Name" value={name} onChange={e => setName(e.target.value)} />
        <textarea placeholder="Description (optional)" value={description} onChange={e => setDescription(e.target.value)} />
        <div className="modal-actions">
          <button onClick={onClose}>Cancel</button>
          <button onClick={handleSave} disabled={!name}>Save</button>
        </div>
      </div>
    </div>
  );
}