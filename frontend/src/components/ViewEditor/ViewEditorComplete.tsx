/* eslint-disable @typescript-eslint/no-unused-vars */
import React, { useState, useEffect, useRef, useCallback, useMemo } from 'react';
import { devDebug } from '../../utils/devLogger';
import { Box } from '@mui/material';
import {
  useAvailableSources,
  useExtendsOptions,
  useAvailableCubes,
  ViewHeader,
  PropertiesSection,
  type ExtendsOption,
} from './index';

interface ViewEditorCompleteProps {
  viewName: string;
  viewData: any;
  setViewData: (data: any) => void;
  onSave: () => void;
  onValidate: () => void;
  isSaving: boolean;
  isValidating: boolean;
  validationResult: any;
  tenantId?: string;
  datasourceId?: string;
}



const ViewEditorComplete: React.FC<ViewEditorCompleteProps> = ({
  viewName,
  viewData,
  setViewData,
  onSave,
  onValidate,
  isSaving,
  isValidating,
  validationResult,
  tenantId,
  datasourceId
}) => {
  const [propertiesExpanded] = useState(true);
  // componentsExpanded state removed — individual sections handle their own expansion
  const [primaryCube, setPrimaryCube] = useState<any | null>(null);
  const [isEditingPrimaryCube, setIsEditingPrimaryCube] = useState(true);

  // Use the modular hooks for data fetching
  const isValidTenantScope = useCallback(() => {
    const valid = !tenantId || !datasourceId ? false : (() => {
      // Basic UUID format validation
      const uuidRegex = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;
      return uuidRegex.test(tenantId) && uuidRegex.test(datasourceId);
    })();
    return valid;
  }, [tenantId, datasourceId]);

  // View properties state - keep local state for UI interactions
  const [extendsInputValue, setExtendsInputValue] = useState<string>((viewData?.extends as string) || '');
  const [extendsQuery, setExtendsQuery] = useState<string>((viewData?.extends as string) || '');
  // Internal refs used by typeahead/enrichment logic moved into the typeahead hooks
  // Keep an explicit selected option/value so we can force the Autocomplete to show
  // the friendly option object (name/title) when we resolve a UUID via fetch.
  const [selectedExtendsValue, setSelectedExtendsValue] = useState<ExtendsOption | string | null>(null);
  const [_resolvingExtendsSingle, _setResolvingExtendsSingle] = useState(false);

  // Available sources and items state moved into PropertiesSection's Components tab

  // Search and cube management state removed — panes were consolidated into tabs
  const hasTenantScope = Boolean(tenantId && datasourceId);

  // Keep a ref to the latest viewData so callbacks can read the newest values
  const viewDataRef = useRef(viewData);
  useEffect(() => { viewDataRef.current = viewData; }, [viewData]);
  
  const pushDebugLog = (_msg: string) => {
    // ViewEditor debug logging disabled. Preserve function for future re-enable.
    // const entry = { when: Date.now(), msg };
    // setDebugLog(prev => [entry, ...prev].slice(0, 200));
    // try { (window as any).__viewEditor_debugLog = (window as any).__viewEditor_debugLog || []; (window as any).__viewEditor_debugLog.unshift(entry); } catch (e) {}
    return;
  };
  
  

  // Debounce user typing into the extends input
  useEffect(() => {
    const t = setTimeout(() => {
      setExtendsQuery(extendsInputValue || '');
    }, 250);
    return () => clearTimeout(t);
  }, [extendsInputValue]);

  // Build a memoized set of selected references (cubes/join_paths/extends) for quick checks
  const selectedRefs = useMemo(() => {
    const s = new Set<string>();

    const addRef = (value?: string | null) => {
      if (!value) return;
      const normalized = value.trim().toLowerCase();
      if (normalized) {
        s.add(normalized);
        // Also add stripped variants to match model_key/display_name variants
        const stripped = normalized
          .replace(/\s*\(custom\)\s*/gi, '')
          .replace(/\s*\(core\)\s*/gi, '')
          .replace(/^\/public\//, '')
          .replace(/^\//, '')
          .trim();
        if (stripped && stripped !== normalized) s.add(stripped);
      }
    };
    
    // Add cubes
    const cubesRefList = Array.isArray(viewData?.cubes) ? viewData.cubes : [];
    cubesRefList.forEach((c: any) => {
      if (typeof c === 'string') {
        addRef(c);
      } else if (c && typeof c === 'object') {
        addRef(c.id ? String(c.id) : undefined);
        addRef(c.model_key ? String(c.model_key) : undefined);
        addRef(c.name ? String(c.name) : undefined);
      }
    });
    
    // Add join paths
    const joinPathsList = Array.isArray(viewData?.join_paths) ? viewData.join_paths : [];
    joinPathsList.forEach((jp: any) => {
      if (typeof jp === 'string') {
        addRef(jp);
      } else if (jp && typeof jp === 'object') {
        addRef(jp.id ? String(jp.id) : undefined);
        addRef(jp.path ? String(jp.path) : undefined);
        addRef(jp.label ? String(jp.label) : undefined);
      }
    });
    
    // Add extends
    if (typeof viewData?.extends === 'string' && viewData.extends.trim()) {
      addRef(viewData.extends);
    }
    
  // expose computed refs to window for manual inspection in devtools
  try { (window as any).__viewEditorSelectedRefs = Array.from(s); } catch (e) {}
    return s;
  }, [viewData?.cubes, viewData?.join_paths, viewData?.extends]);

  const extendsOptionsData = useExtendsOptions(isValidTenantScope, tenantId, datasourceId, viewName, viewData);
  const { extendsOptions, extendsLoading, fetchExtendsOptions: fetchExtendsOptionsFromHook } = extendsOptionsData;
  const availableSourcesData = useAvailableSources(isValidTenantScope, tenantId, datasourceId, viewData, selectedRefs);
  const { fetchAvailableSources } = availableSourcesData;
  const availableCubesData = useAvailableCubes(isValidTenantScope, tenantId, datasourceId, viewData);

  useEffect(() => {
    const currentCube = viewData.cubes?.[0];
    if (currentCube) {
      const cubeId = typeof currentCube === 'object' ? currentCube.id : currentCube;
      // Find the full cube details from availableCubes
      const cubeDetails = availableCubesData.availableCubes.find((c: any) => c.id === cubeId);
      setPrimaryCube(cubeDetails || currentCube);
      setIsEditingPrimaryCube(false);
    } else {
      setPrimaryCube(null);
      setIsEditingPrimaryCube(true);
    }
  }, [viewData.cubes, availableCubesData.availableCubes]);

  // Load data when tenant/datasource or the extends query changes.
  // The hooks will automatically fetch data when their dependencies change
  useEffect(() => {
    if (!isValidTenantScope()) {
      // Data will be cleared by the hooks when scope becomes invalid
      return;
    }

    // Trigger available sources fetch
    fetchAvailableSources();

    // Trigger extends options search when query changes
    if (extendsQuery) {
      fetchExtendsOptionsFromHook(extendsQuery);
    }
  }, [tenantId, datasourceId, extendsQuery, selectedRefs]);

  // Auto-resolve a typed extends name/title to the option id (UUID) when an exact match exists
  useEffect(() => {
    if (!extendsInputValue || !extendsOptions || extendsOptions.length === 0) return;
    const isUuid = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(extendsInputValue.trim());
    if (isUuid) return; // already a UUID

    const match = extendsOptions.find(opt => {
      const name = (opt.name || '').toLowerCase();
      const title = (opt.title || '').toLowerCase();
      const input = extendsInputValue.trim().toLowerCase();
      return name === input || title === input;
    });

    if (match && match.id && String(viewData?.extends) !== String(match.id)) {
      // convert to UUID-based extends
      const idStr = String(match.id);
      setViewData({ ...viewData, extends: idStr });
      // Don't update input here - let UUID resolution handle showing friendly names
    }
  }, [extendsOptions, extendsInputValue, viewData?.extends]); // Removed setViewData from deps to avoid cycles

  // When viewData.extends changes, update the input value to show friendly names
  useEffect(() => {
    const rawExtends = typeof viewData?.extends === 'string' ? viewData.extends.trim() : '';
    if (!rawExtends) return;

    const key = rawExtends.toLowerCase();
    
    // Try to find a matching option from the current extendsOptions
    const matchById = extendsOptions.find(opt => opt.id && String(opt.id).toLowerCase() === key);
    if (matchById) {
      const friendlyName = matchById.name || matchById.title || rawExtends;
      if (extendsInputValue !== friendlyName) {
        setExtendsInputValue(friendlyName);
      }
      if (!selectedExtendsValue || (selectedExtendsValue as any)?.id !== matchById.id) {
        setSelectedExtendsValue(matchById);
      }
      return;
    }

    const matchByLabel = extendsOptions.find(opt => (opt.name || '').toLowerCase() === key || (opt.title || '').toLowerCase() === key);
    if (matchByLabel) {
      const friendlyName = matchByLabel.name || matchByLabel.title || rawExtends;
      if (extendsInputValue !== friendlyName) {
        setExtendsInputValue(friendlyName);
      }
      if (!selectedExtendsValue || (selectedExtendsValue as any)?.id !== matchByLabel.id) {
        setSelectedExtendsValue(matchByLabel);
      }
      return;
    }

    // If no match found, just ensure the input shows the raw value
    if (extendsInputValue !== rawExtends) {
      setExtendsInputValue(rawExtends);
    }
  }, [viewData?.extends, extendsOptions, extendsInputValue]);

  // Filtering and selection helpers for the former side panels removed —
  // components now live inside `PropertiesSection` tabs and manage their own state.

  // Previously some side-panel variables/handlers were referenced here as
  // a short-lived workaround to keep the linter quiet while panels were
  // being refactored into tabs. Those temporary references have been removed
  // as the tabs implementation is now permanent.

  // Cube selection handled inline via primary cube controls; bulk-add removed

  // Ensure loaded viewData uses canonical shapes: cubes as [{id,name}], join_paths as [{id,path,label}], extends as UUID
  useEffect(() => {
    if (!viewData) return;

    let changed = false;
    const next = { ...viewData };

    // Normalize cubes
    const cubesArr = Array.isArray(next.cubes) ? next.cubes : [];
    const normCubes = cubesArr.map((c: any) => {
      if (!c) return c;
      if (typeof c === 'string') {
        // treat as id
        const found = availableCubesData.availableCubes.find(ac => String(ac.id) === String(c));
        changed = true;
        return { id: c, name: found?.display_name || found?.model_key || c };
      }
      // object
      if (c.id && !c.name) {
        const found = availableCubesData.availableCubes.find(ac => String(ac.id) === String(c.id));
        if (found) { changed = true; return { ...c, name: found.display_name || found.model_key }; }
      }
      return c;
    });
    next.cubes = normCubes;

    // Normalize join_paths
    const jpArr = Array.isArray(next.join_paths) ? next.join_paths : [];
    const normJp = jpArr.map((jp: any) => {
      if (!jp) return jp;
      if (typeof jp === 'string') {
        // try to find cube by model_key or id
        const found = availableCubesData.availableCubes.find(ac => String(ac.model_key) === String(jp) || String(ac.id) === String(jp));
        changed = true;
  return { id: found?.id || jp, path: jp, label: found?.display_name || pjLabel(jp), is_core: Boolean(found?.is_core), is_custom: Boolean(found?.is_custom) };
      }
      // object already
      if (jp.id && !jp.path) {
        changed = true;
        return { id: jp.id, path: jp.path || jp.label || String(jp.id), label: jp.label || String(jp.id), is_core: Boolean(jp.is_core), is_custom: Boolean(jp.is_custom) };
      }
      return jp;
    });
    next.join_paths = normJp;

    // Normalize extends: if extends is a name, attempt to find option id from extendsOptions
    if (next.extends && typeof next.extends === 'string') {
      const isUuid = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(next.extends);
      if (!isUuid) {
        const found = extendsOptions.find(opt => (opt.name || '').toLowerCase() === String(next.extends).toLowerCase() || (opt.title || '').toLowerCase() === String(next.extends).toLowerCase());
        if (found && found.id) {
          next.extends = found.id;
          changed = true;
        }
      }
    }

    if (changed) {
      setViewData(next);
    }
  }, [viewData, availableCubesData.availableCubes, extendsOptions, setViewData]);

  function pjLabel(p: string) { return p; }

  // collapse/expand removed: sources are always rendered expanded in the new UI

  // The functions for selecting/adding/removing items from external side-panels
  // have been moved into the PropertiesSection Components tab. Keeping this
  // file focused on rendering the editor and header.

  const handleExtendsSelection = (value: any) => {
    devDebug('[ViewEditorComplete] handleExtendsSelection invoked with:', value);
    // If an option object with an id is selected, store the id (UUID) as the extends value
    if (value && typeof value === 'object' && value.id) {
      const idStr = String(value.id);
      // Keep the resolved object in the parent so the Autocomplete `value` becomes
      // the object (friendly name) while we still preserve canonical id for saves.
      setViewData({ ...viewData, extends: value });
      setExtendsInputValue(value.name || idStr);
      setExtendsQuery(idStr);
      setSelectedExtendsValue(value);
      try { (window as any).__viewEditor_lastExtendsSelection = { when: Date.now(), id: idStr, name: value.name }; } catch (e) {}
      try { pushDebugLog(`handleExtendsSelection: applied extends ${idStr}`); } catch (e) {}
      // Immediately refresh available sources so the newly extended view's items appear
      // Use a short timeout to allow React state to flush; fetchAvailableSources reads viewDataRef
      setTimeout(() => {
        try { fetchAvailableSources(); } catch (e) { /* ignore */ }
      }, 50);
      return;
    }

    const stringValue = typeof value === 'string' ? value : value?.name || '';
    setViewData({ ...viewData, extends: stringValue });
    setExtendsInputValue(stringValue);
    setExtendsQuery(stringValue);
    // If the user cleared the selection, also clear the local selected value so
    // the Autocomplete shows the empty input instead of reverting to the old UUID.
    if (!stringValue) {
      setSelectedExtendsValue(null);
    }
  };

  const handleSave = () => {
    // If the PropertiesSection registered a local save function, call it first.
    if (typeof registeredSave === 'function') {
      try { registeredSave(); return; } catch (e) { /* fallthrough */ }
    }
    onSave();
  };

  // allow PropertiesSection to register its save handler so the header Save can call it
  const [registeredSave, setRegisteredSave] = useState<(() => Promise<any | void>) | null>(null);
  // stable register function to avoid re-creating on every render
  const registerSave = React.useCallback((saveFn: () => Promise<any | void>) => {
    setRegisteredSave(() => saveFn);
  }, [setRegisteredSave]);

  const handleValidate = () => {
    onValidate();
  };

  // validationIcon is rendered inside ViewHeader; no local usage here

  return (
    <Box sx={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
      <ViewHeader
        viewName={viewName}
        viewData={viewData}
        onSave={handleSave}
        onValidate={handleValidate}
        isSaving={isSaving}
        isValidating={isValidating}
        validationResult={validationResult}
      />

  <PropertiesSection
    propertiesExpanded={propertiesExpanded}
    viewData={viewData}
    setViewData={setViewData}
    primaryCube={primaryCube}
    setPrimaryCube={setPrimaryCube}
    isEditingPrimaryCube={isEditingPrimaryCube}
    setIsEditingPrimaryCube={setIsEditingPrimaryCube}
    extendsOptions={extendsOptions}
    extendsLoading={extendsLoading}
    extendsInputValue={extendsInputValue}
    setExtendsInputValue={setExtendsInputValue}
    selectedExtendsValue={selectedExtendsValue}
    handleExtendsSelection={handleExtendsSelection}
    hasTenantScope={hasTenantScope}
    tenantId={tenantId}
    datasourceId={datasourceId}
    onRegisterSave={registerSave}
  />

      {/* Components UI moved into the PropertiesSection Components tab; external panels removed */}
    </Box>
  );
};

export default ViewEditorComplete;

