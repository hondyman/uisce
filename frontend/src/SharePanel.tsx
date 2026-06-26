// React default import removed (not used as a value)
import type { TabState } from './TabsManager';
import { saveToDashboard } from './api';
import { useNotification } from './hooks/useNotification';

interface SharePanelProps {
  tab: TabState;
}

export default function SharePanel({ tab }: SharePanelProps) {
  const notification = useNotification();
  const handlePin = async () => {
    if (!tab.view || !tab.viz) {
      notification.warning('A view and visualization must be selected to pin to a dashboard.');
      return;
    }

  // Narrow the shape of `tab.view` locally to avoid blanket `as any` casts
  const view = tab.view as { id?: string; name?: string; title?: string } | undefined;
  const viewIdentifier = view?.id || view?.name || '';
  const displayName = view?.title || view?.name || String(view?.id || '').slice(0, 8);

    const tile = {
      title: `${tab.title} - ${displayName}`,
      viewName: viewIdentifier,
      query: tab.query,
      viz: tab.viz,
      refresh: 'daily' as const, // default
    };

    try {
      await saveToDashboard(tile);
      notification.success('Pinned to dashboard!');
    } catch (e) {
      notification.error(`Failed to pin to dashboard: ${(e as Error).message}`);
    }
  };

  return (
    <div className="share-panel">
      <h4>Actions</h4>
      <button onClick={handlePin} disabled={!tab.view || !tab.viz}>
        Pin to dashboard
      </button>
    </div>
  );
}