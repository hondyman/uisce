import { useState } from 'react';
import { useNotification } from './hooks/useNotification';
import { getFolderDiff } from './api';
import type { FolderDiff, FolderItemDetail } from './types';

interface FolderDiffViewerProps {
  folderId: string;
  onClose: () => void;
}

function renderItem(item: FolderItemDetail) {
  return <li key={item.item_id}>{item.name} ({item.item_type})</li>;
}

export default function FolderDiffViewer({ folderId, onClose }: FolderDiffViewerProps) {
  const [from, setFrom] = useState(new Date(Date.now() - 7 * 24 * 60 * 60 * 1000).toISOString());
  const [to, setTo] = useState(new Date().toISOString());
  const [diff, setDiff] = useState<FolderDiff | null>(null);
  const [loading, setLoading] = useState(false);
  const notification = useNotification();

  const loadDiff = async () => {
    setLoading(true);
    try {
      const data = await getFolderDiff(folderId, from, to);
      setDiff(data);
    } catch (error) {
      notification.error(`Failed to load diff: ${(error as Error).message}`);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="modal-overlay">
      <div className="modal-content">
        <h2>Folder Diff Viewer</h2>
        <div className="diff-controls">
          {/* In a real app, use a proper date picker component */}
          <label>From: <input type="datetime-local" value={from.substring(0, 16)} onChange={e => setFrom(new Date(e.target.value).toISOString())} /></label>
          <label>To: <input type="datetime-local" value={to.substring(0, 16)} onChange={e => setTo(new Date(e.target.value).toISOString())} /></label>
          <button onClick={loadDiff} disabled={loading}>{loading ? 'Loading...' : 'Compare'}</button>
        </div>
        {diff && (
          <div className="diff-results">
            <h4>Added ({diff.added.length})</h4>
            <ul>{diff.added.map(renderItem)}</ul>
            <h4>Removed ({diff.removed.length})</h4>
            <ul>{diff.removed.map(renderItem)}</ul>
            {/* Add Modified/Unchanged sections here */}
          </div>
        )}
        <div className="modal-actions">
          <button onClick={onClose}>Close</button>
        </div>
      </div>
    </div>
  );
}