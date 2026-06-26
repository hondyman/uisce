import { FC, useState, useRef, useMemo, useCallback, useEffect, MouseEvent as ReactMouseEvent } from 'react';
import { devLog, devWarn } from '../../utils/devLogger';
import { CodePanel } from './CodePanel';
import CanvasPanel from './CanvasPanel';
import SemanticPalette from './SemanticPalette';
import './SemanticModelEditor.css';
import { ShowCode, SemanticModel } from './types';
import { useGlobalSearch } from '../../contexts/GlobalSearchContext';
import { CoreOption } from './financialCalculations';
import EnhancedTileForm from './EnhancedTileForm';
import ExtendsForm, { BaseModelOption } from './ExtendsForm';

interface SemanticModelEditorProps {
  semanticModel: any;
  modelName: string;
  showCode: ShowCode | null;
  onToggleCode: (fmt: ShowCode | null) => void;
  generateJSON: () => string;
  generateYAML: () => string;
  generateCustomJSON?: () => string;
  generateCustomYAML?: () => string;
  generateCoreJSON?: () => string;
  generateCoreYAML?: () => string;
  generateMergedModelObject?: () => any;
  codeEditable?: boolean;
  extendsModel?: any;
  selectedModel?: any;
  selectedColumn: any;
  onAdd?: (type: 'dimension' | 'measure' | 'filter' | 'join' | 'extends', targetTable?: string | { id: string; qualified_path: string } | null) => void;
  removeSemanticElement: (id: string) => void;
  toggleElementEdit: (type: 'dimensions' | 'measures' | 'filters' | 'joins', id: string) => void;
  updateSemanticElement: (type: string, id: string, updates: any) => void;
  coreOptions?: CoreOption[];
  editMode?: boolean;
  availableBaseModels?: BaseModelOption[];
  onChangeExtends: (extendsModel: any) => void;
  onImportCode?: (text: string, format: 'json'|'yaml'|'jsonc'|null) => Promise<void> | void;
  allowExtendsChangeOutsideEdit?: boolean;
  model?: any;
  onChange?: (model: any) => void;
  onDelete?: (id: string) => void;
  onEdit?: (id: string) => void;
  onDuplicate?: (id: string) => void;
  onMove?: (id: string, direction: 'up' | 'down') => void;
  onToggle?: (id: string) => void;
  onSelect?: (id: string) => void;
  selectedElementId?: string;
  setSelectedElementId?: (id: string | null) => void;
  readOnly?: boolean;
  showAddButtons?: boolean;
  showDeleteButtons?: boolean;
  showEditButtons?: boolean;
  showDuplicateButtons?: boolean;
  showMoveButtons?: boolean;
  showToggleButtons?: boolean;
  showSelectButtons?: boolean;
}

const SemanticModelEditor: FC<SemanticModelEditorProps> = ({
  semanticModel,
  modelName,
  showCode,
  onToggleCode,
  generateJSON,
  generateYAML,
  generateCustomJSON,
  generateCustomYAML,
  generateCoreJSON,
  generateCoreYAML,
  generateMergedModelObject,
  codeEditable = false,
  extendsModel = null,
  selectedModel = null,
  selectedColumn,
  onAdd,
  removeSemanticElement,
  toggleElementEdit,
  updateSemanticElement,
  coreOptions = [],
  editMode = false,
  availableBaseModels = [],
  onChangeExtends,
  onImportCode,
  allowExtendsChangeOutsideEdit = true,
  // other props...
  selectedElementId: controlledSelectedElementId,
  setSelectedElementId: controlledSetSelectedElementId,
}) => {
  const [_matchIndex, setMatchIndex] = useState(0);
  const [_matchCount, setMatchCount] = useState(0);
  const { /* searchTerm: ctxTerm, setSearchTerm: ctxSet */ } = useGlobalSearch();
  // Store only the selected element id; always derive the latest element object from the model
  // If parent provides controlled selection, use that instead of local state.
  const [internalSelectedElementId, setInternalSelectedElementId] = useState<string | null>(null);
  const selectedElementId = controlledSelectedElementId ?? internalSelectedElementId;
  const setSelectedElementId = controlledSetSelectedElementId ?? setInternalSelectedElementId;
  // Track model identity to avoid re-selecting on every array change
  const lastModelKeyRef = useRef<string>('');

  // Always mirror canvas selections locally so synthetic elements (like Extends) render immediately
  const handleElementSelect = useCallback((el: unknown) => {
    const rec = el as Record<string, unknown> | null;
    const id = (typeof rec?.id === 'string' ? rec.id : (typeof rec?.name === 'string' ? rec.name : null)) as string | null;
    // Update local selection for immediate rendering within this editor
    setInternalSelectedElementId(id);
    // Also forward to parent when a controlled setter is provided
    try { controlledSetSelectedElementId?.(id); } catch {}
  }, [controlledSetSelectedElementId]);

  // Derive the selected element from the current semanticModel by id so it stays in sync
  const selectedElement = useMemo(() => {
    if (!selectedElementId || !semanticModel) return null;
    const sm = semanticModel as unknown as {
      dimensions?: unknown[];
      measures?: unknown[];
      filters?: unknown[];
      joins?: unknown[];
      pre_aggregations?: unknown[];
    } | null;
    const lists: unknown[][] = [sm?.dimensions || [], sm?.measures || [], sm?.filters || [], sm?.joins || [], sm?.pre_aggregations || []];
    for (const arr of lists) {
      const found = arr.find((it) => {
        const r = it as Record<string, unknown> | null;
        if (!r) return false;
        if (typeof r.id === 'string' && r.id === selectedElementId) return true;
        if (typeof r.name === 'string' && r.name === selectedElementId) return true;
        return false;
      });
      if (found) return found;
    }
    // If user selected the synthetic Extends tile, surface a virtual element so the ExtendsForm renders
    if (typeof selectedElementId === 'string' && selectedElementId.startsWith('extends__')) {
      const baseName = (extendsModel || selectedElementId.replace(/^extends__/, '')) as string;
      return { id: selectedElementId, type: 'extends', name: baseName, title: 'Extends' } as unknown as Record<string, unknown>;
    }
    return null;
  }, [semanticModel, selectedElementId, extendsModel]);

  // Draggable palette overlay state/refs
  const wrapperRef = useRef<HTMLDivElement | null>(null);
  const overlayRef = useRef<HTMLDivElement | null>(null);
  const draggingRef = useRef(false);
  const dragOffsetRef = useRef<{ dx: number; dy: number }>({ dx: 0, dy: 0 });
  const [palettePos, setPalettePos] = useState<{ x: number; y: number }>({ x: 8, y: 8 });
  // Persist palette position per model
  const paletteStorageKey = useMemo(() => {
    const selModelRec = selectedModel as Record<string, unknown> | null;
    const id = String(selModelRec?.id ?? selModelRec?.model_key ?? modelName ?? 'default');
    return `semlayer.palette.position.${id}`;
  }, [selectedModel, modelName]);

  // Clamp position within wrapper bounds - allow full edge access
  const clampToWrapper = useCallback((x: number, y: number) => {
    const wrapper = wrapperRef.current;
    const overlay = overlayRef.current;
    if (!wrapper || !overlay) return { x, y };
    const wr = wrapper.getBoundingClientRect();
    const ow = overlay.offsetWidth || 0;
    const oh = overlay.offsetHeight || 0;
    // Allow palette to go to very edges (no margin restriction)
    const maxX = Math.max(0, wr.width - ow);
    const maxY = Math.max(0, wr.height - oh);
    return { x: Math.min(Math.max(0, x), maxX), y: Math.min(Math.max(0, y), maxY) };
  }, []);

  useEffect(() => {
    const onMove = (e: MouseEvent) => {
      if (!draggingRef.current) return;
      const wrap = wrapperRef.current; if (!wrap) return;
      const rect = wrap.getBoundingClientRect();
      const x = e.clientX - rect.left - dragOffsetRef.current.dx;
      const y = e.clientY - rect.top - dragOffsetRef.current.dy;
      const clamped = clampToWrapper(x, y);
      setPalettePos(clamped);
      e.preventDefault();
    };
    const onUp = () => {
      if (!draggingRef.current) return;
      draggingRef.current = false;
      document.body.classList.remove('palette-dragging');
      // Snap to nearest corner and persist
      try {
        const wrap = wrapperRef.current; const overlay = overlayRef.current; if (!wrap || !overlay) return;
        const wr = wrap.getBoundingClientRect();
        const ow = overlay.offsetWidth || 0; const oh = overlay.offsetHeight || 0;
        const maxX = Math.max(0, wr.width - ow);
        const maxY = Math.max(0, wr.height - oh);
        const corners = [
          { x: 8, y: 8 },
          { x: maxX, y: 8 },
          { x: 8, y: maxY },
          { x: maxX, y: maxY },
        ];
        const curr = palettePos;
        let best = corners[0];
        let bestDist = Number.POSITIVE_INFINITY;
        for (const c of corners) {
          const dx = (c.x - curr.x); const dy = (c.y - curr.y);
          const dist = Math.hypot(dx, dy);
          if (dist < bestDist) { bestDist = dist; best = c; }
        }
        const SNAP_THRESHOLD = 120;
        const snapped = bestDist <= SNAP_THRESHOLD ? best : clampToWrapper(curr.x, curr.y);
        setPalettePos(snapped);
        try { window.localStorage.setItem(paletteStorageKey, JSON.stringify(snapped)); } catch {}
      } catch {}
    };
    window.addEventListener('mousemove', onMove);
    window.addEventListener('mouseup', onUp);
    window.addEventListener('mouseleave', onUp);
    return () => {
      window.removeEventListener('mousemove', onMove);
      window.removeEventListener('mouseup', onUp);
      window.removeEventListener('mouseleave', onUp);
    };
  }, [clampToWrapper, palettePos, paletteStorageKey]);

  // Re-clamp on resize
  useEffect(() => {
    const onResize = () => setPalettePos((p) => clampToWrapper(p.x, p.y));
    window.addEventListener('resize', onResize);
    return () => window.removeEventListener('resize', onResize);
  }, [clampToWrapper]);

  // Apply CSS variables for top/left to avoid inline style rules violations
  useEffect(() => {
    const el = overlayRef.current;
    if (!el) return;
    try {
      el.style.setProperty('--palette-left', `${palettePos.x}px`);
      el.style.setProperty('--palette-top', `${palettePos.y}px`);
    } catch {}
  }, [palettePos.x, palettePos.y]);

  // Load persisted position for current model
  useEffect(() => {
    try {
      const raw = window.localStorage.getItem(paletteStorageKey);
      if (raw) {
        const pos = JSON.parse(raw);
        if (typeof pos?.x === 'number' && typeof pos?.y === 'number') {
          setPalettePos(clampToWrapper(pos.x, pos.y));
          return;
        }
      }
    } catch {}
    setPalettePos({ x: 8, y: 8 });
  }, [paletteStorageKey, clampToWrapper]);

  const startDrag = (e: ReactMouseEvent<HTMLDivElement>) => {
    const wrap = wrapperRef.current; const overlay = overlayRef.current;
    if (!wrap || !overlay) return;
    const overlayRect = overlay.getBoundingClientRect();
    draggingRef.current = true;
    document.body.classList.add('palette-dragging');
    dragOffsetRef.current = {
      dx: e.clientX - overlayRect.left,
      dy: e.clientY - overlayRect.top,
    };
    // Ensure cursor doesn't select text while dragging
    e.preventDefault();
  };

  // Navigation events are dispatched directly by UI handlers elsewhere.

  // Focus the canvas when the selected model changes (core or custom)
  useEffect(() => {
    try {
      window.dispatchEvent(new CustomEvent('semlayer.focusCanvas'));
    } catch {
      // no-op
    }
  }, [selectedModel?.model_key, selectedModel?.name]);

  // Auto-select the first item only when a model is first loaded or when no selection exists
  useEffect(() => {
    if (!semanticModel) return;
  // If parent controls selection, do not auto-select
  if (controlledSelectedElementId !== undefined) return;
  // Compute a stable key for the current model context
  const selModelRec = selectedModel as Record<string, unknown> | null;
  const modelKey = String(selModelRec?.id ?? selModelRec?.model_key ?? semanticModel?.name ?? 'default');
    const isModelChanged = lastModelKeyRef.current !== modelKey;
    if (isModelChanged) {
      lastModelKeyRef.current = modelKey;
      // Clear previous selection when switching models; selection will be re-initialized below
      setSelectedElementId(null);
    }
    // Don't override user selection
    if (selectedElementId) return;
    try {
  const smRec = semanticModel as unknown as Record<string, unknown> | null;
  const dims = Array.isArray(smRec?.dimensions) ? (smRec!.dimensions as unknown[]) : [];
  const meas = Array.isArray(smRec?.measures) ? (smRec!.measures as unknown[]) : [];
  const filts = Array.isArray(smRec?.filters) ? (smRec!.filters as unknown[]) : [];
  const joins = Array.isArray(smRec?.joins) ? (smRec!.joins as unknown[]) : [];
  const preAggs = Array.isArray(smRec?.pre_aggregations) ? (smRec!.pre_aggregations as unknown[]) : [];
  const first: unknown = dims[0] || meas[0] || filts[0] || joins[0] || preAggs[0] || null;
  const firstRec = first as Record<string, unknown> | null;
  const firstId = typeof firstRec?.id === 'string' ? firstRec.id : null;
  if (firstId) setSelectedElementId(firstId);
    } catch {}
  }, [semanticModel, selectedModel, selectedElementId, controlledSelectedElementId]);

  // When entering edit mode, auto-select first tile if none is selected and focus form section
  useEffect(() => {
    if (!editMode) return;
  // If parent controls selection, do not auto-select
  if (controlledSelectedElementId !== undefined) return;
    try {
      if (!selectedElementId) {
        const dims = (semanticModel as any)?.dimensions || [];
        const meas = (semanticModel as any)?.measures || [];
        const filts = (semanticModel as any)?.filters || [];
        const joins = ((semanticModel as any)?.joins || []);
        const preAggs = (semanticModel as any)?.pre_aggregations || [];
        const first: any = dims[0] || meas[0] || filts[0] || joins[0] || preAggs[0] || null;
        if (first && first.id) setSelectedElementId(first.id);
      }
      // Focus the form panel to signal edit context
      const form = document.querySelector('.form-section') as HTMLElement | null;
      if (form) form.focus({ preventScroll: true } as any);
    } catch {}
  }, [editMode, selectedElementId, semanticModel, controlledSelectedElementId]);

  // Debug: log generated code and tile counts when selection changes
  useEffect(() => {
    if (!selectedModel) return;
    try {
      const key = (selectedModel as any)?.model_key || (selectedModel as any)?.name || 'unknown';
      const isCustom = Boolean((selectedModel as any)?.is_custom);
      const json = isCustom
        ? (generateCustomJSON ? generateCustomJSON() : generateJSON())
        : generateJSON();
      const yaml = isCustom
        ? (generateCustomYAML ? generateCustomYAML() : generateYAML())
        : generateYAML();
  // Print full code for troubleshooting via dev logger
  devLog('[SemanticModelEditor] Selected model:', { key, isCustom, extendsModel });
  devLog('[SemanticModelEditor] Generated JSON for model', key, '\n', json);
  devLog('[SemanticModelEditor] Generated YAML for model', key, '\n', yaml);
    } catch (e) {
      devWarn('[SemanticModelEditor] Failed to generate code for selected model', e);
    }
    try {
      const d = (semanticModel?.dimensions || []).length;
      const m = (semanticModel?.measures || []).length;
      const f = (semanticModel?.filters || []).length;
      const j = ((semanticModel?.joins || [])).length;
  devLog('[SemanticModelEditor] Tile counts => dimensions:', d, 'measures:', m, 'filters:', f, 'joins:', j);
    } catch {}
  }, [selectedModel?.model_key, selectedModel?.name]);
  const selRec = selectedElement as Record<string, unknown> | null;

  return (
  <div className={"semantic-model-editor " + (!editMode ? 'no-palette ' : '') + (showCode ? 'code-mode' : '') + (editMode ? 'edit-mode' : '')}>
      {/* Palette, Canvas, Form, and Code Panels go here */}

      {!showCode && (
        <div className="canvas-overlay-wrapper">
          <div className="canvas-form-container">
            <div className="canvas-section" ref={wrapperRef}>
              {editMode && selectedModel?.is_custom && (
                <div
                  className="palette-overlay has-position"
                  role="region"
                  aria-label="Modeling palette"
                  ref={overlayRef}
                  data-left={palettePos.x}
                  data-top={palettePos.y}
                >
                  <div
                    className="palette-drag-handle"
                    role="button"
                    aria-label="Move palette"
                    title="Drag to move palette"
                    tabIndex={0}
                    onMouseDown={startDrag}
                  />
                  <SemanticPalette
                    onAdd={(type) => onAdd && onAdd(type, selectedColumn ? { id: selectedColumn.nodeId, qualified_path: selectedColumn.tableName } : null)}
                    horizontal={true}
                    enableDrag={false}
                    // Prevent adding an Extends tile when one already exists on the canvas
                    canAddExtends={!!(editMode && selectedModel?.is_custom && !extendsModel)}
                  />
                </div>
              )}
              <CanvasPanel
                semanticModel={semanticModel as SemanticModel}
                coreOptions={coreOptions}
                extendsModel={extendsModel}
                editMode={editMode}
                removeSemanticElement={removeSemanticElement || (()=>{})}
                toggleElementEdit={toggleElementEdit || (()=>{})}
                updateSemanticElement={updateSemanticElement || (()=>{})}
                onElementSelect={handleElementSelect}
                allModelKeys={[
                  ...((availableBaseModels || []).map(b => b.key).filter(Boolean)),
                  ...([selectedModel?.model_key].filter(Boolean) as string[])
                ]}
              />
            </div>
            <div className="form-section" tabIndex={-1}>
              <div className="form-section-header">
                <h3>Element Details</h3>
                <span className={`edit-mode-flag ${editMode ? 'editing' : 'readonly'}`} aria-label={editMode ? 'Editing' : 'Read only'}>
                  {editMode ? 'Editing' : 'Read only'}
                </span>
              </div>
              <div className="form-section-content">
                {selectedElement ? (
                  selRec?.type === 'extends' ? (
                    <ExtendsForm
                      currentBase={extendsModel}
                      options={availableBaseModels}
                      disabled={!allowExtendsChangeOutsideEdit && !editMode}
                          onChange={(val) => {
                            try { devLog('[SemanticModelEditor] onChange called, forwarding to onChangeExtends', { val, hasHandler: !!onChangeExtends }); } catch {}
                            try { onChangeExtends?.(val); } catch {}
                          }}
                    />
                  ) : (
                    (() => {
                      // Robustly infer the editor type for the selected element
                      const rawType = String((selRec?.type as string) || '').toLowerCase();
                      const isMeasure = ['sum','count','avg','average','min','max'].includes(rawType) || Boolean(selRec?.aggregationType);
                      const isJoin = Boolean(selRec?.joinType || selRec?.relationship);
                      const isDimension = Boolean(selRec?.sourceColumn) || ['string','number','boolean','time','date'].includes(rawType);
                      const inferredType: 'dimension'|'measure'|'filter'|'join' = isJoin ? 'join' : (isMeasure ? 'measure' : (isDimension ? 'dimension' : 'filter'));
                      const isCoreFlag = selectedModel?.is_custom
                        ? (selRec?.is_custom === false)
                        : !(selRec?.is_custom === true);

                      const toPlural = (t: 'dimension'|'measure'|'filter'|'join'): 'dimensions'|'measures'|'filters'|'joins' => (
                        t === 'dimension' ? 'dimensions' : t === 'measure' ? 'measures' : t === 'filter' ? 'filters' : 'joins'
                      );

                      return (
                        <EnhancedTileForm
                          element={selectedElement}
                          type={inferredType}
                          isCore={isCoreFlag}
                          isOverride={false}
                          isNew={false}
                          coreOptions={coreOptions}
                          modelName={semanticModel.name}
                          onUpdate={(updates) => {
                            const elementType = inferredType;
                            updateSemanticElement?.(toPlural(elementType), (selRec?.id as string) || '', updates);
                          }}
                          // Keep selection after save/cancel/close to avoid "snapping back"
                          onSave={() => { /* no-op: keep selection */ }}
                          onCancel={() => { /* no-op */ }}
                          onClose={() => { /* no-op */ }}
                          readOnly={!editMode}
                        />
                      );
                    })()
                  )
                ) : extendsModel ? (
                  <div className="extends-info">
                    <h4>Extends: {extendsModel}</h4>
                    <p>This model extends the base model "{extendsModel}". The extended model provides additional customizations on top of the base functionality.</p>
                  </div>
                ) : (
                  <div className="form-placeholder">
                    Select an element to view and edit its details here.
                  </div>
                )}
              </div>
            </div>
          </div>
        </div>
      )}
      {showCode && (
        <>
          <CodePanel
            showCode={showCode}
            modelName={modelName}
            onToggleFormat={onToggleCode}
            generateJSON={generateJSON}
            generateYAML={generateYAML}
            generateCustomJSON={generateCustomJSON}
            generateCustomYAML={generateCustomYAML}
            generateCoreJSON={generateCoreJSON}
            generateCoreYAML={generateCoreYAML}
            generateMergedModelObject={generateMergedModelObject}
            codeEditable={codeEditable}
            extendsModel={extendsModel}
            semanticModel={semanticModel}
            selectedModel={selectedModel}
            setMatchIndex={setMatchIndex}
            setMatchCount={setMatchCount}
            onImportCode={onImportCode}
          />
        </>
      )}
    </div>
  );
};

export default SemanticModelEditor;
