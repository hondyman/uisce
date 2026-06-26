import React, { useState } from 'react';
import { devDebug, devError } from '../../../utils/devLogger';
import {
  Grid,
  TextField,
  FormControlLabel,
  Switch,
  Tabs,
  Tab,
  Box,
  Typography,
  IconButton,
  Tooltip,
  Select,
  MenuItem,
  Button,
  Snackbar,
  Alert,
  CircularProgress,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Stack,
} from '@mui/material';
import { Search } from '@mui/icons-material';
import ArrowForwardIcon from '@mui/icons-material/ArrowForward';
import ArrowBackIcon from '@mui/icons-material/ArrowBack';

import { ExtendsOption } from '../hooks/useAvailableSources';
import { useAvailableSources, AvailableSource } from '../hooks/useAvailableSources';
import { useRouteBlocker } from '../../RouteBlocker/RouteBlocker';
import useBlockableNavigate from '../../RouteBlocker/useBlockableNavigate';
import { ViewComponentsPanel } from './ViewComponentsPanel';
import ViewTypeahead from '../../common/ViewTypeahead';
import CubeTypeahead from '../../common/CubeTypeahead'; // Re-added missing import for CubeTypeahead
import type { ViewOption } from '../../common/ViewTypeahead';
import ViewCodeEditor from '../../common/ViewCodeEditor';
import { AvailableComponentsPanel } from './AvailableComponentsPanel';

interface PropertiesSectionProps {
  propertiesExpanded: boolean;
  viewData: any;
  setViewData: (data: any) => void;
  primaryCube: any;
  setPrimaryCube: (cube: any) => void;
  isEditingPrimaryCube: boolean;
  setIsEditingPrimaryCube: (editing: boolean) => void;
  extendsOptions: ExtendsOption[];
  extendsLoading: boolean;
  extendsInputValue: string;
  setExtendsInputValue: (value: string) => void;
  selectedExtendsValue: ExtendsOption | string | null;
  handleExtendsSelection: (value: any) => void;
  hasTenantScope: boolean;
  tenantId?: string;
  datasourceId?: string;
  onRegisterSave?: (saveFn: () => Promise<any | void>) => void;
}

export const PropertiesSection: React.FC<PropertiesSectionProps> = ({
  viewData,
  setViewData: setViewDataProp,
  primaryCube,
  setPrimaryCube,
  isEditingPrimaryCube,
  setIsEditingPrimaryCube,
  extendsOptions,
  extendsLoading,
  extendsInputValue,
  setExtendsInputValue,
  selectedExtendsValue,
  handleExtendsSelection,
  hasTenantScope,
  tenantId,
  datasourceId,
  onRegisterSave,
  propertiesExpanded,
}) => {
  // Create a safe wrapper around the incoming setViewData prop to avoid
  // repeated no-op updates that can cause render loops. The wrapper accepts
  // either a value or an updater function, matching the native setState API.
  const setViewData = React.useCallback((arg: any) => {
    // Keep identity stable (only depends on setViewDataProp)
    try {
      if (typeof arg === 'function') {
        setViewDataProp((prev: any) => {
          const next = arg(prev);
          try {
            if (JSON.stringify(prev) === JSON.stringify(next)) return prev;
          } catch (e) {
            // fall through to set when we can't compare
          }
          return next;
        });
      } else {
        setViewDataProp((prev: any) => {
          try {
            if (JSON.stringify(prev) === JSON.stringify(arg)) return prev;
          } catch (e) {
            // fall through
          }
          return arg;
        });
      }
    } catch (e) {
      // fallback
      try { setViewDataProp(arg as any); } catch (er) { /* ignore */ }
    }
  }, [setViewDataProp]);

  // Local state for components panel
  const [searchQuery, setSearchQuery] = useState<string>('');
  const [sourceFilter, setSourceFilter] = useState<string>('all');
  const [typeFilter, setTypeFilter] = useState<'all' | 'dimension' | 'measure'>('all');
  const [selectedAvailableItems, setSelectedAvailableItems] = useState<Set<string>>(new Set());
  const [highlightMap, setHighlightMap] = useState<Record<string, 'added' | 'exists'>>({});
  const [targetHighlightMap, setTargetHighlightMap] = useState<Record<string, boolean>>({});
  const lastAddedRef = React.useRef<{ dimensions: any[]; measures: any[] } | null>(null);
  const [lastSelectedIndex, setLastSelectedIndex] = useState<number | null>(null);
  const [selectedViewItems, setSelectedViewItems] = useState<Set<string>>(new Set());
  const [lastSelectedViewIndex, setLastSelectedViewIndex] = useState<number | null>(null);
  const [leftStuck, setLeftStuck] = useState(false);
  const [rightStuck, setRightStuck] = useState(false);
  const [dirty, setDirty] = useState(false);
  const lastSavedSnapshot = React.useRef<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [saveMessage, setSaveMessage] = useState<{ severity: 'success' | 'error' | 'info'; text: string } | null>(null);
  const lastSavedVersion = React.useRef<string | null>(null);
  const [savedIndicatorVisible, setSavedIndicatorVisible] = useState(false);

  // Draft/local autosave
  const draftKey = React.useMemo(() => {
    const name = viewData?.name || viewData?.id || viewData?.qualifiedName || 'unsaved_view';
    return `view_draft_${String(name)}`;
  }, [viewData?.name, viewData?.id, viewData?.qualifiedName]);
  const [hasDraft, setHasDraft] = useState(false);
  const [showRestoreDraftPrompt, setShowRestoreDraftPrompt] = useState(false);

  // Conflict handling state
  const [conflictOpen, setConflictOpen] = useState(false);
  const [conflictServerSnapshot, setConflictServerSnapshot] = useState<any | null>(null);
  // Route-change modal state
  const [navAttempt, setNavAttempt] = useState<{ pathname?: string; action?: any } | null>(null);
  const [showNavPrompt, setShowNavPrompt] = useState(false);
  let navigate: any = null;
  try {
    navigate = useBlockableNavigate();
  } catch (e) {
    navigate = null;
  }
  // Acquire route blocker instance at top-level so we don't call hooks inside effects
  let routeBlocker: any = null;
  try {
    routeBlocker = useRouteBlocker();
  } catch (e) {
    routeBlocker = null;
  }
  // Modal state for info/edit
  const [editorOpen, setEditorOpen] = useState(false);
  const [editorItem, setEditorItem] = useState<any>(null);
  const [editorMode, setEditorMode] = useState<'view' | 'edit'>('view');
  const [editorMetaError, setEditorMetaError] = useState<string | null>(null);

  const openInfoModal = (item: any) => {
    setEditorItem(item);
    setEditorMode('view');
    setEditorOpen(true);
  };

  const openEditModal = (item: any) => {
    setEditorItem(item);
    setEditorMode('edit');
    setEditorOpen(true);
  };

  const closeEditor = () => {
    setEditorOpen(false);
    setEditorItem(null);
  };

  const saveEditor = (changes: any) => {
    // For view-level items, update viewData dimensions/measures matching qualifiedName/id
    if (!editorItem) return;
    const qual = editorItem.id || editorItem.qualifiedName;
    setViewData((prev: any) => {
      const dims = Array.isArray(prev?.dimensions) ? prev.dimensions.map((d: any) => {
        const id = d.qualifiedName || d.id || d.name;
        if (String(id) === String(qual)) return { ...d, ...changes };
        return d;
      }) : prev.dimensions;
      const meas = Array.isArray(prev?.measures) ? prev.measures.map((m: any) => {
        const id = m.qualifiedName || m.id || m.name;
        if (String(id) === String(qual)) return { ...m, ...changes };
        return m;
      }) : prev.measures;
      return { ...prev, dimensions: dims, measures: meas };
    });
    closeEditor();
  };

  // mark dirty whenever viewData changes compared to last saved snapshot
  React.useEffect(() => {
    try {
      const snapshot = JSON.stringify(viewData || {});
      const last = lastSavedSnapshot.current;
      const isDirty = last !== null ? String(snapshot) !== String(last) : true;
      setDirty(Boolean(isDirty));
    } catch (e) {
      setDirty(true);
    }
  }, [viewData]);

  const saveView = async (opts?: { force?: boolean; isAutosave?: boolean }) => {
    // simple safety
    if (!viewData) return;
    const name = viewData.name || viewData.id || viewData?.qualifiedName || '';
    if (!name) {
      setSaveMessage({ severity: 'error', text: 'Cannot save: view has no name/id' });
      return;
    }
    // Ensure we send canonical shapes: if extends is an object, convert to id
    const payload = { ...viewData } as any;
    if (payload.extends && typeof payload.extends === 'object') {
      payload.extends = String(payload.extends.id || payload.extends.ID || payload.extends.name || '');
    }
  setSaving(true);
    try {
      const tenantQuery = tenantId ? `?tenant_id=${encodeURIComponent(String(tenantId))}${datasourceId ? `&tenant_instance_id=${encodeURIComponent(String(datasourceId))}` : ''}` : '';
      const url = `/api/views/${encodeURIComponent(String(name))}${tenantQuery}`;
      const headers: Record<string, string> = {
        'Content-Type': 'application/json',
        ...(tenantId ? { 'X-Tenant-ID': String(tenantId) } : {}),
        ...(datasourceId ? { 'X-Tenant-Datasource-ID': String(datasourceId) } : {}),
      };
      if (opts?.force) headers['X-Force-Save'] = '1';
      const resp = await fetch(url, {
        method: 'PUT',
        headers,
        body: JSON.stringify(payload),
      });
      if (!resp.ok) {
        const txt = await resp.text();
        if (resp.status === 409) {
          let serverJson = null;
          try { serverJson = await resp.json(); } catch (e) { /* ignore */ }
          setConflictServerSnapshot(serverJson || txt || { message: txt });
          setConflictOpen(true);
          setSaveMessage({ severity: 'error', text: `Save conflict: server has newer changes.` });
        } else {
          setSaveMessage({ severity: 'error', text: `Save failed: ${resp.status} ${txt}` });
        }
        setSaving(false);
        return resp;
      } else {
        // On success capture etag/version if provided
        const etag = resp.headers.get('ETag') || resp.headers.get('etag') || null;
        if (etag) lastSavedVersion.current = etag;
        try {
          const json = await resp.json().catch(() => null);
          if (json && (json.version || json.updated_at || json.updatedAt)) {
            lastSavedVersion.current = String(json.version || json.updated_at || json.updatedAt);
          }
        } catch (e) {}
        lastSavedSnapshot.current = JSON.stringify(viewData || {});
        setDirty(false);
        setSaveMessage({ severity: 'success', text: opts?.isAutosave ? 'Autosaved' : 'View saved' });
        setSavedIndicatorVisible(true);
        setTimeout(() => setSavedIndicatorVisible(false), 2500);
        try { localStorage.removeItem(draftKey); setHasDraft(false); } catch (e) {}
      }
    } catch (e: any) {
      setSaveMessage({ severity: 'error', text: `Save error: ${String(e?.message || e)}` });
    } finally {
      setSaving(false);
    }
    return undefined;
  };

  // keep a ref to the latest saveView so we can register a stable callback once
  const saveViewRef = React.useRef<typeof saveView | null>(null);
  React.useEffect(() => { saveViewRef.current = saveView; }, [saveView]);

  // If parent wants to be able to trigger the local save (e.g. header Save button), register a stable wrapper
  React.useEffect(() => {
    if (typeof onRegisterSave === 'function') {
      try {
        onRegisterSave(() => {
          // Call the latest saveView via ref to avoid re-registering every render
          const fn = saveViewRef.current;
          return fn ? fn() : Promise.resolve();
        });
      } catch (e) {
        // ignore
      }
    }
    // intentionally run only when onRegisterSave changes
  }, [onRegisterSave]);

  // warn on window unload when dirty
  React.useEffect(() => {
    const before = (e: BeforeUnloadEvent) => {
      if (dirty) {
        e.preventDefault();
        e.returnValue = '';
        return '';
      }
      return undefined;
    };
    window.addEventListener('beforeunload', before);
    return () => window.removeEventListener('beforeunload', before);
  }, [dirty]);

  // Persist local draft to localStorage (debounced)
  React.useEffect(() => {
    if (!viewData) return undefined;
    let timer: any = null;
    const saveDraft = () => {
      try {
        const snapshot = JSON.stringify(viewData || {});
        localStorage.setItem(draftKey, snapshot);
        setHasDraft(true);
      } catch (e) {}
    };
    if (dirty) {
      timer = setTimeout(saveDraft, 2000);
    }
    return () => { if (timer) clearTimeout(timer); };
  }, [viewData, dirty, draftKey]);

  // On mount, check for draft
  React.useEffect(() => {
    try {
      const draft = localStorage.getItem(draftKey);
      if (draft) {
        const current = JSON.stringify(viewData || {});
        if (draft !== current) {
          setHasDraft(true);
          setShowRestoreDraftPrompt(true);
        }
      }
    } catch (e) {}
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // In-app navigation blocker using react-router's internal navigator.block
  // pattern. This intercepts internal navigations (useNavigate, Link, back/forward)
  // and allows us to show a Save/Discard/Cancel modal. We rely on the
  // UNSAFE_NavigationContext navigator.block API; when the user confirms we
  // call the transition.retry() to continue the navigation.
  React.useEffect(() => {
    // Register a blocker via the centralized RouteBlocker provider. Keep registration
    // stable by depending only on the routeBlocker instance and `dirty` flag so
    // we don't re-register on unrelated identity changes.
    let unregister: (() => void) | null = null;
    try {
      if (routeBlocker && typeof routeBlocker.register === 'function') {
        unregister = routeBlocker.register((tx: any) => {
          if (!dirty) return true;
          // keep the transition object so confirmNavigate can proceed with retry
          setNavAttempt({ pathname: tx.location?.pathname, action: tx });
          setShowNavPrompt(true);
          // Returning false pauses the navigation; confirmNavigate will call retry()
          return false;
        });
      }
    } catch (e) {
      // best effort
    }
    return () => { try { if (unregister) unregister(); } catch (e) {} };
  }, [routeBlocker, dirty]);

  const confirmNavigate = async (action: 'save' | 'discard' | 'cancel') => {
    setShowNavPrompt(false);
    if (action === 'cancel') {
      setNavAttempt(null);
      return;
    }
    if (action === 'discard') {
      // clear dirty by resetting lastSavedSnapshot to current view so guard won't reappear
      lastSavedSnapshot.current = JSON.stringify(viewData || {});
      setDirty(false);
      setNavAttempt(null);
      // allow native navigation to proceed by manually calling navigate to the captured path
      if (navAttempt && navAttempt.action && typeof navAttempt.action.retry === 'function') {
        try { navAttempt.action.retry(); } catch (e) { if (navAttempt && navAttempt.pathname && navigate) navigate(navAttempt.pathname); }
      } else if (navAttempt && navAttempt.pathname && navigate) navigate(navAttempt.pathname);
      return;
    }
    if (action === 'save') {
      // Attempt to save, then continue navigation on success
      const resp = await saveView();
      // If saveView returned a Response (non-ok handled), we treat as failure
      if (resp && (resp.status && resp.status !== 200 && resp.status !== 204)) {
        // keep user on page; message already set by saveView
        setNavAttempt(null);
        return;
      }
      // saved, now continue the blocked transition via retry or fall back to navigate
      if (navAttempt && navAttempt.action && typeof navAttempt.action.retry === 'function') {
        try { navAttempt.action.retry(); } catch (e) { if (navAttempt && navAttempt.pathname && navigate) navigate(navAttempt.pathname); }
      } else if (navAttempt && navAttempt.pathname && navigate) navigate(navAttempt.pathname);
      setNavAttempt(null);
      return;
    }
  };

  // Conflict resolution handlers
  const applyServerSnapshot = () => {
    if (!conflictServerSnapshot) return;
    try {
      if (typeof conflictServerSnapshot === 'string') {
        // can't parse
      } else {
        setViewData((prev: any) => ({ ...prev, ...conflictServerSnapshot }));
        lastSavedSnapshot.current = JSON.stringify(conflictServerSnapshot || {});
        setDirty(false);
      }
    } catch (e) {}
    setConflictOpen(false);
    setConflictServerSnapshot(null);
  };

  const forceKeepLocal = async () => {
    setConflictOpen(false);
    setConflictServerSnapshot(null);
    await saveView({ force: true });
  };

  // Periodic optimistic autosave (best-effort).
  // Attempts to save every 30 seconds if dirty. If a conflict (409) or error
  // occurs, it surfaces a message and stops autosaving to avoid thrashing.
  React.useEffect(() => {
    let stopped = false;
    const interval = setInterval(async () => {
      if (stopped) return;
      if (!dirty) return;
      try {
        setSaving(true);
        const resp = await saveView();
        // If saveView returned a Response with non-ok, stop autosave and show message
        if (resp && resp.status && ![200, 201, 204].includes(resp.status)) {
          setSaveMessage({ severity: 'error', text: `Autosave failed (${resp.status}). Automatic saves paused.` });
          stopped = true;
        } else {
          // autosave succeeded; transient info
          setSaveMessage({ severity: 'info', text: 'Autosaved' });
        }
      } catch (e) {
        stopped = true;
      } finally {
        setSaving(false);
      }
    }, 30000);
    return () => {
      clearInterval(interval);
    };
  }, [dirty]);

  // Build selectedRefs similar to the editor so availableSources can filter appropriately
  const selectedRefs = React.useMemo(() => {
    const s = new Set<string>();
    const addRef = (value?: string | null) => {
      if (!value) return;
      const normalized = String(value).trim().toLowerCase();
      if (normalized) s.add(normalized);
    };
    const cubes = Array.isArray(viewData?.cubes) ? viewData.cubes : [];
    cubes.forEach((c: any) => {
      if (typeof c === 'string') addRef(c);
      else if (c && typeof c === 'object') {
        addRef(c.id ? String(c.id) : undefined);
        addRef(c.model_key ? String(c.model_key) : undefined);
        addRef(c.name ? String(c.name) : undefined);
      }
    });
    const joins = Array.isArray(viewData?.join_paths) ? viewData.join_paths : [];
    joins.forEach((jp: any) => {
      if (typeof jp === 'string') addRef(jp);
      else if (jp && typeof jp === 'object') {
        addRef(jp.id ? String(jp.id) : undefined);
        addRef(jp.path ? String(jp.path) : undefined);
        addRef(jp.label ? String(jp.label) : undefined);
      }
    });
    if (typeof viewData?.extends === 'string' && viewData.extends.trim()) addRef(viewData.extends);
    else if (viewData?.extends && typeof viewData.extends === 'object') {
      const id = viewData.extends.id || viewData.extends.ID || viewData.extends.name || undefined;
      if (id) addRef(id);
    }
    if (selectedExtendsValue) {
      if (typeof selectedExtendsValue === 'string') addRef(selectedExtendsValue);
      else if (selectedExtendsValue && typeof selectedExtendsValue === 'object') {
        const id = (selectedExtendsValue as any).id || (selectedExtendsValue as any).ID || (selectedExtendsValue as any).name;
        if (id) addRef(id);
      }
    }
    return s;
  }, [viewData?.cubes, viewData?.join_paths, viewData?.extends, selectedExtendsValue]);

  const { availableSources, fetchAvailableSources } = useAvailableSources(() => Boolean(tenantId && datasourceId), tenantId, datasourceId, viewData, selectedRefs as any);

  const filteredAvailableSources: AvailableSource[] = React.useMemo(() => {
    const q = searchQuery.trim().toLowerCase();
    const base = (availableSources || []).map(src => ({
      ...src,
      items: (src.items || []).filter(item => {
        if (q && !(item.name.toLowerCase().includes(q) || (item.description || '').toLowerCase().includes(q))) return false;
        if (sourceFilter && sourceFilter !== 'all' && String(src.id) !== String(sourceFilter)) return false;
        return true;
      })
    })).filter(s => (s.items || []).length > 0);

    // Promote explicit extends if not present (keeps behavior from earlier implementation)
    try {
      const explicitExtendsId = typeof viewData?.extends === 'string' && viewData.extends.trim() ? String(viewData.extends).toLowerCase() : '';
      if (explicitExtendsId) {
        const alreadyIncluded = base.some(s => String(s.id).toLowerCase() === explicitExtendsId);
        if (!alreadyIncluded) {
          const original = availableSources.find(s => String(s.id).toLowerCase() === explicitExtendsId);
          if (original) {
            const filteredVersion = {
              ...original,
              items: (original.items || []).filter(item => {
                if (q && !(item.name.toLowerCase().includes(q) || (item.description || '').toLowerCase().includes(q))) return false;
                if (sourceFilter && sourceFilter !== 'all' && String(original.id) !== String(sourceFilter)) return false;
                return true;
              })
            };
            const toInsert = (filteredVersion.items && filteredVersion.items.length > 0) ? filteredVersion : original;
            return [toInsert, ...base];
          }
          if (selectedExtendsValue && typeof selectedExtendsValue === 'object' && (((selectedExtendsValue as any).dimensions) || ((selectedExtendsValue as any).measures))) {
            const viewObj: any = selectedExtendsValue as any;
            const items = [] as any[];
            if (Array.isArray(viewObj.dimensions)) {
              viewObj.dimensions.forEach((dim: any, i: number) => {
                items.push({
                  id: `${viewObj.id || viewObj.ID || viewObj.name}.${dim.name || dim.id || `dimension_${i}`}`,
                  name: dim.title || dim.name || dim.id || `dimension_${i}`,
                  type: 'dimension',
                  source: viewObj.id || viewObj.ID || viewObj.name,
                  description: dim.description,
                  datatype: dim.datatype || dim.type,
                });
              });
            }
            if (Array.isArray(viewObj.measures)) {
              viewObj.measures.forEach((m: any, i: number) => {
                items.push({
                  id: `${viewObj.id || viewObj.ID || viewObj.name}.${m.name || m.id || `measure_${i}`}`,
                  name: m.title || m.name || m.id || `measure_${i}`,
                  type: 'measure',
                  source: viewObj.id || viewObj.ID || viewObj.name,
                  description: m.description,
                  datatype: m.datatype || m.type,
                });
              });
            }
            if (items.length > 0) {
              const promoted = {
                id: viewObj.id || viewObj.ID || viewObj.name,
                name: viewObj.title || viewObj.name || 'Extended View',
                type: 'extended_view',
                items,
                expanded: true,
                filteredOutCount: 0,
                isCore: Boolean(viewObj.is_core || viewObj.isCore),
                isCustom: Boolean(viewObj.is_custom || viewObj.isCustom),
              } as any;
              return [promoted, ...base];
            }
          }
        }
      }
    } catch (e) {
      // ignore
    }

    return base;
  }, [availableSources, searchQuery, sourceFilter, viewData?.extends, selectedExtendsValue]);
  const flattenedAvailableItems = React.useMemo(() => {
    const rows: Array<{
      id: string;
      name: string;
      type: 'dimension' | 'measure' | 'view';
      description?: string;
      datatype?: string;
      _sourceId: string;
      _sourceName: string;
      _sourceType: AvailableSource['type'];
      source: string;
    }> = [];
    filteredAvailableSources.forEach((src) => {
      (src.items || []).forEach((item) => {
        // apply typeFilter (dimensions/measures) from parent
        if (typeFilter !== 'all' && item.type !== typeFilter) return;
        rows.push({
          ...item,
          _sourceId: String(src.id),
          _sourceName: src.name,
          _sourceType: src.type,
          source: item.source || String(src.id),
        });
      });
    });
    // Sort alphabetically by display name (case-insensitive) so the Available pane is ordered
    rows.sort((a, b) => {
      const A = String(a.name || '').toLowerCase();
      const B = String(b.name || '').toLowerCase();
      if (A < B) return -1;
      if (A > B) return 1;
      return 0;
    });
    return rows;
  }, [filteredAvailableSources, typeFilter]);

  const availableTotals = React.useMemo(() => {
    let dims = 0;
    let meas = 0;
    (filteredAvailableSources || []).forEach((src) => {
      (src.items || []).forEach((item: any) => {
        if (item.type === 'dimension') dims += 1;
        if (item.type === 'measure') meas += 1;
      });
    });
    return { dimensionCount: dims, measureCount: meas };
  }, [filteredAvailableSources]);

  const normalizedExtendsId = React.useMemo(() => {
    if (!viewData) return '';
    if (typeof viewData.extends === 'string') return String(viewData.extends);
    if (viewData.extends && typeof viewData.extends === 'object') return String(viewData.extends.id || viewData.extends.ID || viewData.extends.name || '');
    return '';
  }, [viewData?.extends, selectedExtendsValue]);

  const normalizedPrimaryCubeId = React.useMemo(() => {
    if (primaryCube && typeof primaryCube === 'object') return String((primaryCube as any).id || (primaryCube as any).ID || (primaryCube as any).name || '');
    if (Array.isArray(viewData?.cubes) && viewData.cubes.length > 0) {
      const c = viewData.cubes[0];
      if (typeof c === 'string') return String(c);
      if (c && typeof c === 'object') return String(c.id || c.model_key || c.name || '');
    }
    return '';
  }, [primaryCube, viewData?.cubes]);

  const enrichedAvailableSources = React.useMemo(() => {
    return (availableSources || []).map((s: any) => {
      const filtered = (filteredAvailableSources || []).find((fs: any) => String(fs.id) === String(s.id));
      return {
        ...s,
        items: s.items || [],
        filteredOutCount: Math.max(0, (s.items || []).length - ((filtered?.items || []).length || 0)),
        expanded: true,
        isCore: Boolean(s.is_core || s.isCore),
        isCustom: Boolean(s.is_custom || s.isCustom),
      };
    });
  }, [availableSources, filteredAvailableSources]);

  const availableSourceSummaries = React.useMemo(() => (enrichedAvailableSources || []).map((s: any) => ({ id: s.id, name: s.name })), [enrichedAvailableSources]);

  const dimensionsSignature = React.useMemo(() => {
    const dims = Array.isArray(viewData?.dimensions) ? viewData.dimensions : [];
    return dims
      .map((d: any) => String(d?.qualifiedName || d?.id || d?.name || ''))
      .filter(Boolean)
      .sort()
      .join('|');
  }, [viewData?.dimensions]);

  const measuresSignature = React.useMemo(() => {
    const meas = Array.isArray(viewData?.measures) ? viewData.measures : [];
    return meas
      .map((m: any) => String(m?.qualifiedName || m?.id || m?.name || ''))
      .filter(Boolean)
      .sort()
      .join('|');
  }, [viewData?.measures]);

  // Ensure available sources are fetched when scope/viewData changes.
  // Note: `fetchAvailableSources` may have an unstable identity coming from the hook
  // so we intentionally exclude it from the dependency list and only refetch when
  // tenantId, datasourceId, selectedRefs, or the current component signatures change.
  // eslint-disable-next-line react-hooks/exhaustive-deps
  React.useEffect(() => {
    if (!tenantId || !datasourceId) return;
    try { fetchAvailableSources(); } catch (e) {}
  }, [tenantId, datasourceId, selectedRefs, dimensionsSignature, measuresSignature]);

  const extendsChangeRef = React.useRef<string>('');
  React.useEffect(() => {
    const next = normalizedExtendsId.trim().toLowerCase();
    if (extendsChangeRef.current === next) return;
    extendsChangeRef.current = next;
    setSelectedAvailableItems(new Set());
    setHighlightMap({});
    setTargetHighlightMap({});
    if (!next) {
      if (sourceFilter !== 'all') setSourceFilter('all');
    }
  }, [normalizedExtendsId, sourceFilter]);

  const cubeChangeRef = React.useRef<string>('');
  React.useEffect(() => {
    const next = normalizedPrimaryCubeId.trim().toLowerCase();
    if (cubeChangeRef.current === next) return;
    cubeChangeRef.current = next;
    setSelectedAvailableItems(new Set());
    setHighlightMap({});
    setTargetHighlightMap({});
  }, [normalizedPrimaryCubeId]);

  const pendingExtendsFilterRef = React.useRef<string | null>(null);
  React.useEffect(() => {
    const next = normalizedExtendsId.trim().toLowerCase();
    pendingExtendsFilterRef.current = next || null;
  }, [normalizedExtendsId]);

  React.useEffect(() => {
    if (!pendingExtendsFilterRef.current) return;
    const target = pendingExtendsFilterRef.current;
  const match = enrichedAvailableSources.find((src: any) => String(src.id).toLowerCase() === target);
    if (match) {
      const idStr = String(match.id);
      if (sourceFilter !== idStr) setSourceFilter(idStr);
      pendingExtendsFilterRef.current = null;
    }
  }, [enrichedAvailableSources, sourceFilter]);

  React.useEffect(() => {
    if (sourceFilter === 'all') return;
  const exists = enrichedAvailableSources.some((src: any) => String(src.id) === String(sourceFilter));
    if (!exists) {
      setSourceFilter('all');
    }
  }, [enrichedAvailableSources, sourceFilter]);

  const handleAvailableItemClick = React.useCallback(
    (itemId: string, index: number, modifiers: { additive?: boolean; range?: boolean }) => {
      const additive = Boolean(modifiers.additive);
      const range = Boolean(modifiers.range);

      setSelectedAvailableItems((prev) => {
        let next = new Set(prev);

        if (range && lastSelectedIndex !== null && flattenedAvailableItems.length > 0) {
          if (!additive) next = new Set();
          const start = Math.min(lastSelectedIndex, index);
          const end = Math.max(lastSelectedIndex, index);
          for (let i = start; i <= end; i += 1) {
            const row = flattenedAvailableItems[i];
            if (row) next.add(row.id);
          }
          return next;
        }

        if (additive) {
          if (next.has(itemId)) next.delete(itemId);
          else next.add(itemId);
          return next;
        }

        return new Set([itemId]);
      });

      setLastSelectedIndex(index);
    },
    [flattenedAvailableItems, lastSelectedIndex]
  );

  const handleViewItemClick = React.useCallback(
    (itemId: string, index: number, modifiers: { additive?: boolean; range?: boolean }) => {
      const additive = Boolean(modifiers.additive);
      const range = Boolean(modifiers.range);

      setSelectedViewItems((prev) => {
        let next = new Set(prev);

        if (range && lastSelectedViewIndex !== null) {
          if (!additive) next = new Set();
          const start = Math.min(lastSelectedViewIndex, index);
          const end = Math.max(lastSelectedViewIndex, index);
          for (let i = start; i <= end; i += 1) {
            const row = viewItems[i];
            if (row) next.add(row.id);
          }
          return next;
        }

        if (additive) {
          if (next.has(itemId)) next.delete(itemId);
          else next.add(itemId);
          return next;
        }

        return new Set([itemId]);
      });

      setLastSelectedViewIndex(index);
    },
    [lastSelectedViewIndex]
  );

  // Build viewItems from viewData.dimensions and viewData.measures (these are the
  // components currently attached to the view). This is what appears in the
  // ViewComponentsPanel.
  const viewItems = React.useMemo(() => {
    const rows: Array<any> = [];
    const dims = Array.isArray(viewData?.dimensions) ? viewData.dimensions : [];
    const meas = Array.isArray(viewData?.measures) ? viewData.measures : [];
    dims.forEach((d: any, i: number) => {
      rows.push({
        id: d.qualifiedName || d.id || `${d.name || d.title || 'dim'}_${i}`,
        name: d.title || d.name || d.qualifiedName || d.id || `Dimension ${i + 1}`,
        description: d.description,
        datatype: d.type || d.datatype || 'string',
        type: 'dimension',
        originalIndex: i,
        // prefer an explicit _sourceName if present (preserved from available items)
        _sourceName: d._sourceName || d.source_name || d.source || '',
        _sourceId: d._sourceId || d.source || '',
        _sourceType: 'extended_view'
      });
    });
    meas.forEach((m: any, i: number) => {
      rows.push({
        id: m.qualifiedName || m.id || `${m.name || m.title || 'meas'}_${i}`,
        name: m.title || m.name || m.qualifiedName || m.id || `Measure ${i + 1}`,
        description: m.description,
        datatype: m.type || m.datatype || 'number',
        type: 'measure',
        originalIndex: i,
        _sourceName: m._sourceName || m.source_name || m.source || '',
        _sourceId: m._sourceId || m.source || '',
        _sourceType: 'extended_view'
      });
    });
    // Sort the view items alphabetically by display name (case-insensitive)
    rows.sort((a, b) => {
      const A = String(a.name || '').toLowerCase();
      const B = String(b.name || '').toLowerCase();
      if (A < B) return -1;
      if (A > B) return 1;
      return 0;
    });
    return rows;
  }, [viewData?.dimensions, viewData?.measures]);

  const handleRemoveSelectedViewItems = React.useCallback(() => {
    if (!selectedViewItems || selectedViewItems.size === 0) return;
    // Remove from viewData.dimensions/measures based on qualifiedName/id
    const dims = Array.isArray(viewData?.dimensions) ? viewData.dimensions.filter((d: any) => !selectedViewItems.has(d.qualifiedName || d.id || String(d.name))) : [];
    const meas = Array.isArray(viewData?.measures) ? viewData.measures.filter((m: any) => !selectedViewItems.has(m.qualifiedName || m.id || String(m.name))) : [];
    setViewData({ ...viewData, dimensions: dims, measures: meas });
    // Clear selection and remove highlight/added flags (remove both id and qualifiedName variants)
    const newHighlight = { ...highlightMap };
    selectedViewItems.forEach(k => {
      // remove exact key
      delete newHighlight[k];
      // also remove any matching qualifiedName/id variants just in case
      Object.keys(newHighlight).forEach(existingKey => {
        if (String(existingKey).toLowerCase() === String(k).toLowerCase()) delete newHighlight[existingKey];
      });
    });
    setHighlightMap(newHighlight);
    setSelectedViewItems(new Set());
    setLastSelectedViewIndex(null);
  }, [selectedViewItems, viewData, setViewData, highlightMap]);

  const handleAddSelectedItems = () => {
    const itemsToAdd: { dimensions: any[]; measures: any[] } = { dimensions: [], measures: [] };
    selectedAvailableItems.forEach(itemId => {
      const source = enrichedAvailableSources.find((s: any) => (s.items || []).some((i: any) => i.id === itemId));
      const item = source?.items.find((i: any) => i.id === itemId);
      if (!item || !source) return;
      const namePart = String(item.id).split('.').pop();
      // Build a view-level component that preserves available item metadata so
      // the view panel shows the exact same chips, icons and descriptions.
      const viewItem: any = {
        // identification
        id: item.id,
        qualifiedName: item.id,
        // display
        name: item.name,
        title: item.name,
        description: item.description,
        // source metadata
        _sourceName: source.name,
        _sourceType: source.type,
        source: source.id || source.name,
        // datatype and SQL guidance
        datatype: item.datatype || (item.type === 'measure' ? 'number' : 'string'),
        sql: `\${${source.name}.${namePart}}`,
        // keep original component type (dimension/measure) for bookkeeping
        componentType: item.type,
      };

      if (item.type === 'dimension' || item.type === 'view') itemsToAdd.dimensions.push(viewItem);
      if (item.type === 'measure') itemsToAdd.measures.push(viewItem);
    });

    if (itemsToAdd.dimensions.length === 0 && itemsToAdd.measures.length === 0) return;
    lastAddedRef.current = itemsToAdd;
    setViewData((prev: any) => ({
      ...prev,
      dimensions: [...(prev.dimensions || []), ...itemsToAdd.dimensions],
      measures: [...(prev.measures || []), ...itemsToAdd.measures],
    }));
    // highlight items in view and mark selected available items as 'added'
    const newHighlights: Record<string, boolean> = {};
    const newAdded: Record<string, 'added' | 'exists'> = {};
    itemsToAdd.dimensions.forEach(d => { if (d.qualifiedName) newHighlights[d.qualifiedName] = true; if (d.qualifiedName) newAdded[d.qualifiedName] = 'added'; });
    itemsToAdd.measures.forEach(m => { if (m.qualifiedName) newHighlights[m.qualifiedName] = true; if (m.qualifiedName) newAdded[m.qualifiedName] = 'added'; });
    setTargetHighlightMap(ph => ({ ...ph, ...newHighlights }));
    setHighlightMap(h => ({ ...h, ...newAdded }));
    setTimeout(() => setTargetHighlightMap(ph => { const copy = { ...ph }; Object.keys(newHighlights).forEach(k => delete copy[k]); return copy; }), 900);
    setSelectedAvailableItems(new Set());
    setLastSelectedIndex(null);
  };

  

  // per-row removal is handled via the center "Remove selected view components" action
  // Note: cube-derived flattened items are no longer used in this panel; view-level
  // components are computed via `viewItems` and available components are provided
  // via `flattenedAvailableItems` above.

  // Icons for data types are rendered inside the ViewComponentsPanel; retained here earlier during refactor.
  // Local tab state controls which tab is visible. This is intentionally local so the
  // parent doesn't also toggle its own view and cause duplicate rendering of components.
  const [tabIndex, setTabIndex] = useState<number>(propertiesExpanded ? 0 : 1);
  // Input state for cube typeahead so fetchOptions=true will react to typing
  const [cubeInputValue, setCubeInputValue] = useState<string>('');
  
  // render instrumentation to detect runaway re-renders
  const renderCount = React.useRef(0);
    React.useEffect(() => {
    renderCount.current += 1;
    if (renderCount.current > 50) {
      // dump a broader set of signals to help track churn
      try {
        devError('PropertiesSection render count high:', renderCount.current);
        devDebug('PS debug:', {
          dirty,
          saving,
          tabIndex,
          searchQuery,
          sourceFilter,
          typeFilter,
          normalizedExtendsId,
          normalizedPrimaryCubeId,
        });
      } catch (e) {}
    }
  });

  // Track changes to key derived values so we can see what flips between renders
  const _prevDebugRef = React.useRef<any>(null);
  React.useEffect(() => {
    try {
      const cur = {
        normalizedExtendsId,
        normalizedPrimaryCubeId,
        selectedRefsSize: (selectedRefs && typeof selectedRefs.size === 'number') ? selectedRefs.size : undefined,
        enrichedAvailableSourcesLen: (enrichedAvailableSources || []).length,
        filteredAvailableSourcesLen: (filteredAvailableSources || []).length,
        flattenedAvailableItemsLen: (flattenedAvailableItems || []).length,
        viewItemsLen: (viewItems || []).length,
        dimensionsSig: dimensionsSignature,
        measuresSig: measuresSignature,
      };
      const prev = _prevDebugRef.current;
      if (prev) {
        const diffs: any = {};
        Object.keys(cur).forEach((k) => {
          if (String(prev[k]) !== String((cur as any)[k])) diffs[k] = { from: prev[k], to: (cur as any)[k] };
        });
        if (Object.keys(diffs).length > 0) {
          devDebug('PropertiesSection derived diffs:', diffs);
        }
      }
      _prevDebugRef.current = cur;
    } catch (e) {}
  });

  // helper ref to avoid oscillating extends/name <-> id conversions (removed unused ref)

  return (
    <>
      {/* Debug overlay - visible only in development */}
      {(typeof process !== 'undefined' && process.env?.NODE_ENV === 'development') || import.meta.env.DEV ? (
        <Box sx={{ position: 'fixed', right: 12, bottom: 12, zIndex: 2000 }}>
          <Box sx={{ bgcolor: 'background.paper', p: 1, borderRadius: 1, boxShadow: 3, fontSize: 12 }}>
            <Typography variant="subtitle2" sx={{ fontWeight: 600 }}>Extends debug</Typography>
            <Typography variant="caption">selectedExtendsValue: {JSON.stringify(selectedExtendsValue ?? '')}</Typography>
            <Typography variant="caption">viewData.extends: {JSON.stringify(viewData?.extends ?? '')}</Typography>
          </Box>
        </Box>
      ) : null}
      {/* Tabs: Properties | Components | Code */}
      <Box sx={{ borderBottom: 1, borderColor: 'divider', mb: 2 }}>
        <Tabs
          value={tabIndex}
          onChange={(_, v) => setTabIndex(v)}
          aria-label="View editor tabs"
        >
          <Tab label="Properties" />
          <Tab label="Components" />
          <Tab label="Code" />
        </Tabs>
      </Box>

      {/* Properties tab */}
      {tabIndex === 0 && (
        <Grid container spacing={3}>
          <Grid item xs={12} md={4}>
            <TextField
              label="Name"
              value={viewData.name || viewData.title || ''}
              onChange={(e) => setViewData({ ...viewData, name: e.target.value })}
              fullWidth
              size="small"
              placeholder="Unique view name"
            />
          </Grid>

          <Grid item xs={12} md={4}>
            <ViewTypeahead
              options={extendsOptions as ViewOption[]}
              loading={extendsLoading}
              inputValue={extendsInputValue}
              onInputChange={(v) => {
                setExtendsInputValue(v);
                // If the user cleared the input, propagate clear to the parent so
                // the controlled `value` (viewData.extends) and selectedExtendsValue
                // are cleared and the Autocomplete doesn't revert to the UUID.
                if (!v) handleExtendsSelection('');
              }}
              value={selectedExtendsValue}
              onChange={(v) => handleExtendsSelection(v)}
              placeholder={hasTenantScope ? 'Search views to extend (optional)' : 'Select tenant & datasource'}
              disabled={!hasTenantScope}
              fetchOptions={true}
              resolveOnFetch={false}
              tenantId={tenantId}
              datasourceId={datasourceId}
              status="published"
              pageSize={100}
            />
          </Grid>

          <Grid item xs={12} md={4}>
            <FormControlLabel
              control={
                <Switch
                  checked={viewData.public !== false}
                  onChange={(e) => setViewData({ ...viewData, public: e.target.checked })}
                />
              }
              label="Public View"
              sx={{ mt: 1 }}
            />
          </Grid>

          <Grid item xs={12} md={6}>
            <Typography variant="subtitle2" gutterBottom>
              Primary Cube
            </Typography>
            {isEditingPrimaryCube ? (
              <CubeTypeahead
                fetchOptions={true}
                tenantId={tenantId}
                datasourceId={datasourceId}
                pageSize={100}
                disabled={!hasTenantScope}
                value={primaryCube}
                inputValue={cubeInputValue}
                onInputChange={(_, v) => setCubeInputValue(v)}
                onChange={(_, newValue: any) => {
                  if (newValue) {
                    setPrimaryCube(newValue);
                    setViewData({
                      ...viewData,
                      cubes: [{ id: newValue.id, name: newValue.display_name || newValue.model_key }]
                    });
                    setIsEditingPrimaryCube(false);
                  }
                }}
                placeholder={hasTenantScope ? 'Search cubes (published)' : 'Select tenant & datasource'}
              />
            ) : (
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, width: '100%' }}>
                <Box sx={{ flex: 1, minWidth: 0 }}>
                  <CubeTypeahead
                    fetchOptions={true}
                    tenantId={tenantId}
                    datasourceId={datasourceId}
                    pageSize={100}
                    options={primaryCube ? [primaryCube] : []}
                    value={primaryCube}
                    disabled={!hasTenantScope}
                    onChange={(_, newValue: any) => {
                      if (newValue) {
                        setPrimaryCube(newValue);
                        setViewData({
                          ...viewData,
                          cubes: [{ id: newValue.id, name: newValue.display_name || newValue.model_key }]
                        });
                      }
                    }}
                    placeholder={hasTenantScope ? '' : 'Select tenant & datasource'}
                  />
                </Box>
              </Box>
            )}
          </Grid>

          <Grid item xs={12} md={6}>
            <TextField
              label="Description"
              value={viewData.description || ''}
              onChange={(e) => setViewData({ ...viewData, description: e.target.value })}
              fullWidth
              multiline
              minRows={4}
              size="small"
              placeholder="Human-readable description of this view"
            />
          </Grid>
        </Grid>
      )}

      {/* Components tab */}
      {tabIndex === 1 && (
        <Box>
          <Box
            sx={{
              display: 'flex',
              gap: 2,
              mb: 2,
              alignItems: 'center',
              position: 'sticky',
              top: 0,
              zIndex: 10,
              backgroundColor: 'background.paper',
              paddingTop: 1,
              paddingBottom: 1,
              borderBottom: 1,
              borderColor: (leftStuck || rightStuck) ? 'secondary.main' : 'transparent',
              boxShadow: (leftStuck || rightStuck) ? '0 8px 20px rgba(2,6,23,0.10)' : 'none',
              transition: 'border-color 140ms ease, box-shadow 140ms ease',
            }}
          >
            <TextField
              size="small"
              placeholder="Search components..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              InputProps={{ startAdornment: <Search sx={{ mr: 1, color: 'text.secondary' }} /> }}
              aria-label="Search components"
              sx={{ flex: 1, minWidth: 0 }}
            />

            <Select
              size="small"
              value={sourceFilter}
              onChange={(e) => setSourceFilter(String(e.target.value))}
              displayEmpty
              inputProps={{ 'aria-label': 'Filter sources' }}
              sx={{ minWidth: 200 }}
            >
              <MenuItem value="all">All Sources</MenuItem>
              {enrichedAvailableSources.map((s: any) => (
                <MenuItem key={s.id} value={s.id}>{s.name}</MenuItem>
              ))}
            </Select>
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, ml: 1 }}>
              {/* transient inline saved indicator */}
              {savedIndicatorVisible ? (
                <Typography variant="body2" color="success.main" aria-live="polite">Saved</Typography>
              ) : null}
              {/* local draft indicator */}
              {hasDraft ? (
                <Tooltip title="A local draft is available" aria-label="Local draft available">
                  <Typography variant="body2" color="warning.main">Draft</Typography>
                </Tooltip>
              ) : null}
            </Box>
          </Box>

          <Box sx={{ display: 'flex', gap: 2, alignItems: 'stretch', height: '60vh' }}>
            <Box sx={{ flex: 1, minWidth: 0, display: 'flex', flexDirection: 'column' }}>
              <AvailableComponentsPanel
                filteredAvailableSources={filteredAvailableSources}
                items={flattenedAvailableItems}
                selectedAvailableItems={selectedAvailableItems}
                onItemClick={handleAvailableItemClick}
                highlightMap={highlightMap}
                searchQuery={searchQuery}
                onSearchChange={(v: string) => setSearchQuery(v)}
                sourceFilter={sourceFilter}
                onSourceFilterChange={(v: string) => setSourceFilter(v)}
                availableSourceSummaries={availableSourceSummaries}
                  onScrollStuckChange={(stuck: boolean) => setLeftStuck(stuck)}
                  active={leftStuck}
                  onInfoClick={openInfoModal}
                  typeFilter={typeFilter}
                  setTypeFilter={setTypeFilter}
                  availableTotals={availableTotals}
              />
            </Box>

            <Box
              sx={{
                width: 220,
                display: 'flex',
                flexDirection: 'column',
                alignItems: 'stretch',
                justifyContent: 'flex-start',
                gap: 1,
                zIndex: 2,
                p: 1,
              }}
            >
              <Box sx={{ display: 'flex', justifyContent: 'center', mt: 2 }}>
                <Tooltip title="Add selected components" placement="right">
                  <span>
                    <IconButton
                      color="primary"
                      size="medium"
                      disabled={selectedAvailableItems.size === 0}
                      onClick={handleAddSelectedItems}
                    >
                      <ArrowForwardIcon />
                    </IconButton>
                  </span>
                </Tooltip>
              </Box>

              <Box sx={{ display: 'flex', justifyContent: 'center' }}>
                <Tooltip title="Remove selected view components" placement="left">
                  <span>
                    <IconButton
                      color="primary"
                      size="medium"
                      disabled={selectedViewItems.size === 0}
                      onClick={handleRemoveSelectedViewItems}
                    >
                      <ArrowBackIcon />
                    </IconButton>
                  </span>
                </Tooltip>
              </Box>
            </Box>

            <Box sx={{ flex: 1, minWidth: 0, display: 'flex', flexDirection: 'column' }}>
              <ViewComponentsPanel
                items={viewItems}
                dimensionCount={(viewData?.dimensions || []).length}
                measureCount={(viewData?.measures || []).length}
                selectedItems={selectedViewItems}
                onItemClick={handleViewItemClick}
                targetHighlightMap={targetHighlightMap}
                availableSources={availableSourceSummaries}
                searchQuery={searchQuery}
                onSearchChange={(v: string) => setSearchQuery(v)}
                sourceFilter={sourceFilter}
                onSourceFilterChange={(v: string) => setSourceFilter(v)}
                onScrollStuckChange={(stuck: boolean) => setRightStuck(stuck)}
                active={rightStuck}
                onEditClick={openEditModal}
              />
            </Box>
          </Box>
        </Box>
      )}

      {/* Code tab */}
      {tabIndex === 2 && (
        <Box>
          <ViewCodeEditor viewData={viewData} setViewData={setViewData} />
        </Box>
      )}

      {/* Reusable modal for available info (read-only) and view edits (editable) */}
      <Dialog open={Boolean(editorOpen)} onClose={closeEditor} fullWidth maxWidth="sm">
        <DialogTitle>{editorMode === 'edit' ? 'Edit Component' : 'Component Info'}</DialogTitle>
        <DialogContent>
          <Stack spacing={2} sx={{ mt: 1 }}>
            <TextField
              label="Title"
              value={editorItem?.title || editorItem?.name || ''}
              InputProps={{ readOnly: editorMode === 'view' }}
              onChange={(e) => setEditorItem((prev: any) => ({ ...prev, title: e.target.value }))}
              fullWidth
              size="small"
            />
            <TextField
              label="Description"
              value={editorItem?.description || ''}
              InputProps={{ readOnly: editorMode === 'view' }}
              onChange={(e) => setEditorItem((prev: any) => ({ ...prev, description: e.target.value }))}
              fullWidth
              size="small"
              multiline
              minRows={2}
            />
            <TextField
              label="Format"
              value={editorItem?.format || ''}
              InputProps={{ readOnly: editorMode === 'view' }}
              onChange={(e) => setEditorItem((prev: any) => ({ ...prev, format: e.target.value }))}
              fullWidth
              size="small"
            />
            <TextField
              label="Alias"
              value={editorItem?.alias || ''}
              InputProps={{ readOnly: editorMode === 'view' }}
              onChange={(e) => setEditorItem((prev: any) => ({ ...prev, alias: e.target.value }))}
              fullWidth
              size="small"
            />
            <TextField
              label="Meta (JSON)"
              value={
                editorItem?.meta
                  ? typeof editorItem.meta === 'string'
                    ? editorItem.meta
                    : JSON.stringify(editorItem.meta, null, 2)
                  : ''
              }
              InputProps={{ readOnly: editorMode === 'view' }}
              onChange={(e) => {
                const v = e.target.value;
                try {
                  const parsed = JSON.parse(v || '{}');
                  setEditorItem((prev: any) => ({ ...prev, meta: parsed }));
                  setEditorMetaError(null);
                } catch (err) {
                  // keep the raw text while showing an error
                  setEditorItem((prev: any) => ({ ...prev, meta: v }));
                  setEditorMetaError('Invalid JSON');
                }
              }}
              error={Boolean(editorMetaError)}
              helperText={editorMetaError || ''}
              fullWidth
              size="small"
              multiline
              minRows={3}
            />
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button onClick={closeEditor}>Cancel</Button>
          {editorMode === 'edit' ? (
            <Button
              onClick={() => saveEditor({ title: editorItem?.title, description: editorItem?.description, format: editorItem?.format, alias: editorItem?.alias, meta: editorItem?.meta })}
              variant="contained"
              disabled={Boolean(editorMetaError)}
            >
              Save
            </Button>
          ) : null}
        </DialogActions>
      </Dialog>
      {/* In-app navigation prompt dialog (Save / Discard / Cancel) */}
      <Dialog open={Boolean(showNavPrompt)} onClose={() => confirmNavigate('cancel')} maxWidth="xs" fullWidth>
        <DialogTitle>Unsaved changes</DialogTitle>
        <DialogContent>
          <Typography>There are unsaved changes. What would you like to do?</Typography>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => confirmNavigate('cancel')}>Cancel</Button>
          <Button onClick={() => confirmNavigate('discard')} color="inherit">Discard</Button>
          <Button onClick={() => confirmNavigate('save')} variant="contained" disabled={saving}>{saving ? <CircularProgress size={16} color="inherit" sx={{ mr: 1 }} /> : null} Save</Button>
        </DialogActions>
      </Dialog>

      {/* Restore draft prompt */}
      <Dialog open={Boolean(showRestoreDraftPrompt)} onClose={() => setShowRestoreDraftPrompt(false)} maxWidth="sm" fullWidth>
        <DialogTitle id="restore-draft-title">Restore local draft?</DialogTitle>
        <DialogContent>
          <Typography id="restore-draft-desc">A local draft for this view was found. Would you like to restore your draft (keep local changes) or discard it and keep the server version?</Typography>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => { try { localStorage.removeItem(draftKey); setHasDraft(false); } catch (e) {} setShowRestoreDraftPrompt(false); }}>Discard Draft</Button>
          <Button onClick={() => {
            try {
              const draft = localStorage.getItem(draftKey);
              if (draft) setViewData(JSON.parse(draft));
            } catch (e) {}
            setShowRestoreDraftPrompt(false);
          }} variant="contained">Restore Draft</Button>
        </DialogActions>
      </Dialog>

      {/* Conflict resolution dialog */}
      <Dialog open={Boolean(conflictOpen)} onClose={() => setConflictOpen(false)} maxWidth="md" fullWidth>
        <DialogTitle id="conflict-title">Save conflict</DialogTitle>
        <DialogContent>
          <Typography id="conflict-desc" sx={{ mb: 2 }}>The server has newer changes for this view. Choose how to proceed:</Typography>
          <Box sx={{ display: 'flex', gap: 2 }}>
            <Box sx={{ flex: 1 }}>
              <Typography variant="subtitle2">Server version</Typography>
              <Box component="pre" sx={{ whiteSpace: 'pre-wrap', maxHeight: 240, overflow: 'auto', bgcolor: 'background.paper', p: 1, borderRadius: 1 }}>
                {conflictServerSnapshot ? JSON.stringify(conflictServerSnapshot, null, 2) : 'No server details provided.'}
              </Box>
            </Box>
            <Box sx={{ flex: 1 }}>
              <Typography variant="subtitle2">Your local changes</Typography>
              <Box component="pre" sx={{ whiteSpace: 'pre-wrap', maxHeight: 240, overflow: 'auto', bgcolor: 'background.paper', p: 1, borderRadius: 1 }}>
                {JSON.stringify(viewData || {}, null, 2)}
              </Box>
            </Box>
          </Box>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => { setConflictOpen(false); setConflictServerSnapshot(null); }}>Cancel</Button>
          <Button onClick={() => { applyServerSnapshot(); }} color="inherit">Load Server</Button>
          <Button onClick={() => { forceKeepLocal(); }} variant="contained">Keep Local & Force Save</Button>
        </DialogActions>
      </Dialog>

      <Snackbar open={Boolean(saveMessage)} autoHideDuration={4000} onClose={() => setSaveMessage(null)}>
        <Alert severity={saveMessage?.severity || 'info'} onClose={() => setSaveMessage(null)} sx={{ width: '100%' }}>{saveMessage?.text || ''}</Alert>
      </Snackbar>
    </>
  );
};