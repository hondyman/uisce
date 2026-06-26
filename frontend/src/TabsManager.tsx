import { useState, useEffect, useCallback } from 'react';
import { useNotification } from './hooks/useNotification';
/* eslint-disable jsx-a11y/aria-proptypes */
import type { ViewMeta, QueryState, CompileResult, PageInfo, ColumnMeta, Explain, VizConfig, FullSavedQuery, FullWorkbook } from './types';
import { getViewIdentifier } from './types/views';
import { listViews } from './api';
import ExplorerTab from './ExplorerTab';
import SaveWorkbookModal from './SaveWorkbookModal';
import TourRunner from './TourRunner';

export type TabState = {
  id: string;
  title: string;
  savedId?: string;
  view?: ViewMeta;
  query: QueryState;
  compile?: CompileResult;
  result?: { rows: Record<string, unknown>[]; columns: ColumnMeta[]; page: PageInfo };
  viz?: VizConfig;
  explain?: Explain;
};

const createNewTab = (title: string): TabState => ({
  id: crypto.randomUUID(),
  title,
  query: { measures: [], dimensions: [], filters: [], order: [], limit: 50, offset: 0 },
});

export default function TabsManager() {
  const [tabs, setTabs] = useState<TabState[]>(() => [createNewTab('Tab 1')]);
  const [activeId, setActiveId] = useState<string | null>(tabs[0]?.id || null);
  const [views, setViews] = useState<ViewMeta[]>([]);
  const [activeTourId, setActiveTourId] = useState<string | null>(null);
  const [isSaveModalOpen, setIsSaveModalOpen] = useState(false);

  useEffect(() => {
    listViews().then(setViews);
  }, []);

  const addTab = () => {
    const newTab = createNewTab(`Tab ${tabs.length + 1}`);
    setTabs(t => [...t, newTab]);
    setActiveId(newTab.id);
  };

  const updateTab = useCallback((id: string, patch: Partial<TabState>) => {
    setTabs(ts => ts.map(t => (t.id === id ? { ...t, ...patch } : t)));
  }, []);

  const openSavedQuery = useCallback((q: FullSavedQuery) => {
    const id = crypto.randomUUID();
    // Allow saved queries to reference views by id or by name
  const viewForQuery = views.find(v => getViewIdentifier(v) === q.view_name || v.name === q.view_name);
    if (!viewForQuery) {
      const notification = useNotification();
      notification.error(`Could not open saved query: View '${q.view_name}' not found.`);
      return;
    }

    const newTab: TabState = {
      id,
      title: q.name,
      savedId: q.id,
      view: viewForQuery,
      query: (q.query as QueryState) || { measures: [], dimensions: [], filters: [], order: [], limit: 50, offset: 0 },
      viz: q.viz_config,
    };
    setTabs(t => [...t, newTab]);
    setActiveId(id);
  }, [views]);

  const openWorkbook = useCallback((workbook: FullWorkbook) => {
    const newTabs: TabState[] = workbook.tabs.map(wt => ({
      id: crypto.randomUUID(),
      title: wt.title,
  view: views.find(v => getViewIdentifier(v) === wt.view_name || v.name === wt.view_name),
      query: (wt.query as QueryState) || { measures: [], dimensions: [], filters: [], order: [], limit: 50, offset: 0 },
      viz: wt.viz_config,
      // Reset runtime state
      result: undefined,
      compile: undefined,
      explain: undefined,
    }));
    setTabs(newTabs);
    setActiveId(newTabs[0]?.id || null);
  }, [views]);

  const closeTab = (id: string) => {
    const tabIndex = tabs.findIndex(t => t.id === id);
    const newTabs = tabs.filter(t => t.id !== id);
    setTabs(newTabs);

    if (activeId === id) {
      if (newTabs.length > 0) {
        // Activate the previous tab, or the first tab if the closed one was the first
        const newActiveIndex = Math.max(0, tabIndex - 1);
        setActiveId(newTabs[newActiveIndex].id);
      } else {
        setActiveId(null);
        // Optional: create a new blank tab if the last one is closed
        addTab();
      }
    }
  };

  return (
    <div className="tabs-manager">
      <div className="tab-bar">
        <div role="tablist" aria-label="Explorer Tabs">
          {tabs.map((t, index) => {
            const isSelected = t.id === activeId;
            const controlId = t.id != null ? `tabpanel-${t.id}` : undefined;
            const ariaSelected = isSelected ? 'true' : undefined;
            return (
            <button
              /* webhint-disable axe/aria */
              key={t.id != null && typeof t.id !== 'object' ? String(t.id) : ''}
              id={`tab-${t.id}`}
              role="tab" // This is a valid ARIA role.
              
              aria-controls={controlId}
              {...(ariaSelected ? { 'aria-selected': ariaSelected } : {})}
              className={`tab-item-container ${t.id === activeId ? "active" : ""}`}
              onClick={(e) => {
                if ((e.target as HTMLElement).classList.contains('close-tab-btn')) {
                  e.stopPropagation();
                  closeTab(t.id);
                } else {
                  setActiveId(t.id);
                }
              }}
              onKeyDown={(e: React.KeyboardEvent) => {
                let nextIndex = -1;
                if (e.key === 'ArrowRight') nextIndex = (index + 1) % tabs.length;
                else if (e.key === 'ArrowLeft') nextIndex = (index - 1 + tabs.length) % tabs.length;
                if (nextIndex !== -1) {
                  e.preventDefault();
                  document.getElementById(`tab-${tabs[nextIndex].id}`)?.focus();
                }
              }}
              tabIndex={t.id === activeId ? 0 : -1}
            >
              <span className="tab-item-button">{t.title}</span>
              <span className="close-tab-btn" aria-hidden="true">&times;</span>
            </button>
            );
          })}
        </div>
        <button className="add-tab-btn" aria-label="Add new tab" onClick={addTab}>+</button>
        <button className="save-workbook-btn" onClick={() => setIsSaveModalOpen(true)}>Save Workbook</button>
      </div>
      <div className="tab-body">
        {tabs.filter(t => t.id === activeId).map(t => (
          <ExplorerTab
            key={t.id != null && typeof t.id !== 'object' ? String(t.id) : ''}
            tab={t}
            views={views}
            onChange={(patch) => updateTab(t.id, patch)}
            onOpenSavedQuery={openSavedQuery}
            onOpenWorkbook={openWorkbook}
            onStartTour={setActiveTourId}
          />
        ))}
      </div>
      {isSaveModalOpen && (
        <SaveWorkbookModal tabs={tabs} onClose={() => setIsSaveModalOpen(false)} onSaved={() => { /* Maybe show a confirmation */ }} />
      )}
      {activeTourId && (
        <TourRunner tourId={activeTourId} onComplete={() => setActiveTourId(null)} />
      )}
    </div>
  );
}