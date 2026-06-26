import { useState, useMemo, useEffect, useCallback, useRef, type FC, type KeyboardEvent as ReactKeyboardEvent } from 'react';
import * as TablerIcons from '@tabler/icons-react';
import { IconCopy } from '@tabler/icons-react';
import IconEdit from '@tabler/icons-react/dist/esm/icons/IconEdit.mjs';
import IconCube from '@tabler/icons-react/dist/esm/icons/IconCube.mjs';
import IconDotsVertical from '@tabler/icons-react/dist/esm/icons/IconDotsVertical.mjs'; // legacy, may remove
import IconMenu2 from '@tabler/icons-react/dist/esm/icons/IconMenu2.mjs';
import IconPlus from '@tabler/icons-react/dist/esm/icons/IconPlus.mjs';
import IconAlertTriangle from '@tabler/icons-react/dist/esm/icons/IconAlertTriangle.mjs';
import IconCheck from '@tabler/icons-react/dist/esm/icons/IconCheck.mjs';
// Reference imported icons so the compiler doesn't mark them as unused. These
// icons may be referenced dynamically elsewhere (stories/tests/runtime).
void IconEdit; void IconCube; void IconDotsVertical; void IconMenu2; void IconPlus; void IconAlertTriangle; void IconCheck;
// IconX kept via TablerIcons for inline rename cancel; direct import removed
import IconArchive from '@tabler/icons-react/dist/esm/icons/IconArchive.mjs';
import { Tooltip, Chip, Stack, IconButton as _IconButton, Popover } from '@mui/material';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import EditIcon from '@mui/icons-material/Edit';
import ArchiveIcon from '@mui/icons-material/Archive';
// Note: Radix install failed in this environment; using an accessible custom dropdown with focus-trap instead
import { useDebounce as _useDebounce } from 'use-debounce';
import { devLog } from '../../utils/devLogger';
import BuilderTabs from './BuilderTabs';
import type { ModelCatalogNode } from '../../types/model';
import { useGlobalSearch } from '../../contexts/GlobalSearchContext';
// AccessibleActionsMenu previously imported but not used; removed to fix linter warning
import ModelInfoModal from './ModelInfoModal';
import { toast } from '../ui/sonner';
import ConfirmDeleteModal from './ConfirmDeleteModal';
import ProfessionalSearchInput, { SearchSuggestion } from '../common/ProfessionalSearchInput';

const _RenameModal: FC<{open:boolean; initial:string; onCancel:()=>void; onSave:(v:string)=>void;}> = ({ open, initial, onCancel, onSave }) => {
  const [val,setVal]=useState(initial);
  useEffect(()=>{setVal(initial);},[initial]);
  if(!open) return null;
  return (
    <div className="modal-overlay" onClick={onCancel}>
      <div className="modal-content" onClick={e=>e.stopPropagation()}>
        <header className="modal-header"><h3>Rename Model</h3></header>
        <div className="modal-body"><input value={val} onChange={e=>setVal(e.target.value)} placeholder="New name" /></div>
        <footer className="modal-actions">
          <button className="btn" onClick={onCancel}>Cancel</button>
          <button className="btn btn-primary" disabled={!val.trim()} onClick={()=>onSave(val.trim())}>Save</button>
        </footer>
      </div>
    </div>
  );
};
// Reference to avoid TS6133 (may be used by tests/runtime via dynamic access)
void _RenameModal;

interface ModelCatalogSidebarProps {
  models: ModelCatalogNode[];
  // search handled via GlobalSearchContext
  // optional legacy props kept for backward compatibility with stories/tests
  searchTerm?: string;
  setSearchTerm?: (s: string) => void;
  selectedModel: ModelCatalogNode | null;
  setSelectedModel: (model: ModelCatalogNode | null) => void;
  onCreateCustomModel?: (baseModelKey: string) => void;
  onCloneModel?: (baseModelKey: string) => void;
  onDeleteModel?: (modelId: string, isCore?: boolean, modelKey?: string) => void;
  onArchiveModel?: (modelId: string, isCore?: boolean, modelKey?: string) => void;
  onPublishModel?: (modelId: string) => void;
  onDraftModel?: (modelId: string) => void;
  onRenameModel?: (modelId: string, newName: string) => void;
  onJumpToSection?: (modelId: string, section: string) => void;
  /**
   * Called when a model is selected, to show code in both canvas and configuration editor.
   * @param model The selected model
   * @param targetTab Which tab to activate ('core' or 'custom')
   */
  onModelSelect?: (model: ModelCatalogNode, targetTab: 'core' | 'custom') => void;
  /**
   * Called when the active tab changes, to update the configuration editor tab.
   * @param tab The active tab ('core' or 'custom')
   */
  onTabChange?: (tab: 'core' | 'custom') => void;
  /**
   * Controlled active tab - when provided, overrides internal state
   */
  activeTab?: 'core' | 'custom';
  loading?: boolean;
  error?: string;
}

const ModelCatalogSidebar: React.FC<ModelCatalogSidebarProps> = ({
  models,
  // remove searchTerm/setSearchTerm from props
  selectedModel,
  setSelectedModel,
  onCreateCustomModel,
  onCloneModel,
  onDeleteModel,
  onArchiveModel,
  onPublishModel,
  onDraftModel,
  onRenameModel,
  onJumpToSection,
  onModelSelect,
  onTabChange,
  activeTab: controlledActiveTab,
  loading = false,
  error = null,
  /* onEnterEditMode removed */
}) => {
  const { searchTerm: term, setSearchTerm: setTerm } = useGlobalSearch();
  const [activeTab, setActiveTab] = useState<'core' | 'custom'>('core');
  const [statusFilter, setStatusFilter] = useState<'active' | 'published' | 'draft' | 'archived'>('active');
  // local status overrides for instant UI feedback (id -> status)
  const [localStatus, setLocalStatus] = useState<Record<string, 'draft'|'published'|'archived'>>({});
  
  // Use controlled activeTab if provided, otherwise use internal state
  const currentActiveTab = controlledActiveTab ?? activeTab;
  const [filteredModels, setFilteredModels] = useState<ModelCatalogNode[]>([]);
  const [showSuggestions, setShowSuggestions] = useState<boolean>(false);
  const [highlightedIndex, setHighlightedIndex] = useState<number>(-1);
  const itemRefs = useRef<Record<string, HTMLDivElement | null>>({});
  const [currentMatchIndex, setCurrentMatchIndex] = useState(0);
  const [openActionsFor, setOpenActionsFor] = useState<string | null>(null);
  const [infoModalFor, setInfoModalFor] = useState<string | null>(null);
    // (Removed unused getStatusColor helper after refactor)
  const [confirmDeleteFor, setConfirmDeleteFor] = useState<string | null>(null); const [confirmIsCore, setConfirmIsCore] = useState<boolean>(false); const [confirmModelKey, setConfirmModelKey] = useState<string | undefined>(undefined);
  const [inlineRenameId, setInlineRenameId] = useState<string | null>(null);
  const [inlineNameValue, setInlineNameValue] = useState<string>('');
  const [statusPopoverAnchor, setStatusPopoverAnchor] = useState<HTMLElement | null>(null);
  const [statusPopoverFor, setStatusPopoverFor] = useState<ModelCatalogNode | null>(null);

  const statusLabels = { published: 'Published', draft: 'Draft', archived: 'Archived' };
  const statusIcons = {
    published: <CheckCircleIcon color="success" fontSize="small" />,
    draft: <EditIcon color="warning" fontSize="small" />,
    archived: <ArchiveIcon color="info" fontSize="small" />,
  };

  // Search suggestions state
  const [searchSuggestions, setSearchSuggestions] = useState<SearchSuggestion[]>([]);
  const [searchHighlightedIndex, setSearchHighlightedIndex] = useState<number>(-1);

  // Ensure the actions menu only stays open for the selected row
  useEffect(() => {
    if (openActionsFor && selectedModel?.id !== openActionsFor) {
      setOpenActionsFor(null);
    }
  }, [selectedModel?.id, openActionsFor]);

  const closeAllDropdowns = useCallback(() => {
    setOpenActionsFor(null);
  }, []);

  const startInlineRename = (model: ModelCatalogNode) => { setInlineRenameId(model.id); setInlineNameValue(model.display_name || model.model_key); };
  const commitInlineRename = () => {
    if (!inlineRenameId) return;
    const newName = inlineNameValue.trim();
    if (newName) {
      // Call the onRenameModel prop to save to backend
      onRenameModel?.(inlineRenameId, newName);
      // Also dispatch the event for backward compatibility
      document.dispatchEvent(new CustomEvent('model.rename',{ detail:{ id: inlineRenameId, title:newName }}));
    }
    setInlineRenameId(null);
    setInlineNameValue('');
  };
  const cancelInlineRename = () => { setInlineRenameId(null); setInlineNameValue(''); };

  // Notify parent when active tab changes to update configuration editor
  const _prevActiveTab = useRef<'core' | 'custom'>(currentActiveTab);
  useEffect(() => {
    // Only notify parent when the active tab actually changed and control not provided
    if (onTabChange && !controlledActiveTab && _prevActiveTab.current !== currentActiveTab) {
      onTabChange(currentActiveTab);
    }
    _prevActiveTab.current = currentActiveTab;
  }, [currentActiveTab, onTabChange, controlledActiveTab]);

  // Close on outside click, Escape, and trap Tab focus inside dropdown when open
  useEffect(() => {
    const dropdownSelector = `[data-model-id="${openActionsFor}"] .actions-dropdown`;
    const onDocClick = (e: MouseEvent) => {
      const target = e.target as HTMLElement | null;
      if (!target) return;
      if (target.closest('.actions-dropdown') || target.closest('.actions-toggle')) return;
      closeAllDropdowns();
    };

    const onKeyDown = (e: KeyboardEvent) => {
      const dropdown = document.querySelector(dropdownSelector) as HTMLElement | null;
      if (!dropdown) return;

      if (e.key === 'Escape') {
        e.preventDefault();
        closeAllDropdowns();
        return;
      }

      if (e.key === 'Tab') {
        const focusable = Array.from(dropdown.querySelectorAll<HTMLButtonElement>('.dropdown-item'));
        if (focusable.length === 0) return;
        const active = document.activeElement as HTMLElement | null;
        const first = focusable[0];
        const last = focusable[focusable.length - 1];

        if (!active) return;
        if (!e.shiftKey && active === last) {
          e.preventDefault();
          first.focus();
        } else if (e.shiftKey && active === first) {
          e.preventDefault();
          last.focus();
        }
      }
    };

    document.addEventListener('click', onDocClick);
    document.addEventListener('keydown', onKeyDown);
    // focus first item when opened
    setTimeout(() => {
      const dropdown = document.querySelector(dropdownSelector) as HTMLElement | null;
      const first = dropdown?.querySelector<HTMLButtonElement>('.dropdown-item');
      first?.focus();
    }, 0);

    return () => {
      document.removeEventListener('click', onDocClick);
      document.removeEventListener('keydown', onKeyDown);
    };
  }, [openActionsFor, closeAllDropdowns]);

  // Sort models alphabetically by display_name (fallback to model_key)
  const sortedModels = useMemo(() => {
    return [...models].sort((a, b) => {
      const nameA = (a.display_name || a.model_key || '').toLowerCase();
      const nameB = (b.display_name || b.model_key || '').toLowerCase();
      return nameA.localeCompare(nameB);
    });
  }, [models]);

  const statusCounts = useMemo(() => {
    const counts = { published: 0, draft: 0, archived: 0, total: 0 };
    const list = currentActiveTab === 'core'
      ? sortedModels.filter(m => m.is_core && !m.is_custom)
      : sortedModels.filter(m => m.is_custom);

    list.forEach(model => {
      const effective = localStatus[model.id] ?? model.status;
      if (effective === 'published') counts.published++;
      else if (effective === 'draft') counts.draft++;
      else if (effective === 'archived') counts.archived++;
    });
    counts.total = list.length;
    return counts;
  }, [sortedModels, currentActiveTab, localStatus]);

  // explicit counts for tab labels
  const coreCount = useMemo(() => sortedModels.filter(m => m.is_core && !m.is_custom).length, [sortedModels]);
  const customCount = useMemo(() => sortedModels.filter(m => m.is_custom).length, [sortedModels]);

  const modelsForActiveTab = useMemo(() => {
    let list: ModelCatalogNode[];
    if (currentActiveTab === 'core') {
      list = sortedModels.filter(m => m.is_core && !m.is_custom);
    } else {
      list = sortedModels.filter(m => m.is_custom);
    }
    if (statusFilter === 'active') {
      list = list.filter(m => {
        const s = localStatus[m.id] ?? m.status;
        return s === 'draft' || s === 'published';
      });
    } else if (statusFilter === 'published' || statusFilter === 'draft' || statusFilter === 'archived') {
      list = list.filter(m => (localStatus[m.id] ?? m.status) === statusFilter);
    }
    return list;
  }, [sortedModels, currentActiveTab, statusFilter, localStatus]);

  // Auto-select the first model visible in the sidebar when none is selected yet
  const didAutoSelectRef = useRef(false);
  useEffect(() => {
    if (didAutoSelectRef.current) return;
    if (!selectedModel && modelsForActiveTab.length > 0) {
      didAutoSelectRef.current = true;
      handleModelSelect(modelsForActiveTab[0]);
    }
  }, [selectedModel, modelsForActiveTab]);

  // If parent provides a selectedModel that's custom, ensure the active tab matches so the model is visible
  useEffect(() => {
    if (selectedModel?.is_custom) {
      setActiveTab('custom');
    }
  }, [selectedModel]);

  // Create a stable key for modelsForActiveTab to prevent unnecessary re-renders
  const modelsForActiveTabKey = useMemo(() =>
    // Include display_name in the key so renames (title/display_name changes) cause updates
    modelsForActiveTab.map(m => `${m.id}:${(m.display_name || m.title || m.model_key || '')}`).sort().join(',')
  , [modelsForActiveTab]
  );

  // Filter models based on search term with a stable-key guard to avoid repeated state sets
  const _prevFilteredKey = useRef<string>('');
  useEffect(() => {
    if (!term.trim()) {
      // Only update if the active-tab list actually changed
      if (_prevFilteredKey.current !== modelsForActiveTabKey) {
        setFilteredModels(modelsForActiveTab);
        _prevFilteredKey.current = modelsForActiveTabKey;
      }
      setShowSuggestions(false);
      return;
    }

    const lower = term.toLowerCase();

    const filtered = modelsForActiveTab.filter(model =>
      (model.display_name ?? '').toLowerCase().includes(lower) ||
      (model.description ?? '').toLowerCase().includes(lower) ||
      model.model_key.toLowerCase().includes(lower)
    );

    const filteredKey = filtered.map(m => m.id).sort().join(',');
    if (_prevFilteredKey.current !== filteredKey) {
      setFilteredModels(filtered);
      _prevFilteredKey.current = filteredKey;
      setHighlightedIndex(-1);
      setCurrentMatchIndex(0); // Reset match index when search changes
    }
    setShowSuggestions(true);
  }, [modelsForActiveTabKey, term]);

  // Handle keyboard navigation for typeahead
  const _handleSearchKeyDown = (e: ReactKeyboardEvent) => {
    if (!showSuggestions || filteredModels.length === 0) return;

    switch (e.key) {
      case 'ArrowDown':
        e.preventDefault();
        setHighlightedIndex(prev => 
          prev < filteredModels.length - 1 ? prev + 1 : 0
        );
        break;
      case 'ArrowUp':
        e.preventDefault();
        setHighlightedIndex(prev => 
          prev > 0 ? prev - 1 : filteredModels.length - 1
        );
        break;
      case 'Enter':
        e.preventDefault();
        if (highlightedIndex >= 0 && highlightedIndex < filteredModels.length) {
          handleModelSelect(filteredModels[highlightedIndex]);
          setShowSuggestions(false);
        }
        break;
      case 'Escape':
        e.preventDefault();
        setShowSuggestions(false);
        setHighlightedIndex(-1);
        break;
      case 'ArrowLeft':
  if (term.trim()) {
          e.preventDefault();
          navigateMatch(-1);
        }
        break;
      case 'ArrowRight':
  if (term.trim()) {
          e.preventDefault();
          navigateMatch(1);
        }
        break;
    }
  };
  // referenced to avoid unused-local warnings in some build variants
  void _handleSearchKeyDown;

  const handleModelSelect = (model: ModelCatalogNode) => {
  devLog('[ModelCatalogSidebar] selecting model', { id: model.id, key: model.model_key, is_custom: model.is_custom, is_core: model.is_core });
  setSelectedModel(model);
  setTerm('');
    setShowSuggestions(false);

    // Determine the target tab and the model to show in the editor.
    // If a custom model is selected, populate the core model into the editor (per request).
  const targetTab: 'core' | 'custom' = model.is_custom ? 'custom' : 'core';
  if (onModelSelect) {
  devLog('[ModelCatalogSidebar] calling onModelSelect with targetTab', targetTab);
  onModelSelect(model, targetTab);
  }
    
    // Scroll to model after a short delay to ensure DOM is updated
    setTimeout(() => {
      if (itemRefs.current[model.id]) {
        itemRefs.current[model.id]?.scrollIntoView({
          behavior: 'smooth',
          block: 'nearest'
        });
      }
    }, 100);
    
    // Jump to section if callback provided (keeping for backward compatibility)
    if (onJumpToSection) {
      onJumpToSection(model.id, targetTab);
    }
  };

  // When selectedModel changes (e.g., due to tab switch restore), ensure it's scrolled into view
  useEffect(() => {
    const id = selectedModel?.id;
    if (!id) return;
    const el = itemRefs.current[id];
    if (!el) return;
    // small delay to wait for render
    const t = setTimeout(() => {
      try {
        el.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
      } catch {}
    }, 50);
    return () => clearTimeout(t);
  }, [selectedModel?.id]);

  // Listen for global model deletion events and remove the model from filteredModels
  useEffect(() => {
    const handler = (e: Event) => {
      const detail = (e as CustomEvent).detail as { id?: string } | undefined;
      if (!detail || !detail.id) return;
      const deletedId = detail.id;
      setFilteredModels(prev => prev.filter(m => m.id !== deletedId));
      // If selectedModel is the deleted one, clear selection
      if (selectedModel?.id === deletedId) {
        try { setSelectedModel(null); } catch {}
      }
      // Small QA toast so testers see immediate feedback
      try { toast.success('Model deleted'); } catch {}
    };
    window.addEventListener('model.deleted', handler as any);
    return () => window.removeEventListener('model.deleted', handler as any);
  }, [selectedModel, setSelectedModel]);

  const _handleCreateCustom = (baseModelKey: string) => {
    if (onCreateCustomModel) {
      onCreateCustomModel(baseModelKey);
    }
  };
  // referenced to avoid unused-local warnings in some build variants
  void _handleCreateCustom;

  // Update search suggestions
  useEffect(() => {
    if (term.trim()) {
      const suggestions: SearchSuggestion[] = models
        .filter(model => 
          model.display_name?.toLowerCase().includes(term.toLowerCase()) ||
          model.model_key?.toLowerCase().includes(term.toLowerCase())
        )
        .slice(0, 10) // Limit to 10 suggestions
        .map(model => ({
          id: model.id,
          title: model.display_name || model.model_key,
          subtitle: model.model_key !== model.display_name ? model.model_key : undefined,
          type: model.is_core ? 'Core' : 'Custom',
          description: model.description
        }));
      setSearchSuggestions(suggestions);
    } else {
      setSearchSuggestions([]);
    }
  }, [term, models]);

  const navigateMatch = (direction: number) => {
    if (filteredModels.length === 0) return;
    const newIndex = (currentMatchIndex + direction + filteredModels.length) % filteredModels.length;
    setCurrentMatchIndex(newIndex);
    // Scroll to the matched item
    const model = filteredModels[newIndex];
    if (model && itemRefs.current[model.id]) {
      itemRefs.current[model.id]?.scrollIntoView({
        behavior: 'smooth',
        block: 'nearest'
      });
    }
  };

  // Render a single model row
  const renderModelItem = (model: ModelCatalogNode, isNested = false) => {
    const isSelected = selectedModel?.id === model.id;
    const canCreate = !!(model.is_custom && !model.custom_model_exists && model.metadata?.can_create);
  let rawName = model.display_name || model.model_key;
  // Remove any duplicate (Custom) markers
  rawName = rawName.replace(/\(Custom\)\s*\(Custom\)/gi, '(Custom)');
  // If we're on the custom tab and name ends with (Custom), strip it from display (but preserve tooltip original)
  const displayName = currentActiveTab === 'custom' ? rawName.replace(/\s*\(Custom\)$/i, '') : rawName;

  const currentStatus = localStatus[model.id] ?? model.status ?? 'draft';

    const handleClick = () => {
      if (!canCreate) handleModelSelect(model);
    };

  return (
      <div
        key={model.id}
        data-model-id={model.id}
        ref={(el) => { itemRefs.current[model.id] = el; }} // Add hover-reveal-actions to your CSS
  className={`model-item ${isSelected ? 'selected' : ''} ${isNested ? 'nested' : ''} ${canCreate ? 'can-create' : ''} ${(localStatus[model.id] ?? model.status) === 'archived' ? 'archived' : ''}`}
        onClick={handleClick}
      >
        <div className="model-content">
          <div className="model-header">
            <div className="model-icon">
              {model.is_core || model.is_custom ? (
                // Use cube icon for both core and custom models (same style)
                <IconCube size={16} className="core-icon" />
              ) : (
                <IconCube size={16} className="model-icon" />
              )}
            </div>
            <div className="model-info">
              <div className="model-name-line right-justified">
                <div className="model-name-text-wrapper">
                  {inlineRenameId === model.id ? (
                    <div className="inline-rename">
                      <input
                        className="inline-rename-input"
                        autoFocus
                        aria-label="Model name"
                        placeholder="Model name"
                        value={inlineNameValue}
                        onChange={(e)=>setInlineNameValue(e.target.value)}
                        onKeyDown={(e)=>{ if(e.key==='Enter'){ e.preventDefault(); commitInlineRename(); } if(e.key==='Escape'){ e.preventDefault(); cancelInlineRename(); }}}
                        onClick={(e)=>e.stopPropagation()}
                      />
                      <button className="inline-rename-save" onClick={(e)=>{ e.stopPropagation(); commitInlineRename(); }} aria-label="Save name"><TablerIcons.IconCheck size={14} /></button>
                      <button className="inline-rename-cancel" onClick={(e)=>{ e.stopPropagation(); cancelInlineRename(); }} aria-label="Cancel rename"><TablerIcons.IconX size={14} /></button>
                    </div>
                  ) : (
                    <span className="model-name-text" title={displayName} aria-label={displayName}>
                      {displayName}
                    </span>
                  )}
                </div>
                <div className="model-status-actions-cluster">
                  <Tooltip title={statusLabels[currentStatus]} placement="top" arrow>
                    <button
                      type="button"
                      className="quick-action status-selector-trigger"
                      aria-label={`Change status from ${statusLabels[currentStatus]}`}
                      onClick={(e) => {
                        e.stopPropagation();
                        setStatusPopoverAnchor(e.currentTarget);
                        setStatusPopoverFor(model);
                      }}
                    >
                      {statusIcons[currentStatus]}
                    </button>
                  </Tooltip>

                  {/* Actions are now revealed on hover via CSS on the parent .model-item */}
                    <div className="inline-actions">
                      <Tooltip title="Rename model" placement="top" arrow>
                        <button
                          type="button"
                          className="quick-action rename"
                          aria-label="Rename model"
                          title="Rename model"
                          onClick={(e) => { e.stopPropagation(); startInlineRename(model); }}
                        >
                          <IconEdit size={14} />
                        </button>
                      </Tooltip>
                      <Tooltip title="Clone this model" placement="top" arrow>
                        <button
                          type="button"
                          className="quick-action clone"
                          aria-label="Clone model"
                          title="Clone this model"
                          onClick={(e) => { e.stopPropagation(); (onCloneModel ?? onCreateCustomModel)?.(model.model_key); }}
                        >
                          <IconCopy size={14} />
                        </button>
                      </Tooltip>
                      {/* For core models we expose an additional 'create custom' action and a different delete tooltip */}
                      {model.is_core && (
                        <Tooltip title="Create custom model with extends syntax" placement="top" arrow>
                          <button
                            type="button"
                            className="quick-action add-custom"
                            aria-label="Create custom model"
                            title="Create custom model with extends syntax"
                            onClick={(e) => { e.stopPropagation(); onCreateCustomModel?.(model.model_key); }}
                          >
                            <TablerIcons.IconPlus size={14} />
                          </button>
                        </Tooltip>
                      )}

                      <Tooltip title={model.is_core ? 'Delete core model and its custom(s)' : 'Delete model'} placement="top" arrow>
                        <button
                          type="button"
                          className="quick-action delete"
                          aria-label={model.is_core ? 'Delete core model' : 'Delete model'}
                          title={model.is_core ? 'Delete core model and its custom(s)' : 'Delete model'}
                          onClick={(e) => { e.stopPropagation(); setConfirmDeleteFor(model.id); setConfirmIsCore(!!model.is_core); setConfirmModelKey(model.model_key); }}
                        >
                          <TablerIcons.IconTrash size={14} />
                        </button>
                      </Tooltip>
                    </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    );
  };

  const variantGroups = useMemo(()=>{
    if (currentActiveTab !== 'custom') return null;
    const groups: Record<string, ModelCatalogNode[]> = {};
    filteredModels.forEach(m=>{
      const base = m.parent_model_key || m.model_key.replace(/_custom.*$/,'');
      if(!groups[base]) groups[base]=[];
      groups[base].push(m);
    });
    return groups;
  },[filteredModels,currentActiveTab]);

  const renderModels = () => {
    if(currentActiveTab !== 'custom' || !variantGroups) return filteredModels.map(model => renderModelItem(model));
    return Object.entries(variantGroups).sort((a,b)=>a[0].localeCompare(b[0])).map(([base,mods])=> (
      <div key={base} className="variant-group">
        {mods.sort((a,b)=>(a.display_name||a.model_key).localeCompare(b.display_name||b.model_key)).map(m=> renderModelItem(m,true))}
      </div>
    ));
  };

  if (loading) {
    return (
      <div className="model-catalog-sidebar">
        <div className="sidebar-header"><h2>Model Catalog</h2></div>
        <div className="loading-state"><TablerIcons.IconSettings size={24} className="spinner" /><p>Loading models...</p></div>
      </div>
    );
  }
  if (error) {
    return (
      <div className="model-catalog-sidebar">
        <div className="sidebar-header"><h2>Model Catalog</h2></div>
        <div className="error-state"><p>Error loading models: {error}</p></div>
      </div>
    );
  }

  return (
    <div className={`model-catalog-sidebar ${currentActiveTab}-tab`}>
      <div className="sidebar-header">
        <div className="sidebar-title-row">
          <h2>Model Catalog</h2>
          <div className="header-actions">
            <button
              className="btn-create-custom"
              onClick={() => {
                // The handler from the parent opens the modal without a base model.
                if (onCreateCustomModel) {
                  onCreateCustomModel(''); // Pass empty string for consistency
                }
              }}
              title="Create a new custom model"
              aria-label="Create custom model"
            >
              <TablerIcons.IconPlus size={14} />
              New Model
            </button>
          </div>
        </div>
        
        <div className="search-section">
          <div className="search-and-stats-row">
            {/* Search input removed; universal page search handles queries */}
            <Stack direction="row" spacing={1} sx={{ ml: 'auto' }}>
              <Tooltip title={`Active (Draft + Published)`} placement="bottom" arrow>
                <Chip
                  className={`status-chip active ${statusFilter === 'active' ? 'filled' : 'outlined'}`}
                  icon={<IconCube size={14} />}
                  label={statusCounts.published + statusCounts.draft}
                  aria-label={`Filter: active (${statusCounts.published + statusCounts.draft})`}
                  variant={statusFilter === 'active' ? 'filled' : 'outlined'}
                  onClick={() => setStatusFilter('active')}
                  size="small"
                />
              </Tooltip>
              <Tooltip title={`Published: ${statusCounts.published}`} placement="bottom" arrow>
                <Chip
                  className={`status-chip published ${statusFilter === 'published' ? 'filled' : 'outlined'}`}
                  icon={<IconCheck size={14} />}
                  label={statusCounts.published}
                  aria-label={`Filter: published (${statusCounts.published})`}
                  variant={statusFilter === 'published' ? 'filled' : 'outlined'}
                  onClick={() => setStatusFilter(prev => prev === 'published' ? 'active' : 'published')}
                  size="small"
                />
              </Tooltip>
              <Tooltip title={`Draft: ${statusCounts.draft}`} placement="bottom" arrow>
                <Chip
                  className={`status-chip draft ${statusFilter === 'draft' ? 'filled' : 'outlined'}`}
                  icon={<IconAlertTriangle size={14} />}
                  label={statusCounts.draft}
                  aria-label={`Filter: draft (${statusCounts.draft})`}
                  variant={statusFilter === 'draft' ? 'filled' : 'outlined'}
                  onClick={() => setStatusFilter(prev => prev === 'draft' ? 'active' : 'draft')}
                  size="small"
                />
              </Tooltip>
              <Tooltip title={`Archived: ${statusCounts.archived}`} placement="bottom" arrow>
                <Chip
                  className={`status-chip archived ${statusFilter === 'archived' ? 'filled' : 'outlined'}`}
                  icon={<IconArchive size={14} />}
                  label={statusCounts.archived}
                  aria-label={`Filter: archived (${statusCounts.archived})`}
                  variant={statusFilter === 'archived' ? 'filled' : 'outlined'}
                  onClick={() => setStatusFilter(prev => prev === 'archived' ? 'active' : 'archived')}
                  size="small"
                />
              </Tooltip>
            </Stack>
          </div>
        </div>
      </div>

      {/* Typeahead Search */}
      <div className="sidebar-search">
        <ProfessionalSearchInput
          value={term}
          onChange={setTerm}
          onClear={() => setTerm('')}
          placeholder="Search models..."
          suggestions={searchSuggestions}
          onSuggestionSelect={(suggestion) => {
            // Handle suggestion selection, e.g., select the model
            const model = models.find(m => m.id === suggestion.id);
            if (model) {
              onModelSelect?.(model, currentActiveTab);
            }
          }}
          showSuggestions={showSuggestions}
          mode="filter"
          highlightedIndex={searchHighlightedIndex}
          onHighlightChange={setSearchHighlightedIndex}
          onFocus={() => setShowSuggestions(true)}
          onBlur={() => setTimeout(() => setShowSuggestions(false), 200)}
          size="md"
          variant="enhanced"
        />
      </div>

      <div className="catalog-tabs">
        <BuilderTabs
          activeTab={currentActiveTab}
          setActiveTab={(tab) => {
            if (!controlledActiveTab) {
              setActiveTab(tab as 'core' | 'custom');
            }
            if (onTabChange) {
              onTabChange(tab as 'core' | 'custom');
            }
          }}
          tabs={[
            { id: 'core', label: `Core (${coreCount})` },
            { id: 'custom', label: `Custom (${customCount})` }
          ]}
        />
        {/* Relocate total models to a subtle summary under tabs */}
      </div>
      
      <div className="models-section">
        <div className="models-list">
          {filteredModels.length === 0 ? (
            <div className="empty-state">
              <TablerIcons.IconCube size={32} className="empty-icon" />
              <p>No models found</p>
              {term && (
                <button
                  onClick={() => setTerm && setTerm('')}
                  className="btn-clear-search"
                >
                  Clear search
                </button>
              )}
            </div>
          ) : (
            renderModels()
          )}
        </div>
      </div>

      {/* Model info modal for Info action */}
      <ModelInfoModal
        model={models.find(m => m.id === infoModalFor) || null}
        onClose={() => setInfoModalFor(null)}
      />

      <ConfirmDeleteModal
        open={!!confirmDeleteFor}
        title={confirmIsCore ? "Confirm Core Model Deletion" : "Confirm Model Deletion"}
        message={
          confirmIsCore 
            ? "This will permanently delete the core model and all associated custom model(s)." 
            : "Are you sure you want to delete this custom model? This action cannot be undone."
        }
        associated={
          confirmIsCore && confirmModelKey 
            ? models
                .filter(m => (m.is_custom && m.custom_model_exists) && (m.parent_model_key === confirmModelKey || m.model_key === `${confirmModelKey}_custom`))
                .map(m => ({ 
                  id: m.id, 
                  display_name: m.display_name, 
                  model_key: m.model_key 
                })) 
            : []
        }
        onCancel={() => { 
          setConfirmDeleteFor(null); 
          setConfirmModelKey(undefined); 
          setConfirmIsCore(false); 
        }}
        onConfirm={async () => {
          if (onDeleteModel && confirmDeleteFor) {
            await onDeleteModel(confirmDeleteFor, confirmIsCore, confirmModelKey);
          }
          setConfirmDeleteFor(null);
          setConfirmModelKey(undefined);
          setConfirmIsCore(false);
        }}
      />
  {/* Inline rename replaces modal */}
      <Popover
        open={!!statusPopoverAnchor}
        anchorEl={statusPopoverAnchor}
        onClose={() => { setStatusPopoverAnchor(null); setStatusPopoverFor(null); }}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'center' }}
        transformOrigin={{ vertical: 'top', horizontal: 'center' }}
        onClick={(e) => e.stopPropagation()}
        onMouseDown={(e) => e.stopPropagation()}
      >
        {statusPopoverFor && (() => {
          const handleStatusChange = (newStatus: 'published' | 'draft' | 'archived') => {
            setLocalStatus(s => ({ ...s, [statusPopoverFor!.id]: newStatus }));
            if (newStatus === 'published') onPublishModel?.(statusPopoverFor!.id); else if (newStatus === 'draft') onDraftModel?.(statusPopoverFor!.id); else if (newStatus === 'archived') onArchiveModel?.(statusPopoverFor!.id, !!statusPopoverFor!.is_core, statusPopoverFor!.model_key);
            setStatusPopoverAnchor(null); setStatusPopoverFor(null);
          };
          return (
            <Stack direction="row" p={1}>
              {(Object.keys(statusLabels) as Array<keyof typeof statusLabels>).map((s) => (
                <Tooltip key={s} title={statusLabels[s]} placement="top" arrow>
                  <_IconButton onClick={() => handleStatusChange(s)}>
                    {statusIcons[s]}
                  </_IconButton>
                </Tooltip>
              ))}
            </Stack>
          );
        })()}
      </Popover>
    </div>
  );
};

export default ModelCatalogSidebar;
// End of file