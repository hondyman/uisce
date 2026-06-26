import { useEffect, useMemo, useState, useRef } from 'react';
import useBlockableNavigate from '../../../components/RouteBlocker/useBlockableNavigate';
// API helpers used elsewhere
import getErrorMessage from '../../../utils/errors';
import styles from './ViewsCatalogPage.module.css';
import { devLog } from '../../../utils/devLogger';
import resolveApiUrl from '../../../utils/resolveApiUrl';
import { BundleForm } from '../../../types/bundles';
import { View, getViewExtendsDisplay } from '../../../types/views';
import { useTenant } from '../../../contexts/TenantContext';
import { useNotification } from '../../../hooks/useNotification';
import TextPromptDialog from '../../../components/TextPromptDialog';
import { useConfirm } from '../../../components/ConfirmProvider';
import { useAuth } from '../../../contexts/AuthContext';
import { BundleEditor } from './BundleEditor'; // Import the new component (local to pages)


type ViewItem = View;

type Bundle = BundleForm;

// Minimal JSON -> YAML converter (covers common primitives/arrays/objects)
function toYAML(value: any, indent = 0): string {
  const sp = '  '.repeat(indent);
  if (value === null || value === undefined) return 'null';
  if (typeof value === 'number' || typeof value === 'boolean') return String(value);
  if (typeof value === 'string') {
    // Quote if contains special chars
    if (/[:\-?{}\[\],&*!|>'"%@`\n]/.test(value)) {
      return JSON.stringify(value);
    }
    return value;
  }
  if (Array.isArray(value)) {
    if (value.length === 0) return '[]';
    return value.map(v => `${sp}- ${toYAML(v, indent + 1).replace(/^\s+/, '')}`).join('\n');
  }
  if (typeof value === 'object') {
    const keys = Object.keys(value);
    if (keys.length === 0) return '{}';
    return keys.map(k => {
      const v = (value as any)[k];
      const rendered = toYAML(v, indent + 1);
      if (typeof v === 'object' && v !== null && !Array.isArray(v)) {
        return `${sp}${k}:\n${rendered}`;
      }
      if (Array.isArray(v)) {
        const lines = rendered.split('\n').map((ln, i) => (i === 0 ? ln : ln));
        return `${sp}${k}: ${lines.shift()}` + (lines.length ? `\n${lines.join('\n')}` : '');
      }
      return `${sp}${k}: ${rendered}`;
    }).join('\n');
  }
  return String(value);
}

const ViewsCatalogPage: React.FC = () => {
  const { tenant, datasource, isSelected } = useTenant();
  const { isCoreAdmin, canManageCustomAssets } = useAuth();
  const canManageCore = isCoreAdmin();
  const canManageCustom = canManageCustomAssets();
  const isCoreView = (view: ViewItem | undefined): boolean => {
    if (!view) return false;
    if (typeof view.is_core === 'boolean') return view.is_core;
    if (typeof view.isCore === 'boolean') return view.isCore;
    const tags = Array.isArray(view.tags) ? view.tags : [];
    if (tags.some(tag => typeof tag === 'string' && tag.toLowerCase() === 'core')) {
      return true;
    }
    if (typeof view.name === 'string') {
      return view.name.toLowerCase().startsWith('core_');
    }
    return false;
  };
  const canMutateView = (view: ViewItem | undefined): boolean => {
    if (!view) {
      return canManageCustom;
    }
    return isCoreView(view) ? canManageCore : canManageCustom;
  };
  const navigate = useBlockableNavigate();
  const tenantId = tenant?.id || '';
  const datasourceId = (datasource as any)?.id || (datasource as any)?.tenant_instance_id || '';
  const [items, setItems] = useState<ViewItem[]>([]);
  const [viewsLookup, setViewsLookup] = useState<Record<string, ViewItem>>({});
  // track pending fetches and permanently-missing extends targets to avoid refetch storms
  const pendingExtendsFetch = useRef<Set<string>>(new Set());
  const missingExtends = useRef<Set<string>>(new Set());
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [q, setQ] = useState('');
  const [qDebounced, setQDebounced] = useState('');
  const debounceTimer = useRef<number | null>(null);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(25);
  const [total, setTotal] = useState(0);
  const [listETag, setListETag] = useState<string | null>(null);
  // Compare modal state
  const [compareSelection, setCompareSelection] = useState<string[]>([]);
  const [compareOpen, setCompareOpen] = useState(false);
  const notification = useNotification();
  const confirm = useConfirm();
  const [namePrompt, setNamePrompt] = useState<{ open: boolean; title: string; defaultValue?: string; onSubmit?: (v: string) => void } | null>(null);
  const [compareViewsData, setCompareViewsData] = useState<{left?: any; right?: any}>({});

  const [bundleEditorOpen, setBundleEditorOpen] = useState(false);
  const fetchList = async () => {
    setLoading(true);
    setError(null);
    try {
      if (!isSelected) {
        setItems([]);
        setTotal(0);
        setLoading(false);
      }
  const u = new URL(resolveApiUrl('/api/views'));
      if (qDebounced) u.searchParams.set('q', qDebounced);
      if (tenantId) u.searchParams.set('tenant_id', tenantId);
      if (datasourceId) u.searchParams.set('tenant_instance_id', String(datasourceId));
      u.searchParams.set('page', String(page));
      u.searchParams.set('page_size', String(pageSize));
      const res = await fetch(u.toString(), {
        headers: listETag ? { 'If-None-Match': listETag } : undefined,
      });
      if (res.status === 304) { setLoading(false); return; }
      if (!res.ok) throw new Error(`${res.status} ${res.statusText}`);
      const tag = res.headers.get('ETag');
      if (tag) setListETag(tag);
      const data = await res.json();
      const rawViews: ViewItem[] = Array.isArray(data.views) ? data.views : [];
  // build a lookup by id and name to help resolve extends -> display name
  const lookupById: Record<string, any> = {};
  const lookupByName: Record<string, any> = {};
  rawViews.forEach((rv: any) => { if (rv?.id) lookupById[String(rv.id).toLowerCase()] = rv; if (rv?.name) lookupByName[String(rv.name).toLowerCase()] = rv; });
      const normalizedViews: ViewItem[] = rawViews.map((view: any) => {
        const tags = Array.isArray(view.tags) ? view.tags : [];
        const candidate: ViewItem = { ...view, tags };
        const core = isCoreView(candidate);
        // compute a human-friendly extends display value: prefer the target view's title or name when possible
        let extends_display: string | undefined = undefined;
        try {
          const ext = view?.extends;
          if (ext && typeof ext === 'string') {
            const lowered = ext.toLowerCase();
            const target = lookupById[lowered] || lookupByName[lowered];
            if (target) extends_display = target.title || target.name || String(ext);
            else extends_display = ext;
          }
        } catch (e) { /* ignore */ }
        return { ...candidate, is_core: core, isCore: core, extends_display };
      });
  setItems(normalizedViews);
  // store a merged lookup for render-time resolution
  setViewsLookup({ ...lookupById, ...lookupByName });
      setTotal(data.total || 0);
    } catch (e: unknown) {
      setError(getErrorMessage(e, 'Failed to load views'));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    void fetchList();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [qDebounced, page, pageSize, tenantId, datasourceId]);

  // Effect: detect any view.extends values that are raw identifiers not present in viewsLookup
  // and fetch those views on-demand to resolve a human-friendly title/name for display.
  useEffect(() => {
    if (!isSelected) return;
    // collect original extends values from items
    const extMap: Record<string, string> = {};
    items.forEach(v => {
      const ext = (v as any)?.extends;
      if (ext && typeof ext === 'string') {
        extMap[String(ext).toLowerCase()] = ext;
      }
    });
    const toFetch: string[] = [];
    Object.keys(extMap).forEach(key => {
      if (pendingExtendsFetch.current.has(key)) return; // already fetching
      if (missingExtends.current.has(key)) return; // already known missing
      // if we already have a lookup for this key (id or name), skip
      if (viewsLookup && (viewsLookup as any)[key]) return;
      toFetch.push(key);
    });
    if (toFetch.length === 0) return;
    // build original value map for encoded identifiers
    const origMap = extMap;
    // mark pending
    toFetch.forEach(k => pendingExtendsFetch.current.add(k));

    // perform fetches in parallel (safe, small number expected). Merge successful results into viewsLookup.
    (async () => {
      try {
        const results = await Promise.allSettled(toFetch.map(async (key) => {
          const orig = origMap[key] || key;
          const identifier = encodeURIComponent(orig);
          const urlObj = new URL(resolveApiUrl(`/api/views/${identifier}`));
          if (tenantId) urlObj.searchParams.set('tenant_id', tenantId);
          if (datasourceId) urlObj.searchParams.set('tenant_instance_id', String(datasourceId));
          const res = await fetch(urlObj.toString());
          if (!res.ok) {
            const txt = await res.text().catch(() => res.statusText || '');
            throw new Error(`${res.status} ${res.statusText} ${txt}`);
          }
          const data = await res.json();
          const view = data.view ?? data;
          return { key, view } as { key: string; view: any };
        }));

        let mergedAny = false;
        setViewsLookup(prev => {
          const copy: Record<string, any> = { ...prev };
          results.forEach((r: any) => {
            if (r.status === 'fulfilled' && r.value && r.value.view) {
              const view = r.value.view;
              // add by id and by name (both lowercased) so lookups succeed either way
              if (view.id) copy[String(view.id).toLowerCase()] = view;
              if (view.name) copy[String(view.name).toLowerCase()] = view;
              mergedAny = true;
              // ensure we don't try again
              pendingExtendsFetch.current.delete(String(view.id || view.name).toLowerCase());
            } else if (r.status === 'rejected') {
              // could not fetch this extended view; we'll mark it as missing below
            }
          });
          return copy;
        });

        // For any that were rejected, mark them as permanently missing to avoid retries
        results.forEach((r: any, i: number) => {
          if (r.status === 'rejected') {
            const key = toFetch[i];
            pendingExtendsFetch.current.delete(key);
            missingExtends.current.add(key);
            devLog(`Failed to fetch extended view ${key}:`, r.reason);
          }
        });
        if (mergedAny) {
          devLog('Merged extended views into viewsLookup');
        }
      } catch (e) {
        // clear pending for these keys so we can retry later
        toFetch.forEach(k => pendingExtendsFetch.current.delete(k));
        devLog('Error while fetching extended views', e);
      }
    })();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [items, tenantId, datasourceId, isSelected]);

  // Debounce search input for typeahead
  useEffect(() => {
    if (debounceTimer.current) window.clearTimeout(debounceTimer.current);
    debounceTimer.current = window.setTimeout(() => setQDebounced(q.trim()), 250);
    return () => {
      if (debounceTimer.current) window.clearTimeout(debounceTimer.current);
    };
  }, [q]);

  const totalPages = useMemo(() => Math.max(1, Math.ceil(total / pageSize)), [total, pageSize]);

  const [snack, setSnack] = useState<{open: boolean; message: string; severity: 'success'|'error'|'info'|'warning'}>({open:false,message:'',severity:'info'});

  const openDetail = (v: ViewItem) => {
  const identifier = v.id || v.name;
  if (!identifier) {
    notification.error(`View has no valid identifier. ID: "${v.id}", Name: "${v.name}"`);
    return;
  }
  void navigate(`/views/${identifier}`);
  };

  const download = (v: ViewItem) => {
  const identifier = v.id || v.name;
  const downloadUrl = new URL(resolveApiUrl(`/api/views/${encodeURIComponent(identifier)}/download`));
  if (tenantId) downloadUrl.searchParams.set('tenant_id', tenantId);
  if (datasourceId) downloadUrl.searchParams.set('tenant_instance_id', String(datasourceId));
    const a = document.createElement('a');
    a.href = downloadUrl.toString();
    a.download = `${v.name}.json`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
  };

  const downloadYAML = async (v: ViewItem) => {
    try {
  const identifier = v.id || v.name;
  const urlObj = new URL(resolveApiUrl(`/api/views/${encodeURIComponent(identifier)}`));
  if (tenantId) urlObj.searchParams.set('tenant_id', tenantId);
  if (datasourceId) urlObj.searchParams.set('tenant_instance_id', String(datasourceId));
  const res = await fetch(urlObj.toString());
      if (!res.ok) throw new Error(await res.text());
      const data = await res.json();
      const obj = data.view ?? data;
      const yaml = toYAML(obj);
      const blob = new Blob([yaml], { type: 'text/yaml;charset=utf-8' });
      const a = document.createElement('a');
      a.href = URL.createObjectURL(blob);
      a.download = `${v.name}.yaml`;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(a.href);
    } catch (e) {
      notification.error(`Download YAML failed: ${String(e)}`);
    }
  };

  const copyJSON = async (v: ViewItem) => {
    try {
  const identifier = v.id || v.name;
  const urlObj = new URL(resolveApiUrl(`/api/views/${encodeURIComponent(identifier)}`));
  if (tenantId) urlObj.searchParams.set('tenant_id', tenantId);
  if (datasourceId) urlObj.searchParams.set('tenant_instance_id', String(datasourceId));
  const res = await fetch(urlObj.toString());
      if (!res.ok) throw new Error(await res.text());
      const data = await res.json();
      await navigator.clipboard.writeText(JSON.stringify(data.view ?? data, null, 2));
      setSnack({ open: true, message: 'JSON copied to clipboard', severity: 'success' });
    } catch (e) {
      notification.error(`Copy failed: ${String(e)}`);
    }
  };

  // publishView removed: publishing flow handled elsewhere in UI

  const publishAsBundle = async (v: ViewItem) => {
    const core = isCoreView(v);
    if (core) {
      notification.error('Core views are templates and cannot be published directly. Clone it to a custom view first.');
      return;
    }
    if (!canManageCustom) {
      notification.warning('You need admin access to publish bundles.');
      return;
    }
    try {
      // This would call a new API endpoint: POST /api/bundles
      // The backend would create a versioned, immutable bundle from the view definition.
  devLog(`Publishing view "${v.name}" as a new bundle version...`);
      setSnack({ open: true, message: `Published ${v.name} as new bundle version.`, severity: 'success' });
    } catch (e) {
      notification.error(`Publish failed: ${String(e)}`);
    }
  };

  const handleSaveBundle = async (bundle: Bundle) => {
    try {
      // This would be a real API call, e.g., saveBundle(bundle)
  devLog('Saving bundle:', bundle);
      setSnack({ open: true, message: `Bundle "${bundle.name}" created successfully.`, severity: 'success' });
    } catch (e) {
      setSnack({ open: true, message: `Failed to save bundle: ${String(e)}`, severity: 'error' });
    }
  };

  const toggleCompareSelect = (name: string) => {
    setCompareSelection(prev => {
      const exists = prev.includes(name);
      let next = exists ? prev.filter(n => n !== name) : [...prev, name];
      if (next.length > 2) next = next.slice(next.length - 2); // keep last two
      return next;
    });
  };

  const openCompareModalIfReady = async () => {
    if (compareSelection.length !== 2) {
      setSnack({ open: true, message: 'Select two views to compare', severity: 'info' });
      return;
    }
    try {
      const [a, b] = compareSelection;
    const [ra, rb] = await Promise.all([
  fetch(new URL(resolveApiUrl(`/api/views/${encodeURIComponent(a)}`)).toString()).then(r=>r.json()),
  fetch(new URL(resolveApiUrl(`/api/views/${encodeURIComponent(b)}`)).toString()).then(r=>r.json()),
    ]);
      setCompareViewsData({ left: ra.view ?? ra, right: rb.view ?? rb });
      setCompareOpen(true);
    } catch (e) {
      setSnack({ open: true, message: `Compare failed: ${String(e)}`, severity: 'error' });
    }
  };

  // CRUD helpers for view definitions
  const createNewView = async () => {
    if (!canManageCustom) {
      notification.warning('You need admin access to create new views.');
      return;
    }
    setNamePrompt({ open: true, title: 'Enter new view name (alphanumeric, underscore, dash):', defaultValue: '', onSubmit: async (name) => {
      if (!name) return;
    const safe = /^[A-Za-z0-9_\-]+$/.test(name);
      if (!safe) { notification.error('Invalid name'); return; }
    const base = { name, title: name, description: '', cubes: [], dimensions: [], measures: [], folders: [] };
    try {
  // Use PUT upsert to be compatible with servers that may not implement POST /api/views
  // Always target /api/views/{name} with optional tenant/datasource query params
  const viewUrlObj = new URL(resolveApiUrl(`/api/views/${encodeURIComponent(name)}`));
      if (tenantId) viewUrlObj.searchParams.set('tenant_id', tenantId);
      if (datasourceId) viewUrlObj.searchParams.set('tenant_instance_id', String(datasourceId));
  const viewUrl = viewUrlObj.toString();
      const authToken = typeof localStorage !== 'undefined' ? (localStorage.getItem('ADMIN_API_KEY') || localStorage.getItem('auth_token')) : null;
      const headers: Record<string, string> = { 'Content-Type': 'application/json' };
      if (authToken) headers['Authorization'] = `Bearer ${authToken}`;
  const res = await fetch(viewUrl, {
        method: 'PUT',
        headers,
        body: JSON.stringify(base)
      });
      if (!res.ok) throw new Error(await res.text());
      await fetchList();
  // After creating, open details page (use raw name so the router receives a normal param)
  void navigate(`/views/${name}`);
    } catch (e) { notification.error(`Create failed: ${String(e)}`); }
      } });
  };
  const deleteView = async (v: ViewItem) => {
    const core = isCoreView(v);
    if (core && !canManageCore) {
      notification.warning('Core views are read-only for your role.');
      return;
    }
    if (!core && !canManageCustom) {
      notification.warning('You need admin access to delete custom views.');
      return;
    }
    if (!(await confirm({ title: 'Delete view', description: `Delete view ${v.name}? This action cannot be undone.` }))) return;
    try {
      // Always target /api/views/{identifier} with optional tenant/datasource query params
  const identifier = v.id || v.name;
  const viewUrlObj = new URL(resolveApiUrl(`/api/views/${encodeURIComponent(identifier)}`));
  if (tenantId) viewUrlObj.searchParams.set('tenant_id', tenantId);
  if (datasourceId) viewUrlObj.searchParams.set('tenant_instance_id', String(datasourceId));
  const authToken = typeof localStorage !== 'undefined' ? (localStorage.getItem('ADMIN_API_KEY') || localStorage.getItem('auth_token')) : null;
  const headers: Record<string, string> = {};
  if (authToken) headers['Authorization'] = `Bearer ${authToken}`;
  const res = await fetch(viewUrlObj.toString(), { method: 'DELETE', headers });
      if (!res.ok) throw new Error(await res.text());
      // Optimistically remove from the table for instant feedback
      setItems(prev => prev.filter(i => i.name !== v.name));
      setSnack({ open: true, message: 'View deleted', severity: 'success' });
      // Refresh in background to ensure totals/ETag stay correct
      void fetchList();
    } catch (e) { notification.error(`Delete failed: ${String(e)}`); }
  };

  const cloneView = async (v: ViewItem) => {
    const core = isCoreView(v);
    if (core && !canManageCore) {
      notification.warning('Core views are read-only for your role.');
      return;
    }
    if (!core && !canManageCustom) {
      notification.warning('You need admin access to clone custom views.');
      return;
    }
    try {
  const identifier = v.id || v.name;
  const urlObj = new URL(resolveApiUrl(`/api/views/${encodeURIComponent(identifier)}`));
  if (tenantId) urlObj.searchParams.set('tenant_id', tenantId);
  if (datasourceId) urlObj.searchParams.set('tenant_instance_id', String(datasourceId));
  const res = await fetch(urlObj.toString());
      if (!res.ok) throw new Error(await res.text());
      const data = await res.json();
      const base = data.view ?? data;
      setNamePrompt({ open: true, title: 'Clone as name:', defaultValue: `${v.name}_copy`, onSubmit: async (newName) => {
        if (!newName) return;
      base.name = newName;
  const viewUrlObj2 = new URL(resolveApiUrl(`/api/views/${encodeURIComponent(newName)}`));
      if (tenantId) viewUrlObj2.searchParams.set('tenant_id', tenantId);
      if (datasourceId) viewUrlObj2.searchParams.set('tenant_instance_id', String(datasourceId));
  const viewUrl = viewUrlObj2.toString();
      const authToken = typeof localStorage !== 'undefined' ? (localStorage.getItem('ADMIN_API_KEY') || localStorage.getItem('auth_token')) : null;
      const headers: Record<string, string> = { 'Content-Type': 'application/json' };
      if (authToken) headers['Authorization'] = `Bearer ${authToken}`;
  const put = await fetch(viewUrl, { method: 'PUT', headers, body: JSON.stringify(base) });
      if (!put.ok) throw new Error(await put.text());
      await fetchList();
  void navigate(`/views/${newName}`);
      } });
    } catch (e) {
      notification.error(`Clone failed: ${String(e)}`);
    }
  };

  return (
    <div className={styles.container}>
      <nav className={styles.breadcrumbs} aria-label="Breadcrumb">
        <ol>
          <li><a href="/">Home</a></li>
          <li aria-current="page">Views</li>
        </ol>
      </nav>
      
      <div className={styles.pageHeader}>
        <div>
          <h2 className={styles.pageTitle}>Views Catalog</h2>
          <p className={styles.pageSubtitle}>Manage and explore your semantic views</p>
        </div>
        <div className={styles.pageActions}>
          <button 
            className={styles.primaryIconButton}
            onClick={createNewView}
            disabled={!isSelected || !canManageCustom}
            title="Create new view"
          >
            ➕ New View
          </button>
          <button
            className={styles.secondaryIconButton}
            onClick={() => setBundleEditorOpen(true)}
            disabled={!isSelected || !canManageCustom}
            title="Create new data bundle"
          >
            📦 New Bundle
          </button>
        </div>
      </div>
      {namePrompt && (
        <TextPromptDialog
          open={Boolean(namePrompt.open)}
          title={namePrompt.title}
          defaultValue={namePrompt.defaultValue}
          onClose={() => setNamePrompt(null)}
          onSubmit={(v: string) => {
            setNamePrompt(null);
            namePrompt.onSubmit?.(v);
          }}
        />
      )}
      {!isSelected && (
        <div className={styles.error} role="alert">Select a tenant and datasource (via Connections) to view the scoped catalog.</div>
      )}
      <div className={styles.toolbar}>
        <div className={styles.toolbarLeft}>
          <input
            className={styles.search}
            placeholder="Search views..."
            value={q}
            onChange={(e) => setQ(e.target.value)}
          />
        </div>
        <div className={styles.toolbarRight}>
          <button 
            className={styles.iconButton} 
            title="Compare views" 
            aria-label="Compare views"
            onClick={openCompareModalIfReady} 
            disabled={!isSelected || compareSelection.length !== 2}
          >
            ⚖️ Compare ({compareSelection.length}/2)
          </button>
          <select 
            aria-label="Page size" 
            value={pageSize} 
            onChange={(e) => { setPageSize(parseInt(e.target.value) || 25); setPage(1); }}
            className={styles.pageSize}
          >
            {[10,25,50,100].map(n => <option key={n} value={n}>{n}/page</option>)}
          </select>
          <div className={styles.pagination}>
            <button 
              disabled={page<=1} 
              onClick={() => setPage(p => Math.max(1, p-1))}
              className={styles.iconButton}
            >
              Previous
            </button>
            <span>{page} / {totalPages}</span>
            <button 
              disabled={page>=totalPages} 
              onClick={() => setPage(p => Math.min(totalPages, p+1))}
              className={styles.iconButton}
            >
              Next
            </button>
          </div>
        </div>
      </div>
      {loading && <div>Loading…</div>}
      {error && <div className={styles.error}>{error}</div>}
  {/* compare errors are surfaced via snackbar */}
      <div className={styles.tableWrap}>
        <table className={styles.table} role="grid" aria-label="Views">
          <thead>
            <tr>
              <th>Name</th>
              <th>Title</th>
              <th>Description</th>
              <th>Extends</th>
              <th>Cubes</th>
              <th>Folders</th>
              <th>Modified</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {[...items].sort((a,b)=>a.name.localeCompare(b.name)).map((v) => {
              const canMutate = canMutateView(v);
              return (
              <tr key={v.name} className={styles.row}>
                <td>
                  <label className={styles.rowSelectLabel}>
                    <input type="checkbox" aria-label={`Select ${v.name} for compare`} checked={compareSelection.includes(v.name)} onChange={() => toggleCompareSelect(v.name)} />
                  </label>
                  <button className={styles.rowLink} onClick={() => openDetail(v)}>{v.name}</button>
                </td>
                <td>{v.title || '—'}</td>
                <td className={styles.truncate}>{v.description || '—'}</td>
                <td className={styles.truncate}>{getViewExtendsDisplay(v, viewsLookup) || '—'}</td>
                <td>{v.cube_count ?? '—'}</td>
                <td>{v.folder_count ?? '—'}</td>
                <td>{v.modified_at ? new Date(v.modified_at).toLocaleString() : '—'}</td>
                <td>
                  <div className={styles.rowActions}>
                    <button className={styles.iconButton} title="Edit" onClick={() => openDetail(v)} disabled={!canMutate}>✎</button>
                    <button className={styles.iconDangerButton} title="Delete" onClick={() => void deleteView(v)} disabled={!canMutate}>🗑</button>
                    <div className={styles.menu}>
                      <details>
                        <summary aria-label="Actions" title="Actions" className={styles.hamburgerButton}>
                          <span className={styles.hamburgerIcon} aria-hidden>≡</span>
                        </summary>
                        <div className={styles.menuList} role="menu">
                          <button onClick={() => publishAsBundle(v)} role="menuitem" disabled={!canMutate}>Publish as Bundle</button>
                          <button onClick={() => openDetail(v)} role="menuitem" disabled={!canMutate}>Edit</button>
                          <button onClick={() => cloneView(v)} role="menuitem" disabled={!canMutate}>Clone</button>
                          <button onClick={() => navigator.clipboard.writeText(v.name)} role="menuitem">Copy name</button>
                          <button onClick={() => copyJSON(v)} role="menuitem">Copy JSON</button>
                          <button onClick={() => download(v)} role="menuitem">Download JSON</button>
                          <button onClick={() => downloadYAML(v)} role="menuitem">Download YAML</button>
                        </div>
                      </details>
                    </div>
                  </div>
                </td>
              </tr>
            );
            })}
          </tbody>
        </table>
      </div>
  {/* Compare Modal */}
      {compareOpen && (
        <div className={styles.modalOverlay} role="dialog" aria-modal="true">
          <div className={styles.modal}>
            <div className={styles.modalHeader}>
              <strong>Compare</strong>
              <button className={styles.iconButton} onClick={() => setCompareOpen(false)} aria-label="Close">✕</button>
            </div>
            <div className={styles.modalBodySplit}>
              <div>
                <h4>{compareSelection[0]}</h4>
                <pre className={styles.pre}>{JSON.stringify(compareViewsData.left, null, 2)}</pre>
              </div>
              <div>
                <h4>{compareSelection[1]}</h4>
                <pre className={styles.pre}>{JSON.stringify(compareViewsData.right, null, 2)}</pre>
              </div>
            </div>
          </div>
        </div>
      )}

      <BundleEditor
        open={bundleEditorOpen}
        onClose={() => setBundleEditorOpen(false)}
        onSave={handleSaveBundle}
        existingViews={items}
      />
      {snack.open && (
        <div role="status" aria-live="polite" className={styles.snackbar}>
          {snack.message}
          <button onClick={() => setSnack({ ...snack, open: false })} className={styles.snackbarClose}>Close</button>
        </div>
      )}
    </div>
  );
};

export default ViewsCatalogPage;
