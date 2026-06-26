import React, { useState, useEffect, useMemo, useCallback } from 'react';
import { Autocomplete, TextField, CircularProgress } from '@mui/material';
import { useLazyQuery } from '@apollo/client';
import { GET_TABLES_FOR_DATASOURCE } from '../../graphql/queries/datasourceQueries';
import './SimpleTableAutocomplete.css';
import { devDebug } from '../../utils/devLogger';

export interface SimpleTableOption {
  id: string;
  node_name?: string;
  qualified_path?: string;
  description?: string;
  catalog_type_name?: string;
  node_type_id?: string;
  catalog_defn?: any;
}

interface ModelTableAutocompleteProps {
  datasourceId?: string;
  semanticModel?: any;
  value: string | SimpleTableOption | null;
  onChange: (v: string | SimpleTableOption | null) => void;
  returnType?: 'object' | 'id';
  limit?: number;
  debounceMs?: number;
  placeholder?: string;
  disabled?: boolean;
  className?: string;
  fullWidth?: boolean;
  autoFocus?: boolean;
  showAllOnFocus?: boolean; // fetch % on focus/open when no query
  minChars?: number; // minimum characters before triggering a search (default 2)
}

// Simple in-memory cache (datasourceId||q||limit) => { ts, rows }
const TABLE_CACHE = new Map<string, { ts: number; rows: SimpleTableOption[] }>();
const CACHE_TTL_MS = 60_000; // 1 minute
const RECENT_KEY_PREFIX = 'sta_recent_tables';
const MAX_RECENTS = 10;

const ModelTableAutocomplete: React.FC<ModelTableAutocompleteProps> = ({
  datasourceId,
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  semanticModel,
  value,
  onChange,
  returnType = 'object',
  limit = 50,
  debounceMs = 250,
  placeholder = 'schema.table_name',
  disabled = false,
  className,
  fullWidth = true,
  autoFocus = false,
  showAllOnFocus = true,
  minChars = 2,
}) => {
  const [input, setInput] = useState('');
  const [options, setOptions] = useState<SimpleTableOption[]>([]);
  const [recentOptions, setRecentOptions] = useState<SimpleTableOption[]>([]);
  // no controlled open now; let MUI handle it
  const [error, setError] = useState<string | null>(null);

  // Use semanticModel to avoid unused variable error
  useEffect(() => {
    if (semanticModel) {
      // TODO: filter options based on semanticModel
    }
  }, [semanticModel]);
  const [runQuery, { loading }] = useLazyQuery(GET_TABLES_FOR_DATASOURCE, { fetchPolicy: 'network-only' });

  // Fallback: if no datasourceId passed via props but tenant selection cached in localStorage
  const effectiveDatasourceId = useMemo(() => {
    if (datasourceId) return datasourceId;
    if (typeof window === 'undefined') return undefined;
    try {
      const stored = window.localStorage.getItem('selected_datasource');
      if (stored) {
        const parsed = JSON.parse(stored);
        return parsed?.id;
      }
    } catch { /* ignore */ }
    return undefined;
  }, [datasourceId]);

  // Decide how to show current value
  const selectedOption = useMemo(() => {
    if (!value) return null;
    const searchPools = [recentOptions, options];
    if (typeof value === 'string') {
      for (const pool of searchPools) {
        const found = pool.find(o => o.id === value || o.qualified_path === value);
        if (found) return found;
      }
      return { id: value, qualified_path: value };
    }
    return value;
  }, [value, options, recentOptions]);

  // Combined display options (prepend recents when no query)
  const displayOptions = useMemo(() => {
    if (input.trim()) return options; // when searching, just fetched options
    // merge recentOptions + options (dedupe by id)
    const seen = new Set<string>();
    const merged: SimpleTableOption[] = [];
    for (const list of [recentOptions, options]) {
      for (const o of list) {
        const key = o.id || o.qualified_path || JSON.stringify(o);
        if (key && !seen.has(key)) {
          seen.add(key);
          merged.push(o);
        }
      }
    }
    return merged;
  }, [input, options, recentOptions]);

  // Debug: log options count in dev
  useEffect(() => {
    const sample = displayOptions.slice(0,3).map(o=>o.qualified_path||o.node_name);
    devDebug('[STA] displayOptions', { count: displayOptions.length, sample, rawOptions: options.length, recents: recentOptions.length, input });
  }, [displayOptions.length, options.length, recentOptions.length]);

  // Debounced fetch + cache
  useEffect(() => {
    if (!effectiveDatasourceId) { setOptions([]); return; }
    const q = input.trim();
    const normQ = q.length > 0 ? q : ''; // we'll map to % for broad search
    // don't run broad searches for short inputs unless showAllOnFocus is used
    if (normQ && normQ.length < minChars) {
      setOptions([]);
      return;
    }
    const cacheKey = `${effectiveDatasourceId}||${normQ}||${limit}`;
    const cached = TABLE_CACHE.get(cacheKey);
    const now = Date.now();
    if (cached && (now - cached.ts) < CACHE_TTL_MS) {
      setOptions(cached.rows);
      return; // serve from cache
    }
    let active = true;
  const t = setTimeout(async () => {
      try {
        const vars: any = { datasourceId, limit };
        // GraphQL filter expects pattern; use '%' for match-all when no query typed
        vars.q = normQ ? `%${normQ}%` : '%';
  const res: any = await runQuery({ variables: { ...vars, datasourceId: effectiveDatasourceId } });
        if (!active) return;
        const rows = (res?.data?.catalog_node_vw || []).map((r: any) => ({
          id: r.node_id || r.id,
          node_name: r.node_name,
          qualified_path: r.qualified_path,
          description: r.description,
          catalog_type_name: r.catalog_type_name,
          node_type_id: r.node_type_id,
          catalog_defn: r.catalog_defn,
        })) as SimpleTableOption[];
        setOptions(rows);
        TABLE_CACHE.set(cacheKey, { ts: Date.now(), rows });
        setError(null);
      } catch {
        if (active) {
          setOptions([]);
          setError('Failed to load tables');
        }
      }
    }, debounceMs);
    return () => { active = false; clearTimeout(t); };
  }, [input, effectiveDatasourceId, limit, debounceMs, runQuery]);

  // If we have a string value not yet in options, trigger a broad fetch (empty input) once when mounted / datasource changes.
  useEffect(() => {
    if (!effectiveDatasourceId) return;
    if (!value || typeof value !== 'string') return;
    if (options.some(o => o.id === value || o.qualified_path === value)) return;
    // Kick off a broad search if user hasn't typed yet
    if (!input) setInput('');
  }, [value, options, effectiveDatasourceId, input]);

  // Load recents when datasource changes
  useEffect(() => {
    if (!effectiveDatasourceId) { setRecentOptions([]); return; }
    try {
      const raw = localStorage.getItem(`${RECENT_KEY_PREFIX}:${effectiveDatasourceId}`);
      if (raw) {
        const parsed = JSON.parse(raw);
        if (Array.isArray(parsed)) setRecentOptions(parsed.slice(0, MAX_RECENTS));
      }
    } catch { /* ignore */ }
  }, [effectiveDatasourceId]);

  const persistRecent = useCallback((opt: SimpleTableOption) => {
    if (!effectiveDatasourceId) return;
    setRecentOptions(prev => {
      const existing = prev.filter(p => p.id !== opt.id);
      const next = [opt, ...existing].slice(0, MAX_RECENTS);
      try { localStorage.setItem(`${RECENT_KEY_PREFIX}:${effectiveDatasourceId}`, JSON.stringify(next)); } catch { /* ignore */ }
      return next;
    });
  }, [effectiveDatasourceId]);

  const handleChange = (_: any, newVal: any) => {
    if (!newVal) { onChange(null); return; }
    if (returnType === 'id') {
      const idOut = typeof newVal === 'string' ? newVal : (newVal.id || newVal.qualified_path);
      onChange(idOut || null);
    } else {
      onChange(newVal);
    }
    // If object, persist as recent
    if (newVal && typeof newVal !== 'string') persistRecent(newVal as SimpleTableOption);
  };

  // Highlight helper
  const escapeRegExp = (s: string) => s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
  const highlight = (text: string, query: string) => {
    if (!query) return text;
    try {
      // Avoid the global flag here. Using re.test repeatedly with the 'g' flag
      // moves the internal index and yields alternating results. Use case-insensitive only.
      const re = new RegExp(`(${escapeRegExp(query)})`, 'i');
      const parts = text.split(new RegExp(`(${escapeRegExp(query)})`, 'i'));
      return parts.map((part, i) => re.test(part) ? <mark key={i} className="sta-match">{part}</mark> : <span key={i}>{part}</span>);
    } catch { return text; }
  };

  return (
    <Autocomplete
      options={displayOptions}
      value={selectedOption as any}
      onChange={handleChange}
      getOptionLabel={(o: any) => {
        if (!o) return '';
        if (typeof o === 'string') return o;
        // Prefer qualified_path; fallback to node_name
        const qp = o.qualified_path || '';
        if (qp) return qp; // keep leading slash so user sees schema context
        return o.node_name || '';
      }}
      isOptionEqualToValue={(o: any, v: any) => !!o && !!v && (o.id === v.id || o.qualified_path === v.qualified_path)}
      inputValue={input}
      onInputChange={(_, val) => setInput(val)}
      loading={loading}
      className={className}
      disabled={disabled || !effectiveDatasourceId}
      fullWidth={fullWidth}
      slotProps={{ popper: { sx: { zIndex: 999999, backgroundColor: 'white', border: '1px solid red' } } as any }}
      filterOptions={(x) => x}
      openOnFocus
      freeSolo
  disablePortal={false}
  onOpen={() => devDebug('ModelTableAutocomplete opened')}
  onClose={() => devDebug('ModelTableAutocomplete closed')}
      onFocus={() => {
        if (showAllOnFocus && !input && options.length === 0 && !loading) setInput('');
      }}
      renderOption={(props, opt: any) => {
        const labelPrimary = opt.qualified_path || opt.node_name || '';
        const labelSecondary = (opt.node_name && opt.qualified_path && opt.qualified_path !== opt.node_name) ? opt.node_name : '';
        const q = input.trim();
        const typeLabel = (opt.catalog_type_name || '').toLowerCase();
  // Debug hook (emits via dev logger when DEV is enabled)
  devDebug('[STA] render option', opt.id, labelPrimary);
        return (
          <li {...props} key={opt.id || opt.qualified_path} className="sta-option">
            <div className="sta-option-top-row">
              <span className="sta-option-primary">{highlight(labelPrimary, q)}</span>
              {typeLabel ? <span className={`sta-badge sta-badge-${typeLabel}`}>{typeLabel}</span> : null}
            </div>
            {labelSecondary ? <span className="sta-option-secondary">{highlight(labelSecondary, q)}</span> : null}
          </li>
        );
      }}
      loadingText="Loading tables..."
      noOptionsText={(!effectiveDatasourceId && 'Select a datasource') || (input ? 'No matches' : 'No tables')}      
      renderInput={(params) => (
        <TextField
          {...params}
          size="small"
          autoFocus={autoFocus}
          placeholder={placeholder}
          inputProps={{ ...params.inputProps, autoComplete: 'off' }}
          InputProps={{
            ...params.InputProps,
            endAdornment: (
              <>
                {(input || value) && !loading ? (
                  <button
                    type="button"
                    className="sta-clear-btn"
                    onClick={(e) => { e.stopPropagation(); setInput(''); handleChange(null, null); }}
                    aria-label="Clear"
                  >×</button>
                ) : null}
                {loading ? <CircularProgress size={16} /> : null}
                {params.InputProps.endAdornment}
              </>
            )
          }}
          helperText={(!effectiveDatasourceId && 'Select a datasource to enable search') || error || ' '}
          FormHelperTextProps={{ style: { minHeight: 16 } }}
          onKeyDown={(e) => {
            if (e.key === 'Enter') {
              if (!selectedOption && displayOptions.length > 0) {
                e.preventDefault();
                handleChange(null, displayOptions[0]);
              }
            }
          }}
        />
      )}
    />
  );
};

export default ModelTableAutocomplete;
