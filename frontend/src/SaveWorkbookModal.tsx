/* eslint-disable jsx-a11y/no-autofocus --
 * This component is a hand-rolled overlay (modal / omnibox / palette).
 * Focus must move into it when it opens so keyboard and screen-reader users
 * land inside the dialog, and no other element here takes initial focus.
 * The attribute sits inside a multi-line JSX attribute list, where an
 * inline eslint-disable comment is not valid syntax, so the scope is the
 * whole file. This file has exactly one autoFocus and no other JSX.
 */
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
  const notification = useNotification();

  const [name, setName] = useState('');
  const [description, setDescription] = useState('');

  const handleSave = async () => {
    if (!name) {
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
      notification.success(`Workbook "${savedWorkbook.name}" saved!`);
      onSaved(savedWorkbook.id);
      onClose();
    } catch (error) {
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