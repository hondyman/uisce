import { useMemo, useState, useEffect, useRef, type FC, type ReactNode, type MouseEvent } from 'react';
import { devLog } from '../../utils/devLogger';
import './SemanticModelOverview.css';
import { IconDatabase, IconChartBar, IconFilter, IconEdit, IconTrash, IconChevronDown } from '@tabler/icons-react';
import { GitBranch as LucideGitBranch, ChevronsDown as LucideChevronsDown, ChevronsUp as LucideChevronsUp } from 'lucide-react';
import { useGlobalSearch } from '../../contexts/GlobalSearchContext';
import { useTenant } from '../../contexts/TenantContext';
import { useExtensionsService } from '../../services/extensions';
import { useToast } from '../../hooks/use-toast';
import { SemanticModel } from './types';
import { CoreOption } from './financialCalculations';

interface SemanticModelOverviewProps {
  semanticModel: SemanticModel;
  children?: ReactNode;
  coreOptions?: CoreOption[];
  removeSemanticElement: (type: 'dimensions' | 'measures' | 'filters' | 'joins', id: string) => void;
  toggleElementEdit: (type: 'dimensions' | 'measures' | 'filters' | 'joins', id: string) => void;
  updateSemanticElement: (type: 'dimensions' | 'measures' | 'filters' | 'joins', id: string, updates: any) => void;
  extendsModel?: string | null;
  editMode?: boolean;
  onElementSelect?: (element: any) => void;
  allModelKeys?: string[]; // list of all known model keys to validate base existence
}

// Sanitize and normalize model object to a shape the backend expects for validation
const sanitizeModelForValidation = (input: any) => {
  if (!input || typeof input !== 'object') return {};
  let out: any = {};
  try { out = JSON.parse(JSON.stringify(input)); } catch { out = { ...input }; }
  const allowedTop = new Set(['extends', 'metadata', 'cubes', 'title', 'description', 'measures', 'dimensions', 'pre_aggregations', 'tenant_instance_id']);
  Object.keys(out).forEach(k => { if (!allowedTop.has(k)) delete out[k]; });

  try {
    if (out.extends && typeof out.extends === 'string') {
      let v = out.extends.trim(); if (v.includes('.')) v = '/' + v.replace('.', '/'); if (!v.startsWith('/')) v = '/' + v; out.extends = v.toLowerCase();
    }
    if (out.metadata && typeof out.metadata === 'object') {
      const inh = out.metadata['inherits_from'];
      if (inh && typeof inh === 'string') { let v = inh.trim(); if (v.includes('.')) v = '/' + v.replace('.', '/'); if (!v.startsWith('/')) v = '/' + v; out.metadata['inherits_from'] = v.toLowerCase(); }
    }
  } catch {}

  if (!Array.isArray(out.cubes)) out.cubes = [];
  out.cubes = out.cubes.map((c: any) => {
    if (!c || typeof c !== 'object') return null;
    const cubeOut: any = {};
    if (c.name && typeof c.name === 'string') cubeOut.name = c.name.toLowerCase();
    if (c.measures && typeof c.measures === 'object') {
      cubeOut.measures = {};
      Object.entries(c.measures).forEach(([mk, mvRaw]) => {
        const mv: any = mvRaw as any; if (!mv || typeof mv !== 'object') return; const m: any = {}; if (mv.name) m.name = mv.name; if (mv.title) m.title = mv.title; if (mv.format) m.format = mv.format; if (mv.sql) m.sql = mv.sql; if (mv.type) m.type = mv.type; cubeOut.measures[mk] = m;
      });
    }
    if (c.dimensions && typeof c.dimensions === 'object') {
      cubeOut.dimensions = {};
      Object.entries(c.dimensions).forEach(([dk, dvRaw]) => { const dv: any = dvRaw as any; if (!dv || typeof dv !== 'object') return; const d: any = {}; if (dv.name) d.name = dv.name; if (dv.title) d.title = dv.title; if (dv.format) d.format = dv.format; if (dv.sql) d.sql = dv.sql; if (dv.type) d.type = dv.type; cubeOut.dimensions[dk] = d; });
    }
    if (c.joins && typeof c.joins === 'object') {
      cubeOut.joins = {};
      Object.entries(c.joins).forEach(([jk, jvRaw]) => { const jv: any = jvRaw as any; if (!jv || typeof jv !== 'object') return; const j: any = {}; if (jv.name) j.name = jv.name; if (jv.sql) j.sql = jv.sql; if (jv.relationship) j.relationship = jv.relationship; if (jv.foreign_key) j.foreign_key = jv.foreign_key; if (jv.type) j.type = jv.type; cubeOut.joins[jk] = j; });
    }
    return cubeOut.name ? cubeOut : null;
  }).filter(Boolean);

  const arrayToMap = (arr: any[]) => { const m: any = {}; arr.forEach(item => { if (!item || typeof item !== 'object') return; const key = item.name || item.id || item.key; if (!key) return; m[String(key)] = item; }); return m; };
  if (Array.isArray(out.measures)) out.measures = arrayToMap(out.measures as any[]);
  if (Array.isArray(out.dimensions)) out.dimensions = arrayToMap(out.dimensions as any[]);

  if ((!out.cubes || out.cubes.length === 0) && ((out.measures && Object.keys(out.measures).length > 0) || (out.dimensions && Object.keys(out.dimensions).length > 0))) {
    const fallbackName = (out.title && String(out.title).toLowerCase()) || (out.metadata && out.metadata.name && String(out.metadata.name).toLowerCase()) || 'unnamed';
    const fallbackCube: any = { name: fallbackName };
    if (out.measures && typeof out.measures === 'object' && Object.keys(out.measures).length > 0) fallbackCube.measures = out.measures;
    if (out.dimensions && typeof out.dimensions === 'object' && Object.keys(out.dimensions).length > 0) fallbackCube.dimensions = out.dimensions;
    out.cubes = [fallbackCube]; delete out.measures; delete out.dimensions;
  }

  if (out.pre_aggregations && Array.isArray(out.pre_aggregations)) out.pre_aggregations = out.pre_aggregations.map((p: any) => { if (!p || typeof p !== 'object') return null; const po: any = {}; if (p.name) po.name = p.name; if (p.type) po.type = p.type; if (p.partition) po.partition = p.partition; if (p.sql) po.sql = p.sql; if (p.indexes) po.indexes = p.indexes; return po; }).filter(Boolean);

  const MAX_STR = 100000;
  const walkTrim = (obj: any) => { if (!obj || typeof obj !== 'object') return; Object.keys(obj).forEach(k => { const v = obj[k]; if (typeof v === 'string' && v.length > MAX_STR) obj[k] = v.slice(0, MAX_STR); else if (typeof v === 'object') walkTrim(v); }); };
  walkTrim(out);
  return out;
};

const SemanticModelOverview: FC<SemanticModelOverviewProps> = ({
  semanticModel,
  children: _children,
  coreOptions = [],
  removeSemanticElement: baseRemove,
  toggleElementEdit: baseToggle,
  updateSemanticElement: baseUpdate,
  extendsModel = null,
  editMode = false,
  onElementSelect,
  allModelKeys = [],
}) => {
  const { searchTerm: globalSearchTerm = '' } = useGlobalSearch();
  const searchTerm = globalSearchTerm || '';
  const [selectedItems, setSelectedItems] = useState<Set<string>>(new Set());
  const [collapsedSections, setCollapsedSections] = useState<Set<string>>(new Set());
  const [validationIssues, setValidationIssues] = useState<any[] | null>(null);
  const [validating, setValidating] = useState(false);
  const [erroredMap, setErroredMap] = useState<Record<string, string[]>>({});
  const toast = useToast();
  const tenantCtx = useTenant();
  const { validateExtension } = (() => {
    try { return useExtensionsService(); } catch { return { validateExtension: async () => ({ issues: [] }) } as any; }
  })();

  const storageKey = useMemo(() => {
    const id = (semanticModel as any)?.id || semanticModel?.name || 'default-model';
    return `semlayer.collapsedSections.${id}`;
  }, [semanticModel]);

  useEffect(() => {
    try {
      const raw = window.localStorage.getItem(storageKey);
      if (raw) {
        const arr = JSON.parse(raw);
        if (Array.isArray(arr)) setCollapsedSections(new Set(arr));
      }
    } catch {}
  }, [storageKey]);
  useEffect(() => {
    try { window.localStorage.setItem(storageKey, JSON.stringify(Array.from(collapsedSections))); } catch {}
  }, [collapsedSections, storageKey]);

  const isCustomModel = (semanticModel as any).is_custom;
  const baseDimensions = useMemo(() => { const list = (semanticModel as any).dimensions || []; return isCustomModel ? list.filter((e: any) => e.is_custom) : list; }, [(semanticModel as any).dimensions, isCustomModel]);
  const baseMeasures = useMemo(() => { const list = (semanticModel as any).measures || []; return isCustomModel ? list.filter((e: any) => e.is_custom) : list; }, [(semanticModel as any).measures, isCustomModel]);
  const baseFilters = useMemo(() => { const list = (semanticModel as any).filters || []; return isCustomModel ? list.filter((e: any) => e.is_custom) : list; }, [(semanticModel as any).filters, isCustomModel]);
  const baseJoins = useMemo(() => { const list = ((semanticModel as any).joins || []); return isCustomModel ? list.filter((e: any) => e.is_custom) : list; }, [(semanticModel as any).joins, isCustomModel]);
  const basePreAggregations = useMemo(() => { const list = ((semanticModel as any).pre_aggregations || []); return isCustomModel ? list.filter((e: any) => e.is_custom) : list; }, [(semanticModel as any).pre_aggregations, isCustomModel]);

  const displayDimensions = baseDimensions;
  const displayMeasures = baseMeasures;
  const displayFilters = baseFilters;
  const displayJoins = baseJoins;
  const displayPreAggregations = basePreAggregations;

  const _toggleElementEdit = (type: 'dimensions' | 'measures' | 'filters' | 'joins', id: string) => baseToggle(type, id);
  const removeSemanticElement = (type: 'dimensions' | 'measures' | 'filters' | 'joins', id: string) => baseRemove(type, id);
  const _updateSemanticElement = (type: 'dimensions' | 'measures' | 'filters' | 'joins', id: string, updates: any) => baseUpdate(type, id, updates);
  // referenced to avoid unused-local errors in some build variants/tests
  void _toggleElementEdit; void _updateSemanticElement;

  const modelKey = useMemo(() => (semanticModel as any)?.id || semanticModel?.name || 'default', [semanticModel]);
  const lastModelKeyRef = useRef<string>('');
  const didAutoSelectFirstRef = useRef<boolean>(false);
  useEffect(() => { if (lastModelKeyRef.current !== modelKey) { lastModelKeyRef.current = modelKey; didAutoSelectFirstRef.current = false; } }, [modelKey]);
  // Auto-select the first element only once per model change. Avoid re-running on list recomputations
  useEffect(() => {
    if (didAutoSelectFirstRef.current) return;
    const first = (displayDimensions[0] || displayMeasures[0] || displayFilters[0] || displayJoins[0] || displayPreAggregations[0] || null) as any;
    if (first && first.id) {
      didAutoSelectFirstRef.current = true;
      setSelectedItems(new Set([first.id]));
      try { onElementSelect?.(first); } catch {}
    }
    // Only run on model changes
  }, [modelKey]);

  const handleItemClick = (id: string, event: MouseEvent) => {
    if ((event.target as HTMLElement).closest('.element-actions, .header-right')) return;
    const element = (
      (semanticModel as any).dimensions.find((d: any) => d.id === id) ||
      (semanticModel as any).measures.find((m: any) => m.id === id) ||
      (semanticModel as any).filters.find((f: any) => f.id === id) ||
      ((semanticModel as any).joins || []).find((j: any) => j.id === id) ||
      (((semanticModel as any).pre_aggregations || []) as any).find((p: any) => p.id === id) ||
      null
    );
    setSelectedItems(new Set([id]));
    onElementSelect?.(element);
  };

  const scrollToSection = (sectionType: string) => { const element = document.getElementById(`section-${sectionType}`); if (element) element.scrollIntoView({ behavior: 'smooth', block: 'start' }); };
  const _scrollToItem = (type: string, id: string) => {
    try {
      const selector = `[data-element-id="${type}-${id}"]`;
      const el = document.querySelector(selector) as HTMLElement | null;
      if (el) { el.scrollIntoView({ behavior: 'smooth', block: 'center' }); setSelectedItems(() => new Set([id])); try { el.classList.add('element-flash'); window.setTimeout(() => el.classList.remove('element-flash'), 1100); } catch {} }
      else scrollToSection(type + 's');
    } catch (e) { scrollToSection(type + 's'); }
  };
  void _scrollToItem;

  const isSectionCollapsed = (section: 'joins'|'pre_aggregations'|'filters'|'dimensions'|'measures') => collapsedSections.has(section);
  const toggleSection = (section: 'joins'|'pre_aggregations'|'filters'|'dimensions'|'measures') => setCollapsedSections(prev => { const next = new Set(prev); if (next.has(section)) next.delete(section); else next.add(section); return next; });
  const collapseAll = () => setCollapsedSections(new Set(['joins','pre_aggregations','filters','dimensions','measures']));
  const expandAll = () => setCollapsedSections(new Set());

  useEffect(() => {
    const onKeyDown = (e: KeyboardEvent) => { const target = e.target as HTMLElement | null; if (target && (target.tagName === 'INPUT' || target.tagName === 'TEXTAREA' || target.isContentEditable)) return; const meta = e.metaKey || e.ctrlKey; if (meta && e.shiftKey && (e.key === '=' || e.key === '+')) { e.preventDefault(); expandAll(); } else if (meta && e.shiftKey && (e.key === '-' || e.key === '_')) { e.preventDefault(); collapseAll(); } };
    window.addEventListener('keydown', onKeyDown); return () => window.removeEventListener('keydown', onKeyDown);
  }, []);

  // Map validation issues to tiles
  useEffect(() => {
    if (!validationIssues || !Array.isArray(validationIssues) || validationIssues.length === 0) { setErroredMap({}); return; }
    const map: Record<string, string[]> = {};
    const push = (id: string, msg: string) => { if (!id) return; if (!map[id]) map[id] = []; map[id].push(msg); };
    const smAny: any = semanticModel as any;
    const findByKey = (key: string) => {
      if (!key) return null; const norm = String(key).toLowerCase();
      const lists: Array<any> = [smAny.dimensions || [], smAny.measures || [], smAny.filters || [], smAny.joins || [], smAny.pre_aggregations || []];
      for (const arr of lists) { for (const item of arr) { if (!item) continue; const cand = String(item.id || item.name || item.key || '').toLowerCase(); if (!cand) continue; if (cand === norm || cand.endsWith('.' + norm) || cand.includes(norm)) return item.id || item.name || null; } }
      return null;
    };
  validationIssues.forEach((it: any) => {
      try {
        const msg = it.message || it.code || String(it);
    // prefer explicit element_id from backend
    const explicitElement = it.element_id || it.elementId || it.id || it.source || undefined;
    if (explicitElement) { push(String(explicitElement), msg); return; }
        if (it.key && typeof it.key === 'string') { const found = findByKey(it.key); if (found) { push(String(found), msg); return; } }
        if (Array.isArray(it.path) && it.path.length > 0) { const joined = it.path.join('.'); const found = findByKey(joined) || findByKey(it.path[it.path.length - 1]); if (found) { push(String(found), msg); return; } }
        const m = String(msg).match(/'([^']+)'/); if (m && m[1]) { const found = findByKey(m[1]); if (found) { push(String(found), msg); return; } }
      } catch (e) {}
    });
    setErroredMap(map);
  }, [validationIssues, semanticModel]);

  useEffect(() => {
    const handler = (e: any) => { const issues = e?.detail?.issues as Array<any> | undefined; if (!issues) return; setValidationIssues(issues); };
    window.addEventListener('semlayer.validationIssues', handler as EventListener);
    return () => window.removeEventListener('semlayer.validationIssues', handler as EventListener);
  }, []);

  const renderErrorWrapper = (id: string, children: React.ReactNode) => {
    const errors = erroredMap[id] || [];
    return (
      <div className={`element-wrapper ${errors.length ? 'has-error' : ''}`} data-element-id-wrapper={id}>
        {errors.length > 0 && <div className="error-badge" aria-hidden>{errors.length}</div>}
        {children}
        {errors.length > 0 && <div className="error-tooltip" role="tooltip">{errors.join('\n')}</div>}
      </div>
    );
  };
  void renderErrorWrapper;

  const filteredDimensions = displayDimensions.filter((dim: any) => !dim.isEditing && ((dim.title && dim.title.toLowerCase().includes(searchTerm.toLowerCase())) || (dim.name && dim.name.toLowerCase().includes(searchTerm.toLowerCase())) || (dim.description && dim.description.toLowerCase().includes(searchTerm.toLowerCase()))));
  const filteredMeasures = displayMeasures.filter((measure: any) => !measure.isEditing && ((measure.title && measure.title.toLowerCase().includes(searchTerm.toLowerCase())) || (measure.name && measure.name.toLowerCase().includes(searchTerm.toLowerCase())) || (measure.description && measure.description.toLowerCase().includes(searchTerm.toLowerCase()))));
  const filteredFilters = displayFilters.filter((filter: any) => !filter.isEditing && ((filter.title && filter.title.toLowerCase().includes(searchTerm.toLowerCase())) || (filter.name && filter.name.toLowerCase().includes(searchTerm.toLowerCase())) || (filter.description && filter.description.toLowerCase().includes(searchTerm.toLowerCase()))));
  const filteredJoins = displayJoins.filter((join: any) => !join.isEditing && ((join.name && join.name.toLowerCase().includes(searchTerm.toLowerCase())) || (join.sql && join.sql.toLowerCase().includes(searchTerm.toLowerCase()))));
  const filteredPreAggregations = displayPreAggregations.filter((preagg: any) => !preagg.isEditing && ((preagg.name && (preagg.name as string).toLowerCase().includes(searchTerm.toLowerCase())) || (preagg.description && preagg.description.toLowerCase().includes(searchTerm.toLowerCase()))));
  const showNoResults = searchTerm && filteredJoins.length === 0 && filteredFilters.length === 0 && filteredDimensions.length === 0 && filteredMeasures.length === 0 && filteredPreAggregations.length === 0;
  const hasModelItems = baseDimensions.length > 0 || baseMeasures.length > 0 || baseFilters.length > 0 || baseJoins.length > 0 || basePreAggregations.length > 0;
  const showEmptyModel = !searchTerm && !hasModelItems;
  return (
    <div className="content-main full-width">
      <div className="elements-toolbar">
        <div className="toolbar-left">
          {isCustomModel && extendsModel && (
            <span className="extends-badge" title={`Extends ${extendsModel}`} aria-label={`Extends ${extendsModel}`}>
              <LucideGitBranch size={14} />
              <span className="extends-text">Extends: {extendsModel}</span>
            </span>
          )}
        </div>
        <div className="toolbar-right">
          <button className="btn btn-secondary btn-sm" onClick={expandAll} aria-label="Expand all" title="Expand all (⌘/Ctrl+Shift=)"><LucideChevronsDown size={16} /></button>
          <button className="btn btn-secondary btn-sm" onClick={collapseAll} aria-label="Collapse all" title="Collapse all (⌘/Ctrl+Shift+-)"><LucideChevronsUp size={16} /></button>
          {isCustomModel && (
            <button className="btn btn-primary btn-sm" onClick={async () => {
              try {
                setValidating(true);
                let baseKey = extendsModel || (semanticModel as any)?.parent_model_key || '';
                if (baseKey) { baseKey = String(baseKey).trim(); if (baseKey.includes('.')) baseKey = '/' + baseKey.replace('.', '/'); if (!baseKey.startsWith('/')) baseKey = '/' + baseKey; baseKey = baseKey.toLowerCase(); }
                // Pre-validate that the base model actually exists on the client side
                const normalize = (s: string) => {
                  let v = (s || '').toString().trim().toLowerCase();
                  if (!v) return '';
                  if (v.includes('.')) v = v.replace(/\.+/g, '/');
                  if (!v.startsWith('/')) v = '/' + v;
                  v = v.replace(/\/+/, '/');
                  return v;
                };
                const known = new Set((allModelKeys || []).map(k => normalize(k)));
                if (baseKey && !known.has(baseKey)) {
                  const issue = { level: 'error', code: 'base_not_found', message: `Base model not found: ${baseKey}`, element_id: `extends__${baseKey}` } as any;
                  setValidationIssues([issue]);
                  try { window.dispatchEvent(new CustomEvent('semlayer.validationIssues', { detail: { issues: [issue] } })); } catch {}
                  return;
                }
                const sanitizedModel = sanitizeModelForValidation(semanticModel);
                const payload = { base_model_key: baseKey, model_object: sanitizedModel } as any;
                const sm: any = semanticModel as any;
                let dsId = sm.tenant_instance_id || sm.datasourceId || sm.datasource || sm.datasourceUuid || sm.datasource_uuid || '';
                if (!dsId && tenantCtx && tenantCtx.datasource && tenantCtx.datasource.id) dsId = tenantCtx.datasource.id;
                if (!dsId) { setValidationIssues([{ level: 'error', message: 'tenant_instance_id is required for validation (missing from model). Please ensure the model has a tenant_instance_id.' }]); setValidating(false); return; }
                const res = await validateExtension(dsId, payload as any);
                const issues = res.issues || [];
                setValidationIssues(issues);
                try { window.dispatchEvent(new CustomEvent('semlayer.validationIssues', { detail: { issues } })); } catch {}
                if (!issues || issues.length === 0) { try { toast.toast({ title: 'Validation passed', description: 'No validation issues found', open: true }); } catch {} }
                else {
                  try { toast.toast({ title: 'Validation returned issues', description: `${issues.length} issue(s) found — opening code view`, open: true }); } catch {}
                  try { window.dispatchEvent(new CustomEvent('semlayer.openCode')); } catch {}
                  try {
                    const pick = issues[0]; const msg: string = (pick && (pick.message || pick.code || String(pick))) || '';
                    const m = msg.match(/'([^']+)'/); const ident = m ? m[1] : undefined; const lower = msg.toLowerCase(); let section = 'dimensions';
                    if (lower.includes('measure')) section = 'measures'; else if (lower.includes('join')) section = 'joins'; else if (lower.includes('pre-aggregation') || lower.includes('pre_aggregation') || lower.includes('pre aggregation')) section = 'pre_aggregations'; else if (lower.includes('filter')) section = 'filters';
                    const detail: any = { section }; if (ident) detail.key = ident; try { window.dispatchEvent(new CustomEvent('semlayer.jumpToSection', { detail })); } catch {}
                  } catch (err) {}
                }
              } catch (err: any) {
                const eid = `extends__${extendsModel || (semanticModel as any)?.parent_model_key || 'base'}`;
                const issue: any = { level: 'error', code: 'validation_error', message: err?.message || String(err), element_id: eid };
                setValidationIssues([issue]);
                try { window.dispatchEvent(new CustomEvent('semlayer.validationIssues', { detail: { issues: [issue] } })); } catch {}
              } finally { setValidating(false); }
            }} aria-label="Validate model" title="Validate model against base">{validating ? 'Validating…' : 'Validate'}</button>
          )}
        </div>
      </div>

      {validationIssues && (
        <div className="validation-issues">
          <h5>Validation issues</h5>
          <ul>
            {validationIssues.map((it: any, idx: number) => (<li key={idx}><strong>{it.level || it.code || 'issue'}</strong>: {it.message}</li>))}
          </ul>
        </div>
      )}

      <div className="semantic-elements enhanced">
        {isCustomModel && extendsModel && !searchTerm && (
          <div className="element-section enhanced" id="section-extends">
            <div className="elements-grid enhanced">
              {(() => {
                const eid = `extends__${extendsModel}`;
                const isSelected = selectedItems.has(eid);
                const element = { id: eid, type: 'extends', name: String(extendsModel), title: 'Extends', is_custom: true } as any;
                const extendErrors = erroredMap[eid] || [];
                return (
                  <div key={eid} data-element-id={`extends-${extendsModel}`} className={`element-card extends enhanced ${isSelected ? 'selected' : ''} ${extendErrors.length ? 'error' : ''}`} onClick={() => { try { devLog('[SemanticModelOverview] extends tile clicked', { eid, extendsModel }); } catch {} setSelectedItems(new Set([eid])); onElementSelect?.(element); }} title={`This custom model extends the base model: ${extendsModel}`} role="button" aria-label={`Extends ${extendsModel}`}>
                    {extendErrors.length > 0 && (<div className="error-badge" aria-hidden>{extendErrors.length}</div>)}
                    <div className="element-view">
                      <div className="element-header enhanced compact">
                        <div className="title-type-row">
                          <span className="element-title enhanced"><LucideGitBranch size={14} style={{ marginRight: 6 }} />Extends: {extendsModel}</span>
                          <span className="element-badge custom">Custom</span>
                        </div>
                      </div>
                      <div className="element-details enhanced"><span className="element-source">Base model: {extendsModel}</span></div>
                      {extendErrors.length > 0 && (<div className="error-tooltip" role="tooltip">{extendErrors.join('\n')}</div>)}
                    </div>
                  </div>
                );
              })()}
            </div>
          </div>
        )}

        {searchTerm && (<div className="search-results-summary">Found {filteredJoins.length + filteredFilters.length + filteredDimensions.length + filteredMeasures.length + filteredPreAggregations.length} elements matching "{searchTerm}"</div>)}

        {filteredJoins.length > 0 && (
          <div id="section-joins" className={`element-section enhanced ${isSectionCollapsed('joins') ? 'collapsed' : ''}`}>
            <h4 role="button" tabIndex={0} onClick={() => toggleSection('joins')} onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); toggleSection('joins'); } }} aria-controls="content-joins">
              <span className="section-title"><IconDatabase size={16} /> Joins ({filteredJoins.length})</span>
              <span className="section-toggle" aria-hidden><IconChevronDown className="chevron" size={16} /></span>
            </h4>
            <div id="content-joins" className={`elements-grid enhanced collapse-container ${isSectionCollapsed('joins') ? 'collapsed' : 'expanded'}`}>
              {filteredJoins.map((join: any) => {
                const errs = erroredMap[join.id] || [];
                return (
                  <div key={join.id} data-element-id={`join-${join.id}`} className={`element-card join enhanced ${selectedItems.has(join.id) ? 'selected' : ''} ${errs.length ? 'error' : ''}`} onClick={(e) => handleItemClick(join.id, e)}>
                    {errs.length > 0 && (<div className="error-badge" aria-hidden>{errs.length}</div>)}
                    <div className="element-view">
                      <div className="element-header enhanced compact">
                        <div className="title-type-row">
                          <span className="element-title enhanced">{join.name}</span>
                          {join.is_custom ? (<span className={`element-badge ${(/* @ts-ignore */ (coreOptions || []).some((c: any) => c?.name === join.name && c?.type === 'join')) ? 'override' : 'custom'}`}>{(/* @ts-ignore */ (coreOptions || []).some((c: any) => c?.name === join.name && c?.type === 'join')) ? 'Override' : 'Custom'}</span>) : (<span className="element-badge core">Core</span>)}
                          <div className="header-right">
                            <span className="element-type-badge join">{(join as any).joinType || (join as any).relationship || 'inner'}</span>
                            {editMode && join.is_custom && (<>
                              <button className="edit-btn" title="Edit Join" onClick={(e) => { e.stopPropagation(); setSelectedItems(new Set([join.id])); onElementSelect?.(join); }}><IconEdit size={14} /></button>
                              <button className="remove-btn" title="Remove Join" onClick={(e) => { e.stopPropagation(); removeSemanticElement('joins', join.id); }}><IconTrash size={14} /></button>
                            </>) }
                          </div>
                        </div>
                      </div>
                      <div className="element-details enhanced"><span className="element-source">{((join as any).leftTable && (typeof (join as any).leftTable === 'object') ? ((join as any).leftTable.node_name || (join as any).leftTable.id) : (join as any).leftTable) || ''} → {((join as any).rightTable && (typeof (join as any).rightTable === 'object') ? ((join as any).rightTable.node_name || (join as any).rightTable.id) : (join as any).rightTable) || ''}</span></div>
                      {(join as any).description && (<div className="element-description">{(join as any).description}</div>)}
                    </div>
                    {errs.length > 0 && (<div className="error-tooltip" role="tooltip">{errs.join('\n')}</div>)}
                  </div>
                );
              })}
            </div>
          </div>
        )}

        {filteredPreAggregations.length > 0 && (
          <div id="section-pre_aggregations" className={`element-section enhanced ${isSectionCollapsed('pre_aggregations') ? 'collapsed' : ''}`}>
            <h4 id="pre_aggregations" role="button" tabIndex={0} onClick={() => toggleSection('pre_aggregations')} onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); toggleSection('pre_aggregations'); } }} aria-controls="content-pre_aggregations">
              <IconDatabase size={16} /><IconDatabase size={16} /><span className="section-title">Pre-Aggregations ({filteredPreAggregations.length})</span>
              <span className="section-toggle" aria-hidden><IconChevronDown className="chevron" size={16} /></span>
            </h4>
            <div id="content-pre_aggregations" className={`elements-grid enhanced collapse-container ${isSectionCollapsed('pre_aggregations') ? 'collapsed' : 'expanded'}`}>
              {filteredPreAggregations.map((preagg: any) => {
                const errs = erroredMap[preagg.id] || [];
                return (
                  <div key={preagg.id} data-element-id={`pre_aggregation-${preagg.id}`} className={`element-card preagg enhanced ${selectedItems.has(preagg.id) ? 'selected' : ''} ${errs.length ? 'error' : ''}`} onClick={(e) => handleItemClick(preagg.id, e)}>
                    {errs.length > 0 && (<div className="error-badge" aria-hidden>{errs.length}</div>)}
                    <div className="element-view">
                      <div className="element-header enhanced compact">
                        <div className="title-type-row">
                          <span className="element-title enhanced">{preagg.name}</span>
                          {preagg.is_custom ? (<span className={`element-badge ${(/* @ts-ignore */ (coreOptions || []).some((c: any) => c?.name === preagg.name && c?.type === 'pre_aggregations')) ? 'override' : 'custom'}`}>{(/* @ts-ignore */ (coreOptions || []).some((c: any) => c?.name === preagg.name && c?.type === 'pre_aggregations')) ? 'Override' : 'Custom'}</span>) : (<span className="element-badge core">Core</span>)}
                        </div>
                      </div>
                      <div className="element-details enhanced"><span className="element-source">{preagg.type || 'unknown'}</span></div>
                    </div>
                    {errs.length > 0 && (<div className="error-tooltip" role="tooltip">{errs.join('\n')}</div>)}
                  </div>
                );
              })}
            </div>
          </div>
        )}

        {filteredFilters.length > 0 && (
          <div id="section-filters" className={`element-section enhanced ${isSectionCollapsed('filters') ? 'collapsed' : ''}`}>
            <h4 role="button" tabIndex={0} onClick={() => toggleSection('filters')} onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); toggleSection('filters'); } }} aria-controls="content-filters">
              <span className="section-title"><IconFilter size={16} /> Filters ({filteredFilters.length})</span>
              <span className="section-toggle" aria-hidden><IconChevronDown className="chevron" size={16} /></span>
            </h4>
            <div id="content-filters" className={`elements-grid enhanced collapse-container ${isSectionCollapsed('filters') ? 'collapsed' : 'expanded'}`}>
              {filteredFilters.map((filter: any) => {
                const errs = erroredMap[filter.id] || [];
                return (
                  <div key={filter.id} data-element-id={`filter-${filter.id}`} className={`element-card filter enhanced ${selectedItems.has(filter.id) ? 'selected' : ''} ${errs.length ? 'error' : ''}`} onClick={(e) => handleItemClick(filter.id, e)}>
                    {errs.length > 0 && (<div className="error-badge" aria-hidden>{errs.length}</div>)}
                    <div className="element-view">
                      <div className="element-header enhanced compact">
                        <div className="title-type-row">
                          <span className="element-title enhanced">{filter.title}</span>
                          {filter.is_custom ? (<span className={`element-badge ${(/* @ts-ignore */ (coreOptions || []).some((c: any) => c?.name === filter.title && c?.type === 'filter')) ? 'override' : 'custom'}`}>{(/* @ts-ignore */ (coreOptions || []).some((c: any) => c?.name === filter.title && c?.type === 'filter')) ? 'Override' : 'Custom'}</span>) : (<span className="element-badge core">Core</span>)}
                          <div className="header-right">
                            <span className="element-type-badge filter">{filter.type || 'string'}</span>
                            {editMode && filter.is_custom && (<>
                              <button className="edit-btn" title="Edit Filter" onClick={(e) => { e.stopPropagation(); setSelectedItems(new Set([filter.id])); onElementSelect?.(filter); }}><IconEdit size={14} /></button>
                              <button className="remove-btn" title="Remove Filter" onClick={(e) => { e.stopPropagation(); removeSemanticElement('filters', filter.id); }}><IconTrash size={14} /></button>
                            </>) }
                          </div>
                        </div>
                      </div>
                      <div className="element-details enhanced"><span className="element-source">{(filter.sourceTable && (typeof filter.sourceTable === 'object' ? filter.sourceTable.node_name || filter.sourceTable.id : filter.sourceTable)) || ''}.{filter.sourceColumn}</span></div>
                      {filter.description && (<div className="element-description">{filter.description}</div>)}
                    </div>
                    {errs.length > 0 && (<div className="error-tooltip" role="tooltip">{errs.join('\n')}</div>)}
                  </div>
                );
              })}
            </div>
          </div>
        )}

        {filteredDimensions.length > 0 && (
          <div id="section-dimensions" className={`element-section enhanced ${isSectionCollapsed('dimensions') ? 'collapsed' : ''}`}>
            <h4 role="button" tabIndex={0} onClick={() => toggleSection('dimensions')} onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); toggleSection('dimensions'); } }} aria-controls="content-dimensions">
              <span className="section-title"><IconDatabase size={16} /> Dimensions ({filteredDimensions.length})</span>
              <span className="section-toggle" aria-hidden><IconChevronDown className="chevron" size={16} /></span>
            </h4>
            <div id="content-dimensions" className={`elements-grid enhanced collapse-container ${isSectionCollapsed('dimensions') ? 'collapsed' : 'expanded'}`}>
              {filteredDimensions.map((dim: any) => {
                const errs = erroredMap[dim.id] || [];
                return (
                  <div key={dim.id} data-element-id={`dimension-${dim.id}`} className={`element-card dimension enhanced ${selectedItems.has(dim.id) ? 'selected' : ''} ${errs.length ? 'error' : ''}`} onClick={(e) => handleItemClick(dim.id, e)}>
                    {errs.length > 0 && (<div className="error-badge" aria-hidden>{errs.length}</div>)}
                    <div className="element-view">
                      <div className="element-header enhanced compact">
                        <div className="title-type-row">
                          <span className="element-title enhanced">{dim.title || dim.name}</span>
                          {dim.is_custom ? (<span className="element-badge custom">Custom</span>) : (<span className="element-badge core">Core</span>)}
                          <div className="header-right">
                            <span className="element-type-badge dimension">{dim.type || 'string'}</span>
                            {editMode && dim.is_custom && (<>
                              <button className="edit-btn" title="Edit Dimension" onClick={(e) => { e.stopPropagation(); setSelectedItems(new Set([dim.id])); onElementSelect?.(dim); }}><IconEdit size={14} /></button>
                              <button className="remove-btn" title="Remove Dimension" onClick={(e) => { e.stopPropagation(); removeSemanticElement('dimensions', dim.id); }}><IconTrash size={14} /></button>
                            </>) }
                          </div>
                        </div>
                      </div>
                      <div className="element-details enhanced"><span className="element-source">{(dim.sourceTable && (typeof dim.sourceTable === 'object' ? dim.sourceTable.node_name || dim.sourceTable.id : dim.sourceTable)) || ''}.{dim.sourceColumn}</span></div>
                      {dim.description && (<div className="element-description">{dim.description}</div>)}
                    </div>
                    {errs.length > 0 && (<div className="error-tooltip" role="tooltip">{errs.join('\n')}</div>)}
                  </div>
                );
              })}
            </div>
          </div>
        )}

        {filteredMeasures.length > 0 && (
          <div id="section-measures" className={`element-section enhanced ${isSectionCollapsed('measures') ? 'collapsed' : ''}`}>
            <h4 role="button" tabIndex={0} onClick={() => toggleSection('measures')} onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); toggleSection('measures'); } }} aria-controls="content-measures">
              <span className="section-title"><IconChartBar size={16} /> Measures ({filteredMeasures.length})</span>
              <span className="section-toggle" aria-hidden><IconChevronDown className="chevron" size={16} /></span>
            </h4>
            <div id="content-measures" className={`elements-grid enhanced collapse-container ${isSectionCollapsed('measures') ? 'collapsed' : 'expanded'}`}>
              {filteredMeasures.map((m: any) => {
                const errs = erroredMap[m.id] || [];
                return (
                  <div key={m.id} data-element-id={`measure-${m.id}`} className={`element-card measure enhanced ${selectedItems.has(m.id) ? 'selected' : ''} ${errs.length ? 'error' : ''}`} onClick={(e) => handleItemClick(m.id, e)}>
                    {errs.length > 0 && (<div className="error-badge" aria-hidden>{errs.length}</div>)}
                    <div className="element-view">
                      <div className="element-header enhanced compact">
                        <div className="title-type-row">
                          <span className="element-title enhanced">{m.title || m.name}</span>
                          {m.is_custom ? (<span className="element-badge custom">Custom</span>) : (<span className="element-badge core">Core</span>)}
                          <div className="header-right">
                            <span className="element-type-badge measure">{m.type || m.sql || ''}</span>
                            {editMode && m.is_custom && (<>
                              <button className="edit-btn" title="Edit Measure" onClick={(e) => { e.stopPropagation(); setSelectedItems(new Set([m.id])); onElementSelect?.(m); }}><IconEdit size={14} /></button>
                              <button className="remove-btn" title="Remove Measure" onClick={(e) => { e.stopPropagation(); removeSemanticElement('measures', m.id); }}><IconTrash size={14} /></button>
                            </>) }
                          </div>
                        </div>
                      </div>
                      <div className="element-details enhanced"><span className="element-source">{m.type || m.sql || ''}</span></div>
                    </div>
                    {errs.length > 0 && (<div className="error-tooltip" role="tooltip">{errs.join('\n')}</div>)}
                  </div>
                );
              })}
            </div>
          </div>
        )}

        {showNoResults && (<div className="no-results">No results found for "{searchTerm}"</div>)}
        {showEmptyModel && (<div className="empty-model">This model has no custom elements yet. Use the builder or paste JSON/YAML to get started.</div>)}

      </div>
    </div>
  );
};

export default SemanticModelOverview;