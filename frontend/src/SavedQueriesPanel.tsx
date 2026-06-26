import { Suspense, useMemo, useEffect, useCallback, useState } from 'react';
import { devError } from './utils/devLogger';
import { useDrag } from 'react-dnd';
import { listSavedQueries, cloneQuery, deleteQuery, getSavedQuery, getPreview } from './api';
import { useConfirm } from './components/ConfirmProvider';
import { useNotification } from './hooks/useNotification';
import type { SavedQuery, FullSavedQuery, ViewMeta, ExecuteResult } from './types';
import { LazyECharts } from './components/ChartLoader';
import { useDebounce } from './hooks/useDebounce';
import { ItemTypes } from './FolderBrowser';
import DuplicateQueriesPanel from './DuplicateQueriesPanel';

// eslint-disable-next-line @typescript-eslint/no-unused-vars
function DraggableQueryRow({ q, children }: { q: SavedQuery; children: React.ReactNode }) {
  const [{ isDragging }, drag] = useDrag(() => ({
    type: ItemTypes.SAVED_ITEM,
    item: { id: q.id, type: 'query' },
    // narrow the monitor shape locally to avoid `any` in many call sites
    collect: (monitor: { isDragging?: () => boolean } | undefined) => ({
      isDragging: !!monitor?.isDragging?.(),
    }),
  }));

  return (
    <div ref={drag} className={`draggable-row ${isDragging ? 'dragging' : ''}`}>
      {children}
    </div>
  );
}
interface SavedQueriesPanelProps {
  onOpen: (q: FullSavedQuery) => void;
  views: ViewMeta[];
}

export default function SavedQueriesPanel({ onOpen, views }: SavedQueriesPanelProps) {
  const [saved, setSaved] = useState<SavedQuery[]>([]);
  const [scope, setScope] = useState<'mine' | 'shared' | 'duplicates'>('mine');
  const [search, setSearch] = useState('');
  const [previewData, setPreviewData] = useState<ExecuteResult | null>(null);
  const [previewLoading, setPreviewLoading] = useState(false);
  const [hoveredItem, setHoveredItem] = useState<string | null>(null);
  const [selectedDatasource, setSelectedDatasource] = useState<string>('');
  const debouncedSearch = useDebounce(search, 300);

  const datasources = useMemo(() => [...new Set(views.map(v => v.schema))], [views]);

  useEffect(() => {
    if (datasources.length > 0 && !selectedDatasource && datasources[0]) {
      setSelectedDatasource(datasources[0]);
    }
  }, [datasources, selectedDatasource]);

  const fetchSaved = useCallback(() => {
    if (scope !== 'duplicates') {
      listSavedQueries({ scope, search: debouncedSearch }).then(setSaved).catch(err => devError('Failed to load saved queries:', err));
    }
  }, [scope, debouncedSearch]);

  useEffect(fetchSaved, [fetchSaved]);

  const confirm = useConfirm();
  const notification = useNotification();

  const handleOpen = async (id: string) => {
    try {
      const fullQuery = await getSavedQuery(id);
      onOpen(fullQuery);
    } catch (error) {
      const notification = useNotification();
      notification.error(`Failed to open query: ${(error as Error).message}`);
    }
  };

  const handleClone = async (id: string) => {
    await cloneQuery(id);
    fetchSaved(); // Refetch to get the new list with the clone
  };

  const handleDelete = async (id: string) => {
    const confirm = useConfirm();
    const notification = useNotification();
    if (await confirm({ title: 'Delete saved query', description: 'Are you sure you want to delete this query? This cannot be undone.' })) {
      try {
        await deleteQuery(id);
        setSaved(s => s.filter(q => q.id !== id));
        notification.success('Query deleted');
      } catch (err) {
        notification.error('Failed to delete query');
      }
    }
  };

  const handleMouseEnter = (q: SavedQuery) => {
    setHoveredItem(q.id);
    if (!q.preview_available) return;
    setPreviewLoading(true);
    getPreview(q.id)
      .then(setPreviewData)
      .catch(() => setPreviewData(null))
      .finally(() => setPreviewLoading(false));
  };

  const handleMouseLeave = () => {
    setHoveredItem(null);
    setPreviewData(null);
  };

  return (
    <div className="saved-queries-panel">
      <h4>Saved Queries</h4>
      <div className="panel-controls">
        <div className="scope-tabs">
          <button onClick={() => setScope('mine')} className={scope === 'mine' ? 'active' : ''}>My Queries</button>
          <button onClick={() => setScope('shared')} className={scope === 'shared' ? 'active' : ''}>Shared</button>
          <button onClick={() => setScope('duplicates')} className={scope === 'duplicates' ? 'active' : ''}>Duplicates</button>
        </div>
        {scope === 'duplicates' ? (
          <select aria-label="Select Datasource" value={selectedDatasource} onChange={e => setSelectedDatasource(e.target.value)}>
            {datasources.map(ds => <option key={ds} value={ds}>{ds}</option>)}
          </select>
        ) : (
          <input type="search" placeholder="Search name/description..." value={search} onChange={e => setSearch(e.target.value)} />
        )}
      </div>
      {scope === 'duplicates' ? (
        <DuplicateQueriesPanel datasourceId={selectedDatasource} />
      ) : (
        <>
          <ul>
            {saved.map(_q => (
              <li key={_q.id} onMouseEnter={() => handleMouseEnter(_q)} onMouseLeave={handleMouseLeave}>
                <DraggableQueryRow q={_q}>
                  <div className="query-row-content">
                    <div className="query-info" onClick={() => handleOpen(_q.id)}>
                      <span className="query-name" title={`Open "${_q.name}"`}>{_q.name}</span>
                      <small className="query-meta">
                        in <strong>{_q.view_name}</strong>
                        {scope === 'shared' && ` by ${_q.owner_user_id}`}
                      </small>
                      <small className="query-meta">
                        {_q.last_run_at ? `Ran ${new Date(_q.last_run_at).toLocaleDateString()}` : 'Never run'}
                      </small>
                    </div>
                    <div className="query-actions">
                      <button onClick={() => handleClone(_q.id)} title="Clone">Clone</button>
                      <button onClick={() => handleDelete(_q.id)} title="Delete">Delete</button>
                    </div>
                  </div>
                </DraggableQueryRow>
              </li>
            ))}
          </ul>
          {hoveredItem && (
            <>
              {previewData && (
                  <div className="preview-tooltip">
                    {/* The backend preview may return either an execute result (rows/columns)
                        or a chart preview shape. Narrow chart into a local `previewChart` variable. */}
                    {(() => {
                      const previewChart = (previewData as unknown as { chart?: unknown })?.chart;
                      return previewChart ? (
                        <Suspense fallback={<div className="chart-loading-small">Loading preview...</div>}>
                          <div className="chart-preview-size"><LazyECharts option={previewChart as unknown} /></div>
                        </Suspense>
                      ) : <pre>{JSON.stringify(previewData, null, 2)}</pre>;
                    })()}
                  </div>
              )}
              {previewLoading && <div className="preview-tooltip">Loading preview...</div>}
            </>
          )}
        </>
      )}
    </div>
  );
}