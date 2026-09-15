import { useState, useEffect, useCallback } from 'react';
import { useNotification } from './hooks/useNotification';
import { DndContext, useDroppable, type DragEndEvent } from '@dnd-kit/core';
import { listFolders, addItemToFolder } from './api';
import type { FullFolder } from './types';
import FolderAnalyticsPanel from './FolderAnalyticsPanel';
import FolderDiffViewer from './FolderDiffViewer';

export const ItemTypes = {
  SAVED_ITEM: 'savedItem',
};

function Folder({ folder, onDropItem }: { folder: FullFolder; onDropItem: (_folderId: string, _item: any) => void }) {
  const { isOver, setNodeRef } = useDroppable({
    id: folder.id,
    data: { folderId: folder.id },
  });
  void onDropItem; // drop handling is wired centrally in FolderBrowser's DndContext

  const [isAnalyticsVisible, setIsAnalyticsVisible] = useState(false);
  const [isDiffVisible, setIsDiffVisible] = useState(false);

  return (
    <div ref={setNodeRef} className={`folder ${isOver ? 'over' : ''} ${isOver ? 'can-drop' : ''}`}>
      <div className="folder-header">
        <h4>📁 {folder.name}</h4>
        <div className="folder-actions">
          <button onClick={() => setIsDiffVisible(true)} title="Compare Versions">🔄</button>
          <button onClick={() => setIsAnalyticsVisible(!isAnalyticsVisible)} title="Toggle Analytics">📊</button>
        </div>
      </div>
      {isAnalyticsVisible && <FolderAnalyticsPanel folderId={folder.id} />}
      <ul>
        {folder.items.map(item => (
          <li key={item.item_id}>{item.name}</li>
        ))}
        {folder.items.length === 0 && <li className="placeholder">Drop items here</li>}
      </ul>
      {isDiffVisible && <FolderDiffViewer folderId={folder.id} onClose={() => setIsDiffVisible(false)} />}
    </div>
  );
}

export default function FolderBrowser() {
  const [folders, setFolders] = useState<FullFolder[]>([]);
  const notification = useNotification();

  const fetchFolders = useCallback(() => {
  listFolders().then(setFolders).catch((e) => { import('./utils/devLogger').then(({ devError }) => devError(e)).catch(() => {}); });
  }, []);

  useEffect(fetchFolders, [fetchFolders]);

  const handleDropItem = useCallback(async (folderId: string, item: { id: string; type: 'query' | 'workbook' }) => {
    try {
      await addItemToFolder(folderId, item.id, item.type);
      fetchFolders(); // Refresh folders to show the new item
    } catch (error) {
      notification.error(`Failed to add item to folder: ${(error as Error).message}`);
    }
  }, [fetchFolders]);

  const handleDragEnd = useCallback((event: DragEndEvent) => {
    const { active, over } = event;
    if (!over) return;
    const item = active.data.current as { id: string; type: 'query' | 'workbook' } | undefined;
    if (!item) return;
    handleDropItem(String(over.id), item);
  }, [handleDropItem]);

  return (
    <DndContext onDragEnd={handleDragEnd}>
      <div className="folder-browser">
        <h3>Folders</h3>
        <button onClick={fetchFolders}>Refresh</button>
        {folders.map(folder => (
          <Folder key={folder.id} folder={folder} onDropItem={handleDropItem} />
        ))}
        {folders.length === 0 && <p className="text-placeholder">No folders found.</p>}
      </div>
    </DndContext>
  );
}
