import { useState } from 'react';
import { saveQuery, updateQuery } from './api';
import { useNotification } from './hooks/useNotification';
import type { TabState } from './TabsManager';
import { getViewIdentifier } from './types/views';
import type { FullSavedQuery } from './types';
import type { View } from './types/views';

/* eslint-disable no-unused-vars */
/* eslint-disable @typescript-eslint/no-unused-vars */
interface SaveControlsProps {
  tab: TabState;
  onSave: (_savedId: string, _name: string) => void;
}
/* eslint-enable @typescript-eslint/no-unused-vars */
/* eslint-enable no-unused-vars */

export default function SaveControls({ tab, onSave }: SaveControlsProps) {
  const [name, setName] = useState(tab.title || '');
  const notification = useNotification();

  const handleSave = async () => {
    if (!tab.view) {
      notification.error('A view must be selected to save a query.');
      return;
    }

    const payload: Omit<FullSavedQuery, 'id'> = {
      name,
      view_name: getViewIdentifier(tab.view as Partial<View> | undefined),
      query: tab.query,
      viz_config: tab.viz,
    };

    if (tab.savedId) {
      await updateQuery(tab.savedId, payload);
      onSave(tab.savedId, name);
      notification.success('Query updated!');
    } else {
      const saved = await saveQuery(payload);
      onSave(saved.id, name);
      notification.success('Query saved!');
    }
  };

  return (
    <div className="save-controls">
      <input value={name} onChange={(e) => setName(e.target.value)} placeholder="Query name" />
      <button onClick={handleSave} disabled={!tab.view || !name}>{tab.savedId ? 'Update' : 'Save'}</button>
    </div>
  );
}