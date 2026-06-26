// Install these packages first:
// npm install @dnd-kit/core @dnd-kit/sortable @dnd-kit/utilities lucide-react

import React, { useState, useEffect } from 'react';
import { devDebug, devWarn, devError } from '../utils/devLogger';
import { ToastProvider, useToast } from '../components/Toast';
import ErrorSummary from '../components/ui/ErrorSummary';
import ConfirmModal from '../components/ui/ConfirmModal';
import SlideOver from '../components/ui/SlideOver';
import { fetchSavedLayoutsApi, saveLayoutApi, loadLayoutApi, deleteLayoutApi } from './layoutsApi';
import {
  DndContext,
  DragOverlay,
  closestCenter,
  KeyboardSensor,
  PointerSensor,
  useSensor,
  useSensors,
  DragStartEvent,
  DragEndEvent,
  DragOverEvent,
  UniqueIdentifier,
} from '@dnd-kit/core';
import {
  arrayMove,
  SortableContext,
  sortableKeyboardCoordinates,
  useSortable,
  verticalListSortingStrategy,
} from '@dnd-kit/sortable';
import { CSS } from '@dnd-kit/utilities';
import { 
  Plus, Trash2, Settings, Eye, Save, Grid, List, Link2, 
  ChevronDown, ChevronRight, GripVertical, Copy, Move 
} from 'lucide-react';
import styles from './DynamicUIGeneratorPage.module.css';

// Use canonical types from the project's Northwind definitions and adapt them
// into a small UI-friendly shape used by this page.
import {
  BusinessObjectDefinition,
  FieldDefinition,
  getNorthwindBOs,
} from '../types/northwind';

// UI-friendly (local) types used by this page. We map the canonical types into
// this shape so the rest of the component logic (which expects `id`, `label`)
// stays unchanged.
type BORelationship = {
  id: string;
  name: string;
  relatedBO: string;
  relationshipType: 'one-to-one' | 'one-to-many' | 'many-to-one' | 'many-to-many';
  foreignKey: string;
  label: string;
};

type BOField = {
  id: string; // maps to FieldDefinition.key
  name: string; // technical name
  label: string; // displayName
  type: 'string' | 'number' | 'date' | 'boolean' | 'reference' | 'picklist';
  required: boolean;
};

type BusinessObject = {
  id: string;
  name: string;
  tableName: string;
  fields?: BOField[]; // deprecated - keep for backward compatibility
  coreFields?: BOField[]; // normalized fields (core attributes)
  customFields?: BOField[]; // normalized fields (custom attributes)
  relationships: BORelationship[]; // Northwind types do not include relationships; keep empty
};

interface LayoutSection {
  id: string;
  title: string;
  type: 'fields' | 'related_list' | 'custom';
  columns: 1 | 2 | 3 | 4;
  collapsible: boolean;
  fieldIds?: string[];
  relatedBO?: string;
  relationshipId?: string;
}

interface PageLayout {
  id: string;
  name: string;
  primaryBO: string;
  layoutType: 'detail' | 'form' | 'list';
  sections: LayoutSection[];
}

// Build UI-friendly BOs from the canonical Northwind registry. Northwind's
// BusinessObjectDefinition splits coreFields/customFields and does not encode
// relationships, so we combine fields and leave relationships empty for now.
function mapFieldDefinitionToUI(f: FieldDefinition): BOField {
  // Map the canonical field types into our UI-friendly set.
  const mapType = (t: FieldDefinition['type']): BOField['type'] => {
    switch (t) {
      case 'number':
      case 'currency':
        return 'number';
      case 'date':
      case 'datetime':
        return 'date';
      case 'boolean':
        return 'boolean';
      case 'reference':
        return 'reference';
      case 'text':
      case 'email':
      case 'json':
      case 'array':
      case 'image':
      default:
        return 'string';
    }
  };

  return {
    id: f.key,
    name: f.technicalName || f.name,
    label: f.displayName || f.name,
    type: mapType(f.type),
    required: !!f.required,
  };
}

function mapBODefinitionToUI(b: BusinessObjectDefinition): BusinessObject {
  const fields = [...(b.coreFields || []), ...(b.customFields || [])].map(mapFieldDefinitionToUI);
  return {
    id: `bo_${b.key}`,
    name: b.displayName || b.name,
    tableName: b.technicalName,
    fields,
    relationships: [] as BORelationship[], // not present in canonical model
  };
}

const BUSINESS_OBJECTS: BusinessObject[] = getNorthwindBOs().map(mapBODefinitionToUI);

// ==================== DRAGGABLE FIELD CARD (NEW) ====================

const DraggableFieldCard: React.FC<{ field: BOField; isDragging?: boolean }> = ({ field, isDragging }) => {
  const { attributes, listeners, setNodeRef, transform, transition } = useSortable({
    id: `palette-${field.id}`,
    data: { type: 'field', field }
  });

  const localRef = React.useRef<HTMLDivElement | null>(null);
  const setCombinedRef = (node: HTMLDivElement | null) => {
    localRef.current = node;
    setNodeRef(node);
  };

  React.useEffect(() => {
    const el = localRef.current;
    if (!el) return;
    el.style.transform = CSS.Transform.toString(transform) || '';
    if (transition) el.style.transition = transition as string;
    el.style.opacity = isDragging ? '0.5' : '1';
  }, [transform, transition, isDragging]);

  return (
    <div
      ref={setCombinedRef}
      className={styles.fieldCard}
      {...attributes}
      {...listeners}
    >
      <GripVertical size={16} className={styles.gripIcon} />
      <div className="meta">
        <div className="label">{field.label}</div>
        <div className="sub">{field.type}</div>
      </div>
      {field.required && <span className={styles.requiredStar}>*</span>}
    </div>
  );
};

// ==================== ENHANCED FIELD PALETTE WITH DND ====================

const FieldPalette: React.FC<{
  fields: BOField[];
  selectedFieldIds: string[];
  onFieldsChange: (fieldIds: string[]) => void;
}> = ({ fields, selectedFieldIds, onFieldsChange: _onFieldsChange }) => {
  const availableFields = fields.filter((f: BOField) => !selectedFieldIds.includes(f.id));

  return (
  <div className={styles.palette}>
      <div className={styles.paletteHeader}>
        <GripVertical size={14} />
        Drag Fields Here
      </div>

      <SortableContext items={availableFields.map(f => `palette-${f.id}`)} strategy={verticalListSortingStrategy}>
        {availableFields.map((field: BOField) => (
            <DraggableFieldCard key={field.id} field={field} />
          ))}
      </SortableContext>

      {availableFields.length === 0 && (
          <div className={styles.emptyPalette}>
          <p className={`${styles.smallMuted} ${styles.noMargin}`}>All fields added to sections</p>
        </div>
      )}
    </div>
  );
};

// ==================== DROP ZONE INDICATOR (NEW) ====================

const _DropZoneIndicator: React.FC<{ isActive: boolean }> = ({ isActive }) => (
  <div className={`${styles.dropZone} ${isActive ? styles.dropZoneActive : ''}`} />
);

// ==================== ENHANCED SECTION CONFIGURATOR WITH DND ====================

const SectionConfigurator: React.FC<{
  section: LayoutSection;
  primaryBO: BusinessObject;
  onUpdate: (section: LayoutSection) => void;
  onDelete: () => void;
  isOver?: boolean;
  onOpenEditor?: (section: LayoutSection) => void;
}> = ({ section, primaryBO, onUpdate, onDelete, isOver, onOpenEditor }) => {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({ 
    id: section.id,
    data: { type: 'section', section }
  });
  const [isExpanded, setIsExpanded] = useState(true);

  const _style = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.5 : 1,
  };

  const allBoFields = primaryBO ? [...(primaryBO.coreFields || []), ...(primaryBO.customFields || [])] : [];
  const getFieldById = (fieldId: string) => allBoFields.find((f: BOField) => f.id === fieldId);

  const localRef = React.useRef<HTMLDivElement | null>(null);
  const setCombinedRef = (node: HTMLDivElement | null) => {
    localRef.current = node;
    setNodeRef(node);
  };

  React.useEffect(() => {
    const el = localRef.current;
    if (!el) return;
    el.style.transform = CSS.Transform.toString(transform) || '';
    if (transition) el.style.transition = transition as string;
    el.style.opacity = isDragging ? '0.5' : '1';
  }, [transform, transition, isDragging]);

  return (
    <div
      ref={setCombinedRef}
      className={`${styles.sectionCard} ${isOver ? styles.sectionCardActive : ''}`}
    >
      <div className={styles.sectionHeader}>
        <div className={styles.sectionHeaderLeft}>
          <button 
            onClick={() => setIsExpanded(!isExpanded)} 
            aria-label={isExpanded ? 'Collapse section' : 'Expand section'}
            className={styles.iconButton}
          >
            {isExpanded ? <ChevronDown size={20} /> : <ChevronRight size={20} />}
          </button>
          
          {/* Drag Handle */}
          <div {...attributes} {...listeners} className={styles.grabHandle}>
            <GripVertical size={20} className={styles.gripIcon} />
          </div>

          <input
            id={`section-title-${section.id}`}
            type="text"
            value={section.title}
            onChange={(e) => onUpdate({ ...section, title: e.target.value })}
            className={styles.sectionTitleInput}
            placeholder="Section Title"
          />
        </div>
  <div className={styles.controls}>
            <select
                aria-label={`Section type for ${section.title}`}
                value={section.type}
                onChange={(e) => onUpdate({ ...section, type: e.target.value as any })}
                className={styles.select}
              >
            <option value="fields">Fields</option>
            <option value="related_list">Related List</option>
            <option value="custom">Custom</option>
          </select>
          <button onClick={onDelete} aria-label={`Delete section ${section.title}`} className={styles.btnDanger}>
            <Trash2 size={16} />
          </button>
          {onOpenEditor && (
            <button onClick={() => onOpenEditor(section)} aria-label={`Edit section ${section.title}`} className={styles.iconButton} title="Edit section">
              <Settings size={16} />
            </button>
          )}
        </div>
      </div>

      {isExpanded && (
          <div className={`${styles.flexCol} ${styles.indentLarge}`}>
          {section.type === 'fields' && (
            <>
              {/* Column Selector */}
                  <div>
                    <label className={styles.label}>
                      Number of Columns
                    </label>
                  <div className={`${styles.flex} ${styles.gapHalf}`}>
                  {([1, 2, 3, 4] as Array<1 | 2 | 3 | 4>).map((col: 1 | 2 | 3 | 4) => (
                    <button
                      key={col}
                      onClick={() => onUpdate({ ...section, columns: col })}
                        className={`${styles.colButton} ${section.columns === col ? styles.colButtonActive : ''}`}
                    >
                      {col} Column{col > 1 ? 's' : ''}
                    </button>
                  ))}
                </div>
              </div>

              {/* Selected Fields List with Drag-and-Drop Reordering */}
              <div>
                <label className={styles.label}>
                  Selected Fields ({section.fieldIds?.length || 0})
                  <span className={styles.smallMutedNote}>
                    Drag to reorder
                  </span>
                </label>
                
                {section.fieldIds && section.fieldIds.length > 0 ? (
                  <div className={styles.dashedArea}>
                    <SortableContext items={section.fieldIds.map(id => `field-${section.id}-${id}`)} strategy={verticalListSortingStrategy}>
                      {section.fieldIds.map((fieldId, _index) => {
                        const field = getFieldById(fieldId);
                        if (!field) return null;

                        return (
                          <SortableFieldItem
                            key={fieldId}
                            id={`field-${section.id}-${fieldId}`}
                            field={field}
                            onRemove={() => {
                              const updated = section.fieldIds!.filter(id => id !== fieldId);
                              onUpdate({ ...section, fieldIds: updated });
                            }}
                          />
                        );
                      })}
                    </SortableContext>
                  </div>
                ) : (
                  <div className={`${styles.dashedArea} ${styles.dashedAreaEmpty}`}>
                    <Grid size={32} className={styles.emptyPaletteIcon} />
                    <p className={`${styles.smallMuted} ${styles.noMargin}`}>Drag fields here from palette above</p>
                  </div>
                )}
              </div>

              {/* Field Palette */}
              <FieldPalette
                fields={allBoFields}
                selectedFieldIds={section.fieldIds || []}
                onFieldsChange={(fieldIds: string[]) => onUpdate({ ...section, fieldIds })}
              />
            </>
          )}

          {/* Section Type: Related List */}
          {section.type === 'related_list' && (
            <>
              <div>
                <label className={styles.label} htmlFor={`rel-select-${section.id}`}>
                  Select Related Business Object
                </label>
                <select
                  id={`rel-select-${section.id}`}
                  aria-label={`Related business object for ${section.title}`}
                  value={section.relationshipId || ''}
                  onChange={(e) => {
                    const rel = primaryBO.relationships.find(r => r.id === e.target.value);
                    onUpdate({
                      ...section,
                      relationshipId: e.target.value,
                      relatedBO: rel?.relatedBO
                    });
                  }}
                  className={styles.select}
                >
                  <option value="">Select relationship...</option>
                  {primaryBO.relationships.map(rel => (
                    <option key={rel.id} value={rel.id}>
                      {rel.label} ({rel.relatedBO}) - {rel.relationshipType}
                    </option>
                  ))}
                </select>
              </div>

              {section.relatedBO && (
                <div className={styles.infoCard}>
                  <div className={styles.infoTitle}>
                    <Link2 size={16} className={styles.linkIcon} />
                    This will display a list of {section.relatedBO} records
                  </div>
                  <div className={styles.infoSub}>
                    Related via: {primaryBO.relationships.find(r => r.id === section.relationshipId)?.foreignKey}
                  </div>
                </div>
              )}
            </>
          )}

          <label className={styles.checkboxLabel}>
            <input
              type="checkbox"
              checked={!!section.collapsible}
              onChange={(e: React.ChangeEvent<HTMLInputElement>) => onUpdate({ ...section, collapsible: e.target.checked })}
              className={styles.checkboxInput}
            />
            <span className={styles.collapsibleText}>Make this section collapsible</span>
          </label>
        </div>
      )}
    </div>
  );
};

// ==================== SORTABLE FIELD ITEM (NEW) ====================

const SortableFieldItem: React.FC<{
  id: string;
  field: BOField;
  onRemove: () => void;
}> = ({ id, field, onRemove }) => {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({ id });

  const _style = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.5 : 1,
  };

  const localRef = React.useRef<HTMLDivElement | null>(null);
  const setCombinedRef = (node: HTMLDivElement | null) => {
    localRef.current = node;
    setNodeRef(node);
  };

  React.useEffect(() => {
    const el = localRef.current;
    if (!el) return;
    el.style.transform = CSS.Transform.toString(transform) || '';
    if (transition) el.style.transition = transition as string;
    el.style.opacity = isDragging ? '0.5' : '1';
  }, [transform, transition, isDragging]);

  return (
    <div
      ref={setCombinedRef}
      className={styles.sortableFieldItem}
    >
      <div {...attributes} {...listeners} className={styles.grabHandle}>
        <GripVertical size={14} className={styles.gripIcon} />
      </div>
      <span className={styles.fieldLabel}>{field.label}</span>
      <span className={styles.typeBadge}>{field.type}</span>
      <button onClick={onRemove} aria-label={`Remove ${field.label}`} className={styles.iconButtonRed}>
        <Trash2 size={14} />
      </button>
    </div>
  );
};

// Simple slide-over editor for a section
const SectionEditor: React.FC<{
  section: LayoutSection;
  onSave: (s: LayoutSection) => void;
  onCancel: () => void;
}> = ({ section, onSave, onCancel }) => {
  const [draft, setDraft] = useState<LayoutSection>({ ...section });

  return (
    <div>
      <div className={styles.panelCard}>
        <label className={styles.label}>Section Title</label>
  <input placeholder="Section title" className={styles.input} value={draft.title} onChange={(e) => setDraft({ ...draft, title: e.target.value })} />

        <label className={styles.label}>Columns</label>
        <select title="Columns" className={styles.select} value={draft.columns} onChange={(e) => setDraft({ ...draft, columns: Number(e.target.value) as any })}>
          <option value={1}>1</option>
          <option value={2}>2</option>
          <option value={3}>3</option>
          <option value={4}>4</option>
        </select>

        <label className={styles.checkboxLabel}>
          <input type="checkbox" checked={!!draft.collapsible} onChange={(e) => setDraft({ ...draft, collapsible: e.target.checked })} className={styles.checkboxInput} />
          <span className={styles.collapsibleText}>Collapsible</span>
        </label>

        <div className={`${styles.rowCenterGap} ${styles.mtSmall}`}>
          <button onClick={() => { onSave(draft); }} className={styles.btnPrimary}>Save</button>
          <button onClick={onCancel} className={styles.btnSecondary}>Cancel</button>
        </div>
      </div>
    </div>
  );
};

// ==================== MAIN LAYOUT MANAGER WITH ENHANCED DND ====================

const LayoutManager: React.FC = () => {
  const [layout, setLayout] = useState<PageLayout>({
    id: 'layout_1',
    name: 'Customer Detail Page',
    primaryBO: 'bo_customer',
    layoutType: 'detail',
    sections: [{
      id: 'sec_1',
      title: 'Basic Information',
      type: 'fields',
      columns: 2,
      collapsible: false,
      fieldIds: ['f1', 'f2', 'f3']
    }]
  });

  const [showPreview, setShowPreview] = useState(true);
  const [activeId, setActiveId] = useState<UniqueIdentifier | null>(null);
  const [overId, setOverId] = useState<UniqueIdentifier | null>(null);
  const STORAGE_KEY = 'dui_layout_v1';
  const [savedAt, setSavedAt] = useState<number | null>(null);
  const [savedLayouts, setSavedLayouts] = useState<Array<{ id: string; name: string; updated_at?: string }>>([]);
  const [serverAvailable, setServerAvailable] = useState<boolean | null>(null);
  const [saveName, setSaveName] = useState<string>(layout.name || '');
  const [defaultSaveToServer, setDefaultSaveToServer] = useState<boolean>(false);
  const [savedListVisible, _setSavedListVisible] = useState<boolean>(false);
  const [confirmOpen, setConfirmOpen] = useState<boolean>(false);
  const [confirmMessage, setConfirmMessage] = useState<string>('');
  const [confirmHandler, setConfirmHandler] = useState<(() => Promise<void> | void) | null>(null);
  const [_pendingPrimaryBO, setPendingPrimaryBO] = useState<string | null>(null);
  const [errorSummaryOpen, setErrorSummaryOpen] = useState<boolean>(false);
  const [validationErrors, setValidationErrors] = useState<Array<{ fieldId: string; label: string; message: string }>>([]);
  const [slideOverOpen, setSlideOverOpen] = useState<boolean>(false);
  const [editingSection, setEditingSection] = useState<LayoutSection | null>(null);
  const toast = useToast();

  const sensors = useSensors(
    useSensor(PointerSensor, {
      activationConstraint: {
        distance: 8, // 8px movement before drag starts
      },
    }),
    useSensor(KeyboardSensor, {
      coordinateGetter: sortableKeyboardCoordinates,
    })
  );

  const primaryBO = BUSINESS_OBJECTS.find(bo => bo.id === layout.primaryBO)!;

  // Load saved layout metadata on mount (don't auto-load the layout)
  useEffect(() => {
    try {
      const raw = localStorage.getItem(STORAGE_KEY);
      if (raw) {
        const parsed = JSON.parse(raw);
        if (parsed && parsed.savedAt) setSavedAt(parsed.savedAt);
      }
    } catch (e) {
      // ignore
    }
  }, []);

  // Load user's default save target preference
  useEffect(() => {
    try {
      const v = localStorage.getItem('dui_default_save_to_server');
      if (v !== null) setDefaultSaveToServer(v === '1');
    } catch (e) {
      // ignore
    }
  }, []);

  // Helper functions
  function addSection(type: 'fields' | 'related_list') {
    const newSection: LayoutSection = {
      id: `sec_${Date.now()}`,
      title: type === 'fields' ? 'New Section' : 'Related Records',
      type,
      columns: 2,
      collapsible: false,
      fieldIds: type === 'fields' ? [] : undefined,
      relationshipId: undefined
    } as LayoutSection;

    setLayout({ ...layout, sections: [...layout.sections, newSection] });
  }

  function handlePrimaryBOChange(newBO: string) {
    if (newBO === layout.primaryBO) return;
    if (layout.sections && layout.sections.length > 0) {
      // show accessible confirmation modal and defer the change
      setPendingPrimaryBO(newBO);
      setConfirmMessage('Changing the primary Business Object will remove all existing sections. Continue?');
      setConfirmHandler(() => async () => {
        setLayout({ ...layout, primaryBO: newBO, sections: [] });
        setPendingPrimaryBO(null);
        toast.showToast('Primary Business Object changed', 'success');
      });
      setConfirmOpen(true);
      return;
    }
    setLayout({ ...layout, primaryBO: newBO, sections: [] });
  }

  function updateSection(id: string, updated: LayoutSection) {
    setLayout({ ...layout, sections: layout.sections.map(s => s.id === id ? updated : s) });
  }

  function deleteSection(id: string) {
    setLayout({ ...layout, sections: layout.sections.filter(s => s.id !== id) });
  }

  function saveLayout() {
    // persist to localStorage as a quick-save
    try {
      const payload = { layout, savedAt: Date.now() };
      localStorage.setItem(STORAGE_KEY, JSON.stringify(payload));
      setSavedAt(payload.savedAt);
      devDebug('Save layout', layout);
      return true;
    } catch (e) {
      devError('Failed to save layout', e);
      return false;
    }
  }

  // Helper to perform a save according to user preference
  async function performSave() {
    // run validation first
    const errs = validateLayout();
    if (errs.length > 0) {
      setValidationErrors(errs);
      setErrorSummaryOpen(true);
      return false;
    }

    if (defaultSaveToServer) {
      const res = await saveLayoutToServer();
      if (res) return true;
      return false;
    }
    return saveLayout();
  }

  async function fetchSavedLayouts() {
    try {
      const list = await fetchSavedLayoutsApi();
      setSavedLayouts(list);
      setServerAvailable(true);
    } catch (e) {
      devWarn('Failed to fetch saved layouts', e);
      setServerAvailable(false);
    }
  }

  async function saveLayoutToServer(id?: string) {
    try {
      const body = { id: id || undefined, name: saveName || layout.name || 'Untitled', layout };
      const data = await saveLayoutApi(body);
      await fetchSavedLayouts();
      toast.showToast('Layout saved to server', 'success');
      return data;
    } catch (e) {
      devError('Failed to save layout to server', e);
      toast.showToast('Failed to save to server, saved locally instead', 'error');
      saveLayout();
      return null;
    }
  }

  async function loadLayoutFromServer(id: string) {
    try {
      const data = await loadLayoutApi(id);
      if (data && data.layout) {
        setLayout(data.layout as PageLayout);
        setSavedAt(Date.now());
        toast.showToast('Loaded layout from server', 'success');
      }
    } catch (e) {
      devError('Failed to load layout from server', e);
      toast.showToast('Failed to load layout from server', 'error');
    }
  }

  async function deleteSavedLayout(id: string) {
    // show accessible confirmation modal before deleting
    setConfirmMessage('Delete saved layout? This action cannot be undone.');
    setConfirmHandler(() => async () => {
      try {
        await deleteLayoutApi(id);
        await fetchSavedLayouts();
        toast.showToast('Deleted saved layout', 'success');
      } catch (e) {
        devError('Failed to delete layout', e);
        toast.showToast('Failed to delete layout', 'error');
      }
    });
    setConfirmOpen(true);
  }

  // Lazy-load saved layouts only when user opens the saved list
  useEffect(() => {
    if (savedListVisible) fetchSavedLayouts();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [savedListVisible]);

  function _exportLayout() {
    const json = JSON.stringify(layout, null, 2);
    const blob = new Blob([json], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `${layout.name || 'layout'}.json`;
    a.click();
    URL.revokeObjectURL(url);
  }

  function loadSavedLayout() {
    try {
      const raw = localStorage.getItem(STORAGE_KEY);
      if (!raw) {
        toast.showToast('No saved layout found locally', 'error');
        return;
      }
      const parsed = JSON.parse(raw);
      if (parsed && parsed.layout) {
        setLayout(parsed.layout);
        setSavedAt(parsed.savedAt || Date.now());
        toast.showToast('Loaded saved layout', 'success');
      }
    } catch (e) {
      devError('Failed to load saved layout', e);
      toast.showToast('Failed to load saved layout', 'error');
    }
  }

  function clearSavedLayout() {
    localStorage.removeItem(STORAGE_KEY);
    setSavedAt(null);
    toast.showToast('Local saved layout cleared', 'success');
  }

  // Validation / Error summary helpers
  function validateLayout(): Array<{ fieldId: string; label: string; message: string }> {
    const errs: Array<{ fieldId: string; label: string; message: string }> = [];
    if (!layout.name || layout.name.trim() === '') {
      errs.push({ fieldId: 'layout-name', label: 'Layout name', message: 'Required' });
    }
    layout.sections.forEach(s => {
      if (s.type === 'fields') {
        if (!s.fieldIds || s.fieldIds.length === 0) {
          errs.push({ fieldId: `section-${s.id}`, label: `Section: ${s.title || 'Untitled'}`, message: 'No fields selected' });
        }
      }
    });
    return errs;
  }

  function jumpToField(fieldId: string) {
    try {
      if (fieldId === 'layout-name') {
        const el = document.getElementById('layout-name-input') as HTMLElement | null;
        el?.focus();
        return;
      }
      if (fieldId.startsWith('section-')) {
        const id = fieldId.replace('section-', '');
        const el = document.getElementById(`section-title-${id}`) as HTMLElement | null;
        el?.focus();
      }
    } catch (e) {
      // ignore
    }
  }

  function handleDragStart(event: DragStartEvent) {
    setActiveId(event.active.id);
  }

  function handleDragOver(event: DragOverEvent) {
    setOverId(event.over?.id || null);
  }

  function handleDragEnd(event: DragEndEvent) {
    const { active, over } = event;

    if (!over) {
      setActiveId(null);
      setOverId(null);
      return;
    }

    const activeData = active.data.current;
    const overData = over.data.current;

    // Case 1: Reordering sections
    if (activeData?.type === 'section' && overData?.type === 'section') {
      const oldIndex = layout.sections.findIndex(s => s.id === active.id);
      const newIndex = layout.sections.findIndex(s => s.id === over.id);

      if (oldIndex !== newIndex) {
        setLayout({
          ...layout,
          sections: arrayMove(layout.sections, oldIndex, newIndex),
        });
      }
    }

    // Case 2: Dragging field from palette to section
    if (activeData?.type === 'field' && String(over.id).startsWith('sec_')) {
      const targetSection = layout.sections.find(s => s.id === over.id);
      if (targetSection && targetSection.type === 'fields') {
        const field = activeData.field as BOField;
        const updatedFieldIds = [...(targetSection.fieldIds || []), field.id];
        
        setLayout({
          ...layout,
          sections: layout.sections.map(s =>
            s.id === targetSection.id ? { ...s, fieldIds: updatedFieldIds } : s
          )
        });
      }
    }

    // Case 3: Reordering fields within a section
    if (String(active.id).startsWith('field-') && String(over.id).startsWith('field-')) {
      const [, sectionId] = String(active.id).split('-');
      const section = layout.sections.find(s => s.id === sectionId);
      
      if (section && section.fieldIds) {
        const activeFieldId = String(active.id).split('-')[2];
        const overFieldId = String(over.id).split('-')[2];
        
        const oldIndex = section.fieldIds.indexOf(activeFieldId);
        const newIndex = section.fieldIds.indexOf(overFieldId);

        if (oldIndex !== newIndex) {
          const reorderedFields = arrayMove(section.fieldIds, oldIndex, newIndex);
          
          setLayout({
            ...layout,
            sections: layout.sections.map(s =>
              s.id === section.id ? { ...s, fieldIds: reorderedFields } : s
            )
          });
        }
      }
    }

    setActiveId(null);
    setOverId(null);
  }

  const activeSection = activeId ? layout.sections.find(s => s.id === activeId) : null;

  // Simple preview component defined inline to avoid extra imports
  const LayoutPreview: React.FC<{ layout: PageLayout; primaryBO: BusinessObject }> = ({ layout: p, primaryBO: bo }) => {
    return (
      <div className={styles.previewBox}>
        <h4 className={styles.previewTitle}>{p.name}</h4>
        <p className={styles.previewSubtitle}>{bo.name} ({bo.id})</p>
        <div className={styles.previewList}>
          {p.sections.map(section => (
            <div key={section.id} className={styles.previewSection}>
              <div className={styles.bold600}>{section.title}</div>
              <div className={styles.previewFieldWrapper}>
                {(section.fieldIds || []).map(fid => {
                  const boAllFields = [...(bo.coreFields || []), ...(bo.customFields || bo.fields || [])];
                  const f = boAllFields.find(x => x.id === fid);
                  return f ? <div key={fid} className={styles.previewFieldTag}>{f.label}</div> : null;
                })}
              </div>
            </div>
          ))}
        </div>
      </div>
    );
  };

  // Calculate all fields for display
  const displayAllBoFields = primaryBO ? [...(primaryBO.coreFields || []), ...(primaryBO.customFields || primaryBO.fields || [])] : [];

  return (
    <DndContext
      sensors={sensors}
      collisionDetection={closestCenter}
      onDragStart={handleDragStart}
      onDragOver={handleDragOver}
      onDragEnd={handleDragEnd}
    >
      <div className={styles.page}>
        <div className={styles.container}>
          {/* Header */}
          <div className={styles.headerCard}>
            <div className={styles.headerInner}>
              <div>
                <h1 className={styles.headerTitle}>
                  Page Layout Manager
                </h1>
                <p className={`${styles.smallMuted} ${styles.headerSubtitleWhite}`}>
                  Design pages with related business objects - Workday-style
                </p>
              </div>
              <div className={styles.headerActions}>
                <button onClick={() => setShowPreview(!showPreview)} className={styles.btnSecondary}>
                  <Eye size={20} />
                  {showPreview ? 'Hide' : 'Show'} Preview
                </button>
                <button onClick={async () => {
                  const ok = await performSave();
                  if (ok) toast.showToast('Layout saved', 'success');
                  else toast.showToast('Save failed', 'error');
                }} className={styles.btnPrimary}>
                  <Save size={20} />
                  Save Layout
                </button>
                <button onClick={() => loadSavedLayout()} className={styles.btnSecondary} title="Load last saved layout">
                  <Copy size={18} />
                  Load Last
                </button>
                <button onClick={() => { clearSavedLayout(); }} className={styles.btnSecondary} title="Clear saved layout">
                  <Trash2 size={16} />
                </button>
              </div>
            </div>
          </div>

          <div className={`${styles.gridCols} ${showPreview ? styles.gridWithPreview : ''}`}>
            <div>
              {/* Layout Settings */}
              <div className={styles.panelCard}>
                <h3 className={styles.sectionListTitle}>Layout Settings</h3>
                <div className={styles.threeColGrid}>
                  <div>
                    <label className={styles.label}>Layout Name</label>
                    <input
                      id="layout-name-input"
                      type="text"
                      value={layout.name}
                      onChange={(e) => setLayout({ ...layout, name: e.target.value })}
                      className={styles.input}
                      placeholder="Layout name"
                    />
                  </div>

                  <div>
                    <label className={styles.label}>Primary Business Object</label>
                    <select
                      aria-label="Primary Business Object"
                      title="Primary Business Object"
                      value={layout.primaryBO}
                      onChange={(e) => handlePrimaryBOChange(e.target.value)}
                      className={styles.select}
                    >
                      {BUSINESS_OBJECTS.map(bo => (
                        <option key={bo.id} value={bo.id}>{bo.name}</option>
                      ))}
                    </select>
                  </div>

                  <div>
                    <label className={styles.label}>Layout Type</label>
                    <select
                      aria-label="Layout Type"
                      title="Layout Type"
                      value={layout.layoutType}
                      onChange={(e) => setLayout({ ...layout, layoutType: e.target.value as any })}
                      className={styles.select}
                    >
                      <option value="detail">Detail Page</option>
                      <option value="form">Form</option>
                      <option value="list">List View</option>
                    </select>
                  </div>
                  <div>
                    <label className={styles.label}>Default Save Target</label>
                    <div className={styles.rowCenterGap}>
                      <label className={`${styles.checkboxLabel} ${styles.noMargin}`}>
                        <input
                          type="checkbox"
                          checked={defaultSaveToServer}
                          onChange={(e) => {
                            const v = e.target.checked;
                            setDefaultSaveToServer(v);
                            try { localStorage.setItem('dui_default_save_to_server', v ? '1' : '0'); } catch (err) {}
                          }}
                          className={styles.checkboxInput}
                        />
                        <span className={styles.collapsibleText}>Save to server by default</span>
                      </label>
                    </div>
                  </div>
                </div>
                {savedAt && (
                  <div className={styles.savedNote}>
                    Last saved: {new Date(savedAt).toLocaleString()}
                  </div>
                )}

                {/* Saved layouts server UI */}
                <div className={styles.savedServerPanel}>
                  <label className={styles.label}>Save Name</label>
                  <input type="text" value={saveName} onChange={(e) => setSaveName(e.target.value)} className={styles.input} placeholder="Name this saved layout" />
                  <div className={`${styles.rowCenterGap} ${styles.mtSmall}`}>
                    <button onClick={() => saveLayoutToServer()} className={styles.btnPrimary}><Save size={16} /> Save to Server</button>
                    <button onClick={async () => { await fetchSavedLayouts(); toast.showToast('Refreshed saved list', 'success'); }} className={styles.btnSecondary}><Move size={14} /> Refresh</button>
                  </div>

                  <div className={styles.mtSmall}>
                    <h4 className={styles.smallMuted}>Saved Layouts</h4>
                    {serverAvailable === false && <div className={styles.smallMuted}>Server unavailable or tenant not selected</div>}
                    {savedLayouts.length === 0 ? (
                          <div className={styles.smallMuted}>No saved layouts</div>
                        ) : (
                      <div className={styles.savedList}>
                        {savedLayouts.map(s => (
                          <div key={s.id} className={styles.savedListItem}>
                            <div className={styles.savedListName}>{s.name}</div>
                            <div className={styles.savedListActions}>
                              <button onClick={() => loadLayoutFromServer(s.id)} className={styles.iconButton} title="Load"><Copy size={14} /></button>
                              <button onClick={() => saveLayoutToServer(s.id)} className={styles.iconButton} title="Overwrite"><Save size={14} /></button>
                              <button onClick={() => deleteSavedLayout(s.id)} className={styles.iconButtonRed} title="Delete"><Trash2 size={14} /></button>
                            </div>
                          </div>
                        ))}
                      </div>
                    )}
                  </div>
                </div>
              </div>

              {/* BO Info Card */}
              <div className={styles.infoCard}>
                <div className={styles.infoTitle}>
                  Primary BO: {primaryBO?.name}
                </div>
                <div className={styles.infoSub}>
                  {primaryBO ? `${displayAllBoFields.length} fields • ${primaryBO.relationships.length} relationships` : 'No BO selected'}
                </div>
              </div>

              {/* Add Section Buttons */}
              <div className={styles.panelCard}>
                <h3 className={styles.sectionListTitle}>Add Section</h3>
                <div className={styles.twoColGrid}>
                  <button onClick={() => addSection('fields')} className={styles.cardButton}>
                    <div className={styles.rowCenterGap}> 
                      <div className={styles.iconPillBlue}>
                        <Grid size={18} className={styles.iconPrimary} />
                      </div>
                      <div className={styles.bold600}>Field Section</div>
                    </div>
                    <div className={styles.smallMuted}>Display fields from primary BO</div>
                  </button>

                  <button onClick={() => addSection('related_list')} className={styles.cardButton}>
                    <div className={styles.rowCenterGap}>
                      <div className={styles.iconPillPurple}>
                        <List size={18} className={styles.iconPrimary} />
                      </div>
                      <div className={styles.bold600}>Related List</div>
                    </div>
                    <div className={styles.smallMuted}>Show related BO records</div>
                  </button>
                </div>
              </div>

              {/* Sections List */}
              <div>
                <h3 className={styles.sectionListTitle}>
                  Page Sections ({layout.sections.length})
                  <span className={`${styles.smallMuted} ${styles.smallMutedSpacing}`}>
                    Drag to reorder
                  </span>
                </h3>
                
                {layout.sections.length === 0 ? (
                  <div className={styles.emptyState}>
                    <Plus size={48} className={styles.iconMuted} />
                      <p className={styles.emptyTitle}>
                        No sections added yet
                      </p>
                      <p className={`${styles.mutedText} ${styles.emptySubSpacing}`}>
                        Click "Add Section" above to start building
                      </p>
                  </div>
                ) : (
                  <SortableContext items={layout.sections.map(s => s.id)} strategy={verticalListSortingStrategy}>
                    {layout.sections.map(section => (
                      <SectionConfigurator
                        key={section.id}
                        section={section}
                        primaryBO={primaryBO}
                        onUpdate={(updated) => updateSection(section.id, updated)}
                        onDelete={() => deleteSection(section.id)}
                        isOver={overId === section.id}
                        onOpenEditor={(s) => { setEditingSection(s); setSlideOverOpen(true); }}
                      />
                    ))}
                  </SortableContext>
                )}
              </div>
            </div>

            {showPreview && (
              <div className={styles.stickyPreview}>
                <LayoutPreview layout={layout} primaryBO={primaryBO} />
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Confirm modal and ErrorSummary */}
  <ConfirmModal open={confirmOpen} title="Confirm" message={<p>{confirmMessage}</p>} onClose={() => setConfirmOpen(false)} onConfirm={async () => { setConfirmOpen(false); try { await confirmHandler?.(); } catch (e) { devError('Confirm handler error', e); } }} />
      <ErrorSummary open={errorSummaryOpen} onClose={() => setErrorSummaryOpen(false)} errors={validationErrors} title="Validation errors" onJumpToField={(fid) => { setErrorSummaryOpen(false); jumpToField(fid); }} />

      <SlideOver open={slideOverOpen} onClose={() => setSlideOverOpen(false)} title={editingSection ? `Edit: ${editingSection.title}` : 'Edit section'}>
        {editingSection ? (
          <SectionEditor
            section={editingSection}
            onSave={(updated) => {
              updateSection(editingSection.id, updated);
              setSlideOverOpen(false);
              setEditingSection(null);
            }}
            onCancel={() => { setSlideOverOpen(false); setEditingSection(null); }}
          />
        ) : null}
      </SlideOver>

      {/* Drag Overlay */}
      <DragOverlay>
        {activeSection && (
          <div className={styles.dragOverlay}>
            <div className={styles.flexAlignCenterGap}>
                <GripVertical size={20} className={`${styles.linkIcon} ${styles.gripColored}`} />
                <span className={styles.bold600}>{activeSection.title}</span>
              </div>
          </div>
        )}
      </DragOverlay>
    </DndContext>
  );
};

const Wrapped: React.FC = () => (
  <ToastProvider>
    <LayoutManager />
  </ToastProvider>
);

export default Wrapped;
